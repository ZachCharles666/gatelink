# GateLink API Service (Dev-B)

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
│   ├── engine/       # Dev-A internal interface client
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

## Dev-A Internal API

Base URL: `http://engine:8081/internal/v1`
See: `/Users/tvwoo/Projects/gatelink/engine/internal/api/` and `/Users/tvwoo/Projects/gatelink/docs/dev-b/DEV_A_HANDOFF.md`
