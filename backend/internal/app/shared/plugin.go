package shared

import (
	"errors"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

type PluginSMSCapability struct {
	Send bool `json:"send"`
}

type PluginPaymentCapability struct {
	Methods []string `json:"methods"`
}

type PluginKYCCapability struct {
	Start       bool `json:"start"`
	QueryResult bool `json:"query_result"`
}

type PluginAutomationCapability struct {
	Features            []string          `json:"features"`
	NotSupportedReasons map[string]string `json:"not_supported_reasons,omitempty"`
}

type PluginCapabilities struct {
	SMS        *PluginSMSCapability        `json:"sms,omitempty"`
	Payment    *PluginPaymentCapability    `json:"payment,omitempty"`
	KYC        *PluginKYCCapability        `json:"kyc,omitempty"`
	Automation *PluginAutomationCapability `json:"automation,omitempty"`
}

type PluginManifest struct {
	PluginID     string             `json:"plugin_id"`
	Name         string             `json:"name"`
	Version      string             `json:"version"`
	Description  string             `json:"description,omitempty"`
	Binaries     map[string]string  `json:"binaries,omitempty"`
	Capabilities PluginCapabilities `json:"capabilities"`
}

type PluginEntryInfo struct {
	Platform           string   `json:"platform"`
	EntryPath          string   `json:"entry_path"`
	EntrySupported     bool     `json:"entry_supported"`
	SupportedPlatforms []string `json:"supported_platforms"`
}

type PluginListItem struct {
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
	Capabilities    PluginManifest               `json:"manifest"`
	Entry           PluginEntryInfo              `json:"entry"`
}

type PluginDiscoverItem struct {
	Category        string                       `json:"category"`
	PluginID        string                       `json:"plugin_id"`
	Name            string                       `json:"name"`
	Version         string                       `json:"version"`
	SignatureStatus domain.PluginSignatureStatus `json:"signature_status"`
	Entry           PluginEntryInfo              `json:"entry"`
}

type PluginPaymentMethodState struct {
	Method  string `json:"method"`
	Enabled bool   `json:"enabled"`
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
	if code := strings.TrimSpace(e.Code); code != "" {
		return code
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
