package plugins

import (
	"context"
	"io"

	"fmt"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type AdminManager struct {
	inner *Manager
}

func NewAdminManager(inner *Manager) *AdminManager {
	if inner == nil {
		return nil
	}
	return &AdminManager{inner: inner}
}

func (m *AdminManager) List(ctx context.Context) ([]appshared.PluginListItem, error) {
	if m == nil || m.inner == nil {
		return nil, fmt.Errorf("plugins disabled")
	}
	items, err := m.inner.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]appshared.PluginListItem, 0, len(items))
	for _, it := range items {
		out = append(out, mapListItem(it))
	}
	return out, nil
}

func (m *AdminManager) DiscoverOnDisk(ctx context.Context) ([]appshared.PluginDiscoverItem, error) {
	if m == nil || m.inner == nil {
		return nil, fmt.Errorf("plugins disabled")
	}
	items, err := m.inner.DiscoverOnDisk(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]appshared.PluginDiscoverItem, 0, len(items))
	for _, it := range items {
		out = append(out, appshared.PluginDiscoverItem{
			Category:        it.Category,
			PluginID:        it.PluginID,
			Name:            it.Name,
			Version:         it.Version,
			SignatureStatus: it.SignatureStatus,
			Entry: appshared.PluginEntryInfo{
				Platform:           it.Entry.Platform,
				EntryPath:          it.Entry.EntryPath,
				EntrySupported:     it.Entry.EntrySupported,
				SupportedPlatforms: it.Entry.SupportedPlatforms,
			},
		})
	}
	return out, nil
}

func (m *AdminManager) Install(ctx context.Context, filename string, r io.Reader) (domain.PluginInstallation, error) {
	if m == nil || m.inner == nil {
		return domain.PluginInstallation{}, fmt.Errorf("plugins disabled")
	}
	return m.inner.Install(ctx, filename, r)
}

func (m *AdminManager) Uninstall(ctx context.Context, category, pluginID string) error {
	if m == nil || m.inner == nil {
		return fmt.Errorf("plugins disabled")
	}
	return m.inner.Uninstall(ctx, category, pluginID)
}

func (m *AdminManager) SignatureStatusOnDisk(category, pluginID string) (domain.PluginSignatureStatus, error) {
	if m == nil || m.inner == nil {
		return domain.PluginSignatureUntrusted, fmt.Errorf("plugins disabled")
	}
	return m.inner.SignatureStatusOnDisk(category, pluginID)
}

func (m *AdminManager) ImportFromDisk(ctx context.Context, category, pluginID string) (domain.PluginInstallation, error) {
	if m == nil || m.inner == nil {
		return domain.PluginInstallation{}, fmt.Errorf("plugins disabled")
	}
	return m.inner.ImportFromDisk(ctx, category, pluginID)
}

func (m *AdminManager) EnableInstance(ctx context.Context, category, pluginID, instanceID string) error {
	if m == nil || m.inner == nil {
		return fmt.Errorf("plugins disabled")
	}
	return m.inner.EnableInstance(ctx, category, pluginID, instanceID)
}

func (m *AdminManager) DisableInstance(ctx context.Context, category, pluginID, instanceID string) error {
	if m == nil || m.inner == nil {
		return fmt.Errorf("plugins disabled")
	}
	return m.inner.DisableInstance(ctx, category, pluginID, instanceID)
}

func (m *AdminManager) DeleteInstance(ctx context.Context, category, pluginID, instanceID string) error {
	if m == nil || m.inner == nil {
		return fmt.Errorf("plugins disabled")
	}
	return m.inner.DeleteInstance(ctx, category, pluginID, instanceID)
}

func (m *AdminManager) GetConfigSchemaInstance(ctx context.Context, category, pluginID, instanceID string) (string, string, error) {
	if m == nil || m.inner == nil {
		return "", "", fmt.Errorf("plugins disabled")
	}
	return m.inner.GetConfigSchemaInstance(ctx, category, pluginID, instanceID)
}

func (m *AdminManager) GetConfigInstance(ctx context.Context, category, pluginID, instanceID string) (string, error) {
	if m == nil || m.inner == nil {
		return "", fmt.Errorf("plugins disabled")
	}
	return m.inner.GetConfigInstance(ctx, category, pluginID, instanceID)
}

func (m *AdminManager) UpdateConfigInstance(ctx context.Context, category, pluginID, instanceID string, configJSON string) error {
	if m == nil || m.inner == nil {
		return fmt.Errorf("plugins disabled")
	}
	if err := m.inner.UpdateConfigInstance(ctx, category, pluginID, instanceID, configJSON); err != nil {
		if cfgErr, ok := AsConfigValidationError(err); ok && cfgErr != nil {
			return &appshared.ConfigValidationError{
				Code:          cfgErr.Code,
				Message:       cfgErr.Message,
				MissingFields: cfgErr.MissingFields,
				RedirectPath:  cfgErr.RedirectPath,
			}
		}
		return err
	}
	return nil
}

func (m *AdminManager) CreateInstance(ctx context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error) {
	if m == nil || m.inner == nil {
		return domain.PluginInstallation{}, fmt.Errorf("plugins disabled")
	}
	return m.inner.CreateInstance(ctx, category, pluginID, instanceID)
}

func (m *AdminManager) DeletePluginFiles(ctx context.Context, category, pluginID string) error {
	if m == nil || m.inner == nil {
		return fmt.Errorf("plugins disabled")
	}
	return m.inner.DeletePluginFiles(ctx, category, pluginID)
}

func mapListItem(it ListItem) appshared.PluginListItem {
	out := appshared.PluginListItem{
		Category:        it.Category,
		PluginID:        it.PluginID,
		InstanceID:      it.InstanceID,
		Name:            it.Name,
		Version:         it.Version,
		SignatureStatus: it.SignatureStatus,
		Enabled:         it.Enabled,
		Loaded:          it.Loaded,
		InstalledAt:     it.InstalledAt,
		UpdatedAt:       it.UpdatedAt,
		LastHealthAt:    it.LastHealthAt,
		HealthStatus:    it.HealthStatus,
		HealthMessage:   it.HealthMessage,
		Capabilities: appshared.PluginManifest{
			PluginID:    it.Capabilities.PluginID,
			Name:        it.Capabilities.Name,
			Version:     it.Capabilities.Version,
			Description: it.Capabilities.Description,
			Binaries:    it.Capabilities.Binaries,
			Capabilities: appshared.PluginCapabilities{
				SMS: mapSMSCapability(it.Capabilities.Capabilities.SMS),
				Payment: &appshared.PluginPaymentCapability{
					Methods: nil,
				},
				KYC:        mapKYCCapability(it.Capabilities.Capabilities.KYC),
				Automation: mapAutomationCapability(it.Capabilities.Capabilities.Automation),
			},
		},
		Entry: appshared.PluginEntryInfo{
			Platform:           it.Entry.Platform,
			EntryPath:          it.Entry.EntryPath,
			EntrySupported:     it.Entry.EntrySupported,
			SupportedPlatforms: it.Entry.SupportedPlatforms,
		},
	}
	if it.Capabilities.Capabilities.Payment == nil {
		out.Capabilities.Capabilities.Payment = nil
	} else {
		out.Capabilities.Capabilities.Payment.Methods = it.Capabilities.Capabilities.Payment.Methods
	}
	return out
}

func mapSMSCapability(in *struct {
	Send bool "json:\"send\""
}) *appshared.PluginSMSCapability {
	if in == nil {
		return nil
	}
	return &appshared.PluginSMSCapability{Send: in.Send}
}

func mapKYCCapability(in *struct {
	Start       bool "json:\"start\""
	QueryResult bool "json:\"query_result\""
}) *appshared.PluginKYCCapability {
	if in == nil {
		return nil
	}
	return &appshared.PluginKYCCapability{Start: in.Start, QueryResult: in.QueryResult}
}

func mapAutomationCapability(in *struct {
	Features           []string          "json:\"features\""
	NotSupportedReason map[string]string "json:\"not_supported_reasons,omitempty\""
}) *appshared.PluginAutomationCapability {
	if in == nil {
		return nil
	}
	return &appshared.PluginAutomationCapability{
		Features:            in.Features,
		NotSupportedReasons: in.NotSupportedReason,
	}
}
