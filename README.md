# mu3ic

A music streaming app — upload your own library, stream it to your own devices. See [PROJECT.md](PROJECT.md) for the full spec. (Moving toward a hosted private-pilot "music locker" as of Phase 8.)

- [`server/`](server/README.md) — Go backend
- [`app/`](app/README.md) — Expo (React Native) client
- [`docs/DEPLOY-RENDER-NEON.md`](docs/DEPLOY-RENDER-NEON.md) — **step-by-step production deploy**: Go API + Expo web on Render, data layer (Postgres + object storage) on Neon. Blueprint is [`render.yaml`](render.yaml).
- [`deploy/`](deploy/DEPLOYMENT.md) — superseded single-VPS deploy (docker compose, Caddy, local Postgres, Cloudflare R2); kept as an alternative
- [`docs/STATUS.md`](docs/STATUS.md) — current build phase and what's next
- [`docs/DECISIONS.md`](docs/DECISIONS.md) — architecture decisions log

## Quick start

```bash
./start.sh   # Postgres, API on :8080, and the Expo client together
```

Or run each piece separately:

```bash
docker compose up -d          # Postgres
cd server && go run ./cmd/server   # API on :8080
cd app && npx expo start           # client
```

Physical devices on the same Wi-Fi/LAN reach the API directly at the host machine's LAN IP (e.g. `http://192.168.1.23:8080`) — no tunneling needed. Find it with `ip addr` (Linux) and set it in `app/.env` (see [`app/README.md`](app/README.md)). The Go server binds to `0.0.0.0` so it's reachable from other devices on the network.
