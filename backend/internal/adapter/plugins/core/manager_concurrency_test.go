package plugins

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/pkg/pluginsign"
)

// mutexPluginRepo is a thread-safe in-memory PluginInstallationRepository with
// an optional per-op delay used to widen the read-modify-write window so a
// lost-update (were one present) would actually be observed.
type mutexPluginRepo struct {
	mu    sync.Mutex
	inst  domain.PluginInstallation
	delay time.Duration
}

func (r *mutexPluginRepo) GetPluginInstallation(_ context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.delay > 0 {
		time.Sleep(r.delay)
	}
	if r.inst.Category == category && r.inst.PluginID == pluginID && r.inst.InstanceID == instanceID {
		return r.inst, nil
	}
	return domain.PluginInstallation{}, errors.New("not found")
}

func (r *mutexPluginRepo) UpsertPluginInstallation(_ context.Context, inst *domain.PluginInstallation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.delay > 0 {
		time.Sleep(r.delay)
	}
	r.inst = *inst
	return nil
}

func (r *mutexPluginRepo) ListPluginInstallations(context.Context) ([]domain.PluginInstallation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return []domain.PluginInstallation{r.inst}, nil
}

func (r *mutexPluginRepo) DeletePluginInstallation(context.Context, string, string, string) error {
	return nil
}

// TestConcurrentEnableAndUpdateConfigDoesNotLoseState guards the lifecycle
// lost-update (runtime F6 / exec_race F1): EnableInstance and UpdateConfigInstance
// both read the whole plugin_installations row and upsert it back. Without a
// per-instance operation lock they can interleave so Enable's upsert overwrites
// the freshly-written config (or Update's upsert marks the instance disabled
// while its process is running). After serialization the final repository row
// must be BOTH enabled AND carry the updated config.
func TestConcurrentEnableAndUpdateConfigDoesNotLoseState(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "automation", "demo", false)
	digest, err := pluginsign.ContentDigest(filepath.Join(base, "automation", "demo"))
	if err != nil {
		t.Fatal(err)
	}
	cipher := testCipher(t)
	oldCipher, err := cipher.EncryptToString([]byte(`{"base_url":"https://old.example.com"}`))
	if err != nil {
		t.Fatal(err)
	}
	repo := &mutexPluginRepo{delay: 20 * time.Millisecond, inst: domain.PluginInstallation{
		Category:      "automation",
		PluginID:      "demo",
		InstanceID:    "default",
		Enabled:       false,
		ContentDigest: digest,
		ConfigCipher:  oldCipher,
	}}
	factory := &fakeClientFactory{}
	m := NewManager(base, repo, cipher, nil)
	m.runtime.newClient = factory.new
	t.Cleanup(func() { m.runtime.Stop("automation", "demo", "default") })

	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = m.EnableInstance(ctx, "automation", "demo", "default")
	}()
	go func() {
		defer wg.Done()
		_ = m.UpdateConfigInstance(ctx, "automation", "demo", "default", `{"base_url":"https://new.example.com"}`)
	}()
	wg.Wait()

	inst, err := repo.GetPluginInstallation(ctx, "automation", "demo", "default")
	if err != nil {
		t.Fatalf("get final installation: %v", err)
	}
	if !inst.Enabled {
		t.Fatal("enable was lost: instance is disabled after concurrent enable+update")
	}
	plain, err := m.decryptConfig(inst.ConfigCipher)
	if err != nil {
		t.Fatalf("decrypt final config: %v", err)
	}
	if !strings.Contains(plain, "https://new.example.com") {
		t.Fatalf("config update was lost by the concurrent enable: %s", plain)
	}
}
