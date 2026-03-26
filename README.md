# GateLink

GateLink 是一个面向 AI API 交易与调度场景的多服务仓库。

当前开发分支规则：
- `main`：稳定主干
- `dev`：日常集成分支
- `feat/*` / `fix/*` / `chore/*`：日常开发分支

当前主要目录：
- `engine/`：核心引擎
- `api/`：业务 API
- `web/`：前端控制台
- `docs/product/v0.3/`：当前正式产品方案
- `archive/`：历史版本与旧资料

## 当前文档入口

- 当前版本标记：`docs/product/CURRENT.md`
- 文档总入口：`docs/README.md`
- 产品总文档：`docs/product/v0.3/PRD.md`
- 买家端：`docs/product/v0.3/BUYER.md`
- 卖家端：`docs/product/v0.3/SELLER.md`
- 管理端：`docs/product/v0.3/ADMIN.md`
- 平台后台：`docs/product/v0.3/PLATFORM_BACKEND.md`

## 版本说明

- 当前正式版本：`v0.3`
- `v0.2` 与 MVP 启动期资料已归档到 `archive/`
- `WORKFLOW.md` 与 `RELEASES.md` 是跨版本通用规则

## 项目结构

```text
GateLink/
├── engine/                 # 核心引擎代码
├── api/                    # 业务 API
├── web/                    # 前端控制台
├── docs/
│   ├── README.md
│   └── product/
│       ├── CURRENT.md
│       └── v0.3/
├── archive/                # 历史版本与旧资料
├── STATUS.md               # 仓库当前状态
├── WORKFLOW.md             # 协作规则
└── RELEASES.md             # 发布规则
```

## 快速开始

### 前置依赖

- Go 1.21+
- Docker & Docker Compose

### 启动开发环境

```bash
git clone https://github.com/ZachCharles666/gatelink.git
cd gatelink

cp engine/.env.example engine/.env
docker compose up -d postgres redis

cd engine
go run ./cmd/migrate/main.go
go run ./cmd/engine/main.go
```

默认服务地址：`http://localhost:8081`

## 开发

```bash
go build ./...
go vet ./...
go test ./... -v
```

## License

MIT
