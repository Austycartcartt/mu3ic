# mu3ic

A self-hosted music streaming app. See [PROJECT.md](PROJECT.md) for the full spec.

- [`server/`](server/README.md) — Go backend
- [`app/`](app/README.md) — Expo (React Native) client

## Quick start

```bash
docker compose up -d          # Postgres
cd server && go run ./cmd/server   # API on :8080
cd app && npx expo start           # client
```

Physical devices on the same Wi-Fi/LAN reach the API directly at the host machine's LAN IP (e.g. `http://192.168.1.23:8080`) — no tunneling needed. Find it with `ip addr` (Linux) and set it in `app/.env` (see [`app/README.md`](app/README.md)). The Go server binds to `0.0.0.0` so it's reachable from other devices on the network.
