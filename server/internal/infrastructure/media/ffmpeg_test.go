package media

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func sample(t *testing.T, f *FFmpeg, codec string) []byte {
	t.Helper()
	path := filepath.Join(t.TempDir(), "s.mp4")
	cmd := exec.Command(f.ffmpeg, "-v", "error", "-f", "lavfi", "-i", "testsrc=size=640x360:rate=10:duration=1", "-c:v", codec, "-tag:v", map[string]string{"libx265": "hvc1", "libx264": "avc1"}[codec], "-pix_fmt", "yuv420p", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("cannot make %s sample: %v %s", codec, err, out)
	}
	data, _ := os.ReadFile(path)
	return data
}

func TestEnsureWebPlayableConvertsHEVC(t *testing.T) {
	f := NewFFmpeg()
	if f == nil {
		t.Skip("ffmpeg not installed")
	}
	ctx := context.Background()
	hevc := sample(t, f, "libx265")
	out, ext, ct, converted, err := f.EnsureWebPlayable(ctx, hevc, "mp4")
	if err != nil || !converted || ext != "mp4" || ct != "video/mp4" {
		t.Fatalf("converted=%v ext=%s ct=%s err=%v", converted, ext, ct, err)
	}
	path := filepath.Join(t.TempDir(), "o.mp4")
	_ = os.WriteFile(path, out, 0o600)
	if codec, _ := f.VideoCodec(ctx, path); codec != "h264" {
		t.Fatalf("codec after convert = %q", codec)
	}
	h264 := sample(t, f, "libx264")
	same, _, _, converted, err := f.EnsureWebPlayable(ctx, h264, "mp4")
	if err != nil || converted || !bytes.Equal(same, h264) {
		t.Fatalf("h264 must pass through: converted=%v err=%v", converted, err)
	}
}
