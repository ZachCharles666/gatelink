# Phase 1 · Week 1 执行指令
**主题：项目初始化 + DB Schema 对齐 + 鉴权系统**
**工时预算：18 小时（3-4h/天 × 5 天）**
**完成标准：卖家和买家能注册登录拿到 JWT，买家 api_key 自动生成，Docker 本地启动成功**

---

## 前置检查

```bash
# 确认 Dev-A 环境已就绪（engine 服务运行）
curl -s http://localhost:8081/health | python3 -m json.tool
# 期望：{"code":0,"msg":"ok","data":{"service":"engine",...}}

# 确认数据库 7 张基础表已由 Dev-A migration 创建
docker exec -it postgres psql -U postgres -d tokenglide -c "\dt"
# 期望：看到 vendors_pricing / sellers / buyers / accounts / usage_records / health_events / settlements

# 确认 Redis 可连接
docker exec -it redis redis-cli ping
# 期望：PONG
```

---

## Day 1 · 项目初始化 + Docker 配置

**目标：Go 项目骨架可编译，Docker Compose 追加 api/web 服务**

### Step 1：初始化 Go 项目

```bash
mkdir -p api/cmd/api api/internal/{auth,seller,buyer,proxy,accounting,engine,poller,db} api/internal/db/migrations
cd api

go mod init github.com/ZachCharles666/gatelink/api
go get github.com/gin-gonic/gin
go get github.com/golang-jwt/jwt/v5
go get github.com/jackc/pgx/v5
go get github.com/redis/go-redis/v9
go get github.com/joho/godotenv
go get github.com/rs/zerolog
```

### Step 2：统一响应格式（与 Dev-A 保持一致）

```bash
mkdir -p api/internal/response

cat > api/internal/response/response.go << 'EOF'
package response

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

const (
    CodeOK             = 0
    CodeInvalidParam   = 1001
    CodeUnauthorized   = 1002
    CodeForbidden      = 1003
    CodeNotFound       = 1004
    CodeInsufficientBal= 1005
    CodeInternalError  = 5000
)

type R struct {
    Code int         `json:"code"`
    Msg  string      `json:"msg"`
    Data interface{} `json:"data,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, R{Code: CodeOK, Msg: "ok", Data: data})
}
func Fail(c *gin.Context, httpStatus, code int, msg string) {
    c.JSON(httpStatus, R{Code: code, Msg: msg})
}
func BadRequest(c *gin.Context, msg string)    { Fail(c, 400, CodeInvalidParam, msg) }
func Unauthorized(c *gin.Context)              { Fail(c, 401, CodeUnauthorized, "unauthorized") }
func NotFound(c *gin.Context, msg string)      { Fail(c, 404, CodeNotFound, msg) }
func InternalError(c *gin.Context)             { Fail(c, 500, CodeInternalError, "internal server error") }
func InsufficientBalance(c *gin.Context)       { Fail(c, 402, CodeInsufficientBal, "insufficient balance") }
EOF
```

### Step 3：健康检查 + 主入口

```bash
cat > api/cmd/api/main.go << 'EOF'
package main

import (
    "fmt"
    "net/http"
    "os"
    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"
    "github.com/rs/zerolog/log"
)

func main() {
    godotenv.Load()

    if os.Getenv("ENV") == "production" {
        gin.SetMode(gin.ReleaseMode)
    }

    r := gin.New()
    r.Use(gin.Recovery())

    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": gin.H{"service": "api", "version": "0.1.0"}})
    })

    port := os.Getenv("API_PORT")
    if port == "" { port = "8080" }
    log.Info().Str("port", port).Msg("api service started")
    if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
        log.Fatal().Err(err).Msg("server error")
    }
}
EOF
```

### Step 4：追加 docker-compose 服务（提 PR 给 Dev-A）

```bash
# 在 Dev-A 的 docker-compose.yml 中追加以下服务（提 PR，Dev-A review 后合并）
cat > api/docker-compose-devb.yml << 'EOF'
# Dev-B 追加服务，合并到 Dev-A 的 docker-compose.yml
services:
  api:
    build: ./api
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
      - JWT_SECRET=${JWT_SECRET}
      - ENGINE_INTERNAL_URL=http://engine:8081
      - API_PORT=8080
    depends_on:
      engine:
        condition: service_healthy
      postgres:
        condition: service_healthy
    restart: unless-stopped

  web:
    build: ./web
    ports:
      - "3000:3000"
    environment:
      - NEXT_PUBLIC_API_URL=http://localhost:8080
    depends_on:
      - api
    restart: unless-stopped
EOF
```

### Step 5：.env.example

```bash
cat > api/.env.example << 'EOF'
# Dev-B 专属变量
JWT_SECRET=your-64-byte-hex-secret-here
ENGINE_INTERNAL_URL=http://engine:8081
API_PORT=8080

# 共用变量（与 Dev-A 保持一致）
DATABASE_URL=postgres://postgres:postgres@localhost:5432/tokenglide?sslmode=disable
REDIS_URL=redis://localhost:6379
EOF
```

**✅ Day 1 验收**
```bash
cd api && go build ./...
# 期望：编译无错误

go run ./cmd/api &
sleep 2
curl -s http://localhost:8080/health
# 期望：{"code":0,"msg":"ok","data":{"service":"api",...}}
kill %1
```

---

## Day 2 · DB Schema 对齐 + topup_records 表

**目标：与 Dev-A 确认所有表结构，创建 Dev-B 新增的 topup_records 表**

### Step 1：核对 Schema（对照 `engine/` 实现与 `DEV_A_HANDOFF.md`）

与 Dev-A 确认以下字段格式（**Week 1 结束前必须锁定**）：

| 确认项 | 期望值 |
|--------|--------|
| `sellers.id` 类型 | UUID |
| `buyers.api_key` 长度 | VARCHAR(64) |
| `accounts.vendor` 枚举 | anthropic/openai/gemini/qwen/glm/kimi |
| `vendor_pricing.platform_discount` 精度 | DECIMAL(4,3) |

### Step 2：Dev-B 新增表 migration

```bash
cat > api/internal/db/migrations/001_topup_records.sql << 'EOF'
-- Dev-B 新增：充值申请记录表
CREATE TABLE IF NOT EXISTS topup_records (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id     UUID        NOT NULL REFERENCES buyers(id),
    amount_usd   DECIMAL(12,4) NOT NULL CHECK(amount_usd > 0),
    tx_hash      VARCHAR(100) UNIQUE NOT NULL,
    network      VARCHAR(20)  NOT NULL CHECK(network IN ('TRC20','ERC20')),
    status       VARCHAR(20)  NOT NULL DEFAULT 'pending'
                 CHECK(status IN ('pending','confirmed','rejected')),
    confirmed_at TIMESTAMPTZ,
    rejected_at  TIMESTAMPTZ,
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_topup_buyer ON topup_records(buyer_id, created_at DESC);
CREATE INDEX idx_topup_status ON topup_records(status) WHERE status = 'pending';
EOF
```

> **流程：** 将此 migration SQL 发给 Dev-A review，Dev-A 合并到 engine 的 migration 目录后，统一执行。

**✅ Day 2 验收**
```bash
# 确认 Dev-A migration 已执行，topup_records 表存在
docker exec -it postgres psql -U postgres -d tokenglide -c "\d topup_records"
# 期望：看到表结构
```

---

## Day 3 · 鉴权系统——卖家和买家注册登录

**目标：JWT 生成验证，卖家注册，买家注册时自动生成 api_key**

### Step 1：JWT 工具包

```bash
cat > api/internal/auth/jwt.go << 'EOF'
package auth

import (
    "errors"
    "os"
    "time"
    "github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
    UserID string `json:"sub"`
    Role   string `json:"role"` // seller / buyer / admin
    jwt.RegisteredClaims
}

func GenerateToken(userID, role string) (string, error) {
    claims := Claims{
        UserID: userID,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func ParseToken(tokenStr string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, ErrInvalidToken
        }
        return []byte(os.Getenv("JWT_SECRET")), nil
    })
    if err != nil || !token.Valid {
        return nil, ErrInvalidToken
    }
    return token.Claims.(*Claims), nil
}
EOF
```

### Step 2：权限中间件

```bash
cat > api/internal/auth/middleware.go << 'EOF'
package auth

import (
    "strings"
    "github.com/gin-gonic/gin"
    "github.com/ZachCharles666/gatelink/api/internal/response"
)

// RequireRole JWT 鉴权中间件，校验 role
func RequireRole(role string) gin.HandlerFunc {
    return func(c *gin.Context) {
        header := c.GetHeader("Authorization")
        if !strings.HasPrefix(header, "Bearer ") {
            response.Unauthorized(c)
            c.Abort()
            return
        }
        claims, err := ParseToken(strings.TrimPrefix(header, "Bearer "))
        if err != nil || claims.Role != role {
            response.Unauthorized(c)
            c.Abort()
            return
        }
        c.Set("user_id", claims.UserID)
        c.Set("user_role", claims.Role)
        c.Next()
    }
}
EOF
```

### Step 3：api_key 生成工具

```bash
cat > api/internal/auth/apikey.go << 'EOF'
package auth

import (
    "crypto/rand"
    "encoding/hex"
)

// GenerateBuyerAPIKey 生成 64 位随机 api_key
// 这是买家调用 /v1/* 代理端点的凭证，与厂商 Key 完全隔离
func GenerateBuyerAPIKey() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}
EOF
```

### Step 4：卖家注册/登录接口

```bash
cat > api/internal/seller/handler.go << 'EOF'
package seller

import (
    "github.com/gin-gonic/gin"
    "github.com/ZachCharles666/gatelink/api/internal/auth"
    "github.com/ZachCharles666/gatelink/api/internal/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// POST /api/v1/seller/auth/register
func (h *Handler) Register(c *gin.Context) {
    var req struct {
        Phone    string `json:"phone" binding:"required"`
        Code     string `json:"code"  binding:"required"` // 验证码（MVP 用固定值 "123456"）
        DisplayName string `json:"display_name"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }

    // MVP：验证码固定为 "123456"
    if req.Code != "123456" {
        response.BadRequest(c, "invalid verification code")
        return
    }

    seller, err := h.svc.Register(c.Request.Context(), req.Phone, req.DisplayName)
    if err != nil {
        response.InternalError(c)
        return
    }

    token, _ := auth.GenerateToken(seller.ID, "seller")
    response.OK(c, gin.H{"token": token, "seller_id": seller.ID})
}

// POST /api/v1/seller/auth/login
func (h *Handler) Login(c *gin.Context) {
    var req struct {
        Phone string `json:"phone" binding:"required"`
        Code  string `json:"code"  binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }
    if req.Code != "123456" {
        response.Unauthorized(c)
        return
    }
    seller, err := h.svc.FindByPhone(c.Request.Context(), req.Phone)
    if err != nil {
        response.Unauthorized(c)
        return
    }
    token, _ := auth.GenerateToken(seller.ID, "seller")
    response.OK(c, gin.H{"token": token, "seller_id": seller.ID})
}
EOF
```

### Step 5：买家注册时自动生成 api_key

```bash
cat > api/internal/buyer/handler.go << 'EOF'
package buyer

import (
    "github.com/gin-gonic/gin"
    "github.com/ZachCharles666/gatelink/api/internal/auth"
    "github.com/ZachCharles666/gatelink/api/internal/response"
)

type Handler struct{ svc *Service }
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// POST /api/v1/buyer/auth/register
func (h *Handler) Register(c *gin.Context) {
    var req struct {
        Email    string `json:"email"`
        Phone    string `json:"phone"`
        Password string `json:"password"`
        Code     string `json:"code"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }

    // 生成买家 api_key（注册时自动创建）
    apiKey, err := auth.GenerateBuyerAPIKey()
    if err != nil {
        response.InternalError(c)
        return
    }

    buyer, err := h.svc.Register(c.Request.Context(), req.Email, req.Phone, req.Password, apiKey)
    if err != nil {
        response.InternalError(c)
        return
    }

    token, _ := auth.GenerateToken(buyer.ID, "buyer")
    response.OK(c, gin.H{
        "token":    token,
        "buyer_id": buyer.ID,
        "api_key":  apiKey, // 仅注册时返回一次
    })
}
EOF
```

**✅ Day 3 验收**
```bash
# 卖家注册
curl -s -X POST http://localhost:8080/api/v1/seller/auth/register \
  -H "Content-Type: application/json" \
  -d '{"phone":"13800138000","code":"123456","display_name":"Test Seller"}' | python3 -m json.tool
# 期望：{"code":0,...,"data":{"token":"eyJ...","seller_id":"uuid"}}

# 买家注册
curl -s -X POST http://localhost:8080/api/v1/buyer/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"buyer@test.com","password":"test123"}' | python3 -m json.tool
# 期望：{"code":0,...,"data":{"token":"eyJ...","api_key":"64位hex字符串"}}

# 验证 api_key 长度
API_KEY=$(curl -s -X POST ... | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['api_key'])")
echo ${#API_KEY}
# 期望：64
```

---

## Day 4 · 代理端 api_key 鉴权中间件 + 路由组织

**目标：`/v1/*` 端点用 api_key 鉴权，`/api/v1/seller/*` 和 `/api/v1/buyer/*` 用 JWT**

### Step 1：代理端 api_key 鉴权中间件

```bash
cat > api/internal/auth/apikey_middleware.go << 'EOF'
package auth

import (
    "strings"
    "github.com/gin-gonic/gin"
    "github.com/ZachCharles666/gatelink/api/internal/response"
)

// BuyerAPIKeyMiddleware /v1/* 端点用 api_key 鉴权
// Header: Authorization: Bearer <buyer_api_key>
func BuyerAPIKeyMiddleware(db BuyerRepo) gin.HandlerFunc {
    return func(c *gin.Context) {
        header := c.GetHeader("Authorization")
        if !strings.HasPrefix(header, "Bearer ") {
            response.Unauthorized(c)
            c.Abort()
            return
        }
        apiKey := strings.TrimPrefix(header, "Bearer ")

        buyer, err := db.FindByAPIKey(c.Request.Context(), apiKey)
        if err != nil || buyer.Status != "active" {
            response.Unauthorized(c)
            c.Abort()
            return
        }

        c.Set("buyer_id", buyer.ID)
        c.Set("buyer_balance", buyer.BalanceUSD)
        c.Set("buyer_tier", buyer.Tier)
        c.Next()
    }
}

type BuyerRepo interface {
    FindByAPIKey(ctx interface{}, apiKey string) (*BuyerInfo, error)
}

type BuyerInfo struct {
    ID         string
    BalanceUSD float64
    Tier       string
    Status     string
}
EOF
```

### Step 2：路由组织

```bash
cat > api/internal/router.go << 'EOF'
package internal

import (
    "github.com/gin-gonic/gin"
    "github.com/ZachCharles666/gatelink/api/internal/auth"
    "github.com/ZachCharles666/gatelink/api/internal/seller"
    "github.com/ZachCharles666/gatelink/api/internal/buyer"
)

func SetupRoutes(r *gin.Engine, sellerH *seller.Handler, buyerH *buyer.Handler, buyerRepo auth.BuyerRepo) {
    // 健康检查
    r.GET("/health", handleHealth)

    // 卖家端路由（JWT 鉴权）
    sellerGroup := r.Group("/api/v1/seller")
    sellerGroup.POST("/auth/register", sellerH.Register)
    sellerGroup.POST("/auth/login", sellerH.Login)
    sellerAuth := sellerGroup.Group("", auth.RequireRole("seller"))
    {
        sellerAuth.GET("/accounts", sellerH.ListAccounts)
        sellerAuth.POST("/accounts", sellerH.AddAccount)
        // ... 其他接口在 Phase 2 实现
    }

    // 买家端路由（JWT 鉴权）
    buyerGroup := r.Group("/api/v1/buyer")
    buyerGroup.POST("/auth/register", buyerH.Register)
    buyerGroup.POST("/auth/login", buyerH.Login)
    buyerAuth := buyerGroup.Group("", auth.RequireRole("buyer"))
    {
        buyerAuth.GET("/balance", buyerH.GetBalance)
        // ... 其他接口在 Phase 3 实现
    }

    // 代理端路由（api_key 鉴权，Phase 3 实现）
    // proxyGroup := r.Group("/v1", auth.BuyerAPIKeyMiddleware(buyerRepo))
}
EOF
```

**✅ Day 4 验收**
```bash
# 无 token 访问 seller 接口应 401
curl -s http://localhost:8080/api/v1/seller/accounts
# 期望：{"code":1002,"msg":"unauthorized"}

# 用 buyer token 访问 seller 接口应 401
BUYER_TOKEN="eyJ..."
curl -s -H "Authorization: Bearer $BUYER_TOKEN" http://localhost:8080/api/v1/seller/accounts
# 期望：{"code":1002,"msg":"unauthorized"}
```

---

## Day 5 · Week 1 全面验收

### Step 1：运行所有测试

```bash
cd api
go test ./...
```

### Step 2：执行验收脚本

```bash
cat > scripts/week1_verify.sh << 'EOF'
#!/bin/bash
set -e
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
PASS=0; FAIL=0

check() {
    if eval "$2" | grep -q "$3"; then
        echo -e "${GREEN}✅ PASS${NC} $1"; PASS=$((PASS+1))
    else
        echo -e "${RED}❌ FAIL${NC} $1"; FAIL=$((FAIL+1))
    fi
}

echo -e "\n${YELLOW}=== Week 1 验收 ===${NC}\n"

go run ./cmd/api &
API_PID=$!
sleep 3

check "API 健康检查" \
    "curl -s http://localhost:8080/health" '"code":0'

check "卖家注册返回 token" \
    "curl -s -X POST http://localhost:8080/api/v1/seller/auth/register -H 'Content-Type: application/json' -d '{\"phone\":\"13900000001\",\"code\":\"123456\"}'" '"token"'

BUYER_RESP=$(curl -s -X POST http://localhost:8080/api/v1/buyer/auth/register \
    -H 'Content-Type: application/json' \
    -d '{"email":"test_w1@example.com","password":"pass123"}')

check "买家注册返回 api_key" "echo '$BUYER_RESP'" '"api_key"'

API_KEY=$(echo $BUYER_RESP | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['api_key'])")
check "api_key 长度为 64" "echo -n $API_KEY | wc -c | tr -d ' '" "64"

check "无 token 访问卖家接口 401" \
    "curl -s http://localhost:8080/api/v1/seller/accounts" '"code":1002'

kill $API_PID 2>/dev/null

echo -e "\n${YELLOW}结果：通过 $PASS，失败 $FAIL${NC}"
[ $FAIL -eq 0 ] && echo -e "${GREEN}🎉 Week 1 全部验收通过！${NC}" || exit 1
EOF
chmod +x scripts/week1_verify.sh
bash scripts/week1_verify.sh
```

**✅ Week 1 完成标准**
- [ ] `GET /health` 返回 `{"code":0,...}`
- [ ] 卖家注册/登录返回 JWT
- [ ] 买家注册自动生成 64 位 api_key
- [ ] JWT 权限中间件拦截越权访问
- [ ] topup_records 表已经 Dev-A review 并执行
- [ ] DB Schema 与 Dev-A 对齐确认（seller_id 格式、vendor 枚举等）

---

## 本周产出清单

| 产出物 | 路径 |
|--------|------|
| 统一响应格式 | `api/internal/response/response.go` |
| JWT 工具 | `api/internal/auth/jwt.go` |
| 权限中间件 | `api/internal/auth/middleware.go` |
| api_key 生成 | `api/internal/auth/apikey.go` |
| 代理鉴权中间件 | `api/internal/auth/apikey_middleware.go` |
| 卖家注册/登录 | `api/internal/seller/handler.go` |
| 买家注册/登录 | `api/internal/buyer/handler.go` |
| 路由组织 | `api/internal/router.go` |
| topup_records migration | `api/internal/db/migrations/001_topup_records.sql` |

## 下周预告（Week 2）

Week 2 实现：
- Engine 客户端（封装所有 Dev-A 内部接口调用）
- 卖家业务 API（账号添加、授权、收益查询）

前置条件：Dev-A Week 2 完成（dispatch、verify 接口就绪）
