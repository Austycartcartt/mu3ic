package library

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewR2Storage_Validation(t *testing.T) {
	tests := []struct {
		name string
		cfg  R2Config
	}{
		{"empty bucket", R2Config{Endpoint: "https://acc.r2.cloudflarestorage.com"}},
		{"empty endpoint", R2Config{Bucket: "b"}},
		{"garbage endpoint", R2Config{Bucket: "b", Endpoint: "://not a url"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewR2Storage(tc.cfg); err == nil {
				t.Errorf("NewR2Storage(%+v) error = nil, want non-nil", tc.cfg)
			}
		})
	}
}

// PresignedGetObject signs locally with no network call, so the URL shape
// is testable offline.
func TestR2Storage_PresignGet(t *testing.T) {
	s, err := NewR2Storage(R2Config{
		Endpoint:        "https://account123.r2.cloudflarestorage.com",
		Bucket:          "mu3ic-audio",
		AccessKeyID:     "AKIAEXAMPLE",
		SecretAccessKey: "secretexample",
	})
	if err != nil {
		t.Fatalf("NewR2Storage: %v", err)
	}

	raw, err := s.PresignGet(context.Background(), "abc-key", "audio/flac", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parsing presigned url %q: %v", raw, err)
	}
	if u.Host != "account123.r2.cloudflarestorage.com" {
		t.Errorf("host = %q, want the R2 endpoint host", u.Host)
	}
	if !strings.Contains(u.Path, "mu3ic-audio") || !strings.Contains(u.Path, "abc-key") {
		t.Errorf("path = %q, want it to contain bucket and key", u.Path)
	}
	q := u.Query()
	if q.Get("X-Amz-Signature") == "" {
		t.Error("presigned url has no X-Amz-Signature")
	}
	if got := q.Get("response-content-type"); got != "audio/flac" {
		t.Errorf("response-content-type = %q, want %q", got, "audio/flac")
	}
	if q.Get("X-Amz-Expires") != "900" {
		t.Errorf("X-Amz-Expires = %q, want 900", q.Get("X-Amz-Expires"))
	}
}
