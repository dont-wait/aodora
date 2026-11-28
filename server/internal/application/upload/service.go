// Package upload validates admin image uploads and stores them in object storage.
package upload

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"slices"
	"time"

	"golang.org/x/image/webp"

	"github.com/your-org/aodora/server/internal/infrastructure/storage"
)

const (
	MaxBytes     = 10 << 20
	MinShortSide = 400
	MaxLongSide  = 8000
	CacheControl = "public, max-age=31536000, immutable"
)

// Purposes decide the key prefix.
var Purposes = []string{"product", "category", "content", "accessory"}

// Invalid is a user-correctable problem with the uploaded file.
type Invalid struct{ Field, Message string }

func (e Invalid) Error() string { return e.Field + ": " + e.Message }

// ErrUnavailable is returned when object storage is missing or failing.
var ErrUnavailable = errors.New("image storage unavailable")

// ErrNotConfigured is returned when RUSTFS_* credentials are missing.
var ErrNotConfigured = errors.New("image storage is not configured")

type Result struct {
	URL         string `json:"url"`
	Key         string `json:"key"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Size        int    `json:"size"`
	ContentType string `json:"content_type"`
}

type Service struct {
	transcoder VideoTranscoder
	// OnTranscodeError is told when a video had to be stored unconverted.
	OnTranscodeError func(error)
	store            storage.ObjectStore
	now              func() time.Time
	newID            func() string
}

// NewService accepts a nil store: every upload then answers ErrNotConfigured.
func NewService(store storage.ObjectStore) *Service {
	return &Service{store: store, now: time.Now, newID: uuid}
}

// Info is what Inspect learns from the bytes.
type Info struct {
	ContentType string
	Ext         string
	Width       int
	Height      int
}

// Inspect identifies JPEG/PNG/WebP by magic bytes (never by file name or Content-Type) and reads the size.
func Inspect(data []byte) (Info, error) {
	var info Info
	var cfg image.Config
	var err error
	switch {
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		info.ContentType, info.Ext = "image/jpeg", "jpg"
		cfg, err = jpeg.DecodeConfig(bytes.NewReader(data))
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		info.ContentType, info.Ext = "image/png", "png"
		cfg, err = png.DecodeConfig(bytes.NewReader(data))
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		info.ContentType, info.Ext = "image/webp", "webp"
		cfg, err = webp.DecodeConfig(bytes.NewReader(data))
	case bytes.HasPrefix(data, []byte("GIF8")):
		return info, Invalid{"file", "Không nhận ảnh GIF. Hãy dùng JPG, PNG hoặc WebP."}
	case bytes.Contains(firstBytes(data, 512), []byte("<svg")) || bytes.Contains(firstBytes(data, 512), []byte("<?xml")):
		return info, Invalid{"file", "Không nhận ảnh SVG. Hãy dùng JPG, PNG hoặc WebP."}
	case len(data) >= 12 && bytes.Equal(data[4:8], []byte("ftyp")):
		return info, Invalid{"file", "Không nhận ảnh HEIC/HEIF. Hãy xuất ảnh sang JPG hoặc PNG rồi tải lại."}
	default:
		return info, Invalid{"file", "Chỉ nhận ảnh JPG, PNG hoặc WebP."}
	}
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return info, Invalid{"file", "Không đọc được ảnh, file có thể bị hỏng."}
	}
	info.Width, info.Height = cfg.Width, cfg.Height
	short, long := min(cfg.Width, cfg.Height), max(cfg.Width, cfg.Height)
	if short < MinShortSide {
		return info, Invalid{"file", fmt.Sprintf("Ảnh quá nhỏ (%d×%d). Cạnh ngắn cần ít nhất %d px.", cfg.Width, cfg.Height, MinShortSide)}
	}
	if long > MaxLongSide {
		return info, Invalid{"file", fmt.Sprintf("Ảnh quá lớn (%d×%d). Cạnh dài tối đa %d px.", cfg.Width, cfg.Height, MaxLongSide)}
	}
	return info, nil
}

func firstBytes(data []byte, n int) []byte {
	if len(data) < n {
		return data
	}
	return data[:n]
}

// ValidPurpose reports whether purpose is one of Purposes.
func ValidPurpose(purpose string) bool { return slices.Contains(Purposes, purpose) }

// Key builds uploads/<purpose>/<yyyy>/<mm>/<uuid>.<ext>.
func Key(purpose string, at time.Time, id, ext string) string {
	at = at.UTC()
	return fmt.Sprintf("uploads/%s/%04d/%02d/%s.%s", purpose, at.Year(), int(at.Month()), id, ext)
}

func (s *Service) Upload(ctx context.Context, purpose string, data []byte) (Result, error) {
	if s.store == nil {
		return Result{}, ErrNotConfigured
	}
	if !ValidPurpose(purpose) {
		return Result{}, Invalid{"purpose", "purpose phải là product, category, content hoặc accessory"}
	}
	if len(data) == 0 {
		return Result{}, Invalid{"file", "File trống."}
	}
	if len(data) > MaxBytes {
		return Result{}, Invalid{"file", "Ảnh quá lớn, tối đa 10 MB."}
	}
	info, err := Inspect(data)
	if err != nil {
		return Result{}, err
	}
	key := Key(purpose, s.now(), s.newID(), info.Ext)
	if err := s.store.Put(ctx, key, info.ContentType, CacheControl, bytes.NewReader(data), int64(len(data))); err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return Result{URL: s.store.PublicURL() + "/" + key, Key: key, Width: info.Width, Height: info.Height, Size: len(data), ContentType: info.ContentType}, nil
}

// PublicURL returns the bucket's public base URL ("" without storage).
func (s *Service) PublicURL() string {
	if s.store == nil {
		return ""
	}
	return s.store.PublicURL()
}

func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
