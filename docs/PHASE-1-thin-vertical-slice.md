# Phase 1: Thin Vertical Slice

**Status:** Complete (2026-07-14)

## Goal

Prove Go ↔ Expo connectivity end-to-end: file upload to disk, a track list endpoint backed by Postgres, and audio streaming via HTTP range requests.

## Scope

- `GET /api/health` — proves Go ↔ Expo connectivity
- `POST /api/tracks` — multipart upload, writes to `server/data/`, inserts a DB row
- `GET /api/tracks` — track list as JSON
- `GET /api/tracks/{id}/stream` — serves audio via `http.ServeContent` with range-request support
- Client: single screen listing tracks, tap to play via `expo-audio`

See [DECISIONS.md](DECISIONS.md) for the transport choice (HTTP range requests over WebRTC/HLS) and the decision to defer auth until after this slice.

Full scaffold spec: [`../PROJECT.md`](../PROJECT.md).
