# API Spec v0.3

## 1. 买家端接口

- `POST /auth/wallet/challenge`
- `POST /auth/wallet/verify`
- `GET /buyer/profile`
- `GET /buyer/credits`
- `POST /buyer/deposits/intents`
- `GET /buyer/deposits`
- `POST /buyer/apikeys`
- `GET /buyer/apikeys`
- `DELETE /buyer/apikeys/:id`
- `GET /buyer/usage`
- `POST /buyer/refunds`
- `GET /buyer/refunds`

## 2. 卖家端接口

- `GET /seller/profile`
- `PUT /seller/payout-wallet`
- `GET /seller/resources`
- `POST /seller/resources`
- `PUT /seller/resources/:id`
- `GET /seller/earnings`
- `POST /seller/settlements`
- `GET /seller/settlements`

## 3. 管理端接口

- `GET /admin/overview`
- `GET /admin/deposits`
- `GET /admin/refunds`
- `POST /admin/refunds/:id/retry`
- `GET /admin/settlements`
- `POST /admin/settlements/:id/retry`
- `GET /admin/risk/events`
- `GET /admin/reconciliation`
- `GET /admin/audit-logs`

## 4. 代理接口

- `POST /v1/chat/completions`
- `POST /v1/responses`

说明：
- 这里记录的是 v0.3 业务接口清单，不等于当前代码已全部实现。
- 具体请求/响应字段应在后续实现阶段继续补齐。
