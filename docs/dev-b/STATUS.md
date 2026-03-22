# Dev-B 开发进度

## 当前状态

**当前 Phase：Phase 8 · Week 8**
**当前 Day：Phase 8 · Week 8 · 全部完成**

---

## 进度追踪

### Phase 1 · Week 1：项目初始化 + DB Schema + 鉴权
- [x] Day 1：Go 项目骨架 + Docker 配置
- [x] Day 2：DB Schema 对齐 + topup_records 表（Dev-B 已完成，待 Dev-A review / DB 验收）
- [x] Day 3：JWT 工具 + 卖家/买家注册登录
- [x] Day 4：代理端 api_key 鉴权中间件 + 路由组织
- [x] Day 5：Week 1 全面验收（`bash scripts/week1_verify.sh`）

### Phase 2 · Week 2：Engine 客户端 + 卖家业务 API
- [x] Day 1：`api/internal/engine/client.go`
- [x] Day 2-3：卖家 5 个业务接口
- [x] Day 4：结算接口 + 联调准备
- [ ] Day 5：Week 2 全面验收（`bash scripts/week2_verify.sh`，本地已通过，真实联调待完成）

### Phase 3 · Week 3：买家业务 API + 代理端点（非流式）
- [x] Day 1：买家余额 + 充值接口
- [x] Day 2：代理 Handler + 记账 Service
- [x] Day 3：路由整合 + 代理鉴权中间件
- [x] Day 4：非流式全链路联调（本地已通过；live 已开始）
- [ ] Day 5：Week 3 全面验收（`bash scripts/week3_verify.sh`）

### Phase 4 · Week 4：流式 SSE + 记账系统
- [x] Day 1-2：流式 SSE 代理
- [x] Day 3：管理员接口（充值审核 + 结算付款）
- [x] Day 4-5：Week 4 全面验收（`bash scripts/week4_verify.sh`）

### Phase 5 · Week 5：结算系统 + DB 轮询 + 卖家前端启动
- [x] Day 1：结算 Service
- [x] Day 2：DB 轮询模块（`internal/poller/account.go`）
- [x] Day 3-5：Next.js 初始化 + 卖家登录/注册页
- [x] Week 5 验收（`bash scripts/week5_verify.sh`）

### Phase 6 · Week 6：卖家前端完成 + 买家前端启动
- [x] Day 1-2：卖家账号列表 + 添加账号页
- [x] Day 3：卖家收益概览 + 结算历史页
- [x] Day 4-5：买家登录/注册页
- [x] Week 6 验收（`bash scripts/week6_verify.sh`）

### Phase 7 · Week 7：买家前端完成 + 管理后台
- [x] Day 1-2：买家控制台主页 + 充值页
- [x] Day 3：买家用量明细 + API Key 管理页
- [x] Day 4-5：管理后台（充值审核 + 结算付款）
- [x] Week 7 验收（`bash scripts/week7_verify.sh`）

### Phase 8 · Week 8：全链路压测 + MVP 验收
- [x] Day 1：生产 Dockerfile + Docker Compose
- [x] Day 2：性能优化
- [x] Day 3：压测（`bash scripts/load_test.sh` 已完成并通过真实 `ab` 压测）
- [x] Day 4-5：MVP 全面验收（`bash scripts/mvp_verify.sh` test mode / live mode 均已通过）

---

## 已知问题 / 待确认

- 2026-03-21：`api` 初始化依赖时曾被错误解析到 `go 1.25.0`，会影响与 Dev-A 的 Go 1.22 规划对齐及后续 Docker/联调构建。已通过将 `github.com/gin-gonic/gin` 固定为 `v1.10.1`、清理误写入的高版本 indirect 依赖、重新执行 `go mod tidy -go=1.22 -compat=1.22` 解决。详见 `docs/dev-b/DEV_A_HANDOFF.md`。
- 2026-03-21：`topup_records` migration 已在 Dev-B 侧完成，但尚未经过 Dev-A review，也尚未在共享 PostgreSQL 中执行。此前本机 Docker daemon 一度不可用；当前 Docker 已恢复，但共享 PostgreSQL / Dev-A 验收仍未完成。已在 `docs/dev-b/DEV_A_HANDOFF.md` 中记录需要 Dev-A 执行的动作和回传项。
- 2026-03-21：Week 2 的 seller API 已通过本地 `go build`、`go test` 和 `httptest` 路由验证，但真实 `engine:8081` 联调和共享 PostgreSQL 联调仍待 Dev-A 环境可用后完成。
- 2026-03-21：参考 GitHub `engine` 真实实现后，已确认 `POST /internal/v1/accounts/:id/verify` 依赖共享 `accounts` 记录，不能仅靠 Dev-B 本地内存 `account_id` 完成 live verify。Dev-B 已对齐 `diff/events/audit` 响应结构，并将该阻塞记录到 `docs/dev-b/DEV_A_HANDOFF.md`。
- 2026-03-21：Week 3 / Week 4 live 联调已开始，真实 `engine` 与 `api` 都已启动。管理员充值确认已清掉买家 `balance_usd = 0` 的阻塞，当前非流式与流式 live 请求都会继续打到真实 `engine`；但真实 `engine` 仍返回 `4001 no available account in pool`，因此当前 live 未全通的唯一阻塞是“账号池为空”。
- 2026-03-21：Week 5 Day 2 的 `poller/account.go` 当前基于 Dev-B 现有内存 `seller.Service` 做状态快照轮询，而不是 Phase 文档里示意的共享 PostgreSQL 版 `pgx` 轮询器。这是对当前架构现实的对齐，便于后续在共享 DB 接通后再替换成真正的数据库轮询实现。
- 2026-03-21：Week 5 前端验收已通过 `npm run build` + 干净端口上的 `next start` 检查。手动长期运行的 `next dev` 在复用旧 `.next` 热更新缓存时偶发出现 `Cannot find module './819.js'`；当前可重复验证路径是 `bash api/scripts/week5_verify.sh`，脚本会自行起一个干净的前端进程做检查。
- 2026-03-22：Week 6 买家前端认证页已按真实 `buyer` handler 对齐为“邮箱或手机号 + 密码”模式，而不是 Phase 文档示例里的验证码登录。这是遵循本地实现的必要调整，Week 7 其余买家页面也应继续沿用这一真实认证形状。
- 2026-03-22：Week 7 管理后台页已按真实 `admin` handler 对齐；其中账号强制暂停接口真实需要 `account_id`，不能直接用 `seller_id` 代替，前端已改为手动输入真实账号 ID 后再执行。
- 2026-03-22：Week 8 Day 3 的 `load_test.sh` 已支持 `ab/hey` 双路径。本机未安装 `hey`，但已使用系统自带 `ab` 完成真实压测，并输出 `/tmp/load_test_report.txt`。
- 2026-03-22：Week 8 Day 4-5 的 `mvp_verify.sh` 已通过 `MVP_VERIFY_MODE=test` 与 `MVP_VERIFY_MODE=live` 双重验收。为完成 live DB 校验，这轮已将 `api/internal/db/migrations/001_topup_records.sql` 执行到本地 GateLink PostgreSQL；当前 `buyers`、`sellers`、`settlements`、`topup_records` 已在本地 live 库中存在。

---

## 上次停在哪

已完成 Phase 1 · Week 1：
- `api` 最小可运行骨架已建立，`/health` 已通过本地验收
- `topup_records` migration 已完成并记录到交接文档，待 Dev-A review / 共享 PostgreSQL 验收
- JWT、卖家/买家注册登录、路由组织与 `week1_verify.sh` 已完成本地验收

已完成 Phase 2 · Week 2 · Day 1-4：
- `api/internal/engine/client.go` 已完成并通过编译
- seller 侧已实现添加账号、调整授权、撤回授权、账号用量、收益概览
- seller 侧已实现账号列表、账号详情、结算历史
- Week 2 当前已通过本地 `go build`、`go test` 和 `httptest` 路由验证

Phase 2 · Week 2 · Day 5 当前状态：
- `api/scripts/week2_verify.sh` 已完成并通过本地 fallback 验收
- 已引入参考 GitHub `engine` 仓库进行只读核对，并确认真实内部接口实现
- Dev-B 已按真实 `engine` 对齐 `engine client` 的 `diff/events/audit` 响应结构
- `seller AddAccount` 在真实 `verify` 缺少共享 `accounts` 记录时，已改为返回明确的联调阻塞错误，而不是模糊 `500`

已完成 Phase 3 · Week 3 · Day 1：
- buyer 侧已实现余额查询、充值申请、充值记录、用量记录、API Key 重置
- 已新增 `api/internal/buyer/service.go`
- `go build ./internal/buyer/...`、buyer 定向测试和全量 `go test ./...` 已通过
- Week 2 的本地验收脚本已回归通过，确认本轮 buyer 改动没有破坏既有 seller 能力

已完成 Phase 3 · Week 3 · Day 2：
- 已新增 `api/internal/proxy/handler.go` 和 `api/internal/accounting/service.go`
- 本地已实现非流式代理骨架、余额预检、审核前置和 dispatch 后内存记账
- `go build ./internal/proxy/...`、`go build ./internal/accounting/...`、定向测试和全量 `go test ./...` 已通过
- 当前仍保持“本地可验证，真实 `engine:8081` / 共享 DB 联调待外部条件”的推进策略

已完成 Phase 3 · Week 3 · Day 3：
- 已在 `api/internal/api/router.go` 中接入 `/v1/chat/completions` 和 `/v1/models`
- `api/cmd/api/main.go` 已实例化 `accounting.Service` 和 `proxy.Handler`
- 已新增 `api/internal/api/router_test.go`，验证代理端点已挂入总路由且走 API Key 鉴权
- `go test ./internal/api -run TestSetupRoutesMountsProxyEndpointsWithAPIKeyAuth -v`、全量 `go build ./...` 和 `go test ./...` 已通过

已完成 Phase 3 · Week 3 · Day 4：
- 已新增 `api/internal/api/router_integration_test.go`，覆盖总路由级非流式代理链路
- 本地端到端链路已验证：买家注册、API Key 鉴权、模型列表、审核拦截、dispatch 成功后记账、无可用账号分支
- 真实 `engine` 已在 `http://localhost:8081` 跑起，真实 `api` 也已在 `http://localhost:8080` 跑起
- live smoke 实测结果：
  - `/v1/chat/completions` 无 API Key 返回 `401`
  - `/v1/models` 返回正常模型列表
  - 带 API Key 的代理请求当前返回 `1005 insufficient balance`
  - 直接调用真实 `engine /internal/v1/dispatch` 返回 `4001 no available account in pool`

已完成 Phase 3 · Week 3 · Day 5 准备：
- 已新增 `api/scripts/week3_verify.sh`
- `WEEK3_VERIFY_MODE=test` 已通过本地验收
- `WEEK3_VERIFY_MODE=live` 已能准确报告当前真实阻塞，而不是脚本错误

已完成 Phase 4 · Week 4 · Day 1-2：
- 已新增 `api/internal/proxy/stream.go`
- `stream=true` 已不再返回 `501`，而是先走余额预检和审核，再进入真实流式分支
- `api/internal/engine/client.go` 已补充流式调度入口，仍保持为 Dev-B 调用 `engine` 的唯一合法入口
- 已新增流式相关定向测试，覆盖 SSE 透传、流结束后记账、无可用账号返回 `503`
- 新版 live 容器在 `http://localhost:8083` 的 smoke test 已确认：`stream=true` 当前会返回 `503 no available account`，说明请求已走到真实流式分支而非占位逻辑

已完成 Phase 4 · Week 4 · Day 3：
- 已新增 `api/internal/admin/handler.go`
- 已接入管理员接口：待审核充值列表、充值确认/拒绝、待结算列表、结算付款、账号强制暂停
- 已新增 `api/internal/admin/handler_test.go`
- 本地全量 `go test ./...` 已通过
- live smoke test 已确认：管理员确认充值后，买家余额可从 `0` 正常变为 `100`

已完成 Phase 4 · Week 4 · Day 4-5 准备：
- 已新增 `api/scripts/week4_verify.sh`
- `WEEK4_VERIFY_MODE=test` 已通过本地验收
- `WEEK4_VERIFY_MODE=live` 已通过真实 smoke 验收，并准确报告当前唯一 live 阻塞是 `engine` 无可用账号池

已完成 Phase 5 · Week 5 · Day 1：
- 已新增 `api/internal/accounting/settlement.go`
- 已新增卖家主动申请结算接口 `POST /api/v1/seller/settlements/request`
- 已新增结算相关定向测试，覆盖申请结算、最低提现门槛、批量结算周期
- `go build ./internal/accounting/...`、定向测试和全量 `go test ./...` 已通过

已完成 Phase 5 · Week 5 · Day 2：
- 已新增 `api/internal/poller/account.go`
- 已在 `api/cmd/api/main.go` 中启动账号状态轮询器
- 当前轮询器基于内存版 `seller.Service` 账号快照工作，检测到 `suspended/active/revoked/expired` 状态变化时会记录日志
- `go build ./internal/poller/...`、`go test ./internal/poller -v` 和全量 `go test ./...` 已通过

已完成 Phase 5 · Week 5 · Day 3-5：
- 已在 `web/` 下初始化最小 Next.js 14 + TypeScript 工程
- 已新增卖家前端 API 封装层：`web/lib/api/client.ts`、`web/lib/api/seller.ts`
- 已新增卖家登录页和注册页：`web/app/seller/login/page.tsx`、`web/app/seller/register/page.tsx`
- 已新增 `api/scripts/week5_verify.sh`
- `cd web && npm run build` 已通过
- `WEB_BASE_URL=http://127.0.0.1:3100 WEB_HOST=127.0.0.1 WEB_PORT=3100 bash api/scripts/week5_verify.sh` 已通过，结果为 `passed 5, failed 0`

已完成 Phase 6 · Week 6：
- 已新增卖家账号控制台页与添加账号页：`web/app/seller/dashboard/page.tsx`、`web/app/seller/accounts/add/page.tsx`
- 已新增卖家收益概览页与结算历史页：`web/app/seller/earnings/page.tsx`、`web/app/seller/settlements/page.tsx`
- 已新增买家登录页与注册页：`web/app/buyer/login/page.tsx`、`web/app/buyer/register/page.tsx`
- 已新增买家 API 封装层：`web/lib/api/buyer.ts`
- 已新增 `api/scripts/week6_verify.sh`
- `WEB_BASE_URL=http://127.0.0.1:3200 WEB_HOST=127.0.0.1 WEB_PORT=3200 bash api/scripts/week6_verify.sh` 已通过，结果为 `passed 7, failed 0`

已完成 Phase 7 · Week 7：
- 已新增买家控制台主页与充值页：`web/app/buyer/dashboard/page.tsx`、`web/app/buyer/topup/page.tsx`
- 已新增买家用量明细页与 API Key 管理页：`web/app/buyer/usage/page.tsx`、`web/app/buyer/apikey/page.tsx`
- 已新增管理后台页：`web/app/admin/page.tsx`
- 已新增 `api/scripts/week7_verify.sh`
- `bash api/scripts/week7_verify.sh` 已通过，结果为 `passed 6, failed 0`

下一步：
- 已完成 Phase 8 · Week 8 · Day 1：
  - 已更新 `api/Dockerfile`、`api/docker-compose-devb.yml`
  - 已新增 `web/Dockerfile`
  - 已将 `web/next.config.mjs` 调整为 `output: "standalone"`
  - 已通过真实 `docker build -t gatelink-api ./api` 和 `docker build -t gatelink-web ./web`
- 已完成 Phase 8 · Week 8 · Day 2：
  - 已新增 `api/internal/auth/apikey_cache.go`
  - 已在 `api/cmd/api/main.go` 以可选 Redis 方式接入 Buyer API Key 缓存
  - 买家重置 API Key 时已增加缓存失效
  - `go build ./...`、`go test ./...` 与买家定向测试已通过
- 已完成 Phase 8 · Week 8 · Day 3：
  - 已新增并验证 `api/scripts/load_test.sh`
  - 已使用系统自带 `ab` 完成真实压测
  - `/health`、`/v1/models`、`/api/v1/buyer/balance` 均为 0 failed requests
  - 压测报告已输出到 `/tmp/load_test_report.txt`
- 已完成 Phase 8 · Week 8 · Day 4-5：
  - 已新增并验证 `api/scripts/mvp_verify.sh`
  - `MVP_VERIFY_MODE=test bash api/scripts/mvp_verify.sh` 已通过，结果为 `passed 10, failed 0`
  - `MVP_VERIFY_MODE=live bash api/scripts/mvp_verify.sh` 已通过，结果为 `passed 23, failed 0`
  - 已修复 `week1_verify.sh` 的清理逻辑，避免聚合验收时残留临时 API 进程
  - 为 live DB 校验补齐了 `topup_records` 本地 migration

最终状态：
- Dev-B 规划内 Phase 1 - Phase 8 已全部完成
- 本地代码验收、Docker 构建、真实压测、MVP live 验收均已跑通
- Week 2 / Week 3 中那些与 Dev-A 共享环境强相关的历史阻塞，已在本地 Week 8 live 验收里尽可能打通；后续若进入正式交付阶段，重点转向与 Dev-A 合库/共享环境一致性确认
