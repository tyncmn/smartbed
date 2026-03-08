-- Migration 002: baseline_ranges and user_baseline_overrides
-- ─────────────────────────────────────────────────────────

CREATE TYPE metric_type AS ENUM (
    'heart_rate',
    'spo2',
    'stress_level',
    'sleep_duration',
    'skin_temperature',
    'movement_score',
    'sleep_stage',
    'respiration',
    'blood_pressure_systolic',
    'blood_pressure_diastolic'
);

CREATE TYPE age_group_type AS ENUM ('20-29','30-39','40-49','50-59','60-69','70-79','80+');

CREATE TABLE baseline_ranges (
    id            UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    age_group     age_group_type NOT NULL,
    sex           sex_type       NOT NULL,
    metric_type   metric_type    NOT NULL,
    min_value     NUMERIC(10,4)  NOT NULL,
    max_value     NUMERIC(10,4)  NOT NULL,
    normal_value  NUMERIC(10,4)  NOT NULL,   -- midpoint / median reference
    unit          TEXT           NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT now(),
    CONSTRAINT uq_baseline_ranges UNIQUE (age_group, sex, metric_type)
);

CREATE INDEX idx_baseline_ranges_lookup ON baseline_ranges (age_group, sex, metric_type);

-- ─────────────────────────────────────────────────────────

CREATE TABLE user_baseline_overrides (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    metric_type  metric_type NOT NULL,
    min_value    NUMERIC(10,4) NOT NULL,
    max_value    NUMERIC(10,4) NOT NULL,
    normal_value NUMERIC(10,4) NOT NULL,
    set_by_doctor UUID       NOT NULL REFERENCES users(id),
    valid_from   TIMESTAMPTZ NOT NULL DEFAULT now(),
    valid_until  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_overrides UNIQUE (user_id, metric_type, valid_from)
);

CREATE INDEX idx_user_overrides_lookup ON user_baseline_overrides (user_id, metric_type, valid_from);
