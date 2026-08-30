# mu3ic server

Go backend: stdlib `net/http`, `database/sql` (via `pgx`), no ORM, no web framework.

## Prerequisites

- Go 1.26.x
- Postgres running (see the root `docker-compose.yml`)

## Setup

```bash
# from the repo root — start.sh also exports a dev JWT_SECRET
./start.sh

# ...or run the server directly:
docker compose up -d
cd server
JWT_SECRET=dev-secret-not-for-production-0123456789abcdef go run ./cmd/server
```

That's it — migrations in `migrations/` are applied automatically on startup.
`JWT_SECRET` is required (see Config below); everything else has a dev default. The server listens on `0.0.0.0:8080` (configurable via `PORT`), so it's reachable from other devices on your LAN as well as `localhost`.

### Config (env vars)

| Var | Default | Notes |
|---|---|---|
| `PORT` | `8080` | |
| `DATABASE_URL` | `postgres://mu3ic:mu3ic@localhost:5432/mu3ic?sslmode=disable` | |
| `DATA_DIR` | `./data` | upload staging; also the object dir when `STORAGE_BACKEND=fs` |
| `MIGRATIONS_DIR` | `./migrations` | |
| `JWT_SECRET` | — | **required**; server exits if unset, the old dev default, or < 32 chars |
| `STORAGE_BACKEND` | `fs` | `fs` (local files), `neon` (Neon Object Storage), or `r2` (Cloudflare R2); both object backends use presigned streaming |
| `AWS_ENDPOINT_URL_S3` / `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` / `NEON_STORAGE_BUCKET` | — | required when `STORAGE_BACKEND=neon` (`AWS_*` come from `neon env pull`; `AWS_REGION` optional, defaults to `us-east-2`) |
| `R2_ENDPOINT` / `R2_BUCKET` / `R2_ACCESS_KEY_ID` / `R2_SECRET_ACCESS_KEY` | — | required when `STORAGE_BACKEND=r2` |
| `STREAM_URL_TTL` | `15m` | lifetime of presigned stream/artwork URLs |
| `REGISTRATION_OPEN` | `false` | `true` lets anyone register (dev/staging only) |
| `REGISTRATION_INVITE_CODE` | — | when set, `register` requires a matching `inviteCode`; the first account is always allowed |
| `TRUST_PROXY` | `false` | `true` reads the client IP from `X-Real-IP` (set only behind a trusted proxy) |

For local dev, `./start.sh` exports a throwaway `JWT_SECRET` for you.

## Trying it out

Every endpoint except `/api/health` and `/api/auth/*` now requires a bearer
token. Register (or log in) to get one:

```bash
# health check (public)
curl localhost:8080/api/health

# register — returns {"id", "email", "token", "expiresAt"}
curl -X POST localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"password123"}'

TOKEN=<paste the token>

# upload a file
curl -F "audio=@/path/to/song.mp3;type=audio/mpeg" \
  -H "Authorization: Bearer $TOKEN" localhost:8080/api/tracks

# list your tracks
curl -H "Authorization: Bearer $TOKEN" localhost:8080/api/tracks

# stream (supports Range requests / 206 Partial Content). The audio player
# can't set headers, so /stream and /artwork also accept ?token=
curl -v -H "Range: bytes=0-1023" \
  "localhost:8080/api/tracks/1/stream?token=$TOKEN" -o /dev/null
```

## Tests

```bash
go vet ./...
go test ./...
```

## Finding your LAN IP

Physical devices (running the Expo app) need to reach this server over the network, not `localhost`. Find your machine's LAN IP with:

```bash
ip addr show | grep 'inet ' | grep -v 127.0.0.1
```

Use that address (e.g. `http://192.168.1.23:8080`) as `EXPO_PUBLIC_API_URL` in `app/.env`.
