-- Migration 011: seed default admin user (idempotent)

INSERT INTO users (email, password_hash, role, is_active)
VALUES (
    'admin@smartbed.local',
    crypt('Admin@2024!', gen_salt('bf', 10)),
    'admin',
    TRUE
)
ON CONFLICT (email) DO NOTHING;
