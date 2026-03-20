CREATE TABLE buyers (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    phone               VARCHAR(20) UNIQUE,
    email               VARCHAR(100) UNIQUE,
    api_key             VARCHAR(64) UNIQUE NOT NULL,
    balance_usd         DECIMAL(12,4) NOT NULL DEFAULT 0,
    total_consumed_usd  DECIMAL(12,4) NOT NULL DEFAULT 0,
    tier                VARCHAR(20) NOT NULL DEFAULT 'standard'
                        CHECK(tier IN ('standard','pro')),
    status              VARCHAR(20) NOT NULL DEFAULT 'active'
                        CHECK(status IN ('active','suspended')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_buyers_api_key ON buyers(api_key);
CREATE INDEX idx_buyers_status  ON buyers(status);

COMMENT ON TABLE buyers IS '买家账号表';
COMMENT ON COLUMN buyers.api_key IS '买家调用平台API时使用，与厂商Key完全隔离';
