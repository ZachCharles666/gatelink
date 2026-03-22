# Bug Tracker

> 记录已知 Bug、复现步骤、修复状态。
> 状态：`open` / `in-progress` / `fixed` / `wont-fix`

---

## 模板

```
### BUG-XXX: 标题
- **状态**: open
- **优先级**: P0 / P1 / P2
- **发现日期**: YYYY-MM-DD
- **影响范围**: 哪个模块 / 端点 / 场景
- **复现步骤**:
  1. ...
- **预期行为**: ...
- **实际行为**: ...
- **根因分析**: （修复后填写）
- **修复 commit**: （修复后填写）
```

---

## 已知问题

### BUG-001: todo.md 中 Week 5-6 两条未勾选
- **状态**: open
- **优先级**: P2
- **发现日期**: 2026-03-20
- **影响范围**: `docs/todo.md` 文档
- **说明**: `GET /internal/v1/accounts/:id/console-usage` 和 `GET /internal/v1/accounts/:id/diff` 在 Week 9 已实现，但 Week 5-6 章节的对应条目仍为 `[ ]` 未勾选。
- **修复方式**: 将两行 `[ ]` 改为 `[x]`，无需代码改动。

---

### BUG-002: `POST /accounts/:id/verify` 仅做格式校验
- **状态**: open
- **优先级**: P1
- **发现日期**: 2026-03-20
- **影响范围**: `internal/api/accounts.go` — HandleVerify
- **说明**: 接口文档要求验证 Key 有效性，当前实现只检查 Key 字符串格式（非空/前缀），未发真实 HTTP 请求到厂商，无法检测 Key 是否过期或被撤销。
- **根因分析**: MVP 阶段有意降级，待 B-01 backlog 项实现后关闭。
- **关联 backlog**: B-01

---

### BUG-003: Console sync stub 适配器静默失败
- **状态**: open
- **优先级**: P1
- **发现日期**: 2026-03-20
- **影响范围**: `internal/sync/adapters/stub.go` — Gemini / Qwen / GLM / Kimi
- **说明**: 四个厂商的 Console 适配器直接返回 error，syncer 会将此计为对账失败并触发 `reconcile_fail` 健康扣分，影响这些厂商账号的健康分。
- **根因分析**: stub 实现未区分"功能未实现"与"对账失败"，应在 stub 中返回 `ErrNotImplemented` 并让 syncer 对此类错误跳过扣分。
- **关联 backlog**: B-02 ~ B-05

---

### BUG-004: API 缺少 CORS 中间件，前端页面全部报 Network Error
- **状态**: fixed
- **优先级**: P0
- **发现日期**: 2026-03-22
- **影响范围**: `api/internal/api/router.go`，所有前端页面
- **复现步骤**:
  1. 启动全套服务 `docker compose up -d`
  2. 浏览器访问 `http://localhost:3000/seller/login`
  3. 尝试注册或登录，页面提示 Network Error
- **预期行为**: 正常调用 API，返回注册/登录结果
- **实际行为**: 浏览器 CORS preflight 被拒绝，请求无法发出
- **根因分析**: `router.go` 未注册任何 CORS 中间件，浏览器从 `:3000` 跨域请求 `:8080` 时缺少 `Access-Control-Allow-Origin` 响应头，被浏览器拦截
- **修复**: 引入 `github.com/gin-contrib/cors`，在 `SetupRoutes` 入口处注册，允许 `http://localhost:3000`；同步将 `api/Dockerfile` 基础镜像从 `golang:1.22` 升至 `golang:1.23`（依赖要求）

*暂无其他已知 Bug。*
