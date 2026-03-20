CREATE TABLE vendor_pricing (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor      VARCHAR(20)     NOT NULL,
    model       VARCHAR(100)    NOT NULL,
    input_price_per_1m  DECIMAL(10,6) NOT NULL,  -- 官方输入价（美元/百万token）
    output_price_per_1m DECIMAL(10,6) NOT NULL,  -- 官方输出价（美元/百万token）
    platform_discount   DECIMAL(4,3)  NOT NULL DEFAULT 0.88,  -- 平台折扣系数
    currency    VARCHAR(10)     NOT NULL DEFAULT 'USD',
    is_active   BOOLEAN         NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE(vendor, model)
);

CREATE INDEX idx_vendor_pricing_vendor ON vendor_pricing(vendor);
CREATE INDEX idx_vendor_pricing_active ON vendor_pricing(is_active);

-- 插入初始定价数据
INSERT INTO vendor_pricing (vendor, model, input_price_per_1m, output_price_per_1m) VALUES
('anthropic', 'claude-opus-4-6',    15.00,  75.00),
('anthropic', 'claude-sonnet-4-6',   3.00,  15.00),
('anthropic', 'claude-haiku-4-5',    0.80,   4.00),
('openai',    'gpt-4o',              2.50,  10.00),
('openai',    'gpt-4o-mini',         0.15,   0.60),
('openai',    'o1',                 15.00,  60.00);

COMMENT ON TABLE vendor_pricing IS '厂商模型定价配置，平台售价 = 官方价 × platform_discount';
