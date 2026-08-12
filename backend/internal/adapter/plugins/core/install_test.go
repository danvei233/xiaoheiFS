package plugins

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"xiaoheiplay/internal/domain"
)

func demoManifestJSON(platform string) string {
	return `{"plugin_id":"demo","name":"Demo","version":"1.0.0","binaries":{"` + platform + `":"bin/` + platform + `/plugin"}}`
}

func buildTestZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestInstallPackageConcurrentSamePlugin verifies that concurrent installs of
// the same plugin do not corrupt the destination: exactly one succeeds and the
// final directory is complete.
func TestInstallPackageConcurrentSamePlugin(t *testing.T) {
	platform := CurrentPlatformKey()
	pkg := buildTestZip(t, map[string]string{
		"plugins/automation/demo/manifest.json":               demoManifestJSON(platform),
		"plugins/automation/demo/bin/" + platform + "/plugin": "binary-content",
	})
	baseDir := t.TempDir()

	const attempts = 2
	var wg sync.WaitGroup
	errs := make([]error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = InstallPackage(baseDir, "demo.zip", bytes.NewReader(pkg), nil, true)
		}(i)
	}
	wg.Wait()

	success := 0
	for _, err := range errs {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("expected exactly one successful install, got %d (errors: %v)", success, errs)
	}

	// The surviving install must be complete.
	if _, err := os.Stat(filepath.Join(baseDir, "automation", "demo", "bin", platform, "plugin")); err != nil {
		t.Fatalf("installed plugin binary missing: %v", err)
	}
}

// TestInstallPackageLeavesNoTempDirs guards the atomic (temp + rename) install:
// after a successful install no temporary directories may remain that a later
// bootstrap scan could pick up.
func TestInstallPackageLeavesNoTempDirs(t *testing.T) {
	platform := CurrentPlatformKey()
	pkg := buildTestZip(t, map[string]string{
		"plugins/automation/demo/manifest.json":               demoManifestJSON(platform),
		"plugins/automation/demo/bin/" + platform + "/plugin": "binary-content",
	})
	baseDir := t.TempDir()

	if _, err := InstallPackage(baseDir, "demo.zip", bytes.NewReader(pkg), nil, true); err != nil {
		t.Fatalf("install: %v", err)
	}

	catEntries, err := os.ReadDir(filepath.Join(baseDir, "automation"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range catEntries {
		if strings.Contains(e.Name(), "install") {
			t.Fatalf("unexpected leftover temp dir after install: %s", e.Name())
		}
	}
}

// TestInstallPackageVerifyDoesNotWriteToBaseDir guards the pre-commit trust gate
// (P2-14): validating an archive with commit=false must return the signature
// status without writing any files under the plugin base dir. An untrusted
// plugin can therefore satisfy a second factor BEFORE its payload is on disk.
func TestInstallPackageVerifyDoesNotWriteToBaseDir(t *testing.T) {
	platform := CurrentPlatformKey()
	pkg := buildTestZip(t, map[string]string{
		"plugins/automation/demo/manifest.json":               demoManifestJSON(platform),
		"plugins/automation/demo/bin/" + platform + "/plugin": "binary-content",
	})
	baseDir := t.TempDir()

	res, err := InstallPackage(baseDir, "demo.zip", bytes.NewReader(pkg), nil, false)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if res.SignatureStatus != domain.PluginSignatureUnsigned {
		t.Fatalf("signature status = %q, want unsigned", res.SignatureStatus)
	}
	if res.PluginDir != "" {
		t.Fatalf("verify must not set a final plugin dir, got %q", res.PluginDir)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "automation", "demo")); !os.IsNotExist(err) {
		t.Fatalf("verify wrote plugin files under baseDir: %v", err)
	}
	if _, err := os.Stat(baseDir); err != nil || !isDirEmpty(t, baseDir) {
		t.Fatalf("verify left files under baseDir: %v", err)
	}
}

// TestInstallPackageVerifyThenCommit runs the pre-commit verify followed by the
// commit against a fresh stream, mirroring the two-phase handler flow.
func TestInstallPackageVerifyThenCommit(t *testing.T) {
	platform := CurrentPlatformKey()
	pkg := buildTestZip(t, map[string]string{
		"plugins/automation/demo/manifest.json":               demoManifestJSON(platform),
		"plugins/automation/demo/bin/" + platform + "/plugin": "binary-content",
	})
	baseDir := t.TempDir()

	if _, err := InstallPackage(baseDir, "demo.zip", bytes.NewReader(pkg), nil, false); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if _, err := InstallPackage(baseDir, "demo.zip", bytes.NewReader(pkg), nil, true); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "automation", "demo", "bin", platform, "plugin")); err != nil {
		t.Fatalf("committed plugin binary missing: %v", err)
	}
}

func isDirEmpty(t *testing.T, dir string) bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return len(entries) == 0
}

// TestVerifyReturnsPackageDigest ensures the pre-commit verify pass returns a
// digest the trust confirmation can be bound to.
func TestVerifyReturnsPackageDigest(t *testing.T) {
	platform := CurrentPlatformKey()
	pkg := buildTestZip(t, map[string]string{
		"plugins/automation/demo/manifest.json":               demoManifestJSON(platform),
		"plugins/automation/demo/bin/" + platform + "/plugin": "binary-content",
	})
	baseDir := t.TempDir()

	res, err := InstallPackage(baseDir, "demo.zip", bytes.NewReader(pkg), nil, false)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(res.Digest) != 64 {
		t.Fatalf("expected a 64-hex-char package digest, got %q", res.Digest)
	}
}

// TestInstallVerifiedRejectsSwappedPackage guards the validate-A/commit-B swap:
// the commit stream must hash to the digest the trust confirmation was granted
// for. Committing a different package with the approved digest must be refused
// before any file is written into the plugin base directory.
func TestInstallVerifiedRejectsSwappedPackage(t *testing.T) {
	platform := CurrentPlatformKey()
	good := buildTestZip(t, map[string]string{
		"plugins/automation/demo/manifest.json":               demoManifestJSON(platform),
		"plugins/automation/demo/bin/" + platform + "/plugin": "binary-content",
	})
	evil := buildTestZip(t, map[string]string{
		"plugins/automation/evil/manifest.json":               demoManifestJSON(platform),
		"plugins/automation/evil/bin/" + platform + "/plugin": "evil-content",
	})

	baseDir := t.TempDir()
	repo := &fakePluginInstallationRepo{}
	m := NewManager(baseDir, repo, testCipher(t), nil)

	// Phase 1: admin approves the good package (digest bound to the 2FA).
	preview, err := m.VerifyInstallPackage(context.Background(), "demo.zip", bytes.NewReader(good))
	if err != nil {
		t.Fatalf("verify good: %v", err)
	}

	// Phase 2: attacker swaps in a different package but reuses the approved digest.
	if _, err := m.InstallVerified(context.Background(), "evil.zip", bytes.NewReader(evil), preview.Digest); err == nil {
		t.Fatal("expected digest-mismatch refusal for the swapped package")
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "automation", "evil")); !os.IsNotExist(statErr) {
		t.Fatalf("swapped package was written under baseDir: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "automation", "demo")); !os.IsNotExist(statErr) {
		t.Fatalf("swapped package wrote under the approved plugin dir: %v", statErr)
	}

	// The approved package itself still commits cleanly.
	if _, err := m.InstallVerified(context.Background(), "demo.zip", bytes.NewReader(good), preview.Digest); err != nil {
		t.Fatalf("approved package commit failed: %v", err)
	}
}

// TestInstallPackageSurvivesLeftoverTempDir guards the crash window: a stale
// ".install-" temp dir left behind by a crash mid-copy must not block a fresh
// install (the atomic rename target is still free), and the partial dir must
// not be picked up as an installed plugin by a later bootstrap scan.
func TestInstallPackageSurvivesLeftoverTempDir(t *testing.T) {
	platform := CurrentPlatformKey()
	pkg := buildTestZip(t, map[string]string{
		"plugins/automation/demo/manifest.json":               demoManifestJSON(platform),
		"plugins/automation/demo/bin/" + platform + "/plugin": "binary-content",
	})
	baseDir := t.TempDir()

	// Simulate a crash: a leftover hidden temp dir with partial content.
	stale := filepath.Join(baseDir, "automation", ".demo.install-123")
	if err := os.MkdirAll(filepath.Join(stale, "bin", platform), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, "partial"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := InstallPackage(baseDir, "demo.zip", bytes.NewReader(pkg), nil, true); err != nil {
		t.Fatalf("install after leftover temp dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "automation", "demo", "bin", platform, "plugin")); err != nil {
		t.Fatalf("installed plugin binary missing: %v", err)
	}
	// A bootstrap scan must ignore the partial hidden dir entirely.
	found, err := scanDiskPlugins(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range found {
		if strings.Contains(filepath.Base(p.Dir), ".install-") || strings.HasPrefix(filepath.Base(p.Dir), ".") {
			t.Fatalf("crash leftover temp dir was imported as a plugin: %+v", p)
		}
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// TestInstallPackageRejectsOversizedEntry verifies zip extraction refuses
// entries whose uncompressed size exceeds the per-file limit (zip bomb guard).
func TestInstallPackageRejectsOversizedEntry(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "huge.bin", Method: zip.Deflate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.CopyN(w, zeroReader{}, maxPluginExtractPerFile+1); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = InstallPackage(t.TempDir(), "pkg.zip", bytes.NewReader(buf.Bytes()), nil, true)
	if err == nil {
		t.Fatal("expected error for oversized archive entry")
	}
	if !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestInstallPackageRejectsTooManyFiles verifies zip extraction refuses
// archives with an excessive number of entries.
func TestInstallPackageRejectsTooManyFiles(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i := 0; i < maxPluginExtractFiles+1; i++ {
		w, err := zw.Create("f" + strconv.Itoa(i))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	_, err := InstallPackage(t.TempDir(), "pkg.zip", bytes.NewReader(buf.Bytes()), nil, true)
	if err == nil {
		t.Fatal("expected error for archive with too many files")
	}
	if !strings.Contains(err.Error(), "too many") {
		t.Fatalf("unexpected error: %v", err)
	}
}
