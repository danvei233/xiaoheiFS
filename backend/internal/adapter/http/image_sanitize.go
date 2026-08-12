package http

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"

	"xiaoheiplay/internal/domain"
)

// maxImagePixels caps the total pixel count of an uploaded image so a
// decompression bomb (a tiny file that expands to an enormous bitmap) cannot
// exhaust server memory during decode.
const maxImagePixels int64 = 25_000_000

// sanitizeImage fully decodes an uploaded image and re-encodes it as a
// canonical image of the same format. This discards dangerous metadata and any
// trailing payload (a polyglot that appends a script after a valid image), so
// only the clean re-encoded bytes are stored. It returns the canonical bytes
// and the canonical MIME type.
//
// GIF animation is preserved via DecodeAll/EncodeAll. WebP is not supported by
// the standard-library encoders and is rejected.
func sanitizeImage(raw []byte) ([]byte, string, error) {
	// DecodeConfig reads only the header, so the pixel cap is enforced before
	// the (potentially huge) pixel payload is decoded.
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, "", domain.ErrUnsupportedFileType
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return nil, "", domain.ErrImageTooLarge
	}

	switch format {
	case "gif":
		g, err := gif.DecodeAll(bytes.NewReader(raw))
		if err != nil {
			return nil, "", domain.ErrUnsupportedFileType
		}
		// Cap each animation frame too, not just the first.
		for _, f := range g.Image {
			b := f.Bounds()
			if int64(b.Dx())*int64(b.Dy()) > maxImagePixels {
				return nil, "", domain.ErrImageTooLarge
			}
		}
		var buf bytes.Buffer
		if err := gif.EncodeAll(&buf, g); err != nil {
			return nil, "", domain.ErrUnsupportedFileType
		}
		return buf.Bytes(), "image/gif", nil
	case "jpeg":
		img, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil, "", domain.ErrUnsupportedFileType
		}
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
			return nil, "", domain.ErrUnsupportedFileType
		}
		return buf.Bytes(), "image/jpeg", nil
	case "png":
		img, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil, "", domain.ErrUnsupportedFileType
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", domain.ErrUnsupportedFileType
		}
		return buf.Bytes(), "image/png", nil
	default:
		return nil, "", domain.ErrUnsupportedFileType
	}
}

// supportedImageMime reports whether the detected MIME is one we can decode and
// canonically re-encode.
func supportedImageMime(mime string) bool {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/png", "image/jpeg", "image/gif":
		return true
	default:
		return false
	}
}
