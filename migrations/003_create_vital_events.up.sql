-- Migration 003: vital_events and raw_ingestion_log
-- ─────────────────────────────────────────────────

CREATE TABLE raw_ingestion_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id   TEXT        NOT NULL,
    raw_payload JSONB       NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_raw_ingestion_user    ON raw_ingestion_log (user_id, received_at DESC);
CREATE INDEX idx_raw_ingestion_device  ON raw_ingestion_log (device_id);

-- ─────────────────────────────────────────────────

CREATE TABLE vital_events (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id        TEXT        NOT NULL,
    source_timestamp TIMESTAMPTZ NOT NULL,
    ingestion_id     UUID        NOT NULL REFERENCES raw_ingestion_log(id),
    metric_type      metric_type NOT NULL,
    value            NUMERIC(12,4) NOT NULL,
    unit             TEXT        NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Idempotency: same device + timestamp + metric = same reading
    CONSTRAINT uq_vital_events_idempotent UNIQUE (device_id, source_timestamp, metric_type, user_id)
);

-- TimescaleDB-compatible: single-column time index
CREATE INDEX idx_vital_events_user_time   ON vital_events (user_id, source_timestamp DESC);
CREATE INDEX idx_vital_events_metric_time ON vital_events (metric_type, source_timestamp DESC);
CREATE INDEX idx_vital_events_device      ON vital_events (device_id, source_timestamp DESC);
