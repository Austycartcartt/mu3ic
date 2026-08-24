-- Album artist, distinct from each track's own artist. Grouping albums by
-- (album, artist) alone can't tell a various-artists compilation (one
-- album, many track artists — should be one album row) apart from two
-- different real albums that happen to share a title (should stay two
-- rows) — both look identical: multiple artists under one album title.
-- album_artist resolves the ambiguity explicitly, the same way the
-- ALBUMARTIST/TPE2 tag does in real files.
ALTER TABLE tracks
    ADD COLUMN album_artist TEXT NOT NULL DEFAULT '';

-- Backfill existing rows to their own track artist, preserving today's
-- (album, artist) grouping for anything already uploaded. A track only
-- gets a distinct album_artist going forward via an explicit tag or
-- override (see library.IngestFile).
UPDATE tracks SET album_artist = artist;
