-- Phase 2: rework tracks around UUID storage keys instead of original
-- filenames. This is a destructive migration by design: the dev library
-- is disposable, so we drop and recreate rather than write a one-off
-- re-keying migration for old filename-based rows (see
-- docs/PHASE-2-upload-and-metadata.md). Re-upload your library after this
-- runs.
DROP TABLE tracks;

CREATE TABLE tracks (
    id                BIGSERIAL PRIMARY KEY,
    storage_key       UUID NOT NULL UNIQUE,        -- bare UUID, also the on-disk filename
    mime_type         TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    size              BIGINT NOT NULL,
    title             TEXT NOT NULL,
    artist            TEXT NOT NULL DEFAULT 'Unknown',
    album             TEXT NOT NULL DEFAULT 'Unknown',
    duration_seconds  INTEGER,                      -- nullable; not extracted this phase
    artwork_ext       TEXT,                          -- nullable; e.g. '.jpg' — NULL means no artwork
    uploaded_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
