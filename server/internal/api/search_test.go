package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

func search(t *testing.T, s *Server, userID int64, q string) []store.Track {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/search?q="+q, nil)
	rec := httptest.NewRecorder()
	s.handleSearch(rec, reqAs(userID, req))
	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d, want 200 (body %s)", rec.Code, rec.Body)
	}
	var tracks []store.Track
	if err := json.Unmarshal(rec.Body.Bytes(), &tracks); err != nil {
		t.Fatalf("decoding search results: %v", err)
	}
	return tracks
}

func TestSearchMatchesTitleArtistAlbum(t *testing.T) {
	s, db, _, user := testServer(t)
	byTitle := insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "Midnight City", Artist: "M83", Album: "Hurry Up"})
	byArtist := insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "Outro", Artist: "Midnight Juggernauts", Album: "Dystopia"})
	byAlbum := insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "Nothing", Artist: "Nobody", Album: "The Midnight Sessions"})
	insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "Noon", Artist: "Daylight", Album: "Bright"})

	// Case-insensitive, substring, across all three fields.
	got := map[int64]bool{}
	for _, tr := range search(t, s, user.ID, "midNIGHT") {
		got[tr.ID] = true
	}
	for _, want := range []store.Track{byTitle, byArtist, byAlbum} {
		if !got[want.ID] {
			t.Errorf("track %d (%q) missing from results", want.ID, want.Title)
		}
	}
	if len(got) != 3 {
		t.Errorf("got %d results, want exactly 3", len(got))
	}
}

func TestSearchIsScopedToUser(t *testing.T) {
	s, db, _, user := testServer(t)
	other := createTestUser(t, s, db, uniqueEmail(t))
	insertTestTrack(t, s, db, other.ID, store.NewTrack{Title: "Secret Song", Artist: "Them"})

	if got := search(t, s, user.ID, "Secret"); len(got) != 0 {
		t.Fatalf("got %d results for another user's track, want 0", len(got))
	}
}

func TestSearchBlankQueryReturnsEmpty(t *testing.T) {
	s, db, _, user := testServer(t)
	insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "Anything"})

	if got := search(t, s, user.ID, ""); len(got) != 0 {
		t.Fatalf("blank query returned %d tracks, want 0", len(got))
	}
	if got := search(t, s, user.ID, "%20%20"); len(got) != 0 {
		t.Fatalf("whitespace-only query returned %d tracks, want 0", len(got))
	}
}
