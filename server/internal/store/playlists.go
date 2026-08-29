package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrReorderMismatch is returned by ReorderPlaylist when the given track id
// list isn't exactly the playlist's current membership (a track missing,
// an extra id, or a duplicate). The handler maps it to 400 — the client
// sent a stale or malformed ordering.
var ErrReorderMismatch = errors.New("track id list does not match playlist membership")

// Playlist mirrors the playlists table plus a derived TrackCount. Like the
// artist/album summaries, it's a per-user view: every method here takes a
// userID and scopes on it, so another user's playlist id is a 404,
// indistinguishable from one that doesn't exist.
type Playlist struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	TrackCount int       `json:"track_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (s *Store) ListPlaylists(ctx context.Context, userID int64) ([]Playlist, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT p.id, p.name, COUNT(pt.track_id) AS track_count, p.created_at, p.updated_at
		 FROM playlists p
		 LEFT JOIN playlist_tracks pt ON pt.playlist_id = p.id
		 WHERE p.user_id = $1
		 GROUP BY p.id
		 ORDER BY p.name`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing playlists: %w", err)
	}
	defer rows.Close()

	playlists := []Playlist{} // not nil, so an empty result serializes as [] rather than null
	for rows.Next() {
		var p Playlist
		if err := rows.Scan(&p.ID, &p.Name, &p.TrackCount, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning playlist: %w", err)
		}
		playlists = append(playlists, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating playlists: %w", err)
	}
	return playlists, nil
}

// CreatePlaylist inserts an empty playlist owned by userID. TrackCount is
// always 0 for a fresh row, so it's set here rather than re-queried.
func (s *Store) CreatePlaylist(ctx context.Context, userID int64, name string) (Playlist, error) {
	var p Playlist
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO playlists (user_id, name) VALUES ($1, $2)
		 RETURNING id, name, created_at, updated_at`,
		userID, name,
	).Scan(&p.ID, &p.Name, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return Playlist{}, fmt.Errorf("creating playlist: %w", err)
	}
	return p, nil
}

// GetPlaylist scopes the lookup to userID, so another user's playlist id
// comes back as sql.ErrNoRows — the same as a missing one, which the
// handlers map to 404.
func (s *Store) GetPlaylist(ctx context.Context, id, userID int64) (Playlist, error) {
	var p Playlist
	err := s.db.QueryRowContext(ctx,
		`SELECT p.id, p.name, COUNT(pt.track_id) AS track_count, p.created_at, p.updated_at
		 FROM playlists p
		 LEFT JOIN playlist_tracks pt ON pt.playlist_id = p.id
		 WHERE p.id = $1 AND p.user_id = $2
		 GROUP BY p.id`, id, userID,
	).Scan(&p.ID, &p.Name, &p.TrackCount, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return Playlist{}, fmt.Errorf("getting playlist %d: %w", id, err)
	}
	return p, nil
}

// RenamePlaylist updates the name (and updated_at) of a playlist the user
// owns. A no-op UPDATE (wrong id or not owned) is reported as
// sql.ErrNoRows so the handler can 404, matching GetPlaylist.
func (s *Store) RenamePlaylist(ctx context.Context, id, userID int64, name string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE playlists SET name = $1, updated_at = now() WHERE id = $2 AND user_id = $3`,
		name, id, userID)
	if err != nil {
		return fmt.Errorf("renaming playlist %d: %w", id, err)
	}
	return errNoRowsIfUnaffected(res)
}

func (s *Store) DeletePlaylist(ctx context.Context, id, userID int64) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM playlists WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("deleting playlist %d: %w", id, err)
	}
	return errNoRowsIfUnaffected(res)
}

// ListPlaylistTracks returns the playlist's tracks in position order. The
// JOIN on playlists enforces ownership: a missing or non-owned playlist
// yields no rows, which is ambiguous with a genuinely empty playlist — the
// handler calls GetPlaylist first to tell those apart (404 vs. []).
func (s *Store) ListPlaylistTracks(ctx context.Context, playlistID, userID int64) ([]Track, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+prefixedTrackColumns("t")+`
		 FROM playlist_tracks pt
		 JOIN playlists p ON p.id = pt.playlist_id
		 JOIN tracks t    ON t.id = pt.track_id
		 WHERE pt.playlist_id = $1 AND p.user_id = $2
		 ORDER BY pt.position`, playlistID, userID)
	if err != nil {
		return nil, fmt.Errorf("listing playlist %d tracks: %w", playlistID, err)
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

// AddTrackToPlaylist appends a track to the end of the playlist. Both the
// playlist and the track must belong to userID, or it's sql.ErrNoRows
// (404) — you can't add someone else's track, and probing with a
// non-owned playlist id looks identical to the playlist not existing. The
// ON CONFLICT makes a repeat add a no-op rather than an error, so the
// client can fire it without first checking membership.
func (s *Store) AddTrackToPlaylist(ctx context.Context, playlistID, userID, trackID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning add-track tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	if err := ownsPlaylist(ctx, tx, playlistID, userID); err != nil {
		return err
	}

	var exists bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM tracks WHERE id = $1 AND user_id = $2)`, trackID, userID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("checking track %d ownership: %w", trackID, err)
	}
	if !exists {
		return sql.ErrNoRows
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO playlist_tracks (playlist_id, track_id, position)
		 VALUES ($1, $2, COALESCE((SELECT MAX(position) + 1 FROM playlist_tracks WHERE playlist_id = $1), 0))
		 ON CONFLICT (playlist_id, track_id) DO NOTHING`,
		playlistID, trackID); err != nil {
		return fmt.Errorf("adding track %d to playlist %d: %w", trackID, playlistID, err)
	}
	if err := touchPlaylist(ctx, tx, playlistID); err != nil {
		return err
	}
	return tx.Commit()
}

// RemoveTrackFromPlaylist deletes one membership row. Removing a track
// that isn't in the playlist is a no-op (not an error) — same rationale as
// the idempotent add. Leftover position gaps are fine.
func (s *Store) RemoveTrackFromPlaylist(ctx context.Context, playlistID, userID, trackID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning remove-track tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	if err := ownsPlaylist(ctx, tx, playlistID, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM playlist_tracks WHERE playlist_id = $1 AND track_id = $2`,
		playlistID, trackID); err != nil {
		return fmt.Errorf("removing track %d from playlist %d: %w", trackID, playlistID, err)
	}
	if err := touchPlaylist(ctx, tx, playlistID); err != nil {
		return err
	}
	return tx.Commit()
}

// ReorderPlaylist rewrites every position from the given order. trackIDs
// must be exactly the current membership — same size, same ids, no
// duplicates — otherwise ErrReorderMismatch (the client's ordering is
// stale). Done in one transaction so a partial renumber can't be observed.
func (s *Store) ReorderPlaylist(ctx context.Context, playlistID, userID int64, trackIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning reorder tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	if err := ownsPlaylist(ctx, tx, playlistID, userID); err != nil {
		return err
	}

	rows, err := tx.QueryContext(ctx,
		`SELECT track_id FROM playlist_tracks WHERE playlist_id = $1`, playlistID)
	if err != nil {
		return fmt.Errorf("reading playlist %d membership: %w", playlistID, err)
	}
	current := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scanning membership: %w", err)
		}
		current[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating membership: %w", err)
	}

	if len(trackIDs) != len(current) {
		return ErrReorderMismatch
	}
	seen := map[int64]bool{}
	for _, id := range trackIDs {
		if seen[id] || !current[id] {
			return ErrReorderMismatch
		}
		seen[id] = true
	}

	for pos, id := range trackIDs {
		if _, err := tx.ExecContext(ctx,
			`UPDATE playlist_tracks SET position = $1 WHERE playlist_id = $2 AND track_id = $3`,
			pos, playlistID, id); err != nil {
			return fmt.Errorf("setting position for track %d: %w", id, err)
		}
	}
	if err := touchPlaylist(ctx, tx, playlistID); err != nil {
		return err
	}
	return tx.Commit()
}

// ownsPlaylist returns sql.ErrNoRows unless playlistID exists and belongs
// to userID. Used inside the mutation transactions so "not found" and "not
// yours" are one indistinguishable 404.
func ownsPlaylist(ctx context.Context, tx *sql.Tx, playlistID, userID int64) error {
	var exists bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM playlists WHERE id = $1 AND user_id = $2)`, playlistID, userID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("checking playlist %d ownership: %w", playlistID, err)
	}
	if !exists {
		return sql.ErrNoRows
	}
	return nil
}

func touchPlaylist(ctx context.Context, tx *sql.Tx, playlistID int64) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE playlists SET updated_at = now() WHERE id = $1`, playlistID); err != nil {
		return fmt.Errorf("touching playlist %d: %w", playlistID, err)
	}
	return nil
}

func errNoRowsIfUnaffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("reading rows affected: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
