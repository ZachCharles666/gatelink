CREATE TABLE accounts (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id              UUID        NOT NULL REFERENCES sellers(id),
    vendor                 VARCHAR(20) NOT NULL
                           CHECK(vendor IN ('anthropic','openai','gemini','qwen','glm','kimi')),
    api_key_encrypted      TEXT        NOT NULL,
    api_key_hint           VARCHAR(20),
    total_credits_usd      DECIMAL(12,4) NOT NULL DEFAULT 0,
    authorized_credits_usd DECIMAL(12,4) NOT NULL DEFAULT 0,
    consumed_credits_usd   DECIMAL(12,4) NOT NULL DEFAULT 0,
    expected_rate          DECIMAL(4,3) NOT NULL DEFAULT 0.75,
    expire_at              TIMESTAMPTZ NOT NULL,
    health_score           SMALLINT    NOT NULL DEFAULT 80
                           CHECK(health_score >= 0 AND health_score <= 100),
    status                 VARCHAR(20) NOT NULL DEFAULT 'pending_verify'
                           CHECK(status IN ('pending_verify','active','suspended','revoked','expired')),
    last_synced_at         TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounts_seller_id    ON accounts(seller_id);
CREATE INDEX idx_accounts_vendor       ON accounts(vendor);
CREATE INDEX idx_accounts_status       ON accounts(status);
CREATE INDEX idx_accounts_expire_at    ON accounts(expire_at);
CREATE INDEX idx_accounts_health_score ON accounts(health_score);
CREATE INDEX idx_accounts_dispatch ON accounts(vendor, status, health_score, expire_at)
    WHERE status = 'active';

COMMENT ON TABLE accounts IS '卖家托管的API账号，Key加密存储，调度引擎核心数据源';
COMMENT ON COLUMN accounts.api_key_encrypted IS 'AES-256-GCM加密，解密密钥存环境变量，永不在日志中出现';
COMMENT ON COLUMN accounts.expected_rate IS '0.75 = 卖家期望拿回官方价的75%作为收益';
