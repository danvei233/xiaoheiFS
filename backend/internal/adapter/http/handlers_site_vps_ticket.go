package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type vpsFirewallRuleCreatePayload struct {
	Direction string `json:"direction" binding:"required,max=32"`
	Protocol  string `json:"protocol" binding:"required,max=32"`
	Method    string `json:"method" binding:"required,max=32"`
	Port      string `json:"port" binding:"required,max=64"`
	IP        string `json:"ip" binding:"required,max=128"`
	Priority  *int   `json:"priority"`
}

type vpsIDURI struct {
	ID int64 `uri:"id" binding:"required,gt=0"`
}

type vpsSnapshotURI struct {
	ID         int64 `uri:"id" binding:"required,gt=0"`
	SnapshotID int64 `uri:"snapshotId" binding:"required,gt=0"`
}

type vpsBackupURI struct {
	ID       int64 `uri:"id" binding:"required,gt=0"`
	BackupID int64 `uri:"backupId" binding:"required,gt=0"`
}

type vpsFirewallRuleURI struct {
	ID     int64 `uri:"id" binding:"required,gt=0"`
	RuleID int64 `uri:"ruleId" binding:"required,gt=0"`
}

type vpsPortMappingURI struct {
	ID        int64 `uri:"id" binding:"required,gt=0"`
	MappingID int64 `uri:"mappingId" binding:"required,gt=0"`
}

type vpsKeywordsQuery struct {
	Keywords string `form:"keywords" binding:"omitempty,max=128"`
}

type ticketStatusQuery struct {
	Status string `form:"status" binding:"omitempty,max=32"`
}

func (h *Handler) VPSList(c *gin.Context) {
	limit, offset := paging(c)
	var query struct {
		Status     string `form:"status" binding:"omitempty,oneof=provisioning running stopped reinstalling reinstall_failed expired_locked rescue cracking_password locked unknown failed deleting"`
		Keyword    string `form:"keyword" binding:"omitempty,max=128"`
		Region     string `form:"region" binding:"omitempty,max=128"`
		ExpireDays int    `form:"expire_days" binding:"omitempty,min=1,max=3650"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	// P3-43: the user console pages and filters server-side instead of relying
	// on a full-table return contract. total is the unfiltered-count independent
	// page size so the frontend can render the filtered total.
	filter := appports.VPSUserInstanceFilter{
		Status:  strings.TrimSpace(query.Status),
		Keyword: strings.TrimSpace(query.Keyword),
		Region:  strings.TrimSpace(query.Region),
	}
	if query.ExpireDays > 0 {
		before := time.Now().AddDate(0, 0, query.ExpireDays)
		filter.ExpireBefore = &before
	}
	items, total, err := h.vpsSvc.ListByUserFiltered(c.Request.Context(), getUserID(c), filter, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrVpsListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": h.toUserVPSInstanceDTOsWithLifecycle(c, items), "total": total})
}

func (h *Handler) VPSDetail(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toUserVPSInstanceDTOWithLifecycle(c, inst))
}

// VPSCredentials is the explicit, least-privilege credentials endpoint (P2-4).
// List/detail/monitor responses never carry AccessInfo; a user must explicitly
// request the panel/VNC/OS credentials here. Every successful issuance writes
// an audit log so credential exposure is traceable.
func (h *Handler) VPSCredentials(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	creds := parseMapJSON(inst.AccessInfoJSON)
	if len(creds) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.adminSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrAuditWriteFailed.Error()})
		return
	}
	auditActorID := getUserID(c)
	if actor := requestActor(c); actor.ImpersonatorID > 0 {
		auditActorID = actor.ImpersonatorID
	}
	if err := h.adminSvc.AuditRequired(c.Request.Context(), auditActorID, "vps.credentials", "vps", strconv.FormatInt(inst.ID, 10), map[string]any{
		"ip":                   c.ClientIP(),
		"impersonated_user_id": getUserID(c),
	}); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrAuditWriteFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_info": creds})
}

func (h *Handler) VPSRefresh(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	updated, err := h.vpsSvc.RefreshStatus(c.Request.Context(), inst)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toUserVPSInstanceDTOWithLifecycle(c, updated))
}

func (h *Handler) VPSPanel(c *gin.Context) {
	url, ok := h.vpsAccessURL(c, "panel_login", "面板登录", h.vpsSvc.GetPanelURL)
	if !ok {
		return
	}
	c.Redirect(http.StatusFound, url)
}

func (h *Handler) VPSPanelURL(c *gin.Context) {
	url, ok := h.vpsAccessURL(c, "panel_login", "面板登录", h.vpsSvc.GetPanelURL)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *Handler) vpsAccessURL(c *gin.Context, feature string, label string, getter func(context.Context, domain.VPSInstance) (string, error)) (string, bool) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return "", false
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return "", false
	}
	if h.denyIfFeatureDisabled(c, inst, feature, label) {
		return "", false
	}
	url, err := getter(c.Request.Context(), inst)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return "", false
	}
	// web_url_content F6：url 来自自动化插件，插件被攻陷/第三方可返回
	// javascript:/data://evil.com 等。VPSPanel/VPSVNC 会直接 302 到该地址
	// （open redirect / scheme 导航），必须在写 Location 之前服务端校验。
	if !isSafeVPSAccessURL(url) {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrUnsafeExternalURL.Error()})
		return "", false
	}
	return url, true
}

// isSafeVPSAccessURL 允许面板/VNC 的 http(s) 地址（含 LAN/内网 IP 与 dev
// localhost http），拒绝 javascript:/data:/file:/协议相对/userinfo/无 host。
func isSafeVPSAccessURL(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" || strings.HasPrefix(value, "//") {
		return false
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	u, err := url.Parse(value)
	if err != nil || u.Host == "" {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}
	if u.User != nil {
		return false
	}
	return true
}

func (h *Handler) VPSMonitor(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if refreshed, err := h.vpsSvc.RefreshStatus(c.Request.Context(), inst); err == nil {
		inst = refreshed
	}
	payload := gin.H{
		"status":           string(inst.Status),
		"automation_state": inst.AutomationState,
		"spec":             parseRawJSON(inst.SpecJSON),
	}
	monitor, err := h.vpsSvc.Monitor(c.Request.Context(), inst)
	if err != nil {
		// A GET must not mutate VPS state. Do not rewrite the status based on
		// error text — RefreshStatus above already reflects provider state.
		payload["monitor_error"] = err.Error()
		payload["cpu"] = 0
		payload["memory"] = 0
		payload["bytes_in"] = int64(0)
		payload["bytes_out"] = int64(0)
		payload["storage"] = 0
		c.JSON(http.StatusOK, payload)
		return
	}
	payload["cpu"] = monitor.CPUPercent
	payload["memory"] = monitor.MemoryPercent
	payload["bytes_in"] = monitor.BytesIn
	payload["bytes_out"] = monitor.BytesOut
	payload["storage"] = monitor.StoragePercent
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) VPSVNC(c *gin.Context) {
	url, ok := h.vpsAccessURL(c, "vnc", "VNC", h.vpsSvc.VNCURL)
	if !ok {
		return
	}
	c.Redirect(http.StatusFound, url)
}

func (h *Handler) VPSVNCURL(c *gin.Context) {
	url, ok := h.vpsAccessURL(c, "vnc", "VNC", h.vpsSvc.VNCURL)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *Handler) VPSStart(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if err := h.vpsSvc.Start(c.Request.Context(), inst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSShutdown(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if err := h.vpsSvc.Shutdown(c.Request.Context(), inst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSReboot(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if err := h.vpsSvc.Reboot(c.Request.Context(), inst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSResetOS(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "reinstall", "重装系统") {
		return
	}
	var payload map[string]any
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	parseInt := func(val any) int64 {
		switch v := val.(type) {
		case float64:
			return int64(v)
		case string:
			parsed, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			return parsed
		default:
			return 0
		}
	}
	hostID := parseInt(payload["host_id"])
	templateID := parseInt(payload["template_id"])
	password, _ := payload["password"].(string)
	if hostID != 0 && hostID != uri.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	if templateID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	var matchedSystemID int64
	// Validate template against instance line to prevent cross-line image
	// reinstall. When the line cannot be determined (line_id == 0 and no
	// package/plan lookup), validate against the global enabled image set so a
	// disabled or unknown template is still rejected.
	lineID := inst.LineID
	if lineID <= 0 && inst.PackageID > 0 && h.catalogSvc != nil {
		if pkg, pkgErr := h.catalogSvc.GetPackage(c.Request.Context(), inst.PackageID); pkgErr == nil && pkg.PlanGroupID > 0 {
			if plan, planErr := h.catalogSvc.GetPlanGroup(c.Request.Context(), pkg.PlanGroupID); planErr == nil {
				lineID = plan.LineID
			}
		}
	}
	if h.catalogSvc == nil {
		// Cannot validate the template without the catalog service; fail closed.
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	allowedImages, listErr := h.catalogSvc.ListSystemImages(c.Request.Context(), lineID)
	if listErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	allowed := false
	for _, img := range allowedImages {
		if !img.Enabled {
			continue
		}
		if img.ImageID == templateID || img.ID == templateID {
			allowed = true
			matchedSystemID = img.ID
			break
		}
	}
	if !allowed {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	if err := h.vpsSvc.ResetOS(c.Request.Context(), inst, templateID, strings.TrimSpace(password)); err != nil {
		if err == appshared.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if matchedSystemID > 0 && h.vpsSvc != nil {
		_ = h.vpsSvc.UpdateLocalSystemID(c.Request.Context(), inst, matchedSystemID)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSResetOSPassword(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "reset_password", "重置密码") {
		return
	}
	var payload struct {
		Password string `json:"password"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	if err := h.vpsSvc.ResetOSPassword(c.Request.Context(), inst, strings.TrimSpace(payload.Password)); err != nil {
		if err == appshared.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSSnapshots(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "snapshot", "快照") {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListSnapshots(c.Request.Context(), inst)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, appshared.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		if err := h.vpsSvc.CreateSnapshot(c.Request.Context(), inst); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, appshared.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": domain.ErrMethodNotAllowed.Error()})
	}
}

func (h *Handler) VPSSnapshotDelete(c *gin.Context) {
	var uri vpsSnapshotURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "snapshot", "快照") {
		return
	}
	if err := h.vpsSvc.DeleteSnapshot(c.Request.Context(), inst, uri.SnapshotID); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, appshared.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSSnapshotRestore(c *gin.Context) {
	var uri vpsSnapshotURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "snapshot", "快照") {
		return
	}
	if err := h.vpsSvc.RestoreSnapshot(c.Request.Context(), inst, uri.SnapshotID); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, appshared.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSBackups(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "backup", "备份") {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListBackups(c.Request.Context(), inst)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, appshared.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		if err := h.vpsSvc.CreateBackup(c.Request.Context(), inst); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, appshared.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": domain.ErrMethodNotAllowed.Error()})
	}
}

func (h *Handler) VPSBackupDelete(c *gin.Context) {
	var uri vpsBackupURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "backup", "备份") {
		return
	}
	if err := h.vpsSvc.DeleteBackup(c.Request.Context(), inst, uri.BackupID); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, appshared.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSBackupRestore(c *gin.Context) {
	var uri vpsBackupURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "backup", "备份") {
		return
	}
	if err := h.vpsSvc.RestoreBackup(c.Request.Context(), inst, uri.BackupID); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, appshared.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSFirewallRules(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "firewall", "防火墙") {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListFirewallRules(c.Request.Context(), inst)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, appshared.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		var payload vpsFirewallRuleCreatePayload
		if err := bindJSON(c, &payload); err != nil {
			writeBodyError(c, err)
			return
		}
		req := appshared.AutomationFirewallRuleCreate{
			Direction: strings.TrimSpace(payload.Direction),
			Protocol:  strings.TrimSpace(payload.Protocol),
			Method:    strings.TrimSpace(payload.Method),
			Port:      strings.TrimSpace(payload.Port),
			IP:        strings.TrimSpace(payload.IP),
		}
		if payload.Priority != nil {
			req.Priority = *payload.Priority
		}
		if req.Direction == "" || req.Protocol == "" || req.Method == "" || req.Port == "" || req.IP == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
			return
		}
		if err := h.vpsSvc.AddFirewallRule(c.Request.Context(), inst, req); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, appshared.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": domain.ErrMethodNotAllowed.Error()})
	}
}

func (h *Handler) VPSFirewallDelete(c *gin.Context) {
	var uri vpsFirewallRuleURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "firewall", "防火墙") {
		return
	}
	if err := h.vpsSvc.DeleteFirewallRule(c.Request.Context(), inst, uri.RuleID); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, appshared.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSPortMappings(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "port_mapping", "端口映射") {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListPortMappings(c.Request.Context(), inst)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, appshared.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		var payload map[string]any
		if err := bindJSON(c, &payload); err != nil {
			writeBodyError(c, err)
			return
		}
		name := strings.TrimSpace(fmt.Sprint(payload["name"]))
		sport := strings.TrimSpace(fmt.Sprint(payload["sport"]))
		if sport == "<nil>" {
			sport = ""
		}
		if len(name) > 64 || len(sport) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
			return
		}
		dport, ok := parsePortValue(payload["dport"])
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
			return
		}
		req := appshared.AutomationPortMappingCreate{
			Name:  name,
			Sport: sport,
			Dport: dport,
		}
		if err := h.vpsSvc.AddPortMapping(c.Request.Context(), inst, req); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, appshared.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": domain.ErrMethodNotAllowed.Error()})
	}
}

func parsePortValue(value any) (int64, bool) {
	const maxPort = 65535
	switch v := value.(type) {
	case float64:
		if v <= 0 || v > maxPort {
			return 0, false
		}
		return int64(v), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil || parsed <= 0 || parsed > maxPort {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func (h *Handler) VPSPortCandidates(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "port_mapping", "端口映射") {
		return
	}
	var query vpsKeywordsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	keywords := strings.TrimSpace(query.Keywords)
	items, err := h.vpsSvc.FindPortCandidates(c.Request.Context(), inst, keywords)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, appshared.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *Handler) VPSPortMappingDelete(c *gin.Context) {
	var uri vpsPortMappingURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "port_mapping", "端口映射") {
		return
	}
	if err := h.vpsSvc.DeletePortMapping(c.Request.Context(), inst, uri.MappingID); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, appshared.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) TicketCreate(c *gin.Context) {
	var payload struct {
		Subject   string `json:"subject"`
		Content   string `json:"content"`
		Resources []struct {
			ResourceType string `json:"resource_type"`
			ResourceID   int64  `json:"resource_id"`
			ResourceName string `json:"resource_name"`
		} `json:"resources"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	resources := make([]domain.TicketResource, 0, len(payload.Resources))
	for _, res := range payload.Resources {
		resources = append(resources, domain.TicketResource{ResourceType: res.ResourceType, ResourceID: res.ResourceID, ResourceName: res.ResourceName})
	}
	ticket, messages, resItems, err := h.ticketSvc.Create(c.Request.Context(), getUserID(c), payload.Subject, payload.Content, resources)
	if err != nil {
		if err == appshared.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	msgDTOs := make([]TicketMessageDTO, 0, len(messages))
	for _, msg := range messages {
		msgDTOs = append(msgDTOs, toTicketMessageDTO(msg, msg.SenderName, msg.SenderQQ))
	}
	resDTOs := make([]TicketResourceDTO, 0, len(resItems))
	for _, res := range resItems {
		resDTOs = append(resDTOs, toTicketResourceDTO(res))
	}
	c.JSON(http.StatusOK, gin.H{"ticket": toTicketDTO(ticket), "messages": msgDTOs, "resources": resDTOs})
}

func (h *Handler) TicketList(c *gin.Context) {
	var query ticketStatusQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	status := strings.TrimSpace(query.Status)
	limit, offset := paging(c)
	userID := getUserID(c)
	filter := appshared.TicketFilter{UserID: &userID, Status: status, Limit: limit, Offset: offset}
	items, total, err := h.ticketSvc.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]TicketDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toTicketDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func (h *Handler) TicketDetail(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	ticket, messages, resources, err := h.ticketSvc.GetDetail(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	// P3-10: a ticket belonging to another user is indistinguishable from a
	// missing ticket (404) so the endpoint does not leak which IDs exist.
	if ticket.UserID != getUserID(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	msgDTOs := make([]TicketMessageDTO, 0, len(messages))
	for _, msg := range messages {
		msgDTOs = append(msgDTOs, toTicketMessageDTO(msg, msg.SenderName, msg.SenderQQ))
	}
	resDTOs := make([]TicketResourceDTO, 0, len(resources))
	for _, res := range resources {
		resDTOs = append(resDTOs, toTicketResourceDTO(res))
	}
	c.JSON(http.StatusOK, gin.H{"ticket": toTicketDTO(ticket), "messages": msgDTOs, "resources": resDTOs})
}

func (h *Handler) TicketMessageCreate(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	ticket, err := h.ticketSvc.Get(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	// P3-10: non-owner is indistinguishable from missing (404).
	if ticket.UserID != getUserID(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	var payload struct {
		Content string `json:"content"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	msg, err := h.ticketSvc.AddMessage(c.Request.Context(), ticket, getUserID(c), "user", payload.Content)
	if err != nil {
		if err == appshared.ErrForbidden {
			c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrTicketClosed.Error()})
			return
		}
		if err == appshared.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toTicketMessageDTO(msg, msg.SenderName, msg.SenderQQ))
}

func (h *Handler) TicketClose(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	ticket, err := h.ticketSvc.Get(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if err := h.ticketSvc.Close(c.Request.Context(), ticket, getUserID(c)); err != nil {
		// P3-10: closing another user's ticket must be indistinguishable from a
		// missing ticket (404) rather than a 403 existence oracle.
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSEmergencyRenew(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	_, err = h.orderSvc.CreateEmergencyRenewOrder(c.Request.Context(), getUserID(c), inst.ID)
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrConflict {
			status = http.StatusConflict
		} else if err == appshared.ErrForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	updated, _ := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	c.JSON(http.StatusOK, h.toUserVPSInstanceDTOWithLifecycle(c, updated))
}

func (h *Handler) VPSRenewOrder(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		RenewDays      int `json:"renew_days"`
		DurationMonths int `json:"duration_months"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	order, err := h.orderSvc.CreateRenewOrder(c.Request.Context(), getUserID(c), uri.ID, payload.RenewDays, payload.DurationMonths)
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrRealNameRequired || err == appshared.ErrForbidden {
			status = http.StatusForbidden
		} else if errors.Is(err, appshared.ErrConflict) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toOrderDTO(order))
}

func (h *Handler) VPSResizeOrder(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Spec            *appshared.CartSpec `json:"spec"`
		TargetPackageID int64               `json:"target_package_id"`
		ResetAddons     bool                `json:"reset_addons"`
		ScheduledAt     string              `json:"scheduled_at"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	var scheduledAt *time.Time
	if strings.TrimSpace(payload.ScheduledAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.ScheduledAt))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidScheduledAt.Error()})
			return
		}
		scheduledAt = &t
	}
	order, _, err := h.orderSvc.CreateResizeOrder(c.Request.Context(), getUserID(c), uri.ID, payload.Spec, payload.TargetPackageID, payload.ResetAddons, scheduledAt)
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrRealNameRequired || err == appshared.ErrForbidden || err == appshared.ErrResizeDisabled {
			status = http.StatusForbidden
		} else if err == appshared.ErrResizeInProgress || err == appshared.ErrConflict {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order)})
}

func (h *Handler) VPSResizeQuote(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Spec            *appshared.CartSpec `json:"spec"`
		TargetPackageID int64               `json:"target_package_id"`
		ResetAddons     bool                `json:"reset_addons"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	quote, targetSpec, err := h.orderSvc.QuoteResizeOrder(c.Request.Context(), getUserID(c), uri.ID, payload.Spec, payload.TargetPackageID, payload.ResetAddons)
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrRealNameRequired || err == appshared.ErrForbidden || err == appshared.ErrResizeDisabled {
			status = http.StatusForbidden
		} else if err == appshared.ErrResizeInProgress || err == appshared.ErrConflict {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	resp := quote.ToPayload(uri.ID, targetSpec)
	resp["charge_amount"] = centsToFloat(quote.ChargeAmount)
	resp["refund_amount"] = centsToFloat(quote.RefundAmount)
	c.JSON(http.StatusOK, gin.H{"quote": resp})
}

func (h *Handler) VPSRefund(c *gin.Context) {
	if h.orderSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrOrdersDisabled.Error()})
		return
	}
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.vpsSvc.Get(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "refund", "退款") {
		return
	}
	var payload struct {
		Reason string `json:"reason"`
	}
	if err := bindJSONOptional(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	order, amount, err := h.orderSvc.CreateRefundOrder(c.Request.Context(), getUserID(c), uri.ID, payload.Reason)
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrForbidden {
			status = http.StatusForbidden
		} else if err == appshared.ErrConflict {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order), "refund_amount": centsToFloat(amount)})
}

func (h *Handler) denyIfFeatureDisabled(c *gin.Context, inst domain.VPSInstance, feature, label string) bool {
	if h.packageFeatureAllowed(c, inst, feature, true) {
		return false
	}
	c.JSON(http.StatusForbidden, gin.H{"error": label + "功能未启用"})
	return true
}
