-- Migration 007: doctor_protocols and protocol_rules
-- ─────────────────────────────────────────────────────

CREATE TYPE protocol_state     AS ENUM ('draft','approved','active','suspended');
CREATE TYPE trigger_outcome    AS ENUM ('notify_only','suggest_action','require_manual_approval','send_dispenser_command');
CREATE TYPE rule_operator_type AS ENUM ('gt','lt','gte','lte','eq');

CREATE TABLE doctor_protocols (
    id               UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id       UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    doctor_id        UUID            NOT NULL REFERENCES users(id),
    name             TEXT            NOT NULL,
    description      TEXT            NOT NULL DEFAULT '',
    state            protocol_state  NOT NULL DEFAULT 'draft',
    trigger_outcome  trigger_outcome NOT NULL DEFAULT 'notify_only',
    doctor_approved  BOOLEAN         NOT NULL DEFAULT FALSE,
    is_locked        BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_protocols_patient ON doctor_protocols (patient_id, state);
CREATE INDEX idx_protocols_doctor  ON doctor_protocols (doctor_id);

CREATE TABLE protocol_rules (
    id               UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    protocol_id      UUID              NOT NULL REFERENCES doctor_protocols(id) ON DELETE CASCADE,
    metric_type      metric_type       NOT NULL,
    operator         rule_operator_type NOT NULL,
    threshold_value  NUMERIC(12,4)     NOT NULL,
    risk_level       risk_level_type   NOT NULL DEFAULT 'mild',
    created_at       TIMESTAMPTZ       NOT NULL DEFAULT now()
);

CREATE INDEX idx_protocol_rules_protocol ON protocol_rules (protocol_id);
