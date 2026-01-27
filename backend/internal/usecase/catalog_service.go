package usecase

import (
	"context"
	"strings"

	"xiaoheiplay/internal/domain"
)

type CatalogService struct {
	catalog CatalogRepository
	images  SystemImageRepository
	billing BillingCycleRepository
}

func NewCatalogService(repo CatalogRepository, images SystemImageRepository, billing BillingCycleRepository) *CatalogService {
	return &CatalogService{catalog: repo, images: images, billing: billing}
}

func (s *CatalogService) Catalog(ctx context.Context) ([]domain.Region, []domain.PlanGroup, []domain.Package, []domain.SystemImage, []domain.BillingCycle, error) {
	regions, err := s.catalog.ListRegions(ctx)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	plans, err := s.catalog.ListPlanGroups(ctx)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	packages, err := s.catalog.ListPackages(ctx)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	images, err := s.images.ListAllSystemImages(ctx)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	var cycles []domain.BillingCycle
	if s.billing != nil {
		cycles, err = s.billing.ListBillingCycles(ctx)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}
	return regions, plans, packages, images, cycles, nil
}

func (s *CatalogService) ListRegions(ctx context.Context) ([]domain.Region, error) {
	return s.catalog.ListRegions(ctx)
}

func (s *CatalogService) ListPlanGroups(ctx context.Context) ([]domain.PlanGroup, error) {
	return s.catalog.ListPlanGroups(ctx)
}

func (s *CatalogService) GetRegion(ctx context.Context, id int64) (domain.Region, error) {
	return s.catalog.GetRegion(ctx, id)
}

func (s *CatalogService) GetPlanGroup(ctx context.Context, id int64) (domain.PlanGroup, error) {
	return s.catalog.GetPlanGroup(ctx, id)
}

func (s *CatalogService) ListPackages(ctx context.Context) ([]domain.Package, error) {
	return s.catalog.ListPackages(ctx)
}

func (s *CatalogService) GetPackage(ctx context.Context, id int64) (domain.Package, error) {
	return s.catalog.GetPackage(ctx, id)
}

func (s *CatalogService) CreateRegion(ctx context.Context, region *domain.Region) error {
	return s.catalog.CreateRegion(ctx, region)
}

func (s *CatalogService) UpdateRegion(ctx context.Context, region domain.Region) error {
	return s.catalog.UpdateRegion(ctx, region)
}

func (s *CatalogService) DeleteRegion(ctx context.Context, id int64) error {
	return s.catalog.DeleteRegion(ctx, id)
}

func (s *CatalogService) CreatePlanGroup(ctx context.Context, plan *domain.PlanGroup) error {
	return s.catalog.CreatePlanGroup(ctx, plan)
}

func (s *CatalogService) UpdatePlanGroup(ctx context.Context, plan domain.PlanGroup) error {
	return s.catalog.UpdatePlanGroup(ctx, plan)
}

func (s *CatalogService) DeletePlanGroup(ctx context.Context, id int64) error {
	return s.catalog.DeletePlanGroup(ctx, id)
}

func (s *CatalogService) CreatePackage(ctx context.Context, pkg *domain.Package) error {
	if pkg.PlanGroupID <= 0 {
		return ErrInvalidInput
	}
	return s.catalog.CreatePackage(ctx, pkg)
}

func (s *CatalogService) UpdatePackage(ctx context.Context, pkg domain.Package) error {
	if pkg.PlanGroupID <= 0 {
		return ErrInvalidInput
	}
	return s.catalog.UpdatePackage(ctx, pkg)
}

func (s *CatalogService) DeletePackage(ctx context.Context, id int64) error {
	return s.catalog.DeletePackage(ctx, id)
}

func (s *CatalogService) ListSystemImages(ctx context.Context, lineID int64) ([]domain.SystemImage, error) {
	if lineID == 0 {
		return s.images.ListAllSystemImages(ctx)
	}
	return s.images.ListSystemImages(ctx, lineID)
}

func (s *CatalogService) ListBillingCycles(ctx context.Context) ([]domain.BillingCycle, error) {
	if s.billing == nil {
		return nil, nil
	}
	return s.billing.ListBillingCycles(ctx)
}

func (s *CatalogService) CreateBillingCycle(ctx context.Context, cycle *domain.BillingCycle) error {
	if s.billing == nil {
		return ErrInvalidInput
	}
	return s.billing.CreateBillingCycle(ctx, cycle)
}

func (s *CatalogService) UpdateBillingCycle(ctx context.Context, cycle domain.BillingCycle) error {
	if s.billing == nil {
		return ErrInvalidInput
	}
	return s.billing.UpdateBillingCycle(ctx, cycle)
}

func (s *CatalogService) DeleteBillingCycle(ctx context.Context, id int64) error {
	if s.billing == nil {
		return ErrInvalidInput
	}
	return s.billing.DeleteBillingCycle(ctx, id)
}

func (s *CatalogService) CreateSystemImage(ctx context.Context, img *domain.SystemImage) error {
	if err := validateSystemImageType(img.Type); err != nil {
		return err
	}
	return s.images.CreateSystemImage(ctx, img)
}

func (s *CatalogService) UpdateSystemImage(ctx context.Context, img domain.SystemImage) error {
	if err := validateSystemImageType(img.Type); err != nil {
		return err
	}
	return s.images.UpdateSystemImage(ctx, img)
}

func (s *CatalogService) DeleteSystemImage(ctx context.Context, id int64) error {
	return s.images.DeleteSystemImage(ctx, id)
}

func (s *CatalogService) SetLineSystemImages(ctx context.Context, lineID int64, systemImageIDs []int64) error {
	if lineID <= 0 {
		return ErrInvalidInput
	}
	return s.images.SetLineSystemImages(ctx, lineID, systemImageIDs)
}

func (s *CatalogService) GetSystemImage(ctx context.Context, id int64) (domain.SystemImage, error) {
	return s.images.GetSystemImage(ctx, id)
}

func validateSystemImageType(t string) error {
	switch strings.ToLower(t) {
	case "", "linux", "windows":
		return nil
	default:
		return ErrInvalidInput
	}
}
