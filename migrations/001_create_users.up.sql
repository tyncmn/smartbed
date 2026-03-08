-- Migration 001: users and user_profiles tables
-- ─────────────────────────────────────────────

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE user_role AS ENUM ('admin', 'doctor', 'caregiver', 'operator', 'patient');
CREATE TYPE sex_type AS ENUM ('male', 'female');

CREATE TABLE users (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email          TEXT        NOT NULL UNIQUE,
    password_hash  TEXT        NOT NULL,
    role           user_role   NOT NULL DEFAULT 'patient',
    is_active      BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_role  ON users (role);

-- ─────────────────────────────────────────────

CREATE TABLE user_profiles (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_name           TEXT        NOT NULL,
    age                 INT         NOT NULL CHECK (age >= 0 AND age <= 120),
    sex                 sex_type    NOT NULL,
    weight_kg           NUMERIC(6,2),
    existing_conditions JSONB       NOT NULL DEFAULT '[]',
    doctor_user_id      UUID        REFERENCES users(id) ON DELETE SET NULL,
    caregiver_user_id   UUID        REFERENCES users(id) ON DELETE SET NULL,
    baseline_profile_id UUID,       -- FK added after baseline table
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_profiles_user_id UNIQUE (user_id)
);

CREATE INDEX idx_user_profiles_user_id ON user_profiles (user_id);
CREATE INDEX idx_user_profiles_doctor  ON user_profiles (doctor_user_id);

-- ─────────────────────────────────────────────

CREATE TABLE emergency_contacts (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    relationship TEXT        NOT NULL,
    phone        TEXT,
    email        TEXT,
    priority     INT         NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_emergency_contacts_user ON emergency_contacts (user_id);
