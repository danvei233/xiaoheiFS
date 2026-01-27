package payment

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRegistry_PluginDirScan(t *testing.T) {
	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "demo.exe")
	if err := os.WriteFile(pluginPath, []byte("demo"), 0o755); err != nil {
		t.Fatalf("write plugin: %v", err)
	}

	specs, err := scanPluginDir(dir)
	if err != nil {
		t.Fatalf("scan plugin dir: %v", err)
	}
	if len(specs) == 0 {
		t.Fatalf("expected plugin specs")
	}
	if pluginSpecHash(specs) == "" {
		t.Fatalf("expected plugin spec hash")
	}

	reg := NewRegistry(nil)
	reg.SetPluginDir(dir)
	ctx, cancel := context.WithCancel(context.Background())
	if err := reg.StartWatcher(ctx, dir); err != nil {
		cancel()
		t.Fatalf("start watcher: %v", err)
	}
	cancel()
	time.Sleep(10 * time.Millisecond)

	if err := reg.refreshDirSpecs(); err != nil {
		t.Fatalf("refresh dir specs: %v", err)
	}
	if _, err := reg.ensurePlugins(nil); err != nil {
		t.Fatalf("ensure plugins: %v", err)
	}
}
