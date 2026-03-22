# Dev-B to Dev-A Handoff

本文件统一记录两类内容：
- 需要 Dev-A 接手、review、执行或回传结果的事项
- 会影响 Dev-B 与 Dev-A 合并、联调、共同构建或交付的问题

原则：
- 不修改 `DEV-A/` 目录中的任何文件
- 所有联调问题和交接待办都收敛到这一份文档
- 每条记录都要写清楚背景、影响、当前状态和后续动作

---

## 已解决问题

### 2026-03-21 · Issue 001 · API Go 版本被依赖图抬升到 1.25

- 状态：已解决
- 影响范围：`api` 服务的本地构建、Docker 构建、与 Dev-A 的联调环境对齐

#### 现象

在为 `api` 初始化依赖时，`go mod tidy` 曾把 [api/go.mod](/Users/tvwoo/Projects/gatelink/api/go.mod) 自动抬升到 `go 1.25.0`，并写入了一批过高版本的 indirect 依赖。

这会带来两个直接风险：
- Dev-B 的 `api` 构建环境要求高于项目规划中的 `Go 1.22+`
- [api/Dockerfile](/Users/tvwoo/Projects/gatelink/api/Dockerfile) 仍使用 `golang:1.22-alpine`，合并后容易出现“本地能跑、统一构建失败”的情况

#### 根因

第一次 `tidy` 时，主模块被写入了一批高版本 indirect 依赖。之后即使把 `gin` 降到 `v1.10.1`，Go 仍会按主模块中已显式要求的更高版本解依赖图，最终把 `go` 指令一并抬高。

#### 最终解决办法

已按以下方式修复：

1. 将 [api/go.mod](/Users/tvwoo/Projects/gatelink/api/go.mod) 的主版本目标恢复为 `go 1.22`
2. 保留直接依赖：
   - `github.com/gin-gonic/gin v1.10.1`
   - `github.com/joho/godotenv v1.5.1`
   - `github.com/rs/zerolog v1.34.0`
3. 删除主模块里误写入的高版本 indirect 依赖块
4. 重新执行：

```bash
cd /Users/tvwoo/Projects/gatelink/api
go mod tidy -go=1.22 -compat=1.22
```

#### 结果

修复后 [api/go.mod](/Users/tvwoo/Projects/gatelink/api/go.mod) 已回到 `Go 1.22` 兼容范围，关键 indirect 依赖也回落到与 `gin v1.10.1` 对齐的版本集合，可继续与 Dev-A 的 `Go 1.22` 规划保持一致。

---

## 待 Dev-A 接手事项

### 2026-03-21 · Issue 002 · 真实 `verify` 接口依赖共享 `accounts` 记录

- 状态：待双方联调确认
- 影响范围：Dev-B 的 `POST /api/v1/seller/accounts` live 联调
- 参考代码：
  - [accounts.go](/Users/tvwoo/Projects/gatelink/engine/internal/api/accounts.go)
  - [router.go](/Users/tvwoo/Projects/gatelink/engine/internal/api/router.go)

#### 现象

参考仓库中的真实 `POST /internal/v1/accounts/:id/verify` 实现，不使用 Dev-B 传入的 `api_key` 请求体，而是先按 `account_id` 去 `accounts` 表读取 `api_key_encrypted` 和 `vendor`，再继续校验。

这意味着：
- 如果 Dev-B 侧只是本地预创建一个内存 `account_id`
- 但该 `account_id` 没有先落到共享 PostgreSQL 的 `accounts` 表

那么真实 engine 会直接返回 `account not found`，live verify 无法成功。

#### 当前 Dev-B 处理

Dev-B 已按参考实现调整本地代码：
- `engine client` 已对齐真实 `diff/events/audit` 响应结构
- `seller AddAccount` 在收到真实 engine 的 `1004 account not found` 时，会明确返回：
  - `503`
  - `code=5000`
  - `msg="engine verify requires shared account persistence before live verification"`

也就是说，当前错误已经变成可识别的联调阻塞，而不是模糊的内部错误。

#### 需要 Dev-A / 联调阶段确认的事项

1. `accounts` 表中的预创建动作最终由哪一侧负责
2. Dev-B 在调用 `verify` 前，是否应先把 account 记录写入共享 PostgreSQL
3. 如果由 Dev-B 先写入共享表，真实字段最小集和状态流转应如何约定
4. `api_key_encrypted` 的落库责任边界如何处理

### 2026-03-21 · Day 2 · `topup_records` migration 待接手

- 状态：本地 GateLink PostgreSQL 已验证；正式共享环境待 Dev-A 接入统一 migration 链
- 来源文件：[001_topup_records.sql](/Users/tvwoo/Projects/gatelink/api/internal/db/migrations/001_topup_records.sql)
- 背景：Dev-B 已完成 Phase 1 · Week 1 · Day 2 的 migration 编写，但按照项目流程，该 migration 需要由 Dev-A review 后合并到其统一 migration 执行链路，再在共享 PostgreSQL 中执行

#### 需要 Dev-A 执行的动作

1. Review `api/internal/db/migrations/001_topup_records.sql`
2. 将该 migration 合并到 Dev-A 负责执行的 migration 流程
3. 在共享 PostgreSQL 中执行 migration
4. 回传 `topup_records` 表结构验收结果

#### Dev-B 期望的回传内容

- `topup_records` 已成功创建
- 表结构与以下约束一致：
  - `buyer_id` 外键关联 `buyers(id)`
  - `amount_usd` 为 `DECIMAL(12,4)`，且 `CHECK(amount_usd > 0)`
  - `network` 仅允许 `TRC20` / `ERC20`
  - `status` 仅允许 `pending` / `confirmed` / `rejected`
  - 已创建索引 `idx_topup_buyer`
  - 已创建部分索引 `idx_topup_status`

#### Day 2 验收目标

可接受的验收回传示例：

```bash
docker exec -it postgres psql -U postgres -d tokenglide -c "\d topup_records"
```

或等价的表结构输出截图 / 文本。

### 2026-03-21 · Issue 003 · Week 3 live 联调曾存在两个真实阻塞

- 状态：本地 Week 8 验收已部分清理；正式共享环境仍需 Dev-A 关注可用账号池
- 影响范围：`POST /v1/chat/completions` 的真实端到端打通

#### 已确认事实

本地真实环境已启动：
- `engine` 已在 `http://localhost:8081` 健康运行
- `api` 已在 `http://localhost:8080` 运行

实际 live 结果如下：
- `POST /v1/chat/completions` 无 API Key 返回 `401`
- `GET /v1/models` 返回正常模型列表
- 带有效 API Key 的 `POST /v1/chat/completions` 当前返回 `1005 insufficient balance`
- 直接调用真实 `POST /internal/v1/dispatch` 当前返回 `4001 no available account in pool`

#### 结论

当前 live 未全通，不是因为服务没启动，而是因为同时存在两层真实阻塞：

1. Dev-B 当前 buyer 仍是本地内存 service，注册后 `balance_usd = 0`
2. 真实 engine 当前账号池为空，直接 dispatch 也没有可用账号

#### 需要继续对齐的事项

1. Week 3 / Week 4 联调阶段，买家测试余额的合法来源是什么
2. 真实 engine 账号池需要由谁、以什么方式提供至少一个 `active` 账号用于联调
3. 在共享 DB 版本切换前，Dev-B 是否允许使用明确标注的联调测试数据进行手工充值 / 测试账号预置

---

## 2026-03-22 · 最终交付摘要

- 状态：Dev-B Phase 1 - Phase 8 已全部完成
- 范围：`api`、`web`、Docker 构建资产、压测脚本、MVP 验收脚本

### Dev-B 已完成的最终交付物

- 后端业务与代理端：
  - 卖家 / 买家鉴权、业务 API、管理员接口、非流式代理、流式代理、记账、结算、轮询器
- 前端控制台：
  - 卖家 5 页、买家 5 页、管理后台 1 页
- 生产构建与验收脚本：
  - [api/Dockerfile](/Users/tvwoo/Projects/gatelink/api/Dockerfile)
  - [web/Dockerfile](/Users/tvwoo/Projects/gatelink/web/Dockerfile)
  - [docker-compose-devb.yml](/Users/tvwoo/Projects/gatelink/api/docker-compose-devb.yml)
  - [load_test.sh](/Users/tvwoo/Projects/gatelink/api/scripts/load_test.sh)
  - [mvp_verify.sh](/Users/tvwoo/Projects/gatelink/api/scripts/mvp_verify.sh)
- 性能优化：
  - 新增 [apikey_cache.go](/Users/tvwoo/Projects/gatelink/api/internal/auth/apikey_cache.go)
  - `BuyerAPIKeyMiddleware` 已支持可选 Redis 缓存
  - 买家重置 API Key 时已做缓存失效

### 本地最终验收结果

- 真实 Docker 构建已通过：
  - `docker build -t gatelink-api ./api`
  - `docker build -t gatelink-web ./web`
- 真实压测已通过：
  - 使用系统自带 `ab` 跑通 `/health`、`/v1/models`、`/api/v1/buyer/balance`
  - 三项均为 `0 failed requests`
  - 报告路径：`/tmp/load_test_report.txt`
- MVP 总验收已通过：
  - `MVP_VERIFY_MODE=test bash api/scripts/mvp_verify.sh`
  - `MVP_VERIFY_MODE=live bash api/scripts/mvp_verify.sh`
  - live 结果：`passed 23, failed 0`

### 本地 live 环境已确认的事实

- 本地 GateLink PostgreSQL 中已存在：
  - `buyers`
  - `sellers`
  - `settlements`
  - `topup_records`
- 为完成 live DB 校验，这轮已将 [001_topup_records.sql](/Users/tvwoo/Projects/gatelink/api/internal/db/migrations/001_topup_records.sql) 执行到本地 GateLink PostgreSQL

### 仍需 Dev-A 在正式共享环境确认的事项

1. 将 [001_topup_records.sql](/Users/tvwoo/Projects/gatelink/api/internal/db/migrations/001_topup_records.sql) 正式并入 Dev-A 维护的 migration 链，而不是只停留在本地验证环境
2. 明确 `accounts` 预创建与 `verify` 前置写库责任，尤其是 `api_key_encrypted` 的边界归属
3. 如果进入正式共享环境的成功 dispatch / 成功流式联调阶段，仍需 Dev-A 提供至少一个可用 `active` 账号进入真实池
4. 将本地 Week 8 验收里使用的 GateLink 环境变量命名（如 `GateLink/GateLink/password`）与正式共享环境统一，避免 compose / `.env` 再次分叉

### 给 Dev-A 的一句话结论

Dev-B 规划内的代码、前端、Docker、压测与本地 live MVP 验收都已经完成；后续重点不再是 Dev-B 补功能，而是 Dev-A 侧把共享 migration、共享 `accounts` 工作流和正式联调环境与这套产物对齐。
