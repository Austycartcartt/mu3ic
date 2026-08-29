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

---

## Client-side filename-heuristic upload preview, explicit override wins

**Date:** 2026-08-21 · **Category:** Upload UX · **Status:** Decided

**Rationale:** Real-world filenames like `Inferno-01-001-Boards of Canada-Introit.wav` or `Aeikus - Going Deeper, Vol. 7 (Free Download) - 20 Cataracta.m4a` often carry better artist/album/title information than missing or wrong embedded tags, but the server's filename fallback (`ingest.go`) only ever used the whole filename as a title. `app/src/utils/parseFilename.ts` parses a small set of known dash-delimited patterns (plus a generic 2/3-token fallback) into an Artist/Album/Title guess, purely from the filename string — no audio bytes or server round-trip needed. The upload screen (`app/src/app/upload.tsx`) shows this guess in an editable preview before anything uploads; on confirm, the (possibly hand-edited) values are sent as optional `title`/`artist`/`album` multipart fields, and `library.Overrides` (`ingest.go`) applies any non-empty one on top of the existing tag-extraction/fallback chain — i.e. an explicit override always wins, even over a real embedded tag.

**Trade-offs:** The parser only pre-fills a field when it's confident in the match, specifically so an unrecognized filename doesn't send a bogus override that clobbers a good embedded tag — but a confident *wrong* guess (e.g. a filename that happens to fit the pattern by coincidence) will still silently override real tag data unless the user notices and edits it in the preview. The parser is a small heuristic, not a general filename grammar; it won't handle every naming convention, and the editable preview is the intended correctness backstop, not full automation.

---

## Folder upload is web-only; native keeps flat multi-select

**Date:** 2026-08-21 · **Category:** Upload UX · **Status:** Decided

**Rationale:** Bulk-picking a whole folder needs a directory-tree picker. Browsers expose one for free via the non-standard `webkitdirectory` input attribute (`app/src/app/upload.tsx`'s `pickFolderWeb`), with no new dependency. Expo has no built-in equivalent for iOS/Android — Android would need Storage Access Framework tree-picking and iOS has no comparable API without custom native code, both out of proportion to the ask. Native keeps today's `expo-document-picker` flat multi-file picker, which already lets a user select many files from a folder in one dialog, just without nested subfolders auto-included.

**Trade-offs:** Native users importing a deeply nested folder tree must flatten it themselves (drag all files into the picker, or pick subfolders one at a time) rather than pointing at the top-level folder. Revisit if this proves painful enough to justify a native module.

---

## Defer external album/artist dataset lookup (e.g. MusicBrainz)

**Date:** 2026-08-21 · **Category:** Upload UX · **Status:** Decided

**Rationale:** Comparing uploaded filenames against a downloadable/online music database could improve metadata accuracy beyond filename parsing, but it means a new network dependency, fuzzy-matching logic, and (for an offline dataset) meaningful storage — a large scope increase relative to the filename-parsing preview this phase actually needed. Shipped the heuristic parser + editable preview instead (see above).

**Trade-offs:** Metadata quality is capped by what the filename itself encodes; genuinely ambiguous or sparse filenames still need manual entry. Revisit as a later phase if manual cleanup proves too frequent.

---

## Add album_artist, distinct from each track's own artist

**Date:** 2026-08-23 · **Category:** Database / Upload UX · **Status:** Decided

**Rationale:** `ListAlbums` grouped by `(album, artist)`. For a various-artists compilation — one album, a different artist per track, e.g. from the filename-parsing preview above — that produced one album row per distinct track artist instead of one row for the whole album (reported as: uploading "Going Deeper, Vol. 7" and seeing it listed multiple times). Grouping by `album` alone was rejected: it can't tell that case apart from two genuinely different albums that happen to share an exact title by different single artists (already covered by an existing test, `TestHandleListAlbums_SameTitleDifferentArtist`) — both look identical as "one album title, multiple artists" without more information. Added an `album_artist` column (`004_album_artist.sql`), mirroring the ALBUMARTIST/TPE2 tag real files use for exactly this: `ListAlbums`/`ListTracksByAlbum` now group/filter on `(album, album_artist)`. `store.InsertTrack` defaults an unset `album_artist` to that track's own artist, so ordinary single-artist albums (and the title-collision test) keep working unchanged — only an explicit album artist (an ALBUMARTIST tag, or the new optional "Album Artist" field on the upload preview screen, applied to every file in a batch) collapses a compilation into one row.

**Trade-offs:** A various-artists compilation uploaded without ALBUMARTIST tags still needs the user to fill in the batch-level "Album Artist" field (e.g. "Various Artists") for it to group correctly — there's no way to infer it from filenames alone. Existing already-uploaded rows were backfilled to `album_artist = artist` in the same migration, so nothing already in the library needs re-uploading to keep working as before.

---

## JWT auth: email login, bearer header + `?token=` fallback, per-user libraries

**Date:** 2026-08-27 · **Category:** Auth · **Status:** Decided (Phase 5)

**Rationale:** Implemented the auth Phase 5 planned. Stateless HS256 JWT (`github.com/golang-jwt/jwt/v5`), 24h expiry, signed with `JWT_SECRET` (env var; insecure dev default with a startup warning). Passwords hashed with `golang.org/x/crypto/bcrypt`. Both are new dependencies — PROJECT.md's stdlib-first rule was waived here because the Phase 5 doc pre-approved them and hand-rolling either (especially password hashing) is the kind of security-sensitive code you don't want to own. New `internal/auth` package (no HTTP types), `internal/store/users.go`, `internal/api/{auth,middleware}.go`. `users` (`006_users.sql`) plus `tracks.user_id` (`007_tracks_user_id.sql`, `NOT NULL REFERENCES users ON DELETE CASCADE`). Every track/artist/album store method now takes a `userID` and filters on it; `GetTrack` is scoped too, so another user's track id is a 404, identical to a missing one. Routes: `/api/health` and `/api/auth/{register,login}` are public; everything else is wrapped by `withAuth`, which reads the token from `Authorization: Bearer` **or** a `?token=` query param — the query param exists because the audio player and `<Image>` fetch `/stream` and `/artwork` directly and can't set headers. `RunMigrations` gained a Postgres advisory lock so parallel `go test ./...` package binaries don't race to apply a new migration.

**Deviations from `docs/PHASE-5-authentication.md`:** (1) login is by **email**, not username — `users.email UNIQUE` with a loose format check. (2) `users.id` is **BIGSERIAL**, not UUID, to match `tracks.id` and the existing `int64` / `strconv.ParseInt` code paths; `user_id` is a plain `BIGINT` FK. (3) Registration is open (anyone can register) — fine on a LAN, revisit before the Phase 8 public deployment.

**Trade-offs:** `007` is destructive — it `DELETE`s all existing `tracks` rather than backfilling an owner (dev library is disposable; same call as `003_uuid_storage.sql`), so the library must be re-uploaded and stale files cleared from `DATA_DIR`. Token-in-URL for streaming means the token can appear in server logs / browser history; acceptable for a personal app, the Phase 5 doc notes a streaming-only short-lived token as the future hardening. On web the token is persisted in `localStorage` (SecureStore has no web backend) — an XSS-exfiltration risk the native Keychain/Keystore path avoids. Open registration + an unset `JWT_SECRET` would let anyone mint tokens; the deploy checklist must cover both.

---

## Playlists & Search: substring search, gapped-integer ordering, in-memory queue

**Date:** 2026-08-28 · **Category:** Database / API Design / Frontend · **Status:** Decided (Phase 6)

**Rationale:** Implemented Phase 6 as planned in `docs/PHASE-6-playlists-search.md`. New tables `playlists` and `playlist_tracks` (`008_playlists.sql`), both `ON DELETE CASCADE` from their parents. Unlike `003_uuid_storage.sql` / `007_tracks_user_id.sql`, `008` is **additive** — it drops nothing and needs no re-upload. `playlist_tracks` uses `PRIMARY KEY (playlist_id, track_id)`, so a track can't be added to the same playlist twice; ordering is a plain gapped `INTEGER position` that reorder rewrites wholesale (gaps from a removal are harmless — only relative order is read). New `internal/store/playlists.go` mirrors `albums.go`: every method takes `userID` and scopes on it, so another user's playlist id is `sql.ErrNoRows` → an indistinguishable 404. `ReorderPlaylist` requires the submitted id list to be *exactly* the current membership (`ErrReorderMismatch` → 400) so a stale client can't half-apply an order. Adds run `INSERT … ON CONFLICT DO NOTHING` (idempotent). Nine new routes under `s.protected`; `withCORS`'s `Access-Control-Allow-Methods` widened from `GET, POST, OPTIONS` to add `PATCH, PUT, DELETE` (the web client now issues them).

**Search** is a deliberate `title/artist/album ILIKE '%q%'` (`store.SearchTracks`, `LIMIT 100`), **not** Postgres full-text search or `pg_trgm`: no stemming or ranking, but zero new machinery (no `tsvector` column, GIN index, generated-column trigger, or extension) and more than adequate at a personal library's scale. A `strings.NewReplacer` escapes `\ % _` with an `ESCAPE '\'` clause; the handler returns `[]` for a blank query rather than dumping the library, since the client hits it on every keystroke.

**Frontend:** `usePlayer` gained a queue — `play(track, uri)` became `playQueue(tracks, startIndex)`, with `playNext`/`playPrevious` and auto-advance on `expo-audio`'s `status.didJustFinish` (ref-gated to fire once per finish). This also makes the Songs/album/artist screens play through their list in context, not just the one tapped track. Reorder UI is an "edit mode" with ▲/▼ move buttons, **not** drag-and-drop: `react-native-draggable-flatlist` would be a new dependency PROJECT.md forbids without justification, and ▲/▼ is fine for short playlists. Search is a dedicated 5th tab. `playlists.tsx` became a `playlists/` folder (`_layout` + `index` + `[id]`), matching `albums/`. `PlaylistNameModal` is a cross-platform text-prompt (RN's `Alert.prompt` is iOS-only).

**Trade-offs:** No duplicate tracks per playlist (composite PK) — revisit if ever wanted. Search quality is capped at substring matching — no typo tolerance or relevance ordering; `pg_trgm`/`tsvector` is the upgrade path if the library outgrows it. Reorder isn't drag-and-drop, so reordering a long playlist is many taps. The playback queue is in-memory only: it doesn't survive an app restart, and there's no "up next" view, shuffle, or repeat yet. `ReorderPlaylist` issues one `UPDATE` per row in a loop inside the transaction (fine at personal scale; a single `UPDATE … FROM (VALUES …)` would scale better).
