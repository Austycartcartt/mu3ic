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
// docker-compose.yml), applying migrations fresh, and creates one
// throwaway user to own the tracks the test ingests (tracks.user_id is
// NOT NULL since Phase 5). Skips instead of failing if Postgres isn't
// reachable, matching the pattern in internal/api/artwork_test.go — this
// is a personal-scale project without a CI Postgres service.
func testStore(t *testing.T) (st *store.Store, db *sql.DB, userID int64) {
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

	st = store.New(db)

	emailKey, err := NewUUID()
	if err != nil {
		t.Fatalf("generating test user email: %v", err)
	}
	user, err := st.CreateUser(ctx, emailKey+"@example.test", "not-a-real-hash")
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM users WHERE id = $1`, user.ID); err != nil {
			t.Errorf("cleaning up test user %d: %v", user.ID, err)
		}
	})

	return st, db, user.ID
}

// ingestFixture is a FileStorage-backed library plus a staging dir, the
// two things every IngestFile test needs. libDir is where FileStorage
// keeps objects, so tests can stat dir/<storage_key> directly.
type ingestFixture struct {
	storage *FileStorage
	libDir  string
	stage   string
}

func newIngestFixture(t *testing.T) ingestFixture {
	t.Helper()
	libDir := t.TempDir()
	storage, err := NewFileStorage(libDir)
	if err != nil {
		t.Fatalf("creating file storage: %v", err)
	}
	return ingestFixture{storage: storage, libDir: libDir, stage: t.TempDir()}
}

// stageFixture copies a testdata fixture into the staging dir and returns
// its path. IngestFile copies from (does not consume) this file, so the
// original testdata fixture is untouched.
func (f ingestFixture) stageFixture(t *testing.T, fixture string) string {
	t.Helper()

	src, err := os.Open(fixture)
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer src.Close()

	dst, err := os.CreateTemp(f.stage, "ingest-test-*")
	if err != nil {
		t.Fatalf("creating staged file: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("staging fixture: %v", err)
	}
	return dst.Name()
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
	st, db, userID := testStore(t)
	f := newIngestFixture(t)
	srcPath := f.stageFixture(t, "testdata/tagged.mp3")

	track, err := IngestFile(context.Background(), st, f.storage, userID, srcPath, "tagged.mp3", "audio/mpeg", Overrides{})
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

	audioPath := filepath.Join(f.libDir, track.StorageKey)
	if _, err := os.Stat(audioPath); err != nil {
		t.Errorf("audio object missing at %s: %v", audioPath, err)
	}
	artworkPath := filepath.Join(f.libDir, track.StorageKey+track.ArtworkExt)
	if _, err := os.Stat(artworkPath); err != nil {
		t.Errorf("artwork object missing at %s: %v", artworkPath, err)
	}

	// IngestFile copies from the staged file rather than consuming it —
	// the caller (the upload handler) owns cleanup.
	if _, err := os.Stat(srcPath); err != nil {
		t.Errorf("staged file should still exist after ingest: %v", err)
	}
}

func TestIngestFile_Untagged(t *testing.T) {
	st, db, userID := testStore(t)
	f := newIngestFixture(t)
	srcPath := f.stageFixture(t, "testdata/untagged.mp3")

	track, err := IngestFile(context.Background(), st, f.storage, userID, srcPath, "untagged.mp3", "audio/mpeg", Overrides{})
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

func TestIngestFile_OverridesReplaceTagsAndFallback(t *testing.T) {
	st, db, userID := testStore(t)

	t.Run("full override replaces tag data", func(t *testing.T) {
		f := newIngestFixture(t)
		srcPath := f.stageFixture(t, "testdata/tagged.mp3")

		track, err := IngestFile(context.Background(), st, f.storage, userID, srcPath, "tagged.mp3", "audio/mpeg", Overrides{
			Title:  "Overridden Title",
			Artist: "Overridden Artist",
			Album:  "Overridden Album",
		})
		if err != nil {
			t.Fatalf("IngestFile() error = %v", err)
		}
		cleanupTrack(t, db, track.ID)

		if track.Title != "Overridden Title" {
			t.Errorf("Title = %q, want %q", track.Title, "Overridden Title")
		}
		if track.Artist != "Overridden Artist" {
			t.Errorf("Artist = %q, want %q", track.Artist, "Overridden Artist")
		}
		if track.Album != "Overridden Album" {
			t.Errorf("Album = %q, want %q", track.Album, "Overridden Album")
		}
	})

	t.Run("partial override only replaces the set field", func(t *testing.T) {
		f := newIngestFixture(t)
		srcPath := f.stageFixture(t, "testdata/tagged.mp3")

		track, err := IngestFile(context.Background(), st, f.storage, userID, srcPath, "tagged.mp3", "audio/mpeg", Overrides{
			Artist: "Overridden Artist",
		})
		if err != nil {
			t.Fatalf("IngestFile() error = %v", err)
		}
		cleanupTrack(t, db, track.ID)

		if track.Title != "Test Title" {
			t.Errorf("Title = %q, want %q (untouched)", track.Title, "Test Title")
		}
		if track.Artist != "Overridden Artist" {
			t.Errorf("Artist = %q, want %q", track.Artist, "Overridden Artist")
		}
		if track.Album != "Test Album" {
			t.Errorf("Album = %q, want %q (untouched)", track.Album, "Test Album")
		}
	})

	t.Run("blank override leaves filename fallback untouched", func(t *testing.T) {
		f := newIngestFixture(t)
		srcPath := f.stageFixture(t, "testdata/untagged.mp3")

		track, err := IngestFile(context.Background(), st, f.storage, userID, srcPath, "untagged.mp3", "audio/mpeg", Overrides{
			Artist: "   ", // whitespace-only counts as unset
		})
		if err != nil {
			t.Fatalf("IngestFile() error = %v", err)
		}
		cleanupTrack(t, db, track.ID)

		if track.Artist != "Unknown" {
			t.Errorf("Artist = %q, want %q", track.Artist, "Unknown")
		}
	})

	t.Run("no album artist override falls back to the track's own artist", func(t *testing.T) {
		f := newIngestFixture(t)
		srcPath := f.stageFixture(t, "testdata/tagged.mp3")

		track, err := IngestFile(context.Background(), st, f.storage, userID, srcPath, "tagged.mp3", "audio/mpeg", Overrides{})
		if err != nil {
			t.Fatalf("IngestFile() error = %v", err)
		}
		cleanupTrack(t, db, track.ID)

		if track.AlbumArtist != track.Artist {
			t.Errorf("AlbumArtist = %q, want %q (fallback to Artist)", track.AlbumArtist, track.Artist)
		}
	})

	t.Run("album artist override applies independently of artist", func(t *testing.T) {
		f := newIngestFixture(t)
		srcPath := f.stageFixture(t, "testdata/tagged.mp3")

		track, err := IngestFile(context.Background(), st, f.storage, userID, srcPath, "tagged.mp3", "audio/mpeg", Overrides{
			Artist:      "Track Artist",
			AlbumArtist: "Various Artists",
		})
		if err != nil {
			t.Fatalf("IngestFile() error = %v", err)
		}
		cleanupTrack(t, db, track.ID)

		if track.Artist != "Track Artist" {
			t.Errorf("Artist = %q, want %q", track.Artist, "Track Artist")
		}
		if track.AlbumArtist != "Various Artists" {
			t.Errorf("AlbumArtist = %q, want %q", track.AlbumArtist, "Various Artists")
		}
	})
}

func TestIngestFile_FlacMimeTypeResolution(t *testing.T) {
	st, db, userID := testStore(t)

	t.Run("declared mime type wins", func(t *testing.T) {
		f := newIngestFixture(t)
		srcPath := f.stageFixture(t, "testdata/tagged.flac")

		track, err := IngestFile(context.Background(), st, f.storage, userID, srcPath, "tagged.flac", "audio/flac", Overrides{})
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
		f := newIngestFixture(t)
		srcPath := f.stageFixture(t, "testdata/tagged.flac")

		track, err := IngestFile(context.Background(), st, f.storage, userID, srcPath, "tagged.flac", "", Overrides{})
		if err != nil {
			t.Fatalf("IngestFile() error = %v", err)
		}
		cleanupTrack(t, db, track.ID)

		if track.MimeType != "audio/flac" {
			t.Errorf("MimeType = %q, want %q", track.MimeType, "audio/flac")
		}
	})
}

func TestIngestFile_InsertFailureCleansUpObjects(t *testing.T) {
	st, _, userID := testStore(t)
	f := newIngestFixture(t)
	srcPath := f.stageFixture(t, "testdata/untagged.mp3")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // force InsertTrack to fail

	_, err := IngestFile(ctx, st, f.storage, userID, srcPath, "untagged.mp3", "audio/mpeg", Overrides{})
	if err == nil {
		t.Fatal("IngestFile() error = nil, want non-nil for a canceled context")
	}

	entries, err := os.ReadDir(f.libDir)
	if err != nil {
		t.Fatalf("reading library dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("objects left in library dir after failed insert: %v", entries)
	}
}
