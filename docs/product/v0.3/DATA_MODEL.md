# Data Model v0.3

## 1. 核心实体

- `users`
- `wallet_bindings`
- `buyers`
- `sellers`
- `buyer_api_keys`
- `deposit_intents`
- `deposit_records`
- `credit_ledger`
- `usage_records_view`
- `seller_earnings_ledger`
- `settlements`
- `refunds`
- `audit_logs`

## 2. 买家相关

### buyers

- `id`
- `user_id`
- `wallet_address`
- `credits_balance`
- `status`
- `created_at`
- `updated_at`

### buyer_api_keys

- `id`
- `buyer_id`
- `name`
- `key_hash`
- `last_used_at`
- `status`
- `created_at`

## 3. 卖家相关

### sellers

- `id`
- `user_id`
- `wallet_address`
- `payout_wallet_address`
- `pending_earn_usd`
- `total_paid_usd`
- `status`
- `created_at`
- `updated_at`

## 4. 资金相关

### deposit_intents

- `id`
- `buyer_id`
- `wallet_address`
- `amount_usdc`
- `status`
- `expires_at`
- `created_at`

### credit_ledger

- `id`
- `buyer_id`
- `entry_type`
- `amount`
- `balance_after`
- `source_type`
- `source_id`
- `created_at`

### settlements

- `id`
- `seller_id`
- `amount_usdc`
- `status`
- `tx_hash`
- `requested_at`
- `paid_at`

### refunds

- `id`
- `buyer_id`
- `amount_usdc`
- `status`
- `tx_hash`
- `requested_at`
- `confirmed_at`

## 5. 账本原则

- 金额字段应使用 decimal / 定点整数，不使用 float 作为最终账务精度
- 所有链上事件必须可追溯到 ledger source
