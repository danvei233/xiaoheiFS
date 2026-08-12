package catalog

import (
	"context"
	"strings"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	catalog appports.CatalogRepository
	images  appports.SystemImageRepository
	billing appports.BillingCycleRepository
	bulk    appports.CatalogBulkDeleteRepository
}

func NewService(repo appports.CatalogRepository, images appports.SystemImageRepository, billing appports.BillingCycleRepository) *Service {
	bulk, _ := repo.(appports.CatalogBulkDeleteRepository)
	return &Service{catalog: repo, images: images, billing: billing, bulk: bulk}
}

func (s *Service) Catalog(ctx context.Context) ([]domain.Region, []domain.PlanGroup, []domain.Package, []domain.SystemImage, []domain.BillingCycle, error) {
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

func (s *Service) ListRegions(ctx context.Context) ([]domain.Region, error) {
	return s.catalog.ListRegions(ctx)
}

func (s *Service) ListPlanGroups(ctx context.Context) ([]domain.PlanGroup, error) {
	return s.catalog.ListPlanGroups(ctx)
}

func (s *Service) SetRegionActive(ctx context.Context, id int64, active bool) error {
	return s.catalog.SetRegionActive(ctx, id, active)
}

func (s *Service) GetRegion(ctx context.Context, id int64) (domain.Region, error) {
	return s.catalog.GetRegion(ctx, id)
}

func (s *Service) GetPlanGroup(ctx context.Context, id int64) (domain.PlanGroup, error) {
	return s.catalog.GetPlanGroup(ctx, id)
}

func (s *Service) ListPackages(ctx context.Context) ([]domain.Package, error) {
	return s.catalog.ListPackages(ctx)
}

func (s *Service) GetPackage(ctx context.Context, id int64) (domain.Package, error) {
	return s.catalog.GetPackage(ctx, id)
}

func (s *Service) CreateRegion(ctx context.Context, region *domain.Region) error {
	return s.catalog.CreateRegion(ctx, region)
}

func (s *Service) UpdateRegion(ctx context.Context, region domain.Region) error {
	return s.catalog.UpdateRegion(ctx, region)
}

func (s *Service) DeleteRegion(ctx context.Context, id int64) error {
	return s.catalog.DeleteRegion(ctx, id)
}

func (s *Service) BulkDeleteRegions(ctx context.Context, ids []int64) error {
	if s.bulk == nil {
		return appshared.ErrNotSupported
	}
	return s.bulk.BulkDeleteRegions(ctx, ids)
}

func (s *Service) CreatePlanGroup(ctx context.Context, plan *domain.PlanGroup) error {
	return s.catalog.CreatePlanGroup(ctx, plan)
}

func (s *Service) UpdatePlanGroup(ctx context.Context, plan domain.PlanGroup) error {
	return s.catalog.UpdatePlanGroup(ctx, plan)
}

func (s *Service) DeletePlanGroup(ctx context.Context, id int64) error {
	return s.catalog.DeletePlanGroup(ctx, id)
}

func (s *Service) BulkDeletePlanGroups(ctx context.Context, ids []int64) error {
	if s.bulk == nil {
		return appshared.ErrNotSupported
	}
	return s.bulk.BulkDeletePlanGroups(ctx, ids)
}

func (s *Service) CreatePackage(ctx context.Context, pkg *domain.Package) error {
	if pkg.PlanGroupID <= 0 || pkg.Monthly < 0 {
		return appshared.ErrInvalidInput
	}
	return s.catalog.CreatePackage(ctx, pkg)
}

func (s *Service) UpdatePackage(ctx context.Context, pkg domain.Package) error {
	if pkg.PlanGroupID <= 0 || pkg.Monthly < 0 {
		return appshared.ErrInvalidInput
	}
	return s.catalog.UpdatePackage(ctx, pkg)
}

func (s *Service) DeletePackage(ctx context.Context, id int64) error {
	return s.catalog.DeletePackage(ctx, id)
}

func (s *Service) BulkDeletePackages(ctx context.Context, ids []int64) error {
	if s.bulk == nil {
		return appshared.ErrNotSupported
	}
	return s.bulk.BulkDeletePackages(ctx, ids)
}

func (s *Service) ListSystemImages(ctx context.Context, lineID int64) ([]domain.SystemImage, error) {
	if lineID == 0 {
		return s.images.ListAllSystemImages(ctx)
	}
	return s.images.ListSystemImages(ctx, lineID)
}

// ListSystemImagesForLines returns the enabled system images scoped to the union
// of the given lines in a single batch query. Any repository failure propagates
// to the caller: the public catalog must not silently return a truncated image
// set when one of the line queries fails.
func (s *Service) ListSystemImagesForLines(ctx context.Context, lineIDs []int64) ([]domain.SystemImage, error) {
	if len(lineIDs) == 0 {
		return []domain.SystemImage{}, nil
	}
	deduped := make([]int64, 0, len(lineIDs))
	seen := make(map[int64]struct{}, len(lineIDs))
	for _, id := range lineIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		deduped = append(deduped, id)
	}
	if len(deduped) == 0 {
		return []domain.SystemImage{}, nil
	}
	return s.images.ListSystemImagesByLineIDs(ctx, deduped)
}

func (s *Service) ListBillingCycles(ctx context.Context) ([]domain.BillingCycle, error) {
	if s.billing == nil {
		return nil, nil
	}
	return s.billing.ListBillingCycles(ctx)
}

func (s *Service) CreateBillingCycle(ctx context.Context, cycle *domain.BillingCycle) error {
	if s.billing == nil {
		return appshared.ErrInvalidInput
	}
	if err := validateBillingCycle(*cycle); err != nil {
		return err
	}
	return s.billing.CreateBillingCycle(ctx, cycle)
}

func (s *Service) UpdateBillingCycle(ctx context.Context, cycle domain.BillingCycle) error {
	if s.billing == nil {
		return appshared.ErrInvalidInput
	}
	if err := validateBillingCycle(cycle); err != nil {
		return err
	}
	return s.billing.UpdateBillingCycle(ctx, cycle)
}

func validateBillingCycle(cycle domain.BillingCycle) error {
	if cycle.Months <= 0 || cycle.Multiplier <= 0 {
		return appshared.ErrInvalidInput
	}
	if cycle.MinQty < 0 || cycle.MaxQty < 0 {
		return appshared.ErrInvalidInput
	}
	if cycle.MaxQty > 0 && cycle.MinQty > cycle.MaxQty {
		return appshared.ErrInvalidInput
	}
	return nil
}

func (s *Service) DeleteBillingCycle(ctx context.Context, id int64) error {
	if s.billing == nil {
		return appshared.ErrInvalidInput
	}
	return s.billing.DeleteBillingCycle(ctx, id)
}

func (s *Service) BulkDeleteBillingCycles(ctx context.Context, ids []int64) error {
	if s.bulk == nil {
		return appshared.ErrNotSupported
	}
	return s.bulk.BulkDeleteBillingCycles(ctx, ids)
}

func (s *Service) CreateSystemImage(ctx context.Context, img *domain.SystemImage) error {
	if err := validateSystemImageType(img.Type); err != nil {
		return err
	}
	return s.images.CreateSystemImage(ctx, img)
}

func (s *Service) UpdateSystemImage(ctx context.Context, img domain.SystemImage) error {
	if err := validateSystemImageType(img.Type); err != nil {
		return err
	}
	return s.images.UpdateSystemImage(ctx, img)
}

func (s *Service) DeleteSystemImage(ctx context.Context, id int64) error {
	return s.images.DeleteSystemImage(ctx, id)
}

func (s *Service) BulkDeleteSystemImages(ctx context.Context, ids []int64) error {
	if s.bulk == nil {
		return appshared.ErrNotSupported
	}
	return s.bulk.BulkDeleteSystemImages(ctx, ids)
}

func (s *Service) SetLineSystemImages(ctx context.Context, lineID int64, systemImageIDs []int64) error {
	if lineID <= 0 {
		return appshared.ErrInvalidInput
	}
	return s.images.SetLineSystemImages(ctx, lineID, systemImageIDs)
}

func (s *Service) ListImageLineNames(ctx context.Context, imageIDs []int64) (map[int64][]string, error) {
	if len(imageIDs) == 0 {
		return map[int64][]string{}, nil
	}
	return s.images.ListImageLineNames(ctx, imageIDs)
}

func (s *Service) GetSystemImage(ctx context.Context, id int64) (domain.SystemImage, error) {
	return s.images.GetSystemImage(ctx, id)
}

func validateSystemImageType(t string) error {
	switch strings.ToLower(t) {
	case "", "linux", "windows", "macos":
		return nil
	default:
		return appshared.ErrInvalidInput
	}
}
