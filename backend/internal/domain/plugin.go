package domain

import "time"

type PluginSignatureStatus string

const (
	PluginSignatureOfficial  PluginSignatureStatus = "official"
	PluginSignatureUntrusted PluginSignatureStatus = "untrusted"
	PluginSignatureUnsigned  PluginSignatureStatus = "unsigned"
)

type PluginInstallation struct {
	ID              int64
	Category        string
	PluginID        string
	InstanceID      string
	Enabled         bool
	SignatureStatus PluginSignatureStatus
	// ContentDigest is the sha256 of the canonical plugin file manifest recorded
	// at install/import time. It is the tamper-evidence anchor for unsigned and
	// untrusted plugins (which have no signature to re-check) and is re-verified
	// on every start, separate from the signature status.
	ContentDigest string
	ConfigCipher string `json:"-"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PluginPaymentMethod struct {
	ID         int64
	Category   string
	PluginID   string
	InstanceID string
	Method     string
	Enabled    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
