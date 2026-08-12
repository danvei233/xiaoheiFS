package automation

import (
	"context"
	"errors"
	"strings"
	"testing"

	plugins "xiaoheiplay/internal/adapter/plugins/core"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type fakeGoodsTypeRepo struct {
	listErr error
	getErr  error
	items   []domain.GoodsType
	byID    map[int64]domain.GoodsType
}

func (f *fakeGoodsTypeRepo) ListGoodsTypes(ctx context.Context) ([]domain.GoodsType, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.items, nil
}

func (f *fakeGoodsTypeRepo) GetGoodsType(ctx context.Context, id int64) (domain.GoodsType, error) {
	if f.getErr != nil {
		return domain.GoodsType{}, f.getErr
	}
	if gt, ok := f.byID[id]; ok {
		return gt, nil
	}
	return domain.GoodsType{}, appshared.ErrNotFound
}

func (f *fakeGoodsTypeRepo) CreateGoodsType(ctx context.Context, gt *domain.GoodsType) error {
	return nil
}
func (f *fakeGoodsTypeRepo) UpdateGoodsType(ctx context.Context, gt domain.GoodsType) error {
	return nil
}
func (f *fakeGoodsTypeRepo) DeleteGoodsType(ctx context.Context, id int64) error { return nil }

func TestClientForGoodsType_PreservesListGoodsTypesError(t *testing.T) {
	sentinel := errors.New("db unavailable")
	repo := &fakeGoodsTypeRepo{listErr: sentinel}
	resolver := NewResolver(repo, nil, nil, nil)

	_, err := resolver.ClientForGoodsType(context.Background(), 0)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "db unavailable") {
		t.Fatalf("expected original list error to be preserved, got: %v", err)
	}
	if strings.Contains(err.Error(), "goods_type_id required") {
		t.Fatalf("expected no misleading goods_type_id required, got: %v", err)
	}
}

func TestClientForGoodsType_CaseInsensitiveCategory(t *testing.T) {
	repo := &fakeGoodsTypeRepo{byID: map[int64]domain.GoodsType{
		7: {
			ID:                   7,
			AutomationCategory:   "Automation", // manager accepts via EqualFold
			AutomationPluginID:   "demo",
			AutomationInstanceID: "line-a",
		},
	}}
	resolver := NewResolver(repo, &plugins.Manager{}, nil, nil)

	cli, err := resolver.ClientForGoodsType(context.Background(), 7)
	if err != nil {
		t.Fatalf("expected case-insensitive category to be accepted, got: %v", err)
	}
	if cli == nil {
		t.Fatalf("expected client")
	}
}

func TestClientForGoodsType_RejectsNonAutomationCategory(t *testing.T) {
	repo := &fakeGoodsTypeRepo{byID: map[int64]domain.GoodsType{
		7: {
			ID:                   7,
			AutomationCategory:   "sms",
			AutomationPluginID:   "demo",
			AutomationInstanceID: "line-a",
		},
	}}
	resolver := NewResolver(repo, nil, nil, nil)

	_, err := resolver.ClientForGoodsType(context.Background(), 7)
	if err == nil {
		t.Fatalf("expected error for non-automation category")
	}
}
