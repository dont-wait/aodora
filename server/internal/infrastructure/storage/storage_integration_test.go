//go:build integration

package storage_test

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/your-org/aodora/server/internal/application/upload"
	"github.com/your-org/aodora/server/internal/config"
	"github.com/your-org/aodora/server/internal/infrastructure/storage"
)

// Needs a running RustFS and RUSTFS_* in the environment (make storage-up; source server/.env).
func TestUploadToRealRustFS(t *testing.T) {
	cfg := config.LoadStorageConfig()
	store, err := storage.New(cfg)
	if err != nil {
		t.Skipf("object storage not configured: %v", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, 420, 420))); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	svc := upload.NewService(store)
	res, err := svc.Upload(ctx, "product", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Remove(context.Background(), res.Key) }()

	resp, err := http.Get(res.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !bytes.Equal(body, buf.Bytes()) {
		t.Fatalf("public URL %s: status %d, %d bytes (want %d)", res.URL, resp.StatusCode, len(body), buf.Len())
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content type %q", ct)
	}
	t.Logf("cache-control=%q url=%s", resp.Header.Get("Cache-Control"), res.URL)
	if err := store.Remove(ctx, res.Key); err != nil {
		t.Fatalf("remove: %v", err)
	}
	gone, err := http.Get(res.URL)
	if err != nil {
		t.Fatal(err)
	}
	gone.Body.Close()
	if gone.StatusCode == http.StatusOK {
		t.Fatal("object still readable after removal")
	}
}
