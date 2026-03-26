# GateLink 项目状态与任务跟踪

> 团队共用状态板。
> 当前以 `dev` 为日常集成分支，以 `docs/product/v0.3/` 为正式产品方案入口。

## 当前状态

- 当前协作模式：`dev -> feat/fix/chore -> dev -> main`
- 当前正式产品版本：`v0.3`
- 当前文档结构：
  - `docs/README.md`
  - `docs/product/CURRENT.md`
  - `docs/product/v0.3/`
  - `archive/`

## 当前基线

- `origin/dev` 最新提交：
  - `410d2176ab5145485c650b112a0b616642e2e648`
  - `2026-03-23 21:59:40 +08:00`
  - `Merge pull request #4 from ZachCharles666/chore/unified-workflow-rules`
- 历史 `v0.2` 文档已归档到 `archive/versions/v0.2/`
- MVP 启动期资料已归档到 `archive/mvp-bootstrap/`

## 最近完成

- `2026-03-23`：统一仓库协作规则进入 `dev`
- `2026-03-23`：补齐 GitHub 协作与发布规则
- `2026-03-26`：按新的文件管理结构重排文档目录
- `2026-03-26`：建立 `docs/product/v0.3/` 文档集
- `2026-03-26`：将历史 `v0.2` 与 MVP 启动期资料移动到根目录 `archive/`

## 当前任务

- [x] 拉取最新 `dev` 作为主整理基线
- [x] 将旧版本文档入口迁移到 `docs/product/v0.3/`
- [x] 将历史文档统一归档到根目录 `archive/`
- [x] 创建 `v0.3` 产品文档骨架
- [ ] 基于 `v0.3` 文档逐步迁移实现层业务逻辑

## 当前版本文档入口

- 文档总入口：`docs/README.md`
- 当前版本标记：`docs/product/CURRENT.md`
- 产品总文档：`docs/product/v0.3/PRD.md`
- 买家端：`docs/product/v0.3/BUYER.md`
- 卖家端：`docs/product/v0.3/SELLER.md`
- 管理端：`docs/product/v0.3/ADMIN.md`
- 平台后台：`docs/product/v0.3/PLATFORM_BACKEND.md`

## 仓库级规则入口

- 协作流程：`WORKFLOW.md`
- 发布流程：`RELEASES.md`

## 历史文档入口

- 归档总入口：`archive/README.md`
- 历史 `v0.2`：`archive/versions/v0.2/README.md`
- MVP 启动期资料：`archive/mvp-bootstrap/README.md`
