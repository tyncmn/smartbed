-- Migration 005: risk_evaluations
-- ─────────────────────────────────────

CREATE TYPE risk_level_type AS ENUM ('normal', 'mild', 'high', 'critical');

CREATE TABLE risk_evaluations (
    id               UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    vital_event_id   UUID            NOT NULL REFERENCES vital_events(id) ON DELETE CASCADE,
    metric_type      metric_type     NOT NULL,
    measured_value   NUMERIC(12,4)   NOT NULL,
    normal_value     NUMERIC(12,4)   NOT NULL,
    deviation        NUMERIC(12,4)   NOT NULL,
    risk_percentage  NUMERIC(8,4)    NOT NULL,
    risk_level       risk_level_type NOT NULL,
    evaluated_at     TIMESTAMPTZ     NOT NULL DEFAULT now(),
    created_at       TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_risk_eval_user_time   ON risk_evaluations (user_id, evaluated_at DESC);
CREATE INDEX idx_risk_eval_vital       ON risk_evaluations (vital_event_id);
CREATE INDEX idx_risk_eval_level       ON risk_evaluations (risk_level, evaluated_at DESC);
