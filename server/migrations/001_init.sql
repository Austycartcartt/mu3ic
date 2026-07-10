CREATE TABLE tracks (
    id         BIGSERIAL PRIMARY KEY,
    filename   TEXT NOT NULL,
    size       BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
