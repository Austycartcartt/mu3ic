package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestLimiter(now func() time.Time) *ipRateLimiter {
	return &ipRateLimiter{buckets: make(map[string]*tokenBucket), now: now}
}

func TestIPRateLimiter_BurstThenThrottle(t *testing.T) {
	fixed := time.Unix(1_700_000_000, 0)
	l := newTestLimiter(func() time.Time { return fixed })

	// The first authBurst requests in the same instant are allowed.
	for i := 0; i < int(authBurst); i++ {
		if !l.allow("1.2.3.4") {
			t.Fatalf("request %d denied, want allowed within burst", i+1)
		}
	}
	if l.allow("1.2.3.4") {
		t.Fatal("request past the burst was allowed, want throttled")
	}

	// A different IP has its own bucket.
	if !l.allow("5.6.7.8") {
		t.Fatal("a fresh IP was throttled")
	}
}

func TestIPRateLimiter_RefillsOverTime(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	l := newTestLimiter(func() time.Time { return now })

	for i := 0; i < int(authBurst); i++ {
		l.allow("9.9.9.9")
	}
	if l.allow("9.9.9.9") {
		t.Fatal("bucket should be empty")
	}

	now = now.Add(2 * time.Second) // refill ~2 tokens at 1/sec
	if !l.allow("9.9.9.9") {
		t.Fatal("bucket should have refilled after 2s")
	}
}

func TestWithRateLimit_Returns429(t *testing.T) {
	s := &Server{authLimiter: newTestLimiter(func() time.Time { return time.Unix(1, 0) })}
	h := s.withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var lastCode int
	for i := 0; i < int(authBurst)+1; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.RemoteAddr = "203.0.113.5:5555"
		h.ServeHTTP(rec, req)
		lastCode = rec.Code
	}
	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("final request status = %d, want 429", lastCode)
	}
}

func TestClientIP_TrustProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234" // the proxy
	req.Header.Set("X-Real-IP", "198.51.100.9")

	if got := clientIP(req, false); got != "10.0.0.1" {
		t.Errorf("clientIP(trustProxy=false) = %q, want the RemoteAddr host", got)
	}
	if got := clientIP(req, true); got != "198.51.100.9" {
		t.Errorf("clientIP(trustProxy=true) = %q, want the X-Real-IP value", got)
	}
}
