package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	want := `{"status":"ok"}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestHandleStream_InvalidID(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/tracks/not-a-number/stream", nil)
	req.SetPathValue("id", "not-a-number")
	rec := httptest.NewRecorder()

	s.handleStream(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
