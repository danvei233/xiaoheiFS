package goodstype

import (
	"context"
	"strings"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	repo    appports.GoodsTypeRepository
	plugins appports.PluginInstallationRepository
}

func NewService(repo appports.GoodsTypeRepository, plugins appports.PluginInstallationRepository) *Service {
	return &Service{repo: repo, plugins: plugins}
}

func (s *Service) List(ctx context.Context) ([]domain.GoodsType, error) {
	if s.repo == nil {
		return nil, appshared.ErrInvalidInput
	}
	return s.repo.ListGoodsTypes(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (domain.GoodsType, error) {
	if s.repo == nil || id <= 0 {
		return domain.GoodsType{}, appshared.ErrInvalidInput
	}
	return s.repo.GetGoodsType(ctx, id)
}

func (s *Service) Create(ctx context.Context, gt *domain.GoodsType) error {
	if s.repo == nil || gt == nil {
		return appshared.ErrInvalidInput
	}
	gt.Name = strings.TrimSpace(gt.Name)
	gt.Code = strings.TrimSpace(gt.Code)
	gt.AutomationCategory = strings.TrimSpace(gt.AutomationCategory)
	gt.AutomationPluginID = strings.TrimSpace(gt.AutomationPluginID)
	gt.AutomationInstanceID = strings.TrimSpace(gt.AutomationInstanceID)
	if gt.Name == "" {
		return appshared.ErrInvalidInput
	}
	if gt.AutomationCategory == "" {
		gt.AutomationCategory = "automation"
	}
	if gt.AutomationCategory != "automation" || gt.AutomationPluginID == "" || gt.AutomationInstanceID == "" {
		return domain.ErrAutomationBindingInvalid
	}
	if s.plugins != nil {
		if _, err := s.plugins.GetPluginInstallation(ctx, gt.AutomationCategory, gt.AutomationPluginID, gt.AutomationInstanceID); err != nil {
			return domain.ErrAutomationPluginInstanceNotFound
		}
	}
	return s.repo.CreateGoodsType(ctx, gt)
}

func (s *Service) Update(ctx context.Context, gt domain.GoodsType) error {
	if s.repo == nil {
		return appshared.ErrInvalidInput
	}
	gt.Name = strings.TrimSpace(gt.Name)
	gt.Code = strings.TrimSpace(gt.Code)
	gt.AutomationCategory = strings.TrimSpace(gt.AutomationCategory)
	gt.AutomationPluginID = strings.TrimSpace(gt.AutomationPluginID)
	gt.AutomationInstanceID = strings.TrimSpace(gt.AutomationInstanceID)
	if gt.ID <= 0 || gt.Name == "" {
		return appshared.ErrInvalidInput
	}
	if gt.AutomationCategory == "" {
		gt.AutomationCategory = "automation"
	}
	if gt.AutomationCategory != "automation" || gt.AutomationPluginID == "" || gt.AutomationInstanceID == "" {
		return domain.ErrAutomationBindingInvalid
	}
	if s.plugins != nil {
		if _, err := s.plugins.GetPluginInstallation(ctx, gt.AutomationCategory, gt.AutomationPluginID, gt.AutomationInstanceID); err != nil {
			return domain.ErrAutomationPluginInstanceNotFound
		}
	}
	return s.repo.UpdateGoodsType(ctx, gt)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if s.repo == nil || id <= 0 {
		return appshared.ErrInvalidInput
	}
	return s.repo.DeleteGoodsType(ctx, id)
}
