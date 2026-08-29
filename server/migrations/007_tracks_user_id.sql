-- Phase 5: tracks are now owned by a user. This is destructive by design:
-- the dev library is disposable (same rationale as 003_uuid_storage.sql),
-- so we wipe existing rows rather than backfill an owner onto them.
-- Re-upload your library after this runs, and clear stale files from
-- DATA_DIR (server/data/) — they're now orphaned.
DELETE FROM tracks;

ALTER TABLE tracks
    ADD COLUMN user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX tracks_user_id_idx ON tracks (user_id);
