package automation

import (
	"context"
	"errors"
	"strings"

	"xiaoheiplay/internal/adapter/plugins"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/usecase"
)

type Resolver struct {
	goodsTypes usecase.GoodsTypeRepository
	pluginMgr  *plugins.Manager
	fallback   usecase.AutomationClient
	settings   usecase.SettingsRepository
	autoLogs   usecase.AutomationLogRepository
}

func NewResolver(goodsTypes usecase.GoodsTypeRepository, pluginMgr *plugins.Manager, fallback usecase.AutomationClient, settings usecase.SettingsRepository, autoLogs usecase.AutomationLogRepository) *Resolver {
	return &Resolver{goodsTypes: goodsTypes, pluginMgr: pluginMgr, fallback: fallback, settings: settings, autoLogs: autoLogs}
}

func (r *Resolver) ClientForGoodsType(ctx context.Context, goodsTypeID int64) (usecase.AutomationClient, error) {
	if goodsTypeID <= 0 {
		if r.fallback != nil {
			return r.fallback, nil
		}
		return nil, errors.New("goods_type_id required")
	}
	if r.goodsTypes == nil {
		return nil, errors.New("goods type repo missing")
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
		return nil, errors.New("invalid automation binding")
	}
	if r.pluginMgr == nil {
		return nil, errors.New("plugin manager missing")
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
