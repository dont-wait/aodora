// Package media makes customer videos playable in every browser by
// re-encoding codecs browsers cannot decode (HEVC from phones) to H.264.
package media

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WebPlayable are video codecs Chrome, Firefox and Safari all decode in software.
var WebPlayable = map[string]bool{"h264": true, "vp8": true, "vp9": true, "av1": true}

type FFmpeg struct {
	ffmpeg, ffprobe string
	Timeout         time.Duration
	// slots bounds concurrent encodes so a few uploads cannot starve the API of CPU.
	slots chan struct{}
}

// NewFFmpeg finds ffmpeg/ffprobe on PATH; nil when either is missing.
func NewFFmpeg() *FFmpeg {
	ff, err1 := exec.LookPath("ffmpeg")
	fp, err2 := exec.LookPath("ffprobe")
	if err1 != nil || err2 != nil {
		return nil
	}
	return &FFmpeg{ffmpeg: ff, ffprobe: fp, Timeout: 2 * time.Minute, slots: make(chan struct{}, 2)}
}

// VideoCodec reports the first video stream's codec name (e.g. "hevc", "h264").
func (f *FFmpeg) VideoCodec(ctx context.Context, path string) (string, error) {
	out, err := exec.CommandContext(ctx, f.ffprobe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name", "-of", "default=nw=1:nk=1", path).Output()
	if err != nil {
		return "", fmt.Errorf("ffprobe: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// EnsureWebPlayable returns data unchanged when the video codec already plays
// in browsers; otherwise an H.264/AAC MP4 (at most 1280 px on the long side).
func (f *FFmpeg) EnsureWebPlayable(ctx context.Context, data []byte, ext string) (out []byte, outExt, contentType string, converted bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, f.Timeout)
	defer cancel()
	dir, err := os.MkdirTemp("", "aodora-video-*")
	if err != nil {
		return nil, "", "", false, err
	}
	defer os.RemoveAll(dir)
	in := filepath.Join(dir, "in."+ext)
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return nil, "", "", false, err
	}
	codec, err := f.VideoCodec(ctx, in)
	if err != nil {
		return nil, "", "", false, err
	}
	if codec == "" {
		return nil, "", "", false, fmt.Errorf("no video stream")
	}
	if WebPlayable[codec] {
		return data, ext, "", false, nil
	}
	select {
	case f.slots <- struct{}{}:
		defer func() { <-f.slots }()
	case <-ctx.Done():
		return nil, "", "", false, ctx.Err()
	}
	dst := filepath.Join(dir, "out.mp4")
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, f.ffmpeg, "-nostdin", "-y", "-i", in,
		"-map", "0:v:0", "-map", "0:a:0?",
		"-vf", "scale='if(gt(iw,ih),min(1280,iw),-2)':'if(gt(iw,ih),-2,min(1280,ih))'",
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "26", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart", dst)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if len(msg) > 400 {
			msg = msg[len(msg)-400:]
		}
		return nil, "", "", false, fmt.Errorf("ffmpeg %s: %w: %s", codec, err, msg)
	}
	out, err = os.ReadFile(dst)
	if err != nil {
		return nil, "", "", false, err
	}
	return out, "mp4", "video/mp4", true, nil
}
