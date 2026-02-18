package automation

import (
	"context"
	"strings"

	"fmt"
	"xiaoheiplay/internal/adapter/plugins/core"
	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type Resolver struct {
	goodsTypes appports.GoodsTypeRepository
	pluginMgr  *plugins.Manager
	settings   appports.SettingsRepository
	autoLogs   appports.AutomationLogRepository
}

func NewResolver(goodsTypes appports.GoodsTypeRepository, pluginMgr *plugins.Manager, settings appports.SettingsRepository, autoLogs appports.AutomationLogRepository) *Resolver {
	return &Resolver{goodsTypes: goodsTypes, pluginMgr: pluginMgr, settings: settings, autoLogs: autoLogs}
}

func (r *Resolver) ClientForGoodsType(ctx context.Context, goodsTypeID int64) (appshared.AutomationClient, error) {
	if r.goodsTypes == nil {
		return nil, fmt.Errorf("goods type repo missing")
	}
	if goodsTypeID <= 0 {
		items, err := r.goodsTypes.ListGoodsTypes(ctx)
		if err != nil {
			return nil, fmt.Errorf("goods_type_id required")
		}
		def := DefaultGoodsType(items)
		if def == nil || def.ID <= 0 {
			return nil, fmt.Errorf("goods_type_id required")
		}
		goodsTypeID = def.ID
	}
	gt, err := r.goodsTypes.GetGoodsType(ctx, goodsTypeID)
	if err != nil {
		return nil, err
	}
	cat := strings.TrimSpace(gt.AutomationCategory)
	if cat == "" {
		cat = "automation"
	}
	if cat != "automation" || strings.TrimSpace(gt.AutomationPluginID) == "" || strings.TrimSpace(gt.AutomationInstanceID) == "" {
		return nil, fmt.Errorf("invalid automation binding")
	}
	if r.pluginMgr == nil {
		return nil, fmt.Errorf("plugin manager missing")
	}
	return NewPluginInstanceClient(r.pluginMgr, gt.AutomationPluginID, gt.AutomationInstanceID, r.settings, r.autoLogs), nil
}

func DefaultGoodsType(items []domain.GoodsType) *domain.GoodsType {
	if len(items) == 0 {
		return nil
	}
	// Prefer the lowest sort_order then id (same as SQL).
	best := items[0]
	for _, it := range items[1:] {
		if it.SortOrder < best.SortOrder || (it.SortOrder == best.SortOrder && it.ID < best.ID) {
			best = it
		}
	}
	return &best
}
