CREATE TABLE settlements (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id           UUID        NOT NULL REFERENCES sellers(id),
    period_start        TIMESTAMPTZ NOT NULL,
    period_end          TIMESTAMPTZ NOT NULL,
    total_consumed_usd  DECIMAL(12,4) NOT NULL,
    seller_earn_usd     DECIMAL(12,4) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending'
                        CHECK(status IN ('pending','paid','disputed')),
    paid_at             TIMESTAMPTZ,
    tx_hash             VARCHAR(200),
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_settlements_seller_id ON settlements(seller_id);
CREATE INDEX idx_settlements_status    ON settlements(status);

COMMENT ON TABLE settlements IS '结算记录，每14天生成一次或卖家手动申请';
