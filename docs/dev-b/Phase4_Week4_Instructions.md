# Phase 4 · Week 4 执行指令
**主题：代理端点流式 SSE + 记账系统**
**工时预算：15 小时（3h/天 × 5 天）**
**完成标准：流式 SSE 全链路打通，流结束后记账，管理员充值审核接口完成**

---

## 前置检查

```bash
# 确认 Week 3 验收通过
bash scripts/week3_verify.sh
# 期望：🎉 Week 3 全部验收通过！

# 确认非流式代理正常工作
BUYER_API_KEY="your-test-key"
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $BUYER_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hi"}],"stream":false}' | python3 -m json.tool
# 期望：正常响应或 503（无可用账号）
```

---

## Day 1-2 · 流式 SSE 代理

**目标：`stream: true` 时透传 SSE，流结束后异步记账**

### Step 1：流式 Dispatch 核心逻辑

```bash
cat > api/internal/proxy/stream.go << 'EOF'
package proxy

import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/ZachCharles666/gatelink/api/internal/accounting"
    "github.com/ZachCharles666/gatelink/api/internal/engine"
)

// streamUsage 从 SSE 末尾 usage chunk 提取用量（OpenAI 格式）
type streamUsage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
}

// handleStream 处理流式请求：透传 SSE，流结束后查 usage_records 记账
func (h *Handler) handleStream(c *gin.Context, buyerID, vendor, model string, req chatRequest) {
    engineURL := os.Getenv("ENGINE_INTERNAL_URL")
    if engineURL == "" {
        engineURL = "http://engine:8081"
    }

    dispatchReq := engine.DispatchRequest{
        BuyerID:     buyerID,
        Vendor:      vendor,
        Model:       model,
        Messages:    req.Messages,
        Stream:      true,
        MaxTokens:   req.MaxTokens,
        Temperature: req.Temperature,
    }

    body, _ := json.Marshal(dispatchReq)
    httpReq, _ := http.NewRequestWithContext(c.Request.Context(),
        "POST", engineURL+"/internal/v1/dispatch", bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 120 * time.Second}
    resp, err := client.Do(httpReq)
    if err != nil {
        c.JSON(http.StatusBadGateway, gin.H{
            "error": gin.H{"message": "upstream connection failed", "type": "upstream_error"},
        })
        return
    }
    defer resp.Body.Close()

    // 检查是否真的是 SSE（engine 无账号时返回 JSON 错误）
    if resp.Header.Get("Content-Type") != "text/event-stream" {
        var errResp engine.EngineResp
        json.NewDecoder(resp.Body).Decode(&errResp)
        if errResp.Code == 4001 {
            c.JSON(http.StatusServiceUnavailable, gin.H{
                "error": gin.H{"message": "no available account", "type": "service_unavailable"},
            })
            return
        }
        c.JSON(http.StatusBadGateway, gin.H{
            "error": gin.H{"message": "upstream error", "type": "upstream_error"},
        })
        return
    }

    // 设置 SSE 响应头
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no")

    // 透传 SSE，同时解析 usage chunk
    var lastUsage streamUsage
    var accountID string

    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        line := scanner.Text()

        // 透传给客户端
        fmt.Fprintf(c.Writer, "%s\n", line)
        if f, ok := c.Writer.(http.Flusher); ok {
            f.Flush()
        }

        // 解析 usage（从 data: {"choices":[],"usage":{...}} 末尾 chunk）
        if strings.HasPrefix(line, "data: ") {
            data := strings.TrimPrefix(line, "data: ")
            if data == "[DONE]" {
                break
            }
            var chunk map[string]interface{}
            if json.Unmarshal([]byte(data), &chunk) == nil {
                if usage, ok := chunk["usage"].(map[string]interface{}); ok {
                    if pt, ok := usage["prompt_tokens"].(float64); ok {
                        lastUsage.PromptTokens = int(pt)
                    }
                    if ct, ok := usage["completion_tokens"].(float64); ok {
                        lastUsage.CompletionTokens = int(ct)
                    }
                }
                // 从 chunk 提取 account_id（engine 可能在最后一帧附带）
                if id, ok := chunk["account_id"].(string); ok && id != "" {
                    accountID = id
                }
            }
        }
    }

    // 流结束后异步记账（不阻塞响应）
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        h.accounting.ChargeAfterStream(ctx, buyerID, accountID, vendor, lastUsage.PromptTokens, lastUsage.CompletionTokens)
    }()
}

// EngineResp 用于流式错误检测
type EngineResp = engine.EngineRespPublic
EOF
```

### Step 2：engine.EngineRespPublic（暴露给 proxy 使用）

```bash
# 在 api/internal/engine/client.go 末尾追加
cat >> api/internal/engine/client.go << 'EOF'

// EngineRespPublic 供其他包读取错误码
type EngineRespPublic struct {
    Code int    `json:"code"`
    Msg  string `json:"msg"`
}
EOF
```

### Step 3：记账 Service 追加流式记账方法

```bash
cat >> api/internal/accounting/service.go << 'EOF'

// ChargeAfterStream 流结束后记账
// 若无法从 SSE chunk 拿到 account_id，则查 usage_records 最新一条匹配的记录
func (s *Service) ChargeAfterStream(ctx context.Context, buyerID, accountID, vendor string, inputTokens, outputTokens int) {
    // 如果 SSE 未携带 account_id，从 usage_records 找最近一条
    if accountID == "" {
        s.db.QueryRow(ctx, `
            SELECT account_id FROM usage_records
            WHERE buyer_id = $1 AND vendor = $2
            ORDER BY created_at DESC LIMIT 1`, buyerID, vendor).Scan(&accountID)
    }
    if accountID == "" {
        return // 无法定位，放弃记账（需告警）
    }

    // 查 vendor_pricing + accounts，计算费用
    var platformDiscount float64
    s.db.QueryRow(ctx, `SELECT platform_discount FROM vendor_pricing WHERE vendor = $1 LIMIT 1`, vendor).Scan(&platformDiscount)
    if platformDiscount == 0 {
        platformDiscount = 0.88
    }

    var sellerID string
    var expectedRate float64
    var inputPer1M, outputPer1M float64
    s.db.QueryRow(ctx, `
        SELECT a.seller_id, a.expected_rate, vp.input_price_per_1m, vp.output_price_per_1m
        FROM accounts a
        JOIN vendor_pricing vp ON vp.vendor = a.vendor
        WHERE a.id = $1 LIMIT 1`, accountID).Scan(&sellerID, &expectedRate, &inputPer1M, &outputPer1M)

    if expectedRate == 0 {
        expectedRate = 0.75
    }

    costUSD := float64(inputTokens)/1e6*inputPer1M + float64(outputTokens)/1e6*outputPer1M
    buyerChargedUSD := costUSD * platformDiscount
    sellerEarnUSD := costUSD * expectedRate

    tx, err := s.db.Begin(ctx)
    if err != nil {
        return
    }
    defer tx.Rollback(ctx)

    tag, _ := tx.Exec(ctx, `
        UPDATE buyers
        SET balance_usd = balance_usd - $1, total_consumed_usd = total_consumed_usd + $1, updated_at = NOW()
        WHERE id = $2 AND balance_usd >= $1`, buyerChargedUSD, buyerID)
    if tag.RowsAffected() == 0 {
        return // 余额不足（告警）
    }

    tx.Exec(ctx, `
        UPDATE sellers SET pending_earn_usd = pending_earn_usd + $1, updated_at = NOW()
        WHERE id = $2`, sellerEarnUSD, sellerID)

    tx.Commit(ctx)
}
EOF
```

### Step 4：更新 proxy/handler.go 的 ChatCompletions，接入流式分支

```go
// 在 ChatCompletions 方法中，将 "流式请求在 Phase 4 实现" 占位替换为：

if req.Stream {
    h.handleStream(c, buyerID, vendor, req.Model, req)
    return
}
```

**✅ Day 1-2 验收**
```bash
cd api && go build ./...
# 期望：编译无错误

# 流式测试（需要有可用账号；池空时验证错误处理）
BUYER_API_KEY="your-test-key"
curl -s -N -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $BUYER_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hi"}],"stream":true}'
# 期望：SSE 事件流（data: {...} 或 503 无账号错误）
```

---

## Day 3 · 管理员接口（充值审核 + 账号强制下线）

**目标：管理员确认/拒绝充值申请，强制暂停账号**

### Step 1：管理员 Handler

```bash
mkdir -p api/internal/admin

cat > api/internal/admin/handler.go << 'EOF'
package admin

import (
    "time"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/ZachCharles666/gatelink/api/internal/response"
)

type Handler struct{ db *pgxpool.Pool }

func NewHandler(db *pgxpool.Pool) *Handler { return &Handler{db: db} }

// GET /api/v1/admin/topup/pending
func (h *Handler) ListPendingTopup(c *gin.Context) {
    rows, _ := h.db.Query(c.Request.Context(), `
        SELECT t.id, t.buyer_id, b.email, t.amount_usd, t.tx_hash, t.network, t.created_at
        FROM topup_records t JOIN buyers b ON b.id = t.buyer_id
        WHERE t.status = 'pending'
        ORDER BY t.created_at ASC LIMIT 50`)
    defer rows.Close()

    var records []gin.H
    for rows.Next() {
        var id, buyerID, email, txHash, network string
        var amountUSD float64
        var createdAt time.Time
        rows.Scan(&id, &buyerID, &email, &amountUSD, &txHash, &network, &createdAt)
        records = append(records, gin.H{
            "id": id, "buyer_id": buyerID, "email": email,
            "amount_usd": amountUSD, "tx_hash": txHash,
            "network": network, "created_at": createdAt,
        })
    }
    response.OK(c, gin.H{"records": records})
}

// POST /api/v1/admin/topup/:id/confirm
func (h *Handler) ConfirmTopup(c *gin.Context) {
    topupID := c.Param("id")

    tx, err := h.db.Begin(c.Request.Context())
    if err != nil {
        response.InternalError(c)
        return
    }
    defer tx.Rollback(c.Request.Context())

    // 查充值记录
    var buyerID string
    var amountUSD float64
    err = tx.QueryRow(c.Request.Context(), `
        SELECT buyer_id, amount_usd FROM topup_records
        WHERE id = $1 AND status = 'pending'`, topupID).
        Scan(&buyerID, &amountUSD)
    if err != nil {
        response.NotFound(c, "topup record not found or already processed")
        return
    }

    // 更新充值状态
    tx.Exec(c.Request.Context(), `
        UPDATE topup_records SET status = 'confirmed', confirmed_at = NOW()
        WHERE id = $1`, topupID)

    // 充值到买家余额
    tx.Exec(c.Request.Context(), `
        UPDATE buyers SET balance_usd = balance_usd + $1, updated_at = NOW()
        WHERE id = $2`, amountUSD, buyerID)

    if err := tx.Commit(c.Request.Context()); err != nil {
        response.InternalError(c)
        return
    }

    response.OK(c, gin.H{
        "topup_id":   topupID,
        "buyer_id":   buyerID,
        "amount_usd": amountUSD,
        "message":    "充值已确认，买家余额已更新",
    })
}

// POST /api/v1/admin/topup/:id/reject
func (h *Handler) RejectTopup(c *gin.Context) {
    topupID := c.Param("id")
    var req struct {
        Reason string `json:"reason"`
    }
    c.ShouldBindJSON(&req)

    tag, _ := h.db.Exec(c.Request.Context(), `
        UPDATE topup_records
        SET status = 'rejected', rejected_at = NOW(), notes = $1
        WHERE id = $2 AND status = 'pending'`, req.Reason, topupID)

    if tag.RowsAffected() == 0 {
        response.NotFound(c, "topup record not found or already processed")
        return
    }
    response.OK(c, gin.H{"message": "充值已拒绝"})
}

// POST /api/v1/admin/accounts/:id/force-suspend
func (h *Handler) ForceSuspend(c *gin.Context) {
    accountID := c.Param("id")
    h.db.Exec(c.Request.Context(), `
        UPDATE accounts SET status = 'suspended', updated_at = NOW() WHERE id = $1`, accountID)
    response.OK(c, gin.H{"account_id": accountID, "status": "suspended"})
}

// GET /api/v1/admin/settlements/pending
func (h *Handler) ListPendingSettlements(c *gin.Context) {
    rows, _ := h.db.Query(c.Request.Context(), `
        SELECT s.id, s.seller_id, se.display_name, s.seller_earn_usd, s.period_start, s.period_end, s.created_at
        FROM settlements s JOIN sellers se ON se.id = s.seller_id
        WHERE s.status = 'pending' ORDER BY s.created_at ASC LIMIT 50`)
    defer rows.Close()

    var settlements []gin.H
    for rows.Next() {
        var id, sellerID, displayName string
        var earnUSD float64
        var periodStart, periodEnd, createdAt time.Time
        rows.Scan(&id, &sellerID, &displayName, &earnUSD, &periodStart, &periodEnd, &createdAt)
        settlements = append(settlements, gin.H{
            "id": id, "seller_id": sellerID, "display_name": displayName,
            "seller_earn_usd": earnUSD, "period_start": periodStart,
            "period_end": periodEnd, "created_at": createdAt,
        })
    }
    response.OK(c, gin.H{"settlements": settlements})
}

// POST /api/v1/admin/settlements/:id/pay
func (h *Handler) PaySettlement(c *gin.Context) {
    settlementID := c.Param("id")
    var req struct {
        TxHash string `json:"tx_hash" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }

    tx, err := h.db.Begin(c.Request.Context())
    if err != nil {
        response.InternalError(c)
        return
    }
    defer tx.Rollback(c.Request.Context())

    var sellerID string
    var earnUSD float64
    tx.QueryRow(c.Request.Context(), `
        SELECT seller_id, seller_earn_usd FROM settlements
        WHERE id = $1 AND status = 'pending'`, settlementID).Scan(&sellerID, &earnUSD)

    tx.Exec(c.Request.Context(), `
        UPDATE settlements SET status = 'paid', paid_at = NOW(), tx_hash = $1
        WHERE id = $2`, req.TxHash, settlementID)

    // 将待结算金额转入已结算
    tx.Exec(c.Request.Context(), `
        UPDATE sellers
        SET pending_earn_usd = pending_earn_usd - $1, total_earned_usd = total_earned_usd + $1, updated_at = NOW()
        WHERE id = $2`, earnUSD, sellerID)

    if err := tx.Commit(c.Request.Context()); err != nil {
        response.InternalError(c)
        return
    }
    response.OK(c, gin.H{"settlement_id": settlementID, "status": "paid", "message": "结算已完成"})
}
EOF
```

### Step 2：在 router.go 中注册管理员路由

```go
// 在 SetupRoutes 函数末尾追加：
adminGroup := r.Group("/api/v1/admin", auth.RequireRole("admin"))
{
    adminGroup.GET("/topup/pending", adminH.ListPendingTopup)
    adminGroup.POST("/topup/:id/confirm", adminH.ConfirmTopup)
    adminGroup.POST("/topup/:id/reject", adminH.RejectTopup)
    adminGroup.GET("/settlements/pending", adminH.ListPendingSettlements)
    adminGroup.POST("/settlements/:id/pay", adminH.PaySettlement)
    adminGroup.POST("/accounts/:id/force-suspend", adminH.ForceSuspend)
}
```

**✅ Day 3 验收**
```bash
cd api && go build ./...

# 创建 admin 用户（需手动插入 DB，MVP 不做 admin 注册接口）
docker exec -it postgres psql -U postgres -d tokenglide \
  -c "UPDATE sellers SET status = 'active' WHERE id = 'admin-seller-id';"

# 查看待审核充值
ADMIN_TOKEN="eyJ..."
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:8080/api/v1/admin/topup/pending | python3 -m json.tool
```

---

## Day 4-5 · Week 4 全面验收

### 流式联调测试（需有可用账号）

```bash
# 如果 Dev-A 有测试账号可用，验证完整流式链路
BUYER_API_KEY="your-test-api-key"
curl -N -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $BUYER_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"say hello"}],"stream":true}'

# 期望：SSE 格式输出
# data: {"id":"msg_xxx","object":"chat.completion.chunk","choices":[...],"model":"claude-sonnet-4-6"}
# ...
# data: [DONE]

# 等待 5 秒后确认记账
sleep 5
BUYER_TOKEN="eyJ..."
curl -s -H "Authorization: Bearer $BUYER_TOKEN" \
  http://localhost:8080/api/v1/buyer/balance | python3 -m json.tool
# 期望：balance_usd 已扣减
```

### 验收脚本

```bash
cat > scripts/week4_verify.sh << 'EOF'
#!/bin/bash
PASS=0; FAIL=0
GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'; NC='\033[0m'

check() {
    if eval "$2" | grep -q "$3"; then
        echo -e "${GREEN}✅ PASS${NC} $1"; PASS=$((PASS+1))
    else
        echo -e "${RED}❌ FAIL${NC} $1"; FAIL=$((FAIL+1))
    fi
}

BUYER_RESP=$(curl -s -X POST http://localhost:8080/api/v1/buyer/auth/register \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"week4_$(date +%s)@test.com\",\"password\":\"pass123\"}")
BUYER_API_KEY=$(echo $BUYER_RESP | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['api_key'])")

check "流式请求格式正确（无账号时 503）" \
    "curl -s -X POST http://localhost:8080/v1/chat/completions \
    -H 'Authorization: Bearer $BUYER_API_KEY' \
    -H 'Content-Type: application/json' \
    -d '{\"model\":\"claude-sonnet-4-6\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}],\"stream\":true}'" \
    'service_unavailable\|event-stream\|DONE'

check "管理员接口编译可访问" \
    "curl -s http://localhost:8080/api/v1/admin/topup/pending" '"code":1002'

check "记账 Service 编译正常" \
    "cd api && go build ./internal/accounting/... 2>&1 && echo SUCCESS" 'SUCCESS'

echo -e "\n通过 $PASS，失败 $FAIL"
[ $FAIL -eq 0 ] && echo -e "${GREEN}🎉 Week 4 全部验收通过！${NC}" || exit 1
EOF
chmod +x scripts/week4_verify.sh
bash scripts/week4_verify.sh
```

**✅ Week 4 完成标准**
- [ ] 流式 SSE 透传正常（有账号时完整流，无账号时 503）
- [ ] 流结束后异步记账（buyer 余额扣减，seller pending 增加）
- [ ] 管理员充值审核接口（confirm/reject）功能正常
- [ ] 管理员结算付款接口功能正常
- [ ] 账号强制暂停接口正常
- [ ] 记账事务：余额不足时不扣款，事务回滚

## 下周预告（Week 5）

Week 5 实现：
- 结算周期自动化（定时触发，14 天周期）
- DB 轮询模块（替代 Redis Pub/Sub 感知账号状态变更）
- 卖家前端启动（Next.js 项目初始化）

前置条件：Week 4 流式 + 记账全链路通过
