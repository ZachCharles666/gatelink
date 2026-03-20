CREATE TABLE health_events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  UUID        NOT NULL REFERENCES accounts(id),
    event_type  VARCHAR(50) NOT NULL,
    score_delta SMALLINT    NOT NULL,
    score_after SMALLINT    NOT NULL,
    detail      JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_health_events_account_time ON health_events(account_id, created_at DESC);
CREATE INDEX idx_health_events_event_type   ON health_events(event_type);

COMMENT ON TABLE health_events IS '账号健康度变化事件日志，用于调试和审计';
