package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Austycartcartt/mu3ic/server/internal/auth"
	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

// minPasswordLen is a deliberately low floor — this is a personal app,
// not a public service. The client enforces the same minimum.
const minPasswordLen = 8

// emailRE is a loose sanity check, not RFC 5322 — it just rejects the
// obviously-not-an-email inputs (no @, no dot in the domain, whitespace).
var emailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// authResponse is the shape returned by both register and login.
type authResponse struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var creds credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	creds.Email = strings.TrimSpace(strings.ToLower(creds.Email))

	if !emailRE.MatchString(creds.Email) {
		writeJSONError(w, http.StatusBadRequest, "invalid email address")
		return
	}
	if len(creds.Password) < minPasswordLen {
		writeJSONError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(creds.Password)
	if err != nil {
		s.logger.Error("hashing password", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	user, err := s.store.CreateUser(r.Context(), creds.Email, hash)
	if err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			writeJSONError(w, http.StatusConflict, "email already registered")
			return
		}
		s.logger.Error("creating user", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	s.writeToken(w, http.StatusCreated, user)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var creds credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	creds.Email = strings.TrimSpace(strings.ToLower(creds.Email))

	user, err := s.store.GetUserByEmail(r.Context(), creds.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Same message and status as a wrong password, so this doesn't
			// leak which emails have accounts.
			writeJSONError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		s.logger.Error("looking up user", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to log in")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, creds.Password) {
		writeJSONError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	s.writeToken(w, http.StatusOK, user)
}

// writeToken issues a fresh token for user and writes the standard auth
// response.
func (s *Server) writeToken(w http.ResponseWriter, status int, user store.User) {
	token, err := auth.GenerateToken(user.ID, s.jwtSecret)
	if err != nil {
		s.logger.Error("generating token", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	writeJSON(w, status, authResponse{
		ID:        user.ID,
		Email:     user.Email,
		Token:     token,
		ExpiresAt: time.Now().Add(auth.TokenTTL()),
	})
}
