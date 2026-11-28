package upload

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func mp4Header(brand string) []byte {
	return append([]byte{0, 0, 0, 0x18, 'f', 't', 'y', 'p'}, []byte(brand+"\x00\x00\x00\x00isommp42")...)
}

func TestInspectEvidenceKinds(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		kind string
		ext  string
	}{
		{"jpeg", jpegBytes(t, 800, 600), "image", "jpg"},
		{"mp4", mp4Header("isom"), "video", "mp4"},
		{"mov", mp4Header("qt  "), "video", "mov"},
		{"webm", []byte{0x1A, 0x45, 0xDF, 0xA3, 0x9f, 0x42, 0x86, 0x81}, "video", "webm"},
	}
	for _, c := range cases {
		kind, _, ext, err := InspectEvidence(c.data)
		if err != nil || kind != c.kind || ext != c.ext {
			t.Errorf("%s: kind=%s ext=%s err=%v", c.name, kind, ext, err)
		}
	}
	if _, _, _, err := InspectEvidence(mp4Header("heic")); err == nil {
		t.Error("HEIC photos must be refused with a hint to convert")
	}
	if _, _, _, err := InspectEvidence([]byte("hello world, not media")); err == nil || !strings.Contains(err.Error(), "video MP4") {
		t.Errorf("unknown files: %v", err)
	}
	big := append(mp4Header("mp42"), make([]byte, MaxEvidenceVideoBytes)...)
	if _, _, _, err := InspectEvidence(big); err == nil || !strings.Contains(err.Error(), "50 MB") {
		t.Errorf("oversized video: %v", err)
	}
}

func TestUploadEvidenceKeyIsPerCustomer(t *testing.T) {
	store := &fakeStore{}
	s := NewService(store)
	s.now = func() time.Time { return time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC) }
	s.newID = func() string { return "id1" }
	res, err := s.UploadEvidence(context.Background(), "abc123", mp4Header("isom"))
	if err != nil {
		t.Fatal(err)
	}
	if store.key != "uploads/returns/abc123/2026/11/id1.mp4" || res.Kind != "video" || !strings.HasPrefix(res.URL, s.EvidencePrefix("abc123")) {
		t.Fatalf("key %s result %+v", store.key, res)
	}
	if _, err := NewService(nil).UploadEvidence(context.Background(), "abc", mp4Header("isom")); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("no store: %v", err)
	}
}

type fakeTranscoder struct{ err error }

func (f fakeTranscoder) EnsureWebPlayable(_ context.Context, data []byte, ext string) ([]byte, string, string, bool, error) {
	if f.err != nil {
		return nil, "", "", false, f.err
	}
	return []byte("h264"), "mp4", "video/mp4", true, nil
}

func TestUploadEvidenceConvertsVideos(t *testing.T) {
	store := &fakeStore{}
	s := NewService(store).WithVideoTranscoder(fakeTranscoder{})
	s.newID = func() string { return "v" }
	res, err := s.UploadEvidence(context.Background(), "c", mp4Header("qt  "))
	if err != nil || !strings.HasSuffix(store.key, "/v.mp4") || string(store.body) != "h264" || res.ContentType != "video/mp4" {
		t.Fatalf("key=%s body=%q res=%+v err=%v", store.key, store.body, res, err)
	}
	failing := NewService(store).WithVideoTranscoder(fakeTranscoder{err: errors.New("boom")})
	failures := 0
	failing.OnTranscodeError = func(error) { failures++ }
	if _, err := failing.UploadEvidence(context.Background(), "c", mp4Header("qt  ")); err != nil || failures != 1 || !strings.HasSuffix(store.key, ".mov") {
		t.Fatalf("a failed conversion must still keep the original: key=%s err=%v", store.key, err)
	}
}
