package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// StorageConfig is the S3-compatible object store (RustFS) used for admin image uploads.
type StorageConfig struct {
	Endpoint  string // API endpoint, e.g. http://localhost:9100
	AccessKey string
	SecretKey string
	Bucket    string
	PublicURL string // public base URL of the bucket, no trailing slash
}

// Configured reports whether uploads can work (credentials present).
func (c StorageConfig) Configured() bool {
	return c.AccessKey != "" && c.SecretKey != "" && c.Bucket != "" && c.Endpoint != ""
}

func rustfsPort() string {
	if port := strings.TrimSpace(os.Getenv("RUSTFS_API_PORT")); port != "" {
		return port
	}
	return "9100"
}

func rustfsBucket() string {
	if bucket := strings.TrimSpace(os.Getenv("RUSTFS_BUCKET")); bucket != "" {
		return bucket
	}
	return "aodora-images"
}

// RustFSPublicURL returns RUSTFS_PUBLIC_URL, or http://localhost:$RUSTFS_API_PORT/$RUSTFS_BUCKET when blank.
// Shared by the API, the seeder and the Sumire importer.
func RustFSPublicURL() string {
	if value := strings.TrimSpace(os.Getenv("RUSTFS_PUBLIC_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return "http://localhost:" + rustfsPort() + "/" + rustfsBucket()
}

// ValidatePublicBaseURL checks that base is a plain http(s) URL (no credentials, query or fragment).
func ValidatePublicBaseURL(base string) error {
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("RUSTFS_PUBLIC_URL must be a public HTTP(S) bucket URL")
	}
	return nil
}

// LoadStorageConfig reads the RustFS settings. Missing credentials are not an error:
// the API still starts and the upload endpoint answers 503 STORAGE_UNAVAILABLE.
func LoadStorageConfig() StorageConfig {
	endpoint := strings.TrimRight(strings.TrimSpace(os.Getenv("RUSTFS_ENDPOINT")), "/")
	if endpoint == "" {
		endpoint = "http://localhost:" + rustfsPort()
	}
	return StorageConfig{
		Endpoint:  endpoint,
		AccessKey: strings.TrimSpace(os.Getenv("RUSTFS_ACCESS_KEY")),
		SecretKey: strings.TrimSpace(os.Getenv("RUSTFS_SECRET_KEY")),
		Bucket:    rustfsBucket(),
		PublicURL: RustFSPublicURL(),
	}
}
