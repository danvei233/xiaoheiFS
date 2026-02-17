package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) CMSBlocksPublic(c *gin.Context) {
	page := strings.TrimSpace(c.Query("page"))
	lang := strings.TrimSpace(c.Query("lang"))
	if lang == "" {
		lang = "zh-CN"
	}
	items, err := h.cmsSvc.ListBlocks(c, page, lang, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]CMSBlockDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCMSBlockDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *Handler) CMSPostsPublic(c *gin.Context) {
	lang := strings.TrimSpace(c.Query("lang"))
	if lang == "" {
		lang = "zh-CN"
	}
	categoryKey := strings.TrimSpace(c.Query("category_key"))
	limit, offset := paging(c)
	items, total, err := h.cmsSvc.ListPosts(c, appshared.CMSPostFilter{CategoryKey: categoryKey, Lang: lang, PublishedOnly: true, Limit: limit, Offset: offset})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]CMSPostDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCMSPostDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func (h *Handler) CMSPostDetailPublic(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	post, err := h.cmsSvc.GetPostBySlug(c, slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if post.Status != "published" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, toCMSPostDTO(post))
}

func (h *Handler) AdminCMSCategories(c *gin.Context) {
	lang := strings.TrimSpace(c.Query("lang"))
	items, err := h.cmsSvc.ListCategories(c, lang, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]CMSCategoryDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCMSCategoryDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *Handler) AdminCMSCategoryCreate(c *gin.Context) {
	var payload struct {
		Key       string `json:"key"`
		Name      string `json:"name"`
		Lang      string `json:"lang"`
		SortOrder int    `json:"sort_order"`
		Visible   *bool  `json:"visible"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	key := strings.TrimSpace(payload.Key)
	name := strings.TrimSpace(payload.Name)
	lang := strings.TrimSpace(payload.Lang)
	if lang == "" {
		lang = "zh-CN"
	}
	if key == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key and name required"})
		return
	}
	visible := true
	if payload.Visible != nil {
		visible = *payload.Visible
	}
	item := domain.CMSCategory{Key: key, Name: name, Lang: lang, SortOrder: payload.SortOrder, Visible: visible}
	if err := h.cmsSvc.CreateCategory(c, &item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSCategoryDTO(item))
}

func (h *Handler) AdminCMSCategoryUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	item, err := h.cmsSvc.GetCategory(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var payload struct {
		Key       *string `json:"key"`
		Name      *string `json:"name"`
		Lang      *string `json:"lang"`
		SortOrder *int    `json:"sort_order"`
		Visible   *bool   `json:"visible"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.Key != nil {
		item.Key = strings.TrimSpace(*payload.Key)
	}
	if payload.Name != nil {
		item.Name = strings.TrimSpace(*payload.Name)
	}
	if payload.Lang != nil {
		item.Lang = strings.TrimSpace(*payload.Lang)
	}
	if payload.SortOrder != nil {
		item.SortOrder = *payload.SortOrder
	}
	if payload.Visible != nil {
		item.Visible = *payload.Visible
	}
	if item.Key == "" || item.Name == "" || item.Lang == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key, name and lang required"})
		return
	}
	if err := h.cmsSvc.UpdateCategory(c, item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSCategoryDTO(item))
}

func (h *Handler) AdminCMSCategoryDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.cmsSvc.DeleteCategory(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminCMSPosts(c *gin.Context) {
	lang := strings.TrimSpace(c.Query("lang"))
	status := strings.TrimSpace(c.Query("status"))
	categoryIDRaw := strings.TrimSpace(c.Query("category_id"))
	limit, offset := paging(c)
	var categoryID *int64
	if categoryIDRaw != "" {
		if v, err := strconv.ParseInt(categoryIDRaw, 10, 64); err == nil {
			categoryID = &v
		}
	}
	items, total, err := h.cmsSvc.ListPosts(c, appshared.CMSPostFilter{CategoryID: categoryID, Status: status, Lang: lang, Limit: limit, Offset: offset})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]CMSPostDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCMSPostDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func (h *Handler) AdminCMSPostCreate(c *gin.Context) {
	var payload struct {
		CategoryID  int64  `json:"category_id"`
		Title       string `json:"title"`
		Slug        string `json:"slug"`
		Summary     string `json:"summary"`
		ContentHTML string `json:"content_html"`
		CoverURL    string `json:"cover_url"`
		Lang        string `json:"lang"`
		Status      string `json:"status"`
		Pinned      bool   `json:"pinned"`
		SortOrder   int    `json:"sort_order"`
		PublishedAt string `json:"published_at"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	lang := strings.TrimSpace(payload.Lang)
	if lang == "" {
		lang = "zh-CN"
	}
	status := strings.TrimSpace(payload.Status)
	if status == "" {
		status = "draft"
	}
	if payload.CategoryID == 0 || strings.TrimSpace(payload.Title) == "" || strings.TrimSpace(payload.Slug) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id, title, slug required"})
		return
	}
	payload.ContentHTML = sanitizeHTML(payload.ContentHTML)
	var publishedAt *time.Time
	if payload.PublishedAt != "" {
		if t, err := time.Parse(time.RFC3339, payload.PublishedAt); err == nil {
			publishedAt = &t
		}
	}
	if status == "published" && publishedAt == nil {
		now := time.Now()
		publishedAt = &now
	}
	post := domain.CMSPost{CategoryID: payload.CategoryID, Title: strings.TrimSpace(payload.Title), Slug: strings.TrimSpace(payload.Slug), Summary: payload.Summary, ContentHTML: payload.ContentHTML, CoverURL: payload.CoverURL, Lang: lang, Status: status, Pinned: payload.Pinned, SortOrder: payload.SortOrder, PublishedAt: publishedAt}
	if err := h.cmsSvc.CreatePost(c, &post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSPostDTO(post))
}

func (h *Handler) AdminCMSPostUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	post, err := h.cmsSvc.GetPost(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var payload struct {
		CategoryID  *int64  `json:"category_id"`
		Title       *string `json:"title"`
		Slug        *string `json:"slug"`
		Summary     *string `json:"summary"`
		ContentHTML *string `json:"content_html"`
		CoverURL    *string `json:"cover_url"`
		Lang        *string `json:"lang"`
		Status      *string `json:"status"`
		Pinned      *bool   `json:"pinned"`
		SortOrder   *int    `json:"sort_order"`
		PublishedAt *string `json:"published_at"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.CategoryID != nil {
		post.CategoryID = *payload.CategoryID
	}
	if payload.Title != nil {
		post.Title = strings.TrimSpace(*payload.Title)
	}
	if payload.Slug != nil {
		post.Slug = strings.TrimSpace(*payload.Slug)
	}
	if payload.Summary != nil {
		post.Summary = *payload.Summary
	}
	if payload.ContentHTML != nil {
		post.ContentHTML = sanitizeHTML(*payload.ContentHTML)
	}
	if payload.CoverURL != nil {
		post.CoverURL = *payload.CoverURL
	}
	if payload.Lang != nil {
		post.Lang = strings.TrimSpace(*payload.Lang)
	}
	if payload.Status != nil {
		post.Status = strings.TrimSpace(*payload.Status)
	}
	if payload.Pinned != nil {
		post.Pinned = *payload.Pinned
	}
	if payload.SortOrder != nil {
		post.SortOrder = *payload.SortOrder
	}
	if payload.PublishedAt != nil {
		if *payload.PublishedAt == "" {
			post.PublishedAt = nil
		} else if t, err := time.Parse(time.RFC3339, *payload.PublishedAt); err == nil {
			post.PublishedAt = &t
		}
	}
	if post.CategoryID == 0 || post.Title == "" || post.Slug == "" || post.Lang == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id, title, slug, lang required"})
		return
	}
	if post.Status == "published" && post.PublishedAt == nil {
		now := time.Now()
		post.PublishedAt = &now
	}
	if err := h.cmsSvc.UpdatePost(c, post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSPostDTO(post))
}

func (h *Handler) AdminCMSPostDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.cmsSvc.DeletePost(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminCMSBlocks(c *gin.Context) {
	page := strings.TrimSpace(c.Query("page"))
	lang := strings.TrimSpace(c.Query("lang"))
	items, err := h.cmsSvc.ListBlocks(c, page, lang, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]CMSBlockDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCMSBlockDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *Handler) AdminCMSBlockCreate(c *gin.Context) {
	var payload struct {
		Page        string `json:"page"`
		Type        string `json:"type"`
		Title       string `json:"title"`
		Subtitle    string `json:"subtitle"`
		ContentJSON string `json:"content_json"`
		CustomHTML  string `json:"custom_html"`
		Lang        string `json:"lang"`
		Visible     *bool  `json:"visible"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	page := strings.TrimSpace(payload.Page)
	typeName := strings.TrimSpace(payload.Type)
	lang := strings.TrimSpace(payload.Lang)
	if lang == "" {
		lang = "zh-CN"
	}
	if page == "" || typeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page and type required"})
		return
	}
	if err := validateCMSPageKey(page); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.ContentJSON != "" && !json.Valid([]byte(payload.ContentJSON)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content_json invalid"})
		return
	}
	if typeName == "custom_html" {
		payload.CustomHTML = sanitizeHTML(payload.CustomHTML)
	}
	visible := true
	if payload.Visible != nil {
		visible = *payload.Visible
	}
	block := domain.CMSBlock{Page: page, Type: typeName, Title: payload.Title, Subtitle: payload.Subtitle, ContentJSON: payload.ContentJSON, CustomHTML: payload.CustomHTML, Lang: lang, Visible: visible, SortOrder: payload.SortOrder}
	if err := h.cmsSvc.CreateBlock(c, &block); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSBlockDTO(block))
}

func (h *Handler) AdminCMSBlockUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	block, err := h.cmsSvc.GetBlock(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var payload struct {
		Page        *string `json:"page"`
		Type        *string `json:"type"`
		Title       *string `json:"title"`
		Subtitle    *string `json:"subtitle"`
		ContentJSON *string `json:"content_json"`
		CustomHTML  *string `json:"custom_html"`
		Lang        *string `json:"lang"`
		Visible     *bool   `json:"visible"`
		SortOrder   *int    `json:"sort_order"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.Page != nil {
		block.Page = strings.TrimSpace(*payload.Page)
	}
	if payload.Type != nil {
		block.Type = strings.TrimSpace(*payload.Type)
	}
	if payload.Title != nil {
		block.Title = *payload.Title
	}
	if payload.Subtitle != nil {
		block.Subtitle = *payload.Subtitle
	}
	if payload.ContentJSON != nil {
		if *payload.ContentJSON != "" && !json.Valid([]byte(*payload.ContentJSON)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "content_json invalid"})
			return
		}
		block.ContentJSON = *payload.ContentJSON
	}
	if payload.CustomHTML != nil {
		if block.Type == "custom_html" {
			block.CustomHTML = sanitizeHTML(*payload.CustomHTML)
		} else {
			block.CustomHTML = *payload.CustomHTML
		}
	}
	if payload.Lang != nil {
		block.Lang = strings.TrimSpace(*payload.Lang)
	}
	if payload.Visible != nil {
		block.Visible = *payload.Visible
	}
	if payload.SortOrder != nil {
		block.SortOrder = *payload.SortOrder
	}
	if block.Page == "" || block.Type == "" || block.Lang == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page, type, lang required"})
		return
	}
	if err := validateCMSPageKey(block.Page); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.cmsSvc.UpdateBlock(c, block); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSBlockDTO(block))
}

func (h *Handler) AdminCMSBlockDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.cmsSvc.DeleteBlock(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
