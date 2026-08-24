# Phase 7: Bulk Import (Watch Folder + rclone)

**Status:** In Progress

## Goal

Bulk library import, deferred from the Phase 2 uploader rework:

- Watch-folder scan endpoint on the server — walk a directory, ingest each file through the same `IngestFile` tag-extraction path established in Phase 2 (see [PHASE-2-upload-and-metadata.md](PHASE-2-upload-and-metadata.md))
- `rclone` copy from Google Drive into the watch folder; iCloud via a one-time browser export
- Background upload sessions, retry/resume, and duplicate detection also live in this phase

The watch-folder/rclone piece above is not yet broken down into an implementation plan.

## Done: smart filename parsing + folder upload (2026-08-21)

Real-world upload batches have messy or missing tags but structured filenames (e.g. `Inferno-01-001-Boards of Canada-Introit.wav`, `Aeikus - Going Deeper, Vol. 7 (Free Download) - 20 Cataracta.m4a`). Shipped:

- `app/src/utils/parseFilename.ts` — heuristic filename → {artist, album, title} guesser, confident-or-blank so it never silently fights a good embedded tag.
- Upload screen (`app/src/app/upload.tsx`) preview step: shows every picked file with its guessed Title/Artist/Album in editable fields before anything uploads.
- A "Choose a folder" picker, web-only (`webkitdirectory`) — native keeps the existing flat multi-file picker.
- Server (`POST /api/tracks`, `library.IngestFile`) accepts optional `title`/`artist`/`album` override fields that win over tag-extraction/fallback when present.

See [DECISIONS.md](DECISIONS.md) for the rationale and trade-offs, including why external dataset lookup (e.g. MusicBrainz) and native folder-tree picking were deferred.
