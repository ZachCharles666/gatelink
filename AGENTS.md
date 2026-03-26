# GateLink 团队协作指引

> 适用于当前仓库中的所有开发者与 AI 助手。
> 项目采用统一仓库协作模式，不再按历史角色拆分当前开发入口。

## 一、协作模式

### 分支约定

- `main`：稳定主干，只接受来自 `dev` 的 PR
- `dev`：日常集成分支
- `feat/*` / `fix/*` / `chore/*`：功能、修复、维护分支

### 每天开始前

```bash
git checkout dev
git pull origin dev
```

### 合并前最少验证

```bash
go build ./...
go test ./... -short
```

如果当前模块有专门脚本，也要补跑对应脚本。

## 二、通用开发原则

1. 不要臆造 API、文件、配置、依赖或行为
2. 有疑问时，先查当前仓库实现，再看文档
3. 非 trivial 任务遵循：Plan → Evidence → Implement → Verify
4. Diff 保持最小化，不顺手改无关代码
5. 完成任务前必须给出真实验证证据

## 三、当前文档来源

当前正式产品文档以以下文件为准：
- `/Users/tvwoo/Projects/TTT/docs/README.md`
- `/Users/tvwoo/Projects/TTT/docs/product/CURRENT.md`
- `/Users/tvwoo/Projects/TTT/docs/product/v0.3/README.md`
- `/Users/tvwoo/Projects/TTT/docs/product/v0.3/PRD.md`
- `/Users/tvwoo/Projects/TTT/docs/product/v0.3/BUYER.md`
- `/Users/tvwoo/Projects/TTT/docs/product/v0.3/SELLER.md`
- `/Users/tvwoo/Projects/TTT/docs/product/v0.3/ADMIN.md`
- `/Users/tvwoo/Projects/TTT/docs/product/v0.3/PLATFORM_BACKEND.md`

仓库级规则文件：
- `/Users/tvwoo/Projects/TTT/STATUS.md`
- `/Users/tvwoo/Projects/TTT/WORKFLOW.md`
- `/Users/tvwoo/Projects/TTT/RELEASES.md`

历史资料：
- `/Users/tvwoo/Projects/TTT/archive/`

说明：
- `archive/` 仅作历史参考，不作为当前版本入口
- 若文档与当前实现冲突，以当前仓库实现为准

## 四、实现约束

1. 所有内部接口格式以仓库中的真实实现为准
2. 对 `engine`、`api`、`web` 的修改都必须基于当前任务需要
3. 流式请求必须在流结束后记账
4. 涉及共享行为变更时，要同步更新当前版本文档或状态文档

## 五、验证标准

至少执行与改动相关的真实验证，并记录：
- 实际执行的命令
- 实际输出或关键结果
- 如果某项验证没法跑，明确说明原因
