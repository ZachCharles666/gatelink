-- Dev-B 新增：充值申请记录表
CREATE TABLE IF NOT EXISTS topup_records (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id     UUID          NOT NULL REFERENCES buyers(id),
    amount_usd   DECIMAL(12,4) NOT NULL CHECK(amount_usd > 0),
    tx_hash      VARCHAR(100)  UNIQUE NOT NULL,
    network      VARCHAR(20)   NOT NULL CHECK(network IN ('TRC20', 'ERC20')),
    status       VARCHAR(20)   NOT NULL DEFAULT 'pending'
                 CHECK(status IN ('pending', 'confirmed', 'rejected')),
    confirmed_at TIMESTAMPTZ,
    rejected_at  TIMESTAMPTZ,
    notes        TEXT,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_topup_buyer ON topup_records(buyer_id, created_at DESC);
CREATE INDEX idx_topup_status ON topup_records(status) WHERE status = 'pending';
