# Phase 2: Upload & Metadata (Reworked Uploader)

Spec for Claude Code. Read fully before writing any code.

## Context

This is a self-hosted music streaming app. Phase 1 (thin vertical slice) is complete:

- **Backend:** Go (latest stable), stdlib `net/http` with `http.ServeMux` method/path patterns (Go 1.22+ style), PostgreSQL via `database/sql` + pgx driver, audio files on local disk, streaming served with `http.ServeContent` (automatic HTTP range / 206 support).
- **Frontend:** Expo SDK 54 + React Native + TypeScript, single codebase for web/iOS/Android, Expo Router, `expo-audio` for playback, plain `fetch` for API calls.
- **Dev environment:** Linux Mint, Docker Compose for Postgres, server binds `0.0.0.0`, physical iOS device tests via LAN IP in Expo Go.
- **Monorepo:** backend and frontend in clearly separated directories (follow the existing layout).

## Goal

Replace the current upload path with a reworked uploader that:

1. Accepts album-scale uploads (10–15 tracks) from the app via the document picker.
2. Extracts metadata (title, artist, album, duration) **at upload time** inside the upload handler.
3. Stores audio files under **UUID storage keys** instead of original filenames.

This phase absorbs what was previously a separate "Metadata Extraction" phase.

## Architectural decisions (already made — do not revisit)

These are logged in the project's Architecture & Decisions database. Implement them as specified.

### UUID storage keys

- The server generates a UUID (v4) per uploaded track and uses it as the storage key: a **bare UUID, no file extension** (e.g. `550e8400-e29b-41d4-a716-446655440000`).
- `mime_type` and `original_filename` are stored as DB columns. The streaming endpoint sets `Content-Type` from the DB, never from the filename.
- Rationale: avoids collisions, weird characters, and path traversal; makes a future move to object storage (MinIO/S3) a drop-in change where the UUID becomes the object key.

### Metadata extraction at upload time

- The upload handler is the single choke point where every track passes exactly once.
- Flow: receive multipart → write to a **temp file** → extract tags → **atomically rename** into the library → insert DB row.
- `github.com/dhowden/tag` requires an `io.ReadSeeker`, which is why the temp file step exists — do not try to extract tags from the streaming multipart body.
- The temp directory MUST be on the same filesystem/volume as the library directory so `os.Rename` stays atomic. Create a `tmp/` subdirectory inside the library root (e.g. `library/tmp/`) rather than using the OS default temp dir.

### Shared ingest function (contract with a future bulk-import phase)

The core ingest logic MUST be a standalone, handler-agnostic function that takes a file path:

```go
// IngestFile takes a file already on local disk, extracts its metadata,
// moves it into the library under a new UUID storage key, and inserts a
// track row. It is the single entry point for adding music to the library.
// The HTTP upload handler is one caller; a future watch-folder scanner
// will be another.
func IngestFile(ctx context.Context, db *sql.DB, cfg Config, srcPath string, originalFilename string) (Track, error)
```

(Exact signature can be adapted to the existing codebase conventions, but the shape — path in, track out, no `http.Request` anywhere in it — is required.)

## Backend work

### 1. Schema migration

Extend the existing `tracks` table (adapt names to the existing schema; add a migration in whatever form phase 1 established, or plain SQL if none exists):

```sql
ALTER TABLE tracks
  ADD COLUMN storage_key      UUID        NOT NULL,          -- bare UUID, also the on-disk filename
  ADD COLUMN mime_type        TEXT        NOT NULL,
  ADD COLUMN original_filename TEXT       NOT NULL,
  ADD COLUMN title            TEXT        NOT NULL,
  ADD COLUMN artist           TEXT        NOT NULL DEFAULT 'Unknown',
  ADD COLUMN album            TEXT        NOT NULL DEFAULT 'Unknown',
  ADD COLUMN duration_seconds INTEGER,                        -- nullable; not all formats expose it cheaply
  ADD COLUMN uploaded_at      TIMESTAMPTZ NOT NULL DEFAULT now();
```

If phase 1 columns overlap (e.g. an existing filename or title column), migrate/rename rather than duplicating. If the existing table stores tracks under original filenames, write a one-off migration note in the README — do NOT build an automated re-keying migration; the dev library can be re-seeded.

### 2. Upload endpoint

`POST /api/tracks` — multipart form, field name `audio`, one file per request (the client loops for albums).

Handler responsibilities, in order:

1. `http.MaxBytesReader` cap at **200 MB** (covers long FLAC files). Return `413` with a JSON error when exceeded.
2. Validate the part's declared content type is audio (`audio/*`); also sniff the first 512 bytes with `http.DetectContentType` as a fallback since browsers/pickers sometimes send `application/octet-stream`. Reject clearly-non-audio uploads with `400`.
3. Stream the multipart part to a temp file in `library/tmp/` (`os.CreateTemp`). Never buffer whole files in memory.
4. Call `IngestFile` with the temp path and the client-supplied original filename.
5. On success return `201` with the track JSON. On any failure, remove the temp file and return a JSON error.

### 3. IngestFile

1. Open the temp file, run `tag.ReadFrom` (`github.com/dhowden/tag`). Missing/unreadable tags are **not an error** — untagged files are a normal case.
2. Fallback chain:
   - `title`: tag title → else original filename with extension stripped.
   - `artist`, `album`: tag value → else `"Unknown"`.
3. Duration: `dhowden/tag` does not decode duration. Leave `duration_seconds` NULL for now and add a `// TODO(phase-later)` comment. Do not pull in an audio-decoding dependency for this — it is explicitly out of scope.
4. Generate the UUID storage key, `os.Rename` the temp file to `library/<uuid>`.
5. Insert the DB row. If the insert fails, best-effort remove the renamed file so the library and DB stay consistent, then return the error.
6. Return the full track record.

### 4. Streaming endpoint update

The existing stream endpoint must now:

- Look up the track by ID, open `library/<storage_key>`.
- Set `Content-Type` from the `mime_type` column before calling `http.ServeContent`.

### 5. Track list endpoint update

`GET /api/tracks` responses should now include `title`, `artist`, `album`, `duration_seconds`, and `original_filename`.

## Frontend work (Expo)

### Upload screen

- Use `expo-document-picker` with `type: 'audio/*'` and `multiple: true`. Install with `npx expo install expo-document-picker`.
- Upload the picked files **sequentially** (one `fetch` + `FormData` at a time), not in parallel. Show per-file progress as a simple counter: `Uploading 3 of 12 — <filename>`.
- Keep the screen awake during the upload loop with `expo-keep-awake` (`npx expo install expo-keep-awake`), activated when uploads start and released when the loop finishes or fails.
- On a per-file failure: record it, continue with the remaining files, and show a summary at the end (`10 uploaded, 2 failed`) with the failed filenames. No retry logic in this phase.
- After the loop, refresh the track list.

### Track list

Display `title` and `artist` (falling back gracefully if artist is `Unknown`) instead of raw filenames.

## Code style (non-negotiable)

- Explicit, readable code with explanatory comments — this is a learning project. No clever abstractions.
- Stdlib-first. **The only new backend dependency permitted in this phase is `github.com/dhowden/tag`.** Justify anything else by stopping and asking.
- Frontend: plain `fetch`, no upload libraries, no state-management additions.

## Explicitly out of scope — do not build any of this

- Bulk import / watch-folder scanning (future phase; `IngestFile` is the only contract with it)
- Background upload sessions, retry/resume, upload queues
- Duplicate detection
- Album art extraction/serving
- Audio duration decoding
- Auth, playlists, search, transcoding
- Any object-storage (MinIO/S3) wiring — UUID keys make that a later drop-in

## Definition of done

1. Picking 10+ audio files (mixed tagged/untagged, at least one FLAC and one MP3) on a physical iOS device uploads them all with a visible progress counter.
2. Files land in `library/` under bare-UUID names; `library/tmp/` is empty afterward.
3. Track list shows tag-derived titles/artists; untagged files show filename-derived titles.
4. Streaming any uploaded track still works, including seeking (range requests), with the correct `Content-Type`.
5. An upload interrupted mid-transfer leaves no partial file in `library/` (temp file only, cleaned up or ignorable).
6. `IngestFile` contains no HTTP types and is callable from a test with just a path.
