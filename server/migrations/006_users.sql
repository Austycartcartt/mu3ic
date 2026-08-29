-- Phase 5: user accounts. Login is by email (not username — see
-- docs/PHASE-5-authentication.md and the DECISIONS.md entry). id is
-- BIGSERIAL to match tracks.id and the int64 code paths, rather than the
-- UUID the phase doc originally sketched.
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,               -- bcrypt
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
