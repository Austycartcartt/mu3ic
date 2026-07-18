ALTER TABLE tracks
    ADD COLUMN title        TEXT,
    ADD COLUMN artist       TEXT,
    ADD COLUMN album        TEXT,
    ADD COLUMN track_number INTEGER,
    ADD COLUMN year         INTEGER,
    ADD COLUMN has_artwork  BOOLEAN NOT NULL DEFAULT FALSE;
