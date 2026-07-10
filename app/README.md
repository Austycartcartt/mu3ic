# mu3ic app

Expo (React Native) client. Single screen: lists tracks from the Go backend, tap one to play via `expo-audio`.

## Prerequisites

- Node.js (LTS) and the Go backend running (see `../server/README.md`)
- A physical device on the same Wi-Fi/LAN as your dev machine (or a simulator/emulator)

## Setup

```bash
cd app
npm install          # already run by the scaffold, but harmless to repeat
cp .env.example .env
```

Edit `.env` and set `EXPO_PUBLIC_API_URL` to your dev machine's LAN IP (not `localhost` — physical devices can't reach that):

```bash
# find your LAN IP
ip addr show | grep 'inet ' | grep -v 127.0.0.1
```

```
EXPO_PUBLIC_API_URL=http://192.168.1.23:8080
```

## Run

```bash
npx expo start
```

Scan the QR code with Expo Go on a physical device on the same LAN, or press `a`/`i` for an emulator/simulator. Upload a track to the backend first (see `../server/README.md`) — the list is empty until you do.

There's no in-app upload UI in this version; uploading is done via `curl` against the server directly.
