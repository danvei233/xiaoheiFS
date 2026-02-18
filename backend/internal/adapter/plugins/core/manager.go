package plugins

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/cryptox"

	"github.com/hashicorp/go-plugin"

	"fmt"
	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type Manager struct {
	baseDir      string
	officialKeys []ed25519.PublicKey
	cipher       *cryptox.AESGCM
	repo         appports.PluginInstallationRepository
	runtime      *Runtime
}

func NewManager(baseDir string, repo appports.PluginInstallationRepository, cipher *cryptox.AESGCM, officialKeys []ed25519.PublicKey) *Manager {
	return &Manager{
		baseDir:      strings.TrimSpace(baseDir),
		officialKeys: officialKeys,
		cipher:       cipher,
		repo:         repo,
		runtime:      NewRuntime(strings.TrimSpace(baseDir)),
	}
}

const DefaultInstanceID = "default"

const (
	automationCategory         = "automation"
	automationConfigJSONSchema = `{"type":"object","properties":{"base_url":{"type":"string","title":"Base URL"},"api_key":{"type":"string","title":"API Key","format":"password"},"timeout_sec":{"type":"integer","title":"Timeout (sec)","minimum":1},"retry":{"type":"integer","title":"Retry","minimum":0},"dry_run":{"type":"boolean","title":"Dry Run"}}}`
	automationConfigUISchema   = `{}`
)

func isAutomationCategory(category string) bool {
	return strings.EqualFold(strings.TrimSpace(category), automationCategory)
}

func (m *Manager) PluginDir(category, pluginID string) string {
	return filepath.Join(m.baseDir, strings.TrimSpace(category), strings.TrimSpace(pluginID))
}

func (m *Manager) EnsureRunning(ctx context.Context, category, pluginID, instanceID string) (*pluginv1.Manifest, error) {
	if m.repo == nil {
		return nil, fmt.Errorf("plugin repo missing")
	}
	category = strings.TrimSpace(category)
	pluginID = strings.TrimSpace(pluginID)
	instanceID = strings.TrimSpace(instanceID)
	if category == "" || pluginID == "" || instanceID == "" {
		return nil, fmt.Errorf("invalid input")
	}
	inst, err := m.repo.GetPluginInstallation(ctx, category, pluginID, instanceID)
	if err != nil {
		return nil, err
	}
	if !inst.Enabled {
		return nil, fmt.Errorf("plugin instance disabled")
	}
	cfg, err := m.decryptConfig(inst.ConfigCipher)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg) == "" {
		cfg = "{}"
	}
	return m.runtime.Start(ctx, category, pluginID, instanceID, cfg)
}

func (m *Manager) GetAutomationClient(ctx context.Context, pluginID, instanceID string) (pluginv1.AutomationServiceClient, *pluginv1.Manifest, error) {
	manifest, err := m.EnsureRunning(ctx, "automation", pluginID, instanceID)
	if err != nil {
		return nil, nil, err
	}
	rp, ok := m.runtime.GetRunning("automation", pluginID, instanceID)
	if !ok || rp == nil {
		return nil, nil, fmt.Errorf("plugin instance not running")
	}
	if rp.automation == nil {
		return nil, nil, fmt.Errorf("automation capability not supported")
	}
	return rp.automation, manifest, nil
}

type ListItem struct {
	Category        string                       `json:"category"`
	PluginID        string                       `json:"plugin_id"`
	InstanceID      string                       `json:"instance_id"`
	Name            string                       `json:"name"`
	Version         string                       `json:"version"`
	SignatureStatus domain.PluginSignatureStatus `json:"signature_status"`
	Enabled         bool                         `json:"enabled"`
	Loaded          bool                         `json:"loaded"`
	InstalledAt     time.Time                    `json:"installed_at"`
	UpdatedAt       time.Time                    `json:"updated_at"`
	LastHealthAt    *time.Time                   `json:"last_health_at"`
	HealthStatus    string                       `json:"health_status"`
	HealthMessage   string                       `json:"health_message"`
	Capabilities    Manifest                     `json:"manifest"`
	Entry           EntryInfo                    `json:"entry"`
}

type ConfigValidationError struct {
	Code          string
	Message       string
	MissingFields []string
	RedirectPath  string
}

func (e *ConfigValidationError) Error() string {
	if e == nil {
		return "invalid config"
	}
	msg := strings.TrimSpace(e.Message)
	if msg != "" {
		return msg
	}
	if strings.TrimSpace(e.Code) != "" {
		return e.Code
	}
	return "invalid config"
}

func AsConfigValidationError(err error) (*ConfigValidationError, bool) {
	var target *ConfigValidationError
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

func (m *Manager) List(ctx context.Context) ([]ListItem, error) {
	if m.repo == nil {
		return nil, fmt.Errorf("plugin repo missing")
	}
	installations, err := m.repo.ListPluginInstallations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(installations))
	for _, inst := range installations {
		dir := filepath.Join(m.baseDir, inst.Category, inst.PluginID)
		manifest, err := ReadManifest(dir)
		if err != nil {
			continue
		}
		entry, _ := ResolveEntry(dir, manifest)
		var lastHealthAt *time.Time
		healthStatus := ""
		healthMessage := ""
		loaded := false
		if rp, ok := m.runtime.GetRunning(inst.Category, inst.PluginID, inst.InstanceID); ok {
			loaded = true
			rp.mu.Lock()
			if !rp.lastHealth.IsZero() {
				t := rp.lastHealth
				lastHealthAt = &t
			}
			if rp.health != nil {
				healthStatus = rp.health.Status.String()
				healthMessage = rp.health.Message
			}
			rp.mu.Unlock()
		}
		out = append(out, ListItem{
			Category:        inst.Category,
			PluginID:        inst.PluginID,
			InstanceID:      inst.InstanceID,
			Name:            manifest.Name,
			Version:         manifest.Version,
			SignatureStatus: inst.SignatureStatus,
			Enabled:         inst.Enabled,
			Loaded:          loaded,
			InstalledAt:     inst.CreatedAt,
			UpdatedAt:       inst.UpdatedAt,
			LastHealthAt:    lastHealthAt,
			HealthStatus:    healthStatus,
			HealthMessage:   healthMessage,
			Capabilities:    manifest,
			Entry:           entry,
		})
	}
	return out, nil
}

func (m *Manager) Install(ctx context.Context, filename string, r io.Reader) (domain.PluginInstallation, error) {
	if m.repo == nil {
		return domain.PluginInstallation{}, fmt.Errorf("plugin repo missing")
	}
	res, err := InstallPackage(m.baseDir, filename, r, m.officialKeys)
	if err != nil {
		return domain.PluginInstallation{}, err
	}
	inst := domain.PluginInstallation{
		Category:        res.Category,
		PluginID:        res.PluginID,
		InstanceID:      DefaultInstanceID,
		Enabled:         false,
		SignatureStatus: res.SignatureStatus,
		ConfigCipher:    "",
	}
	if err := m.repo.UpsertPluginInstallation(ctx, &inst); err != nil {
		_ = os.RemoveAll(res.PluginDir)
		return domain.PluginInstallation{}, err
	}
	return m.repo.GetPluginInstallation(ctx, res.Category, res.PluginID, DefaultInstanceID)
}

func (m *Manager) Uninstall(ctx context.Context, category, pluginID string) error {
	return m.DeleteInstance(ctx, category, pluginID, DefaultInstanceID)
}

func (m *Manager) Enable(ctx context.Context, category, pluginID string) error {
	return m.EnableInstance(ctx, category, pluginID, DefaultInstanceID)
}

func (m *Manager) Disable(ctx context.Context, category, pluginID string) error {
	return m.DisableInstance(ctx, category, pluginID, DefaultInstanceID)
}

func (m *Manager) StartEnabled(ctx context.Context) {
	if m.repo == nil {
		return
	}
	items, err := m.repo.ListPluginInstallations(ctx)
	if err != nil {
		return
	}
	for _, inst := range items {
		if !inst.Enabled {
			continue
		}
		cfg, err := m.decryptConfig(inst.ConfigCipher)
		if err != nil {
			continue
		}
		if strings.TrimSpace(cfg) == "" {
			cfg = "{}"
		}
		_, _ = m.runtime.Start(ctx, inst.Category, inst.PluginID, inst.InstanceID, cfg)
	}
}

func (m *Manager) GetConfigSchema(ctx context.Context, category, pluginID string) (jsonSchema, uiSchema string, err error) {
	return m.GetConfigSchemaInstance(ctx, category, pluginID, DefaultInstanceID)
}

func (m *Manager) GetConfigSchemaInstance(ctx context.Context, category, pluginID, _ string) (jsonSchema, uiSchema string, err error) {
	if isAutomationCategory(category) {
		return automationConfigJSONSchema, automationConfigUISchema, nil
	}
	client, core, _, err := m.dialCore(ctx, category, pluginID)
	if err != nil {
		return "", "", err
	}
	defer client.Kill()
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resp, err := core.GetConfigSchema(cctx, &pluginv1.Empty{})
	if err != nil {
		return "", "", err
	}
	return resp.GetJsonSchema(), resp.GetUiSchema(), nil
}

func (m *Manager) GetConfig(ctx context.Context, category, pluginID string) (string, error) {
	return m.GetConfigInstance(ctx, category, pluginID, DefaultInstanceID)
}

func (m *Manager) redactSecretFields(ctx context.Context, category, pluginID string, configJSON string) (string, error) {
	if isAutomationCategory(category) {
		return configJSON, nil
	}
	client, core, _, err := m.dialCore(ctx, category, pluginID)
	if err != nil {
		return "", err
	}
	defer client.Kill()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	schema, err := core.GetConfigSchema(cctx, &pluginv1.Empty{})
	if err != nil {
		return "", err
	}

	var schemaObj any
	_ = json.Unmarshal([]byte(schema.GetJsonSchema()), &schemaObj)
	secretPaths := collectSecretPaths(schemaObj, nil)
	if len(secretPaths) == 0 {
		return configJSON, nil
	}

	var obj any
	if err := json.Unmarshal([]byte(configJSON), &obj); err != nil {
		obj = map[string]any{}
	}
	for _, path := range secretPaths {
		obj = setPath(obj, path, "")
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (m *Manager) UpdateConfig(ctx context.Context, category, pluginID string, configJSON string) error {
	return m.UpdateConfigInstance(ctx, category, pluginID, DefaultInstanceID, configJSON)
}

func (m *Manager) EnableInstance(ctx context.Context, category, pluginID, instanceID string) error {
	if m.repo == nil {
		return fmt.Errorf("plugin repo missing")
	}
	inst, err := m.repo.GetPluginInstallation(ctx, category, pluginID, instanceID)
	if err != nil {
		return err
	}
	cfg, err := m.decryptConfig(inst.ConfigCipher)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg) == "" {
		cfg = "{}"
	}
	// Automation config is managed per goods-type workflow, not as a hard precondition for plugin enable.
	if !isAutomationCategory(category) {
		if err := m.validateConfig(ctx, category, pluginID, cfg); err != nil {
			return err
		}
	}
	_, err = m.runtime.Start(ctx, category, pluginID, inst.InstanceID, cfg)
	if err != nil {
		return err
	}
	inst.Enabled = true
	return m.repo.UpsertPluginInstallation(ctx, &inst)
}

func (m *Manager) DisableInstance(ctx context.Context, category, pluginID, instanceID string) error {
	if m.repo == nil {
		return fmt.Errorf("plugin repo missing")
	}
	m.runtime.Stop(category, pluginID, instanceID)
	inst, err := m.repo.GetPluginInstallation(ctx, category, pluginID, instanceID)
	if err != nil {
		return err
	}
	inst.Enabled = false
	return m.repo.UpsertPluginInstallation(ctx, &inst)
}

func (m *Manager) GetConfigInstance(ctx context.Context, category, pluginID, instanceID string) (string, error) {
	if m.repo == nil {
		return "", fmt.Errorf("plugin repo missing")
	}
	inst, err := m.repo.GetPluginInstallation(ctx, category, pluginID, instanceID)
	if err != nil {
		return "", err
	}
	cfg, err := m.decryptConfig(inst.ConfigCipher)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(cfg) == "" {
		cfg = "{}"
	}
	redacted, err := m.redactSecretFields(ctx, category, pluginID, cfg)
	if err != nil {
		return "", err
	}
	return redacted, nil
}

func (m *Manager) UpdateConfigInstance(ctx context.Context, category, pluginID, instanceID string, configJSON string) error {
	if m.repo == nil {
		return fmt.Errorf("plugin repo missing")
	}
	inst, err := m.repo.GetPluginInstallation(ctx, category, pluginID, instanceID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(configJSON) == "" {
		configJSON = "{}"
	}

	merged := configJSON
	if !isAutomationCategory(category) {
		oldCfg, err := m.decryptConfig(inst.ConfigCipher)
		if err != nil {
			return err
		}
		if strings.TrimSpace(oldCfg) == "" {
			oldCfg = "{}"
		}
		merged, err = m.mergeSecretFields(ctx, category, pluginID, oldCfg, configJSON)
		if err != nil {
			return err
		}
		if err := m.validateConfig(ctx, category, pluginID, merged); err != nil {
			return err
		}
	} else if !json.Valid([]byte(merged)) {
		return fmt.Errorf("invalid config json")
	}
	cipherText, err := m.encryptConfig(merged)
	if err != nil {
		return err
	}
	inst.ConfigCipher = cipherText
	if err := m.repo.UpsertPluginInstallation(ctx, &inst); err != nil {
		return err
	}
	if inst.Enabled {
		if rp, ok := m.runtime.GetRunning(category, pluginID, instanceID); ok && rp.core != nil {
			cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			resp, err := rp.core.ReloadConfig(cctx, &pluginv1.ReloadConfigRequest{ConfigJson: merged})
			if err != nil {
				return err
			}
			if resp != nil && !resp.Ok {
				if resp.Error != "" {
					return fmt.Errorf("%s", resp.Error)
				}
				return fmt.Errorf("plugin reload failed")
			}
		}
	}
	return nil
}

func (m *Manager) CreateInstance(ctx context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error) {
	if m.repo == nil {
		return domain.PluginInstallation{}, fmt.Errorf("plugin repo missing")
	}
	category = strings.TrimSpace(category)
	pluginID = strings.TrimSpace(pluginID)
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = newInstanceID(category, pluginID)
	}
	if category == "" || pluginID == "" || instanceID == "" {
		return domain.PluginInstallation{}, fmt.Errorf("invalid input")
	}

	pluginDir := filepath.Join(m.baseDir, category, pluginID)
	manifest, err := ReadManifest(pluginDir)
	if err != nil {
		return domain.PluginInstallation{}, err
	}
	if manifest.PluginID != pluginID {
		return domain.PluginInstallation{}, fmt.Errorf("manifest plugin_id mismatch")
	}
	if _, err := ResolveEntry(pluginDir, manifest); err != nil {
		return domain.PluginInstallation{}, err
	}
	sigStatus, err := VerifySignature(pluginDir, m.officialKeys)
	if err != nil {
		return domain.PluginInstallation{}, err
	}

	if _, err := m.repo.GetPluginInstallation(ctx, category, pluginID, instanceID); err == nil {
		return domain.PluginInstallation{}, fmt.Errorf("instance already exists")
	}

	inst := domain.PluginInstallation{
		Category:        category,
		PluginID:        pluginID,
		InstanceID:      instanceID,
		Enabled:         false,
		SignatureStatus: sigStatus,
		ConfigCipher:    "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := m.repo.UpsertPluginInstallation(ctx, &inst); err != nil {
		return domain.PluginInstallation{}, err
	}
	return m.repo.GetPluginInstallation(ctx, category, pluginID, instanceID)
}

func (m *Manager) DeleteInstance(ctx context.Context, category, pluginID, instanceID string) error {
	if m.repo == nil {
		return fmt.Errorf("plugin repo missing")
	}
	category = strings.TrimSpace(category)
	pluginID = strings.TrimSpace(pluginID)
	instanceID = strings.TrimSpace(instanceID)
	if category == "" || pluginID == "" || instanceID == "" {
		return fmt.Errorf("invalid input")
	}
	m.runtime.Stop(category, pluginID, instanceID)
	if err := m.repo.DeletePluginInstallation(ctx, category, pluginID, instanceID); err != nil {
		return err
	}
	// Delete physical files only when no instances remain.
	remain, err := m.repo.ListPluginInstallations(ctx)
	if err != nil {
		return nil
	}
	for _, inst := range remain {
		if inst.Category == category && inst.PluginID == pluginID {
			return nil
		}
	}
	_ = os.RemoveAll(filepath.Join(m.baseDir, category, pluginID))
	return nil
}

func (m *Manager) DeletePluginFiles(ctx context.Context, category, pluginID string) error {
	if m.repo == nil {
		return fmt.Errorf("plugin repo missing")
	}
	category = strings.TrimSpace(category)
	pluginID = strings.TrimSpace(pluginID)
	if category == "" || pluginID == "" {
		return fmt.Errorf("invalid input")
	}
	items, err := m.repo.ListPluginInstallations(ctx)
	if err != nil {
		return err
	}
	for _, inst := range items {
		if inst.Category == category && inst.PluginID == pluginID {
			return fmt.Errorf("cannot delete plugin files: instances still exist")
		}
	}
	_ = os.RemoveAll(filepath.Join(m.baseDir, category, pluginID))
	return nil
}

func (m *Manager) mergeSecretFields(ctx context.Context, category, pluginID string, oldConfigJSON string, newConfigJSON string) (string, error) {
	client, core, _, err := m.dialCore(ctx, category, pluginID)
	if err != nil {
		return "", err
	}
	defer client.Kill()

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	schema, err := core.GetConfigSchema(cctx, &pluginv1.Empty{})
	if err != nil {
		return "", err
	}

	var schemaObj any
	_ = json.Unmarshal([]byte(schema.GetJsonSchema()), &schemaObj)
	secretPaths := collectSecretPaths(schemaObj, nil)
	if len(secretPaths) == 0 {
		return newConfigJSON, nil
	}

	var oldObj any
	if err := json.Unmarshal([]byte(oldConfigJSON), &oldObj); err != nil {
		oldObj = map[string]any{}
	}
	var newObj any
	if err := json.Unmarshal([]byte(newConfigJSON), &newObj); err != nil {
		return "", fmt.Errorf("invalid config json")
	}

	for _, path := range secretPaths {
		ov, okO := getPath(oldObj, path)
		nv, okN := getPath(newObj, path)
		if !okO || !okN {
			continue
		}
		// "留空表示不修改" for secrets:
		// - empty string keeps old value
		// - null keeps old value
		if nv == nil {
			newObj = setPath(newObj, path, ov)
			continue
		}
		if s, ok := nv.(string); ok && strings.TrimSpace(s) == "" {
			newObj = setPath(newObj, path, ov)
			continue
		}
	}

	b, err := json.Marshal(newObj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func collectSecretPaths(schema any, prefix []string) [][]string {
	m, ok := schema.(map[string]any)
	if !ok {
		return nil
	}

	// Check if this node is a secret field.
	if format, ok := m["format"].(string); ok && strings.EqualFold(strings.TrimSpace(format), "password") {
		if len(prefix) > 0 {
			return [][]string{append([]string{}, prefix...)}
		}
	}
	if v, ok := m["x-secret"].(bool); ok && v {
		if len(prefix) > 0 {
			return [][]string{append([]string{}, prefix...)}
		}
	}

	props, _ := m["properties"].(map[string]any)
	if len(props) == 0 {
		return nil
	}
	var out [][]string
	for k, v := range props {
		if strings.TrimSpace(k) == "" {
			continue
		}
		out = append(out, collectSecretPaths(v, append(prefix, k))...)
	}
	return out
}

func getPath(obj any, path []string) (any, bool) {
	cur := obj
	for _, k := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := m[k]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

func setPath(obj any, path []string, value any) any {
	if len(path) == 0 {
		return obj
	}
	m, ok := obj.(map[string]any)
	if !ok {
		m = map[string]any{}
	}
	cur := m
	for i := 0; i < len(path)-1; i++ {
		k := path[i]
		next, ok := cur[k].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[k] = next
		}
		cur = next
	}
	cur[path[len(path)-1]] = value
	return m
}

func (m *Manager) GetPaymentClient(category, pluginID, instanceID string) (pluginv1.PaymentServiceClient, bool) {
	if strings.TrimSpace(instanceID) == "" {
		instanceID = DefaultInstanceID
	}
	if rp, ok := m.runtime.GetRunning(category, pluginID, instanceID); ok && rp.payment != nil {
		return rp.payment, true
	}
	return nil, false
}

func (m *Manager) GetSMSClient(category, pluginID, instanceID string) (pluginv1.SmsServiceClient, bool) {
	if strings.TrimSpace(instanceID) == "" {
		instanceID = DefaultInstanceID
	}
	if rp, ok := m.runtime.GetRunning(category, pluginID, instanceID); ok && rp.sms != nil {
		return rp.sms, true
	}
	return nil, false
}

func (m *Manager) GetKYCClient(category, pluginID, instanceID string) (pluginv1.KycServiceClient, bool) {
	if strings.TrimSpace(instanceID) == "" {
		instanceID = DefaultInstanceID
	}
	if rp, ok := m.runtime.GetRunning(category, pluginID, instanceID); ok && rp.kyc != nil {
		return rp.kyc, true
	}
	return nil, false
}

func (m *Manager) decryptConfig(cipherText string) (string, error) {
	if strings.TrimSpace(cipherText) == "" {
		return "", nil
	}
	if m.cipher == nil {
		return "", fmt.Errorf("plugin cipher missing")
	}
	plain, err := m.cipher.DecryptString(cipherText)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (m *Manager) encryptConfig(configJSON string) (string, error) {
	if m.cipher == nil {
		return "", fmt.Errorf("plugin cipher missing")
	}
	return m.cipher.EncryptToString([]byte(configJSON))
}

func ParseEd25519PublicKeys(base64Keys []string) []ed25519.PublicKey {
	var out []ed25519.PublicKey
	for _, k := range base64Keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		b, err := base64.StdEncoding.DecodeString(k)
		if err != nil || len(b) != ed25519.PublicKeySize {
			continue
		}
		out = append(out, ed25519.PublicKey(b))
	}
	return out
}

func newInstanceID(category, pluginID string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	s := strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), "=")
	return category + "-" + pluginID + "-" + strings.ToLower(s)
}

func (m *Manager) dialCore(ctx context.Context, category, pluginID string) (*plugin.Client, pluginv1.CoreServiceClient, *pluginv1.Manifest, error) {
	pluginDir := filepath.Join(m.baseDir, category, pluginID)
	manifestJSON, err := ReadManifest(pluginDir)
	if err != nil {
		return nil, nil, nil, err
	}
	entry, err := ResolveEntry(pluginDir, manifestJSON)
	if err != nil {
		if len(entry.SupportedPlatforms) > 0 {
			return nil, nil, nil, fmt.Errorf("%s", "unsupported platform "+entry.Platform+", supported: "+strings.Join(entry.SupportedPlatforms, ", "))
		}
		return nil, nil, nil, err
	}
	cmd := exec.Command(entry.EntryPath)
	cmd.Dir = pluginDir
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: pluginsdk.Handshake,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
		Plugins: map[string]plugin.Plugin{
			pluginsdk.PluginKeyCore: &pluginsdk.CoreGRPCPlugin{},
		},
		Cmd: cmd,
	})
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, nil, nil, err
	}
	rawCore, err := rpcClient.Dispense(pluginsdk.PluginKeyCore)
	if err != nil {
		client.Kill()
		return nil, nil, nil, err
	}
	core, ok := rawCore.(pluginv1.CoreServiceClient)
	if !ok {
		client.Kill()
		return nil, nil, nil, fmt.Errorf("invalid core client")
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	manifest, err := core.GetManifest(cctx, &pluginv1.Empty{})
	if err != nil {
		client.Kill()
		return nil, nil, nil, err
	}
	if manifest.GetPluginId() == "" {
		client.Kill()
		return nil, nil, nil, fmt.Errorf("invalid manifest")
	}
	return client, core, manifest, nil
}

func (m *Manager) validateConfig(ctx context.Context, category, pluginID string, configJSON string) error {
	client, core, _, err := m.dialCore(ctx, category, pluginID)
	if err != nil {
		return err
	}
	defer client.Kill()
	sctx, scancel := context.WithTimeout(ctx, 5*time.Second)
	schemaResp, schemaErr := core.GetConfigSchema(sctx, &pluginv1.Empty{})
	scancel()
	if schemaErr == nil && schemaResp != nil {
		missing := missingRequiredConfigFields(schemaResp.GetJsonSchema(), configJSON)
		if len(missing) > 0 {
			return &ConfigValidationError{
				Code:          "missing_required_config",
				Message:       "missing required config: " + strings.Join(missing, ", "),
				MissingFields: missing,
				RedirectPath:  redirectPathForCategory(category),
			}
		}
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := core.ValidateConfig(cctx, &pluginv1.ValidateConfigRequest{ConfigJson: configJSON})
	if err != nil {
		return err
	}
	if resp != nil && !resp.Ok {
		msg := strings.TrimSpace(resp.Error)
		if msg == "" {
			msg = "invalid config"
		}
		missing := parseMissingFieldsFromError(msg)
		code := "invalid_plugin_config"
		if len(missing) > 0 {
			code = "missing_required_config"
		}
		return &ConfigValidationError{
			Code:          code,
			Message:       msg,
			MissingFields: missing,
			RedirectPath:  redirectPathForCategory(category),
		}
	}
	return nil
}

func missingRequiredConfigFields(schemaJSON, configJSON string) []string {
	var schemaAny any
	if err := json.Unmarshal([]byte(strings.TrimSpace(schemaJSON)), &schemaAny); err != nil {
		return nil
	}
	cfgRaw := strings.TrimSpace(configJSON)
	if cfgRaw == "" {
		cfgRaw = "{}"
	}
	var cfgAny any
	if err := json.Unmarshal([]byte(cfgRaw), &cfgAny); err != nil {
		cfgAny = map[string]any{}
	}
	fields := collectMissingRequiredFields(schemaAny, cfgAny, nil)
	if len(fields) == 0 {
		return nil
	}
	uniq := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		uniq[f] = struct{}{}
	}
	out := make([]string, 0, len(uniq))
	for f := range uniq {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

func collectMissingRequiredFields(schema any, cfg any, prefix []string) []string {
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return nil
	}
	props, _ := schemaMap["properties"].(map[string]any)
	requiredRaw, _ := schemaMap["required"].([]any)
	cfgMap, _ := cfg.(map[string]any)

	requiredSet := map[string]struct{}{}
	out := make([]string, 0)
	for _, item := range requiredRaw {
		name, _ := item.(string)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		requiredSet[name] = struct{}{}
		path := appendPath(prefix, name)
		fullName := strings.Join(path, ".")
		val, exists := cfgMap[name]
		if !exists || isEmptyRequiredValue(val) {
			out = append(out, fullName)
			continue
		}
		if childSchema, ok := props[name]; ok {
			out = append(out, collectMissingRequiredFields(childSchema, val, path)...)
		}
	}

	for name, childSchema := range props {
		if _, exists := requiredSet[name]; exists {
			continue
		}
		val, exists := cfgMap[name]
		if !exists || val == nil {
			continue
		}
		out = append(out, collectMissingRequiredFields(childSchema, val, appendPath(prefix, name))...)
	}

	return out
}

func appendPath(prefix []string, name string) []string {
	next := make([]string, 0, len(prefix)+1)
	next = append(next, prefix...)
	next = append(next, name)
	return next
}

func isEmptyRequiredValue(v any) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t) == ""
	}
	return false
}

func parseMissingFieldsFromError(msg string) []string {
	raw := strings.TrimSpace(msg)
	if raw == "" {
		return nil
	}
	lower := strings.ToLower(raw)
	idx := strings.Index(lower, "required")
	if idx <= 0 {
		return nil
	}
	prefix := strings.TrimSpace(raw[:idx])
	prefix = strings.Trim(prefix, ":")
	if prefix == "" {
		return nil
	}
	replacer := strings.NewReplacer(",", "/", ";", "/", "|", "/", " and ", "/", " AND ", "/")
	prefix = replacer.Replace(prefix)
	parts := strings.Split(prefix, "/")
	uniq := map[string]struct{}{}
	for _, p := range parts {
		token := strings.TrimSpace(p)
		if token == "" {
			continue
		}
		if !isLikelyFieldName(token) {
			continue
		}
		uniq[token] = struct{}{}
	}
	if len(uniq) == 0 {
		return nil
	}
	out := make([]string, 0, len(uniq))
	for k := range uniq {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func isLikelyFieldName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func redirectPathForCategory(category string) string {
	if strings.EqualFold(strings.TrimSpace(category), "automation") {
		return "/admin/catalog"
	}
	return ""
}
