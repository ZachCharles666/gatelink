CREATE TABLE usage_records (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id          UUID        NOT NULL REFERENCES accounts(id),
    buyer_id            UUID        NOT NULL REFERENCES buyers(id),
    vendor              VARCHAR(20) NOT NULL,
    model               VARCHAR(100) NOT NULL,
    input_tokens        INTEGER     NOT NULL DEFAULT 0,
    output_tokens       INTEGER     NOT NULL DEFAULT 0,
    cost_usd            DECIMAL(12,6) NOT NULL,
    buyer_charged_usd   DECIMAL(12,6) NOT NULL,
    seller_earn_usd     DECIMAL(12,6) NOT NULL,
    platform_earn_usd   DECIMAL(12,6) NOT NULL,
    vendor_request_id   VARCHAR(100),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT check_amounts CHECK (
        ABS(buyer_charged_usd - seller_earn_usd - platform_earn_usd) < 0.000001
    )
);

CREATE INDEX idx_usage_records_account_time ON usage_records(account_id, created_at DESC);
CREATE INDEX idx_usage_records_buyer_time   ON usage_records(buyer_id, created_at DESC);
CREATE INDEX idx_usage_records_created_at   ON usage_records(created_at DESC);

COMMENT ON TABLE usage_records IS '消耗记录，平台核心账本，每笔消耗一条记录';
COMMENT ON CONSTRAINT check_amounts ON usage_records IS '三方金额守恒约束，数据库层面防止记账错误';
