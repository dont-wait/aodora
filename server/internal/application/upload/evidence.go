package upload

import (
	"bytes"
	"context"
	"fmt"
)

// Return evidence: customers prove a defect with photos or one short video.
const (
	MaxEvidenceImageBytes = MaxBytes
	MaxEvidenceVideoBytes = 50 << 20
)

type EvidenceResult struct {
	URL         string `json:"url"`
	Key         string `json:"key"`
	Kind        string `json:"kind"` // "image" or "video"
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
}

// isoVideoBrands are ISO-BMFF major brands of MP4/MOV video; HEIC/AVIF photos
// share the ftyp box but use other brands and are refused by Inspect.
var isoVideoBrands = map[string]struct{ contentType, ext string }{
	"isom": {"video/mp4", "mp4"}, "iso2": {"video/mp4", "mp4"}, "iso4": {"video/mp4", "mp4"}, "iso5": {"video/mp4", "mp4"}, "iso6": {"video/mp4", "mp4"},
	"mp41": {"video/mp4", "mp4"}, "mp42": {"video/mp4", "mp4"}, "avc1": {"video/mp4", "mp4"}, "M4V ": {"video/mp4", "mp4"}, "dash": {"video/mp4", "mp4"},
	"3gp4": {"video/mp4", "mp4"}, "3gp5": {"video/mp4", "mp4"}, "3gp6": {"video/mp4", "mp4"},
	"qt  ": {"video/quicktime", "mov"},
}

// InspectEvidence accepts JPG/PNG/WebP photos and MP4/MOV/WebM videos, judged
// by magic bytes only.
func InspectEvidence(data []byte) (kind, contentType, ext string, err error) {
	if len(data) >= 12 && bytes.Equal(data[4:8], []byte("ftyp")) {
		if v, ok := isoVideoBrands[string(data[8:12])]; ok {
			if len(data) > MaxEvidenceVideoBytes {
				return "", "", "", Invalid{"file", "Video quá lớn, tối đa 50 MB. Hãy quay ngắn hơn (khoảng 30 giây)."}
			}
			return "video", v.contentType, v.ext, nil
		}
	}
	if len(data) >= 4 && bytes.Equal(data[:4], []byte{0x1A, 0x45, 0xDF, 0xA3}) {
		if len(data) > MaxEvidenceVideoBytes {
			return "", "", "", Invalid{"file", "Video quá lớn, tối đa 50 MB. Hãy quay ngắn hơn (khoảng 30 giây)."}
		}
		return "video", "video/webm", "webm", nil
	}
	if len(data) > MaxEvidenceImageBytes {
		return "", "", "", Invalid{"file", "Ảnh quá lớn, tối đa 10 MB."}
	}
	info, err := Inspect(data)
	if err != nil {
		var invalid Invalid
		if ok := asInvalid(err, &invalid); ok && invalid.Message == "Chỉ nhận ảnh JPG, PNG hoặc WebP." {
			return "", "", "", Invalid{"file", "Chỉ nhận ảnh JPG, PNG, WebP hoặc video MP4, MOV, WebM."}
		}
		return "", "", "", err
	}
	return "image", info.ContentType, info.Ext, nil
}

func asInvalid(err error, out *Invalid) bool {
	v, ok := err.(Invalid)
	if ok {
		*out = v
	}
	return ok
}

// EvidencePrefix is the URL prefix every evidence file of this customer starts
// with; a return request may only reference files under it.
func (s *Service) EvidencePrefix(customerID string) string {
	return s.PublicURL() + "/uploads/returns/" + customerID + "/"
}

// VideoTranscoder re-encodes videos browsers cannot play (HEVC) to H.264.
type VideoTranscoder interface {
	EnsureWebPlayable(ctx context.Context, data []byte, ext string) (out []byte, outExt, contentType string, converted bool, err error)
}

// WithVideoTranscoder enables conversion; without it videos are stored as sent.
func (s *Service) WithVideoTranscoder(t VideoTranscoder) *Service {
	s.transcoder = t
	return s
}

func (s *Service) UploadEvidence(ctx context.Context, customerID string, data []byte) (EvidenceResult, error) {
	if s.store == nil {
		return EvidenceResult{}, ErrNotConfigured
	}
	if customerID == "" {
		return EvidenceResult{}, Invalid{"customer", "Cần đăng nhập để gửi minh chứng."}
	}
	if len(data) == 0 {
		return EvidenceResult{}, Invalid{"file", "File trống."}
	}
	kind, contentType, ext, err := InspectEvidence(data)
	if err != nil {
		return EvidenceResult{}, err
	}
	if kind == "video" && s.transcoder != nil {
		out, outExt, outType, converted, err := s.transcoder.EnsureWebPlayable(ctx, data, ext)
		switch {
		case err != nil:
			// Keep the original: evidence matters more than playback; the admin page offers a download.
			if s.OnTranscodeError != nil {
				s.OnTranscodeError(err)
			}
		case converted:
			data, ext, contentType = out, outExt, outType
		}
	}
	at := s.now().UTC()
	key := fmt.Sprintf("uploads/returns/%s/%04d/%02d/%s.%s", customerID, at.Year(), int(at.Month()), s.newID(), ext)
	if err := s.store.Put(ctx, key, contentType, CacheControl, bytes.NewReader(data), int64(len(data))); err != nil {
		return EvidenceResult{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return EvidenceResult{URL: s.store.PublicURL() + "/" + key, Key: key, Kind: kind, ContentType: contentType, Size: len(data)}, nil
}
