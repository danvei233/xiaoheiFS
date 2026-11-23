package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

// 本文件承载「实例基础设施资源」的用户端接口：
// 网卡 / 数据盘 / 光盘 / 反向代理 / 启动顺序 / PCI / USB 直通。
//
// 每个能力都由插件能力位（features）控制：
// 线路/附加项关闭某个开关后，能力列表里不再包含对应 feature，
// denyIfFeatureDisabled 会直接拒绝请求，前端也不再渲染对应页签。

type vpsNicURI struct {
	ID      int64  `uri:"id" binding:"required,gt=0"`
	NicName string `uri:"nicName" binding:"required,max=64"`
}

type vpsHddURI struct {
	ID      int64  `uri:"id" binding:"required,gt=0"`
	HddName string `uri:"hddName" binding:"required,max=64"`
}

type vpsISOURI struct {
	ID      int64  `uri:"id" binding:"required,gt=0"`
	IsoName string `uri:"isoName" binding:"required,max=128"`
}

type vpsProxyURI struct {
	ID         int64 `uri:"id" binding:"required,gt=0"`
	ProxyIndex int64 `uri:"proxyIndex" binding:"min=0"`
}

// featureOrNotImplemented 统一处理「不支持 → 501、其它 → 400」的错误映射。
func (h *Handler) featureOrNotImplemented(c *gin.Context, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, appshared.ErrNotSupported) {
		status = http.StatusNotImplemented
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

// loadVPSForFeature 复用「取实例 + 能力校验」的前置流程。
// 返回 nil 表示已经写入了错误响应，调用方直接 return。
func (h *Handler) loadVPSForFeature(c *gin.Context, id int64, feature, label string) *domain.VPSInstance {
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return nil
	}
	if h.denyIfFeatureDisabled(c, inst, feature, label) {
		return nil
	}
	return &inst
}

// loadVPSForFeatureWrite 在 loadVPSForFeature 之上增加「只读模式」校验。
// 只读模式下功能仍可见（列表可查），但禁止一切创建/修改/删除操作。
func (h *Handler) loadVPSForFeatureWrite(c *gin.Context, id int64, feature, label string) *domain.VPSInstance {
	inst := h.loadVPSForFeature(c, id, feature, label)
	if inst == nil {
		return nil
	}
	if h.denyIfFeatureReadonly(c, *inst, feature, label) {
		return nil
	}
	return inst
}

// ---- 网卡 ----

func (h *Handler) VPSNetworkCards(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst := h.loadVPSForFeature(c, uri.ID, "nic", "网卡管理")
	if inst == nil {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListNetworkCards(c, *inst)
		if err != nil {
			h.featureOrNotImplemented(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		if h.denyIfFeatureReadonly(c, *inst, "nic", "网卡管理") {
			return
		}
		var payload appshared.AutomationNetworkCardInput
		if err := bindJSON(c, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
			return
		}
		if err := h.vpsSvc.CreateNetworkCard(c, *inst, payload); err != nil {
			h.featureOrNotImplemented(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": domain.ErrMethodNotAllowed.Error()})
	}
}

func (h *Handler) VPSNetworkCardUpdate(c *gin.Context) {
	var uri vpsNicURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	inst := h.loadVPSForFeatureWrite(c, uri.ID, "nic", "网卡管理")
	if inst == nil {
		return
	}
	var payload appshared.AutomationNetworkCardInput
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.vpsSvc.UpdateNetworkCard(c, *inst, uri.NicName, payload); err != nil {
		h.featureOrNotImplemented(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSNetworkCardDelete(c *gin.Context) {
	var uri vpsNicURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	inst := h.loadVPSForFeatureWrite(c, uri.ID, "nic", "网卡管理")
	if inst == nil {
		return
	}
	if err := h.vpsSvc.DeleteNetworkCard(c, *inst, uri.NicName); err != nil {
		h.featureOrNotImplemented(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- 数据盘 ----

func (h *Handler) VPSDisks(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst := h.loadVPSForFeature(c, uri.ID, "disk", "数据盘")
	if inst == nil {
		return
	}
	items, err := h.vpsSvc.ListDataDisks(c, *inst)
	if err != nil {
		h.featureOrNotImplemented(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *Handler) VPSDiskMount(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst := h.loadVPSForFeatureWrite(c, uri.ID, "disk", "数据盘")
	if inst == nil {
		return
	}
	var payload struct {
		// SizeGB 面向用户以 GB 为单位，落到主控时换算成 MB。
		SizeGB  int `json:"size_gb" binding:"required,min=1,max=1024"`
		HddType int `json:"hdd_type" binding:"omitempty,min=0,max=1"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.vpsSvc.AttachDataDisk(c, *inst, payload.SizeGB*1024, payload.HddType); err != nil {
		h.featureOrNotImplemented(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSDiskUnmount(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst := h.loadVPSForFeatureWrite(c, uri.ID, "disk", "数据盘")
	if inst == nil {
		return
	}
	var payload struct {
		HddName string `json:"hdd_name" binding:"required,max=64"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.vpsSvc.DetachDataDisk(c, *inst, payload.HddName); err != nil {
		h.featureOrNotImplemented(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSDiskDelete(c *gin.Context) {
	var uri vpsHddURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	inst := h.loadVPSForFeatureWrite(c, uri.ID, "disk", "数据盘")
	if inst == nil {
		return
	}
	if err := h.vpsSvc.DeleteDataDisk(c, *inst, uri.HddName); err != nil {
		h.featureOrNotImplemented(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- 光盘 ----

func (h *Handler) VPSISOs(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst := h.loadVPSForFeature(c, uri.ID, "iso", "光盘管理")
	if inst == nil {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, available, err := h.vpsSvc.ListISOs(c, *inst)
		if err != nil {
			h.featureOrNotImplemented(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items, "available": available})
	case http.MethodPost:
		if h.denyIfFeatureReadonly(c, *inst, "iso", "光盘管理") {
			return
		}
		var payload appshared.AutomationISOInput
		if err := bindJSON(c, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
			return
		}
		if err := h.vpsSvc.MountISO(c, *inst, payload); err != nil {
			h.featureOrNotImplemented(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": domain.ErrMethodNotAllowed.Error()})
	}
}

func (h *Handler) VPSISOUnmount(c *gin.Context) {
	var uri vpsISOURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	inst := h.loadVPSForFeatureWrite(c, uri.ID, "iso", "光盘管理")
	if inst == nil {
		return
	}
	if err := h.vpsSvc.UnmountISO(c, *inst, uri.IsoName); err != nil {
		h.featureOrNotImplemented(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- 反向代理 ----

func (h *Handler) VPSReverseProxies(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst := h.loadVPSForFeature(c, uri.ID, "reverse_proxy", "反向代理")
	if inst == nil {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListReverseProxies(c, *inst)
		if err != nil {
			h.featureOrNotImplemented(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		if h.denyIfFeatureReadonly(c, *inst, "reverse_proxy", "反向代理") {
			return
		}
		var payload appshared.AutomationReverseProxyInput
		if err := bindJSON(c, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
			return
		}
		if err := h.vpsSvc.CreateReverseProxy(c, *inst, payload); err != nil {
			h.featureOrNotImplemented(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": domain.ErrMethodNotAllowed.Error()})
	}
}

func (h *Handler) VPSReverseProxyUpdate(c *gin.Context) {
	var uri vpsProxyURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	inst := h.loadVPSForFeatureWrite(c, uri.ID, "reverse_proxy", "反向代理")
	if inst == nil {
		return
	}
	var payload appshared.AutomationReverseProxyInput
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.vpsSvc.UpdateReverseProxy(c, *inst, uri.ProxyIndex, payload); err != nil {
		h.featureOrNotImplemented(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSReverseProxyDelete(c *gin.Context) {
	var uri vpsProxyURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	inst := h.loadVPSForFeatureWrite(c, uri.ID, "reverse_proxy", "反向代理")
	if inst == nil {
		return
	}
	if err := h.vpsSvc.DeleteReverseProxy(c, *inst, uri.ProxyIndex); err != nil {
		h.featureOrNotImplemented(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- 启动顺序 ----

func (h *Handler) VPSBootOrder(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst := h.loadVPSForFeature(c, uri.ID, "boot_order", "启动顺序")
	if inst == nil {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.GetBootOrder(c, *inst)
		if err != nil {
			h.featureOrNotImplemented(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPut, http.MethodPost:
		if h.denyIfFeatureReadonly(c, *inst, "boot_order", "启动顺序") {
			return
		}
		var payload struct {
			Items []appshared.AutomationBootOption `json:"items" binding:"required,min=1,max=32,dive"`
		}
		if err := bindJSON(c, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
			return
		}
		if err := h.vpsSvc.SetBootOrder(c, *inst, payload.Items); err != nil {
			h.featureOrNotImplemented(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": domain.ErrMethodNotAllowed.Error()})
	}
}

// ---- PCI / USB 直通 ----

func (h *Handler) VPSPCIDevices(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst := h.loadVPSForFeature(c, uri.ID, "pci", "PCI 直通")
	if inst == nil {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListPCIDevices(c, *inst)
		if err != nil {
			h.featureOrNotImplemented(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		if h.denyIfFeatureReadonly(c, *inst, "pci", "PCI 直通") {
			return
		}
		var payload appshared.AutomationPassthroughInput
		if err := bindJSON(c, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
			return
		}
		if err := h.vpsSvc.SetupPCIDevice(c, *inst, payload); err != nil {
			h.featureOrNotImplemented(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": domain.ErrMethodNotAllowed.Error()})
	}
}

func (h *Handler) VPSUSBDevices(c *gin.Context) {
	var uri vpsIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst := h.loadVPSForFeature(c, uri.ID, "usb", "USB 直通")
	if inst == nil {
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListUSBDevices(c, *inst)
		if err != nil {
			h.featureOrNotImplemented(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		if h.denyIfFeatureReadonly(c, *inst, "usb", "USB 直通") {
			return
		}
		var payload appshared.AutomationPassthroughInput
		if err := bindJSON(c, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
			return
		}
		if err := h.vpsSvc.SetupUSBDevice(c, *inst, payload); err != nil {
			h.featureOrNotImplemented(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": domain.ErrMethodNotAllowed.Error()})
	}
}
