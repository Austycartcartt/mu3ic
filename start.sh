#!/usr/bin/env bash
# Starts Postgres, the Go API server, and the Expo app together.
# Ctrl+C stops the server and Expo processes (Postgres keeps running via docker compose).
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

docker compose up -d

pids=()
cleanup() {
    trap - INT TERM EXIT
    for pid in "${pids[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    wait 2>/dev/null || true
}
trap cleanup INT TERM EXIT

( cd server && go run ./cmd/server 2>&1 | sed -e "s/^/[server] /" ) &
pids+=($!)

# Expo runs in the foreground (not piped) so it keeps a real TTY —
# that's what gives it the QR code and interactive keypress menu (w/a/i/etc).
( cd app && npx expo start )
