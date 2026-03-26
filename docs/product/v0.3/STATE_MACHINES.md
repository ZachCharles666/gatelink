# State Machines v0.3

## 1. 充值状态

- `intent_created`
- `onchain_pending`
- `confirmed`
- `credited`
- `failed`
- `expired`

## 2. 退款状态

- `requested`
- `queued`
- `sent`
- `confirmed`
- `failed`

## 3. 结算状态

- `pending`
- `approved`
- `sent`
- `paid`
- `failed`

## 4. API 调用结算

- `request_received`
- `dispatching`
- `streaming`
- `completed`
- `billed`
- `failed`

规则：
- `streaming` 阶段不扣费
- 只有在 `completed` 后才能进入 `billed`

## 5. 钱包绑定状态

- `unverified`
- `verified`
- `rebind_pending`
- `frozen`
