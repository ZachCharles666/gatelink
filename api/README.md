# GateLink API Service

Business API + buyer proxy endpoint + frontend console.

## Structure

```
api/
├── cmd/api/          # Service entry point
├── internal/
│   ├── api/          # HTTP router setup
│   ├── auth/         # JWT + api_key auth
│   ├── seller/       # Seller business logic
│   ├── buyer/        # Buyer business logic
│   ├── proxy/        # Buyer proxy endpoints (/v1/*)
│   ├── accounting/   # Billing and settlement
│   ├── engine/       # Internal engine interface client
│   ├── poller/       # Account status DB polling
│   ├── config/       # Config loading
│   └── db/           # Database connection + migrations
├── scripts/          # Verification scripts
└── tests/            # Integration tests
```

## Run

```bash
cp .env.example .env
go run ./cmd/api
```

## Current Product Docs

- `docs/README.md`
- `docs/product/CURRENT.md`
- `docs/product/v0.3/PRD.md`
- `docs/product/v0.3/BUYER.md`
- `docs/product/v0.3/PLATFORM_BACKEND.md`

## Internal Engine API

Base URL: `http://engine:8081/internal/v1`

Source of truth:
- `engine/internal/api/`
- `api/internal/engine/client.go`
