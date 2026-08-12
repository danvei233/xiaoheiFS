package pluginsdk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/go-plugin/runner"

	pluginv1 "xiaoheiplay/plugin/v1"
)

// TestPythonHostAutoMTLSE2E launches the shipped Python example through the
// same go-plugin client configuration used by production. This catches drift
// in the cookie environment, zero-valued port variables, private Unix socket
// directory, AutoMTLS handshake certificate, and generated protobuf services.
func TestPythonHostAutoMTLSE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Python host end-to-end test in short mode")
	}

	python, err := exec.LookPath("python3")
	if err != nil {
		requirePythonHostE2E(t, "python3 is unavailable")
		return
	}
	check := exec.Command(python, "-c", "import cryptography, grpc")
	check.Env = os.Environ()
	if output, checkErr := check.CombinedOutput(); checkErr != nil {
		requirePythonHostE2E(t, "Python SDK dependencies are unavailable: "+strings.TrimSpace(string(output)))
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve Python SDK path")
	}
	backendRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	pythonSDK := filepath.Join(backendRoot, "plugin-sdk", "python")
	pythonPath := []string{filepath.Join(pythonSDK, "src"), pythonSDK}
	if inherited := os.Getenv("PYTHONPATH"); inherited != "" {
		pythonPath = append(pythonPath, inherited)
	}
	t.Setenv("PYTHONPATH", strings.Join(pythonPath, string(os.PathListSeparator)))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			PluginKeyCore: &CoreGRPCPlugin{},
		},
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		AutoMTLS:         true,
		UnixSocketConfig: &plugin.UnixSocketConfig{TempDir: t.TempDir()},
		RunnerFunc: func(_ hclog.Logger, command *exec.Cmd, socketDir string) (runner.Runner, error) {
			command.Path = python
			command.Args = []string{python, "-m", "xiaohei_plugin_sdk", "--plugin-impl", "examples.echo_plugin:build_plugin"}
			command.Env = setEnv(command.Env, "PYTHONPATH", strings.Join(pythonPath, string(os.PathListSeparator)))
			command.Env = setEnv(command.Env, "PLUGIN_UNIX_SOCKET_DIR", socketDir)
			command.Stdin = nil
			return newPythonHostRunner(command)
		},
	})
	t.Cleanup(client.Kill)

	protocolClient, err := client.Client()
	if err != nil {
		t.Fatalf("start Python plugin through Go host: %v", err)
	}
	raw, err := protocolClient.Dispense(PluginKeyCore)
	if err != nil {
		t.Fatalf("dispense Python Core service: %v", err)
	}
	core, ok := raw.(pluginv1.CoreServiceClient)
	if !ok {
		t.Fatalf("unexpected Core service type %T", raw)
	}
	manifest, err := core.GetManifest(ctx, &pluginv1.Empty{})
	if err != nil {
		t.Fatalf("Core.GetManifest over AutoMTLS Unix socket: %v", err)
	}
	if manifest.GetPluginId() != "python_echo" {
		t.Fatalf("unexpected Python manifest plugin id %q", manifest.GetPluginId())
	}
	if reattach := client.ReattachConfig(); reattach == nil || reattach.Addr == nil || reattach.Addr.Network() != "unix" {
		t.Fatalf("Python plugin must negotiate the host's private Unix socket, got %#v", reattach)
	}

	// Exercise plugin.GRPCController/Shutdown as part of the contract. Without
	// it go-plugin waits two seconds and force-kills the Python process.
	client.Kill()
}

type pythonHostRunner struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func newPythonHostRunner(cmd *exec.Cmd) (*pythonHostRunner, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	return &pythonHostRunner{cmd: cmd, stdout: stdout, stderr: stderr}, nil
}

func (r *pythonHostRunner) Start(context.Context) error { return r.cmd.Start() }
func (r *pythonHostRunner) Wait(context.Context) error  { return r.cmd.Wait() }
func (r *pythonHostRunner) Kill(context.Context) error {
	if r.cmd.Process == nil {
		return nil
	}
	err := r.cmd.Process.Kill()
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	return err
}
func (r *pythonHostRunner) Stdout() io.ReadCloser { return r.stdout }
func (r *pythonHostRunner) Stderr() io.ReadCloser { return r.stderr }
func (r *pythonHostRunner) Name() string          { return r.cmd.Path }
func (r *pythonHostRunner) ID() string {
	if r.cmd.Process == nil {
		return ""
	}
	return fmt.Sprintf("%d", r.cmd.Process.Pid)
}
func (r *pythonHostRunner) Diagnose(context.Context) string { return "python host runner" }
func (r *pythonHostRunner) PluginToHost(network, address string) (string, string, error) {
	return network, address, nil
}
func (r *pythonHostRunner) HostToPlugin(network, address string) (string, string, error) {
	return network, address, nil
}

func requirePythonHostE2E(t *testing.T, reason string) {
	t.Helper()
	if os.Getenv("XIAOHEI_PYTHON_HOST_E2E") == "required" {
		t.Fatal(reason)
	}
	t.Skip(reason)
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, item := range env {
		if !strings.HasPrefix(item, prefix) {
			out = append(out, item)
		}
	}
	return append(out, prefix+value)
}
