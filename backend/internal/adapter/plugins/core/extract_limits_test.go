package plugins

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"strings"
	"testing"
)

// withExtractionLimits lowers the package extraction limits for the duration of
// one test so the limit paths can be exercised without materializing hundreds of
// megabytes. Tests in this package run sequentially (no t.Parallel), so mutating
// the package vars is safe.
func withExtractionLimits(t *testing.T, perFile, total, upload int64, files int) {
	t.Helper()
	oldPerFile, oldTotal, oldUpload, oldFiles := maxPluginExtractPerFile, maxPluginExtractBytes, maxPluginUploadBytes, maxPluginExtractFiles
	maxPluginExtractPerFile, maxPluginExtractBytes, maxPluginUploadBytes, maxPluginExtractFiles = perFile, total, upload, files
	t.Cleanup(func() {
		maxPluginExtractPerFile, maxPluginExtractBytes, maxPluginUploadBytes, maxPluginExtractFiles = oldPerFile, oldTotal, oldUpload, oldFiles
	})
}

// TestCopyBoundedCapsActualWrites proves the per-file hard limit is enforced on
// the ACTUAL bytes written, not on an archive-declared size. A reader that
// emits without bound must cause copyBounded to stop after limit bytes and
// return a size-limit error, even though the reader itself never signals EOF or
// reports its own size. This is the "declared size vs actual output" regression:
// a forged archive cannot make the extractor write past the cap before the
// reader reports a mismatch.
func TestCopyBoundedCapsActualWrites(t *testing.T) {
	var buf bytes.Buffer
	n, err := copyBounded(&buf, zeroReader{}, 1024)
	if err == nil {
		t.Fatal("expected a size-limit error from an unbounded reader")
	}
	if !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("unexpected error: %v", err)
	}
	if n > 1024 {
		t.Fatalf("copyBounded reported %d bytes written, want <= 1024", n)
	}
	if buf.Len() > 1024 {
		t.Fatalf("destination received %d bytes, want <= 1024", buf.Len())
	}
}

// TestCopyBoundedAllowsExactLimit verifies that writing exactly the limit is
// allowed — only the byte past the boundary trips.
func TestCopyBoundedAllowsExactLimit(t *testing.T) {
	var buf bytes.Buffer
	if _, err := copyBounded(&buf, &exactReader{n: 4096}, 4096); err != nil {
		t.Fatalf("copyBounded rejected an entry at the exact limit: %v", err)
	}
	if buf.Len() != 4096 {
		t.Fatalf("expected 4096 bytes, got %d", buf.Len())
	}
}

// exactReader emits exactly n zero bytes then EOF.
type exactReader struct {
	n int
}

func (r *exactReader) Read(p []byte) (int, error) {
	if r.n <= 0 {
		return 0, io.EOF
	}
	if len(p) > r.n {
		p = p[:r.n]
	}
	for i := range p {
		p[i] = 0
	}
	r.n -= len(p)
	return len(p), nil
}

// TestExtractZipRejectsDeclaredPerFileOverLimit guards the declared per-file
// fast-fail: a zip whose central directory claims a single entry above the
// per-file limit is refused before any decompression.
func TestExtractZipRejectsDeclaredPerFileOverLimit(t *testing.T) {
	withExtractionLimits(t, 100, 1000, 4096, 100)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("big.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(make([]byte, 101)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := extractZip(t.TempDir(), bytes.NewReader(buf.Bytes())); err == nil {
		t.Fatal("expected a per-file size-limit rejection")
	}
}

// TestExtractZipRejectsDeclaredTotalOverLimit guards the declared-total
// fast-fail: several entries that individually fit but whose declared sizes sum
// over the global limit are refused before their bodies are extracted.
func TestExtractZipRejectsDeclaredTotalOverLimit(t *testing.T) {
	withExtractionLimits(t, 1000, 150, 4096, 100)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	// 3 entries of 60 bytes each = 180 bytes declared > 150-byte total limit.
	for i := 0; i < 3; i++ {
		w, err := zw.Create("f" + string(rune('a'+i)) + ".bin")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(make([]byte, 60)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	err := extractZip(t.TempDir(), bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("expected a total size-limit rejection")
	}
	if !strings.Contains(err.Error(), "total size limit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestInstallPackageCleansTempOnTotalOverflow verifies that when extraction
// fails on the total size limit (the declared sum of the plugin package crosses
// the cap), installPackage's temp-root cleanup removes every partial entry and
// leaves the plugin base directory untouched — a failed install must never leave
// a partial extraction behind that a later bootstrap scan could import.
func TestInstallPackageCleansTempOnTotalOverflow(t *testing.T) {
	// Total is small enough that a manifest + binary crossing it trips the limit.
	withExtractionLimits(t, 4096, 128, 8192, 100)
	platform := CurrentPlatformKey()
	pkg := buildTestZip(t, map[string]string{
		"plugins/automation/demo/manifest.json":               demoManifestJSON(platform),
		"plugins/automation/demo/bin/" + platform + "/plugin": strings.Repeat("x", 80),
	})
	baseDir := t.TempDir()
	if _, err := InstallPackage(baseDir, "demo.zip", bytes.NewReader(pkg), nil, true); err == nil {
		t.Fatal("expected a total size-limit rejection")
	}
	entries, _ := os.ReadDir(baseDir)
	if len(entries) != 0 {
		t.Fatalf("install left %d entries under the plugin base dir after a total-limit failure", len(entries))
	}
}

// TestExtractTarGzRejectsDeclaredPerFileOverLimit guards the tar per-file
// declared fast-fail.
func TestExtractTarGzRejectsDeclaredPerFileOverLimit(t *testing.T) {
	withExtractionLimits(t, 100, 1000, 4096, 100)
	body := buildTarFile(t, "big.bin", make([]byte, 101))
	if err := extractTarGz(t.TempDir(), bytes.NewReader(body)); err == nil {
		t.Fatal("expected a per-file size-limit rejection")
	}
}

// TestExtractTarGzRejectsDeclaredTotalOverLimit guards the tar declared-total
// fast-fail with a hand-crafted archive whose headers declare sizes summing over
// the global limit.
func TestExtractTarGzRejectsDeclaredTotalOverLimit(t *testing.T) {
	withExtractionLimits(t, 1000, 150, 4096, 100)
	body := buildTarFile(t,
		"a.bin", make([]byte, 60),
		"b.bin", make([]byte, 60),
		"c.bin", make([]byte, 60),
	)
	err := extractTarGz(t.TempDir(), bytes.NewReader(body))
	if err == nil {
		t.Fatal("expected a total size-limit rejection")
	}
	if !strings.Contains(err.Error(), "total size limit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// buildTarFile packs name/body pairs into a gzipped tar.
func buildTarFile(t *testing.T, pairs ...any) []byte {
	t.Helper()
	var body bytes.Buffer
	zw := gzip.NewWriter(&body)
	tw := tar.NewWriter(zw)
	for i := 0; i < len(pairs); i += 2 {
		name := pairs[i].(string)
		data := pairs[i+1].([]byte)
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}
