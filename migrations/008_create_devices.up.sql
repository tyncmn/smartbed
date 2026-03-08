-- Migration 008: dispenser_devices, device_commands, device_acknowledgements
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TYPE command_status AS ENUM ('queued','published','acknowledged','executed','failed','timeout');

CREATE TABLE dispenser_devices (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_code TEXT        NOT NULL UNIQUE,
    model       TEXT        NOT NULL DEFAULT '',
    firmware_ver TEXT       NOT NULL DEFAULT '',
    is_online   BOOLEAN     NOT NULL DEFAULT FALSE,
    last_seen_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_devices_patient ON dispenser_devices (patient_id);
CREATE INDEX idx_devices_code    ON dispenser_devices (device_code);

CREATE TABLE device_commands (
    id               UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id        UUID            NOT NULL REFERENCES dispenser_devices(id) ON DELETE CASCADE,
    protocol_id      UUID            REFERENCES doctor_protocols(id),
    issued_by        UUID            NOT NULL REFERENCES users(id),
    command_type     TEXT            NOT NULL,  -- dispense | hold | reset | check_status
    payload          JSONB           NOT NULL DEFAULT '{}',
    status           command_status  NOT NULL DEFAULT 'queued',
    published_at     TIMESTAMPTZ,
    acknowledged_at  TIMESTAMPTZ,
    executed_at      TIMESTAMPTZ,
    timeout_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_device_commands_device ON device_commands (device_id, created_at DESC);
CREATE INDEX idx_device_commands_status ON device_commands (status);

CREATE TABLE device_acknowledgements (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    device_command_id UUID        NOT NULL REFERENCES device_commands(id) ON DELETE CASCADE,
    device_code       TEXT        NOT NULL,
    status            TEXT        NOT NULL,  -- executed | failed
    message           TEXT        NOT NULL DEFAULT '',
    received_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_device_ack_command ON device_acknowledgements (device_command_id);
