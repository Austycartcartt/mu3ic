package library

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewNeonStorage_Validation(t *testing.T) {
	tests := []struct {
		name string
		cfg  NeonConfig
	}{
		{"empty bucket", NeonConfig{Endpoint: "https://br-x.storage.c-5.us-east-2.aws.neon.tech"}},
		{"empty endpoint", NeonConfig{Bucket: "b"}},
		{"garbage endpoint", NeonConfig{Bucket: "b", Endpoint: "://not a url"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewNeonStorage(tc.cfg); err == nil {
				t.Errorf("NewNeonStorage(%+v) error = nil, want non-nil", tc.cfg)
			}
		})
	}
}

// PresignedGetObject signs locally with no network call, so the URL shape
// is testable offline.
func TestNeonStorage_PresignGet(t *testing.T) {
	s, err := NewNeonStorage(NeonConfig{
		Endpoint:        "https://br-cool-dew-aytkwyv4.storage.c-5.us-east-2.aws.neon.tech",
		Bucket:          "mu3ic-audio",
		AccessKeyID:     "nak_live_example",
		SecretAccessKey: "nsk_live_example",
		Region:          "us-east-2",
	})
	if err != nil {
		t.Fatalf("NewNeonStorage: %v", err)
	}

	raw, err := s.PresignGet(context.Background(), "abc-key", "audio/flac", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parsing presigned url %q: %v", raw, err)
	}
	if u.Host != "br-cool-dew-aytkwyv4.storage.c-5.us-east-2.aws.neon.tech" {
		t.Errorf("host = %q, want the Neon storage endpoint host", u.Host)
	}
	// Path-style addressing: /<bucket>/<key>.
	if !strings.HasPrefix(u.Path, "/mu3ic-audio/") || !strings.Contains(u.Path, "abc-key") {
		t.Errorf("path = %q, want path-style /<bucket>/<key>", u.Path)
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
	// Neon signs with the real region, not R2's "auto".
	if cred := q.Get("X-Amz-Credential"); !strings.Contains(cred, "/us-east-2/s3/aws4_request") {
		t.Errorf("X-Amz-Credential = %q, want it scoped to the us-east-2 s3 service", cred)
	}
}

// An unset Region falls back to the beta's only region rather than
// producing an unsigned-region client.
func TestNewNeonStorage_DefaultRegion(t *testing.T) {
	s, err := NewNeonStorage(NeonConfig{
		Endpoint:        "https://br-x.storage.c-5.us-east-2.aws.neon.tech",
		Bucket:          "mu3ic-audio",
		AccessKeyID:     "nak_live_example",
		SecretAccessKey: "nsk_live_example",
	})
	if err != nil {
		t.Fatalf("NewNeonStorage: %v", err)
	}
	raw, err := s.PresignGet(context.Background(), "k", "", time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
	u, _ := url.Parse(raw)
	if cred := u.Query().Get("X-Amz-Credential"); !strings.Contains(cred, "/"+neonDefaultRegion+"/s3/aws4_request") {
		t.Errorf("X-Amz-Credential = %q, want default region %q", cred, neonDefaultRegion)
	}
}
