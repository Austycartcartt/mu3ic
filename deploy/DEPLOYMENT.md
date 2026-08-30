# Deploying mu3ic (private pilot)

> **Superseded (2026-08-29).** Production now runs on **Render + Neon** —
> see the repo-root `render.yaml` and the 2026-08-29 entry in
> `docs/DECISIONS.md` / `docs/PHASE-8-deployment.md`. This single-VPS +
> Caddy + local-Postgres + Cloudflare R2 walkthrough is kept as an
> alternative and is no longer the path in use.

The pilot runs on **one Linux VPS** with `docker compose`: Postgres, the Go
API, and Caddy (TLS + static web + reverse proxy). Uploaded audio and
artwork live in **Cloudflare R2**, not on the VPS — the server only holds
the metadata database and streams by handing clients short-lived
presigned R2 URLs.

```
                 ┌─────────────────────── VPS ───────────────────────┐
  browser / app ─┼─▶ caddy :443 ──▶ server :8080 ──▶ db :5432          │
                 │        │            │                               │
                 │        └─ /srv/www  └─ presign ──┐                  │
                 └──────────────────────────────────┼──────────────────┘
                                                    ▼
        browser / app ◀── 302 ────────────  Cloudflare R2 bucket
```

## 1. Prerequisites

- A VPS: 2 vCPU / 2–4 GB RAM / 20–40 GB disk (audio is in R2, so disk is
  just the OS, Postgres, and local DB backups). Ubuntu 24.04 assumed below.
- A domain, e.g. `music.example.com`.
- A Cloudflare account (free tier is fine for R2 at pilot scale).

## 2. DNS

Create an `A` record (and `AAAA` if you have IPv6) for your hostname
pointing at the VPS public IP. Wait for it to resolve before step 6 —
Caddy needs it to obtain a certificate.

## 3. Provision the VPS

```bash
# as root, then switch to a sudo user for the rest
adduser deploy && usermod -aG sudo deploy

ufw allow OpenSSH
ufw allow 80,443/tcp
ufw enable

# Docker Engine + compose plugin (official repo)
curl -fsSL https://get.docker.com | sh
usermod -aG docker deploy
```

Log back in as `deploy`.

## 4. Create the R2 bucket

In the Cloudflare dashboard → R2:

1. Create a bucket, e.g. `mu3ic-audio`.
2. **Settings → Enable versioning**, and add a lifecycle rule expiring
   *noncurrent* versions after ~30 days (cheap insurance against an
   accidental or buggy delete).
3. **Manage R2 API Tokens → Create API token**, scoped to *Object Read &
   Write* on that bucket. Record the **Access Key ID**, **Secret Access
   Key**, and the **S3 API endpoint**
   (`https://<account-id>.r2.cloudflarestorage.com`).
4. **Settings → CORS policy**: allow `GET, HEAD` from
   `https://music.example.com` (so the web `<audio>`/`<img>` elements can
   fetch presigned URLs cross-origin).

## 5. Clone and configure

```bash
sudo mkdir -p /opt/mu3ic && sudo chown deploy:deploy /opt/mu3ic
git clone <repo-url> /opt/mu3ic
cd /opt/mu3ic/deploy

cp .env.example .env
chmod 600 .env
# Fill in .env — at minimum SITE_ADDRESS, ACME_EMAIL, POSTGRES_PASSWORD,
# JWT_SECRET, REGISTRATION_INVITE_CODE, and the four R2_* values.
#   JWT_SECRET:               openssl rand -hex 32
#   REGISTRATION_INVITE_CODE: openssl rand -hex 24
#   POSTGRES_PASSWORD:        openssl rand -base64 30
```

Run compose commands from `/opt/mu3ic/deploy` so it picks up `.env`.

## 6. Launch

```bash
docker compose -f docker-compose.yml up -d --build
docker compose -f docker-compose.yml logs -f
```

Watch for: `applied migration 00X …` (server), `starting server` (server),
and Caddy obtaining a certificate for your hostname. First build takes a
few minutes (Go compile + `npm ci` + `expo export`).

## 7. Smoke test

```bash
curl https://music.example.com/api/health          # {"status":"ok"}
```

Open `https://music.example.com/` — the web app should load.

## 8. Create the first user

Registration is closed, but the **first** account is allowed as a
bootstrap:

```bash
curl -X POST https://music.example.com/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"a-good-password"}'
```

After that, `POST /api/auth/register` returns `403` unless the request
body carries an `inviteCode` matching `REGISTRATION_INVITE_CODE`. That
value is already set in `.env`, so hand the code to pilot users. It's read
at server startup — if you change it later, run `docker compose up -d` to
recreate the server container.

## 9. Client check

Log in from the web app, upload a track, and play it. In browser
DevTools, `GET /api/tracks/{id}/stream?token=…` should be a **302** to an
`*.r2.cloudflarestorage.com` URL, and audio should play from there. Native
clients (iOS/Android): set `EXPO_PUBLIC_API_URL=https://music.example.com`
in `app/.env` for a local `expo run` or a future EAS build.

## 10. Backups

```bash
sudo mkdir -p /var/backups/mu3ic && sudo chown deploy:deploy /var/backups/mu3ic
/opt/mu3ic/deploy/backup.sh                 # run once, confirm a .dump appears
pg_restore --list /var/backups/mu3ic/mu3ic-*.dump | head   # sanity check

crontab -e
# 15 3 * * * /opt/mu3ic/deploy/backup.sh >> /var/log/mu3ic-backup.log 2>&1
```

Restore (metadata only; audio is in R2):

```bash
cd /opt/mu3ic/deploy
docker compose -f docker-compose.yml stop server
docker compose -f docker-compose.yml exec -T db \
  pg_restore -U mu3ic -d mu3ic --clean --if-exists < /var/backups/mu3ic/<dump>
docker compose -f docker-compose.yml start server
```

## 11. Monitoring

Point an external uptime monitor (healthchecks.io, UptimeRobot, …) at
`https://music.example.com/api/health` (alert on non-200) and at
`https://music.example.com/` (catches cert/edge problems). Logs:
`docker compose logs -f` — structured JSON from the server, token-redacted
access logs from Caddy, both rotated by the compose `logging:` limits.

## 12. Updating

```bash
cd /opt/mu3ic && git pull
cd deploy && docker compose -f docker-compose.yml up -d --build
```

Migrations apply automatically on server start, serialized by a Postgres
advisory lock.

## Alternatives / notes

- **Lighter Caddy image:** instead of the Node build stage in
  `Caddy.Dockerfile`, run `npx expo export -p web` on the host (or in CI)
  and bind-mount `app/dist` into a stock `caddy:2-alpine`. Keeps Node off
  the VPS at the cost of a manual build step.
- **Migrating an existing local library into R2:** the storage keys are
  already bare filenames, so `rclone copy <old DATA_DIR> r2:<bucket>` is
  all it takes — no code involved.
- **Local `fs` backend:** `STORAGE_BACKEND=fs` (the default) keeps audio
  under `DATA_DIR` and serves it directly. That's the dev path; the pilot
  uses `r2`.
