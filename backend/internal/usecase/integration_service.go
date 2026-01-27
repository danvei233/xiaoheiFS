package usecase

import (
	"context"
	"fmt"
	"strconv"
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
	automation AutomationClient
	logs       IntegrationLogRepository
}

type SyncResult struct {
	Lines    int `json:"lines"`
	Products int `json:"products"`
	Images   int `json:"images"`
}

func NewIntegrationService(settings SettingsRepository, catalog CatalogRepository, images SystemImageRepository, automation AutomationClient, logs IntegrationLogRepository) *IntegrationService {
	return &IntegrationService{settings: settings, catalog: catalog, images: images, automation: automation, logs: logs}
}

func (s *IntegrationService) GetAutomationConfig(ctx context.Context) (AutomationConfig, error) {
	cfg := AutomationConfig{}
	cfg.BaseURL = getSettingValue(ctx, s.settings, "automation_base_url")
	cfg.APIKey = getSettingValue(ctx, s.settings, "automation_api_key")
	cfg.Enabled = strings.ToLower(getSettingValue(ctx, s.settings, "automation_enabled")) == "true"
	if v := getSettingValue(ctx, s.settings, "automation_timeout_sec"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.TimeoutSec = i
		}
	}
	if v := getSettingValue(ctx, s.settings, "automation_retry"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Retry = i
		}
	}
	cfg.DryRun = strings.ToLower(getSettingValue(ctx, s.settings, "automation_dry_run")) == "true"
	return cfg, nil
}

func (s *IntegrationService) UpdateAutomationConfig(ctx context.Context, adminID int64, cfg AutomationConfig) error {
	if s.settings == nil {
		return ErrInvalidInput
	}
	_ = s.settings.UpsertSetting(ctx, domain.Setting{Key: "automation_base_url", ValueJSON: cfg.BaseURL})
	_ = s.settings.UpsertSetting(ctx, domain.Setting{Key: "automation_api_key", ValueJSON: cfg.APIKey})
	_ = s.settings.UpsertSetting(ctx, domain.Setting{Key: "automation_enabled", ValueJSON: boolToString(cfg.Enabled)})
	if cfg.TimeoutSec > 0 {
		_ = s.settings.UpsertSetting(ctx, domain.Setting{Key: "automation_timeout_sec", ValueJSON: fmt.Sprintf("%d", cfg.TimeoutSec)})
	}
	if cfg.Retry >= 0 {
		_ = s.settings.UpsertSetting(ctx, domain.Setting{Key: "automation_retry", ValueJSON: fmt.Sprintf("%d", cfg.Retry)})
	}
	_ = s.settings.UpsertSetting(ctx, domain.Setting{Key: "automation_dry_run", ValueJSON: boolToString(cfg.DryRun)})
	return nil
}

func (s *IntegrationService) SyncAutomation(ctx context.Context, mode string) (SyncResult, error) {
	if s.automation == nil {
		return SyncResult{}, ErrInvalidInput
	}
	if mode == "" {
		mode = "merge"
	}
	areas, err := s.automation.ListAreas(ctx)
	if err != nil {
		s.appendSyncLog(ctx, "area", mode, "failed", err.Error())
		return SyncResult{}, err
	}
	lines, err := s.automation.ListLines(ctx)
	if err != nil {
		s.appendSyncLog(ctx, "line", mode, "failed", err.Error())
		return SyncResult{}, err
	}
	regions, _ := s.catalog.ListRegions(ctx)
	planGroups, _ := s.catalog.ListPlanGroups(ctx)
	packages, _ := s.catalog.ListPackages(ctx)
	knownImages, _ := s.images.ListAllSystemImages(ctx)

	regionByCode := map[string]domain.Region{}
	regionByName := map[string]domain.Region{}
	for _, r := range regions {
		regionByCode[r.Code] = r
		if r.Name != "" {
			regionByName[r.Name] = r
		}
	}
	planByLine := map[int64]domain.PlanGroup{}
	for _, p := range planGroups {
		if p.LineID > 0 {
			planByLine[p.LineID] = p
		}
	}
	packageByKey := map[string]domain.Package{}
	for _, pkg := range packages {
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

	// ensure default region/plan group if none exist
	var defaultRegionID int64
	if len(regions) > 0 {
		defaultRegionID = regions[0].ID
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
				existing.Code = code
				existing.Name = areaName
				if mode == "override" {
					existing.Active = area.State == 1
				}
				_ = s.catalog.UpdateRegion(ctx, existing)
				region = existing
			} else {
				region = domain.Region{Code: code, Name: areaName, Active: area.State == 1}
				_ = s.catalog.CreateRegion(ctx, &region)
			}
			regionByCode[code] = region
			if region.Name != "" {
				regionByName[region.Name] = region
			}
		} else if mode == "override" {
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
			existing.Name = line.Name
			existing.RegionID = region.ID
			if mode == "override" {
				existing.Active = line.State == 1
			}
			_ = s.catalog.UpdatePlanGroup(ctx, existing)
		} else {
			pg := domain.PlanGroup{
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
		region := domain.Region{Code: "default", Name: "Default", Active: true}
		_ = s.catalog.CreateRegion(ctx, &region)
		defaultRegionID = region.ID
	}
	createdProducts := 0
	for _, line := range lines {
		plan, ok := planByLine[line.ID]
		if !ok || plan.ID == 0 {
			continue
		}
		products, err := s.automation.ListProducts(ctx, line.ID)
		if err != nil {
			s.appendSyncLog(ctx, "product", mode, "failed", err.Error())
			return SyncResult{}, err
		}
		for _, product := range products {
			key := fmt.Sprintf("%d:%d", plan.ID, product.ID)
			if existing, ok := packageByKey[key]; ok {
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
		images, err := s.automation.ListImages(ctx, line.ID)
		if err != nil {
			s.appendSyncLog(ctx, "mirror_image", mode, "failed", err.Error())
			return SyncResult{}, err
		}
		lineImageIDs := make([]int64, 0, len(images))
		seen := map[int64]struct{}{}
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
			} else {
				newImg := domain.SystemImage{ImageID: img.ImageID, Name: img.Name, Type: imageType, Enabled: true}
				_ = s.images.CreateSystemImage(ctx, &newImg)
				imageByID[img.ImageID] = newImg
				createdImages++
				if newImg.ID > 0 {
					lineImageIDs = append(lineImageIDs, newImg.ID)
					seen[newImg.ID] = struct{}{}
				}
			}
		}
		if err := s.images.SetLineSystemImages(ctx, line.ID, lineImageIDs); err != nil {
			s.appendSyncLog(ctx, "mirror_image", mode, "failed", err.Error())
			return SyncResult{}, err
		}
	}

	s.appendSyncLog(ctx, "area", mode, "ok", fmt.Sprintf("areas=%d", len(areas)))
	s.appendSyncLog(ctx, "line", mode, "ok", fmt.Sprintf("lines=%d", len(lines)))
	s.appendSyncLog(ctx, "product", mode, "ok", fmt.Sprintf("products=%d", createdProducts))
	s.appendSyncLog(ctx, "mirror_image", mode, "ok", fmt.Sprintf("images=%d", createdImages))

	return SyncResult{Lines: createdLines, Products: createdProducts, Images: createdImages}, nil
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

func boolToString(v bool) string {
	if v {
		return "true"
	}
	return "false"
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

func getSettingValue(ctx context.Context, repo SettingsRepository, key string) string {
	if repo == nil {
		return ""
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return ""
	}
	return setting.ValueJSON
}
