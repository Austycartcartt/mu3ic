# mu3ic server

Go backend: stdlib `net/http`, `database/sql` (via `pgx`), no ORM, no web framework.

## Prerequisites

- Go 1.26.x
- Postgres running (see the root `docker-compose.yml`)

## Setup

```bash
# from the repo root
docker compose up -d

cd server
go run ./cmd/server
```

That's it — migrations in `migrations/` are applied automatically on startup. The server listens on `0.0.0.0:8080` (configurable via `PORT`), so it's reachable from other devices on your LAN as well as `localhost`.

### Config (env vars, all optional)

| Var | Default |
|---|---|
| `PORT` | `8080` |
| `DATABASE_URL` | `postgres://mu3ic:mu3ic@localhost:5432/mu3ic?sslmode=disable` |
| `DATA_DIR` | `./data` |
| `MIGRATIONS_DIR` | `./migrations` |
| `JWT_SECRET` | `dev-secret-change-in-production` (logs a warning when unset) — **set this to a random string before exposing the server** |

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
