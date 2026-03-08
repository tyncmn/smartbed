-- Migration 004: sleep_sessions and sleep_stage_events
-- ─────────────────────────────────────────────────────

CREATE TABLE sleep_sessions (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_time        TIMESTAMPTZ NOT NULL,
    end_time          TIMESTAMPTZ,
    duration_minutes  INT         NOT NULL DEFAULT 0,
    quality_score     NUMERIC(5,2) NOT NULL DEFAULT 0,     -- 0-100
    disturbance_count INT         NOT NULL DEFAULT 0,
    is_disturbed      BOOLEAN     NOT NULL DEFAULT FALSE,
    avg_heart_rate    NUMERIC(6,2),
    avg_spo2          NUMERIC(6,2),
    avg_movement_score NUMERIC(6,4),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sleep_sessions_user_time ON sleep_sessions (user_id, start_time DESC);

-- ─────────────────────────────────────────────────────

CREATE TYPE sleep_stage_type AS ENUM ('light', 'deep', 'rem', 'awake');

CREATE TABLE sleep_stage_events (
    id               UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    sleep_session_id UUID             NOT NULL REFERENCES sleep_sessions(id) ON DELETE CASCADE,
    stage_type       sleep_stage_type NOT NULL,
    start_time       TIMESTAMPTZ      NOT NULL,
    end_time         TIMESTAMPTZ      NOT NULL,
    duration_mins    INT              NOT NULL,
    created_at       TIMESTAMPTZ      NOT NULL DEFAULT now()
);

CREATE INDEX idx_sleep_stage_session ON sleep_stage_events (sleep_session_id, start_time);
