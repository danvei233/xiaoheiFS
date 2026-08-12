package payment

import (
	"gorm.io/gorm"
	"context"
	"errors"
	"testing"

	"xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

type failingSettingsRepo struct {
	getErr error
	value  string
}

func (f *failingSettingsRepo) UpdateSettingAtomic(ctx context.Context, key string, mutate func(current string) (string, error)) error {
	current, err := f.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			current = domain.Setting{}
		} else {
			return err
		}
	}
	next, err := mutate(current.ValueJSON)
	if err != nil {
		return err
	}
	return f.UpsertSetting(ctx, domain.Setting{Key: key, ValueJSON: next})
}
func (f *failingSettingsRepo) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	if f.getErr != nil {
		return domain.Setting{}, f.getErr
	}
	return domain.Setting{Key: key, ValueJSON: f.value}, nil
}

func (f *failingSettingsRepo) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	return nil
}

func (f *failingSettingsRepo) UpsertSettings(ctx context.Context, settings []domain.Setting) error {
	return nil
}

func (f *failingSettingsRepo) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	return nil, nil
}

func (f *failingSettingsRepo) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	return nil, nil
}

func (f *failingSettingsRepo) GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error) {
	return domain.EmailTemplate{}, nil
}

func (f *failingSettingsRepo) UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {
	return nil
}

func (f *failingSettingsRepo) DeleteEmailTemplate(ctx context.Context, id int64) error {
	return nil
}

func TestRegistry_ListAndUpdate(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	reg := NewRegistry(repo)
	ctx := context.Background()

	providers, err := reg.ListProviders(ctx, true)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(providers) == 0 {
		t.Fatalf("expected providers")
	}

	cfg, enabled, err := reg.GetProviderConfig(ctx, "approval")
	if err != nil {
		t.Fatalf("get provider config: %v", err)
	}
	if cfg != "" || !enabled {
		t.Fatalf("expected approval default config empty and enabled status")
	}

	if err := reg.UpdateProviderConfig(ctx, "approval", true, ``); err != nil {
		t.Fatalf("update provider config: %v", err)
	}
	provider, err := reg.GetProvider(ctx, "approval")
	if err != nil {
		t.Fatalf("get provider: %v", err)
	}
	if provider.Key() != "approval" {
		t.Fatalf("unexpected provider key: %s", provider.Key())
	}
	if _, err := provider.CreatePayment(ctx, shared.PaymentCreateRequest{OrderID: 1, UserID: 2, Amount: 1000, Subject: "test"}); err == nil {
		t.Fatalf("expected approval provider create payment unsupported")
	}

	if err := reg.UpdateProviderConfig(ctx, "approval", false, ``); err != nil {
		t.Fatalf("disable provider: %v", err)
	}
	if _, err := reg.GetProvider(ctx, "approval"); err == nil {
		t.Fatalf("expected forbidden")
	}
}

func TestRegistry_SceneEnabledPersistence(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	reg := NewRegistry(repo)
	ctx := context.Background()

	enabled, err := reg.GetProviderSceneEnabled(ctx, "approval", "order")
	if err != nil {
		t.Fatalf("get default provider scene enabled: %v", err)
	}
	if !enabled {
		t.Fatalf("expected scene enabled by default")
	}

	if err := reg.UpdateProviderSceneEnabled(ctx, "approval", "order", false); err != nil {
		t.Fatalf("disable provider scene: %v", err)
	}
	enabled, err = reg.GetProviderSceneEnabled(ctx, "approval", "order")
	if err != nil {
		t.Fatalf("get updated provider scene enabled: %v", err)
	}
	if enabled {
		t.Fatalf("expected order scene disabled after update")
	}

	enabled, err = reg.GetProviderSceneEnabled(ctx, "approval", "wallet")
	if err != nil {
		t.Fatalf("get other scene enabled state: %v", err)
	}
	if !enabled {
		t.Fatalf("expected other scene to remain enabled")
	}
}

func TestRegistry_UpdateProviderSceneEnabled_InitializesNilProviderMap(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	if err := repo.UpsertSetting(ctx, domain.Setting{
		Key:       settingPaymentScene,
		ValueJSON: `{"approval":null}`,
	}); err != nil {
		t.Fatalf("seed scene setting: %v", err)
	}

	reg := NewRegistry(repo)
	if err := reg.UpdateProviderSceneEnabled(ctx, "approval", "order", false); err != nil {
		t.Fatalf("update provider scene with nil map: %v", err)
	}

	enabled, err := reg.GetProviderSceneEnabled(ctx, "approval", "order")
	if err != nil {
		t.Fatalf("get provider scene after update: %v", err)
	}
	if enabled {
		t.Fatalf("expected order scene disabled after update from nil provider map")
	}
}

func TestRegistry_GetProviderSceneEnabled_ErrorsWhenSceneSettingLoadFails(t *testing.T) {
	ctx := context.Background()
	repoErr := errors.New("db unavailable")
	reg := NewRegistry(&failingSettingsRepo{getErr: repoErr})

	enabled, err := reg.GetProviderSceneEnabled(ctx, "approval", "order")
	if err == nil {
		t.Fatalf("expected error when scene setting load fails")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got %v", err)
	}
	if enabled {
		t.Fatalf("expected disabled state when loading scene setting fails")
	}
}

func TestRegistry_GetProviderSceneEnabled_ErrorsWhenSceneSettingJSONInvalid(t *testing.T) {
	ctx := context.Background()
	reg := NewRegistry(&failingSettingsRepo{value: "{invalid json"})

	enabled, err := reg.GetProviderSceneEnabled(ctx, "approval", "order")
	if err == nil {
		t.Fatalf("expected error when scene setting JSON is invalid")
	}
	if enabled {
		t.Fatalf("expected disabled state when scene setting JSON is invalid")
	}
}

func TestRegistry_GetProviderSceneEnabled_StillDefaultsEnabledWhenSceneSettingMissing(t *testing.T) {
	ctx := context.Background()
	reg := NewRegistry(&failingSettingsRepo{getErr: shared.ErrNotFound})

	enabled, err := reg.GetProviderSceneEnabled(ctx, "approval", "order")
	if err != nil {
		t.Fatalf("expected nil error for missing scene setting, got %v", err)
	}
	if !enabled {
		t.Fatalf("expected enabled by default when scene setting is missing")
	}
}
