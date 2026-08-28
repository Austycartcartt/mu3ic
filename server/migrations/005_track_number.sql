-- Track position within an album. Nullable: an untagged (or non-album)
-- file has no position to show, and NULL orders after any real number
-- rather than colliding with track 0/1 (see store.ListTracksByAlbum).
ALTER TABLE tracks
    ADD COLUMN track_number INTEGER;
