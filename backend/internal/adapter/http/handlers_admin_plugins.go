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

func (h *Handler) AdminPaymentProviders(c *gin.Context) {
	if h.paymentSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPaymentDisabled.Error()})
		return
	}
	includeDisabled := strings.EqualFold(strings.TrimSpace(c.Query("include_disabled")), "true")
	includeLegacy := strings.EqualFold(strings.TrimSpace(c.Query("include_legacy")), "true")
	scene := strings.TrimSpace(c.Query("scene"))
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
	key := c.Param("key")
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
	trimmedKey := strings.TrimSpace(key)
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
	if err := h.paymentSvc.UpdateProvider(c, key, enabled, payload.ConfigJSON); err != nil {
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
	category := strings.TrimSpace(c.Query("category"))
	pluginID := strings.TrimSpace(c.Query("plugin_id"))
	instanceID := strings.TrimSpace(c.Query("instance_id"))
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
	category := c.Param("category")
	pluginID := c.Param("plugin_id")

	var payload struct {
		AdminPassword string `json:"admin_password"`
	}
	if err := bindJSONOptional(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}

	targetSig, err := h.pluginAdmin.SignatureStatusOnDisk(category, pluginID)
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

	inst, err := h.pluginAdmin.ImportFromDisk(c, category, pluginID)
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
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	if err := h.pluginAdmin.EnableInstance(c, category, pluginID, apppluginadmin.DefaultInstanceID); err != nil {
		writePluginHandlerError(c, err)
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.enable", "plugin", category+"/"+pluginID+"/"+apppluginadmin.DefaultInstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginDisable(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	if err := h.pluginAdmin.DisableInstance(c, category, pluginID, apppluginadmin.DefaultInstanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.disable", "plugin", category+"/"+pluginID+"/"+apppluginadmin.DefaultInstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginUninstall(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	if err := h.pluginAdmin.DeleteInstance(c, category, pluginID, apppluginadmin.DefaultInstanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.uninstall", "plugin", category+"/"+pluginID+"/"+apppluginadmin.DefaultInstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginConfigSchema(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	jsonSchema, uiSchema, err := h.pluginAdmin.GetConfigSchemaInstance(c, category, pluginID, apppluginadmin.DefaultInstanceID)
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
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	cfg, err := h.pluginAdmin.GetConfigInstance(c, category, pluginID, apppluginadmin.DefaultInstanceID)
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
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
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
	if err := h.pluginAdmin.UpdateConfigInstance(c, category, pluginID, apppluginadmin.DefaultInstanceID, payload.ConfigJSON); err != nil {
		writePluginHandlerError(c, err)
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.config_update", "plugin", category+"/"+pluginID+"/"+apppluginadmin.DefaultInstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstanceCreate(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	var payload struct {
		InstanceID string `json:"instance_id"`
		ConfigJSON string `json:"config_json"`
	}
	if err := bindJSONOptional(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}

	inst, err := h.pluginAdmin.CreateInstance(c, category, pluginID, payload.InstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(payload.ConfigJSON) != "" {
		if err := validatePluginConfigJSON(payload.ConfigJSON); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := h.pluginAdmin.UpdateConfigInstance(c, category, pluginID, inst.InstanceID, payload.ConfigJSON); err != nil {
			_ = h.pluginAdmin.DeleteInstance(c, category, pluginID, inst.InstanceID)
			writePluginHandlerError(c, err)
			return
		}
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.instance_create", "plugin", category+"/"+pluginID+"/"+inst.InstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "plugin": inst})
}

func (h *Handler) AdminPluginInstanceEnable(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	instanceID := c.Param("instance_id")
	if err := h.pluginAdmin.EnableInstance(c, category, pluginID, instanceID); err != nil {
		writePluginHandlerError(c, err)
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.enable", "plugin", category+"/"+pluginID+"/"+instanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstanceDisable(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	instanceID := c.Param("instance_id")
	if err := h.pluginAdmin.DisableInstance(c, category, pluginID, instanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.disable", "plugin", category+"/"+pluginID+"/"+instanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstanceDelete(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	instanceID := c.Param("instance_id")
	if err := h.pluginAdmin.DeleteInstance(c, category, pluginID, instanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.instance_delete", "plugin", category+"/"+pluginID+"/"+instanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstanceConfigSchema(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	instanceID := c.Param("instance_id")
	jsonSchema, uiSchema, err := h.pluginAdmin.GetConfigSchemaInstance(c, category, pluginID, instanceID)
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
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	instanceID := c.Param("instance_id")
	cfg, err := h.pluginAdmin.GetConfigInstance(c, category, pluginID, instanceID)
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
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	instanceID := c.Param("instance_id")
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
	if err := h.pluginAdmin.UpdateConfigInstance(c, category, pluginID, instanceID, payload.ConfigJSON); err != nil {
		writePluginHandlerError(c, err)
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.config_update", "plugin", category+"/"+pluginID+"/"+instanceID, map[string]any{})
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
		return fmt.Errorf("config_json contains double-encoded json")
	}
	if !json.Valid([]byte(trimmed)) {
		return fmt.Errorf("config_json expects valid json")
	}
	return nil
}

func (h *Handler) AdminPluginDeleteFiles(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginsDisabled.Error()})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	if err := h.pluginAdmin.DeletePluginFiles(c, category, pluginID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.delete_files", "plugin", category+"/"+pluginID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
