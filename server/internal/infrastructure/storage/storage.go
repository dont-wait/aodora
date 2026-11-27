// Package storage talks to the S3-compatible object store (RustFS).
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/your-org/aodora/server/internal/config"
)

// ErrNotConfigured means credentials are missing; uploads answer 503.
var ErrNotConfigured = errors.New("object storage is not configured")

// ObjectStore is the small surface the upload service needs (faked in unit tests).
type ObjectStore interface {
	// Put stores the object and returns after it is durable.
	Put(ctx context.Context, key, contentType, cacheControl string, body io.Reader, size int64) error
	// Remove deletes one object (used by tests and clean-up tooling).
	Remove(ctx context.Context, key string) error
	// PublicURL is the base URL under which keys are readable, without a trailing slash.
	PublicURL() string
}

type s3Store struct {
	client *minio.Client
	bucket string
	public string
}

// New builds the S3 client. It returns ErrNotConfigured when credentials are missing.
func New(cfg config.StorageConfig) (ObjectStore, error) {
	if !cfg.Configured() {
		return nil, ErrNotConfigured
	}
	endpoint, secure, err := splitEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	client, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""), Secure: secure, Region: "us-east-1"})
	if err != nil {
		return nil, fmt.Errorf("create object storage client: %w", err)
	}
	return &s3Store{client: client, bucket: cfg.Bucket, public: cfg.PublicURL}, nil
}

func splitEndpoint(raw string) (host string, secure bool, err error) {
	switch {
	case len(raw) > 8 && raw[:8] == "https://":
		return raw[8:], true, nil
	case len(raw) > 7 && raw[:7] == "http://":
		return raw[7:], false, nil
	}
	return "", false, fmt.Errorf("RUSTFS_ENDPOINT must start with http:// or https://")
}

func (s *s3Store) Put(ctx context.Context, key, contentType, cacheControl string, body io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{ContentType: contentType, CacheControl: cacheControl})
	return err
}

func (s *s3Store) Remove(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *s3Store) PublicURL() string { return s.public }
