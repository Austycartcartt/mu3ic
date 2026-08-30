# Status

Last updated: 2026-08-30

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
| 8 | [Deployment](PHASE-8-deployment.md) | In progress | 2026-08-28 — private-pilot infra + auth hardening. 2026-08-29: production topology set to **Render + Neon** (`render.yaml`); data layer (Postgres + object storage) on Neon `us-east-2`. **2026-08-30: deployed** — `mu3ic-api` + `mu3ic-web` live at their default `*.onrender.com` hostnames, `/api/health` green, migrations applied. Older single-VPS `deploy/` stack kept as a superseded alternative |
| 9 | [Transcoding (Deferred)](PHASE-9-transcoding.md) | Planned | Only build if adaptive bitrate becomes necessary |

## Active phase

**Phase 8 (Deployment)** is in progress — the private-pilot infrastructure landed 2026-08-28: an object-storage backend behind `library.Storage` (presigned-redirect streaming, `STORAGE_BACKEND=fs|r2|neon`), a single-VPS `docker compose` stack (`deploy/`) with the Go server (distroless image) + Caddy (auto-TLS, serves the Expo web export, proxies `/api/*`), and core auth hardening (fatal `JWT_SECRET`, invite-code registration with first-user bootstrap, per-IP auth rate limiting, DB-checked `/api/health`). **2026-08-29:** production topology set to **Render + Neon** — a repo-root `render.yaml` runs the Go API (`mu3ic-api`, Docker) and the Expo web export (`mu3ic-web`, static site); the data layer is entirely on Neon (Lakebase Postgres + Neon Object Storage, project `sweet-star-53712486`, branch `production`, region `us-east-2`). **2026-08-30:** the blueprint was deployed. `mu3ic-api` and `mu3ic-web` are live at `https://mu3ic-api.onrender.com` / `https://mu3ic-web.onrender.com` (Render's default hostnames — no collision), `/api/health` returns `200 {"status":"ok"}`, and migrations `001`–`008` applied against the fresh Neon Postgres. Snags fixed en route: a one-char-truncated `DATABASE_URL` paste (server exited on boot), the deprecated `autoDeploy` field (→ `autoDeployTrigger: commit`), and the invite-code bootstrap only applying when `REGISTRATION_INVITE_CODE` is unset (so the first account is a `curl` with `inviteCode`). Remaining before calling the demo done: an end-to-end upload → stream round-trip, and the `mu3ic-audio` bucket CORS rule only if the browser needs it. The `deploy/` single-VPS + Caddy + local-Postgres + R2 stack is unchanged and kept as a superseded alternative. See the 2026-08-29 [DECISIONS.md](DECISIONS.md) entry and the step-by-step [DEPLOY-RENDER-NEON.md](DEPLOY-RENDER-NEON.md) runbook. Billing, quotas, email, and legal pages are deferred to later phases. A formal **Phase 3** pass over the already-shipped player UI is still outstanding. Phase 7 (Bulk Import) is **skipped** — the watch-folder / rclone / dedup piece was attempted on 2026-08-28 and shelved on the `bulk-upload` branch; the filename-parsing + folder-upload work that shipped 2026-08-21 stays on `main`. See [DECISIONS.md](DECISIONS.md) for the skip rationale. Phase 6 (Playlists & Search) shipped 2026-08-28: `playlists` / `playlist_tracks` tables (additive migration `008`), full playlist CRUD + membership + ▲/▼ reorder, `title/artist/album` substring search on a new tab, and an in-memory playback queue in `usePlayer` (auto-advance + ⏭/⏮). See [DECISIONS.md](DECISIONS.md) for the Phase 6 entry.

## Architecture decisions

See [DECISIONS.md](DECISIONS.md) for the running log of technical decisions and their rationale.
