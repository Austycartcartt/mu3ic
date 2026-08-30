# Self-Hosted Music Streaming App — Project Specification

This document describes the project structure, ground rules, and initial scope. It is intended as input to Claude Code for generating the initial project scaffold.

## What This Is

A music streaming application. Users upload their personal music library and stream it from web, iOS, and Android clients.

> **Update (2026-08-28, Phase 8):** the product is moving from "self-hosted, run it on your own LAN" toward a hosted, multi-tenant **private-pilot "music locker"** — each user's library is private to them and only streams back to their own devices. This changes deployment, storage, and auth (see `docs/PHASE-8-deployment.md`, `docs/DECISIONS.md`, and `deploy/DEPLOYMENT.md`), but not the data model — per-user scoping has been in place since Phase 5. The ground rules and stack below still hold; the "LAN-only / filesystem storage / open registration" specifics are now the *dev* configuration.

This is also a learning project. The two goals are equally weighted:
1. Ship a working product
2. Learn Go (backend) and mobile development with Expo/React Native (frontend)

Because learning is a goal, prefer explicit, readable code over clever abstractions. Comments explaining *why* Go idioms are used the way they are is welcome.

## Ground Rules

These are non-negotiable constraints for all generated code:

1. **Latest stable versions.** Go 1.26.x for the backend. For the frontend, use whatever `npx create-expo-app@latest` produces (current stable Expo SDK and its bundled React Native version) — do not pin to an older SDK.
   - **Exception (2026-07-13):** `app/` is currently pinned to Expo SDK 54 instead of latest (was SDK 57) because Expo Go on the App Store only supports SDK 54 and the dev-build alternative requires a paid Apple Developer account. Do not bump back to latest without confirming Expo Go / Apple Developer account status first.
2. **Idiomatic best practices.** Follow standard Go conventions (Effective Go, standard project layout) and current Expo/React Native conventions (Expo Router, TypeScript, functional components with hooks).
3. **Standard library first.** Do not add a dependency until the standard library demonstrably falls short. Every third-party package must be justified. The approved dependency list below is exhaustive for the initial scaffold — do not add anything beyond it without flagging it.

## Repository Layout (Monorepo)

```
music-app/
├── PROJECT.md              # this document
├── docker-compose.yml      # PostgreSQL (dev services only)
├── server/                 # Go backend
└── app/                    # Expo frontend
```

## Backend (`server/`)

### Structure

```
server/
├── go.mod                  # module: standard naming, go 1.26
├── cmd/
│   └── server/
│       └── main.go         # entry point: wiring, config, graceful shutdown
├── internal/
│   ├── api/                # HTTP handlers, routing, middleware
│   │   ├── router.go
│   │   ├── tracks.go       # track list + streaming handlers
│   │   └── upload.go       # upload handler
│   ├── store/              # PostgreSQL access (database/sql, no ORM)
│   │   ├── store.go
│   │   └── tracks.go
│   └── library/            # domain logic: file storage, track management
│       └── library.go
├── migrations/             # plain .sql files, numbered (001_init.sql)
└── data/                   # local audio file storage (gitignored)
```

### Conventions

- **Routing:** Use the standard library `net/http.ServeMux`. Go 1.22+ supports method and path patterns (`mux.HandleFunc("GET /api/tracks/{id}", ...)`), so no router library is needed. Do not add chi unless we hit a concrete limitation.
- **Streaming:** Serve audio with `http.ServeContent` (or `http.ServeFile`). It handles `Range` headers and `206 Partial Content` responses automatically — do not hand-roll range parsing.
- **Database:** `database/sql` with raw SQL queries. No ORM, no query builder. Migrations are plain SQL files applied in order.
- **Storage:** Audio files go to the local filesystem under `server/data/`, organized by track ID. MinIO/S3 is explicitly deferred — design the storage access behind a small interface so it can be swapped later, but implement only the filesystem version.
- **Config:** Environment variables read in `main.go` (stdlib `os.Getenv` with sane defaults). No config library.
- **Logging:** `log/slog` (stdlib structured logging).
- **Errors:** Idiomatic error wrapping with `fmt.Errorf("...: %w", err)`. Handlers return JSON error bodies with appropriate status codes.
- **Testing:** stdlib `testing` package, table-driven tests where natural. `net/http/httptest` for handler tests.

### Approved backend dependencies

| Package | Purpose | Justification |
|---|---|---|
| `github.com/jackc/pgx/v5` (as `database/sql` driver via `pgx/v5/stdlib`) | PostgreSQL driver | stdlib has no Postgres driver; pgx is the maintained standard (`lib/pq` is in maintenance mode) |
| `github.com/dhowden/tag` | Audio metadata extraction | Phase 2 only — do not include in the initial scaffold |
| `github.com/golang-jwt/jwt/v5`, `golang.org/x/crypto` (bcrypt) | Auth tokens + password hashing | Phase 5 — security-sensitive code you don't want to hand-roll |
| `github.com/minio/minio-go/v7` | Cloudflare R2 (S3 API) client | Phase 8 — object storage backend; lighter than `aws-sdk-go-v2`, and SigV4 signing isn't code to own |

That's it. No web frameworks, no ORM, no config/env libraries, no logging libraries.

## Frontend (`app/`)

### Structure

Scaffold with `npx create-expo-app@latest` using the default TypeScript template, then simplify. Target structure:

```
app/
├── app/                    # Expo Router file-based routes
│   ├── _layout.tsx
│   └── index.tsx           # track list screen (v1: the only screen)
├── src/
│   ├── api/
│   │   └── client.ts       # typed fetch wrapper for the Go backend
│   ├── components/
│   │   └── TrackList.tsx
│   └── hooks/
│       └── usePlayer.ts    # wraps expo-audio playback state
├── app.json
└── tsconfig.json
```

### Conventions

- **TypeScript strict mode** throughout.
- **Audio:** Use `expo-audio`. Never `expo-av` — it is deprecated, and any tutorial or example referencing it should be ignored.
- **Navigation:** Expo Router (file-based). It ships with the default template.
- **Data fetching:** Plain `fetch` with a thin typed wrapper. No axios, no react-query — revisit only if state synchronization becomes painful.
- **State:** React built-ins (`useState`, `useContext`). No Redux/Zustand/Jotai until there's demonstrated need.
- **Styling:** React Native `StyleSheet`. No styling libraries.
- **API base URL:** Read from an environment variable (`EXPO_PUBLIC_API_URL`) so physical devices can point at the WSL2 host / tunnel address.

### Approved frontend dependencies

Only what the Expo template ships with, plus `expo-audio` (installed via `npx expo install expo-audio`). Nothing else in v1.

## Dev Environment

- **Linux Mint**, native development (no VMs, no WSL).
- Go installed natively (official tarball or via the distro's toolchain — prefer the official go.dev tarball to guarantee the latest stable).
- PostgreSQL runs via **Docker Compose** (Docker Engine + compose plugin, not Docker Desktop). Compose file defines only Postgres for now; MinIO deferred.
- Physical devices on the same LAN can reach the dev server directly at the machine's LAN IP — no tunneling needed. The Go server should bind to `0.0.0.0`, and `EXPO_PUBLIC_API_URL` should point at the LAN IP (e.g. `http://192.168.x.x:8080`). Document this in the README.

## Initial Scope: Thin Vertical Slice (Phase 1)

Build only the following, end to end:

1. **Health check:** `GET /api/health` returns 200 — proves Go ↔ Expo connectivity.
2. **Upload:** `POST /api/tracks` accepts a multipart audio file, writes it to `server/data/`, inserts a row in PostgreSQL (id, filename, size, created_at). A curl example in the README is sufficient — no in-app upload UI yet.
3. **List:** `GET /api/tracks` returns the track list as JSON.
4. **Stream:** `GET /api/tracks/{id}/stream` serves the audio file via `http.ServeContent` with range-request support.
5. **Client:** Single screen listing tracks; tapping a track plays it via `expo-audio`.

### Explicitly Deferred (do not scaffold, stub, or "prepare for")

_Scaffold-era list. Most items below have since shipped in their own phases (auth = 5, playlists/search = 6); MinIO/S3 storage and deployment config landed in Phase 8 (as Cloudflare R2 + `deploy/`). `docs/STATUS.md` is the current picture._

- Authentication
- Metadata extraction (Phase 2 — `dhowden/tag`)
- Background playback / lock-screen controls
- Playlists, search
- In-app upload UI
- Transcoding (ffmpeg)
- MinIO/S3 storage
- HLS/DASH — HTTP range requests are the chosen transport; adaptive bitrate is out of scope unless it becomes a proven need
- Deployment configuration

Deferred means deferred: no empty auth middleware, no placeholder playlist tables, no half-wired abstractions. The only forward-looking design allowance is the small storage interface noted above.

## Definition of Done for the Scaffold

- `docker compose up -d` starts Postgres
- `go run ./cmd/server` starts the API with migrations applied (or a documented one-command migration step)
- `npx expo start` in `app/` launches the client
- Uploading a file via curl, seeing it in the app, and playing it works end to end
- README in each of `server/` and `app/` covering setup in under 5 minutes
