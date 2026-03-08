-- Migration 006: alerts
-- ─────────────────────────────────────

CREATE TYPE alert_type AS ENUM (
    'abnormal_heart_rate',
    'low_oxygen',
    'elevated_temperature',
    'disturbed_sleep',
    'high_risk_health_event',
    'critical_health_event'
);

CREATE TABLE alerts (
    id                UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alert_type        alert_type      NOT NULL,
    risk_level        risk_level_type NOT NULL,
    message           TEXT            NOT NULL,
    metric_type       metric_type,
    metric_value      NUMERIC(12,4),
    is_acknowledged   BOOLEAN         NOT NULL DEFAULT FALSE,
    acknowledged_by   UUID            REFERENCES users(id) ON DELETE SET NULL,
    acknowledged_at   TIMESTAMPTZ,
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_alerts_user_time  ON alerts (user_id, created_at DESC);
CREATE INDEX idx_alerts_type       ON alerts (alert_type, created_at DESC);
CREATE INDEX idx_alerts_unread     ON alerts (user_id, is_acknowledged) WHERE is_acknowledged = FALSE;
