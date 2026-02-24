package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"image/draw"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	appcatalog "xiaoheiplay/internal/app/catalog"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/permissions"
)

func (h *Handler) AdminGoodsTypes(c *gin.Context) {
	if h.goodsTypes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	items, err := h.goodsTypes.List(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminGoodsTypeCreate(c *gin.Context) {
	if h.goodsTypes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload struct {
		Code               string `json:"code"`
		Name               string `json:"name"`
		Active             bool   `json:"active"`
		SortOrder          int    `json:"sort_order"`
		AutomationPluginID string `json:"automation_plugin_id"`
		AutomationInstance string `json:"automation_instance_id"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	gt := &domain.GoodsType{
		Code:                 strings.TrimSpace(payload.Code),
		Name:                 strings.TrimSpace(payload.Name),
		Active:               payload.Active,
		SortOrder:            payload.SortOrder,
		AutomationCategory:   "automation",
		AutomationPluginID:   strings.TrimSpace(payload.AutomationPluginID),
		AutomationInstanceID: strings.TrimSpace(payload.AutomationInstance),
	}
	if err := h.goodsTypes.Create(c, gt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gt)
}

func (h *Handler) AdminGoodsTypeUpdate(c *gin.Context) {
	if h.goodsTypes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Code               string `json:"code"`
		Name               string `json:"name"`
		Active             bool   `json:"active"`
		SortOrder          int    `json:"sort_order"`
		AutomationPluginID string `json:"automation_plugin_id"`
		AutomationInstance string `json:"automation_instance_id"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	gt := domain.GoodsType{
		ID:                   id,
		Code:                 strings.TrimSpace(payload.Code),
		Name:                 strings.TrimSpace(payload.Name),
		Active:               payload.Active,
		SortOrder:            payload.SortOrder,
		AutomationCategory:   "automation",
		AutomationPluginID:   strings.TrimSpace(payload.AutomationPluginID),
		AutomationInstanceID: strings.TrimSpace(payload.AutomationInstance),
	}
	if err := h.goodsTypes.Update(c, gt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminGoodsTypeDelete(c *gin.Context) {
	if h.goodsTypes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.goodsTypes.Delete(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminGoodsTypeSyncAutomation(c *gin.Context) {
	if h.integration == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	mode := c.Query("mode")
	result, err := h.integration.SyncAutomationForGoodsType(c, id, mode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) AdminUploadCreate(c *gin.Context) {
	if h.uploadSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrFileRequired.Error()})
		return
	}
	const maxUploadSize = 20 << 20
	if file.Size > maxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrFileTooLarge.Error()})
		return
	}
	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrFileOpenFailed.Error()})
		return
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(opened, head)
	_ = opened.Close()
	detected := http.DetectContentType(head[:n])
	allowed := map[string]bool{
		"image/png":  true,
		"image/jpeg": true,
		"image/gif":  true,
		"image/webp": true,
	}
	if !allowed[detected] {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrUnsupportedFileType.Error()})
		return
	}
	dateDir := time.Now().Format("20060102")
	if err := os.MkdirAll(filepath.Join("uploads", dateDir), 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrUploadDirError.Error()})
		return
	}
	name := buildUploadName(file.Filename)
	localPath := filepath.Join("uploads", dateDir, name)
	if err := c.SaveUploadedFile(file, localPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSaveFailed.Error()})
		return
	}
	url := "/uploads/" + dateDir + "/" + name
	item := domain.Upload{Name: file.Filename, Path: localPath, URL: url, Mime: detected, Size: file.Size, UploaderID: getUserID(c)}
	if err := h.uploadSvc.Create(c, &item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUploadDTO(item))
}

func (h *Handler) AdminUploads(c *gin.Context) {
	if h.uploadSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	limit, offset := paging(c)
	items, total, err := h.uploadSvc.List(c, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]UploadDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toUploadDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func validateCMSPageKey(page string) error {
	page = strings.TrimSpace(page)
	if page == "" {
		return domain.ErrCMSPageRequired
	}
	if strings.Contains(page, "..") || strings.ContainsAny(page, "/\\") {
		return domain.ErrCMSPageInvalid
	}
	switch strings.ToLower(page) {
	case "api", "admin", "uploads", "assets", "static", "install":
		return domain.ErrCMSPageReserved
	default:
		return nil
	}
}

func buildUploadName(original string) string {
	ext := filepath.Ext(original)
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	random := fmt.Sprintf("%x", buf)
	return time.Now().Format("150405") + "_" + random + ext
}

func (h *Handler) AdminPermissions(c *gin.Context) {
	if h.permissionSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	perms, err := h.permissionSvc.ListPermissions(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tree := buildPermissionTree(perms)
	c.JSON(http.StatusOK, tree)
}

func (h *Handler) AdminPermissionsList(c *gin.Context) {
	if h.permissionSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	perms, err := h.permissionSvc.ListPermissions(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]permissionItemDTO, 0, len(perms))
	for _, perm := range perms {
		items = append(items, toPermissionDTO(perm))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminPermissionDetail(c *gin.Context) {
	if h.permissionSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	code := c.Param("code")
	perm, err := h.permissionSvc.GetPermissionByCode(c, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrPermissionNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, toPermissionDTO(perm))
}

func (h *Handler) AdminPermissionsUpdate(c *gin.Context) {
	if h.permissionSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	code := c.Param("code")
	var payload struct {
		Name         *string `json:"name"`
		FriendlyName *string `json:"friendly_name"`
		Category     *string `json:"category"`
		ParentCode   *string `json:"parent_code"`
		SortOrder    *int    `json:"sort_order"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	perm, err := h.permissionSvc.GetPermissionByCode(c, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrPermissionNotFound.Error()})
		return
	}
	if payload.Name != nil {
		perm.Name = strings.TrimSpace(*payload.Name)
	}
	if payload.FriendlyName != nil {
		perm.FriendlyName = strings.TrimSpace(*payload.FriendlyName)
	}
	if payload.Category != nil {
		perm.Category = strings.TrimSpace(*payload.Category)
	}
	if payload.ParentCode != nil {
		perm.ParentCode = strings.TrimSpace(*payload.ParentCode)
	}
	if payload.SortOrder != nil {
		perm.SortOrder = *payload.SortOrder
	}
	if perm.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNameRequired.Error()})
		return
	}
	if perm.Category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrCategoryRequired.Error()})
		return
	}
	if err := h.permissionSvc.UpsertPermission(c, &perm); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPermissionDTO(perm))
}

func (h *Handler) AdminPermissionsSync(c *gin.Context) {
	if h.permissionSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	perms := permissions.GetDefinitions()
	if err := h.permissionSvc.RegisterPermissions(c, perms); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "total": len(perms)})
}

type permissionItemDTO struct {
	Code         string               `json:"code"`
	Name         string               `json:"name"`
	FriendlyName string               `json:"friendly_name"`
	Category     string               `json:"category"`
	ParentCode   string               `json:"parent_code,omitempty"`
	SortOrder    int                  `json:"sort_order"`
	Children     []*permissionItemDTO `json:"children,omitempty"`
}

func toPermissionDTO(perm domain.Permission) permissionItemDTO {
	return permissionItemDTO{
		Code:         perm.Code,
		Name:         perm.Name,
		FriendlyName: perm.FriendlyName,
		Category:     perm.Category,
		ParentCode:   perm.ParentCode,
		SortOrder:    perm.SortOrder,
	}
}

func buildPermissionTree(perms []domain.Permission) []*permissionItemDTO {
	nodes := make(map[string]*permissionItemDTO, len(perms))
	for _, perm := range perms {
		item := toPermissionDTO(perm)
		nodes[perm.Code] = &item
	}

	roots := make([]*permissionItemDTO, 0)
	for _, perm := range perms {
		node := nodes[perm.Code]
		if perm.ParentCode != "" {
			parent, ok := nodes[perm.ParentCode]
			if ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}

	sortPermissionNodes(roots)

	return roots
}

func sortPermissionNodes(nodes []*permissionItemDTO) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].SortOrder != nodes[j].SortOrder {
			return nodes[i].SortOrder < nodes[j].SortOrder
		}
		return nodes[i].Code < nodes[j].Code
	})
	for i := range nodes {
		if len(nodes[i].Children) == 0 {
			continue
		}
		sortPermissionNodes(nodes[i].Children)
	}
}

func renderCaptcha(code string) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 120, 40))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{240, 240, 240, 255}}, image.Point{}, draw.Src)
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{30, 30, 30, 255}),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(10, 25),
	}
	d.DrawString(code)
	return img
}

func parseHostIDLocal(v string) int64 {
	var id int64
	_, _ = fmt.Sscan(v, &id)
	return id
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func paging(c *gin.Context) (int, int) {
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = v
		}
	}
	page := 0
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			page = v
		}
	}
	if p := c.Query("pages"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			limit = v
		}
	}
	if p := c.Query("page_size"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			limit = v
		}
	}
	if page > 0 && limit > 0 {
		offset = (page - 1) * limit
	}
	return limit, offset
}

func listVisiblePlanGroups(catalog *appcatalog.Service, ctx *gin.Context) []domain.PlanGroup {
	items, err := catalog.ListPlanGroups(ctx)
	if err != nil {
		return nil
	}
	return filterVisiblePlanGroups(items)
}

func filterVisiblePlanGroups(items []domain.PlanGroup) []domain.PlanGroup {
	if len(items) == 0 {
		return items
	}
	out := make([]domain.PlanGroup, 0, len(items))
	for _, item := range items {
		if item.Active && item.Visible {
			out = append(out, item)
		}
	}
	return out
}

func filterVisibleRegions(items []domain.Region) []domain.Region {
	if len(items) == 0 {
		return items
	}
	out := make([]domain.Region, 0, len(items))
	for _, item := range items {
		if item.Active && item.Visible {
			out = append(out, item)
		}
	}
	return out
}

func filterVisiblePackages(items []domain.Package, plans []domain.PlanGroup) []domain.Package {
	if len(items) == 0 {
		return items
	}
	planIndex := make(map[int64]struct{}, len(plans))
	for _, plan := range plans {
		planIndex[plan.ID] = struct{}{}
	}
	out := make([]domain.Package, 0, len(items))
	for _, item := range items {
		if !item.Active || !item.Visible {
			continue
		}
		if _, ok := planIndex[item.PlanGroupID]; !ok {
			continue
		}
		out = append(out, item)
	}
	return out
}

func filterEnabledSystemImages(items []domain.SystemImage, plans []domain.PlanGroup) []domain.SystemImage {
	if len(items) == 0 {
		return items
	}
	out := make([]domain.SystemImage, 0, len(items))
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		out = append(out, item)
	}
	return out
}

// filterRegionsWithPackages 过滤掉没有商品的地区
func filterRegionsWithPackages(regions []domain.Region, plans []domain.PlanGroup, packages []domain.Package) []domain.Region {
	if len(regions) == 0 || len(packages) == 0 {
		return []domain.Region{}
	}
	
	// 构建套餐组到地区的映射
	planToRegion := make(map[int64]int64)
	for _, plan := range plans {
		planToRegion[plan.ID] = plan.RegionID
	}
	
	// 构建地区ID索引：记录哪些地区有商品
	regionHasPackage := make(map[int64]bool)
	
	// 遍历所有商品，通过 PlanGroupID 找到对应的 RegionID
	for _, pkg := range packages {
		if regionID, ok := planToRegion[pkg.PlanGroupID]; ok {
			regionHasPackage[regionID] = true
		}
	}
	
	out := make([]domain.Region, 0, len(regions))
	for _, region := range regions {
		if regionHasPackage[region.ID] {
			out = append(out, region)
		}
	}
	return out
}

// filterPlanGroupsWithPackages 过滤掉没有商品的套餐组
func filterPlanGroupsWithPackages(plans []domain.PlanGroup, packages []domain.Package) []domain.PlanGroup {
	if len(plans) == 0 || len(packages) == 0 {
		return []domain.PlanGroup{}
	}
	// 构建套餐组ID索引
	planHasPackage := make(map[int64]bool)
	for _, pkg := range packages {
		planHasPackage[pkg.PlanGroupID] = true
	}
	
	out := make([]domain.PlanGroup, 0, len(plans))
	for _, plan := range plans {
		if planHasPackage[plan.ID] {
			out = append(out, plan)
		}
	}
	return out
}

func verifyHMAC(body []byte, secret string, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := fmt.Sprintf("%x", mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expected)))
}
