package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestDeleteInstanceRejectsTraversal guards P0-12: category/plugin_id values
// containing traversal components must never reach os.RemoveAll.
func TestDeleteInstanceRejectsTraversal(t *testing.T) {
	base := t.TempDir()
	// Create a victim file that a traversal delete must not be able to remove.
	victimDir := filepath.Join(t.TempDir(), "victim")
	if err := os.MkdirAll(victimDir, 0o755); err != nil {
		t.Fatal(err)
	}
	victim := filepath.Join(victimDir, "keep.txt")
	if err := os.WriteFile(victim, []byte("do-not-delete"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := &fakePluginInstallationRepo{}
	m := NewManager(base, repo, newTestCipher(t), nil)

	for _, tc := range []struct{ category, pluginID string }{
		{"..", "x"},
		{"../..", "x"},
		{"a/b", "x"},
		{".", "x"},
		{"x", "../.."},
		{"x", "../../" + filepath.Base(victimDir)},
		{"..\\..", "x"},
	} {
		err := m.DeleteInstance(context.Background(), tc.category, tc.pluginID, "default")
		if err == nil {
			t.Fatalf("DeleteInstance(%q,%q) succeeded; expected traversal refusal", tc.category, tc.pluginID)
		}
		if _, statErr := os.Stat(victim); statErr != nil {
			t.Fatalf("DeleteInstance(%q,%q) deleted a file outside the plugin base dir: %v", tc.category, tc.pluginID, statErr)
		}
	}
}

// TestDeletePluginFilesRejectsTraversal mirrors the DeleteInstance guard for the
// explicit file-deletion endpoint.
func TestDeletePluginFilesRejectsTraversal(t *testing.T) {
	base := t.TempDir()
	repo := &fakePluginInstallationRepo{}
	m := NewManager(base, repo, newTestCipher(t), nil)

	for _, tc := range []struct{ category, pluginID string }{
		{"..", "x"},
		{"a/../b", "x"},
		{"x", "../victim"},
	} {
		if err := m.DeletePluginFiles(context.Background(), tc.category, tc.pluginID); err == nil {
			t.Fatalf("DeletePluginFiles(%q,%q) succeeded; expected traversal refusal", tc.category, tc.pluginID)
		}
	}
}

// TestSafePluginDirStaysInsideBase verifies the safe-path helper only returns
// paths under baseDir.
func TestSafePluginDirStaysInsideBase(t *testing.T) {
	base := t.TempDir()
	m := NewManager(base, nil, nil, nil)

	dir, err := m.safePluginDir("automation", "demo")
	if err != nil {
		t.Fatalf("safePluginDir: %v", err)
	}
	want := filepath.Join(base, "automation", "demo")
	if dir != want {
		t.Fatalf("safePluginDir = %q, want %q", dir, want)
	}

	for _, bad := range []struct{ category, pluginID string }{
		{"..", "x"},
		{"../x", "y"},
		{"x", ".."},
		{"x", "../../etc"},
		{"a/b", "c"},
	} {
		if _, err := m.safePluginDir(bad.category, bad.pluginID); err == nil {
			t.Fatalf("safePluginDir(%q,%q) accepted a traversal path", bad.category, bad.pluginID)
		}
	}
}
