package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Austycartcartt/mu3ic/server/internal/library"
	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

// presignStorage is a library.Storage that also implements
// library.Presigner, so the stream/artwork handlers take their
// redirect path.
type presignStorage struct {
	lastKey         string
	lastContentType string
}

func (p *presignStorage) Put(context.Context, string, io.Reader, int64, string) error { return nil }
func (p *presignStorage) Open(context.Context, string) (library.Object, error)        { return nil, nil }
func (p *presignStorage) Delete(context.Context, ...string) error                     { return nil }

func (p *presignStorage) PresignGet(_ context.Context, key, contentType string, _ time.Duration) (string, error) {
	p.lastKey = key
	p.lastContentType = contentType
	return "https://r2.example/" + key + "?sig=abc", nil
}

func TestHandleStream_RedirectsWhenBackendCanPresign(t *testing.T) {
	s, db, _, user := testServer(t)
	fake := &presignStorage{}
	s.storage = fake

	track := insertTestTrack(t, s, db, user.ID, store.NewTrack{MimeType: "audio/flac"})

	req := httptest.NewRequest(http.MethodGet, "/api/tracks/x/stream", nil)
	req.SetPathValue("id", strconv.FormatInt(track.ID, 10))
	rec := httptest.NewRecorder()

	s.handleStream(rec, reqAs(user.ID, req))

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (body: %s)", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "https://r2.example/"+track.StorageKey+"?sig=abc" {
		t.Fatalf("Location = %q, want the presigned URL", loc)
	}
	if fake.lastContentType != "audio/flac" {
		t.Errorf("presigned content type = %q, want the DB mime type", fake.lastContentType)
	}
}

func TestHandleStream_NotFoundForOtherUsersTrack(t *testing.T) {
	s, db, _, user := testServer(t)
	s.storage = &presignStorage{}
	track := insertTestTrack(t, s, db, user.ID, store.NewTrack{})
	other := createTestUser(t, s, db, uniqueEmail(t))

	req := httptest.NewRequest(http.MethodGet, "/api/tracks/x/stream", nil)
	req.SetPathValue("id", strconv.FormatInt(track.ID, 10))
	rec := httptest.NewRecorder()

	s.handleStream(rec, reqAs(other.ID, req))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
