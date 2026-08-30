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

// NeonConfig is the connection info for a Neon Object Storage bucket
// (S3-compatible). Endpoint, Region, AccessKeyID and SecretAccessKey are
// the branch-scoped values Neon injects as AWS_ENDPOINT_URL_S3,
// AWS_REGION, AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY (see
// `neon env pull`); Bucket is the name passed to `neon buckets create`,
// which Neon does not inject.
//
// Endpoint looks like
// https://br-<branch>.storage.c-5.us-east-2.aws.neon.tech — note it
// encodes the branch, so pointing this backend at a different Neon branch
// means a different endpoint, not just different credentials.
type NeonConfig struct {
	Endpoint        string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
}

// NeonStorage stores audio and artwork objects in a Neon Object Storage
// bucket and serves them via short-lived presigned GET URLs, so track
// bytes never transit the app server. It implements Storage and
// Presigner.
//
// It is deliberately a near-twin of R2Storage rather than a shared
// implementation: both are minio-go S3 clients, but they differ exactly
// where a client is constructed — Neon requires path-style addressing and
// signs with a real region (AWS_REGION, e.g. "us-east-2"), whereas R2
// uses virtual-host style and the fixed pseudo-region "auto". Keeping
// them apart makes each backend's requirements legible in one file.
type NeonStorage struct {
	client *minio.Client
	bucket string
}

// neonDefaultRegion is the only region Neon Object Storage serves during
// its public beta; used when AWS_REGION is not set.
const neonDefaultRegion = "us-east-2"

func NewNeonStorage(cfg NeonConfig) (*NeonStorage, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("neon storage: bucket is required")
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("neon storage: invalid endpoint %q", cfg.Endpoint)
	}

	region := cfg.Region
	if region == "" {
		region = neonDefaultRegion
	}

	client, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: u.Scheme != "http",
		Region: region,
		// Neon only supports path-style addressing
		// (https://<endpoint>/<bucket>/<key>); the minio default of
		// auto-detecting would pick virtual-host style for some hosts.
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("neon storage: creating client: %w", err)
	}
	return &NeonStorage{client: client, bucket: cfg.Bucket}, nil
}

func (n *NeonStorage) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	_, err := n.client.PutObject(ctx, n.bucket, key, body, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("neon storage: putting %q: %w", key, err)
	}
	return nil
}

// Open streams an object's bytes. Like R2Storage.Open it's the cold path
// — handlers presign instead — so the preliminary StatObject that turns a
// missing key into fs.ErrNotExist is an acceptable extra round-trip.
func (n *NeonStorage) Open(ctx context.Context, key string) (Object, error) {
	if _, err := n.client.StatObject(ctx, n.bucket, key, minio.StatObjectOptions{}); err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, fmt.Errorf("neon storage: %q: %w", key, fs.ErrNotExist)
		}
		return nil, fmt.Errorf("neon storage: stat %q: %w", key, err)
	}
	obj, err := n.client.GetObject(ctx, n.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("neon storage: getting %q: %w", key, err)
	}
	return obj, nil // *minio.Object is an io.ReadSeekCloser
}

func (n *NeonStorage) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		err := n.client.RemoveObject(ctx, n.bucket, key, minio.RemoveObjectOptions{})
		if err != nil && minio.ToErrorResponse(err).Code != "NoSuchKey" {
			return fmt.Errorf("neon storage: deleting %q: %w", key, err)
		}
	}
	return nil
}

// PresignGet returns a time-limited direct GET URL for key, with the
// response Content-Type pinned to contentType (the DB-authoritative
// value) so it doesn't matter what was stored as object metadata at Put
// time.
func (n *NeonStorage) PresignGet(ctx context.Context, key, contentType string, ttl time.Duration) (string, error) {
	params := url.Values{}
	if contentType != "" {
		params.Set("response-content-type", contentType)
	}
	u, err := n.client.PresignedGetObject(ctx, n.bucket, key, ttl, params)
	if err != nil {
		return "", fmt.Errorf("neon storage: presigning %q: %w", key, err)
	}
	return u.String(), nil
}
