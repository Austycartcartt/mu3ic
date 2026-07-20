# Phase 7: Bulk Import (Watch Folder + rclone)

**Status:** Planned — not yet scoped

## Goal

Bulk library import, deferred from the Phase 2 uploader rework:

- Watch-folder scan endpoint on the server — walk a directory, ingest each file through the same `IngestFile` tag-extraction path established in Phase 2 (see [PHASE-2-upload-and-metadata.md](PHASE-2-upload-and-metadata.md))
- `rclone` copy from Google Drive into the watch folder; iCloud via a one-time browser export
- Background upload sessions, retry/resume, and duplicate detection also live in this phase

Not yet broken down into an implementation plan.
