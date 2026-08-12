package plugins

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestRejectTestOnlyManifest covers the runtime backstop: a manifest that
// declares `test_only: true` (e.g. mockpay) must be refused on install and on
// launch unless an explicit dev flag allows it.
func TestRejectTestOnlyManifest(t *testing.T) {
	m := Manifest{
		PluginID: "mockpay",
		Name:     "MockPay",
		Version:  "1.0.0",
		TestOnly: true,
	}
	if err := m.RejectTestOnly(false); err == nil {
		t.Fatal("test_only manifest must be rejected by default")
	}
	if err := m.RejectTestOnly(true); err != nil {
		t.Fatalf("test_only manifest allowed by explicit dev flag: %v", err)
	}
	normal := Manifest{PluginID: "alipay", Name: "Alipay", Version: "1.0.0"}
	if err := normal.RejectTestOnly(false); err != nil {
		t.Fatalf("non-test-only manifest must pass: %v", err)
	}
}

// TestInstallPackageRejectsTestOnly proves the admin upload path refuses a
// packaged test_only plugin end-to-end.
func TestInstallPackageRejectsTestOnly(t *testing.T) {
	baseDir := t.TempDir()
	// Build a minimal zip package that mirrors the mockpay manifest shape.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("payment/mockpay/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(`{
		"plugin_id": "mockpay",
		"name": "MockPay Disk Plugin",
		"version": "1.0.0",
		"test_only": true,
		"binaries": {"linux_amd64": "bin/linux_amd64/plugin"},
		"capabilities": {"payment": {"methods": ["mock"]}}
	}`)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = InstallPackage(baseDir, "mockpay.zip", bytes.NewReader(buf.Bytes()), nil, false)
	if err == nil {
		t.Fatal("InstallPackage must reject a test_only package")
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "payment", "mockpay")); !os.IsNotExist(statErr) {
		t.Fatal("test_only plugin directory must not be left on disk")
	}
}
