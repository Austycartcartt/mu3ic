package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Austycartcartt/mu3ic/server/internal/auth"
	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

// postJSON builds a POST request with a JSON body.
func postJSON(target, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHandleRegisterAndLogin(t *testing.T) {
	s, db, _, _ := testServer(t)
	email := uniqueEmail(t)

	// Register.
	rec := httptest.NewRecorder()
	s.handleRegister(rec, postJSON("/api/auth/register", `{"email":`+strconv.Quote(email)+`,"password":"password123"}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	var reg authResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &reg); err != nil {
		t.Fatalf("decoding register response: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE id = $1`, reg.ID) })
	if reg.Token == "" || reg.Email != email {
		t.Fatalf("register response = %+v, want a token and the email echoed", reg)
	}
	if uid, err := auth.ParseToken(reg.Token, s.jwtSecret); err != nil || uid != reg.ID {
		t.Fatalf("register token: uid=%d err=%v, want uid=%d", uid, err, reg.ID)
	}

	// Duplicate email → 409.
	rec = httptest.NewRecorder()
	s.handleRegister(rec, postJSON("/api/auth/register", `{"email":`+strconv.Quote(email)+`,"password":"password123"}`))
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate register status = %d, want 409", rec.Code)
	}

	// Login with the right password → 200 + usable token.
	rec = httptest.NewRecorder()
	s.handleLogin(rec, postJSON("/api/auth/login", `{"email":`+strconv.Quote(email)+`,"password":"password123"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var login authResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &login); err != nil {
		t.Fatalf("decoding login response: %v", err)
	}
	if uid, err := auth.ParseToken(login.Token, s.jwtSecret); err != nil || uid != reg.ID {
		t.Fatalf("login token: uid=%d err=%v, want uid=%d", uid, err, reg.ID)
	}

	// Login with a wrong password → 401.
	rec = httptest.NewRecorder()
	s.handleLogin(rec, postJSON("/api/auth/login", `{"email":`+strconv.Quote(email)+`,"password":"wrongpassword"}`))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad-password login status = %d, want 401", rec.Code)
	}

	// Login with an unknown email → 401 (same as wrong password).
	rec = httptest.NewRecorder()
	s.handleLogin(rec, postJSON("/api/auth/login", `{"email":"nobody-`+email+`","password":"password123"}`))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unknown-email login status = %d, want 401", rec.Code)
	}
}

func TestHandleRegister_Validation(t *testing.T) {
	s, _, _, _ := testServer(t)

	cases := []struct {
		name string
		body string
	}{
		{"missing @", `{"email":"not-an-email","password":"password123"}`},
		{"short password", `{"email":"someone@example.test","password":"short"}`},
		{"malformed JSON", `{"email":`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			s.handleRegister(rec, postJSON("/api/auth/register", tc.body))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCrossUserIsolation(t *testing.T) {
	s, db, _, userA := testServer(t)
	trackA := insertTestTrack(t, s, db, userA.ID, store.NewTrack{Artist: "A's Artist", Title: "A's Song"})
	userB := createTestUser(t, s, db, uniqueEmail(t))

	// B's track list does not include A's track.
	req := httptest.NewRequest(http.MethodGet, "/api/tracks", nil)
	rec := httptest.NewRecorder()
	s.handleList(rec, reqAs(userB.ID, req))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Fatalf("B's track list = %s, want []", got)
	}

	// B cannot stream A's track — it 404s exactly like a missing id.
	req = httptest.NewRequest(http.MethodGet, "/api/tracks/x/stream", nil)
	req.SetPathValue("id", strconv.FormatInt(trackA.ID, 10))
	rec = httptest.NewRecorder()
	s.handleStream(rec, reqAs(userB.ID, req))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("B streaming A's track: status = %d, want 404", rec.Code)
	}
}
