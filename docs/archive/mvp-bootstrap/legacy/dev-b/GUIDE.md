# Dev-B 开发指南 · 每次开发前必读

> **身份约定：** 你是 AI 开发助手，负责 GateLink 平台的 Dev-B 部分（业务 API + 前端控制台 + 记账结算）。
> **铁律：永远不要修改或创建 `/Users/tvwoo/Projects/gatelink/engine/` 目录下的任何文件；如需核对接口，只允许只读查看必要实现。**

---

## 一、每次开发前必读文档（按顺序）

| 优先级 | 文档 | 路径 | 读取时机 |
|--------|------|------|--------|
| 🔴 必读 | Dev-A 联调交接记录 | `/Users/tvwoo/Projects/gatelink/docs/dev-b/DEV_A_HANDOFF.md` | 每次开发前都要读，了解联调边界、待确认事项和已知差异 |
| 🔴 必读 | 当前进度 | `/Users/tvwoo/Projects/gatelink/docs/dev-b/STATUS.md` | 每次开发前必读，了解当前停在哪 |
| 🟠 必读 | 当前 Phase 指令 | `/Users/tvwoo/Projects/gatelink/docs/dev-b/Phase{N}_Week{N}_Instructions.md` | 开始该 Phase 工作前读 |
| 🟡 参考 | Dev-B 主文档 | `/Users/tvwoo/Projects/gatelink/docs/dev-b/DEVB_MASTER.md` | 需要回顾完整需求和边界时读 |
| 🟡 参考 | 项目核心指南 | `/Users/tvwoo/Projects/gatelink/docs/PROJECT_GUIDE.md` | 需要回顾整体架构时读 |

> ⚠️ **冲突规则：** 当文档与本仓库 `engine/` 实现有出入时，**以 `engine/` 实现为准**，并将差异记录到 `docs/dev-b/DEV_A_HANDOFF.md`。

---

## 二、项目目录结构（权威定义）

> **所有代码必须写在以下目录内，不得在此结构之外创建文件。**
> 每个目录旁标注了对应的开发 Phase，写代码前先确认目录已存在。

```
/Users/tvwoo/Projects/gatelink/     ← GateLink 仓库根目录
│
├── engine/  (DEV-A，禁止触碰)
│
├── api/                            ← Dev-B 后端服务根目录
│   ├── cmd/
│   │   └── api/
│   │       └── main.go             ← Week 1 Day 1：服务入口
│   ├── internal/
│   │   ├── api/
│   │   │   └── router.go           ← Week 1 Day 4：路由组织
│   │   ├── auth/
│   │   │   ├── jwt.go              ← Week 1 Day 3：JWT 工具
│   │   │   ├── middleware.go       ← Week 1 Day 3：JWT 鉴权中间件
│   │   │   ├── apikey.go           ← Week 1 Day 3：买家 api_key 生成
│   │   │   ├── apikey_middleware.go← Week 1 Day 4：代理端鉴权中间件
│   │   │   └── apikey_cache.go     ← Week 4 Day 2：Redis 缓存优化
│   │   ├── seller/
│   │   │   ├── handler.go          ← Week 1 Day 3（注册/登录）+ Week 2 Day 2-4（业务接口）
│   │   │   └── service.go          ← Week 2 Day 2：卖家 DB 操作层
│   │   ├── buyer/
│   │   │   ├── handler.go          ← Week 1 Day 3（注册/登录）+ Week 3 Day 1（业务接口）
│   │   │   └── service.go          ← Week 3 Day 1：买家 DB 操作层
│   │   ├── proxy/
│   │   │   ├── handler.go          ← Week 3 Day 2：非流式代理
│   │   │   └── stream.go           ← Week 4 Day 1：流式 SSE 代理
│   │   ├── accounting/
│   │   │   ├── service.go          ← Week 3 Day 2：记账核心逻辑
│   │   │   └── settlement.go       ← Week 5 Day 1：结算周期
│   │   ├── engine/
│   │   │   └── client.go           ← Week 2 Day 1：Dev-A 内部接口客户端（唯一入口）
│   │   ├── poller/
│   │   │   └── account.go          ← Week 5 Day 2：账号状态 DB 轮询（替代 Pub/Sub）
│   │   ├── admin/
│   │   │   └── handler.go          ← Week 4 Day 3：管理员接口
│   │   ├── config/
│   │   │   └── config.go           ← Week 1 Day 1：环境变量加载
│   │   └── db/
│   │       ├── db.go               ← Week 1 Day 1：数据库连接
│   │       └── migrations/
│   │           └── 001_topup_records.sql ← Week 1 Day 2：Dev-B 新增表
│   ├── scripts/
│   │   ├── week1_verify.sh         ← Week 1 Day 5
│   │   ├── week2_verify.sh         ← Week 2 Day 5
│   │   ├── week3_verify.sh         ← Week 3 Day 5
│   │   ├── week4_verify.sh         ← Week 4 Day 5
│   │   ├── week5_verify.sh         ← Week 5 Day 5
│   │   ├── week6_verify.sh         ← Week 6 Day 5
│   │   ├── week7_verify.sh         ← Week 7 Day 5
│   │   ├── load_test.sh            ← Week 8 Day 3
│   │   └── mvp_verify.sh           ← Week 8 Day 4-5
│   ├── tests/                      ← 集成测试（各 Week 补充）
│   ├── go.mod                      ← github.com/ZachCharles666/gatelink/api
│   ├── Dockerfile                  ← Week 8 Day 1
│   └── .env.example                ← Week 1 Day 1
│
├── web/                            ← Dev-B 前端服务根目录
│   ├── app/
│   │   ├── seller/
│   │   │   ├── login/page.tsx      ← Week 5 Day 3-5
│   │   │   ├── register/page.tsx   ← Week 5 Day 3-5
│   │   │   ├── dashboard/page.tsx  ← Week 6 Day 1-2
│   │   │   ├── accounts/
│   │   │   │   └── add/page.tsx    ← Week 6 Day 1-2
│   │   │   ├── earnings/page.tsx   ← Week 6 Day 3
│   │   │   └── settlements/page.tsx← Week 6 Day 3
│   │   ├── buyer/
│   │   │   ├── login/page.tsx      ← Week 6 Day 4-5
│   │   │   ├── register/page.tsx   ← Week 6 Day 4-5
│   │   │   ├── dashboard/page.tsx  ← Week 7 Day 1-2
│   │   │   ├── topup/page.tsx      ← Week 7 Day 1-2
│   │   │   ├── usage/page.tsx      ← Week 7 Day 3
│   │   │   └── apikey/page.tsx     ← Week 7 Day 3
│   │   └── admin/
│   │       └── page.tsx            ← Week 7 Day 4-5
│   ├── components/                 ← 共用 UI 组件
│   └── lib/
│       └── api/
│           ├── client.ts           ← Week 5 Day 3：axios 封装
│           ├── seller.ts           ← Week 5 Day 3：卖家 API 方法
│           └── buyer.ts            ← Week 6 Day 4-5：买家 API 方法
│
└── docs/dev-b/                     ← 仓库内 Dev-B 规划与交接文档
    ├── GUIDE.md                    ← 本文件
    ├── STATUS.md                   ← 开发进度追踪
    ├── DEVB_MASTER.md
    └── Phase1~8_Instructions.md
```

---

## 三、文档位置索引

```
/Users/tvwoo/Projects/gatelink/
├── AGENTS.md                          AI 开发助手强指引
├── docs/PROJECT_GUIDE.md              项目整体架构指南
├── engine/                            ⛔ Dev-A 代码，只读参考
│   └── internal/api/                 🔴 内部 API 的实现级真相来源
└── docs/dev-b/                        ✅ Dev-B 规划与交接文档
    ├── GUIDE.md                       📌 本文件
    ├── STATUS.md                      📌 当前进度（每次必读）
    ├── DEV_A_HANDOFF.md               📌 联调交接与已知问题
    ├── DEVB_MASTER.md                 完整需求与边界
    ├── Phase1_Week1_Instructions.md   Week 1：项目初始化 + DB Schema + 鉴权
    ├── Phase2_Week2_Instructions.md   Week 2：Engine 客户端 + 卖家业务 API
    ├── Phase3_Week3_Instructions.md   Week 3：买家业务 API + 代理端点（非流式）
    ├── Phase4_Week4_Instructions.md   Week 4：代理端点（流式）+ 记账系统
    ├── Phase5_Week5_Instructions.md   Week 5：结算 + DB 轮询 + 卖家前端启动
    ├── Phase6_Week6_Instructions.md   Week 6：卖家前端完成 + 买家前端启动
    ├── Phase7_Week7_Instructions.md   Week 7：买家前端完成 + 管理后台
    └── Phase8_Week8_Instructions.md   Week 8：全链路压测 + MVP 验收
```

---

## 四、Dev-B 边界速查

### 我负责的服务
- `api` 服务（`:8080`，对外）：业务 API + 买家代理端点
- `web` 服务（`:3000`，对外）：Next.js 前端

### 我调用的 Dev-A 内部接口
> Base URL：`http://engine:8081/internal/v1`
> **所有调用必须通过 `api/internal/engine/client.go`，不得在其他包直接发 HTTP 请求**

| 接口 | 使用场景 |
|------|---------|
| `POST /dispatch` | 买家发起请求时转发到厂商 |
| `POST /audit` | dispatch 前内容审核 |
| `GET /pool/status` | Dashboard 展示各厂商可用账号数 |
| `GET /accounts/:id/health` | 卖家账号详情页健康度 |
| `POST /accounts/:id/verify` | 卖家添加账号时格式校验（MVP 仅格式校验）|
| `GET /accounts/:id/console-usage` | 平台消耗记录（非官方 Console，标注"平台记录"）|
| `GET /accounts/:id/diff` | 对账事件列表 |
| `GET /accounts/:id/events` | 账号健康事件历史 |

### 我不应该做的事
- ❌ 写入 `usage_records`（Dev-A 在 dispatch 后自动写入）
- ❌ 读取 `accounts.api_key_encrypted`
- ❌ 修改 `engine/` 下任何文件
- ❌ 直接发 HTTP 到 `engine:8081`（必须通过 `engine/client.go`）
- ❌ 在流式请求中间扣费（等流结束后才能记账）

### 与规划文档的关键差异（以 `engine/` 实现与 `DEV_A_HANDOFF.md` 为准）

| 差异点 | 规划说 | 实际情况 |
|--------|--------|---------|
| dispatch 响应格式 | 裸结构体 | 包裹在 `{code, msg, data}` 中 |
| audit 响应 | `passed: bool` | `safe: bool` + `level: int(0-4)` |
| pool/status 响应 | 含金额汇总 | 只有各厂商账号数量 |
| verify 接口 | 真实验证 Key | MVP 仅格式校验，展示用"格式检查通过" |
| console-usage | 官方 Console 数据 | 平台 usage_records 聚合，标注"平台记录" |
| diff 接口 | 结构化数值 | 对账事件列表（pass/fail） |
| Redis Pub/Sub | 4 个事件通道 | **未实现**，用 DB 轮询替代 |

---

## 五、记账逻辑速查（最敏感，不能出错）

```
每次 dispatch 成功后：

1. 从 dispatch 响应读取 cost_usd（Dev-A 已计算）
2. 查 vendor_pricing 表拿 platform_discount
3. 查 accounts 表拿 expected_rate（卖家期望回收率）

buyer_charged_usd = cost_usd × platform_discount
seller_earn_usd   = cost_usd × expected_rate
platform_earn_usd = buyer_charged_usd - seller_earn_usd

事务（原子，3步）：
  UPDATE buyers SET balance_usd = balance_usd - buyer_charged_usd
    WHERE id = ? AND balance_usd >= buyer_charged_usd  ← 防超扣，行数=0 则回滚返 402
  UPDATE sellers SET pending_earn_usd = pending_earn_usd + seller_earn_usd
    WHERE id = ?
  （usage_records 已由 Dev-A 写入，Dev-B 不再写）
```

---

## 六、DB Schema 变更约定

| 场景 | 流程 | 上限 |
|------|------|------|
| 新增字段 | Dev-B 写 migration SQL → Dev-A review → 合并 | 24h |
| 修改字段类型/约束 | Dev-B 写 migration + 影响分析 → 双方讨论 | 48h |
| 删除字段 | 面对面确认，评估所有引用点 | 面对面 |

**Dev-B 主写表：** `sellers`, `buyers`, `settlements`, `topup_records`
**Dev-B 只读表：** `accounts`, `usage_records`, `health_events`
**共同维护表：** `vendor_pricing`

---

## 七、环境变量（Dev-B 专属）

| 变量 | 说明 | 示例 |
|------|------|------|
| `JWT_SECRET` | JWT 签名密钥 | 随机 64 字节 hex |
| `ENGINE_INTERNAL_URL` | Dev-A 引擎地址 | `http://engine:8081` |
| `API_PORT` | 业务 API 端口 | `8080` |
| `DATABASE_URL` | 共用 PostgreSQL | `postgres://postgres:postgres@localhost:5432/tokenglide?sslmode=disable` |
| `REDIS_URL` | 共用 Redis | `redis://localhost:6379` |
| `NEXT_PUBLIC_API_URL` | 前端调用后端地址 | `http://localhost:8080` |

---

## 八、铁律重申

> **作为 Dev-B 的 AI 开发助手，你必须遵守以下规则，没有例外：**
>
> 1. **永远不要修改、创建 `/Users/tvwoo/Projects/gatelink/engine/` 下的任何文件；如需核对接口，只允许只读查看必要实现**
> 2. **永远不要写入 `usage_records` 表**
> 3. **永远不要接触 `accounts.api_key_encrypted` 字段**
> 4. **流式请求必须等流结束后才能记账**
> 5. **内部接口格式以本仓库 `engine/internal/api/` 实现为准，联调差异记录以 `docs/dev-b/DEV_A_HANDOFF.md` 为准**
> 6. **所有文件必须写在第二节定义的目录结构内**
