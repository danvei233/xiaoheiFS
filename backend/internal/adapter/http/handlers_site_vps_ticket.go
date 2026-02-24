package http

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type vpsFirewallRuleCreatePayload struct {
	Direction string `json:"direction" validate:"required"`
	Protocol  string `json:"protocol" validate:"required"`
	Method    string `json:"method" validate:"required"`
	Port      string `json:"port" validate:"required"`
	IP        string `json:"ip" validate:"required"`
	Priority  *int   `json:"priority"`
}

func (h *Handler) VPSList(c *gin.Context) {
	items, err := h.vpsSvc.ListByUser(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrVpsListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": h.toVPSInstanceDTOsWithLifecycle(c, items)})
}

func (h *Handler) VPSDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, inst))
}

func (h *Handler) VPSRefresh(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	updated, err := h.vpsSvc.RefreshStatus(c, inst)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, updated))
}

func (h *Handler) VPSPanel(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "panel_login", "面板登录") {
		return
	}
	url, err := h.vpsSvc.GetPanelURL(c, inst)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, url)
}

func (h *Handler) VPSMonitor(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if refreshed, err := h.vpsSvc.RefreshStatus(c, inst); err == nil {
		inst = refreshed
	}
	payload := gin.H{
		"status":           string(inst.Status),
		"automation_state": inst.AutomationState,
		"access_info":      parseMapJSON(inst.AccessInfoJSON),
		"spec":             parseRawJSON(inst.SpecJSON),
	}
	monitor, err := h.vpsSvc.Monitor(c, inst)
	if err != nil {
		if strings.Contains(err.Error(), "创建中") {
			_ = h.vpsSvc.SetStatus(c, inst, domain.VPSStatusProvisioning, 0)
			payload["status"] = string(domain.VPSStatusProvisioning)
			payload["automation_state"] = 0
		}
		payload["monitor_error"] = err.Error()
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "vnc", "VNC") {
		return
	}
	url, err := h.vpsSvc.VNCURL(c, inst)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, url)
}

func (h *Handler) VPSStart(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if err := h.vpsSvc.Start(c, inst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSShutdown(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if err := h.vpsSvc.Shutdown(c, inst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSReboot(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if err := h.vpsSvc.Reboot(c, inst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSResetOS(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "reinstall", "重装系统") {
		return
	}
	var payload map[string]any
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
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
	if hostID != 0 && hostID != id {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	if templateID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	var matchedSystemID int64
	// Validate template against instance line to prevent cross-line image reinstall.
	lineID := inst.LineID
	if lineID <= 0 && inst.PackageID > 0 {
		if pkg, pkgErr := h.catalogSvc.GetPackage(c, inst.PackageID); pkgErr == nil && pkg.PlanGroupID > 0 {
			if plan, planErr := h.catalogSvc.GetPlanGroup(c, pkg.PlanGroupID); planErr == nil {
				lineID = plan.LineID
			}
		}
	}
	if lineID > 0 {
		allowedImages, listErr := h.catalogSvc.ListSystemImages(c, lineID)
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
	}
	if matchedSystemID == 0 {
		if img, imgErr := h.catalogSvc.GetSystemImage(c, templateID); imgErr == nil && img.ID > 0 {
			matchedSystemID = img.ID
		}
	}
	if err := h.vpsSvc.ResetOS(c, inst, templateID, strings.TrimSpace(password)); err != nil {
		if err == appshared.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if matchedSystemID > 0 && h.vpsSvc != nil {
		_ = h.vpsSvc.UpdateLocalSystemID(c, inst, matchedSystemID)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSResetOSPassword(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.vpsSvc.ResetOSPassword(c, inst, strings.TrimSpace(payload.Password)); err != nil {
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "snapshot", "快照") {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListSnapshots(c, inst)
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
		if err := h.vpsSvc.CreateSnapshot(c, inst); err != nil {
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	snapshotID, _ := strconv.ParseInt(c.Param("snapshotId"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "snapshot", "快照") {
		return
	}
	if err := h.vpsSvc.DeleteSnapshot(c, inst, snapshotID); err != nil {
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	snapshotID, _ := strconv.ParseInt(c.Param("snapshotId"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "snapshot", "快照") {
		return
	}
	if err := h.vpsSvc.RestoreSnapshot(c, inst, snapshotID); err != nil {
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "backup", "备份") {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListBackups(c, inst)
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
		if err := h.vpsSvc.CreateBackup(c, inst); err != nil {
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	backupID, _ := strconv.ParseInt(c.Param("backupId"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "backup", "备份") {
		return
	}
	if err := h.vpsSvc.DeleteBackup(c, inst, backupID); err != nil {
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	backupID, _ := strconv.ParseInt(c.Param("backupId"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "backup", "备份") {
		return
	}
	if err := h.vpsSvc.RestoreBackup(c, inst, backupID); err != nil {
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "firewall", "防火墙") {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListFirewallRules(c, inst)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
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
		if err := h.vpsSvc.AddFirewallRule(c, inst, req); err != nil {
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ruleID, _ := strconv.ParseInt(c.Param("ruleId"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "firewall", "防火墙") {
		return
	}
	if err := h.vpsSvc.DeleteFirewallRule(c, inst, ruleID); err != nil {
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "port_mapping", "端口映射") {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListPortMappings(c, inst)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
			return
		}
		name := strings.TrimSpace(fmt.Sprint(payload["name"]))
		sport := strings.TrimSpace(fmt.Sprint(payload["sport"]))
		if sport == "<nil>" {
			sport = ""
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
		if err := h.vpsSvc.AddPortMapping(c, inst, req); err != nil {
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
	switch v := value.(type) {
	case float64:
		if v <= 0 {
			return 0, false
		}
		return int64(v), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil || parsed <= 0 {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func (h *Handler) VPSPortCandidates(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "port_mapping", "端口映射") {
		return
	}
	keywords := strings.TrimSpace(c.Query("keywords"))
	items, err := h.vpsSvc.FindPortCandidates(c, inst, keywords)
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	mappingID, _ := strconv.ParseInt(c.Param("mappingId"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if h.denyIfFeatureDisabled(c, inst, "port_mapping", "端口映射") {
		return
	}
	if err := h.vpsSvc.DeletePortMapping(c, inst, mappingID); err != nil {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	resources := make([]domain.TicketResource, 0, len(payload.Resources))
	for _, res := range payload.Resources {
		resources = append(resources, domain.TicketResource{ResourceType: res.ResourceType, ResourceID: res.ResourceID, ResourceName: res.ResourceName})
	}
	ticket, messages, resItems, err := h.ticketSvc.Create(c, getUserID(c), payload.Subject, payload.Content, resources)
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
	status := strings.TrimSpace(c.Query("status"))
	limit, offset := paging(c)
	userID := getUserID(c)
	filter := appshared.TicketFilter{UserID: &userID, Status: status, Limit: limit, Offset: offset}
	items, total, err := h.ticketSvc.List(c, filter)
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticket, messages, resources, err := h.ticketSvc.GetDetail(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if ticket.UserID != getUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrForbidden.Error()})
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticket, err := h.ticketSvc.Get(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if ticket.UserID != getUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrForbidden.Error()})
		return
	}
	var payload struct {
		Content string `json:"content"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	msg, err := h.ticketSvc.AddMessage(c, ticket, getUserID(c), "user", payload.Content)
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticket, err := h.ticketSvc.Get(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if err := h.ticketSvc.Close(c, ticket, getUserID(c)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrForbidden.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSEmergencyRenew(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	_, err = h.orderSvc.CreateEmergencyRenewOrder(c, getUserID(c), inst.ID)
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
	updated, _ := h.vpsSvc.Get(c, id, getUserID(c))
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, updated))
}

func (h *Handler) VPSRenewOrder(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		RenewDays      int `json:"renew_days"`
		DurationMonths int `json:"duration_months"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	order, err := h.orderSvc.CreateRenewOrder(c, getUserID(c), id, payload.RenewDays, payload.DurationMonths)
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Spec            *appshared.CartSpec `json:"spec"`
		TargetPackageID int64               `json:"target_package_id"`
		ResetAddons     bool                `json:"reset_addons"`
		ScheduledAt     string              `json:"scheduled_at"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
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
	order, _, err := h.orderSvc.CreateResizeOrder(c, getUserID(c), id, payload.Spec, payload.TargetPackageID, payload.ResetAddons, scheduledAt)
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
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Spec            *appshared.CartSpec `json:"spec"`
		TargetPackageID int64               `json:"target_package_id"`
		ResetAddons     bool                `json:"reset_addons"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	quote, targetSpec, err := h.orderSvc.QuoteResizeOrder(c, getUserID(c), id, payload.Spec, payload.TargetPackageID, payload.ResetAddons)
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
	resp := quote.ToPayload(id, targetSpec)
	resp["charge_amount"] = centsToFloat(quote.ChargeAmount)
	resp["refund_amount"] = centsToFloat(quote.RefundAmount)
	c.JSON(http.StatusOK, gin.H{"quote": resp})
}

func (h *Handler) VPSRefund(c *gin.Context) {
	if h.orderSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrOrdersDisabled.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	order, amount, err := h.orderSvc.CreateRefundOrder(c, getUserID(c), id, payload.Reason)
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
