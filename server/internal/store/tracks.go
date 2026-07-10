package store

import (
	"context"
	"fmt"
	"time"
)

// Track mirrors the tracks table. JSON tags define the wire format returned
// to clients.
type Track struct {
	ID        int64     `json:"id"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// InsertTrack records a newly uploaded file's metadata. The id is assigned
// by Postgres (BIGSERIAL), so there's no need to generate one in Go.
func (s *Store) InsertTrack(ctx context.Context, filename string, size int64) (Track, error) {
	var t Track
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO tracks (filename, size) VALUES ($1, $2)
		 RETURNING id, filename, size, created_at`,
		filename, size,
	).Scan(&t.ID, &t.Filename, &t.Size, &t.CreatedAt)
	if err != nil {
		return Track{}, fmt.Errorf("inserting track: %w", err)
	}
	return t, nil
}

func (s *Store) ListTracks(ctx context.Context) ([]Track, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, filename, size, created_at FROM tracks ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("listing tracks: %w", err)
	}
	defer rows.Close()

	tracks := []Track{} // not nil, so an empty result serializes as [] rather than null
	for rows.Next() {
		var t Track
		if err := rows.Scan(&t.ID, &t.Filename, &t.Size, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning track: %w", err)
		}
		tracks = append(tracks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tracks: %w", err)
	}
	return tracks, nil
}

func (s *Store) GetTrack(ctx context.Context, id int64) (Track, error) {
	var t Track
	err := s.db.QueryRowContext(ctx,
		`SELECT id, filename, size, created_at FROM tracks WHERE id = $1`, id,
	).Scan(&t.ID, &t.Filename, &t.Size, &t.CreatedAt)
	if err != nil {
		return Track{}, fmt.Errorf("getting track %d: %w", id, err)
	}
	return t, nil
}
