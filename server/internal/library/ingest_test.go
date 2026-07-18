package library

import (
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

// testStore wires a *store.Store against the real dev Postgres (see
// docker-compose.yml), applying migrations fresh. Skips instead of
// failing if Postgres isn't reachable, matching the pattern in
// internal/api/artwork_test.go — this is a personal-scale project
// without a CI Postgres service.
func testStore(t *testing.T) (*store.Store, *sql.DB) {
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

	return store.New(db), db
}

// stageFixture copies a testdata fixture into cfg's temp dir, since
// IngestFile renames (consumes) its source file and the original fixture
// must survive for other tests.
func stageFixture(t *testing.T, cfg Config, fixture string) string {
	t.Helper()

	src, err := os.Open(fixture)
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer src.Close()

	dst, err := os.CreateTemp(cfg.TempDir(), "ingest-test-*")
	if err != nil {
		t.Fatalf("creating staged file: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("staging fixture: %v", err)
	}
	return dst.Name()
}

func newTestConfig(t *testing.T) Config {
	t.Helper()
	cfg := Config{LibraryDir: t.TempDir()}
	if err := os.MkdirAll(cfg.TempDir(), 0o755); err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	return cfg
}

func cleanupTrack(t *testing.T, db *sql.DB, id int64) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM tracks WHERE id = $1`, id); err != nil {
			t.Errorf("cleaning up test track %d: %v", id, err)
		}
	})
}

func TestIngestFile_TaggedWithArtwork(t *testing.T) {
	st, db := testStore(t)
	cfg := newTestConfig(t)
	srcPath := stageFixture(t, cfg, "testdata/tagged.mp3")

	track, err := IngestFile(context.Background(), st, cfg, srcPath, "tagged.mp3", "audio/mpeg")
	if err != nil {
		t.Fatalf("IngestFile() error = %v", err)
	}
	cleanupTrack(t, db, track.ID)

	if track.Title != "Test Title" {
		t.Errorf("Title = %q, want %q", track.Title, "Test Title")
	}
	if track.Artist != "Test Artist" {
		t.Errorf("Artist = %q, want %q", track.Artist, "Test Artist")
	}
	if track.Album != "Test Album" {
		t.Errorf("Album = %q, want %q", track.Album, "Test Album")
	}
	if track.MimeType != "audio/mpeg" {
		t.Errorf("MimeType = %q, want %q", track.MimeType, "audio/mpeg")
	}
	if !track.HasArtwork {
		t.Fatal("HasArtwork = false, want true")
	}

	audioPath := filepath.Join(cfg.LibraryDir, track.StorageKey)
	if _, err := os.Stat(audioPath); err != nil {
		t.Errorf("audio file missing at %s: %v", audioPath, err)
	}
	artworkPath := filepath.Join(cfg.LibraryDir, track.StorageKey+track.ArtworkExt)
	if _, err := os.Stat(artworkPath); err != nil {
		t.Errorf("artwork file missing at %s: %v", artworkPath, err)
	}

	if entries, err := os.ReadDir(cfg.TempDir()); err != nil {
		t.Errorf("reading temp dir: %v", err)
	} else if len(entries) != 0 {
		t.Errorf("temp dir not empty after ingest: %v", entries)
	}
}

func TestIngestFile_Untagged(t *testing.T) {
	st, db := testStore(t)
	cfg := newTestConfig(t)
	srcPath := stageFixture(t, cfg, "testdata/untagged.mp3")

	track, err := IngestFile(context.Background(), st, cfg, srcPath, "untagged.mp3", "audio/mpeg")
	if err != nil {
		t.Fatalf("IngestFile() error = %v", err)
	}
	cleanupTrack(t, db, track.ID)

	if track.Title != "untagged" {
		t.Errorf("Title = %q, want %q", track.Title, "untagged")
	}
	if track.Artist != "Unknown" {
		t.Errorf("Artist = %q, want %q", track.Artist, "Unknown")
	}
	if track.Album != "Unknown" {
		t.Errorf("Album = %q, want %q", track.Album, "Unknown")
	}
	if track.HasArtwork {
		t.Error("HasArtwork = true, want false")
	}
}

func TestIngestFile_FlacMimeTypeResolution(t *testing.T) {
	st, db := testStore(t)

	t.Run("declared mime type wins", func(t *testing.T) {
		cfg := newTestConfig(t)
		srcPath := stageFixture(t, cfg, "testdata/tagged.flac")

		track, err := IngestFile(context.Background(), st, cfg, srcPath, "tagged.flac", "audio/flac")
		if err != nil {
			t.Fatalf("IngestFile() error = %v", err)
		}
		cleanupTrack(t, db, track.ID)

		if track.MimeType != "audio/flac" {
			t.Errorf("MimeType = %q, want %q", track.MimeType, "audio/flac")
		}
	})

	t.Run("falls back to extension map with no declared type", func(t *testing.T) {
		// FLAC isn't in http.DetectContentType's sniff table, so with no
		// declared type this must fall through to the extension map
		// rather than landing on application/octet-stream.
		cfg := newTestConfig(t)
		srcPath := stageFixture(t, cfg, "testdata/tagged.flac")

		track, err := IngestFile(context.Background(), st, cfg, srcPath, "tagged.flac", "")
		if err != nil {
			t.Fatalf("IngestFile() error = %v", err)
		}
		cleanupTrack(t, db, track.ID)

		if track.MimeType != "audio/flac" {
			t.Errorf("MimeType = %q, want %q", track.MimeType, "audio/flac")
		}
	})
}

func TestIngestFile_InsertFailureCleansUpFile(t *testing.T) {
	st, _ := testStore(t)
	cfg := newTestConfig(t)
	srcPath := stageFixture(t, cfg, "testdata/untagged.mp3")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // force InsertTrack to fail

	_, err := IngestFile(ctx, st, cfg, srcPath, "untagged.mp3", "audio/mpeg")
	if err == nil {
		t.Fatal("IngestFile() error = nil, want non-nil for a canceled context")
	}

	entries, err := os.ReadDir(cfg.LibraryDir)
	if err != nil {
		t.Fatalf("reading library dir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "tmp" {
			t.Errorf("leftover file in library dir after failed insert: %s", e.Name())
		}
	}
}
