package http

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	apppluginadmin "xiaoheiplay/internal/app/pluginadmin"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func pluginUploadAllowed() bool {
	return gin.Mode() == gin.DebugMode
}

type pluginKeyURI struct {
	Key string `uri:"key" binding:"required"`
}

type pluginCategoryPluginURI struct {
	Category string `uri:"category" binding:"required"`
	PluginID string `uri:"plugin_id" binding:"required"`
}

type pluginCategoryPluginInstanceURI struct {
	Category   string `uri:"category" binding:"required"`
	PluginID   string `uri:"plugin_id" binding:"required"`
	InstanceID string `uri:"instance_id" binding:"required"`
}

func (h *Handler) AdminPaymentProviders(c *gin.Context) {
	if h.paymentSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPaymentDisabled.Error()})
		return
	}
	var query struct {
		IncludeDisabled string `form:"include_disabled"`
		IncludeLegacy   string `form:"include_legacy"`
		Scene           string `form:"scene"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	includeDisabled := strings.EqualFold(strings.TrimSpace(query.IncludeDisabled), "true")
	includeLegacy := strings.EqualFold(strings.TrimSpace(query.IncludeLegacy), "true")
	scene := strings.TrimSpace(query.Scene)
	items, err := h.paymentSvc.ListProvidersByScene(c, includeDisabled, scene)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !includeLegacy {
		filtered := make([]appshared.PaymentProviderInfo, 0, len(items))
		for _, item := range items {
			k := strings.ToLower(strings.TrimSpace(item.Key))
			if k == "custom" {
				continue
			}
			filtered = append(filtered, item)
		}
		items = filtered
	}
	c.JSON(http.StatusOK, gin.H{"items": toPaymentProviderDTOs(items)})
}

func (h *Handler) AdminPaymentProviderUpdate(c *gin.Context) {
	if h.paymentSvc == nil && h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPaymentDisabled.Error()})
		return
	}
	var uri pluginKeyURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	var payload struct {
		Enabled    *bool  `json:"enabled"`
		ConfigJSON string `json:"config_json"`
		Scene      string `json:"scene"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	enabled := true
	if payload.Enabled != nil {
		enabled = *payload.Enabled
	}
	trimmedKey := strings.TrimSpace(uri.Key)
	trimmedScene := strings.TrimSpace(payload.Scene)
	if trimmedScene != "" {
		if h.paymentSvc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPaymentDisabled.Error()})
			return
		}
		if payload.Enabled == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
			return
		}
		if err := h.paymentSvc.UpdateProviderSceneEnabled(c, trimmedKey, trimmedScene, enabled); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	if strings.Contains(trimmedKey, ".") {
		if h.pluginAdmin == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPaymentMethodRepoMissing.Error()})
			return
		}
		parts := strings.Split(trimmedKey, ".")
		pluginID := ""
		instanceID := apppluginadmin.DefaultInstanceID
		method := ""
		switch len(parts) {
		case 2:
			pluginID = strings.TrimSpace(parts[0])
			method = strings.TrimSpace(parts[1])
		default:
			pluginID = strings.TrimSpace(parts[0])
			instanceID = strings.TrimSpace(parts[1])
			method = strings.TrimSpace(strings.Join(parts[2:], "."))
		}
		if pluginID == "" || instanceID == "" || method == "" || payload.Enabled == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidPluginPaymentKeyOrPayload.Error()})
			return
		}
		if err := h.pluginAdmin.UpsertPaymentMethod(c, "payment", pluginID, instanceID, method, enabled); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if h.adminSvc != nil {
			h.adminSvc.Audit(c, getUserID(c), "plugin.payment_method.update", "plugin", "payment/"+pluginID+"/"+instanceID, map[string]any{
				"method":  method,
				"enabled": enabled,
				"via":     "payments.providers.update",
			})
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	if h.paymentSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPaymentDisabled.Error()})
		return
	}
	if err := h.paymentSvc.UpdateProvider(c, uri.Key, enabled, payload.ConfigJSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPaymentPluginUpload(c *gin.Context) {
	if !pluginUploadAllowed() {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrPluginUploadDebugOnly.Error()})
		return
	}
	password := c.PostForm("password")
	if password == "" {
		password = c.GetHeader("X-Plugin-Password")
	}
	expected := ""
	if h.pluginAdmin != nil {
		expected = h.pluginAdmin.ResolveUploadPassword(c, "")
	} else {
		expected = strings.TrimSpace(h.getSettingValueByKey(c, "payment_plugin_upload_password"))
	}
	if expected == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginUploadPasswordNotConfigured.Error()})
		return
	}
	if password == "" || password != expected {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrInvalidPassword.Error()})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrMissingFile.Error()})
		return
	}
	dir := ""
	if h.pluginAdmin != nil {
		dir = strings.TrimSpace(h.pluginAdmin.ResolveUploadDir(c, ""))
	} else {
		dir = strings.TrimSpace(h.getSettingValueByKey(c, "payment_plugin_dir"))
		if dir == "" {
			dir = "plugins/payment"
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrMkdirFailed.Error()})
		return
	}
	filename := filepath.Base(file.Filename)
	if filename == "." || filename == "" || strings.Contains(filename, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidFilename.Error()})
		return
	}
	dst := filepath.Join(dir, filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrUploadFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "path": dst})
}

func (h *Handler) AdminPluginsList(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	items, err := h.pluginAdmin.List(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminPluginsDiscover(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	items, err := h.pluginAdmin.DiscoverOnDisk(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminPluginPaymentMethodsList(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var query struct {
		Category   string `form:"category"`
		PluginID   string `form:"plugin_id"`
		InstanceID string `form:"instance_id"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	category := strings.TrimSpace(query.Category)
	pluginID := strings.TrimSpace(query.PluginID)
	instanceID := strings.TrimSpace(query.InstanceID)
	items, err := h.pluginAdmin.ListPaymentMethods(c, category, pluginID, instanceID)
	if err != nil {
		if strings.Contains(err.Error(), "required") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "not enabled/loaded") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminPluginPaymentMethodsUpdate(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var payload struct {
		Category   string `json:"category"`
		PluginID   string `json:"plugin_id"`
		InstanceID string `json:"instance_id"`
		Method     string `json:"method"`
		Enabled    *bool  `json:"enabled"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if payload.Enabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginMethodUpdateRequired.Error()})
		return
	}
	err := h.pluginAdmin.UpdatePaymentMethod(c, payload.Category, payload.PluginID, payload.InstanceID, payload.Method, *payload.Enabled)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not enabled/loaded") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	category := strings.TrimSpace(payload.Category)
	pluginID := strings.TrimSpace(payload.PluginID)
	instanceID := strings.TrimSpace(payload.InstanceID)
	if category == "" {
		category = "payment"
	}
	if instanceID == "" {
		instanceID = apppluginadmin.DefaultInstanceID
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.payment_method.update", "plugin", category+"/"+pluginID+"/"+instanceID, map[string]any{
			"method":  strings.TrimSpace(payload.Method),
			"enabled": *payload.Enabled,
		})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstall(c *gin.Context) {
	if !pluginUploadAllowed() {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrPluginUploadDebugOnly.Error()})
		return
	}
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrMissingFile.Error()})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrOpenFileFailed.Error()})
		return
	}
	defer f.Close()

	inst, err := h.pluginAdmin.Install(c, file.Filename, f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if inst.SignatureStatus != domain.PluginSignatureOfficial {
		adminPassword := strings.TrimSpace(c.PostForm("admin_password"))
		if adminPassword == "" {
			_ = h.pluginAdmin.Uninstall(c, inst.Category, inst.PluginID)
			c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrAdminPasswordRequiredForUntrustedPlugin.Error()})
			return
		}
		if h.authSvc == nil {
			_ = h.pluginAdmin.Uninstall(c, inst.Category, inst.PluginID)
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAuthDisabled.Error()})
			return
		}
		if err := h.authSvc.VerifyPassword(c, getUserID(c), adminPassword); err != nil {
			_ = h.pluginAdmin.Uninstall(c, inst.Category, inst.PluginID)
			c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrInvalidAdminPassword.Error()})
			return
		}
	}

	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.install", "plugin", inst.Category+"/"+inst.PluginID, map[string]any{
			"signature_status": inst.SignatureStatus,
		})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "plugin": inst})
}

func (h *Handler) AdminPluginImportFromDisk(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}

	var payload struct {
		AdminPassword string `json:"admin_password"`
	}
	if err := bindJSONOptional(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}

	targetSig, err := h.pluginAdmin.SignatureStatusOnDisk(uri.Category, uri.PluginID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if targetSig != domain.PluginSignatureOfficial {
		adminPassword := strings.TrimSpace(payload.AdminPassword)
		if adminPassword == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrAdminPasswordRequiredForUntrustedPlugin.Error()})
			return
		}
		if h.authSvc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAuthDisabled.Error()})
			return
		}
		if err := h.authSvc.VerifyPassword(c, getUserID(c), adminPassword); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrInvalidAdminPassword.Error()})
			return
		}
	}

	inst, err := h.pluginAdmin.ImportFromDisk(c, uri.Category, uri.PluginID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.import", "plugin", inst.Category+"/"+inst.PluginID, map[string]any{
			"signature_status": inst.SignatureStatus,
		})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "plugin": inst})
}

func (h *Handler) AdminPluginEnable(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	if err := h.pluginAdmin.EnableInstance(c, uri.Category, uri.PluginID, apppluginadmin.DefaultInstanceID); err != nil {
		writePluginHandlerError(c, err)
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.enable", "plugin", uri.Category+"/"+uri.PluginID+"/"+apppluginadmin.DefaultInstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginDisable(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	if err := h.pluginAdmin.DisableInstance(c, uri.Category, uri.PluginID, apppluginadmin.DefaultInstanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.disable", "plugin", uri.Category+"/"+uri.PluginID+"/"+apppluginadmin.DefaultInstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginUninstall(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	if err := h.pluginAdmin.DeleteInstance(c, uri.Category, uri.PluginID, apppluginadmin.DefaultInstanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.uninstall", "plugin", uri.Category+"/"+uri.PluginID+"/"+apppluginadmin.DefaultInstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginConfigSchema(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	jsonSchema, uiSchema, err := h.pluginAdmin.GetConfigSchemaInstance(c, uri.Category, uri.PluginID, apppluginadmin.DefaultInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"json_schema": jsonSchema, "ui_schema": uiSchema})
}

func (h *Handler) AdminPluginConfigGet(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	cfg, err := h.pluginAdmin.GetConfigInstance(c, uri.Category, uri.PluginID, apppluginadmin.DefaultInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"config_json": cfg})
}

func (h *Handler) AdminPluginConfigUpdate(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	var payload struct {
		ConfigJSON string `json:"config_json"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := validatePluginConfigJSON(payload.ConfigJSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.pluginAdmin.UpdateConfigInstance(c, uri.Category, uri.PluginID, apppluginadmin.DefaultInstanceID, payload.ConfigJSON); err != nil {
		writePluginHandlerError(c, err)
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.config_update", "plugin", uri.Category+"/"+uri.PluginID+"/"+apppluginadmin.DefaultInstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstanceCreate(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	var payload struct {
		InstanceID string `json:"instance_id"`
		ConfigJSON string `json:"config_json"`
	}
	if err := bindJSONOptional(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}

	inst, err := h.pluginAdmin.CreateInstance(c, uri.Category, uri.PluginID, payload.InstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(payload.ConfigJSON) != "" {
		if err := validatePluginConfigJSON(payload.ConfigJSON); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := h.pluginAdmin.UpdateConfigInstance(c, uri.Category, uri.PluginID, inst.InstanceID, payload.ConfigJSON); err != nil {
			_ = h.pluginAdmin.DeleteInstance(c, uri.Category, uri.PluginID, inst.InstanceID)
			writePluginHandlerError(c, err)
			return
		}
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.instance_create", "plugin", uri.Category+"/"+uri.PluginID+"/"+inst.InstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "plugin": inst})
}

func (h *Handler) AdminPluginInstanceEnable(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginInstanceURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	if err := h.pluginAdmin.EnableInstance(c, uri.Category, uri.PluginID, uri.InstanceID); err != nil {
		writePluginHandlerError(c, err)
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.enable", "plugin", uri.Category+"/"+uri.PluginID+"/"+uri.InstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstanceDisable(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginInstanceURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	if err := h.pluginAdmin.DisableInstance(c, uri.Category, uri.PluginID, uri.InstanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.disable", "plugin", uri.Category+"/"+uri.PluginID+"/"+uri.InstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstanceDelete(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginInstanceURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	if err := h.pluginAdmin.DeleteInstance(c, uri.Category, uri.PluginID, uri.InstanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.instance_delete", "plugin", uri.Category+"/"+uri.PluginID+"/"+uri.InstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstanceConfigSchema(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginInstanceURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	jsonSchema, uiSchema, err := h.pluginAdmin.GetConfigSchemaInstance(c, uri.Category, uri.PluginID, uri.InstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"json_schema": jsonSchema, "ui_schema": uiSchema})
}

func (h *Handler) AdminPluginInstanceConfigGet(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginInstanceURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	cfg, err := h.pluginAdmin.GetConfigInstance(c, uri.Category, uri.PluginID, uri.InstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"config_json": cfg})
}

func (h *Handler) AdminPluginInstanceConfigUpdate(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginInstanceURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	var payload struct {
		ConfigJSON string `json:"config_json"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := validatePluginConfigJSON(payload.ConfigJSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.pluginAdmin.UpdateConfigInstance(c, uri.Category, uri.PluginID, uri.InstanceID, payload.ConfigJSON); err != nil {
		writePluginHandlerError(c, err)
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.config_update", "plugin", uri.Category+"/"+uri.PluginID+"/"+uri.InstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func writePluginHandlerError(c *gin.Context, err error) {
	if cfgErr, ok := appshared.AsConfigValidationError(err); ok && cfgErr != nil {
		resp := gin.H{
			"error": cfgErr.Error(),
			"code":  strings.TrimSpace(cfgErr.Code),
		}
		if len(cfgErr.MissingFields) > 0 {
			resp["missing_fields"] = cfgErr.MissingFields
		}
		if p := strings.TrimSpace(cfgErr.RedirectPath); p != "" {
			resp["redirect_path"] = p
		}
		c.JSON(http.StatusConflict, resp)
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func validatePluginConfigJSON(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	if isDoubleEncodedContainerJSON(trimmed) {
		return fmt.Errorf("%w: config_json contains double-encoded json", domain.ErrInvalidInput)
	}
	if !json.Valid([]byte(trimmed)) {
		return fmt.Errorf("%w: config_json expects valid json", domain.ErrInvalidInput)
	}
	return nil
}

func (h *Handler) AdminPluginDeleteFiles(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	var uri pluginCategoryPluginURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	if err := h.pluginAdmin.DeletePluginFiles(c, uri.Category, uri.PluginID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.delete_files", "plugin", uri.Category+"/"+uri.PluginID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
