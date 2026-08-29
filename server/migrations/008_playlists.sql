-- Phase 6: playlists. Additive — unlike 003_uuid_storage.sql and
-- 007_tracks_user_id.sql, this drops nothing and needs no re-upload.
--
-- playlist_tracks uses (playlist_id, track_id) as its primary key, so the
-- same track can't be added to one playlist twice. Ordering is a plain
-- gapped INTEGER position: reorder rewrites the affected rows, and gaps
-- left by a removal are harmless since only the relative order is read.
CREATE TABLE playlists (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX playlists_user_id_idx ON playlists (user_id);

CREATE TABLE playlist_tracks (
    playlist_id BIGINT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    track_id    BIGINT NOT NULL REFERENCES tracks(id)    ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    added_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (playlist_id, track_id)
);
CREATE INDEX playlist_tracks_playlist_pos_idx ON playlist_tracks (playlist_id, position);
