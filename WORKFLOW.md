# GateLink 协作工作流

## 1. 适用范围

本文件描述当前仓库的团队协作方式。
它是 **跨版本通用** 的仓库级规则，不绑定某一个产品版本。

当前项目已经进入统一仓库协作模式：

- 不再区分 Dev-A / Dev-B 的代码边界
- 任何成员都可以修改 `engine`、`api`、`web`
- 但必须遵循统一分支流程、验证规则和合并纪律

## 2. 分支规则

### 分支职责

- `main`
  - 稳定主干
  - 禁止直接 push
  - 只接受来自 `dev` 的 PR

- `dev`
  - 日常集成分支
  - 所有功能和修复都从这里拉出
  - 完成后先合回这里

- `feat/xxx`
  - 新功能开发分支

- `fix/xxx`
  - 缺陷修复分支

- `chore/xxx`
  - 文档、脚本、维护性调整分支

### 每天开始前

```bash
git checkout dev
git pull origin dev
```

### 新任务开始方式

从 `dev` 拉分支：

```bash
git checkout dev
git pull origin dev
git checkout -b feat/xxx
```

或：

```bash
git checkout -b fix/xxx
git checkout -b chore/xxx
```

## 3. 提交规范

提交信息格式：

```text
feat / fix / refactor / test / chore: 简短描述
```

示例：

```text
feat: 买家广场页面初版
fix: 修复买家注册跨域问题
chore: 调整版本化文档结构
```

## 4. 合并前必须通过

至少执行：

```bash
go build ./...
go test ./... -short
```

如果当前模块有对应验收脚本，也要补跑，例如：

```bash
bash api/scripts/week6_verify.sh
```

或其他与当前改动直接相关的脚本。

## 5. 冲突处理

优先使用 rebase：

```bash
git fetch origin
git rebase origin/dev
```

当前最容易冲突的文件：

- `cmd/engine/main.go`
- `go.mod`

处理原则：

- 保留双方有效内容
- 不要直接覆盖对方代码
- 如果文档和实现冲突，以当前代码为准，再补文档同步

## 6. 每日同步

每天同步一条简报即可：

- 今天做了什么
- 明天准备做什么
- 当前阻塞项是什么

## 7. 当前团队共识

- 先合到 `dev`
- `main` 只接收已经相对稳定、可对外代表当前版本状态的内容
- 文档、代码、验证结果要一起前进
