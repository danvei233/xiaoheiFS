package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMainGeneratesDocs(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	main()

	if _, err := os.Stat(filepath.Join(dir, "docs", "openapi.yaml")); err != nil {
		t.Fatalf("openapi missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "docs", "api.md")); err != nil {
		t.Fatalf("api.md missing: %v", err)
	}
}
