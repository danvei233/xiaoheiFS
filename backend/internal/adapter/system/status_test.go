package system

import (
	"context"
	"testing"
)

func TestDefaultDiskPath(t *testing.T) {
	path := defaultDiskPath()
	if path == "" {
		t.Fatalf("expected non-empty path")
	}
}

func TestProviderStatus(t *testing.T) {
	p := NewProvider()
	status, err := p.Status(context.Background())
	if err != nil {
		t.Fatalf("status error: %v", err)
	}
	if status.Hostname == "" && status.OS == "" && status.Platform == "" {
		t.Fatalf("expected status fields")
	}
}
