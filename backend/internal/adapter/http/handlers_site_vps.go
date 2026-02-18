package http

import (
	"github.com/gin-gonic/gin"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) SiteSettings(c *gin.Context) {
	if h.settingsSvc == nil && h.adminSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	allowed := map[string]bool{
		"site_name":                true,
		"site_url":                 true,
		"logo_url":                 true,
		"favicon_url":              true,
		"site_description":         true,
		"site_keywords":            true,
		"company_name":             true,
		"contact_phone":            true,
		"contact_email":            true,
		"contact_qq":               true,
		"wechat_qrcode":            true,
		"icp_number":               true,
		"psbe_number":              true,
		"maintenance_mode":         true,
		"maintenance_message":      true,
		"analytics_code":           true,
		"site_nav_items":           true,
		"site_logo":                true,
		"site_icp":                 true,
		"site_maintenance_mode":    true,
		"site_maintenance_message": true,
	}
	aliases := map[string]string{
		"site_logo":                "logo_url",
		"site_icp":                 "icp_number",
		"site_maintenance_mode":    "maintenance_mode",
		"site_maintenance_message": "maintenance_message",
	}
	items, err := h.listSettings(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	filtered := make([]domain.Setting, 0)
	indexed := make(map[string]domain.Setting)
	for _, item := range items {
		if allowed[item.Key] {
			filtered = append(filtered, item)
			indexed[item.Key] = item
		}
	}
	for legacy, current := range aliases {
		if _, ok := indexed[current]; ok {
			continue
		}
		if legacyItem, ok := indexed[legacy]; ok {
			filtered = append(filtered, domain.Setting{Key: current, ValueJSON: legacyItem.ValueJSON})
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": toSettingDTOs(filtered)})
}

func (h *Handler) toVPSInstanceDTOWithLifecycle(c *gin.Context, inst domain.VPSInstance) VPSInstanceDTO {
	dto := toVPSInstanceDTO(inst)
	destroyAt, destroyInDays := h.lifecycleDestroyInfo(c, inst.ExpireAt)
	dto.DestroyAt = destroyAt
	dto.DestroyInDays = destroyInDays
	return dto
}

func (h *Handler) toVPSInstanceDTOsWithLifecycle(c *gin.Context, items []domain.VPSInstance) []VPSInstanceDTO {
	out := make([]VPSInstanceDTO, 0, len(items))
	for _, item := range items {
		out = append(out, h.toVPSInstanceDTOWithLifecycle(c, item))
	}
	return out
}

func (h *Handler) lifecycleDestroyInfo(c *gin.Context, expireAt *time.Time) (*time.Time, *int) {
	if expireAt == nil || (h.settingsSvc == nil && h.adminSvc == nil) {
		return nil, nil
	}
	enabled, ok := h.getSettingBool(c, "auto_delete_enabled")
	if !ok || !enabled {
		return nil, nil
	}
	days, ok := h.getSettingInt(c, "auto_delete_days")
	if !ok {
		days = 0
	}
	if days < 0 {
		days = 0
	}
	destroyAt := expireAt.Add(time.Duration(days) * 24 * time.Hour)
	inDays := int(math.Ceil(destroyAt.Sub(time.Now()).Hours() / 24))
	return &destroyAt, &inDays
}

func (h *Handler) getSettingInt(c *gin.Context, key string) (int, bool) {
	if h.settingsSvc == nil && h.adminSvc == nil {
		return 0, false
	}
	setting, err := h.getSetting(c, key)
	if err != nil {
		return 0, false
	}
	val, err := strconv.Atoi(strings.TrimSpace(setting.ValueJSON))
	if err != nil {
		return 0, false
	}
	return val, true
}

func (h *Handler) getSettingBool(c *gin.Context, key string) (bool, bool) {
	if h.settingsSvc == nil && h.adminSvc == nil {
		return false, false
	}
	setting, err := h.getSetting(c, key)
	if err != nil {
		return false, false
	}
	raw := strings.TrimSpace(setting.ValueJSON)
	if raw == "" {
		return false, false
	}
	switch strings.ToLower(raw) {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	default:
		return false, false
	}
}
