package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func writeManifestFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestReadManifestRejectsUnknownField ensures unexpected manifest fields are
// not silently ignored, so a tampered/foreign manifest is caught at parse time.
func TestReadManifestRejectsUnknownField(t *testing.T) {
	dir := writeManifestFile(t, `{"plugin_id":"demo","name":"Demo","version":"1.0.0","evil_field":"x"}`)
	if _, err := ReadManifest(dir); err == nil {
		t.Fatal("expected error for unknown manifest field")
	}
}

// TestReadManifestRejectsUnknownCapabilityField ensures unexpected fields
// nested under capabilities are rejected too.
func TestReadManifestRejectsUnknownCapabilityField(t *testing.T) {
	dir := writeManifestFile(t, `{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{"payment":{"methods":["x"],"evil":true}}}`)
	if _, err := ReadManifest(dir); err == nil {
		t.Fatal("expected error for unknown capability field")
	}
}

// TestReadManifestRejectsInvalidPlatformKey ensures platform keys in binaries
// cannot smuggle path separators or traversal components.
func TestReadManifestRejectsInvalidPlatformKey(t *testing.T) {
	dir := writeManifestFile(t, `{"plugin_id":"demo","name":"Demo","version":"1.0.0","binaries":{"a/b":"bin/a/b/plugin"}}`)
	if _, err := ReadManifest(dir); err == nil {
		t.Fatal("expected error for invalid platform key")
	}
}

// TestReadManifestRejectsTrailingContent ensures content after the top-level
// manifest object is rejected (previously enforced by json.Unmarshal).
func TestReadManifestRejectsTrailingContent(t *testing.T) {
	dir := writeManifestFile(t, `{"plugin_id":"demo","name":"Demo","version":"1.0.0"} trailing`)
	if _, err := ReadManifest(dir); err == nil {
		t.Fatal("expected error for trailing content")
	}
}
