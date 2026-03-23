# GateLink — AI Credits 授权与调度平台

> 当前仓库已进入统一协作阶段。
> 日常开发分支为 `dev`，功能与修复从 `feat/*`、`fix/*`、`chore/*` 分支拉出，完成后合回 `dev`，再由 `dev` 合入 `main`。

GateLink 是一个面向企业的 AI 调用授权与调度引擎，负责统一管理多供应商 AI API 的凭证、配额、路由与审计。

## 分支用途

- `main`：稳定主干，只接受来自 `dev` 的 PR
- `dev`：日常集成分支
- `feat/*` / `fix/*` / `chore/*`：功能、修复、维护分支

## 功能特性

- **多供应商适配**：支持 OpenAI、Anthropic、Google Gemini、通义千问、Kimi、GLM 等主流 AI 服务
- **额度管理**：账户级 / 项目级 Credits 配额分配与消耗追踪
- **智能调度**：基于健康评分与优先级的供应商路由策略
- **代理转发**：透明代理 AI 请求，支持流式（SSE）响应
- **审计日志**：全链路请求审计，支持合规过滤与分类
- **加密存储**：AES-256-GCM 加密 API Key，防止密钥泄露

## 技术栈

- **语言**：Go 1.25
- **Web 框架**：Gin
- **数据库**：PostgreSQL 16
- **缓存**：Redis 7
- **日志**：zerolog
- **迁移**：golang-migrate

## 项目结构

```
GateLink/
├── engine/                 # 核心服务
│   ├── cmd/
│   │   ├── engine/         # 服务入口
│   │   └── migrate/        # 数据库迁移工具
│   ├── internal/
│   │   ├── api/            # HTTP 路由与处理器
│   │   ├── audit/          # 审计日志与分类
│   │   ├── config/         # 配置加载
│   │   ├── crypto/         # 密钥加密存储
│   │   ├── db/             # 数据库连接与迁移
│   │   ├── health/         # 供应商健康监测
│   │   ├── proxy/          # 请求代理与流转
│   │   ├── scheduler/      # 调度引擎
│   │   └── sync/           # 供应商状态同步
│   ├── pkg/
│   │   └── adapters/       # 各 AI 供应商适配器
│   ├── scripts/            # 数据库迁移脚本
│   └── tests/              # 集成测试
├── api/                    # 业务 API 与代理服务
├── web/                    # 前端控制台
├── docs/
│   ├── versions/           # 当前版本产品与任务文档
│   └── archive/            # 历史启动期资料归档
└── docker-compose.yml      # 本地开发环境
```

## 开发入口

- 项目状态与任务：`STATUS.md`
- 仓库协作规则：`WORKFLOW.md`
- GitHub 发布规则：`RELEASES.md`
- 当前版本文档：`docs/versions/v0.2/README.md`
- 说明：`WORKFLOW.md` 与 `RELEASES.md` 是跨版本通用规则，不只服务于 `v0.2`
- API 说明：`api/README.md`
- 前端说明：`web/README.md`
- 历史归档：`docs/archive/mvp-bootstrap/README.md`

## 快速开始

### 前置依赖

- Go 1.21+
- Docker & Docker Compose

### 启动开发环境

```bash
# 1. 克隆仓库
git clone https://github.com/yourname/gatelink.git
cd gatelink

# 2. 配置环境变量
cp engine/.env.example engine/.env
# 编辑 engine/.env，填写加密密钥等配置

# 3. 启动依赖服务（PostgreSQL + Redis）
docker compose up -d postgres redis

# 4. 运行数据库迁移
cd engine
go run ./cmd/migrate/main.go

# 5. 启动服务
go run ./cmd/engine/main.go
```

服务默认监听 `http://localhost:8081`

### 使用 Docker 运行完整栈

```bash
docker compose up -d
```

## 开发

```bash
# 编译
go build ./...

# 静态检查
go vet ./...

# 运行测试
go test ./... -v
```

## 配置说明

参考 `engine/.env.example`：

| 变量 | 说明 | 示例 |
|------|------|------|
| `DATABASE_URL` | PostgreSQL 连接串 | `postgres://user:pass@localhost:5432/gatelink` |
| `REDIS_URL` | Redis 连接串 | `redis://localhost:6379/0` |
| `ENCRYPTION_KEY` | AES-256 密钥（32字节十六进制）| `openssl rand -hex 32` |
| `ENGINE_PORT` | 服务监听端口 | `8081` |
| `ENV` | 运行环境 | `development` / `production` |

## License

MIT
