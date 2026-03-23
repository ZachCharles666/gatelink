# Phase 2 · Week 2 执行指令
**主题：Engine 客户端 + 卖家业务 API**
**工时预算：15 小时（3h/天 × 5 天）**
**完成标准：卖家可添加账号、查看收益，Engine 客户端能正确调用 Dev-A 所有接口**

---

## 前置检查

```bash
# 确认 Week 1 验收通过
bash scripts/week1_verify.sh
# 期望：🎉 Week 1 全部验收通过！

# 确认 Dev-A Week 2 就绪（verify 接口可用）
curl -s -X POST http://localhost:8081/internal/v1/accounts/test-id/verify \
  -H "Content-Type: application/json" \
  -d '{"api_key":"sk-ant-test"}' | python3 -m json.tool
# 期望：{"code":0,"msg":"ok","data":{"valid":true,...}}

# 确认 pool/status 接口可用
curl -s http://localhost:8081/internal/v1/pool/status | python3 -m json.tool
# 期望：{"code":0,"msg":"ok","data":{"pool_counts":{...}}}
```

---

## Day 1 · Engine 客户端封装

**目标：封装所有对 Dev-A 内部接口的调用，统一处理错误码，其他模块只调此客户端**

### Step 1：Engine 客户端

```bash
cat > api/internal/engine/client.go << 'EOF'
package engine

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "time"
)

// Client Dev-A 引擎内部接口客户端
// 所有对 http://engine:8081 的调用都通过此客户端
type Client struct {
    baseURL    string
    httpClient *http.Client
}

func New() *Client {
    return &Client{
        baseURL:    getEngineURL(),
        httpClient: &http.Client{Timeout: 35 * time.Second}, // 比买家侧超时（60s）短
    }
}

func getEngineURL() string {
    url := os.Getenv("ENGINE_INTERNAL_URL")
    if url == "" {
        return "http://engine:8081"
    }
    return url
}

// engineResp Dev-A 统一响应结构
type engineResp struct {
    Code int             `json:"code"`
    Msg  string          `json:"msg"`
    Data json.RawMessage `json:"data"`
}

// do 执行 HTTP 请求，解析统一响应格式
func (c *Client) do(ctx context.Context, method, path string, body interface{}) (*engineResp, error) {
    var reqBody io.Reader
    if body != nil {
        b, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("marshal request: %w", err)
        }
        reqBody = bytes.NewReader(b)
    }

    req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
    if err != nil {
        return nil, err
    }
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("engine request failed: %w", err)
    }
    defer resp.Body.Close()

    var result engineResp
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("decode engine response: %w", err)
    }
    return &result, nil
}

// --- 接口方法 ---

// VerifyResult 账号格式验证结果
type VerifyResult struct {
    AccountID string `json:"account_id"`
    Vendor    string `json:"vendor"`
    Valid     bool   `json:"valid"`
    ErrorMsg  string `json:"error_msg"`
}

// VerifyAccount 验证 API Key 格式（MVP 仅格式校验，非真实有效性）
func (c *Client) VerifyAccount(ctx context.Context, accountID string, apiKey string) (*VerifyResult, error) {
    resp, err := c.do(ctx, "POST", "/internal/v1/accounts/"+accountID+"/verify",
        map[string]string{"api_key": apiKey})
    if err != nil {
        return nil, err
    }
    if resp.Code != 0 {
        return nil, fmt.Errorf("verify failed: %s (code=%d)", resp.Msg, resp.Code)
    }
    var result VerifyResult
    json.Unmarshal(resp.Data, &result)
    return &result, nil
}

// PoolStatus 账号池状态
type PoolStatus struct {
    PoolCounts map[string]int `json:"pool_counts"`
}

// GetPoolStatus 获取各厂商可用账号数量
func (c *Client) GetPoolStatus(ctx context.Context) (*PoolStatus, error) {
    resp, err := c.do(ctx, "GET", "/internal/v1/pool/status", nil)
    if err != nil {
        return nil, err
    }
    var status PoolStatus
    json.Unmarshal(resp.Data, &status)
    return &status, nil
}

// AccountHealth 账号健康度
type AccountHealth struct {
    AccountID    string        `json:"account_id"`
    HealthScore  int           `json:"health_score"`
    Status       string        `json:"status"`
    RecentEvents []HealthEvent `json:"recent_events"`
}

type HealthEvent struct {
    Type       string      `json:"type"`
    Delta      int         `json:"delta"`
    ScoreAfter int         `json:"score_after"`
    CreatedAt  string      `json:"created_at"`
}

// GetAccountHealth 获取账号健康度
func (c *Client) GetAccountHealth(ctx context.Context, accountID string) (*AccountHealth, error) {
    resp, err := c.do(ctx, "GET", "/internal/v1/accounts/"+accountID+"/health", nil)
    if err != nil {
        return nil, err
    }
    var health AccountHealth
    json.Unmarshal(resp.Data, &health)
    return &health, nil
}

// ConsoleUsage 平台消耗记录（注意：不是官方 Console 数据）
type ConsoleUsage struct {
    AccountID string       `json:"account_id"`
    Records   []UsageRecord `json:"records"`
}

type UsageRecord struct {
    Date         string  `json:"date"`
    TotalCostUSD float64 `json:"total_cost_usd"`
    InputTokens  int     `json:"input_tokens"`
    OutputTokens int     `json:"output_tokens"`
    RequestCount int     `json:"request_count"`
}

// GetConsoleUsage 获取账号消耗记录（来自平台 usage_records 聚合，非官方 Console）
func (c *Client) GetConsoleUsage(ctx context.Context, accountID string) (*ConsoleUsage, error) {
    resp, err := c.do(ctx, "GET", "/internal/v1/accounts/"+accountID+"/console-usage", nil)
    if err != nil {
        return nil, err
    }
    var usage ConsoleUsage
    json.Unmarshal(resp.Data, &usage)
    return &usage, nil
}

// DiffResult 对账事件列表
type DiffResult struct {
    AccountID string      `json:"account_id"`
    Diffs     []DiffEvent `json:"diffs"`
}

type DiffEvent struct {
    Type      string                 `json:"type"` // reconcile_pass / reconcile_fail
    Detail    map[string]interface{} `json:"detail"`
    CreatedAt string                 `json:"created_at"`
}

// GetAccountDiff 获取对账事件列表
func (c *Client) GetAccountDiff(ctx context.Context, accountID string) (*DiffResult, error) {
    resp, err := c.do(ctx, "GET", "/internal/v1/accounts/"+accountID+"/diff", nil)
    if err != nil {
        return nil, err
    }
    var diff DiffResult
    json.Unmarshal(resp.Data, &diff)
    return &diff, nil
}

// AuditRequest 内容审核请求
type AuditRequest struct {
    Messages []string `json:"messages"` // 字符串数组，不是 objects
    BuyerID  string   `json:"buyer_id"`
}

// AuditResult 审核结果
type AuditResult struct {
    Safe   bool   `json:"safe"`
    Level  int    `json:"level"` // 0=safe, 1=low, 2=medium, 3=high, 4=critical
    Reason string `json:"reason"`
}

// Audit 内容审核（level >= 3 时拦截）
// 返回 (nil, err) 代表审核接口失败；返回 result.Safe=false 代表被拦截
func (c *Client) Audit(ctx context.Context, req AuditRequest) (*AuditResult, error) {
    resp, err := c.do(ctx, "POST", "/internal/v1/audit", req)
    if err != nil {
        return nil, err
    }
    // HTTP 400 + code=4003 表示审核拦截
    if resp.Code == 4003 {
        return &AuditResult{Safe: false, Reason: resp.Msg}, nil
    }
    var result AuditResult
    json.Unmarshal(resp.Data, &result)
    return &result, nil
}

// DispatchRequest 代理转发请求
type DispatchRequest struct {
    BuyerID         string                   `json:"buyer_id"`
    Vendor          string                   `json:"vendor"`
    Model           string                   `json:"model"`
    Messages        []map[string]string      `json:"messages"` // OpenAI messages 格式
    Stream          bool                     `json:"stream"`
    MaxTokens       int                      `json:"max_tokens,omitempty"`
    Temperature     float64                  `json:"temperature,omitempty"`
    BuyerChargeRate float64                  `json:"buyer_charge_rate,omitempty"` // 默认 1.10
}

// DispatchResult 代理转发结果（非流式）
type DispatchResult struct {
    Response    json.RawMessage `json:"response"`    // OpenAI ChatCompletion 格式，直接透传给买家
    AccountID   string          `json:"account_id"`
    Vendor      string          `json:"vendor"`
    CostUSD     float64         `json:"cost_usd"`    // 厂商实际成本，用于记账
    InputTokens int             `json:"input_tokens"`
    OutputTokens int            `json:"output_tokens"`
}

// Dispatch 代理转发（非流式）
func (c *Client) Dispatch(ctx context.Context, req DispatchRequest) (*DispatchResult, error) {
    resp, err := c.do(ctx, "POST", "/internal/v1/dispatch", req)
    if err != nil {
        return nil, err
    }
    if resp.Code != 0 {
        return nil, &EngineError{Code: resp.Code, Msg: resp.Msg}
    }
    var result DispatchResult
    json.Unmarshal(resp.Data, &result)
    return &result, nil
}

// EngineError Dev-A 引擎错误
type EngineError struct {
    Code int
    Msg  string
}
func (e *EngineError) Error() string { return fmt.Sprintf("engine error %d: %s", e.Code, e.Msg) }
func IsNoAccount(err error) bool     { e, ok := err.(*EngineError); return ok && e.Code == 4001 }
func IsAuditFail(err error) bool     { e, ok := err.(*EngineError); return ok && e.Code == 4003 }
func IsVendorError(err error) bool   { e, ok := err.(*EngineError); return ok && e.Code == 5001 }
EOF
```

**✅ Day 1 验收**
```bash
cd api && go build ./internal/engine/...
# 期望：编译无错误

# 手动验证 Engine 客户端
cat > /tmp/test_engine.go << 'EOF'
package main
import (
    "context"
    "fmt"
    "github.com/ZachCharles666/gatelink/api/internal/engine"
)
func main() {
    c := engine.New()
    status, err := c.GetPoolStatus(context.Background())
    fmt.Println("pool status:", status, err)
}
EOF
go run /tmp/test_engine.go
```

---

## Day 2-3 · 卖家业务 API

**目标：实现 5 个卖家接口，Week 2 末联调「卖家添加账号」全流程**

### 接口 1：POST /api/v1/seller/accounts（添加账号）

```go
// 在 api/internal/seller/handler.go 追加

// AddAccount 卖家添加托管账号
func (h *Handler) AddAccount(c *gin.Context) {
    sellerID := c.GetString("user_id")
    var req struct {
        Vendor              string  `json:"vendor" binding:"required"`
        APIKey              string  `json:"api_key" binding:"required"`
        AuthorizedCreditsUSD float64 `json:"authorized_credits_usd" binding:"required,gt=0"`
        ExpectedRate        float64 `json:"expected_rate" binding:"required,gte=0.5,lte=0.95"`
        ExpireAt            string  `json:"expire_at" binding:"required"`
        TotalCreditsUSD     float64 `json:"total_credits_usd"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }

    // 在 DB 预创建账号记录（status=pending_verify），拿到 account_id
    // ⚠️ api_key 不由 Dev-B 加密存储，直接传给 Dev-A verify 接口
    // 实际加密由 Dev-A 处理（联调时确认方案）
    accountID, err := h.svc.PreCreateAccount(c.Request.Context(), sellerID, req.Vendor,
        req.AuthorizedCreditsUSD, req.ExpectedRate, req.TotalCreditsUSD, req.ExpireAt)
    if err != nil {
        response.InternalError(c)
        return
    }

    // 调用 Dev-A verify 接口（MVP 仅格式校验）
    verifyResult, err := h.engine.VerifyAccount(c.Request.Context(), accountID, req.APIKey)
    if err != nil || !verifyResult.Valid {
        // 验证失败，删除预创建记录
        h.svc.DeleteAccount(c.Request.Context(), accountID)
        response.BadRequest(c, "API Key format check failed: "+verifyResult.ErrorMsg)
        return
    }

    // 更新账号状态（Dev-A verify 后状态变更由 Dev-A 处理，Dev-B 轮询感知）
    response.OK(c, gin.H{
        "account_id":   accountID,
        "health_score": 80,
        "status":       "pending_verify",
        "message":      "格式检查通过，账号验证中。建议发起一次测试请求验证 Key 实际有效性。",
    })
}
```

### 接口 2：PATCH /accounts/:id/authorization（调整授权额度）

```go
func (h *Handler) UpdateAuthorization(c *gin.Context) {
    sellerID := c.GetString("user_id")
    accountID := c.Param("id")
    var req struct {
        AuthorizedCreditsUSD float64 `json:"authorized_credits_usd" binding:"required,gt=0"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }

    // 校验账号归属
    if err := h.svc.VerifyOwnership(c.Request.Context(), accountID, sellerID); err != nil {
        response.Fail(c, 403, 1003, "forbidden")
        return
    }

    if err := h.svc.UpdateAuthorization(c.Request.Context(), accountID, req.AuthorizedCreditsUSD); err != nil {
        response.InternalError(c)
        return
    }

    response.OK(c, gin.H{
        "account_id":            accountID,
        "authorized_credits_usd": req.AuthorizedCreditsUSD,
    })
}
```

### 接口 3：DELETE /accounts/:id/authorization（撤回授权）

```go
func (h *Handler) RevokeAuthorization(c *gin.Context) {
    sellerID := c.GetString("user_id")
    accountID := c.Param("id")

    if err := h.svc.VerifyOwnership(c.Request.Context(), accountID, sellerID); err != nil {
        response.Fail(c, 403, 1003, "forbidden")
        return
    }

    revokedAmount, err := h.svc.RevokeAuthorization(c.Request.Context(), accountID)
    if err != nil {
        response.InternalError(c)
        return
    }

    response.OK(c, gin.H{
        "revoked_amount_usd": revokedAmount,
        "message":            "授权已撤回，剩余额度归还成功",
    })
}
```

### 接口 4：GET /accounts/:id/usage（账号消耗进度，并发查询）

```go
func (h *Handler) GetAccountUsage(c *gin.Context) {
    sellerID := c.GetString("user_id")
    accountID := c.Param("id")

    if err := h.svc.VerifyOwnership(c.Request.Context(), accountID, sellerID); err != nil {
        response.Fail(c, 403, 1003, "forbidden")
        return
    }

    // 并发查询 3 个数据源，减少延迟
    type result struct {
        health  *engine.AccountHealth
        records *engine.ConsoleUsage
        diff    *engine.DiffResult
        err     error
    }
    ch := make(chan result, 1)

    go func() {
        var r result
        var wg sync.WaitGroup
        wg.Add(3)
        go func() { defer wg.Done(); r.health, _ = h.engine.GetAccountHealth(c.Request.Context(), accountID) }()
        go func() { defer wg.Done(); r.records, _ = h.engine.GetConsoleUsage(c.Request.Context(), accountID) }()
        go func() { defer wg.Done(); r.diff, _ = h.engine.GetAccountDiff(c.Request.Context(), accountID) }()
        wg.Wait()
        ch <- r
    }()

    account, _ := h.svc.GetAccount(c.Request.Context(), accountID)
    r := <-ch

    response.OK(c, gin.H{
        "authorized":      account.AuthorizedCreditsUSD,
        "consumed":        account.ConsumedCreditsUSD,
        "remaining":       account.AuthorizedCreditsUSD - account.ConsumedCreditsUSD,
        "health_score":    getHealthScore(r.health),
        "daily_records":   getRecords(r.records),  // 标注"平台记录"，非官方 Console
        "diff_events":     getDiffs(r.diff),        // 对账事件列表
    })
}
```

### 接口 5：GET /seller/earnings（收益概览）

```go
func (h *Handler) GetEarnings(c *gin.Context) {
    sellerID := c.GetString("user_id")
    seller, _ := h.svc.GetSeller(c.Request.Context(), sellerID)
    settlements, _ := h.svc.GetRecentSettlements(c.Request.Context(), sellerID, 5)

    response.OK(c, gin.H{
        "pending_usd":      seller.PendingEarnUSD,
        "total_earned_usd": seller.TotalEarnedUSD,
        "settlements":      settlements,
    })
}
```

**✅ Day 2-3 验收**
```bash
SELLER_TOKEN="eyJ..."

# 添加账号
curl -s -X POST http://localhost:8080/api/v1/seller/accounts \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"vendor":"anthropic","api_key":"sk-ant-test","authorized_credits_usd":100,"expected_rate":0.75,"expire_at":"2026-09-01T00:00:00Z"}' | python3 -m json.tool
# 期望：{"code":0,...,"data":{"account_id":"uuid","status":"pending_verify","message":"格式检查通过..."}}

# 查看收益
curl -s -H "Authorization: Bearer $SELLER_TOKEN" http://localhost:8080/api/v1/seller/earnings | python3 -m json.tool
# 期望：{"code":0,...,"data":{"pending_usd":0,...}}
```

---

## Day 4 · 收益结算接口 + 周 2 联调准备

### GET /seller/accounts（账号列表）+ GET /seller/accounts/:id

```go
func (h *Handler) ListAccounts(c *gin.Context) {
    sellerID := c.GetString("user_id")
    accounts, _ := h.svc.ListAccountsBySeller(c.Request.Context(), sellerID)
    // 注意：只返回非敏感字段，api_key_encrypted 绝对不返回
    response.OK(c, accounts)
}
```

### GET /seller/settlements（结算历史）

```go
func (h *Handler) ListSettlements(c *gin.Context) {
    sellerID := c.GetString("user_id")
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    settlements, total, _ := h.svc.ListSettlements(c.Request.Context(), sellerID, page, 20)
    response.OK(c, gin.H{"total": total, "settlements": settlements})
}
```

---

## Day 5 · Week 2 全面验收 + 联调

### Week 2 联调检查点（与 Dev-A 共同执行）

```bash
# 联调节点：卖家添加账号端到端
# 1. 调用 POST /api/v1/seller/accounts（Dev-B）
# 2. Dev-B 调用 Dev-A verify 接口
# 3. Dev-A 返回 valid=true
# 4. 账号写入 DB，status=pending_verify
# 5. Dev-B 返回账号 ID

# 验证账号在 DB 中存在
docker exec -it postgres psql -U postgres -d tokenglide \
  -c "SELECT id, vendor, status, health_score FROM accounts LIMIT 5;"
```

### 验收脚本

```bash
cat > scripts/week2_verify.sh << 'EOF'
#!/bin/bash
set -e
PASS=0; FAIL=0
GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'; NC='\033[0m'

check() {
    if eval "$2" | grep -q "$3"; then
        echo -e "${GREEN}✅ PASS${NC} $1"; PASS=$((PASS+1))
    else
        echo -e "${RED}❌ FAIL${NC} $1"; FAIL=$((FAIL+1))
    fi
}

# 获取 token
SELLER_TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/seller/auth/login \
    -H "Content-Type: application/json" \
    -d '{"phone":"13900000001","code":"123456"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")

check "Engine 客户端：pool status 可访问" \
    "curl -s http://localhost:8081/internal/v1/pool/status" '"code":0'

check "卖家账号列表接口" \
    "curl -s -H 'Authorization: Bearer $SELLER_TOKEN' http://localhost:8080/api/v1/seller/accounts" '"code":0'

check "卖家收益接口" \
    "curl -s -H 'Authorization: Bearer $SELLER_TOKEN' http://localhost:8080/api/v1/seller/earnings" '"pending_usd"'

check "无权访问其他卖家账号返回 403" \
    "curl -s -H 'Authorization: Bearer $SELLER_TOKEN' http://localhost:8080/api/v1/seller/accounts/other-seller-account-id" '"code":1003'

echo -e "\n通过 $PASS，失败 $FAIL"
[ $FAIL -eq 0 ] && echo -e "${GREEN}🎉 Week 2 全部验收通过！${NC}" || exit 1
EOF
chmod +x scripts/week2_verify.sh
bash scripts/week2_verify.sh
```

**✅ Week 2 完成标准**
- [ ] Engine 客户端编译，所有接口方法实现
- [ ] 卖家添加账号接口与 Dev-A verify 联调通过
- [ ] 卖家授权调整/撤回接口功能正确
- [ ] 账号归属校验（不能访问其他卖家账号）
- [ ] 收益概览接口返回数据正确

## 下周预告（Week 3）

Week 3 实现：
- 买家业务 API（余额、充值、用量明细、重置 Key）
- 代理端点非流式：`POST /v1/chat/completions` 打通全链路

前置条件：Dev-A Week 3 dispatch 接口就绪
