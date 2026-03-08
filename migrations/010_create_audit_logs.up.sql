-- Migration 010: audit_logs
-- ─────────────────────────────────────

CREATE TABLE audit_logs (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id     UUID        REFERENCES users(id) ON DELETE SET NULL,
    actor_role   user_role,
    user_id      UUID        REFERENCES users(id) ON DELETE SET NULL,
    action       TEXT        NOT NULL,         -- e.g. "protocol.activated"
    entity_type  TEXT        NOT NULL DEFAULT '',
    entity_id    UUID,
    detail       JSONB       NOT NULL DEFAULT '{}',
    ip_address   TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_actor     ON audit_logs (actor_id, created_at DESC);
CREATE INDEX idx_audit_logs_user      ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_logs_action    ON audit_logs (action, created_at DESC);
CREATE INDEX idx_audit_logs_entity    ON audit_logs (entity_type, entity_id);
