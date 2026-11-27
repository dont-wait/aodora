package upload

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	key, contentType, cacheControl string
	body                           []byte
	err                            error
	puts                           int
}

func (f *fakeStore) Put(_ context.Context, key, contentType, cacheControl string, body io.Reader, _ int64) error {
	f.puts++
	if f.err != nil {
		return f.err
	}
	f.key, f.contentType, f.cacheControl = key, contentType, cacheControl
	f.body, _ = io.ReadAll(body)
	return nil
}
func (f *fakeStore) Remove(context.Context, string) error { return nil }
func (f *fakeStore) PublicURL() string                    { return "http://localhost:9100/aodora-images" }

func pngBytes(t testing.TB, w, h int) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.Gray{Y: 200})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func jpegBytes(t testing.TB, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, w, h)), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// webpHeader builds the smallest VP8L header that DecodeConfig accepts (dimensions only).
func webpHeader(w, h int) []byte {
	bits := uint32(w-1) | uint32(h-1)<<14
	chunk := make([]byte, 0, 5)
	chunk = append(chunk, 0x2f)
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], bits)
	chunk = append(chunk, b[:]...)
	var out bytes.Buffer
	out.WriteString("RIFF")
	var size [4]byte
	binary.LittleEndian.PutUint32(size[:], uint32(4+8+len(chunk)))
	out.Write(size[:])
	out.WriteString("WEBPVP8L")
	binary.LittleEndian.PutUint32(size[:], uint32(len(chunk)))
	out.Write(size[:])
	out.Write(chunk)
	return out.Bytes()
}

func TestInspectDetectsTypeBySignatureAndReadsSize(t *testing.T) {
	for _, tc := range []struct {
		name     string
		data     []byte
		wantType string
		ext      string
	}{
		{"jpeg", jpegBytes(t, 500, 400), "image/jpeg", "jpg"},
		{"png", pngBytes(t, 400, 800), "image/png", "png"},
		{"webp", webpHeader(640, 480), "image/webp", "webp"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info, err := Inspect(tc.data)
			if err != nil || info.ContentType != tc.wantType || info.Ext != tc.ext || info.Width == 0 || info.Height == 0 {
				t.Fatalf("info=%+v err=%v", info, err)
			}
		})
	}
	info, _ := Inspect(webpHeader(640, 480))
	if info.Width != 640 || info.Height != 480 {
		t.Fatalf("webp size = %dx%d", info.Width, info.Height)
	}
}

func TestInspectRejectsWrongKindsWithHints(t *testing.T) {
	heic := append([]byte{0, 0, 0, 24}, []byte("ftypheic")...)
	for name, tc := range map[string]struct {
		data []byte
		want string
	}{
		"svg":          {[]byte("<?xml version=\"1.0\"?><svg xmlns=\"http://www.w3.org/2000/svg\"></svg>"), "SVG"},
		"svg no xml":   {[]byte("<svg></svg>"), "SVG"},
		"gif":          {[]byte("GIF89a\x01\x00\x01\x00"), "GIF"},
		"heic":         {heic, "HEIC"},
		"text as jpg":  {[]byte(strings.Repeat("hello ", 100)), "JPG, PNG hoặc WebP"},
		"empty-ish":    {[]byte{1, 2}, "JPG, PNG hoặc WebP"},
		"corrupt jpeg": {[]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}, "hỏng"},
		"corrupt png":  {append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, 1, 2, 3), "hỏng"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Inspect(tc.data)
			var invalid Invalid
			if !errors.As(err, &invalid) || invalid.Field != "file" || !strings.Contains(invalid.Message, tc.want) {
				t.Fatalf("err = %v, want message containing %q", err, tc.want)
			}
		})
	}
}

func TestInspectSizeLimits(t *testing.T) {
	for _, tc := range []struct {
		name    string
		w, h    int
		wantErr string
	}{
		{"short side 399", 399, 800, "quá nhỏ"},
		{"short side 400 ok", 400, 400, ""},
		{"long side 8000 ok", 8000, 400, ""},
		{"long side 8001", 8001, 400, "quá lớn"},
		{"both small", 300, 300, "quá nhỏ"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Inspect(pngBytes(t, tc.w, tc.h))
			if tc.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var invalid Invalid
			if !errors.As(err, &invalid) || !strings.Contains(invalid.Message, tc.wantErr) {
				t.Fatalf("err = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestUploadStoresUnderDatedKeyWithHeaders(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)
	svc.now = func() time.Time { return time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC) }
	svc.newID = func() string { return "11111111-2222-4333-8444-555555555555" }
	data := pngBytes(t, 640, 480)
	res, err := svc.Upload(context.Background(), "product", data)
	if err != nil {
		t.Fatal(err)
	}
	wantKey := "uploads/product/2026/03/11111111-2222-4333-8444-555555555555.png"
	if res.Key != wantKey || store.key != wantKey || res.URL != "http://localhost:9100/aodora-images/"+wantKey {
		t.Fatalf("key/url: %+v / %q", res, store.key)
	}
	if store.contentType != "image/png" || store.cacheControl != "public, max-age=31536000, immutable" {
		t.Fatalf("headers: %q %q", store.contentType, store.cacheControl)
	}
	if res.Width != 640 || res.Height != 480 || res.Size != len(data) || res.ContentType != "image/png" || !bytes.Equal(store.body, data) {
		t.Fatalf("result: %+v", res)
	}
}

func TestUploadRejectsBeforeTouchingTheStore(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)
	good := pngBytes(t, 500, 500)
	for name, call := range map[string]func() error{
		"bad purpose": func() error { _, err := svc.Upload(context.Background(), "avatar", good); return err },
		"empty":       func() error { _, err := svc.Upload(context.Background(), "product", nil); return err },
		"too large": func() error {
			_, err := svc.Upload(context.Background(), "product", make([]byte, MaxBytes+1))
			return err
		},
		"not image": func() error { _, err := svc.Upload(context.Background(), "product", []byte("hello")); return err },
	} {
		var invalid Invalid
		if err := call(); !errors.As(err, &invalid) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if store.puts != 0 {
		t.Fatalf("store was called %d times", store.puts)
	}
	for _, purpose := range Purposes {
		if _, err := svc.Upload(context.Background(), purpose, good); err != nil {
			t.Errorf("purpose %s: %v", purpose, err)
		}
	}
}

func TestUploadStorageFailures(t *testing.T) {
	if _, err := NewService(nil).Upload(context.Background(), "product", pngBytes(t, 500, 500)); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("nil store: %v", err)
	}
	store := &fakeStore{err: errors.New("connection refused")}
	if _, err := NewService(store).Upload(context.Background(), "product", pngBytes(t, 500, 500)); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("store error: %v", err)
	}
}
