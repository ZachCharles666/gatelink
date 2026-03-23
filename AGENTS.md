# GateLink 团队协作指引

> 本文件适用于当前 `gatelink` 仓库中的所有开发者与 AI 助手。
> 项目已经进入统一仓库协作阶段，不再区分 Dev-A / Dev-B 边界。

---

## 一、协作模式

### 分支约定

- `main`：禁止直接 push，只接受来自 `dev` 的 PR
- `dev`：日常集成分支，所有功能从这里拉，完成后合回这里
- `feat/xxx` / `fix/xxx` / `chore/xxx`：个人开发分支

### 每天开始前

```bash
git checkout dev
git pull origin dev
```

### 提交信息格式

```text
feat / fix / refactor / test / chore: 简短描述
```

示例：

```text
feat: 卖家结算申请接口
fix: 添加 CORS 中间件修复前端 Network Error
```

### 合并前必须通过

```bash
go build ./...
go test ./... -short
```

如果当前模块有验收脚本，也必须一并执行，例如：

```bash
bash scripts/test/<对应脚本>.sh
```

### 冲突处理

优先使用 rebase：

```bash
git fetch origin
git rebase origin/dev
```

最容易冲突的文件：

- `cmd/engine/main.go`
- `go.mod`

处理冲突时：

- 保留双方有效内容
- 不要直接覆盖对方代码
- 如果实现与文档冲突，以当前仓库代码为准，并补一条同步说明

### 每日同步

每天同步一条消息即可：

- 今天做了什么
- 明天准备做什么
- 当前阻塞项是什么

---

## 二、通用开发原则

1. 不要臆造 API、文件、配置、依赖或行为
2. 有疑问时，先查当前仓库实现，再看文档
3. 非 trivial 任务遵循：Plan → Evidence → Implement → Verify
4. Diff 保持最小化，不顺手改无关代码
5. 完成任务前必须给出真实验证证据

---

## 三、当前仓库的事实来源

优先级从高到低：

1. 当前仓库代码与配置
2. 当前分支上的测试、脚本、构建文件
3. 仓库内文档

### 开发前建议先看

- `/Users/tvwoo/Projects/gatelink/STATUS.md`
- `/Users/tvwoo/Projects/gatelink/WORKFLOW.md`
- `/Users/tvwoo/Projects/gatelink/RELEASES.md`
- `/Users/tvwoo/Projects/gatelink/docs/versions/v0.2/README.md`
- 当前模块对应的版本化功能文档与任务文档

说明：

- 根目录 `STATUS.md` 是当前团队共用状态板
- 根目录 `WORKFLOW.md` / `RELEASES.md` 是仓库级协作与发布规则
- `docs/versions/v0.2/` 是当前版本功能与任务文档入口
- 仓库级协作规则是跨版本通用的，不跟某个 `vX.Y` 目录绑定
- `docs/archive/mvp-bootstrap/` 是 8 周 MVP 启动期的历史文档归档区
- 若文档与当前实现冲突，以当前仓库实现为准

---

## 四、实现约束

1. 所有内部接口格式以仓库中的真实实现为准
2. 对 `engine`、`api`、`web` 的修改都必须基于当前任务需要，不能跨模块乱改
3. 流式请求仍然必须在流结束后记账，不能在流中途扣款
4. 写代码前先确认目标文件路径已经存在于仓库结构内
5. 涉及共享行为变更时，要把影响记录进状态文档或协作文档

---

## 五、验证标准

任务完成前，至少执行与改动相关的真实验证。

常用命令：

```bash
cd api && go build ./...
cd api && go test ./... -short
cd web && npm run build
```

如果改动影响某一周验收脚本或某个模块：

```bash
bash api/scripts/week{N}_verify.sh
```

必须记录：

- 实际执行的命令
- 实际输出或关键结果
- 如果某项验证没法跑，明确说明原因

---

## 六、当前协作节奏

- `dev -> main` 没有固定频率，看状态而不是看时间
- 满足以下条件即可考虑从 `dev` 合到 `main`：
  1. `go build ./...` 和 `go test ./... -short` 都通过
  2. 当前阶段核心功能能跑，没有明显 broken 的地方
  3. 团队成员都知道要合了

当前阶段建议：

- 完成一个完整功能，或修完一批相关 bug，就合一次到 `dev`
- `dev -> main` 可以按 1-2 周一次的自然节奏进行
