package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

func TestHandleListArtists(t *testing.T) {
	s, db, _, user := testServer(t)

	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Artist A", Album: "Album 1"})
	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Artist A", Album: "Album 2"})
	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Artist B", Album: "Album 3"})

	req := httptest.NewRequest(http.MethodGet, "/api/artists", nil)
	rec := httptest.NewRecorder()
	s.handleListArtists(rec, reqAs(user.ID, req))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var artists []store.ArtistSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &artists); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	counts := map[string]int{}
	for _, a := range artists {
		counts[a.Artist] = a.TrackCount
	}
	if counts["Artist A"] != 2 {
		t.Errorf("Artist A track_count = %d, want 2", counts["Artist A"])
	}
	if counts["Artist B"] != 1 {
		t.Errorf("Artist B track_count = %d, want 1", counts["Artist B"])
	}
}

func TestHandleArtistTracks(t *testing.T) {
	s, db, _, user := testServer(t)

	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Wanted Artist", Title: "Song 1"})
	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Wanted Artist", Title: "Song 2"})
	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Other Artist", Title: "Song 3"})

	req := httptest.NewRequest(http.MethodGet, "/api/artists/x/tracks", nil)
	req.SetPathValue("name", "Wanted Artist")
	rec := httptest.NewRecorder()
	s.handleArtistTracks(rec, reqAs(user.ID, req))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var tracks []store.Track
	if err := json.Unmarshal(rec.Body.Bytes(), &tracks); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("len(tracks) = %d, want 2", len(tracks))
	}
	for _, tr := range tracks {
		if tr.Artist != "Wanted Artist" {
			t.Errorf("got track for artist %q, want only Wanted Artist", tr.Artist)
		}
	}
}

func TestHandleArtistTracks_UnknownArtist(t *testing.T) {
	s, _, _, user := testServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/artists/x/tracks", nil)
	req.SetPathValue("name", "Nobody Ever Uploaded This Artist")
	rec := httptest.NewRecorder()
	s.handleArtistTracks(rec, reqAs(user.ID, req))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "[]\n" {
		t.Errorf("body = %q, want empty JSON array", rec.Body.String())
	}
}
