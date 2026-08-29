// Package store provides PostgreSQL access via database/sql (no ORM).
package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Store wraps a database connection pool. Its methods hold the raw SQL
// queries for the app's domain types (see tracks.go).
type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// migrationsLockID is an arbitrary constant identifying the Postgres
// session-level advisory lock RunMigrations holds while it works. Any
// value works as long as nothing else in this app uses the same one.
const migrationsLockID = 4917283016

// RunMigrations applies any *.sql files in dir that haven't been applied yet,
// in filename order, tracking progress in a schema_migrations table it
// creates itself. There's no migration library in the approved dependency
// list, so this is a small hand-rolled runner instead of golang-migrate/goose.
func RunMigrations(ctx context.Context, db *sql.DB, dir string) error {
	// Serialize concurrent callers against one database. This matters for
	// `go test ./...`, which runs each package's test binary in parallel:
	// without the lock, two of them can both see migration NNN as
	// unapplied and both try to run its CREATE TABLE, and one fails. The
	// lock is taken on a dedicated connection because advisory locks are
	// session-scoped and *sql.DB is a pool; closing the connection
	// releases the lock even if the explicit unlock is skipped.
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquiring migration connection: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, migrationsLockID); err != nil {
		return fmt.Errorf("acquiring migration lock: %w", err)
	}
	defer func() {
		// context.WithoutCancel: still release the lock even when ctx is
		// already done (e.g. a canceled startup).
		if _, err := conn.ExecContext(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, migrationsLockID); err != nil {
			slog.Warn("releasing migration lock", "error", err)
		}
	}()

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading migrations dir %q: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)

	for _, name := range files {
		var applied bool
		err := db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, name,
		).Scan(&applied)
		if err != nil {
			return fmt.Errorf("checking migration %q: %w", name, err)
		}
		if applied {
			continue
		}

		sqlBytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("reading migration %q: %w", name, err)
		}

		if err := applyMigration(ctx, db, name, string(sqlBytes)); err != nil {
			return err
		}
		slog.Info("applied migration", "file", name)
	}

	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, name, sqlText string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for migration %q: %w", name, err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op if the transaction was already committed

	if _, err := tx.ExecContext(ctx, sqlText); err != nil {
		return fmt.Errorf("applying migration %q: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("recording migration %q: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing migration %q: %w", name, err)
	}
	return nil
}
