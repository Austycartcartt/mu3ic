package library

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// R2Config is the connection info for a Cloudflare R2 bucket (S3 API).
// Endpoint is the full S3 API URL, e.g.
// https://<account-id>.r2.cloudflarestorage.com.
type R2Config struct {
	Endpoint        string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

// R2Storage stores audio and artwork objects in a Cloudflare R2 bucket
// and serves them via short-lived presigned GET URLs, so track bytes
// never transit the app server. It implements Storage and Presigner.
type R2Storage struct {
	client *minio.Client
	bucket string
}

// r2Region is the fixed region string R2 expects for SigV4 signing.
const r2Region = "auto"

func NewR2Storage(cfg R2Config) (*R2Storage, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("r2: bucket is required")
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("r2: invalid endpoint %q", cfg.Endpoint)
	}

	client, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: u.Scheme != "http",
		Region: r2Region,
	})
	if err != nil {
		return nil, fmt.Errorf("r2: creating client: %w", err)
	}
	return &R2Storage{client: client, bucket: cfg.Bucket}, nil
}

func (r *R2Storage) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	_, err := r.client.PutObject(ctx, r.bucket, key, body, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("r2: putting %q: %w", key, err)
	}
	return nil
}

// Open streams an object's bytes. It's not the hot path — the handlers
// prefer PresignGet and only fall back to Open for tooling or a backend
// without presign support — so a preliminary StatObject to surface a
// missing key as fs.ErrNotExist is an acceptable extra round-trip.
func (r *R2Storage) Open(ctx context.Context, key string) (Object, error) {
	if _, err := r.client.StatObject(ctx, r.bucket, key, minio.StatObjectOptions{}); err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, fmt.Errorf("r2: %q: %w", key, fs.ErrNotExist)
		}
		return nil, fmt.Errorf("r2: stat %q: %w", key, err)
	}
	obj, err := r.client.GetObject(ctx, r.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("r2: getting %q: %w", key, err)
	}
	return obj, nil // *minio.Object is an io.ReadSeekCloser
}

func (r *R2Storage) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		err := r.client.RemoveObject(ctx, r.bucket, key, minio.RemoveObjectOptions{})
		if err != nil && minio.ToErrorResponse(err).Code != "NoSuchKey" {
			return fmt.Errorf("r2: deleting %q: %w", key, err)
		}
	}
	return nil
}

// PresignGet returns a time-limited direct GET URL for key. The response
// Content-Type is pinned to contentType (the DB-authoritative value) via
// a response override parameter, so it doesn't matter what was stored as
// object metadata at Put time.
func (r *R2Storage) PresignGet(ctx context.Context, key, contentType string, ttl time.Duration) (string, error) {
	params := url.Values{}
	if contentType != "" {
		params.Set("response-content-type", contentType)
	}
	u, err := r.client.PresignedGetObject(ctx, r.bucket, key, ttl, params)
	if err != nil {
		return "", fmt.Errorf("r2: presigning %q: %w", key, err)
	}
	return u.String(), nil
}
