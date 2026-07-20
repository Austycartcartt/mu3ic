# Architecture & Decisions

Append-only log. Add a new entry per decision; don't edit past entries except to change `Status` (e.g. `Decided` → `Superseded`) — add a note explaining why rather than rewriting history.

---

## PostgreSQL for metadata & track index

**Date:** 2026-06-28 · **Category:** Database · **Status:** Decided

**Rationale:** Relational schema for tracks, albums, playlists, users. Simple queries and JOIN support without overengineering.

**Trade-offs:** Not a document database — but relational structure is natural for music library metadata. No denormalization needed upfront.

---

## Defer authentication & complexity

**Date:** 2026-06-28 · **Category:** Auth · **Status:** Decided

**Rationale:** Build a thin vertical slice first to validate end-to-end connectivity. Auth comes after foundational layers are solid. Known complexity: attaching auth headers to stream URLs.

**Trade-offs:** Early phases are less secure. But risk is low in a personal project, and addressing real problems (proving connectivity) first is pragmatic.

---

## HTTP range requests for audio streaming

**Date:** 2026-06-28 · **Category:** Audio Transport · **Status:** Decided

**Rationale:** HTTP range requests with `http.ServeContent` handle `206 Partial Content` automatically. Functionally equivalent to production streaming infrastructure. Avoids WebRTC (wrong paradigm — it's P2P/real-time) and HLS/DASH complexity (not needed until adaptive bitrate over variable connections becomes necessary).

**Trade-offs:** Cannot do adaptive bitrate without segmentation. But this is deferred until it's actually needed; adds complexity with minimal upside when bandwidth is local/predictable.

---

## Expo + React Native for mobile (one codebase)

**Date:** 2026-06-28 · **Category:** Frontend · **Status:** Decided

**Rationale:** Web, iOS, Android from a single Expo/React Native codebase. Uses `expo-audio` (not deprecated `expo-av`). Leverages existing React/TypeScript proficiency.

**Trade-offs:** Mobile performance depends on Expo bridging. But for audio streaming and UI responsiveness, this is acceptable. Native modules can be added later if needed.

---

## Pin Expo SDK 54 instead of latest

**Date:** 2026-07-13 · **Category:** Frontend · **Status:** Decided

**Rationale:** `app/` is pinned to Expo SDK 54 instead of latest (was SDK 57) because Expo Go on the App Store only supports SDK 54, and the dev-build alternative requires a paid Apple Developer account.

**Trade-offs:** Diverges from the "always latest stable" ground rule in `PROJECT.md`. Do not bump back to latest without confirming Expo Go / Apple Developer account status first.

---

## Store audio files under UUID keys, not original filenames

**Date:** 2026-07-16 · **Category:** Database · **Status:** Decided

**Rationale:** The server generates a UUID for each uploaded track and uses it as the storage key (bare UUID, no extension). MIME type and original filename are stored as DB columns; `Content-Type` is set from the DB when serving. This avoids filename collisions, weird characters, and path traversal in one move, and makes a future migration to cloud object storage (MinIO/S3) a drop-in change — the UUID becomes the object key unchanged, and nothing in the DB or API cares where the bytes live.

**Trade-offs:** Library folder is not human-browsable (opaque filenames); original filename must be preserved as a metadata column. Extension-less files mean local tools can't infer type from name — acceptable since all serving goes through the API, which sets `Content-Type` from the DB.

---

## Extract metadata at upload time, not as a separate phase

**Date:** 2026-07-16 · **Category:** API Design · **Status:** Decided

**Rationale:** The upload endpoint is the natural choke point where every track passes exactly once. Flow: receive multipart → buffer to temp file (`os.CreateTemp` on the same volume as the library) → extract tags with `dhowden/tag` (needs an `io.ReadSeeker`, so a temp file is required) → `os.Rename` into the library under its UUID key (atomic on the same filesystem, no half-written files) → insert row. Fallback chain for missing tags: title from tag, else original filename minus extension; artist/album default to `"Unknown"`.

**Trade-offs:** Upload handler does more work per request (tag parsing adds latency, negligible at personal scale). Requires the temp dir to be on the same filesystem as the library to keep the rename atomic. Replaces the previously planned standalone metadata phase, so the build order shifted.
