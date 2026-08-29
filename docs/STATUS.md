# Status

Last updated: 2026-08-28

One-line state of the project: which phase is active, what's done, what's next. Update this whenever a phase changes status — it's the first thing to read to get oriented.

## Phases

| # | Phase | Status | Notes |
|---|-------|--------|-------|
| 1 | [Thin Vertical Slice](PHASE-1-thin-vertical-slice.md) | Complete | 2026-07-14 |
| 2 | [Upload & Metadata (Reworked Uploader)](PHASE-2-upload-and-metadata.md) | Complete | Incl. album_artist (`004`) + track numbers (`005`), through 2026-08-23 |
| 3 | [Audio Player UI](PHASE-3-audio-player-ui.md) | Planned | Player, menu & dock code already landed ahead of a formal pass |
| 4 | [Background Playback & Lock-Screen Controls](PHASE-4-background-playback.md) | Planned | Requires a custom Expo dev build (no Expo Go) |
| 5 | [Authentication](PHASE-5-authentication.md) | Complete | 2026-08-27 — JWT, email login, per-user libraries |
| 6 | [Playlists & Search](PHASE-6-playlists-search.md) | Complete | 2026-08-28 — playlist CRUD + reorder, `ILIKE` track search, playback queue |
| 7 | [Bulk Import (Watch Folder + rclone)](PHASE-7-bulk-import.md) | **In Progress** | Smart filename parsing + folder upload shipped 2026-08-21; watch-folder/rclone not yet scoped |
| 8 | [Deployment](PHASE-8-deployment.md) | Planned | Not yet scoped |
| 9 | [Transcoding (Deferred)](PHASE-9-transcoding.md) | Planned | Only build if adaptive bitrate becomes necessary |

## Active phase

**Phase 7: Bulk Import** (watch-folder / rclone piece — still needs scoping), or a formal **Phase 3** pass over the already-shipped player UI. Phase 6 (Playlists & Search) shipped 2026-08-28: `playlists` / `playlist_tracks` tables (additive migration `008`), full playlist CRUD + membership + ▲/▼ reorder, `title/artist/album` substring search on a new tab, and an in-memory playback queue in `usePlayer` (auto-advance + ⏭/⏮). See [DECISIONS.md](DECISIONS.md) for the Phase 6 entry.

## Architecture decisions

See [DECISIONS.md](DECISIONS.md) for the running log of technical decisions and their rationale.
