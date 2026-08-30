# Phase 8: Deployment

**Status:** In progress — private-pilot infrastructure. The Render + Neon
deployment is fully described in `render.yaml` and self-wiring (Render's
default `*.onrender.com` hostnames are pinned); the first blueprint
deploy is the remaining step (2026-08-29).

## Goal

Take mu3ic from a LAN-only personal app to **private-pilot SaaS
infrastructure**: online, multi-tenant, HTTPS, object storage, backups,
monitoring, and core auth hardening — enough to onboard invited pilot
users. The product framing is a "music locker": each user uploads their
own library, it stays private to them (every query is already scoped by
`user_id`), and it only streams back to their own devices.

Billing, storage-quota enforcement, a marketing site, legal/policy pages,
and email flows are **later phases** on the road to a full public launch
(see *Deferred*).

## Delivered

### Object storage
- `library.Storage` gained a write side (`Put`/`Delete`) and an optional
  `Presigner` capability; the old read-only interface + `os.Rename` in
  `IngestFile` only worked on a local disk.
- Three backends, selected by `STORAGE_BACKEND`: `fs` (local, dev — serves
  via `http.ServeContent`), `neon` (**Neon Object Storage**, the pilot
  default as of 2026-08-29), and `r2` (Cloudflare R2, the original
  implementation, kept as a fallback). Both object backends are `minio-go`
  S3 clients; `/stream` and `/artwork` 302-redirect to a short-lived
  presigned URL, so track bytes never transit the app server.
  `NeonStorage` differs from `R2Storage` only at construction (path-style
  addressing, real `AWS_REGION`).
- No schema change: `storage_key` is still a bare UUID; the artwork key is
  `storage_key + artwork_ext`.

### Hosting (Render + Neon, as of 2026-08-29)
- Step-by-step runbook: [`DEPLOY-RENDER-NEON.md`](DEPLOY-RENDER-NEON.md).
- The data layer is entirely on **Neon** (project `sweet-star-53712486`,
  branch `production`, region `us-east-2`): Lakebase Postgres + Neon
  Object Storage. Nothing stateful runs on the app host.
- The app runs on **Render**, declared in the repo-root `render.yaml`:
  - `mu3ic-api` — Docker web service built from `server/Dockerfile`,
    `healthCheckPath: /api/health`, `region: ohio` (same AWS region as
    Neon), `DATA_DIR=/tmp/mu3ic` for upload staging. Render terminates
    TLS and injects `PORT`.
  - `mu3ic-web` — the `expo export -p web` bundle as a static site, with
    an SPA rewrite to `index.html` and a long cache on `/_expo/*`.
  - Secrets (`DATABASE_URL`, `JWT_SECRET`, `REGISTRATION_INVITE_CODE`,
    `AWS_ACCESS_KEY_ID`/`SECRET`) are `sync: false`; values live in the
    gitignored `deploy/.env`, to be pasted into the Render services on
    the first deploy. `mu3ic-web`'s `EXPO_PUBLIC_API_URL` is inlined in
    `render.yaml` as `https://mu3ic-api.onrender.com` (Render's default
    hostname for the `mu3ic-api` service), so the blueprint wires the web
    build to the API with no manual step — swap in the suffixed hostname
    or a custom domain if Render assigns something else.
- **Superseded:** the `deploy/` single-VPS stack (`docker-compose.yml`
  with a Postgres container, `Caddyfile`, `Caddy.Dockerfile`,
  `backup.sh`, `DEPLOYMENT.md`) is left in the tree unchanged as an
  alternative; it still describes the local-Postgres + Cloudflare R2 path.

### Auth hardening (core)
- `JWT_SECRET` is now **fatal** if unset, the old dev default, or < 32
  chars (was a warning).
- Registration is **closed** by default: `REGISTRATION_INVITE_CODE` gates
  new accounts (constant-time compare), with a zero-users first-run
  bootstrap so a fresh deployment can create its first account.
- The two auth endpoints are rate-limited per client IP (hand-rolled token
  bucket); `X-Real-IP` is trusted only when `TRUST_PROXY=true`.
- `/api/health` now does a 2s database ping and returns `503
  {"status":"degraded"}` when it fails, so an uptime monitor catches a
  server that's up but DB-less.
- Every request gets a short `X-Request-Id`, logged with method / path /
  status / duration.

### Operations (Render + Neon path)
- **Backups:** on this path there is no scheduled `pg_dump` yet — Neon's
  built-in restore history (~6h on Free) is the only DB safety net until a
  cron `pg_dump` or a paid Neon plan is added. Audio durability is Neon
  Object Storage's (it branches with the DB). The `deploy/backup.sh`
  script targets the *superseded* VPS Postgres container, not Neon.
- **Monitoring:** external uptime check on the `mu3ic-api` `/api/health`
  URL; Render's own logs/metrics for the service;
  `neon logs query --source storage` for the bucket.
- **Deploys:** `autoDeployTrigger: commit` in `render.yaml` — push to the
  tracked branch and Render rebuilds. Migrations run on server start
  (advisory lock serialized).

## Verification

`go vet ./... && go test ./...` pass; `docker build ./server` builds the
API image. The Neon backends were verified live from the real Go code
paths: a pgx `SELECT version()` against the `production` branch, and a
`NeonStorage` Put → PresignGet (HTTP 200) → Open → Delete round-trip
against the `mu3ic-audio` bucket. Still outstanding: the first
`render.yaml` blueprint deploy — the blueprint is complete and
self-wiring (`EXPO_PUBLIC_API_URL` is pinned to the default
`https://mu3ic-api.onrender.com`), so a deploy needs only the `sync:
false` secrets pasted in. Then two post-deploy steps: add the
`mu3ic-audio` bucket CORS rule (`GET,HEAD` from
`https://mu3ic-web.onrender.com`, or whatever hostname Render assigns
the web service), and verify end to end — register the bootstrap user
and confirm an upload → presigned `302` → playback round-trip from the
web client.

## Deferred (later phases)

Stripe billing · storage-quota enforcement (per-user bytes = `SUM(size)`
already) · email verification + password reset + emailed invites ·
streaming-scoped / refresh tokens (the full JWT still rides `?token=` on
`/stream`, now narrowed: log redaction + short-lived presigned URLs) ·
marketing site · Terms/Privacy/DMCA/acceptable-use + data export &
deletion · transcoding (Phase 9) · CI/CD · horizontal scaling / HA · a
secrets manager (the pilot uses a `chmod 600` `.env`).
