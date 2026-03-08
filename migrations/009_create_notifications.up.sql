-- Migration 009: notifications
-- ─────────────────────────────────────

CREATE TABLE notifications (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id UUID        NOT NULL REFERENCES users(id),
    channel      TEXT        NOT NULL,  -- email | sms | push | webhook
    subject      TEXT        NOT NULL DEFAULT '',
    body         TEXT        NOT NULL,
    status       TEXT        NOT NULL DEFAULT 'pending',  -- pending | sent | failed
    sent_at      TIMESTAMPTZ,
    failed_at    TIMESTAMPTZ,
    error_msg    TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user      ON notifications (user_id, created_at DESC);
CREATE INDEX idx_notifications_recipient ON notifications (recipient_id, created_at DESC);
CREATE INDEX idx_notifications_status    ON notifications (status);
