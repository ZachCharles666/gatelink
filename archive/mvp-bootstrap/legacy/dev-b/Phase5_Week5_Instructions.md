# Phase 5 · Week 5 执行指令
**主题：结算系统 + DB 轮询 + 卖家前端启动**
**工时预算：18 小时（3-4h/天 × 5 天）**
**完成标准：结算自动触发、账号状态轮询正常、卖家前端项目初始化完成**

---

## 前置检查

```bash
# 确认 Week 4 验收通过
bash scripts/week4_verify.sh
# 期望：🎉 Week 4 全部验收通过！
```

---

## Day 1 · 结算系统

**目标：每 14 天自动为卖家生成结算单，管理员确认后付款**

### Step 1：结算 Service

```bash
mkdir -p api/internal/accounting

cat > api/internal/accounting/settlement.go << 'EOF'
package accounting

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

type SettlementService struct{ db *pgxpool.Pool }

func NewSettlementService(db *pgxpool.Pool) *SettlementService {
    return &SettlementService{db: db}
}

// RunSettlementCycle 对所有有待结算金额的卖家生成结算单
// 建议每 14 天运行一次（通过 cron 或定时 goroutine 触发）
func (s *SettlementService) RunSettlementCycle(ctx context.Context) error {
    periodEnd := time.Now().UTC()
    periodStart := periodEnd.AddDate(0, 0, -14)

    // 查找有待结算金额的卖家
    rows, err := s.db.Query(ctx, `
        SELECT id, pending_earn_usd FROM sellers
        WHERE pending_earn_usd > 0 AND status = 'active'`)
    if err != nil {
        return fmt.Errorf("query sellers: %w", err)
    }
    defer rows.Close()

    var settled int
    for rows.Next() {
        var sellerID string
        var pendingUSD float64
        rows.Scan(&sellerID, &pendingUSD)

        if err := s.createSettlement(ctx, sellerID, pendingUSD, periodStart, periodEnd); err != nil {
            // 记录日志但继续处理其他卖家
            continue
        }
        settled++
    }

    return nil
}

func (s *SettlementService) createSettlement(ctx context.Context, sellerID string, earnUSD float64, start, end time.Time) error {
    // 查本周期消耗总量（从 usage_records 聚合，Dev-B 只读）
    var totalConsumedUSD float64
    s.db.QueryRow(ctx, `
        SELECT COALESCE(SUM(cost_usd), 0) FROM usage_records ur
        JOIN accounts a ON a.id = ur.account_id
        WHERE a.seller_id = $1 AND ur.created_at BETWEEN $2 AND $3`,
        sellerID, start, end).Scan(&totalConsumedUSD)

    tx, err := s.db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // 创建结算单
    tx.Exec(ctx, `
        INSERT INTO settlements (seller_id, period_start, period_end, total_consumed_usd, seller_earn_usd)
        VALUES ($1, $2, $3, $4, $5)`,
        sellerID, start, end, totalConsumedUSD, earnUSD)

    // 清空 pending（转为待管理员付款状态）
    tx.Exec(ctx, `
        UPDATE sellers SET pending_earn_usd = 0, updated_at = NOW() WHERE id = $1`, sellerID)

    return tx.Commit(ctx)
}

// RequestSettlement 卖家主动申请提现
func (s *SettlementService) RequestSettlement(ctx context.Context, sellerID string) error {
    var pendingUSD float64
    err := s.db.QueryRow(ctx, `
        SELECT pending_earn_usd FROM sellers WHERE id = $1`, sellerID).Scan(&pendingUSD)
    if err != nil {
        return err
    }
    if pendingUSD < 10 { // 最低提现 10 USD
        return fmt.Errorf("minimum settlement amount is $10, current: $%.2f", pendingUSD)
    }

    periodEnd := time.Now().UTC()
    periodStart := periodEnd.AddDate(0, 0, -14)
    return s.createSettlement(ctx, sellerID, pendingUSD, periodStart, periodEnd)
}
EOF
```

### Step 2：在卖家 Handler 追加结算申请接口

```go
// 在 api/internal/seller/handler.go 追加

// POST /api/v1/seller/settlements/request
func (h *Handler) RequestSettlement(c *gin.Context) {
    sellerID := c.GetString("user_id")
    if err := h.settlement.RequestSettlement(c.Request.Context(), sellerID); err != nil {
        response.BadRequest(c, err.Error())
        return
    }
    response.OK(c, gin.H{"message": "结算申请已提交，等待管理员处理"})
}
```

### Step 3：定时结算触发（在 main.go 中启动）

```go
// 在 main.go 中追加
import "github.com/ZachCharles666/gatelink/api/internal/accounting"

// 启动结算定时任务（每 14 天）
settlementSvc := accounting.NewSettlementService(db)
go func() {
    ticker := time.NewTicker(14 * 24 * time.Hour)
    defer ticker.Stop()
    for range ticker.C {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
        settlementSvc.RunSettlementCycle(ctx)
        cancel()
    }
}()
```

**✅ Day 1 验收**
```bash
cd api && go build ./internal/accounting/...
# 期望：编译无错误
```

---

## Day 2 · DB 轮询模块（替代 Redis Pub/Sub）

**目标：每 30 秒检测账号状态变更，刷新卖家控制台感知**

### Step 1：账号状态轮询器

```bash
mkdir -p api/internal/poller

cat > api/internal/poller/account.go << 'EOF'
package poller

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rs/zerolog/log"
)

// AccountPoller 每 30 秒轮询 accounts 表状态变更
// 替代 Dev-A 未实现的 Redis Pub/Sub 通知
type AccountPoller struct {
    db          *pgxpool.Pool
    lastCheckAt time.Time
    interval    time.Duration
    onChange    func(accountID, status string) // 状态变更回调
}

type StatusChange struct {
    AccountID string
    SellerID  string
    OldStatus string
    NewStatus string
    UpdatedAt time.Time
}

func NewAccountPoller(db *pgxpool.Pool, onChange func(accountID, status string)) *AccountPoller {
    return &AccountPoller{
        db:          db,
        lastCheckAt: time.Now(),
        interval:    30 * time.Second,
        onChange:    onChange,
    }
}

// Start 启动轮询（阻塞，建议在 goroutine 中运行）
func (p *AccountPoller) Start(ctx context.Context) {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()

    log.Info().Msg("account poller started (30s interval)")

    for {
        select {
        case <-ctx.Done():
            log.Info().Msg("account poller stopped")
            return
        case <-ticker.C:
            if err := p.poll(ctx); err != nil {
                log.Error().Err(err).Msg("account poller error")
            }
        }
    }
}

func (p *AccountPoller) poll(ctx context.Context) error {
    checkFrom := p.lastCheckAt
    p.lastCheckAt = time.Now()

    rows, err := p.db.Query(ctx, `
        SELECT id, status, updated_at
        FROM accounts
        WHERE updated_at > $1
          AND status IN ('suspended', 'active', 'revoked', 'expired')
        ORDER BY updated_at ASC`, checkFrom)
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var accountID, status string
        var updatedAt time.Time
        rows.Scan(&accountID, &status, &updatedAt)

        log.Info().
            Str("account_id", accountID).
            Str("status", status).
            Msg("account status changed")

        if p.onChange != nil {
            p.onChange(accountID, status)
        }
    }
    return nil
}
EOF
```

### Step 2：在 main.go 中启动轮询器

```go
// 在 main.go 中追加
import "github.com/ZachCharles666/gatelink/api/internal/poller"

// 启动账号状态轮询
accountPoller := poller.NewAccountPoller(db, func(accountID, status string) {
    // 当账号状态变为 suspended/revoked 时，可以发通知给卖家
    // MVP：只记录日志，Phase 6 前端完成后可接入 WebSocket 推送
    log.Info().Str("account_id", accountID).Str("status", status).Msg("account status updated")
})
go accountPoller.Start(context.Background())
```

**✅ Day 2 验收**
```bash
cd api && go build ./internal/poller/...
# 期望：编译无错误

# 手动验证轮询（修改一个 account 状态，检查日志）
docker exec -it postgres psql -U postgres -d tokenglide \
  -c "UPDATE accounts SET status = 'suspended', updated_at = NOW() WHERE id = 'some-account-id';"
# 期望：30 秒内 api 日志中出现 "account status changed"
```

---

## Day 3-5 · 卖家前端（Next.js 初始化 + 登录注册页）

**目标：Next.js 项目骨架启动，卖家登录/注册页完成**

### Step 1：初始化 Next.js 项目

```bash
cd /Users/tvwoo/Projects/gatelink
npx create-next-app@14 web \
  --typescript \
  --tailwind \
  --app \
  --no-src-dir \
  --import-alias "@/*"

cd web
npm install axios @tanstack/react-query react-hook-form zod @hookform/resolvers
npm install recharts
npm install -D @types/node
```

### Step 2：API 封装层

```bash
mkdir -p web/lib/api

cat > web/lib/api/client.ts << 'EOF'
import axios from 'axios'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export const apiClient = axios.create({
  baseURL: API_BASE,
  timeout: 30000,
})

// 自动附带 JWT token
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('seller_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 统一响应格式解包
apiClient.interceptors.response.use(
  (res) => {
    if (res.data.code !== 0) {
      return Promise.reject(new Error(res.data.msg || 'unknown error'))
    }
    return res.data.data
  },
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('seller_token')
      window.location.href = '/seller/login'
    }
    return Promise.reject(err)
  }
)
EOF

cat > web/lib/api/seller.ts << 'EOF'
import { apiClient } from './client'

export const sellerAPI = {
  login: (phone: string, code: string) =>
    apiClient.post('/api/v1/seller/auth/login', { phone, code }),

  register: (phone: string, code: string, displayName?: string) =>
    apiClient.post('/api/v1/seller/auth/register', { phone, code, display_name: displayName }),

  getAccounts: () =>
    apiClient.get('/api/v1/seller/accounts'),

  addAccount: (data: {
    vendor: string
    api_key: string
    authorized_credits_usd: number
    expected_rate: number
    expire_at: string
    total_credits_usd?: number
  }) => apiClient.post('/api/v1/seller/accounts', data),

  getAccountUsage: (id: string) =>
    apiClient.get(`/api/v1/seller/accounts/${id}/usage`),

  updateAuthorization: (id: string, authorizedCreditsUSD: number) =>
    apiClient.patch(`/api/v1/seller/accounts/${id}/authorization`, {
      authorized_credits_usd: authorizedCreditsUSD,
    }),

  revokeAuthorization: (id: string) =>
    apiClient.delete(`/api/v1/seller/accounts/${id}/authorization`),

  getEarnings: () =>
    apiClient.get('/api/v1/seller/earnings'),

  getSettlements: (page = 1) =>
    apiClient.get(`/api/v1/seller/settlements?page=${page}`),

  requestSettlement: () =>
    apiClient.post('/api/v1/seller/settlements/request'),
}
EOF
```

### Step 3：卖家登录页

```bash
mkdir -p web/app/seller/login

cat > web/app/seller/login/page.tsx << 'EOF'
'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { sellerAPI } from '@/lib/api/seller'

export default function SellerLoginPage() {
  const router = useRouter()
  const [phone, setPhone] = useState('')
  const [code, setCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const data = await sellerAPI.login(phone, code) as any
      localStorage.setItem('seller_token', data.token)
      router.push('/seller/dashboard')
    } catch (err: any) {
      setError(err.message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full bg-white rounded-lg shadow p-8">
        <h1 className="text-2xl font-bold text-gray-900 mb-6">卖家登录</h1>
        <form onSubmit={handleLogin} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">手机号</label>
            <input
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="13800138000"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">验证码</label>
            <input
              type="text"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="MVP 固定值：123456"
              required
            />
          </div>
          {error && <p className="text-red-500 text-sm">{error}</p>}
          <button
            type="submit"
            disabled={loading}
            className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? '登录中...' : '登录'}
          </button>
          <p className="text-center text-sm text-gray-500">
            没有账号？
            <a href="/seller/register" className="text-blue-600 hover:underline ml-1">注册</a>
          </p>
        </form>
      </div>
    </div>
  )
}
EOF
```

### Step 4：卖家注册页

```bash
mkdir -p web/app/seller/register

cat > web/app/seller/register/page.tsx << 'EOF'
'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { sellerAPI } from '@/lib/api/seller'

export default function SellerRegisterPage() {
  const router = useRouter()
  const [phone, setPhone] = useState('')
  const [code, setCode] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const data = await sellerAPI.register(phone, code, displayName) as any
      localStorage.setItem('seller_token', data.token)
      router.push('/seller/dashboard')
    } catch (err: any) {
      setError(err.message || '注册失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full bg-white rounded-lg shadow p-8">
        <h1 className="text-2xl font-bold text-gray-900 mb-6">卖家注册</h1>
        <form onSubmit={handleRegister} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">手机号</label>
            <input type="tel" value={phone} onChange={(e) => setPhone(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              required />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">验证码（MVP 固定 123456）</label>
            <input type="text" value={code} onChange={(e) => setCode(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              required />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">显示名称（可选）</label>
            <input type="text" value={displayName} onChange={(e) => setDisplayName(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          {error && <p className="text-red-500 text-sm">{error}</p>}
          <button type="submit" disabled={loading}
            className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 disabled:opacity-50">
            {loading ? '注册中...' : '注册'}
          </button>
          <p className="text-center text-sm text-gray-500">
            已有账号？<a href="/seller/login" className="text-blue-600 hover:underline ml-1">登录</a>
          </p>
        </form>
      </div>
    </div>
  )
}
EOF
```

**✅ Day 3-5 验收**
```bash
cd web && npm run build
# 期望：编译无错误

npm run dev &
sleep 5
curl -s http://localhost:3000/seller/login | grep -q "卖家登录" && echo "PASS 登录页" || echo "FAIL 登录页"
```

### Week 5 验收脚本

```bash
cat > scripts/week5_verify.sh << 'EOF'
#!/bin/bash
PASS=0; FAIL=0
GREEN='\033[0;32m'; RED='\033[0;31m'; NC='\033[0m'

check() {
    if eval "$2" | grep -q "$3"; then
        echo -e "${GREEN}✅ PASS${NC} $1"; PASS=$((PASS+1))
    else
        echo -e "${RED}❌ FAIL${NC} $1"; FAIL=$((FAIL+1))
    fi
}

check "结算 Service 编译" \
    "cd api && go build ./internal/accounting/... 2>&1 && echo OK" 'OK'

check "轮询器编译" \
    "cd api && go build ./internal/poller/... 2>&1 && echo OK" 'OK'

check "前端登录页可访问" \
    "curl -s http://localhost:3000/seller/login" '卖家登录'

check "前端注册页可访问" \
    "curl -s http://localhost:3000/seller/register" '卖家注册'

echo -e "\n通过 $PASS，失败 $FAIL"
[ $FAIL -eq 0 ] && echo -e "${GREEN}🎉 Week 5 全部验收通过！${NC}" || exit 1
EOF
chmod +x scripts/week5_verify.sh
bash scripts/week5_verify.sh
```

**✅ Week 5 完成标准**
- [ ] 结算 Service 编译，支持自动周期触发和手动申请
- [ ] DB 轮询器编译，每 30 秒检测账号状态变更
- [ ] 卖家前端 Next.js 项目可正常启动
- [ ] 卖家登录页和注册页渲染正确
- [ ] API 封装层（lib/api/seller.ts）完成

## 下周预告（Week 6）

Week 6 实现：
- 卖家前端完整（账号列表、添加账号、收益概览、结算历史）
- 买家前端启动（注册/登录页）

前置条件：Week 5 前端骨架通过
