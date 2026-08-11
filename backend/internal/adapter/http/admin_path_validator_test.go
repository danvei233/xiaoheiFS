package http

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
)

type stubSettingsService struct {
	valueJSON string
	err       error
	gotCtx    context.Context
	gotKey    string
}

func (s *stubSettingsService) Get(ctx context.Context, key string) (domain.Setting, error) {
	s.gotCtx = ctx
	s.gotKey = key
	if s.err != nil {
		return domain.Setting{}, s.err
	}
	return domain.Setting{Key: key, ValueJSON: s.valueJSON}, nil
}

func (s *stubSettingsService) List(ctx context.Context) ([]domain.Setting, error) {
	return nil, nil
}

func (s *stubSettingsService) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	return nil, nil
}

// M-1: GetAdminPathFromSettings must not pass a nil context into the settings
// service. A nil context can panic in the repo layer (GORM ctx deref) and is a
// latent crash whenever this helper is called.
func TestGetAdminPathFromSettings_NonNilContext(t *testing.T) {
	svc := &stubSettingsService{valueJSON: `"adminSecure01"`}
	got := GetAdminPathFromSettings(svc)
	if svc.gotCtx == nil {
		t.Fatalf("GetAdminPathFromSettings passed a nil context to SettingsService.Get")
	}
	if svc.gotKey != "admin_path" {
		t.Fatalf("expected key admin_path, got %q", svc.gotKey)
	}
	if got != "adminSecure01" {
		t.Fatalf("expected adminSecure01, got %q", got)
	}
}

func TestGetAdminPathFromSettings_InvalidPathReturnsEmpty(t *testing.T) {
	svc := &stubSettingsService{valueJSON: `"admin/with/slash"`}
	if got := GetAdminPathFromSettings(svc); got != "" {
		t.Fatalf("expected empty for invalid path, got %q", got)
	}
}

func TestGetAdminPathFromSettings_NilServiceReturnsEmpty(t *testing.T) {
	if got := GetAdminPathFromSettings(nil); got != "" {
		t.Fatalf("expected empty for nil service, got %q", got)
	}
}
