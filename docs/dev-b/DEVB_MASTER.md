# Dev-B 主文档

> AI Credits 授权调度平台 · 业务 API + 前端控制台 + 记账结算
> **本文档以本仓库 `engine/` 实现和 `docs/dev-b/DEV_A_HANDOFF.md` 为准；两者有出入时以 `engine/` 实现为准。**

---

## 一、基本信息

| 属性 | 内容 |
|------|------|
| 负责人 | Dev-B |
| 后端语言 | Go 1.22+ / Gin |
| 前端 | Next.js 14 + TypeScript |
| 服务名 | `api`（:8080 对外）+ `web`（:3000 对外）|
| 依赖引擎 | `http://engine:8081`（Dev-A，只读调用，不可修改）|
| 总工期 | 8 周 |

---

## 二、项目目录结构

```
api/
├── cmd/api/main.go
├── internal/
│   ├── auth/                 鉴权（JWT + api_key）
│   ├── seller/               卖家业务逻辑
│   ├── buyer/                买家业务逻辑
│   ├── proxy/                买家代理端点（/v1/*）
│   ├── accounting/           记账（扣费事务）
│   │   └── settlement.go     结算周期
│   ├── engine/
│   │   └── client.go         Dev-A 内部接口客户端
│   ├── poller/
│   │   └── account.go        账号状态变更 DB 轮询（替代 Pub/Sub）
│   └── db/                   数据库操作层
├── Dockerfile
└── docker-compose.yml        追加 api/web 服务（PR 给 Dev-A merge）

web/
├── app/
│   ├── seller/               卖家控制台（5 页）
│   └── buyer/                买家控制台（5 页）
├── components/
└── lib/api/                  fetch 封装
```

---

## 三、数据库责任

### Dev-B 主写表

| 表 | 说明 |
|----|------|
| `sellers` | 卖家信息、收款地址、收益金额 |
| `buyers` | 买家信息、平台 api_key、余额 |
| `settlements` | 结算记录 |
| `topup_records` | 充值申请记录（Dev-B 新建此表）|

### Dev-B 只读表

| 表 | 说明 |
|----|------|
| `accounts` | 只读 status、seller_id、health_score 等字段；`api_key_encrypted` 绝对不读 |
| `usage_records` | Dev-A 写入，Dev-B 只读用于账单展示；Dev-B **不写入** |
| `health_events` | Dev-A 写入，Dev-B 只读用于展示 |

### 共同维护

| 表 | 说明 |
|----|------|
| `vendor_pricing` | 厂商定价，Dev-B 读取用于计费和前端展示 |

### 完整 DDL（Dev-B 额外建的表）

```sql
-- topup_records：充值申请记录
CREATE TABLE topup_records (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id     UUID        NOT NULL REFERENCES buyers(id),
    amount_usd   DECIMAL(12,4) NOT NULL,
    tx_hash      VARCHAR(100) UNIQUE NOT NULL,
    network      VARCHAR(20)  NOT NULL,    -- TRC20 / ERC20
    status       VARCHAR(20)  NOT NULL DEFAULT 'pending'
                 CHECK(status IN ('pending','confirmed','rejected')),
    confirmed_at TIMESTAMPTZ,
    rejected_at  TIMESTAMPTZ,
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 四、调用 Dev-A 的内部接口

> Base URL：`http://engine:8081/internal/v1`
> 所有响应格式：`{"code": 0, "msg": "ok", "data": {...}}`

### 4.1 POST /dispatch（核心）

```json
// 请求
{
  "buyer_id": "uuid",
  "vendor": "anthropic",
  "model": "claude-sonnet-4-6",
  "messages": [{"role": "user", "content": "hello"}],
  "stream": false,
  "max_tokens": 1024,
  "buyer_charge_rate": 1.10
}

// 非流式成功响应（data 字段）
{
  "response": { ...OpenAI ChatCompletion 格式... },
  "account_id": "uuid",
  "vendor": "anthropic",
  "cost_usd": 0.000450,
  "input_tokens": 10,
  "output_tokens": 20
}

// 流式：Content-Type: text/event-stream，直接 SSE 透传
// 流式场景下无 JSON 响应体，Dev-B 通过其他方式获取计费信息（见 Phase 4）
```

**错误码：**
- `4001` / HTTP 503：无可用账号
- `4003` / HTTP 400：内容审核拒绝
- `4004` / HTTP 503：厂商限流
- `5001` / HTTP 502：厂商请求失败

### 4.2 POST /audit

```json
// 请求（messages 是字符串数组，不是 objects）
{ "messages": ["消息内容1", "消息内容2"], "buyer_id": "uuid" }

// 通过：{ "safe": true, "level": 0, "reason": "" }
// 拒绝：HTTP 400, code=4003
// 拦截阈值：level >= 3
```

### 4.3 GET /pool/status

```json
// data 字段（只有账号数量，无金额）
{ "pool_counts": { "anthropic": 5, "openai": 3, ... } }
// 如需金额，Dev-B 直接聚合查 accounts 表
```

### 4.4 GET /accounts/:id/health

```json
{ "account_id": "uuid", "health_score": 80, "status": "active", "recent_events": [...] }
```

### 4.5 POST /accounts/:id/verify（MVP 降级：仅格式校验）

```json
{ "account_id": "uuid", "valid": true, "error_msg": "" }
// ⚠️ valid: true 不代表 Key 真实有效，仅格式校验
// 展示时用"格式检查通过"，不用"账号已验证"
```

### 4.6 GET /accounts/:id/console-usage

```json
// ⚠️ 数据来源是平台 usage_records 聚合，非厂商官方 Console
// 前端展示时标注"平台记录"，不要写"官方 Console 数据"
{ "records": [{ "date": "2026-03-20", "total_cost_usd": 1.234, ... }] }
```

### 4.7 GET /accounts/:id/diff

```json
// 返回对账事件列表，不是结构化数值
{ "diffs": [{ "type": "reconcile_pass", "detail": {"diff_pct": 0.02}, ... }] }
```

### 4.8 GET /accounts/:id/events?limit=50

```json
{ "events": [{ "event_type": "success", "score_delta": 1, "score_after": 81, ... }] }
```

---

## 五、Dev-B 账号状态感知（替代 Pub/Sub）

> ⚠️ Dev-A 的 Redis Pub/Sub 事件通知（account.suspended 等）**尚未实现**。
> Dev-B 改用 DB 轮询方案：

```go
// internal/poller/account.go
// 每 30 秒查询 accounts 表中 updated_at > last_check AND status IN ('suspended', 'active')
// 感知账号状态变更，刷新卖家控制台展示
```

---

## 六、外部 API 完整清单

### 卖家端（需 seller JWT）

```
POST   /api/v1/seller/auth/register
POST   /api/v1/seller/auth/login
POST   /api/v1/seller/auth/refresh

GET    /api/v1/seller/accounts
POST   /api/v1/seller/accounts
GET    /api/v1/seller/accounts/:id
GET    /api/v1/seller/accounts/:id/usage
PATCH  /api/v1/seller/accounts/:id/authorization
DELETE /api/v1/seller/accounts/:id/authorization

GET    /api/v1/seller/earnings
GET    /api/v1/seller/settlements
POST   /api/v1/seller/settlements/request

GET    /api/v1/seller/profile
PATCH  /api/v1/seller/profile
```

### 买家端（需 buyer JWT）

```
POST   /api/v1/buyer/auth/register
POST   /api/v1/buyer/auth/login
POST   /api/v1/buyer/auth/refresh

GET    /api/v1/buyer/balance
GET    /api/v1/buyer/usage
POST   /api/v1/buyer/topup
GET    /api/v1/buyer/topup/records
POST   /api/v1/buyer/apikeys/reset
```

### 代理端（买家 api_key 鉴权，不用 JWT）

```
POST   /v1/chat/completions   主代理，支持 stream
POST   /v1/completions        旧版，部分厂商不支持返回 404
POST   /v1/embeddings
GET    /v1/models             直接读 vendor_pricing 表，不调 Dev-A
```

### 管理后台（仅内网，需 admin JWT）

```
GET    /api/v1/admin/topup/pending
POST   /api/v1/admin/topup/:id/confirm
POST   /api/v1/admin/topup/:id/reject
GET    /api/v1/admin/settlements/pending
POST   /api/v1/admin/settlements/:id/pay
POST   /api/v1/admin/accounts/:id/force-suspend
```

---

## 七、记账核心逻辑

```
dispatch 成功后（非流式：立即；流式：等流结束）：

cost_usd ← dispatch 响应中的 cost_usd（Dev-A 已计算）

buyer_charged_usd = cost_usd × vendor_pricing.platform_discount
seller_earn_usd   = cost_usd × accounts.expected_rate
platform_earn_usd = buyer_charged_usd - seller_earn_usd

事务（3 步原子）：
  1. UPDATE buyers
       SET balance_usd = balance_usd - buyer_charged_usd,
           total_consumed_usd = total_consumed_usd + buyer_charged_usd
       WHERE id = ? AND balance_usd >= buyer_charged_usd   -- 行数=0 则回滚返 402
  2. UPDATE sellers
       SET pending_earn_usd = pending_earn_usd + seller_earn_usd
       WHERE id = ?
  3. （不写 usage_records，Dev-A 已经写了）

注意：usage_records 由 Dev-A 在 dispatch 完成后写入，Dev-B 只负责更新 buyers 和 sellers。
```

---

## 八、任务与工时

| Phase | Week | 主要任务 | 工时 |
|-------|------|---------|------|
| Phase 1 | Week 1 | 项目初始化 + DB Schema + 鉴权系统 | 6 天 |
| Phase 2 | Week 2 | Engine 客户端 + 卖家业务 API | 5 天 |
| Phase 3 | Week 3 | 买家业务 API + 代理端点（非流式）| 6 天 |
| Phase 4 | Week 4 | 代理端点（流式）+ 记账系统 | 5 天 |
| Phase 5 | Week 5-6 | 结算 + DB 轮询 + 卖家前端（5 页）| 10 天 |
| Phase 6 | Week 6-7 | 买家前端（5 页）| 6 天 |
| Phase 7 | Week 7 | 管理后台 + 充值流程 | 4 天 |
| Phase 8 | Week 8 | 全链路压测 + MVP 验收 | 2 天 |
| **合计** | | | **44 天** |

---

## 九、前端技术栈

| 技术 | 选型 |
|------|------|
| 框架 | Next.js 14 App Router + TypeScript |
| 图表 | Recharts |
| UI | Tailwind CSS + shadcn/ui |
| 数据获取 | TanStack Query（React Query）|
| 表单 | react-hook-form + zod |
| HTTP | Axios（封装在 `lib/api/`）|
