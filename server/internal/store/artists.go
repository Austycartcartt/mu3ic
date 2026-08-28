package store

import (
	"context"
	"fmt"
)

// ArtistSummary is a grouped view over tracks.artist. Artists aren't a
// normalized entity (no artists table) — this is just an aggregate query
// over the denormalized column.
type ArtistSummary struct {
	Artist     string `json:"artist"`
	TrackCount int    `json:"track_count"`
}

func (s *Store) ListArtists(ctx context.Context) ([]ArtistSummary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT artist, COUNT(*) AS track_count FROM tracks GROUP BY artist ORDER BY artist`)
	if err != nil {
		return nil, fmt.Errorf("listing artists: %w", err)
	}
	defer rows.Close()

	artists := []ArtistSummary{} // not nil, so an empty result serializes as [] rather than null
	for rows.Next() {
		var a ArtistSummary
		if err := rows.Scan(&a.Artist, &a.TrackCount); err != nil {
			return nil, fmt.Errorf("scanning artist: %w", err)
		}
		artists = append(artists, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating artists: %w", err)
	}
	return artists, nil
}

// ListTracksByArtist orders by album, then track_number (untagged tracks
// sort after any tagged track within the album) then title as a fallback.
func (s *Store) ListTracksByArtist(ctx context.Context, artist string) ([]Track, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+trackColumns+` FROM tracks WHERE artist = $1 ORDER BY album, track_number NULLS LAST, title`, artist)
	if err != nil {
		return nil, fmt.Errorf("listing tracks for artist %q: %w", artist, err)
	}
	defer rows.Close()

	tracks := []Track{}
	for rows.Next() {
		track, err := scanTrack(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning track: %w", err)
		}
		tracks = append(tracks, track)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tracks: %w", err)
	}
	return tracks, nil
}
