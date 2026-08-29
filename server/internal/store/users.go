package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrEmailTaken is returned by CreateUser when the email is already
// registered (Postgres unique-violation on users.email).
var ErrEmailTaken = errors.New("email already registered")

// User mirrors the users table. PasswordHash is only used server-side
// (login verification) and never sent to a client.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// CreateUser inserts a new account. The caller is responsible for hashing
// the password (see internal/auth). A duplicate email comes back as
// ErrEmailTaken so the handler can map it to 409 without a separate
// pre-check SELECT (which would race anyway).
func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, email, password_hash, created_at`,
		email, passwordHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("creating user: %w", err)
	}
	return u, nil
}

// GetUserByEmail looks up an account for login. A missing row comes back
// as sql.ErrNoRows, which the login handler treats the same as a wrong
// password (no user enumeration).
func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return User{}, err
	}
	return u, nil
}
