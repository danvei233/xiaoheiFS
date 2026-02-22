package repo

import (
	"context"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) ListGoodsTypes(ctx context.Context) ([]domain.GoodsType, error) {

	var rows []goodsTypeRow
	if err := r.gdb.WithContext(ctx).Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.GoodsType, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.GoodsType{
			ID:                   row.ID,
			Code:                 row.Code,
			Name:                 row.Name,
			Active:               row.Active == 1,
			SortOrder:            row.SortOrder,
			AutomationCategory:   row.AutomationCategory,
			AutomationPluginID:   row.AutomationPluginID,
			AutomationInstanceID: row.AutomationInstanceID,
			CreatedAt:            row.CreatedAt,
			UpdatedAt:            row.UpdatedAt,
		})
	}
	return out, nil

}

func (r *GormRepo) GetGoodsType(ctx context.Context, id int64) (domain.GoodsType, error) {

	var row goodsTypeRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.GoodsType{}, r.ensure(err)
	}
	return domain.GoodsType{
		ID:                   row.ID,
		Code:                 row.Code,
		Name:                 row.Name,
		Active:               row.Active == 1,
		SortOrder:            row.SortOrder,
		AutomationCategory:   row.AutomationCategory,
		AutomationPluginID:   row.AutomationPluginID,
		AutomationInstanceID: row.AutomationInstanceID,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}, nil

}

func (r *GormRepo) CreateGoodsType(ctx context.Context, gt *domain.GoodsType) error {

	row := goodsTypeRow{
		Code:                 strings.TrimSpace(gt.Code),
		Name:                 gt.Name,
		Active:               boolToInt(gt.Active),
		SortOrder:            gt.SortOrder,
		AutomationCategory:   gt.AutomationCategory,
		AutomationPluginID:   gt.AutomationPluginID,
		AutomationInstanceID: gt.AutomationInstanceID,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	gt.ID = row.ID
	gt.CreatedAt = row.CreatedAt
	gt.UpdatedAt = row.UpdatedAt
	return nil

}

func (r *GormRepo) UpdateGoodsType(ctx context.Context, gt domain.GoodsType) error {

	return r.gdb.WithContext(ctx).Model(&goodsTypeRow{}).Where("id = ?", gt.ID).Updates(map[string]any{
		"code":                   strings.TrimSpace(gt.Code),
		"name":                   gt.Name,
		"active":                 boolToInt(gt.Active),
		"sort_order":             gt.SortOrder,
		"automation_category":    gt.AutomationCategory,
		"automation_plugin_id":   gt.AutomationPluginID,
		"automation_instance_id": gt.AutomationInstanceID,
		"updated_at":             time.Now(),
	}).Error

}

func (r *GormRepo) DeleteGoodsType(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&goodsTypeRow{}, id).Error

}

func (r *GormRepo) ListRegions(ctx context.Context) ([]domain.Region, error) {

	var rows []regionRow
	if err := r.gdb.WithContext(ctx).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Region, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Region{
			ID:          row.ID,
			GoodsTypeID: row.GoodsTypeID,
			Code:        row.Code,
			Name:        row.Name,
			Active:      row.Active == 1,
		})
	}
	return out, nil

}

func (r *GormRepo) CreateRegion(ctx context.Context, region *domain.Region) error {

	row := regionRow{
		GoodsTypeID: region.GoodsTypeID,
		Code:        region.Code,
		Name:        region.Name,
		Active:      boolToInt(region.Active),
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	region.ID = row.ID
	return nil

}

func (r *GormRepo) UpdateRegion(ctx context.Context, region domain.Region) error {

	return r.gdb.WithContext(ctx).Model(&regionRow{}).Where("id = ?", region.ID).Updates(map[string]any{
		"goods_type_id": region.GoodsTypeID,
		"code":          region.Code,
		"name":          region.Name,
		"active":        boolToInt(region.Active),
		"updated_at":    time.Now(),
	}).Error

}

func (r *GormRepo) DeleteRegion(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&regionRow{}, id).Error

}

func (r *GormRepo) ListPlanGroups(ctx context.Context) ([]domain.PlanGroup, error) {

	var rows []planGroupRow
	if err := r.gdb.WithContext(ctx).Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PlanGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.PlanGroup{
			ID:                row.ID,
			GoodsTypeID:       row.GoodsTypeID,
			RegionID:          row.RegionID,
			Name:              row.Name,
			LineID:            row.LineID,
			UnitCore:          row.UnitCore,
			UnitMem:           row.UnitMem,
			UnitDisk:          row.UnitDisk,
			UnitBW:            row.UnitBW,
			AddCoreMin:        row.AddCoreMin,
			AddCoreMax:        row.AddCoreMax,
			AddCoreStep:       row.AddCoreStep,
			AddMemMin:         row.AddMemMin,
			AddMemMax:         row.AddMemMax,
			AddMemStep:        row.AddMemStep,
			AddDiskMin:        row.AddDiskMin,
			AddDiskMax:        row.AddDiskMax,
			AddDiskStep:       row.AddDiskStep,
			AddBWMin:          row.AddBWMin,
			AddBWMax:          row.AddBWMax,
			AddBWStep:         row.AddBWStep,
			Active:            row.Active == 1,
			Visible:           row.Visible == 1,
			CapacityRemaining: row.CapacityRemaining,
			SortOrder:         row.SortOrder,
		})
	}
	return out, nil

}

func (r *GormRepo) CreatePlanGroup(ctx context.Context, plan *domain.PlanGroup) error {

	row := planGroupRow{
		GoodsTypeID:       plan.GoodsTypeID,
		RegionID:          plan.RegionID,
		Name:              plan.Name,
		LineID:            plan.LineID,
		UnitCore:          plan.UnitCore,
		UnitMem:           plan.UnitMem,
		UnitDisk:          plan.UnitDisk,
		UnitBW:            plan.UnitBW,
		AddCoreMin:        plan.AddCoreMin,
		AddCoreMax:        plan.AddCoreMax,
		AddCoreStep:       plan.AddCoreStep,
		AddMemMin:         plan.AddMemMin,
		AddMemMax:         plan.AddMemMax,
		AddMemStep:        plan.AddMemStep,
		AddDiskMin:        plan.AddDiskMin,
		AddDiskMax:        plan.AddDiskMax,
		AddDiskStep:       plan.AddDiskStep,
		AddBWMin:          plan.AddBWMin,
		AddBWMax:          plan.AddBWMax,
		AddBWStep:         plan.AddBWStep,
		Active:            boolToInt(plan.Active),
		Visible:           boolToInt(plan.Visible),
		CapacityRemaining: plan.CapacityRemaining,
		SortOrder:         plan.SortOrder,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	plan.ID = row.ID
	return nil

}

func (r *GormRepo) UpdatePlanGroup(ctx context.Context, plan domain.PlanGroup) error {

	return r.gdb.WithContext(ctx).Model(&planGroupRow{}).Where("id = ?", plan.ID).Updates(map[string]any{
		"goods_type_id":      plan.GoodsTypeID,
		"region_id":          plan.RegionID,
		"name":               plan.Name,
		"line_id":            plan.LineID,
		"unit_core":          plan.UnitCore,
		"unit_mem":           plan.UnitMem,
		"unit_disk":          plan.UnitDisk,
		"unit_bw":            plan.UnitBW,
		"add_core_min":       plan.AddCoreMin,
		"add_core_max":       plan.AddCoreMax,
		"add_core_step":      plan.AddCoreStep,
		"add_mem_min":        plan.AddMemMin,
		"add_mem_max":        plan.AddMemMax,
		"add_mem_step":       plan.AddMemStep,
		"add_disk_min":       plan.AddDiskMin,
		"add_disk_max":       plan.AddDiskMax,
		"add_disk_step":      plan.AddDiskStep,
		"add_bw_min":         plan.AddBWMin,
		"add_bw_max":         plan.AddBWMax,
		"add_bw_step":        plan.AddBWStep,
		"active":             boolToInt(plan.Active),
		"visible":            boolToInt(plan.Visible),
		"capacity_remaining": plan.CapacityRemaining,
		"sort_order":         plan.SortOrder,
		"updated_at":         time.Now(),
	}).Error

}

func (r *GormRepo) DeletePlanGroup(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&planGroupRow{}, id).Error

}

func (r *GormRepo) ListPackages(ctx context.Context) ([]domain.Package, error) {

	var rows []packageRow
	if err := r.gdb.WithContext(ctx).Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Package, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Package{
			ID:                   row.ID,
			GoodsTypeID:          row.GoodsTypeID,
			PlanGroupID:          row.PlanGroupID,
			ProductID:            row.ProductID,
			IntegrationPackageID: row.IntegrationPackageID,
			Name:                 row.Name,
			Cores:                row.Cores,
			MemoryGB:             row.MemoryGB,
			DiskGB:               row.DiskGB,
			BandwidthMB:          row.BandwidthMbps,
			CPUModel:             row.CPUModel,
			Monthly:              row.MonthlyPrice,
			PortNum:              row.PortNum,
			SortOrder:            row.SortOrder,
			Active:               row.Active == 1,
			Visible:              row.Visible == 1,
			CapacityRemaining:    row.CapacityRemaining,
		})
	}
	return out, nil

}

func (r *GormRepo) CreatePackage(ctx context.Context, pkg *domain.Package) error {

	row := packageRow{
		GoodsTypeID:          pkg.GoodsTypeID,
		PlanGroupID:          pkg.PlanGroupID,
		ProductID:            pkg.ProductID,
		IntegrationPackageID: pkg.IntegrationPackageID,
		Name:                 pkg.Name,
		Cores:                pkg.Cores,
		MemoryGB:             pkg.MemoryGB,
		DiskGB:               pkg.DiskGB,
		BandwidthMbps:        pkg.BandwidthMB,
		CPUModel:             pkg.CPUModel,
		MonthlyPrice:         pkg.Monthly,
		PortNum:              pkg.PortNum,
		SortOrder:            pkg.SortOrder,
		Active:               boolToInt(pkg.Active),
		Visible:              boolToInt(pkg.Visible),
		CapacityRemaining:    pkg.CapacityRemaining,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	pkg.ID = row.ID
	return nil

}

func (r *GormRepo) UpdatePackage(ctx context.Context, pkg domain.Package) error {

	return r.gdb.WithContext(ctx).Model(&packageRow{}).Where("id = ?", pkg.ID).Updates(map[string]any{
		"goods_type_id":          pkg.GoodsTypeID,
		"plan_group_id":          pkg.PlanGroupID,
		"product_id":             pkg.ProductID,
		"integration_package_id": pkg.IntegrationPackageID,
		"name":                   pkg.Name,
		"cores":                  pkg.Cores,
		"memory_gb":              pkg.MemoryGB,
		"disk_gb":                pkg.DiskGB,
		"bandwidth_mbps":         pkg.BandwidthMB,
		"cpu_model":              pkg.CPUModel,
		"monthly_price":          pkg.Monthly,
		"port_num":               pkg.PortNum,
		"sort_order":             pkg.SortOrder,
		"active":                 boolToInt(pkg.Active),
		"visible":                boolToInt(pkg.Visible),
		"capacity_remaining":     pkg.CapacityRemaining,
		"updated_at":             time.Now(),
	}).Error

}

func (r *GormRepo) DeletePackage(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&packageRow{}, id).Error

}

func (r *GormRepo) GetPackage(ctx context.Context, id int64) (domain.Package, error) {

	var row packageRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.Package{}, r.ensure(err)
	}
	return domain.Package{
		ID:                   row.ID,
		GoodsTypeID:          row.GoodsTypeID,
		PlanGroupID:          row.PlanGroupID,
		ProductID:            row.ProductID,
		IntegrationPackageID: row.IntegrationPackageID,
		Name:                 row.Name,
		Cores:                row.Cores,
		MemoryGB:             row.MemoryGB,
		DiskGB:               row.DiskGB,
		BandwidthMB:          row.BandwidthMbps,
		CPUModel:             row.CPUModel,
		Monthly:              row.MonthlyPrice,
		PortNum:              row.PortNum,
		SortOrder:            row.SortOrder,
		Active:               row.Active == 1,
		Visible:              row.Visible == 1,
		CapacityRemaining:    row.CapacityRemaining,
	}, nil

}

func (r *GormRepo) GetPlanGroup(ctx context.Context, id int64) (domain.PlanGroup, error) {

	var row planGroupRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.PlanGroup{}, r.ensure(err)
	}
	return domain.PlanGroup{
		ID:                row.ID,
		GoodsTypeID:       row.GoodsTypeID,
		RegionID:          row.RegionID,
		Name:              row.Name,
		LineID:            row.LineID,
		UnitCore:          row.UnitCore,
		UnitMem:           row.UnitMem,
		UnitDisk:          row.UnitDisk,
		UnitBW:            row.UnitBW,
		AddCoreMin:        row.AddCoreMin,
		AddCoreMax:        row.AddCoreMax,
		AddCoreStep:       row.AddCoreStep,
		AddMemMin:         row.AddMemMin,
		AddMemMax:         row.AddMemMax,
		AddMemStep:        row.AddMemStep,
		AddDiskMin:        row.AddDiskMin,
		AddDiskMax:        row.AddDiskMax,
		AddDiskStep:       row.AddDiskStep,
		AddBWMin:          row.AddBWMin,
		AddBWMax:          row.AddBWMax,
		AddBWStep:         row.AddBWStep,
		Active:            row.Active == 1,
		Visible:           row.Visible == 1,
		CapacityRemaining: row.CapacityRemaining,
		SortOrder:         row.SortOrder,
	}, nil

}

func (r *GormRepo) GetRegion(ctx context.Context, id int64) (domain.Region, error) {

	var row regionRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.Region{}, r.ensure(err)
	}
	return domain.Region{
		ID:          row.ID,
		GoodsTypeID: row.GoodsTypeID,
		Code:        row.Code,
		Name:        row.Name,
		Active:      row.Active == 1,
	}, nil

}
