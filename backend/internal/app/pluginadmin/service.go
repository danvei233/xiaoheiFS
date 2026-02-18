package pluginadmin

import (
	"context"
	"io"
	"sort"
	"strings"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

const DefaultInstanceID = "default"

type Manager interface {
	List(ctx context.Context) ([]appshared.PluginListItem, error)
	DiscoverOnDisk(ctx context.Context) ([]appshared.PluginDiscoverItem, error)
	Install(ctx context.Context, filename string, r io.Reader) (domain.PluginInstallation, error)
	Uninstall(ctx context.Context, category, pluginID string) error
	SignatureStatusOnDisk(category, pluginID string) (domain.PluginSignatureStatus, error)
	ImportFromDisk(ctx context.Context, category, pluginID string) (domain.PluginInstallation, error)
	EnableInstance(ctx context.Context, category, pluginID, instanceID string) error
	DisableInstance(ctx context.Context, category, pluginID, instanceID string) error
	DeleteInstance(ctx context.Context, category, pluginID, instanceID string) error
	GetConfigSchemaInstance(ctx context.Context, category, pluginID, instanceID string) (jsonSchema, uiSchema string, err error)
	GetConfigInstance(ctx context.Context, category, pluginID, instanceID string) (string, error)
	UpdateConfigInstance(ctx context.Context, category, pluginID, instanceID string, configJSON string) error
	CreateInstance(ctx context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error)
	DeletePluginFiles(ctx context.Context, category, pluginID string) error
}

type Service struct {
	manager        Manager
	paymentMethods appports.PluginPaymentMethodRepository
	settings       appports.SettingsRepository
}

func NewService(manager Manager, paymentMethods appports.PluginPaymentMethodRepository, settings appports.SettingsRepository) *Service {
	return &Service{manager: manager, paymentMethods: paymentMethods, settings: settings}
}

func (s *Service) List(ctx context.Context) ([]appshared.PluginListItem, error) {
	if s.manager == nil {
		return nil, domain.ErrPluginsDisabled
	}
	return s.manager.List(ctx)
}

func (s *Service) DiscoverOnDisk(ctx context.Context) ([]appshared.PluginDiscoverItem, error) {
	if s.manager == nil {
		return nil, domain.ErrPluginsDisabled
	}
	return s.manager.DiscoverOnDisk(ctx)
}

func (s *Service) Install(ctx context.Context, filename string, r io.Reader) (domain.PluginInstallation, error) {
	if s.manager == nil {
		return domain.PluginInstallation{}, domain.ErrPluginsDisabled
	}
	return s.manager.Install(ctx, filename, r)
}

func (s *Service) Uninstall(ctx context.Context, category, pluginID string) error {
	if s.manager == nil {
		return domain.ErrPluginsDisabled
	}
	return s.manager.Uninstall(ctx, category, pluginID)
}

func (s *Service) SignatureStatusOnDisk(category, pluginID string) (domain.PluginSignatureStatus, error) {
	if s.manager == nil {
		return domain.PluginSignatureUntrusted, domain.ErrPluginsDisabled
	}
	return s.manager.SignatureStatusOnDisk(category, pluginID)
}

func (s *Service) ImportFromDisk(ctx context.Context, category, pluginID string) (domain.PluginInstallation, error) {
	if s.manager == nil {
		return domain.PluginInstallation{}, domain.ErrPluginsDisabled
	}
	return s.manager.ImportFromDisk(ctx, category, pluginID)
}

func (s *Service) EnableInstance(ctx context.Context, category, pluginID, instanceID string) error {
	if s.manager == nil {
		return domain.ErrPluginsDisabled
	}
	return s.manager.EnableInstance(ctx, category, pluginID, instanceID)
}

func (s *Service) DisableInstance(ctx context.Context, category, pluginID, instanceID string) error {
	if s.manager == nil {
		return domain.ErrPluginsDisabled
	}
	return s.manager.DisableInstance(ctx, category, pluginID, instanceID)
}

func (s *Service) DeleteInstance(ctx context.Context, category, pluginID, instanceID string) error {
	if s.manager == nil {
		return domain.ErrPluginsDisabled
	}
	return s.manager.DeleteInstance(ctx, category, pluginID, instanceID)
}

func (s *Service) GetConfigSchemaInstance(ctx context.Context, category, pluginID, instanceID string) (string, string, error) {
	if s.manager == nil {
		return "", "", domain.ErrPluginsDisabled
	}
	return s.manager.GetConfigSchemaInstance(ctx, category, pluginID, instanceID)
}

func (s *Service) GetConfigInstance(ctx context.Context, category, pluginID, instanceID string) (string, error) {
	if s.manager == nil {
		return "", domain.ErrPluginsDisabled
	}
	return s.manager.GetConfigInstance(ctx, category, pluginID, instanceID)
}

func (s *Service) UpdateConfigInstance(ctx context.Context, category, pluginID, instanceID, configJSON string) error {
	if s.manager == nil {
		return domain.ErrPluginsDisabled
	}
	return s.manager.UpdateConfigInstance(ctx, category, pluginID, instanceID, configJSON)
}

func (s *Service) CreateInstance(ctx context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error) {
	if s.manager == nil {
		return domain.PluginInstallation{}, domain.ErrPluginsDisabled
	}
	return s.manager.CreateInstance(ctx, category, pluginID, instanceID)
}

func (s *Service) DeletePluginFiles(ctx context.Context, category, pluginID string) error {
	if s.manager == nil {
		return domain.ErrPluginsDisabled
	}
	return s.manager.DeletePluginFiles(ctx, category, pluginID)
}

func (s *Service) UpsertPaymentMethod(ctx context.Context, category, pluginID, instanceID, method string, enabled bool) error {
	if s.paymentMethods == nil {
		return domain.ErrPaymentMethodRepoMissing
	}
	return s.paymentMethods.UpsertPluginPaymentMethod(ctx, &domain.PluginPaymentMethod{
		Category:   strings.TrimSpace(category),
		PluginID:   strings.TrimSpace(pluginID),
		InstanceID: strings.TrimSpace(instanceID),
		Method:     strings.TrimSpace(method),
		Enabled:    enabled,
	})
}

func (s *Service) ListPaymentMethods(ctx context.Context, category, pluginID, instanceID string) ([]appshared.PluginPaymentMethodState, error) {
	if s.manager == nil {
		return nil, domain.ErrPluginsDisabled
	}
	if s.paymentMethods == nil {
		return nil, domain.ErrPaymentMethodRepoMissing
	}
	category = strings.TrimSpace(category)
	pluginID = strings.TrimSpace(pluginID)
	instanceID = strings.TrimSpace(instanceID)
	if category == "" {
		category = "payment"
	}
	if instanceID == "" {
		instanceID = DefaultInstanceID
	}
	if category == "" || pluginID == "" || instanceID == "" {
		return nil, domain.ErrPluginInstanceRequired
	}

	items, err := s.manager.List(ctx)
	if err != nil {
		return nil, err
	}
	var supported []string
	for _, it := range items {
		if it.Category != category || it.PluginID != pluginID || it.InstanceID != instanceID {
			continue
		}
		if !it.Enabled || !it.Loaded {
			return nil, domain.ErrPluginInstanceNotEnabled
		}
		if it.Capabilities.Capabilities.Payment == nil {
			return nil, domain.ErrNotPaymentPluginInstance
		}
		supported = it.Capabilities.Capabilities.Payment.Methods
		break
	}
	if len(supported) == 0 {
		return nil, domain.ErrPluginInstanceNotFound
	}
	overrides, _ := s.paymentMethods.ListPluginPaymentMethods(ctx, category, pluginID, instanceID)
	enabledMap := map[string]bool{}
	for _, ov := range overrides {
		enabledMap[ov.Method] = ov.Enabled
	}
	out := make([]appshared.PluginPaymentMethodState, 0, len(supported))
	for _, m := range supported {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		enabled, ok := enabledMap[m]
		if !ok {
			enabled = true
		}
		out = append(out, appshared.PluginPaymentMethodState{Method: m, Enabled: enabled})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Method < out[j].Method })
	return out, nil
}

func (s *Service) UpdatePaymentMethod(ctx context.Context, category, pluginID, instanceID, method string, enabled bool) error {
	if s.manager == nil {
		return domain.ErrPluginsDisabled
	}
	if s.paymentMethods == nil {
		return domain.ErrPaymentMethodRepoMissing
	}
	category = strings.TrimSpace(category)
	pluginID = strings.TrimSpace(pluginID)
	instanceID = strings.TrimSpace(instanceID)
	method = strings.TrimSpace(method)
	if category == "" {
		category = "payment"
	}
	if instanceID == "" {
		instanceID = DefaultInstanceID
	}
	if category == "" || pluginID == "" || instanceID == "" || method == "" {
		return domain.ErrPluginMethodUpdateRequired
	}

	items, err := s.manager.List(ctx)
	if err != nil {
		return err
	}
	found := false
	for _, it := range items {
		if it.Category != category || it.PluginID != pluginID || it.InstanceID != instanceID {
			continue
		}
		if !it.Enabled || !it.Loaded || it.Capabilities.Capabilities.Payment == nil {
			return domain.ErrPluginInstanceNotEnabled
		}
		supported := false
		for _, m := range it.Capabilities.Capabilities.Payment.Methods {
			if strings.TrimSpace(m) == method {
				supported = true
				break
			}
		}
		if !supported {
			return domain.ErrPluginMethodNotSupported
		}
		found = true
		break
	}
	if !found {
		return domain.ErrPluginInstanceNotFound
	}
	return s.UpsertPaymentMethod(ctx, category, pluginID, instanceID, method, enabled)
}

func (s *Service) ResolveUploadPassword(ctx context.Context, configuredPassword string) string {
	if v := strings.TrimSpace(configuredPassword); v != "" {
		return v
	}
	if s.settings != nil {
		if setting, err := s.settings.GetSetting(ctx, "payment_plugin_upload_password"); err == nil {
			if v := strings.TrimSpace(setting.ValueJSON); v != "" {
				return v
			}
		}
	}
	return ""
}

func (s *Service) ResolveUploadDir(ctx context.Context, configuredDir string) string {
	if v := strings.TrimSpace(configuredDir); v != "" {
		return v
	}
	if s.settings != nil {
		if setting, err := s.settings.GetSetting(ctx, "payment_plugin_dir"); err == nil {
			if v := strings.TrimSpace(setting.ValueJSON); v != "" {
				return v
			}
		}
	}
	return "plugins/payment"
}
