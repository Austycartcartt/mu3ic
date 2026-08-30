# Deploying mu3ic on Render + Neon

The production topology: the app runs on **Render**, all state lives on
**Neon**.

```
                    ┌──────────── Render ────────────┐
  browser / app ───▶│  mu3ic-web  (static Expo build) │
        │           │  mu3ic-api  (Docker, Go)  :PORT │
        │           └───────────────┬────────────────┘
        │                           │ pgx (direct, sslmode=require)
        │                           ▼
        │                   Neon Lakebase Postgres  (branch: production)
        │
        └── GET /stream, /artwork ──▶ mu3ic-api ──302──▶ Neon Object Storage
                                                          (bucket: mu3ic-audio)
  browser / app ◀────────────── presigned GET (bytes) ───┘
```

- `mu3ic-api` holds **no** persistent disk. Uploads are staged in
  `/tmp/mu3ic` and pushed straight to Neon Object Storage; audio and
  artwork bytes are served by a `302` redirect to a short-lived presigned
  URL, so they never transit the API.
- Everything is declared in the repo-root [`render.yaml`](../render.yaml).
  Non-secret config is inlined there; secrets are `sync: false` and
  entered once on the first deploy.
- The older single-VPS stack in [`deploy/`](../deploy/DEPLOYMENT.md)
  (docker compose + Caddy + local Postgres + Cloudflare R2) is a
  **superseded alternative** — ignore it unless you specifically want
  that path.

---

## 0. Prerequisites

- A **Neon** account. Every Neon resource for mu3ic must be in
  **`us-east-2`** — Neon Object Storage is a public beta and serves only
  that region.
- A **Render** account connected to this GitHub repo.
- Local tooling (only needed to fetch connection values):
  - Neon CLI — `npm i -g neon`, then `neon auth`. (Invoked as `neon`.)
  - `openssl` for secret generation, and the AWS CLI (`aws`) for the
    bucket CORS rule in step 4b.
- The code committed and pushed to the branch Render will track
  (`main`).

Two host names are used throughout. With the service names in
`render.yaml` and no global name collision, Render assigns:

| Service     | Default URL                       | Role                    |
|-------------|-----------------------------------|-------------------------|
| `mu3ic-api` | `https://mu3ic-api.onrender.com`  | API origin              |
| `mu3ic-web` | `https://mu3ic-web.onrender.com`  | the app you open (demo) |

If Render appends a suffix because a name was taken, substitute the real
URLs everywhere below (and in `render.yaml` — see step 4).

---

## 1. Neon: create the data layer

If the Neon project already exists (`sweet-star-53712486`, branch
`production`, bucket `mu3ic-audio`), do just **1d** to re-pull
credentials, and grab the connection string per **1b**.

### 1a. Project and branch

In the **Neon console**: create a project, region **AWS `us-east-2`**.
It ships with one branch — this guide calls it `production` (rename the
default, or make a new branch). Then link this repo to it so later CLI
calls need no IDs:

```bash
neon link        # interactive: pick org / project / branch. Writes a
                 # gitignored .neon file (already in .gitignore).
```

`neon link` also pulls the branch's env on the way out.

### 1b. Postgres connection string (direct, not pooled)

mu3ic runs schema migrations on startup and holds a **session-level**
`pg_advisory_lock` while it does. That is incompatible with PgBouncer
transaction pooling, so the app must use the **direct** (non-`-pooler`)
host.

- **Console:** branch → *Connect* → **turn Connection pooling OFF** →
  copy the string.
- **CLI:** `neon connection-string` — it returns the **direct** URL by
  default.

Confirm the host does **not** contain `-pooler` and the string ends with
`?sslmode=require`. Keep it for `DATABASE_URL` in step 2. (`neon env
pull` also writes it as `DATABASE_URL`, plus the pooled one as
`DATABASE_URL_UNPOOLED` — for mu3ic you want the **un**pooled/direct
one.)

### 1c. Object Storage bucket

- **Console:** *Object Storage* → create bucket `mu3ic-audio` on the
  `production` branch, access **private** (the default — mu3ic only ever
  serves presigned URLs, never anonymous reads).
- **CLI:** `neon bucket create mu3ic-audio` then `neon bucket list` to
  verify.

### 1d. Pull the storage credentials

```bash
neon env pull        # reads .neon; writes vars into ./.env (or .env.local)
```

The repo-root `/.env` and `/.env.local` are gitignored. This writes
AWS-standard vars:

| Var                     | Used for                                             |
|-------------------------|-----------------------------------------------------|
| `AWS_ENDPOINT_URL_S3`   | branch S3 endpoint (`https://…storage.…us-east-2.aws.neon.tech`) |
| `AWS_REGION`            | `us-east-2`                                         |
| `AWS_ACCESS_KEY_ID`     | S3 key id (branch-scoped)                           |
| `AWS_SECRET_ACCESS_KEY` | S3 secret                                           |

> **`neon env pull` rotates the storage credential each run.** After
> go-live, only re-pull when you intend to rotate — then update
> `mu3ic-api` with the new key/secret (see *Rotating secrets* in step 6).

The bucket name (`mu3ic-audio`) is **not** injected; mu3ic reads it from
`NEON_STORAGE_BUCKET`, already inlined in `render.yaml`.

---

## 2. Fill in `deploy/.env`

`deploy/.env` is gitignored. It is not read by Render automatically — it
is your worksheet of the values to paste into the Render dashboard in
step 3. Copy the template and fill it in:

```bash
cp deploy/.env.example deploy/.env
chmod 600 deploy/.env
```

The five values Render will prompt for (`sync: false` in `render.yaml`):

| Render env var          | Where it comes from                                             |
|-------------------------|----------------------------------------------------------------|
| `DATABASE_URL`          | step 1b — the **direct** Neon string, `?sslmode=require`        |
| `JWT_SECRET`            | `openssl rand -hex 32` — **≥ 32 chars or the API won't boot**   |
| `REGISTRATION_INVITE_CODE` | `openssl rand -hex 24` — hand this to pilot users            |
| `AWS_ACCESS_KEY_ID`     | step 1d (`neon env pull`)                                       |
| `AWS_SECRET_ACCESS_KEY` | step 1d (`neon env pull`)                                       |

Everything else the API needs is non-secret and already in `render.yaml`:
`STORAGE_BACKEND=neon`, `NEON_STORAGE_BUCKET=mu3ic-audio`,
`AWS_ENDPOINT_URL_S3`, `AWS_REGION=us-east-2`, `DATA_DIR=/tmp/mu3ic`,
`TRUST_PROXY=true`, `REGISTRATION_OPEN=false`, `STREAM_URL_TTL=15m`.

> If you rotated to a **new bucket** or a different branch, also update
> the inlined `AWS_ENDPOINT_URL_S3` / `NEON_STORAGE_BUCKET` in
> `render.yaml` and commit — the endpoint encodes the branch.

---

## 3. Deploy the Render blueprint

1. Render dashboard → **New** → **Blueprint**.
2. Pick this repo. Render reads `render.yaml` and shows two services:
   `mu3ic-api` (Docker) and `mu3ic-web` (static site).
3. It prompts for each `sync: false` var on `mu3ic-api` — paste the five
   values from `deploy/.env`. `mu3ic-web`'s `EXPO_PUBLIC_API_URL` is
   already inlined in `render.yaml` as `https://mu3ic-api.onrender.com`,
   so there is nothing to enter for the web service (revisit only in
   step 4a if the API URL turns out to differ).
4. **Apply**. First build takes a few minutes each (Go compile for the
   API; `npm ci` + `expo export` for the web bundle).

Watch `mu3ic-api` logs for:

```
running migrations … applied migration 00X …
starting server  addr=0.0.0.0:10000
```

Render marks the service healthy once `GET /api/health` returns `200` —
that endpoint does a real 2 s DB ping, so a green check means Neon
Postgres is reachable from Render.

---

## 4. Post-deploy wiring

### 4a. Confirm the API URL the web build baked in

Open the `mu3ic-api` service in Render and copy its actual URL. If it is
**not** `https://mu3ic-api.onrender.com` (name collision → suffix):

1. Edit `render.yaml` → `mu3ic-web` → `EXPO_PUBLIC_API_URL` `value:` to
   the real API URL, commit, push. Render picks the new value up from the
   blueprint on the next sync.
2. Trigger `mu3ic-web` → **Manual Deploy → Clear build cache & deploy**
   so the value is re-baked into the static bundle (an `EXPO_PUBLIC_*`
   change only takes effect on a rebuild).

`EXPO_PUBLIC_*` vars are inlined at **build** time, so changing this
always requires a `mu3ic-web` rebuild, not just a restart.

### 4b. Bucket CORS rule (only if the browser blocks playback)

Plain `<audio>` / `<img>` playback across origins does **not** need CORS,
so this may be unnecessary. Do it if the browser console shows a CORS
error on a `*.storage.*.aws.neon.tech` URL during step 5. Add a rule
allowing `GET`/`HEAD` from the web origin, via the AWS CLI with the vars
from `neon env pull`:

```bash
cat > /tmp/mu3ic-cors.json <<'JSON'
{
  "CORSRules": [
    {
      "AllowedOrigins": ["https://mu3ic-web.onrender.com"],
      "AllowedMethods": ["GET", "HEAD"],
      "AllowedHeaders": ["*"],
      "ExposeHeaders": ["Content-Length", "Content-Range", "Accept-Ranges"],
      "MaxAgeSeconds": 3600
    }
  ]
}
JSON

aws s3api put-bucket-cors \
  --endpoint-url "$AWS_ENDPOINT_URL_S3" \
  --bucket mu3ic-audio \
  --cors-configuration file:///tmp/mu3ic-cors.json
```

Use the real `mu3ic-web` origin if it has a suffix or a custom domain.
Add more origins to `AllowedOrigins` for native/EAS builds or a staging
site. If `put-bucket-cors` is rejected (the beta may not implement it
yet), check the Neon storage docs — but first confirm you actually have a
CORS problem and not, say, a wrong `EXPO_PUBLIC_API_URL`.

---

## 5. Smoke test and first user

```bash
# API is up and can reach the DB
curl https://mu3ic-api.onrender.com/api/health          # {"status":"ok"}
```

Registration is closed, but the **first** account is always allowed as a
bootstrap (zero-users check), no invite code needed:

```bash
curl -X POST https://mu3ic-api.onrender.com/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"a-good-password"}'
```

After that, `POST /api/auth/register` returns `403` unless the body
carries `"inviteCode"` matching `REGISTRATION_INVITE_CODE`. Hand that
code to pilot users.

Then, end to end:

1. Open `https://mu3ic-web.onrender.com`, log in.
2. Upload a track.
3. Play it. In DevTools → Network, `GET /api/tracks/{id}/stream?token=…`
   should be a **302** to a `*.storage.*.aws.neon.tech` URL, and audio
   should play from there with no CORS error in the console.

---

## 6. Operations

**Deploys.** `autoDeploy: true` — push to the tracked branch and Render
rebuilds both services. Migrations run automatically on `mu3ic-api`
start, serialized by the `pg_advisory_lock`.

**Logs.**
- API: Render dashboard → `mu3ic-api` → Logs (structured `slog` lines,
  one per request with `X-Request-Id`, method, path, status, duration).
- Storage: `neon logs query --source storage --since 1h` (add `--branch
  production` if `.neon` points elsewhere). Needs Neon CLI ≥ 3.1.

**Monitoring.** Point an external uptime check (UptimeRobot,
healthchecks.io, …) at `https://mu3ic-api.onrender.com/api/health` and
alert on non-`200` — a `503 {"status":"degraded"}` means the API is up
but Neon is unreachable. Also check `https://mu3ic-web.onrender.com/`.

**Backups.** There is **no** automated DB backup on this path yet. Neon's
built-in restore history is the only net (~6 h on the Free plan). Add a
scheduled `pg_dump` against the direct URL, or move to a paid Neon plan
with longer history, before the pilot holds data you care about. Audio
durability is Neon Object Storage's.

**Cost / cold starts.** On free tiers both services sleep after ~15 min
idle; the next request pays a cold start, and the `/api/health` DB ping
also wakes the Neon compute from scale-to-zero. For an always-on demo,
bump `mu3ic-api` to Render's `starter` plan (`plan: starter` in
`render.yaml`).

**Rotating secrets.** Change the value in the Render dashboard → the
service redeploys. For the storage credential, re-run `neon env pull`,
then update `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` on `mu3ic-api`.

---

## 7. Troubleshooting

| Symptom | Likely cause |
|---|---|
| `mu3ic-api` exits immediately, log says `JWT_SECRET must be set…` | `JWT_SECRET` unset or < 32 chars. |
| Exits with `required environment variable is not set` `var=AWS_…` / `NEON_STORAGE_BUCKET` | `STORAGE_BACKEND=neon` but a storage var is missing. Re-check step 2 and the inlined vars. |
| `connecting to database` error on boot | `DATABASE_URL` wrong, missing `?sslmode=require`, or points at the `-pooler` host. |
| Boot hangs on `running migrations`, or `prepared statement … already exists` | Using the **pooled** connection string. Switch to the direct host. |
| Upload returns 500, log mentions `PutObject` / `SignatureDoesNotMatch` | Stale storage key (a later `neon env pull` rotated it), or `AWS_REGION` ≠ `us-east-2`. |
| Track won't play; console shows a CORS error on the `*.neon.tech` URL | Bucket CORS rule missing or the origin doesn't match. Redo step 4b with the exact `mu3ic-web` origin. |
| App loads but every API call 404s / hits `localhost` | `mu3ic-web` was built with the wrong `EXPO_PUBLIC_API_URL`. Fix and rebuild (step 4a). |
| `/api/health` returns `503 {"status":"degraded"}` | API is up, Neon Postgres is not reachable — check the Neon project isn't suspended and the URL is current. |

---

## 8. Fallbacks

- **Object storage → Cloudflare R2.** The `r2` backend is still in the
  code. Set `STORAGE_BACKEND=r2` and the four `R2_*` vars on `mu3ic-api`
  (see `deploy/.env.example`); no redeploy of the web service needed.
- **Whole stack → single VPS.** [`deploy/DEPLOYMENT.md`](../deploy/DEPLOYMENT.md)
  is the unchanged docker-compose + Caddy + local-Postgres + R2
  walkthrough.
