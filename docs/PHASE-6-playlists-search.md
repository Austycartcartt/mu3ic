# Phase 6: Playlists & Search

**Status:** Complete (2026-08-28)

## Goal

Organize tracks into user-owned playlists (create / rename / delete, add / remove tracks, manual
reorder, play as a queue). Add a search screen that finds tracks by title, artist, or album.

**What shipped vs. this plan:** matches the plan below. Search is a plain `ILIKE '%q%'` substring
match, not Postgres full-text search — decided for personal scale and to avoid new machinery (see the
Phase 6 entry in [DECISIONS.md](DECISIONS.md)). Reorder is an "edit mode" with ▲/▼ move buttons, not
drag-and-drop (a drag library would be a new dependency). Migration `008` is **additive** — no
library wipe, unlike `003`/`007`.

---

## Backend (Go)

### 1. Database Schema — `migrations/008_playlists.sql`

```sql
CREATE TABLE playlists (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX playlists_user_id_idx ON playlists (user_id);

CREATE TABLE playlist_tracks (
    playlist_id BIGINT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    track_id    BIGINT NOT NULL REFERENCES tracks(id)    ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    added_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (playlist_id, track_id)   -- a track appears at most once per playlist
);
CREATE INDEX playlist_tracks_playlist_pos_idx ON playlist_tracks (playlist_id, position);
```

`position` is a gapped integer (0, 1, 2, …). Reorder rewrites the affected rows; a gap left by a
removal is harmless since only the relative order is read. The composite primary key is what forbids
the same track twice in one playlist.

### 2. Store — `internal/store/playlists.go`

Same style as `store/albums.go`: raw SQL, every method takes `userID` and scopes on it, so another
user's playlist id is `sql.ErrNoRows` — a 404 indistinguishable from a missing row.

| Method | Notes |
|---|---|
| `ListPlaylists(ctx, userID)` | `LEFT JOIN playlist_tracks` for `track_count`, `ORDER BY name` |
| `CreatePlaylist(ctx, userID, name)` | `INSERT … RETURNING`; `track_count` 0 |
| `GetPlaylist(ctx, id, userID)` | scoped; `ErrNoRows` if absent/not owned |
| `RenamePlaylist` / `DeletePlaylist` | 0 rows affected → `ErrNoRows` |
| `ListPlaylistTracks(ctx, playlistID, userID)` | `JOIN playlists` enforces ownership; `ORDER BY position` |
| `AddTrackToPlaylist` | verifies playlist **and** track are the caller's; `position = COALESCE(MAX+1, 0)`; `ON CONFLICT DO NOTHING` (idempotent); bumps `updated_at` |
| `RemoveTrackFromPlaylist` | ownership-checked `DELETE`; bumps `updated_at` |
| `ReorderPlaylist(ctx, playlistID, userID, trackIDs)` | one tx; `trackIDs` must be exactly the current membership (no missing/extra/dup) or `ErrReorderMismatch` (→ 400); rewrites `position` |

Search lives in `internal/store/tracks.go`:

```go
func (s *Store) SearchTracks(ctx context.Context, userID int64, query string) ([]Track, error)
```

`WHERE user_id = $1 AND (title ILIKE $2 ESCAPE '\' OR artist ILIKE $2 … OR album ILIKE $2 …) ORDER BY
title LIMIT 100`, with `$2 = "%" + escaped + "%"`. `likeEscaper` neutralises `\ % _` so those
characters match literally. The handler skips the call for a blank query.

### 3. API — `internal/api/playlists.go`, `internal/api/search.go`

| Method & path | Handler | Response |
|---|---|---|
| `GET /api/playlists` | `handleListPlaylists` | `[]Playlist` |
| `POST /api/playlists` | `handleCreatePlaylist` | 201 `Playlist` (400 on empty name) |
| `PATCH /api/playlists/{id}` | `handleRenamePlaylist` | 204 (404 if not owned) |
| `DELETE /api/playlists/{id}` | `handleDeletePlaylist` | 204 |
| `GET /api/playlists/{id}/tracks` | `handlePlaylistTracks` | `[]Track` in order; 404 if playlist absent |
| `POST /api/playlists/{id}/tracks` | `handleAddPlaylistTrack` | 204; body `{"track_id":N}`; 404 if playlist/track not owned |
| `DELETE /api/playlists/{id}/tracks/{trackId}` | `handleRemovePlaylistTrack` | 204 |
| `PUT /api/playlists/{id}/tracks` | `handleReorderPlaylist` | 204; body `{"track_ids":[…]}`; 400 on mismatch |
| `GET /api/search?q=` | `handleSearch` | `[]Track`; blank `q` → `[]` |

All are registered under `s.protected(...)` in `router.go`. `withCORS`'s
`Access-Control-Allow-Methods` was widened to `GET, POST, PATCH, PUT, DELETE, OPTIONS` — the web
client now issues those.

### 4. Tests

`internal/api/playlists_test.go` and `search_test.go`, using the existing `testServer` /
`insertTestTrack` / `reqAs` helpers: create/list, add in order, duplicate add is a no-op, add a
foreign track → 404, reorder (and reject a stale/dup id list → 400), remove, rename, delete, foreign
access → 404; search matches all three fields case-insensitively, is scoped to the caller, and
returns `[]` for a blank query.

---

## Frontend (Expo)

### 1. API client — `src/api/client.ts`

`Playlist` type + `getPlaylists`, `createPlaylist`, `renamePlaylist`, `deletePlaylist`,
`getPlaylistTracks`, `addTrackToPlaylist`, `removeTrackFromPlaylist`, `reorderPlaylist`, and
`searchTracks`, all via the existing `request()` wrapper.

### 2. Playback queue — `src/hooks/usePlayer.tsx`

`play(track, uri)` is replaced by `playQueue(tracks, startIndex)`: callers hand over the whole list
they're showing plus the tapped index. Added `playNext` / `playPrevious` / `hasNext` /
`hasPrevious`, and auto-advance on `status.didJustFinish` (gated by a ref so it fires once per
finish). `PlayerDock` gained ⏮ / ⏭ buttons that dim at the ends. The Songs, album, and artist
screens now call `playQueue`, so playback runs through the list in context.

### 3. Search tab — `src/app/(tabs)/search.tsx`

5th tab (🔍). Controlled `TextInput`, 300 ms debounce, `searchTracks(q)` with the stale-response
`ignore` guard, results in a `TrackList` → `playQueue`. Blank query shows a "Search your library"
hint.

### 4. Playlists tab — `src/app/(tabs)/playlists/`

`playlists.tsx` became a folder: `_layout.tsx` (nested `Stack`, like `albums/`), `index.tsx` (list +
"New" → `PlaylistNameModal`), `[id].tsx` (detail). The detail screen has an **Edit** toggle: in edit
mode each row shows ▲ / ▼ / ✕ (`TrackList`'s new `editControls` prop) and a Rename / Delete bar
appears. Reorder and remove are optimistic, reloading from the server on error.

### 5. Add to playlist

`TrackList` gained an optional `onLongPress(track)`. Long-pressing a row on the Songs / album /
artist / search screens opens `AddToPlaylistSheet` — a bottom sheet listing the user's playlists plus
"New playlist…". `PlaylistNameModal` is a cross-platform text-prompt dialog (`Alert.prompt` is
iOS-only), reused for create and rename.

---

## Testing

**Backend**

```bash
cd server
docker compose -f ../docker-compose.yml up -d
go test ./...
go vet ./...
```

Manual curl (`TOKEN` from `POST /api/auth/login`):

```bash
PID=$(curl -s -X POST localhost:8080/api/playlists -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Roadtrip"}' | jq .id)
curl -s -X POST localhost:8080/api/playlists/$PID/tracks -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"track_id":1}'
curl -s localhost:8080/api/playlists/$PID/tracks -H "Authorization: Bearer $TOKEN"
curl -s -X PUT localhost:8080/api/playlists/$PID/tracks -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"track_ids":[2,1]}'
curl -s "localhost:8080/api/search?q=love" -H "Authorization: Bearer $TOKEN"
```

**Frontend**

```bash
cd app
npx tsc --noEmit
npm run lint
npx expo export --platform web
```

Manual: create a playlist → long-press a song → "Add to playlist" → open the playlist → play (confirm
auto-advance and ⏭/⏮) → edit mode → move a track, remove one → rename, delete. Search tab: type a
partial title/artist/album and confirm matches play.

---

## Implementation Checklist

- [x] `008_playlists.sql` (additive — no wipe)
- [x] `store/playlists.go` — CRUD, add/remove, reorder with membership check
- [x] `store.SearchTracks` + `likeEscaper` in `store/tracks.go`
- [x] `api/playlists.go` (8 handlers) + `api/search.go`
- [x] Routes registered; `withCORS` methods widened to `PATCH/PUT/DELETE`
- [x] `playlists_test.go` + `search_test.go` (all green)
- [x] `client.ts` — playlist + search functions, `Playlist` type
- [x] `usePlayer` queue (`playQueue`, next/prev, auto-advance); `PlayerDock` skip buttons
- [x] Songs / album / artist screens switched to `playQueue`
- [x] Search tab + `_layout.tsx` registration
- [x] `playlists/` folder — `_layout`, `index`, `[id]` with edit mode
- [x] `AddToPlaylistSheet` + `PlaylistNameModal`; `TrackList` `onLongPress` / `editControls`
- [x] typecheck, lint, web export all pass

---

## Known Limitations / Next

1. **No duplicate tracks in a playlist** — the composite PK forbids it; revisit if "add the same song
   twice" is ever wanted.
2. **Reorder is ▲/▼, not drag** — deliberate (no new dependency). Fine for short playlists; a drag
   library can come later if playlists get long.
3. **Search is substring, not full-text** — no stemming/ranking. `pg_trgm` or `tsvector` is the
   upgrade path if the library grows enough to need it.
4. **Queue is in-memory** — it doesn't persist across app restarts, and there's no "up next" view or
   shuffle/repeat yet.
