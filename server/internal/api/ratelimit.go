package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Auth-endpoint rate limit: a token bucket per client IP. Generous enough
// that a human logging in (or a client retrying a flaky request) never
// notices, tight enough that credential brute-forcing against a public
// deployment is pointless. Hand-rolled rather than pulling in
// golang.org/x/time/rate — it's a few dozen lines and PROJECT.md wants
// every dependency justified.
const (
	authRefillPerSec  = 1.0 // sustained requests/sec allowed per IP
	authBurst         = 5.0 // bucket size: back-to-back requests before throttling
	limiterIdleTTL    = 10 * time.Minute
	limiterSweepEvery = 5 * time.Minute
)

type tokenBucket struct {
	tokens   float64
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	now     func() time.Time // swappable in tests
}

func newIPRateLimiter() *ipRateLimiter {
	l := &ipRateLimiter{
		buckets: make(map[string]*tokenBucket),
		now:     time.Now,
	}
	go l.sweepLoop()
	return l
}

// allow reports whether a request from ip may proceed, consuming one
// token if so.
func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b := l.buckets[ip]
	if b == nil {
		b = &tokenBucket{tokens: authBurst, lastSeen: now}
		l.buckets[ip] = b
	} else {
		elapsed := now.Sub(b.lastSeen).Seconds()
		b.tokens += elapsed * authRefillPerSec
		if b.tokens > authBurst {
			b.tokens = authBurst
		}
		b.lastSeen = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *ipRateLimiter) sweepLoop() {
	ticker := time.NewTicker(limiterSweepEvery)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-limiterIdleTTL)
		l.mu.Lock()
		for ip, b := range l.buckets {
			if b.lastSeen.Before(cutoff) {
				delete(l.buckets, ip)
			}
		}
		l.mu.Unlock()
	}
}

// withRateLimit throttles by client IP. It's meant to wrap only the
// unauthenticated auth endpoints.
func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authLimiter.allow(clientIP(r, s.trustProxy)) {
			w.Header().Set("Retry-After", "1")
			writeJSONError(w, http.StatusTooManyRequests, "too many requests, slow down")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP resolves the caller's address. Behind a trusted proxy the TCP
// peer is the proxy, so we read X-Real-IP, which the Phase 8 Caddy config
// sets to the real remote host (Caddy overwrites the header, so a client
// can't spoof it). Without a trusted proxy we use RemoteAddr and ignore
// any client-supplied forwarding headers.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if v := strings.TrimSpace(r.Header.Get("X-Real-IP")); v != "" {
			return v
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
