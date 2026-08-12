package plugins

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/pkg/pluginsdk"
	"xiaoheiplay/pkg/pluginsign"
	pluginv1 "xiaoheiplay/plugin/v1"
)

// --- fake plugin client infrastructure -------------------------------------
//
// Runtime.Start normally spawns a real go-plugin process. For tests we inject a
// fake client factory (via Runtime.newClient) that counts spawns, captures the
// ClientConfig and serves the RPC methods Runtime exercises.

type fakeClientFactory struct {
	mu             sync.Mutex
	handles        []*fakeClientHandle
	configs        []*plugin.ClientConfig
	delay          time.Duration // delay applied to GetManifest to widen race windows
	withSMS        bool
	startExited    bool // new handles report Exited()=true
	startHealthErr error
	smsSendFn      func(ctx context.Context, in *pluginv1.SendSmsRequest) (*pluginv1.SendSmsResponse, error)
}

func (f *fakeClientFactory) new(cfg *plugin.ClientConfig) clientHandle {
	f.mu.Lock()
	defer f.mu.Unlock()
	core := &fakeCoreClient{
		manifest: &pluginv1.Manifest{PluginId: "demo", Name: "Demo Plugin", Version: "1.0.0"},
		delay:    f.delay,
	}
	if f.withSMS {
		core.manifest.Sms = &pluginv1.SmsCapability{Send: true}
	}
	if f.startHealthErr != nil {
		core.setHealthErr(f.startHealthErr)
	}
	h := &fakeClientHandle{
		factory: f,
		cfg:     cfg,
		core:    core,
		sms:     &fakeSMSClient{sendFn: f.smsSendFn},
	}
	h.exited.Store(f.startExited)
	f.handles = append(f.handles, h)
	f.configs = append(f.configs, cfg)
	return h
}

func (f *fakeClientFactory) spawnCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.handles)
}

type fakeClientHandle struct {
	factory  *fakeClientFactory
	cfg      *plugin.ClientConfig
	core     *fakeCoreClient
	sms      *fakeSMSClient
	killed   bool
	exited   atomic.Bool
	exitedFn func() bool
}

func (h *fakeClientHandle) Client() (plugin.ClientProtocol, error) {
	return &fakeProtocol{core: h.core, sms: h.sms}, nil
}

func (h *fakeClientHandle) Kill() {
	h.factory.mu.Lock()
	h.killed = true
	h.factory.mu.Unlock()
}

func (h *fakeClientHandle) Exited() bool {
	if h.exitedFn != nil {
		return h.exitedFn()
	}
	return h.exited.Load()
}

type fakeProtocol struct {
	core *fakeCoreClient
	sms  *fakeSMSClient
}

func (p *fakeProtocol) Dispense(name string) (interface{}, error) {
	switch name {
	case pluginsdk.PluginKeyCore:
		return p.core, nil
	case pluginsdk.PluginKeySMS:
		if p.sms != nil {
			return p.sms, nil
		}
	}
	return nil, fmt.Errorf("plugin %q not dispensed", name)
}

func (p *fakeProtocol) Ping() error  { return nil }
func (p *fakeProtocol) Close() error { return nil }

type fakeCoreClient struct {
	mu        sync.Mutex
	manifest  *pluginv1.Manifest
	delay     time.Duration
	initErr   error
	healthErr error
	healthFn  func() (*pluginv1.HealthCheckResponse, error)
	reloadFn  func(*pluginv1.ReloadConfigRequest) (*pluginv1.ReloadConfigResponse, error)
}

func (c *fakeCoreClient) setHealthErr(err error) {
	c.mu.Lock()
	c.healthErr = err
	c.mu.Unlock()
}

func (c *fakeCoreClient) GetManifest(ctx context.Context, _ *pluginv1.Empty, _ ...grpc.CallOption) (*pluginv1.Manifest, error) {
	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return c.manifest, nil
}

func (c *fakeCoreClient) Init(_ context.Context, _ *pluginv1.InitRequest, _ ...grpc.CallOption) (*pluginv1.InitResponse, error) {
	if c.initErr != nil {
		return nil, c.initErr
	}
	return &pluginv1.InitResponse{Ok: true}, nil
}

func (c *fakeCoreClient) Health(_ context.Context, _ *pluginv1.HealthCheckRequest, _ ...grpc.CallOption) (*pluginv1.HealthCheckResponse, error) {
	c.mu.Lock()
	healthErr := c.healthErr
	healthFn := c.healthFn
	c.mu.Unlock()
	if healthFn != nil {
		return healthFn()
	}
	if healthErr != nil {
		return nil, healthErr
	}
	return &pluginv1.HealthCheckResponse{Status: pluginv1.HealthStatus_HEALTH_STATUS_OK, Message: "ok"}, nil
}

func (c *fakeCoreClient) ReloadConfig(_ context.Context, in *pluginv1.ReloadConfigRequest, _ ...grpc.CallOption) (*pluginv1.ReloadConfigResponse, error) {
	c.mu.Lock()
	reloadFn := c.reloadFn
	c.mu.Unlock()
	if reloadFn != nil {
		return reloadFn(in)
	}
	return &pluginv1.ReloadConfigResponse{Ok: true}, nil
}

func (c *fakeCoreClient) GetConfigSchema(context.Context, *pluginv1.Empty, ...grpc.CallOption) (*pluginv1.ConfigSchema, error) {
	return &pluginv1.ConfigSchema{}, nil
}

func (c *fakeCoreClient) ValidateConfig(context.Context, *pluginv1.ValidateConfigRequest, ...grpc.CallOption) (*pluginv1.ValidateConfigResponse, error) {
	return &pluginv1.ValidateConfigResponse{Ok: true}, nil
}

type fakeSMSClient struct {
	sendFn func(ctx context.Context, in *pluginv1.SendSmsRequest) (*pluginv1.SendSmsResponse, error)
}

func (c *fakeSMSClient) Send(ctx context.Context, in *pluginv1.SendSmsRequest, _ ...grpc.CallOption) (*pluginv1.SendSmsResponse, error) {
	if c.sendFn != nil {
		return c.sendFn(ctx, in)
	}
	return &pluginv1.SendSmsResponse{Ok: true, MessageId: "msg-1"}, nil
}

// writeTestPlugin lays down a minimal on-disk plugin (manifest.json + entry
// binary) so ReadManifest/ResolveEntry succeed before the fake client is used.
func writeTestPlugin(t *testing.T, baseDir, category, pluginID string, withSMS bool) {
	t.Helper()
	dir := filepath.Join(baseDir, category, pluginID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	capabilities := "{}"
	if withSMS {
		capabilities = `{"sms":{"send":true}}`
	}
	manifestJSON := fmt.Sprintf(`{"plugin_id":%q,"name":"Demo Plugin","version":"1.0.0","capabilities":%s}`, pluginID, capabilities)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write plugin binary: %v", err)
	}
}

// --- F1: concurrent Starts must not spawn duplicate plugin processes --------

func TestStartConcurrentDeduplicatesProcesses(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{delay: 150 * time.Millisecond}
	r := NewRuntime(base)
	r.newClient = factory.new
	t.Cleanup(func() { r.Stop("sms", "demo", "default") })

	ctx := context.Background()
	const n = 16
	var wg sync.WaitGroup
	manifests := make([]*pluginv1.Manifest, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			manifests[i], errs[i] = r.Start(ctx, "sms", "demo", "default", `{}`)
		}(i)
	}
	wg.Wait()

	for i := range errs {
		if errs[i] != nil {
			t.Fatalf("start %d failed: %v", i, errs[i])
		}
	}
	if got := factory.spawnCount(); got != 1 {
		t.Fatalf("expected exactly 1 plugin process spawned, got %d", got)
	}
	for i := 1; i < n; i++ {
		if manifests[i].GetPluginId() != manifests[0].GetPluginId() {
			t.Fatalf("concurrent starts returned inconsistent manifests")
		}
	}
	if _, ok := r.GetRunning("sms", "demo", "default"); !ok {
		t.Fatal("expected a running entry after start")
	}
}

func TestStartAndStopSerializeForSameKey(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{delay: 150 * time.Millisecond}
	r := NewRuntime(base)
	r.newClient = factory.new

	ctx := context.Background()
	started := make(chan struct{})
	startDone := make(chan struct{})
	go func() {
		defer close(startDone)
		_, _ = r.Start(ctx, "sms", "demo", "default", `{}`)
		close(started)
	}()

	// Give Start a head start so it is mid-spawn (holding the per-key lock)
	// before we issue a concurrent Stop.
	time.Sleep(30 * time.Millisecond)
	r.Stop("sms", "demo", "default")
	<-started
	<-startDone

	// The in-flight Start must not leave an entry behind after Stop ran.
	if _, ok := r.GetRunning("sms", "demo", "default"); ok {
		t.Fatal("plugin still running after a Stop raced with an in-flight Start")
	}
}

// --- F2: a dead plugin must be dropped/restarted, not returned forever ------

func TestStartRestartsDeadPlugin(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{}
	r := NewRuntime(base)
	r.newClient = factory.new
	r.healthRestartBackoff = 40 * time.Millisecond

	ctx := context.Background()
	if _, err := r.Start(ctx, "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("first start: %v", err)
	}
	if got := factory.spawnCount(); got != 1 {
		t.Fatalf("want 1 spawn after first start, got %d", got)
	}

	rp, ok := r.GetRunning("sms", "demo", "default")
	if !ok {
		t.Fatal("expected running entry")
	}

	// Simulate the plugin process dying underneath the runtime.
	factory.handles[0].exited.Store(true)

	// The immediate respawn is within the cooldown window and must be refused
	// (the dead-client tear-down is recorded in lastKill just like a health-loop
	// kill), so a crash-looping plugin cannot be respawned on every caller.
	if _, err := r.Start(ctx, "sms", "demo", "default", `{}`); err == nil {
		t.Fatal("expected the dead-client restart to be refused by the cooldown")
	}
	if got := factory.spawnCount(); got != 1 {
		t.Fatalf("dead-client restart spawned during cooldown: %d spawns", got)
	}

	// After the cooldown elapses, a fresh process is spawned and replaces the
	// stale entry.
	time.Sleep(60 * time.Millisecond)
	if _, err := r.Start(ctx, "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("restart start: %v", err)
	}
	if got := factory.spawnCount(); got != 2 {
		t.Fatalf("expected a fresh spawn for the dead plugin, got %d spawns", got)
	}
	rp2, ok := r.GetRunning("sms", "demo", "default")
	if !ok {
		t.Fatal("expected running entry after restart")
	}
	if rp2 == rp {
		t.Fatal("expected a new runningPlugin after restart, got the stale one")
	}
}

// TestStartCrashLoopHonorsBackoff is the spawn-storm regression for the "dies
// before the heartbeat ever sees it" window: a client that is already exited on
// registration must not be respawned on every Start call. The first Start spawns
// one process; the immediate retry is refused by the cooldown; after the backoff
// elapses exactly one fresh process is spawned.
func TestStartCrashLoopHonorsBackoff(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{startExited: true}
	r := NewRuntime(base)
	r.newClient = factory.new
	r.healthRestartBackoff = 60 * time.Millisecond
	t.Cleanup(func() { r.Stop("sms", "demo", "default") })

	ctx := context.Background()
	if _, err := r.Start(ctx, "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("first start: %v", err)
	}
	if got := factory.spawnCount(); got != 1 {
		t.Fatalf("want 1 spawn after first start, got %d", got)
	}

	// A burst of retries within the cooldown must all refuse and must not spawn.
	for i := 0; i < 5; i++ {
		if _, err := r.Start(ctx, "sms", "demo", "default", `{}`); err == nil {
			t.Fatalf("retry %d expected a cooldown refusal", i)
		}
	}
	if got := factory.spawnCount(); got != 1 {
		t.Fatalf("crash loop spawned %d processes during the cooldown window", got)
	}

	time.Sleep(80 * time.Millisecond)
	if _, err := r.Start(ctx, "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("restart after backoff: %v", err)
	}
	if got := factory.spawnCount(); got != 2 {
		t.Fatalf("want exactly 2 spawns after the backoff, got %d", got)
	}
}

func TestHeartbeatRemovesExitedPlugin(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{
		startExited:    true, // the spawned "process" is already dead
		startHealthErr: errors.New("connection refused"),
	}
	r := NewRuntime(base)
	r.newClient = factory.new
	r.heartbeatInterval = 20 * time.Millisecond
	r.healthTimeout = 10 * time.Millisecond

	ctx := context.Background()
	if _, err := r.Start(ctx, "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("start: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := r.GetRunning("sms", "demo", "default"); !ok {
			return // removed as expected
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("dead plugin was never removed from the running map")
}

// --- F6: ClientConfig must carry an explicit StartTimeout -------------------

func TestStartSetsStartTimeout(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{}
	r := NewRuntime(base)
	r.newClient = factory.new
	t.Cleanup(func() { r.Stop("sms", "demo", "default") })

	if _, err := r.Start(context.Background(), "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(factory.configs) != 1 {
		t.Fatalf("want 1 captured config, got %d", len(factory.configs))
	}
	if factory.configs[0].StartTimeout <= 0 {
		t.Fatalf("expected StartTimeout to be set explicitly, got %v", factory.configs[0].StartTimeout)
	}
}

// --- F8: heartbeat must clear stale health when the plugin is unresponsive ---

func TestHeartbeatClearsStaleHealthWhenUnresponsive(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{}
	r := NewRuntime(base)
	r.newClient = factory.new
	r.heartbeatInterval = 20 * time.Millisecond
	r.healthTimeout = 10 * time.Millisecond

	if _, err := r.Start(context.Background(), "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("start: %v", err)
	}
	rp, _ := r.GetRunning("sms", "demo", "default")

	// First wait for a successful heartbeat so health is populated.
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		rp.mu.Lock()
		healthy := rp.health != nil && rp.health.Status == pluginv1.HealthStatus_HEALTH_STATUS_OK
		rp.mu.Unlock()
		if healthy {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	rp.mu.Lock()
	if rp.health == nil {
		rp.mu.Unlock()
		t.Fatal("precondition failed: heartbeat never populated health")
	}
	rp.mu.Unlock()

	// Now the plugin is unresponsive but still alive: stale health must clear.
	factory.handles[0].core.setHealthErr(errors.New("timeout"))
	for time.Now().Before(deadline) {
		rp.mu.Lock()
		h := rp.health
		rp.mu.Unlock()
		if h == nil {
			return // stale health cleared
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("stale health was not cleared after heartbeat failed")
}

// TestHeartbeatTerminatesUnresponsivePlugin guards the circuit breaker: a
// process that is alive but never answers Health must be killed and removed
// after the consecutive-failure threshold, so EnsureRunning stops handing
// callers a dead client.
func TestHeartbeatTerminatesUnresponsivePlugin(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{}
	r := NewRuntime(base)
	r.newClient = factory.new
	r.heartbeatInterval = 20 * time.Millisecond
	r.healthTimeout = 10 * time.Millisecond
	r.maxHealthFailures = 2

	if _, err := r.Start(context.Background(), "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("start: %v", err)
	}
	rp, _ := r.GetRunning("sms", "demo", "default")

	// Wait for one successful heartbeat so the plugin is in the healthy state.
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		rp.mu.Lock()
		healthy := rp.health != nil
		rp.mu.Unlock()
		if healthy {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Plugin becomes unresponsive but stays alive.
	factory.handles[0].core.setHealthErr(errors.New("timeout"))

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := r.GetRunning("sms", "demo", "default"); !ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, ok := r.GetRunning("sms", "demo", "default"); ok {
		t.Fatal("unresponsive plugin was never removed from the running set")
	}
	factory.mu.Lock()
	killed := factory.handles[0].killed
	factory.mu.Unlock()
	if !killed {
		t.Fatal("unresponsive plugin client was not killed")
	}
}

// TestStartRespectsRestartBackoff guards the anti-storm cooldown: immediately
// after a health-triggered kill, Start refuses for the backoff window; after it
// elapses, a fresh process is spawned.
func TestStartRespectsRestartBackoff(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{}
	r := NewRuntime(base)
	r.newClient = factory.new
	r.heartbeatInterval = 20 * time.Millisecond
	r.healthTimeout = 10 * time.Millisecond
	r.maxHealthFailures = 1
	r.healthRestartBackoff = 250 * time.Millisecond

	if _, err := r.Start(context.Background(), "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("start: %v", err)
	}
	rp, _ := r.GetRunning("sms", "demo", "default")

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		rp.mu.Lock()
		healthy := rp.health != nil
		rp.mu.Unlock()
		if healthy {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Trigger one failure -> immediate termination (maxHealthFailures=1).
	factory.handles[0].core.setHealthErr(errors.New("timeout"))
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := r.GetRunning("sms", "demo", "default"); !ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, ok := r.GetRunning("sms", "demo", "default"); ok {
		t.Fatal("plugin was not terminated after the health failure")
	}

	// Within the cooldown, Start must refuse instead of spawning a new process.
	if _, err := r.Start(context.Background(), "sms", "demo", "default", `{}`); err == nil {
		t.Fatal("expected a cooling-down refusal right after termination")
	}
	if got := factory.spawnCount(); got != 1 {
		t.Fatalf("start during cooldown must not spawn, got %d spawns", got)
	}

	// After the backoff, Start spawns a fresh instance.
	time.Sleep(r.healthRestartBackoff + 50*time.Millisecond)
	if _, err := r.Start(context.Background(), "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("start after cooldown: %v", err)
	}
	if got := factory.spawnCount(); got != 2 {
		t.Fatalf("expected a fresh spawn after cooldown, got %d spawns", got)
	}
}

// TestStopClearsRestartBackoff guards the explicit-stop path: an operator who
// stops a plugin (disable/restart) must be able to start it again immediately,
// without inheriting a stale health-triggered cooldown.
func TestStopClearsRestartBackoff(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{}
	r := NewRuntime(base)
	r.newClient = factory.new
	r.healthRestartBackoff = time.Hour // a huge cooldown

	// Plant a fake last-kill so a fresh Start would otherwise be refused.
	r.mu.Lock()
	r.lastKill["sms:demo:default"] = time.Now()
	r.mu.Unlock()

	r.Stop("sms", "demo", "default")

	if _, err := r.Start(context.Background(), "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("start after explicit stop must not honor a stale cooldown: %v", err)
	}
}

// TestStartStopStopAllRace exercises Start/Stop/StopAll concurrently while the
// heartbeat loop is firing. It is a race-detector test: it must complete
// without a data race or deadlock.
func TestStartStopStopAllRace(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{}
	r := NewRuntime(base)
	r.newClient = factory.new
	r.heartbeatInterval = 5 * time.Millisecond
	r.healthTimeout = 5 * time.Millisecond

	ctx := context.Background()
	for i := 0; i < 20; i++ {
		_, _ = r.Start(ctx, "sms", "demo", "default", `{}`)
		var wg sync.WaitGroup
		wg.Add(3)
		go func() { defer wg.Done(); r.Stop("sms", "demo", "default") }()
		go func() { defer wg.Done(); r.StopAll() }()
		go func() { defer wg.Done(); _, _ = r.Start(ctx, "sms", "demo", "default", `{}`) }()
		wg.Wait()
	}
	// A final start must leave a consistent running entry.
	if _, err := r.Start(ctx, "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("final start: %v", err)
	}
	if _, ok := r.GetRunning("sms", "demo", "default"); !ok {
		t.Fatal("expected a running entry after the race")
	}
}

// TestStopAllKillsEveryRunningPlugin guards P2-58 (plugin part): StopAll must
// tear down every running plugin client so child processes are not orphaned on
// application shutdown.
func TestStopAllKillsEveryRunningPlugin(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	writeTestPlugin(t, base, "kyc", "demo", false)
	factory := &fakeClientFactory{}
	r := NewRuntime(base)
	r.newClient = factory.new

	ctx := context.Background()
	if _, err := r.Start(ctx, "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("start demo: %v", err)
	}
	if _, err := r.Start(ctx, "kyc", "demo", "default", `{}`); err != nil {
		t.Fatalf("start kyc demo: %v", err)
	}
	if got := factory.spawnCount(); got != 2 {
		t.Fatalf("want 2 spawned, got %d", got)
	}

	r.StopAll()

	if _, ok := r.GetRunning("sms", "demo", "default"); ok {
		t.Fatal("demo plugin still running after StopAll")
	}
	if _, ok := r.GetRunning("kyc", "demo", "default"); ok {
		t.Fatal("kyc demo plugin still running after StopAll")
	}
	factory.mu.Lock()
	defer factory.mu.Unlock()
	for _, h := range factory.handles {
		if !h.killed {
			t.Fatal("a plugin client was not killed by StopAll")
		}
	}
}

// TestStartSetsBoundedGRPCDialOptions verifies the host gRPC client is bounded
// (m040 M-1 host-side residual): a plugin that does not use the SDK cannot send
// an unbounded response that OOMs the host.
func TestStartSetsBoundedGRPCDialOptions(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{}
	r := NewRuntime(base)
	r.newClient = factory.new

	ctx := context.Background()
	if _, err := r.Start(ctx, "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("start: %v", err)
	}

	factory.mu.Lock()
	defer factory.mu.Unlock()
	if len(factory.configs) == 0 {
		t.Fatal("no client config captured")
	}
	cfg := factory.configs[0]
	if len(cfg.GRPCDialOptions) == 0 {
		t.Fatal("expected bounded GRPCDialOptions on the host client")
	}
}

// TestStartRequestsAutoMTLSAndPinsSocket guards the P2-51 plugin transport
// invariants for the runtime start path: the host client must request mutual
// TLS (peer authentication) and pin its control sockets to a dedicated 0700
// directory, never the shared TMPDIR.
func TestStartRequestsAutoMTLSAndPinsSocket(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", false)
	factory := &fakeClientFactory{}
	r := NewRuntime(base)
	r.newClient = factory.new

	ctx := context.Background()
	if _, err := r.Start(ctx, "sms", "demo", "default", `{}`); err != nil {
		t.Fatalf("start: %v", err)
	}
	r.StopAll()

	factory.mu.Lock()
	defer factory.mu.Unlock()
	if len(factory.configs) == 0 {
		t.Fatal("no client config captured")
	}
	cfg := factory.configs[0]
	if !cfg.AutoMTLS {
		t.Fatal("runtime start must request AutoMTLS for plugin peer authentication")
	}
	if cfg.UnixSocketConfig == nil {
		t.Fatal("runtime start must pin its unix socket directory")
	}
	socketDir := filepath.Join(base, socketRootName)
	info, err := os.Stat(socketDir)
	if err != nil {
		t.Fatalf("pinned socket dir missing: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("socket dir perms = %o, want 0700", info.Mode().Perm())
	}
}

// TestDialCoreRequestsAutoMTLSAndPinsSocket guards the same P2-51 invariants on
// the config/schema execution path (dialCore), which spawns plugin code before
// any config page is opened.
func TestDialCoreRequestsAutoMTLSAndPinsSocket(t *testing.T) {
	base := t.TempDir()
	pluginDir := filepath.Join(base, "sms", "demo")
	if err := os.MkdirAll(filepath.Join(pluginDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo Plugin","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("#!/bin/sh\noriginal\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "sms",
		PluginID:        "demo",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   digest,
	}}
	factory := &fakeClientFactory{}
	m := NewManager(base, repo, newTestCipher(t), nil)
	m.newClient = factory.new

	client, core, _, err := m.dialCore(context.Background(), "sms", "demo", "default")
	if err != nil {
		t.Fatalf("dialCore: %v", err)
	}
	if core == nil {
		t.Fatal("expected a core client")
	}
	client.Kill()

	factory.mu.Lock()
	defer factory.mu.Unlock()
	if len(factory.configs) == 0 {
		t.Fatal("no client config captured")
	}
	cfg := factory.configs[0]
	if !cfg.AutoMTLS {
		t.Fatal("dialCore must request AutoMTLS for plugin peer authentication")
	}
	if cfg.UnixSocketConfig == nil {
		t.Fatal("dialCore must pin its unix socket directory")
	}
	socketDir := filepath.Join(base, socketRootName)
	info, err := os.Stat(socketDir)
	if err != nil {
		t.Fatalf("pinned socket dir missing: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("socket dir perms = %o, want 0700", info.Mode().Perm())
	}
}
