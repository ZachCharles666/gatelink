# GateLink 项目状态与任务跟踪

> 团队共用状态板。
> 自 2026-03-23 起，Dev-A / Dev-B 历史边界不再作为日常开发规则，统一按本文件和仓库当前代码推进。

## 当前状态

- 当前协作模式：`dev -> feat/fix/chore -> dev -> main`
- `main`：禁止直接 push，只接受来自 `dev` 的 PR
- `dev`：日常集成分支
- 日常开发起点：

```bash
git checkout dev
git pull origin dev
```

## 当前基线

- 历史 8 周开发任务已完成，项目已进入维护与增量开发阶段
- `origin/dev` 已包含：
  - seller `AddAccount -> POST /internal/v1/accounts` 对接
  - 浏览器跨域 `Network Error` 的 CORS 修复
- 当前本地预览建议基于 `dev` 或从 `dev` 拉出的最新功能分支启动

## 最近完成

- `3c7f016`：合入 seller AddAccount 对接重构
- `225f363`：修复 engine 创建账号时的 seller 外键约束问题
- `94b1337`：添加 API CORS 中间件，修复前端注册/登录 `Network Error`
- `5e20828`：将 CORS 修复合入 `origin/dev`
- `2026-03-23`：新增买家体验重构方案草案，后续已并入 [docs/versions/v0.2/buyer/FEATURES.md](/Users/tvwoo/Projects/gatelink/docs/versions/v0.2/buyer/FEATURES.md)
- `2026-03-23`：新增版本化文档骨架 [docs/versions/v0.2/README.md](/Users/tvwoo/Projects/gatelink/docs/versions/v0.2/README.md)
- `2026-03-23`：将买家草案降级为历史来源，并补齐卖家正式功能文档 [docs/versions/v0.2/seller/FEATURES.md](/Users/tvwoo/Projects/gatelink/docs/versions/v0.2/seller/FEATURES.md)
- `2026-03-23`：将当前协作规则与 GitHub 维护/发布流程写入仓库根目录 [WORKFLOW.md](/Users/tvwoo/Projects/gatelink/WORKFLOW.md) 与 [RELEASES.md](/Users/tvwoo/Projects/gatelink/RELEASES.md)
- `2026-03-23`：明确根目录协作/发布规则为跨版本通用，不与 `docs/versions/v0.2/` 绑定

## 当前任务

- [x] 将团队协作入口从历史 `docs/archive/mvp-bootstrap/legacy/dev-b/STATUS.md` 迁移到仓库根目录 `STATUS.md`
- [x] 将历史 8 周 Dev-A / Dev-B 专属文档整体归档，不再作为当前版本入口
- [x] 在最新 `dev` 基线上重建本地 Docker 预览环境并验证买家注册/登录
- [x] 基于买家草案拆分买家端下一阶段实现任务，并并入版本化文档
- [ ] 将更多当前版本说明逐步迁入 `docs/versions/v0.2/`
- [x] 将当前协作与发布规则迁入仓库根目录
- [x] 将历史单页草案完成迁移后归档到 `docs/archive/mvp-bootstrap/`

## 本轮验证

- `cd /Users/tvwoo/Projects/gatelink/api && env GOCACHE=/tmp/gatelink-go-build GOTOOLCHAIN=go1.25.8+auto go build ./...`
- `cd /Users/tvwoo/Projects/gatelink/api && env GOCACHE=/tmp/gatelink-go-build GOTOOLCHAIN=go1.25.8+auto go test ./... -short`
- `cd /Users/tvwoo/Projects/gatelink/web && npm run build`
- `cd /Users/tvwoo/Projects/gatelink && docker compose up -d --build api web`
- `curl -i -X OPTIONS http://localhost:8080/api/v1/buyer/auth/register ...`
- `curl -i -sS http://localhost:8080/api/v1/buyer/auth/register ...`

关键结果：

- API 构建通过
- API 短测通过
- Web 构建通过
- `OPTIONS /api/v1/buyer/auth/register` 返回 `204 No Content`
- 响应已包含 `Access-Control-Allow-Origin: http://localhost:3000`
- 买家注册接口实际返回 `200 OK`

## 验收基线

合并前至少通过：

```bash
go build ./...
go test ./... -short
```

如果改动影响验收脚本，也要补跑对应脚本。

## 每日同步

每天同步一条消息即可：

- 今天做了什么
- 明天准备做什么
- 当前阻塞项是什么

## 历史文档

- MVP 启动期历史归档入口：`/Users/tvwoo/Projects/gatelink/docs/archive/mvp-bootstrap/README.md`
- 历史 Dev-B 进度：`/Users/tvwoo/Projects/gatelink/docs/archive/mvp-bootstrap/legacy/dev-b/STATUS.md`
- 历史联调/交接记录：`/Users/tvwoo/Projects/gatelink/docs/archive/mvp-bootstrap/legacy/dev-b/DEV_A_HANDOFF.md`

## 当前版本文档入口

- 当前版本：`v0.2`
- 入口文档：[docs/versions/v0.2/README.md](/Users/tvwoo/Projects/gatelink/docs/versions/v0.2/README.md)
- 买家功能：[docs/versions/v0.2/buyer/FEATURES.md](/Users/tvwoo/Projects/gatelink/docs/versions/v0.2/buyer/FEATURES.md)
- 买家任务：[docs/versions/v0.2/buyer/PHASES.md](/Users/tvwoo/Projects/gatelink/docs/versions/v0.2/buyer/PHASES.md)
- 卖家功能：[docs/versions/v0.2/seller/FEATURES.md](/Users/tvwoo/Projects/gatelink/docs/versions/v0.2/seller/FEATURES.md)
- 卖家任务：[docs/versions/v0.2/seller/PHASES.md](/Users/tvwoo/Projects/gatelink/docs/versions/v0.2/seller/PHASES.md)
- 协作流程：[WORKFLOW.md](/Users/tvwoo/Projects/gatelink/WORKFLOW.md)
- 发布流程：[RELEASES.md](/Users/tvwoo/Projects/gatelink/RELEASES.md)
