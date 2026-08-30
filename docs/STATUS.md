# Status

Last updated: 2026-08-29

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
| 7 | [Bulk Import (Watch Folder + rclone)](PHASE-7-bulk-import.md) | **Skipped** | Smart filename parsing + folder upload shipped 2026-08-21 and stays; watch-folder/rclone/dedup attempted 2026-08-28 and parked on the `bulk-upload` branch — not being merged |
| 8 | [Deployment](PHASE-8-deployment.md) | In progress | 2026-08-28 — private-pilot infra + auth hardening. 2026-08-29: production topology set to **Render + Neon** (`render.yaml`); data layer (Postgres + object storage) on Neon `us-east-2`; blueprint complete and self-wiring (default `*.onrender.com` hostnames pinned), first deploy pending. Older single-VPS `deploy/` stack kept as a superseded alternative |
| 9 | [Transcoding (Deferred)](PHASE-9-transcoding.md) | Planned | Only build if adaptive bitrate becomes necessary |

## Active phase

**Phase 8 (Deployment)** is in progress — the private-pilot infrastructure landed 2026-08-28: an object-storage backend behind `library.Storage` (presigned-redirect streaming, `STORAGE_BACKEND=fs|r2|neon`), a single-VPS `docker compose` stack (`deploy/`) with the Go server (distroless image) + Caddy (auto-TLS, serves the Expo web export, proxies `/api/*`), and core auth hardening (fatal `JWT_SECRET`, invite-code registration with first-user bootstrap, per-IP auth rate limiting, DB-checked `/api/health`). **2026-08-29:** production topology set to **Render + Neon** — a repo-root `render.yaml` runs the Go API (`mu3ic-api`, Docker) and the Expo web export (`mu3ic-web`, static site); the data layer is entirely on Neon (Lakebase Postgres + Neon Object Storage, project `sweet-star-53712486`, branch `production`, region `us-east-2`). As of 2026-08-29 the `render.yaml` blueprint is complete and self-wiring — `mu3ic-web`'s `EXPO_PUBLIC_API_URL` is pinned to Render's default `https://mu3ic-api.onrender.com` hostname, so the first deploy needs no manual URL wiring, only the `sync: false` secrets. Remaining before a live demo: run the blueprint deploy, add the `mu3ic-audio` bucket CORS rule for the web origin, and confirm an end-to-end upload → stream round-trip. The `deploy/` single-VPS + Caddy + local-Postgres + R2 stack is unchanged and kept as a superseded alternative. See the 2026-08-29 [DECISIONS.md](DECISIONS.md) entry. Billing, quotas, email, and legal pages are deferred to later phases. See [DECISIONS.md](DECISIONS.md) and `deploy/DEPLOYMENT.md`. A formal **Phase 3** pass over the already-shipped player UI is still outstanding. Phase 7 (Bulk Import) is **skipped** — the watch-folder / rclone / dedup piece was attempted on 2026-08-28 and shelved on the `bulk-upload` branch; the filename-parsing + folder-upload work that shipped 2026-08-21 stays on `main`. See [DECISIONS.md](DECISIONS.md) for the skip rationale. Phase 6 (Playlists & Search) shipped 2026-08-28: `playlists` / `playlist_tracks` tables (additive migration `008`), full playlist CRUD + membership + ▲/▼ reorder, `title/artist/album` substring search on a new tab, and an in-memory playback queue in `usePlayer` (auto-advance + ⏭/⏮). See [DECISIONS.md](DECISIONS.md) for the Phase 6 entry.

## Architecture decisions

See [DECISIONS.md](DECISIONS.md) for the running log of technical decisions and their rationale.
