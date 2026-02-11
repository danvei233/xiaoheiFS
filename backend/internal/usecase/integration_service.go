package usecase

import (
	"context"
	"fmt"
	"strings"

	"xiaoheiplay/internal/domain"
)

type AutomationConfig struct {
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key"`
	Enabled    bool   `json:"enabled"`
	TimeoutSec int    `json:"timeout_sec"`
	Retry      int    `json:"retry"`
	DryRun     bool   `json:"dry_run"`
}

type IntegrationService struct {
	settings   SettingsRepository
	catalog    CatalogRepository
	images     SystemImageRepository
	goodsTypes GoodsTypeRepository
	automation AutomationClientResolver
	logs       IntegrationLogRepository
}

type SyncResult struct {
	Lines    int `json:"lines"`
	Products int `json:"products"`
	Images   int `json:"images"`
}

func NewIntegrationService(settings SettingsRepository, catalog CatalogRepository, images SystemImageRepository, goodsTypes GoodsTypeRepository, automation AutomationClientResolver, logs IntegrationLogRepository) *IntegrationService {
	return &IntegrationService{settings: settings, catalog: catalog, images: images, goodsTypes: goodsTypes, automation: automation, logs: logs}
}

func (s *IntegrationService) SyncAutomation(ctx context.Context, mode string) (SyncResult, error) {
	if s.goodsTypes == nil {
		return SyncResult{}, ErrInvalidInput
	}
	items, err := s.goodsTypes.ListGoodsTypes(ctx)
	if err != nil || len(items) == 0 {
		return SyncResult{}, ErrInvalidInput
	}
	def := items[0]
	for _, it := range items[1:] {
		if it.SortOrder < def.SortOrder || (it.SortOrder == def.SortOrder && it.ID < def.ID) {
			def = it
		}
	}
	return s.SyncAutomationForGoodsType(ctx, def.ID, mode)
}

func (s *IntegrationService) SyncAutomationImagesForLine(ctx context.Context, lineID int64, mode string) (int, error) {
	if s.automation == nil || s.catalog == nil || s.images == nil || lineID <= 0 {
		return 0, ErrInvalidInput
	}
	if mode == "" {
		mode = "merge"
	}
	plans, err := s.catalog.ListPlanGroups(ctx)
	if err != nil {
		return 0, err
	}
	var goodsTypeID int64
	var bestPlanID int64
	for _, plan := range plans {
		if plan.LineID != lineID || plan.GoodsTypeID <= 0 {
			continue
		}
		if goodsTypeID == 0 || plan.ID < bestPlanID {
			goodsTypeID = plan.GoodsTypeID
			bestPlanID = plan.ID
		}
	}
	if goodsTypeID <= 0 {
		plan, err := s.catalog.GetPlanGroup(ctx, lineID)
		if err == nil && plan.GoodsTypeID > 0 && plan.LineID > 0 {
			goodsTypeID = plan.GoodsTypeID
			lineID = plan.LineID
		}
	}
	if goodsTypeID <= 0 {
		return 0, fmt.Errorf("line_id %d not bound to any goods type", lineID)
	}
	cli, err := s.automation.ClientForGoodsType(ctx, goodsTypeID)
	if err != nil {
		return 0, err
	}
	knownImages, _ := s.images.ListAllSystemImages(ctx)
	imageByID := map[int64]domain.SystemImage{}
	for _, img := range knownImages {
		if img.ImageID > 0 {
			imageByID[img.ImageID] = img
		}
	}
	count, created, err := s.syncLineImages(ctx, cli, lineID, mode, imageByID)
	if err != nil {
		s.appendSyncLog(ctx, "mirror_image", mode, "failed", err.Error())
		return 0, err
	}
	s.appendSyncLog(ctx, "mirror_image", mode, "ok", fmt.Sprintf("goods_type_id=%d line_id=%d images=%d created=%d", goodsTypeID, lineID, count, created))
	return count, nil
}

func (s *IntegrationService) SyncAutomationForGoodsType(ctx context.Context, goodsTypeID int64, mode string) (SyncResult, error) {
	if s.automation == nil || s.catalog == nil || goodsTypeID <= 0 {
		return SyncResult{}, ErrInvalidInput
	}
	cli, err := s.automation.ClientForGoodsType(ctx, goodsTypeID)
	if err != nil {
		return SyncResult{}, err
	}
	if mode == "" {
		mode = "merge"
	}
	lines, err := cli.ListLines(ctx)
	if err != nil {
		s.appendSyncLog(ctx, "line", mode, "failed", err.Error())
		return SyncResult{}, err
	}
	areas, err := cli.ListAreas(ctx)
	if err != nil {
		// Some upstream implementations do not provide a standalone area endpoint.
		// Continue sync by deriving minimal area identities from lines.
		s.appendSyncLog(ctx, "area", mode, "warn", "area endpoint unavailable; fallback to line-derived areas: "+err.Error())
		areas = []AutomationArea{}
	}

	allRegions, _ := s.catalog.ListRegions(ctx)
	allPlanGroups, _ := s.catalog.ListPlanGroups(ctx)
	allPackages, _ := s.catalog.ListPackages(ctx)
	knownImages, _ := s.images.ListAllSystemImages(ctx)

	regionByCode := map[string]domain.Region{}
	regionByName := map[string]domain.Region{}
	for _, r := range allRegions {
		if r.GoodsTypeID != goodsTypeID {
			continue
		}
		regionByCode[r.Code] = r
		if r.Name != "" {
			regionByName[r.Name] = r
		}
	}
	planByLine := map[int64]domain.PlanGroup{}
	for _, p := range allPlanGroups {
		if p.GoodsTypeID != goodsTypeID {
			continue
		}
		if p.LineID > 0 {
			planByLine[p.LineID] = p
		}
	}
	packageByKey := map[string]domain.Package{}
	for _, pkg := range allPackages {
		if pkg.GoodsTypeID != goodsTypeID {
			continue
		}
		if pkg.ProductID > 0 && pkg.PlanGroupID > 0 {
			packageByKey[fmt.Sprintf("%d:%d", pkg.PlanGroupID, pkg.ProductID)] = pkg
		}
	}
	imageByID := map[int64]domain.SystemImage{}
	for _, img := range knownImages {
		if img.ImageID > 0 {
			imageByID[img.ImageID] = img
		}
	}

	var defaultRegionID int64
	for _, r := range allRegions {
		if r.GoodsTypeID == goodsTypeID {
			defaultRegionID = r.ID
			break
		}
	}

	areaNameByID := map[int64]AutomationArea{}
	for _, area := range areas {
		areaNameByID[area.ID] = area
	}

	createdLines := 0
	for _, line := range lines {
		code := fmt.Sprintf("area-%d", line.AreaID)
		area := areaNameByID[line.AreaID]
		areaName := strings.TrimSpace(area.Name)
		if areaName == "" {
			areaName = fmt.Sprintf("Area %d", line.AreaID)
		}
		region, ok := regionByCode[code]
		if !ok {
			if existing, ok := regionByName[areaName]; ok {
				existing.GoodsTypeID = goodsTypeID
				existing.Code = code
				existing.Name = areaName
				if mode == "override" {
					existing.Active = area.State == 1
				}
				_ = s.catalog.UpdateRegion(ctx, existing)
				region = existing
			} else {
				region = domain.Region{GoodsTypeID: goodsTypeID, Code: code, Name: areaName, Active: area.State == 1}
				_ = s.catalog.CreateRegion(ctx, &region)
			}
			regionByCode[code] = region
			if region.Name != "" {
				regionByName[region.Name] = region
			}
		} else if mode == "override" {
			region.GoodsTypeID = goodsTypeID
			region.Name = areaName
			region.Active = area.State == 1
			_ = s.catalog.UpdateRegion(ctx, region)
			regionByCode[code] = region
			if region.Name != "" {
				regionByName[region.Name] = region
			}
		}
		if defaultRegionID == 0 && region.ID > 0 {
			defaultRegionID = region.ID
		}
		if existing, ok := planByLine[line.ID]; ok {
			existing.GoodsTypeID = goodsTypeID
			existing.Name = line.Name
			existing.RegionID = region.ID
			if mode == "override" {
				existing.Active = line.State == 1
			}
			_ = s.catalog.UpdatePlanGroup(ctx, existing)
		} else {
			pg := domain.PlanGroup{
				GoodsTypeID:       goodsTypeID,
				RegionID:          region.ID,
				Name:              line.Name,
				LineID:            line.ID,
				UnitCore:          0,
				UnitMem:           0,
				UnitDisk:          0,
				UnitBW:            0,
				Active:            line.State == 1,
				Visible:           true,
				CapacityRemaining: -1,
				SortOrder:         0,
			}
			_ = s.catalog.CreatePlanGroup(ctx, &pg)
			planByLine[line.ID] = pg
			createdLines++
		}
	}

	if defaultRegionID == 0 {
		region := domain.Region{GoodsTypeID: goodsTypeID, Code: "default", Name: "Default", Active: true}
		_ = s.catalog.CreateRegion(ctx, &region)
		defaultRegionID = region.ID
	}

	createdProducts := 0
	for _, line := range lines {
		plan, ok := planByLine[line.ID]
		if !ok || plan.ID == 0 {
			continue
		}
		products, err := cli.ListProducts(ctx, line.ID)
		if err != nil {
			s.appendSyncLog(ctx, "product", mode, "failed", err.Error())
			return SyncResult{}, err
		}
		for _, product := range products {
			key := fmt.Sprintf("%d:%d", plan.ID, product.ID)
			if existing, ok := packageByKey[key]; ok {
				existing.GoodsTypeID = goodsTypeID
				existing.Name = product.Name
				existing.Cores = product.CPU
				existing.MemoryGB = product.MemoryGB
				existing.DiskGB = product.DiskGB
				existing.BandwidthMB = product.Bandwidth
				existing.Monthly = product.Price
				if product.PortNum > 0 {
					existing.PortNum = product.PortNum
				}
				if mode == "override" {
					existing.Active = true
				}
				_ = s.catalog.UpdatePackage(ctx, existing)
			} else {
				portNum := 30
				if product.PortNum > 0 {
					portNum = product.PortNum
				}
				pkg := domain.Package{
					GoodsTypeID:       goodsTypeID,
					PlanGroupID:       plan.ID,
					ProductID:         product.ID,
					Name:              product.Name,
					Cores:             product.CPU,
					MemoryGB:          product.MemoryGB,
					DiskGB:            product.DiskGB,
					BandwidthMB:       product.Bandwidth,
					CPUModel:          "",
					Monthly:           product.Price,
					PortNum:           portNum,
					SortOrder:         0,
					Active:            true,
					Visible:           true,
					CapacityRemaining: -1,
				}
				_ = s.catalog.CreatePackage(ctx, &pkg)
				packageByKey[key] = pkg
				createdProducts++
			}
		}
	}

	createdImages := 0
	for _, line := range lines {
		_, created, err := s.syncLineImages(ctx, cli, line.ID, mode, imageByID)
		if err != nil {
			s.appendSyncLog(ctx, "mirror_image", mode, "failed", err.Error())
			return SyncResult{}, err
		}
		createdImages += created
	}

	s.appendSyncLog(ctx, "area", mode, "ok", fmt.Sprintf("goods_type_id=%d areas=%d", goodsTypeID, len(areas)))
	s.appendSyncLog(ctx, "line", mode, "ok", fmt.Sprintf("goods_type_id=%d lines=%d", goodsTypeID, len(lines)))
	s.appendSyncLog(ctx, "product", mode, "ok", fmt.Sprintf("goods_type_id=%d products=%d", goodsTypeID, createdProducts))
	s.appendSyncLog(ctx, "mirror_image", mode, "ok", fmt.Sprintf("goods_type_id=%d images=%d", goodsTypeID, createdImages))

	return SyncResult{Lines: createdLines, Products: createdProducts, Images: createdImages}, nil
}

func (s *IntegrationService) syncLineImages(ctx context.Context, cli AutomationClient, lineID int64, mode string, imageByID map[int64]domain.SystemImage) (int, int, error) {
	images, err := cli.ListImages(ctx, lineID)
	if err != nil {
		return 0, 0, err
	}
	lineImageIDs := make([]int64, 0, len(images))
	seen := map[int64]struct{}{}
	created := 0
	for _, img := range images {
		imageType := normalizeImageType(img.Type)
		if existing, ok := imageByID[img.ImageID]; ok {
			existing.Name = img.Name
			existing.Type = imageType
			if mode == "override" {
				existing.Enabled = true
			}
			_ = s.images.UpdateSystemImage(ctx, existing)
			if _, ok := seen[existing.ID]; !ok {
				lineImageIDs = append(lineImageIDs, existing.ID)
				seen[existing.ID] = struct{}{}
			}
			continue
		}
		newImg := domain.SystemImage{ImageID: img.ImageID, Name: img.Name, Type: imageType, Enabled: true}
		_ = s.images.CreateSystemImage(ctx, &newImg)
		imageByID[img.ImageID] = newImg
		created++
		if newImg.ID > 0 {
			lineImageIDs = append(lineImageIDs, newImg.ID)
			seen[newImg.ID] = struct{}{}
		}
	}
	if err := s.images.SetLineSystemImages(ctx, lineID, lineImageIDs); err != nil {
		return 0, 0, err
	}
	return len(images), created, nil
}

func (s *IntegrationService) ListSyncLogs(ctx context.Context, target string, limit, offset int) ([]domain.IntegrationSyncLog, int, error) {
	if s.logs == nil {
		return nil, 0, nil
	}
	return s.logs.ListSyncLogs(ctx, target, limit, offset)
}

func (s *IntegrationService) appendSyncLog(ctx context.Context, target, mode, status, message string) {
	if s.logs == nil {
		return
	}
	_ = s.logs.CreateSyncLog(ctx, &domain.IntegrationSyncLog{
		Target:  target,
		Mode:    mode,
		Status:  status,
		Message: message,
	})
}

func normalizeImageType(v string) string {
	normalized := strings.ToLower(strings.TrimSpace(v))
	if strings.Contains(normalized, "win") {
		return "windows"
	}
	if normalized == "" {
		return "linux"
	}
	return "linux"
}
