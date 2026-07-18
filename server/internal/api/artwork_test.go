package api

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/Austycartcartt/mu3ic/server/internal/library"
	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

// testServer wires a Server against the real dev Postgres (see
// docker-compose.yml) and a temp-dir file store, applying migrations fresh.
// The artwork handler's behavior depends on both a real DB row (for
// HasArtwork) and a real file on disk (for ArtworkPath), so a lightweight
// fake would end up re-implementing both — not worth it for one handler.
// Skips instead of failing if Postgres isn't reachable, since this is a
// personal-scale project without a CI Postgres service.
func testServer(t *testing.T) (*Server, *sql.DB, string) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://mu3ic:mu3ic@localhost:5432/mu3ic?sslmode=disable"
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Skipf("skipping: opening database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skipping: Postgres not reachable at %s: %v", databaseURL, err)
	}

	if err := store.RunMigrations(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	libraryDir := t.TempDir()
	storage, err := library.NewFileStorage(libraryDir)
	if err != nil {
		t.Fatalf("initializing storage: %v", err)
	}
	cfg := library.Config{LibraryDir: libraryDir}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return NewServer(store.New(db), storage, cfg, logger), db, libraryDir
}

// insertTestTrack inserts a track (generating a fresh UUID storage key)
// and registers a cleanup to delete that single row afterward, so this
// test never leaves rows behind in the shared dev DB.
func insertTestTrack(t *testing.T, s *Server, db *sql.DB, nt store.NewTrack) store.Track {
	t.Helper()

	storageKey, err := library.NewUUID()
	if err != nil {
		t.Fatalf("generating storage key: %v", err)
	}
	nt.StorageKey = storageKey
	if nt.OriginalFilename == "" {
		nt.OriginalFilename = "test.mp3"
	}
	if nt.MimeType == "" {
		nt.MimeType = "audio/mpeg"
	}
	if nt.Title == "" {
		nt.Title = "Test Title"
	}

	track, err := s.store.InsertTrack(context.Background(), nt)
	if err != nil {
		t.Fatalf("inserting track: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM tracks WHERE id = $1`, track.ID); err != nil {
			t.Errorf("cleaning up test track %d: %v", track.ID, err)
		}
	})
	return track
}

func TestHandleArtwork(t *testing.T) {
	s, db, libraryDir := testServer(t)

	t.Run("404 when track has no artwork", func(t *testing.T) {
		track := insertTestTrack(t, s, db, store.NewTrack{})

		req := httptest.NewRequest(http.MethodGet, "/api/tracks/x/artwork", nil)
		req.SetPathValue("id", strconv.FormatInt(track.ID, 10))
		rec := httptest.NewRecorder()

		s.handleArtwork(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("200 with artwork bytes when present", func(t *testing.T) {
		track := insertTestTrack(t, s, db, store.NewTrack{ArtworkExt: ".png"})

		// Writing the artwork's storage_key.png sibling file directly here
		// mirrors what IngestFile does on ingest — Storage itself no
		// longer has a "save" side, so there's nothing to call.
		artwork := []byte{0x89, 0x50, 0x4e, 0x47} // PNG signature bytes, contents don't matter here
		artworkPath := filepath.Join(libraryDir, track.StorageKey+".png")
		if err := os.WriteFile(artworkPath, artwork, 0o644); err != nil {
			t.Fatalf("writing artwork fixture: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/tracks/x/artwork", nil)
		req.SetPathValue("id", strconv.FormatInt(track.ID, 10))
		rec := httptest.NewRecorder()

		s.handleArtwork(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if got := rec.Body.Bytes(); string(got) != string(artwork) {
			t.Errorf("body = %v, want %v", got, artwork)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
			t.Errorf("Content-Type = %q, want %q", ct, "image/png")
		}
	})
}
