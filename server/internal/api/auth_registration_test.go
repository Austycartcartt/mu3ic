package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// With registration closed and the shared dev DB already holding users,
// CreateFirstUser matches no rows and the handler must return 403 rather
// than creating an account.
func TestHandleRegister_ClosedRejects(t *testing.T) {
	s, _, _, _ := testServer(t, func(o *Options) {
		o.Registration = RegistrationPolicy{} // not open, no invite code
	})

	rec := httptest.NewRecorder()
	s.handleRegister(rec, postJSON("/api/auth/register",
		`{"email":`+strconv.Quote(uniqueEmail(t))+`,"password":"password123"}`))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestHandleRegister_InviteCode(t *testing.T) {
	s, db, _, _ := testServer(t, func(o *Options) {
		o.Registration = RegistrationPolicy{InviteCode: "let-me-in"}
	})

	t.Run("wrong code → 403", func(t *testing.T) {
		rec := httptest.NewRecorder()
		s.handleRegister(rec, postJSON("/api/auth/register",
			`{"email":`+strconv.Quote(uniqueEmail(t))+`,"password":"password123","inviteCode":"nope"}`))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("missing code → 403", func(t *testing.T) {
		rec := httptest.NewRecorder()
		s.handleRegister(rec, postJSON("/api/auth/register",
			`{"email":`+strconv.Quote(uniqueEmail(t))+`,"password":"password123"}`))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("correct code → 201", func(t *testing.T) {
		email := uniqueEmail(t)
		rec := httptest.NewRecorder()
		s.handleRegister(rec, postJSON("/api/auth/register",
			`{"email":`+strconv.Quote(email)+`,"password":"password123","inviteCode":"let-me-in"}`))
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
		}
		t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE email = $1`, email) })
	})
}
