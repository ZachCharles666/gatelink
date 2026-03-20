CREATE TABLE sellers (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    phone            VARCHAR(20) UNIQUE,
    email            VARCHAR(100) UNIQUE,
    display_name     VARCHAR(50),
    usdt_address     VARCHAR(200),
    usdt_network     VARCHAR(20) DEFAULT 'TRC20',
    total_earned_usd DECIMAL(12,4) NOT NULL DEFAULT 0,
    pending_earn_usd DECIMAL(12,4) NOT NULL DEFAULT 0,
    status           VARCHAR(20) NOT NULL DEFAULT 'active'
                     CHECK(status IN ('active','suspended')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sellers_status ON sellers(status);

COMMENT ON TABLE sellers IS '卖家账号表';
COMMENT ON COLUMN sellers.pending_earn_usd IS '已消耗未结算的收益，每次消耗后累加，结算后清零';
