package http

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
)

func validPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x % 255), uint8(y % 255), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// pngHeaderOnly builds a PNG with an IHDR claiming the given dimensions but no
// pixel data. image.DecodeConfig reads the header and reports the dimensions,
// which is enough to enforce the pixel cap before the pixel payload is decoded.
func pngHeaderOnly(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	b.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	chunkData := make([]byte, 13)
	binary.BigEndian.PutUint32(chunkData[0:4], uint32(w))
	binary.BigEndian.PutUint32(chunkData[4:8], uint32(h))
	chunkData[8] = 8 // bit depth
	chunkData[9] = 6 // color type RGBA
	chunkData[10] = 0
	chunkData[11] = 0
	chunkData[12] = 0
	var lenBytes [4]byte
	binary.BigEndian.PutUint32(lenBytes[:], 13)
	b.Write(lenBytes[:])
	b.WriteString("IHDR")
	b.Write(chunkData)
	crc := crc32.ChecksumIEEE(append([]byte("IHDR"), chunkData...))
	var crcBytes [4]byte
	binary.BigEndian.PutUint32(crcBytes[:], crc)
	b.Write(crcBytes[:])
	return b.Bytes()
}

func TestSanitizeImageNormalPNG(t *testing.T) {
	raw := validPNG(t, 8, 8)
	clean, mime, err := sanitizeImage(raw)
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	if mime != "image/png" {
		t.Fatalf("mime = %q, want image/png", mime)
	}
	img, _, err := image.Decode(bytes.NewReader(clean))
	if err != nil {
		t.Fatalf("decoded sanitized image: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 8 || b.Dy() != 8 {
		t.Fatalf("dimensions = %dx%d, want 8x8", b.Dx(), b.Dy())
	}
}

func TestSanitizeImageStripsTrailingPayload(t *testing.T) {
	pngBytes := validPNG(t, 8, 8)
	polyglot := append(append([]byte{}, pngBytes...), []byte("<?php system($_GET['c']); ?>")...)
	clean, _, err := sanitizeImage(polyglot)
	if err != nil {
		t.Fatalf("sanitize polyglot: %v", err)
	}
	if bytes.Contains(clean, []byte("<?php")) {
		t.Fatalf("trailing script payload survived re-encode")
	}
	if !bytes.Equal(clean, polyglot) {
		// The canonical re-encode must not include the trailing payload; the
		// stored bytes must decode as an image.
		if _, _, err := image.Decode(bytes.NewReader(clean)); err != nil {
			t.Fatalf("stored bytes are not a decodable image: %v", err)
		}
	}
}

func TestSanitizeImageRejectsDecompressionBomb(t *testing.T) {
	// 6000x5000 = 30M pixels > 25M cap.
	raw := pngHeaderOnly(t, 6000, 5000)
	if _, _, err := sanitizeImage(raw); err != domain.ErrImageTooLarge {
		t.Fatalf("expected ErrImageTooLarge, got %v", err)
	}
}

func TestSanitizeImageRejectsForgedMIME(t *testing.T) {
	// A file that is not an image at all but is presented with a PNG header
	// must be rejected (DecodeConfig cannot parse it as a real image).
	raw := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00, 0x01, 0x02, 0x03, 0x04}
	if _, _, err := sanitizeImage(raw); err != domain.ErrUnsupportedFileType {
		t.Fatalf("expected ErrUnsupportedFileType, got %v", err)
	}
}

func TestSanitizeImageRejectsCorruptedImage(t *testing.T) {
	raw := validPNG(t, 8, 8)
	// Corrupt the middle of the pixel data.
	raw[len(raw)/2] ^= 0xFF
	if _, _, err := sanitizeImage(raw); err != domain.ErrUnsupportedFileType {
		t.Fatalf("expected ErrUnsupportedFileType for corrupted image, got %v", err)
	}
}

func TestSanitizeImageRejectsUnsupportedFormat(t *testing.T) {
	// A plain text file with no image magic.
	raw := []byte("hello world this is not an image")
	if _, _, err := sanitizeImage(raw); err != domain.ErrUnsupportedFileType {
		t.Fatalf("expected ErrUnsupportedFileType, got %v", err)
	}
}

func TestSupportedImageMime(t *testing.T) {
	for _, mime := range []string{"image/png", "image/jpeg", "image/gif"} {
		if !supportedImageMime(mime) {
			t.Fatalf("expected %s to be supported", mime)
		}
	}
	if supportedImageMime("image/webp") {
		t.Fatalf("webp should not be a supported upload mime (no canonical encoder)")
	}
	if supportedImageMime("text/plain") {
		t.Fatalf("text/plain should not be supported")
	}
	if strings.ToLower("IMAGE/PNG") != "image/png" {
		t.Fatal("helper bug")
	}
}
