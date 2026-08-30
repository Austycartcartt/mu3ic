# Architecture & Decisions

Append-only log. Add a new entry per decision; don't edit past entries except to change `Status` (e.g. `Decided` → `Superseded`) — add a note explaining why rather than rewriting history.

---

## PostgreSQL for metadata & track index

**Date:** 2026-06-28 · **Category:** Database · **Status:** Decided

**Rationale:** Relational schema for tracks, albums, playlists, users. Simple queries and JOIN support without overengineering.

**Trade-offs:** Not a document database — but relational structure is natural for music library metadata. No denormalization needed upfront.

---

## Defer authentication & complexity

**Date:** 2026-06-28 · **Category:** Auth · **Status:** Decided

**Rationale:** Build a thin vertical slice first to validate end-to-end connectivity. Auth comes after foundational layers are solid. Known complexity: attaching auth headers to stream URLs.

**Trade-offs:** Early phases are less secure. But risk is low in a personal project, and addressing real problems (proving connectivity) first is pragmatic.

---

## HTTP range requests for audio streaming

**Date:** 2026-06-28 · **Category:** Audio Transport · **Status:** Decided

**Rationale:** HTTP range requests with `http.ServeContent` handle `206 Partial Content` automatically. Functionally equivalent to production streaming infrastructure. Avoids WebRTC (wrong paradigm — it's P2P/real-time) and HLS/DASH complexity (not needed until adaptive bitrate over variable connections becomes necessary).

**Trade-offs:** Cannot do adaptive bitrate without segmentation. But this is deferred until it's actually needed; adds complexity with minimal upside when bandwidth is local/predictable.

---

## Expo + React Native for mobile (one codebase)

**Date:** 2026-06-28 · **Category:** Frontend · **Status:** Decided

**Rationale:** Web, iOS, Android from a single Expo/React Native codebase. Uses `expo-audio` (not deprecated `expo-av`). Leverages existing React/TypeScript proficiency.

**Trade-offs:** Mobile performance depends on Expo bridging. But for audio streaming and UI responsiveness, this is acceptable. Native modules can be added later if needed.

---

## Pin Expo SDK 54 instead of latest

**Date:** 2026-07-13 · **Category:** Frontend · **Status:** Decided

**Rationale:** `app/` is pinned to Expo SDK 54 instead of latest (was SDK 57) because Expo Go on the App Store only supports SDK 54, and the dev-build alternative requires a paid Apple Developer account.

**Trade-offs:** Diverges from the "always latest stable" ground rule in `PROJECT.md`. Do not bump back to latest without confirming Expo Go / Apple Developer account status first.

---

## Store audio files under UUID keys, not original filenames

**Date:** 2026-07-16 · **Category:** Database · **Status:** Decided

**Rationale:** The server generates a UUID for each uploaded track and uses it as the storage key (bare UUID, no extension). MIME type and original filename are stored as DB columns; `Content-Type` is set from the DB when serving. This avoids filename collisions, weird characters, and path traversal in one move, and makes a future migration to cloud object storage (MinIO/S3) a drop-in change — the UUID becomes the object key unchanged, and nothing in the DB or API cares where the bytes live.

**Trade-offs:** Library folder is not human-browsable (opaque filenames); original filename must be preserved as a metadata column. Extension-less files mean local tools can't infer type from name — acceptable since all serving goes through the API, which sets `Content-Type` from the DB.

---

## Extract metadata at upload time, not as a separate phase

**Date:** 2026-07-16 · **Category:** API Design · **Status:** Decided

**Rationale:** The upload endpoint is the natural choke point where every track passes exactly once. Flow: receive multipart → buffer to temp file (`os.CreateTemp` on the same volume as the library) → extract tags with `dhowden/tag` (needs an `io.ReadSeeker`, so a temp file is required) → `os.Rename` into the library under its UUID key (atomic on the same filesystem, no half-written files) → insert row. Fallback chain for missing tags: title from tag, else original filename minus extension; artist/album default to `"Unknown"`.

**Trade-offs:** Upload handler does more work per request (tag parsing adds latency, negligible at personal scale). Requires the temp dir to be on the same filesystem as the library to keep the rename atomic. Replaces the previously planned standalone metadata phase, so the build order shifted.

---

## Client-side filename-heuristic upload preview, explicit override wins

**Date:** 2026-08-21 · **Category:** Upload UX · **Status:** Decided

**Rationale:** Real-world filenames like `Inferno-01-001-Boards of Canada-Introit.wav` or `Aeikus - Going Deeper, Vol. 7 (Free Download) - 20 Cataracta.m4a` often carry better artist/album/title information than missing or wrong embedded tags, but the server's filename fallback (`ingest.go`) only ever used the whole filename as a title. `app/src/utils/parseFilename.ts` parses a small set of known dash-delimited patterns (plus a generic 2/3-token fallback) into an Artist/Album/Title guess, purely from the filename string — no audio bytes or server round-trip needed. The upload screen (`app/src/app/upload.tsx`) shows this guess in an editable preview before anything uploads; on confirm, the (possibly hand-edited) values are sent as optional `title`/`artist`/`album` multipart fields, and `library.Overrides` (`ingest.go`) applies any non-empty one on top of the existing tag-extraction/fallback chain — i.e. an explicit override always wins, even over a real embedded tag.

**Trade-offs:** The parser only pre-fills a field when it's confident in the match, specifically so an unrecognized filename doesn't send a bogus override that clobbers a good embedded tag — but a confident *wrong* guess (e.g. a filename that happens to fit the pattern by coincidence) will still silently override real tag data unless the user notices and edits it in the preview. The parser is a small heuristic, not a general filename grammar; it won't handle every naming convention, and the editable preview is the intended correctness backstop, not full automation.

---

## Folder upload is web-only; native keeps flat multi-select

**Date:** 2026-08-21 · **Category:** Upload UX · **Status:** Decided

**Rationale:** Bulk-picking a whole folder needs a directory-tree picker. Browsers expose one for free via the non-standard `webkitdirectory` input attribute (`app/src/app/upload.tsx`'s `pickFolderWeb`), with no new dependency. Expo has no built-in equivalent for iOS/Android — Android would need Storage Access Framework tree-picking and iOS has no comparable API without custom native code, both out of proportion to the ask. Native keeps today's `expo-document-picker` flat multi-file picker, which already lets a user select many files from a folder in one dialog, just without nested subfolders auto-included.

**Trade-offs:** Native users importing a deeply nested folder tree must flatten it themselves (drag all files into the picker, or pick subfolders one at a time) rather than pointing at the top-level folder. Revisit if this proves painful enough to justify a native module.

---

## Defer external album/artist dataset lookup (e.g. MusicBrainz)

**Date:** 2026-08-21 · **Category:** Upload UX · **Status:** Decided

**Rationale:** Comparing uploaded filenames against a downloadable/online music database could improve metadata accuracy beyond filename parsing, but it means a new network dependency, fuzzy-matching logic, and (for an offline dataset) meaningful storage — a large scope increase relative to the filename-parsing preview this phase actually needed. Shipped the heuristic parser + editable preview instead (see above).

**Trade-offs:** Metadata quality is capped by what the filename itself encodes; genuinely ambiguous or sparse filenames still need manual entry. Revisit as a later phase if manual cleanup proves too frequent.

---

## Add album_artist, distinct from each track's own artist

**Date:** 2026-08-23 · **Category:** Database / Upload UX · **Status:** Decided

**Rationale:** `ListAlbums` grouped by `(album, artist)`. For a various-artists compilation — one album, a different artist per track, e.g. from the filename-parsing preview above — that produced one album row per distinct track artist instead of one row for the whole album (reported as: uploading "Going Deeper, Vol. 7" and seeing it listed multiple times). Grouping by `album` alone was rejected: it can't tell that case apart from two genuinely different albums that happen to share an exact title by different single artists (already covered by an existing test, `TestHandleListAlbums_SameTitleDifferentArtist`) — both look identical as "one album title, multiple artists" without more information. Added an `album_artist` column (`004_album_artist.sql`), mirroring the ALBUMARTIST/TPE2 tag real files use for exactly this: `ListAlbums`/`ListTracksByAlbum` now group/filter on `(album, album_artist)`. `store.InsertTrack` defaults an unset `album_artist` to that track's own artist, so ordinary single-artist albums (and the title-collision test) keep working unchanged — only an explicit album artist (an ALBUMARTIST tag, or the new optional "Album Artist" field on the upload preview screen, applied to every file in a batch) collapses a compilation into one row.

**Trade-offs:** A various-artists compilation uploaded without ALBUMARTIST tags still needs the user to fill in the batch-level "Album Artist" field (e.g. "Various Artists") for it to group correctly — there's no way to infer it from filenames alone. Existing already-uploaded rows were backfilled to `album_artist = artist` in the same migration, so nothing already in the library needs re-uploading to keep working as before.

---

## JWT auth: email login, bearer header + `?token=` fallback, per-user libraries

**Date:** 2026-08-27 · **Category:** Auth · **Status:** Decided (Phase 5)

**Rationale:** Implemented the auth Phase 5 planned. Stateless HS256 JWT (`github.com/golang-jwt/jwt/v5`), 24h expiry, signed with `JWT_SECRET` (env var; insecure dev default with a startup warning). Passwords hashed with `golang.org/x/crypto/bcrypt`. Both are new dependencies — PROJECT.md's stdlib-first rule was waived here because the Phase 5 doc pre-approved them and hand-rolling either (especially password hashing) is the kind of security-sensitive code you don't want to own. New `internal/auth` package (no HTTP types), `internal/store/users.go`, `internal/api/{auth,middleware}.go`. `users` (`006_users.sql`) plus `tracks.user_id` (`007_tracks_user_id.sql`, `NOT NULL REFERENCES users ON DELETE CASCADE`). Every track/artist/album store method now takes a `userID` and filters on it; `GetTrack` is scoped too, so another user's track id is a 404, identical to a missing one. Routes: `/api/health` and `/api/auth/{register,login}` are public; everything else is wrapped by `withAuth`, which reads the token from `Authorization: Bearer` **or** a `?token=` query param — the query param exists because the audio player and `<Image>` fetch `/stream` and `/artwork` directly and can't set headers. `RunMigrations` gained a Postgres advisory lock so parallel `go test ./...` package binaries don't race to apply a new migration.

**Deviations from `docs/PHASE-5-authentication.md`:** (1) login is by **email**, not username — `users.email UNIQUE` with a loose format check. (2) `users.id` is **BIGSERIAL**, not UUID, to match `tracks.id` and the existing `int64` / `strconv.ParseInt` code paths; `user_id` is a plain `BIGINT` FK. (3) Registration is open (anyone can register) — fine on a LAN, revisit before the Phase 8 public deployment.

**Trade-offs:** `007` is destructive — it `DELETE`s all existing `tracks` rather than backfilling an owner (dev library is disposable; same call as `003_uuid_storage.sql`), so the library must be re-uploaded and stale files cleared from `DATA_DIR`. Token-in-URL for streaming means the token can appear in server logs / browser history; acceptable for a personal app, the Phase 5 doc notes a streaming-only short-lived token as the future hardening. On web the token is persisted in `localStorage` (SecureStore has no web backend) — an XSS-exfiltration risk the native Keychain/Keystore path avoids. Open registration + an unset `JWT_SECRET` would let anyone mint tokens; the deploy checklist must cover both.

---

## Playlists & Search: substring search, gapped-integer ordering, in-memory queue

**Date:** 2026-08-28 · **Category:** Database / API Design / Frontend · **Status:** Decided (Phase 6)

**Rationale:** Implemented Phase 6 as planned in `docs/PHASE-6-playlists-search.md`. New tables `playlists` and `playlist_tracks` (`008_playlists.sql`), both `ON DELETE CASCADE` from their parents. Unlike `003_uuid_storage.sql` / `007_tracks_user_id.sql`, `008` is **additive** — it drops nothing and needs no re-upload. `playlist_tracks` uses `PRIMARY KEY (playlist_id, track_id)`, so a track can't be added to the same playlist twice; ordering is a plain gapped `INTEGER position` that reorder rewrites wholesale (gaps from a removal are harmless — only relative order is read). New `internal/store/playlists.go` mirrors `albums.go`: every method takes `userID` and scopes on it, so another user's playlist id is `sql.ErrNoRows` → an indistinguishable 404. `ReorderPlaylist` requires the submitted id list to be *exactly* the current membership (`ErrReorderMismatch` → 400) so a stale client can't half-apply an order. Adds run `INSERT … ON CONFLICT DO NOTHING` (idempotent). Nine new routes under `s.protected`; `withCORS`'s `Access-Control-Allow-Methods` widened from `GET, POST, OPTIONS` to add `PATCH, PUT, DELETE` (the web client now issues them).

**Search** is a deliberate `title/artist/album ILIKE '%q%'` (`store.SearchTracks`, `LIMIT 100`), **not** Postgres full-text search or `pg_trgm`: no stemming or ranking, but zero new machinery (no `tsvector` column, GIN index, generated-column trigger, or extension) and more than adequate at a personal library's scale. A `strings.NewReplacer` escapes `\ % _` with an `ESCAPE '\'` clause; the handler returns `[]` for a blank query rather than dumping the library, since the client hits it on every keystroke.

**Frontend:** `usePlayer` gained a queue — `play(track, uri)` became `playQueue(tracks, startIndex)`, with `playNext`/`playPrevious` and auto-advance on `expo-audio`'s `status.didJustFinish` (ref-gated to fire once per finish). This also makes the Songs/album/artist screens play through their list in context, not just the one tapped track. Reorder UI is an "edit mode" with ▲/▼ move buttons, **not** drag-and-drop: `react-native-draggable-flatlist` would be a new dependency PROJECT.md forbids without justification, and ▲/▼ is fine for short playlists. Search is a dedicated 5th tab. `playlists.tsx` became a `playlists/` folder (`_layout` + `index` + `[id]`), matching `albums/`. `PlaylistNameModal` is a cross-platform text-prompt (RN's `Alert.prompt` is iOS-only).

**Trade-offs:** No duplicate tracks per playlist (composite PK) — revisit if ever wanted. Search quality is capped at substring matching — no typo tolerance or relevance ordering; `pg_trgm`/`tsvector` is the upgrade path if the library outgrows it. Reorder isn't drag-and-drop, so reordering a long playlist is many taps. The playback queue is in-memory only: it doesn't survive an app restart, and there's no "up next" view, shuffle, or repeat yet. `ReorderPlaylist` issues one `UPDATE` per row in a loop inside the transaction (fine at personal scale; a single `UPDATE … FROM (VALUES …)` would scale better).

---

## Skip Phase 7 bulk import (watch folder + rclone + dedup)

**Date:** 2026-08-28 · **Category:** Scope · **Status:** Decided (Phase 7 skipped)

**Rationale:** Phase 7's remaining scope — a server-side watch-folder scan endpoint, a `009_track_content_hash` migration for duplicate detection, an `import-from-rclone.sh` wrapper, and the ingest/store plumbing for both — was implemented as WIP and then abandoned. It's a large, stateful addition (scan endpoint, content hashing, dedup semantics, an external `rclone` dependency and script) whose only real job is a one-time migration of an existing library into the app, which a plain `curl` loop against the existing `POST /api/tracks` upload endpoint already handles well enough. The value-to-complexity ratio didn't hold up. The filename-heuristic upload preview and web folder picker that also came out of this phase (see the 2026-08-21 entries above) shipped separately, are on `main`, and are unaffected.

**Trade-offs:** No built-in "point the server at a folder / cloud remote and sync" import — bulk loading stays a manual scripted upload against the public endpoint. The WIP is preserved on the `bulk-upload` branch (commit `de82a1c`) as a reference if this is ever revived; it is not merged, so migration `009` and `tracks.content_hash` do **not** exist on `main`. Any future bulk-import work should re-scope from scratch rather than resurrect the branch wholesale.

---

## Product pivot: private-pilot "music locker" SaaS (Phase 8)

**Date:** 2026-08-28 · **Category:** Scope / Product · **Status:** Decided (Phase 8)

**Rationale:** mu3ic is moving from a LAN-only personal app ("a server you control") toward a commercial multi-tenant service: each user uploads *their own* library, it stays private to them, and it only ever streams back to *that same user's* devices — the personal-use framing that keeps a storage/streaming locker clear of distribution/copyright problems. The multi-user primitives already exist from Phase 5 (every store method scoped by `user_id`; another user's track id is an indistinguishable 404), so this is an infrastructure and hardening effort, not a data-model change. Phase 8 deliberately covers **only private-pilot infrastructure** — enough to onboard invited users. Billing, storage quotas, marketing site, and legal/policy pages are explicitly later phases so Phase 8 stays shippable.

**Trade-offs:** The `PROJECT.md` "self-hosted, personal-scale, LAN" framing is now aspirational-past for the hosted deployment (the local `fs` storage backend is effectively dev-only). Single VPS + single Postgres means no HA yet. Open questions punted to later phases: per-user quota enforcement, payment, account lifecycle emails, ToS/DMCA, data export/deletion.

---

## Object storage: Cloudflare R2 behind `library.Storage`, presigned-redirect streaming

**Date:** 2026-08-28 · **Category:** Storage · **Status:** Decided (Phase 8), supersedes "Storage is read-only" note in the 2026-07-16 UUID-keys entry

**Rationale:** A hosted music locker can't keep every user's library on one VPS disk. Added a second `library.Storage` backend for **Cloudflare R2** (S3-compatible, and — decisive for a streaming product — zero egress fees), selected by `STORAGE_BACKEND=fs|r2`. This required reversing the earlier "Storage is read-only, writes go through `os.Rename` in `IngestFile`" decision: `Storage` now has `Put`/`Delete`, and `IngestFile` calls `storage.Put`. The local `FileStorage` keeps its temp-file-then-atomic-rename semantics internally; `R2Storage` does a `PutObject`.

Streaming uses an **optional `Presigner` capability**: when the backend implements it (R2), `handleStream`/`handleArtwork` issue a `302` to a short-lived presigned GET URL (`STREAM_URL_TTL`, default 15m) with `response-content-type` pinned to the DB mime type — track bytes never transit the app server, and Range requests go straight to R2. When it doesn't (filesystem), the handlers fall back to `http.ServeContent` exactly as before. No schema change: `storage_key` stays a bare UUID, artwork key is `storage_key + artwork_ext`.

**Dependency:** `github.com/minio/minio-go/v7` for the S3 client. Chosen over `aws-sdk-go-v2` (roughly doubles the project's module count — disproportionate under PROJECT.md's "justify every dependency") and over a hand-rolled SigV4 signer (don't want to own crypto-signing code; the Phase 5 entry set the precedent of waiving stdlib-first for security-sensitive code). `minio-go` is a single primary import with a small transitive set and is well-tested against R2.

**Trade-offs:** A new dependency and its transitive deps. Presigned-redirect means a leaked stream URL is usable by anyone until it expires (bounded, ≤15m, and narrower than the existing `?token=` exposure). `R2Storage.Open` (the non-presign read path, kept for tooling/parity) does a `StatObject` round-trip to surface a missing key as `fs.ErrNotExist`. R2 Put/Open/Delete are covered by manual verification against a real bucket, not unit tests (offline tests cover interface behavior, `FileStorage`, and presigned-URL shape).

---

## Deploy topology: single VPS, docker compose, Caddy TLS, external object storage

**Date:** 2026-08-28 · **Category:** Deployment · **Status:** Decided (Phase 8)

**Rationale:** For a private pilot, one Linux VPS running `docker compose` is the lowest-moving-parts option that still meets the bar (HTTPS, multi-tenant, backups, monitoring) and matches the project's minimalist ethos better than a PaaS. `deploy/docker-compose.yml`: `db` (Postgres, no published port), `server` (distroless-static image, migrations baked in, auth via `.env`), `caddy` (ports 80/443, automatic Let's Encrypt, serves the `expo export -p web` static bundle with SPA fallback, reverse-proxies `/api/*`, and strips `?token=` from its access log). Object bytes are in R2, so the server needs no persistent volume — just a tmpfs for upload staging. Secrets are a `chmod 600` `deploy/.env` (a real secrets manager is deferred). Backups are host-cron `pg_dump`; audio durability is R2 bucket versioning.

**Trade-offs:** No CI/CD — deploy is `git pull && docker compose up -d --build` on the box. No HA, no read replica, no horizontal scaling. The Caddy image builds the web bundle via a Node stage (heavier image, but keeps Node off the VPS and makes `up --build` reproducible); a host-built bind-mount is documented as the lighter alternative.

---

## Auth hardening for a public deployment (Phase 8)

**Date:** 2026-08-28 · **Category:** Auth · **Status:** Decided (Phase 8), addresses the deploy-checklist items in the 2026-08-27 Phase 5 entry

**Rationale:** The Phase 5 entry flagged open registration and an unset `JWT_SECRET` as things "the deploy checklist must cover." Phase 8 handles them in code rather than a checklist:
- **`JWT_SECRET` is fatal** — the server `os.Exit(1)`s if it's unset, equal to the old dev default, or shorter than 32 chars (was a warn-and-continue). `start.sh` exports a stable dev value so local workflow is unchanged.
- **Registration is closed by default.** `REGISTRATION_INVITE_CODE` (constant-time compared) gates new accounts; `REGISTRATION_OPEN=true` is a staging-only escape hatch. A zero-users **first-run bootstrap** (`store.CreateFirstUser`, a conditional `INSERT ... WHERE NOT EXISTS`) lets a fresh deployment create its first account with no code, then closes. No new table. _Correction (2026-08-30): the bootstrap is the `default` branch of `handleRegister` — it only applies when **no** `REGISTRATION_INVITE_CODE` is configured. When one is set, that branch wins and even the first account must supply the code; the web signup form has no invite field, so the first account is a `curl` with `"inviteCode"` in the body._
- **Auth endpoints are rate-limited** per client IP — a hand-rolled token bucket (`~1 req/s`, burst 5, idle sweep), wrapping only `/api/auth/register` and `/api/auth/login`. Hand-rolled rather than `golang.org/x/time/rate` to avoid a dependency for ~60 lines. Client IP comes from `X-Real-IP` only when `TRUST_PROXY=true` (Caddy sets it, overwriting so it can't be spoofed); otherwise from `RemoteAddr`.
- **`/api/health` does a DB ping** (2s timeout) → `503 {"status":"degraded"}` on failure, so uptime monitoring is meaningful.
- **Request IDs** — every request gets a short `X-Request-Id`, logged alongside method/path/status/duration (status capture is new too).

`NewServer`'s positional signature became `api.Options` to carry the new knobs (registration policy, trust-proxy, presigned-URL TTL).

**Trade-offs:** Still no email verification, password reset, or short-lived/refresh tokens — the full 24h JWT still rides `?token=` on stream URLs (Phase 8 narrows the exposure with Caddy log redaction and ≤15m presigned R2 URLs, but doesn't eliminate token-in-URL). The invite code is a single shared secret, not per-user or emailed. Rate-limit state is in-memory per process (fine for a single-server pilot).

---

## Production moves to Render + Neon (Phase 8)

**Date:** 2026-08-29 · **Category:** Deployment / Storage · **Status:** Decided (Phase 8), supersedes the single-VPS topology in the two 2026-08-28 Phase 8 deploy/storage entries

**Rationale:** Production now runs the Go API on **Render** (managed platform: builds `server/Dockerfile`, terminates TLS, injects `PORT`) with **Neon** as the entire data layer — Lakebase Postgres for the metadata DB and Neon Object Storage (S3-compatible) for audio/artwork bytes — instead of a self-managed single VPS running Postgres + Caddy + Cloudflare R2. This is the recommended Neon shape: the app platform owns the runtime, Neon is the data it talks to; nothing stateful runs on the app host. Neon Object Storage branches *with* the database, so a Neon branch of `production` yields a consistent copy-on-write snapshot of rows and the objects they reference — useful for future staging/restore branches. Everything is in one Neon project (`sweet-star-53712486`, org Austin), branch `production`, region **`us-east-2`** (mandatory: Object Storage's public beta serves only that region; Render's `ohio` region is the same AWS region).

- **Render blueprint:** `render.yaml` at the repo root declares `mu3ic-api` (Docker web service from `server/Dockerfile`, `healthCheckPath: /api/health`) and `mu3ic-web` (the `expo export -p web` bundle as a static site with an SPA rewrite). Non-secret config is inlined; secrets (`DATABASE_URL`, `JWT_SECRET`, `REGISTRATION_INVITE_CODE`, `AWS_ACCESS_KEY_ID/SECRET`) are `sync: false` and pasted on first deploy. `DATA_DIR=/tmp/mu3ic` because Render's distroless-nonroot container can't write outside `/tmp` — that's only upload staging, the bytes go straight to Neon.
- **Postgres:** `DATABASE_URL` is the Neon branch's **direct (non-`-pooler`) host**. `store.RunMigrations` takes a session-level `pg_advisory_lock` on startup, which PgBouncer transaction pooling can't hold safely; a single long-lived server also gains nothing from the pooler. The pooled URL is kept commented in `deploy/.env` for any future serverless path.
- **Object storage:** new `library.NeonStorage` (`internal/library/neon_storage.go`), selected by `STORAGE_BACKEND=neon`. It's a near-twin of `R2Storage` — both are `minio-go` S3 clients with presigned-GET streaming — kept as a separate type rather than a shared impl because the constructors genuinely differ: Neon requires **path-style addressing** (`BucketLookupPath`) and signs with a **real region** (`AWS_REGION`, default `us-east-2`), R2 uses virtual-host style and the pseudo-region `auto`. Config reads the `AWS_*` names `neon env pull` writes, plus `NEON_STORAGE_BUCKET` (Neon doesn't inject the bucket name). No `library.Storage` interface change; the stream/artwork handlers' existing `Presigner` type-assertion picks up the 302 path for free.
- **The `deploy/` single-VPS stack** (`docker-compose.yml` with a Postgres container + `Caddyfile` + `backup.sh` + `DEPLOYMENT.md`) is left in the tree unchanged as a superseded alternative; it still describes the R2 + local-Postgres path. `deploy/.env` is repurposed as the (gitignored) list of values to paste into Render.
- **Tooling:** `neon`/`neon-postgres`/`neon-object-storage` agent-skills installed under `.agents/skills/`; Neon CLI authenticated; Neon MCP server added to Claude Code (local scope).

**Trade-offs:** Object Storage and the region lock are a public-beta dependency — if it regresses, `STORAGE_BACKEND=r2` and the `R2_*` code path are still present as the fallback. `neon env pull` rotates the object-storage credential on every run, so re-pulling after go-live means updating the Render service with the new key. The Neon Free plan (`free_v3`) caps history retention at ~6h and doesn't serve the AI Gateway (enabled on the project but unused — its `NEON_AI_GATEWAY_*` vars are ignored); Render's free web tier spins down after 15 min idle (cold start on the next request, and the `/api/health` DB ping pays the Neon resume cost too) — bump `mu3ic-api` to a paid instance for always-on. There is no CI and no automated DB backup on this path yet: Render redeploys on push, and Neon's built-in ~6h restore window is the only safety net until a scheduled `pg_dump` (or a paid Neon plan) is added. `NeonStorage`'s network paths (Put/Open/Delete) are covered by a manual live round-trip against the real `mu3ic-audio` bucket, not unit tests; offline tests cover construction validation, the default-region fallback, and presigned-URL shape (path-style, `us-east-2` credential scope). The web client fetching presigned Neon URLs cross-origin needs a bucket CORS rule (`PutBucketCors`, `GET,HEAD` from the site origin) — not yet set (no deploy yet). `render.yaml` pins `EXPO_PUBLIC_API_URL` to Render's default `https://mu3ic-api.onrender.com`, so the web origin will be `https://mu3ic-web.onrender.com` barring a name collision, making the CORS rule a one-step post-deploy config. Local dev is unchanged — it still uses the repo-root `docker-compose.yml` Postgres and the `fs` storage backend.
