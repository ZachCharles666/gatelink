# GateLink GitHub 维护与发布流程

## 1. 核心原则

本文件描述当前仓库的 GitHub 维护与发布流程。
它是 **跨版本通用** 的发布规则，不只服务于 `v0.2`。

现在已经没有 Dev-A / Dev-B 的边界。

这意味着：

- 任何成员都可以修改全部代码
- 但 GitHub 维护和发布必须走统一流程
- 发布不是“谁写了谁就发”，而是“团队确认当前状态可以发”

## 2. 日常维护流程

### Step 1：从 `dev` 拉新分支

```bash
git checkout dev
git pull origin dev
git checkout -b feat/xxx
```

### Step 2：本地开发

- 完成代码
- 更新必要文档
- 跑构建 / 测试 / 验收脚本

### Step 3：合回 `dev`

- push 自己的功能分支
- 发 PR 到 `dev`
- review 后合入 `dev`

这一步是日常主流程。

## 3. `dev -> main` 什么时候合

没有固定频率，看状态而不是看时间。

满足以下条件即可考虑从 `dev` 合到 `main`：

1. `go build ./...` 和 `go test ./... -short` 都通过
2. 当前阶段核心功能能跑，没有明显 broken 的地方
3. 团队成员都知道要合了，不做“悄悄合并”

对当前两人团队来说，比较自然的节奏是：

- 完成一个完整功能
- 或修完一批相关 bug
- 就合一次到 `dev`
- `dev -> main` 大约每 1-2 周一次比较自然

## 4. 发布流程建议

### 日常发布

推荐流程：

1. 所有改动先进入 `dev`
2. 在 `dev` 上完成集成验证
3. `dev` 发 PR 到 `main`
4. 合并后视需要打 tag

### 版本发布

如果当前 `main` 代表一个明确可交付版本，可以打 tag，例如：

```bash
git tag -a v0.2-mvp -m "v0.2 MVP release"
git push origin v0.2-mvp
```

是否打 tag，取决于：

- 当前版本范围是否闭环
- 核心功能是否可对外说明
- 团队是否同意这个时间点作为发布节点

## 5. PR 目标建议

### 功能开发时

- `feat/*` -> `dev`
- `fix/*` -> `dev`
- `chore/*` -> `dev`

### 正式发布时

- `dev` -> `main`

不要把日常功能分支直接往 `main` 发。

## 6. 当前适用结论

你之前发给我的那套流程，现在仍然成立，只是语义已经变成：

- 不是 Dev-A 和 Dev-B 分别维护自己的边界
- 而是两个人共同维护同一个仓库
- 共同遵守同一套：
  - `dev` 集成
  - `main` 发布
  - feature/fix/chore 分支开发

## 7. 一句话总结

当前 GitHub 维护 / 发布流程就是：

- 日常开发：`dev -> feat/fix/chore -> dev`
- 正式发布：`dev -> main`
- 满足稳定性条件时再从 `main` 打 tag
