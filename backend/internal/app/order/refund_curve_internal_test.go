package order

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
)

type fakeSettingRepo struct {
	values map[string]string
}

func (f *fakeSettingRepo) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	if v, ok := f.values[key]; ok {
		return domain.Setting{Key: key, ValueJSON: v}, nil
	}
	return domain.Setting{}, ErrNotFound
}

func (f *fakeSettingRepo) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[setting.Key] = setting.ValueJSON
	return nil
}

func (f *fakeSettingRepo) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	return nil, nil
}

func (f *fakeSettingRepo) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	return nil, nil
}

func (f *fakeSettingRepo) GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error) {
	return domain.EmailTemplate{}, ErrNotFound
}

func (f *fakeSettingRepo) UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {
	return nil
}

func (f *fakeSettingRepo) DeleteEmailTemplate(ctx context.Context, id int64) error {
	return nil
}

func TestRefundCurveNormalizeAndRatio(t *testing.T) {
	points := NormalizeRefundCurve([]RefundCurvePoint{
		{Percent: 100, Ratio: 1.5},
		{Percent: 0, Ratio: -1},
		{Percent: 50, Ratio: 0.5},
	})
	if len(points) != 3 {
		t.Fatalf("expected normalized points")
	}
	if ratio, ok := RefundCurveRatio(points, 25); !ok || ratio <= 0 || ratio >= 1 {
		t.Fatalf("unexpected ratio: %v", ratio)
	}
}

func TestLoadRefundCurve(t *testing.T) {
	repo := &fakeSettingRepo{values: map[string]string{
		"refund_curve_json": `[{"hours":0,"ratio":1},{"hours":100,"ratio":0.5}]`,
	}}
	if points, ok := LoadRefundCurve(context.Background(), repo); !ok || len(points) != 2 {
		t.Fatalf("expected refund curve")
	}
}
