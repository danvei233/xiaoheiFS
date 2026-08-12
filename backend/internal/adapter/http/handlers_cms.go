package http

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

// normalizedCMSPage lowercases a page key for the public surface so lookups and
// seeded rows stay consistent (avoiding write amplification on page variants).
func normalizedCMSPage(page string) string {
	return strings.ToLower(strings.TrimSpace(page))
}

// normalizeCMSLang falls back unknown browser locales to zh-CN instead of
// rejecting them. The public surface must keep rendering for en-GB, zh-HK etc.
func normalizeCMSLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return "zh-CN"
	}
	// Accept regional variants of the two supported languages.
	if strings.HasPrefix(lang, "zh") {
		return "zh-CN"
	}
	if strings.HasPrefix(lang, "en") {
		return "en-US"
	}
	return "zh-CN"
}

func (h *Handler) CMSBlocksPublic(c *gin.Context) {
	var query struct {
		Page string `form:"page"`
		Lang string `form:"lang"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	page := normalizedCMSPage(query.Page)
	lang := normalizeCMSLang(query.Lang)
	items, err := h.cmsSvc.ListBlocks(c.Request.Context(), page, lang, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	resp := make([]CMSBlockDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCMSBlockDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *Handler) CMSPostsPublic(c *gin.Context) {
	var query struct {
		Lang        string `form:"lang"`
		CategoryKey string `form:"category_key"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	lang := normalizeCMSLang(query.Lang)
	categoryKey := strings.TrimSpace(query.CategoryKey)
	limit, offset := paging(c)
	items, total, err := h.cmsSvc.ListPosts(c.Request.Context(), appshared.CMSPostFilter{CategoryKey: categoryKey, Lang: lang, PublishedOnly: true, Limit: limit, Offset: offset})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	resp := make([]CMSPostDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCMSPostDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func (h *Handler) CMSPostDetailPublic(c *gin.Context) {
	var uri struct {
		Slug string `uri:"slug" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	slug := strings.TrimSpace(uri.Slug)
	post, err := h.cmsSvc.GetPostBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if post.Status != "published" {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSPostDTO(post))
}

func (h *Handler) AdminCMSCategories(c *gin.Context) {
	var query struct {
		Lang string `form:"lang"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	lang := strings.TrimSpace(query.Lang)
	items, err := h.cmsSvc.ListCategories(c.Request.Context(), lang, true)
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
		writeBodyError(c, err)
		return
	}
	key := strings.TrimSpace(payload.Key)
	name := strings.TrimSpace(payload.Name)
	lang := strings.TrimSpace(payload.Lang)
	if lang == "" {
		lang = "zh-CN"
	}
	if key == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrKeyAndNameRequired.Error()})
		return
	}
	visible := true
	if payload.Visible != nil {
		visible = *payload.Visible
	}
	item := domain.CMSCategory{Key: key, Name: name, Lang: lang, SortOrder: payload.SortOrder, Visible: visible}
	if err := h.cmsSvc.CreateCategory(c.Request.Context(), &item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSCategoryDTO(item))
}

func (h *Handler) AdminCMSCategoryUpdate(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	item, err := h.cmsSvc.GetCategory(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
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
		writeBodyError(c, err)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrKeyNameAndLangRequired.Error()})
		return
	}
	if err := h.cmsSvc.UpdateCategory(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSCategoryDTO(item))
}

func (h *Handler) AdminCMSCategoryDelete(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	// A category that still contains posts must not be deleted, otherwise its
	// posts become orphans invisible on the public site.
	if _, total, err := h.cmsSvc.ListPosts(c.Request.Context(), appshared.CMSPostFilter{CategoryID: &uri.ID, Limit: 1}); err == nil && total > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": domain.ErrConflict.Error()})
		return
	}
	if err := h.cmsSvc.DeleteCategory(c.Request.Context(), uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminCMSPosts(c *gin.Context) {
	var query struct {
		Lang       string `form:"lang"`
		Status     string `form:"status"`
		CategoryID *int64 `form:"category_id" binding:"omitempty,gt=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	lang := strings.TrimSpace(query.Lang)
	status := strings.TrimSpace(query.Status)
	limit, offset := paging(c)
	items, total, err := h.cmsSvc.ListPosts(c.Request.Context(), appshared.CMSPostFilter{CategoryID: query.CategoryID, Status: status, Lang: lang, Limit: limit, Offset: offset})
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
		writeBodyError(c, err)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrCategoryIdTitleSlugRequired.Error()})
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
	if err := h.cmsSvc.CreatePost(c.Request.Context(), &post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSPostDTO(post))
}

func (h *Handler) AdminCMSPostUpdate(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	post, err := h.cmsSvc.GetPost(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
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
		writeBodyError(c, err)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrCategoryIdTitleSlugLangRequired.Error()})
		return
	}
	if post.Status == "published" && post.PublishedAt == nil {
		now := time.Now()
		post.PublishedAt = &now
	}
	if err := h.cmsSvc.UpdatePost(c.Request.Context(), post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSPostDTO(post))
}

func (h *Handler) AdminCMSPostDelete(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.cmsSvc.DeletePost(c.Request.Context(), uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminCMSBlocks(c *gin.Context) {
	var query struct {
		Page string `form:"page"`
		Lang string `form:"lang"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	page := strings.TrimSpace(query.Page)
	lang := strings.TrimSpace(query.Lang)
	items, err := h.cmsSvc.ListBlocks(c.Request.Context(), page, lang, true)
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
		writeBodyError(c, err)
		return
	}
	page := strings.TrimSpace(payload.Page)
	typeName := strings.TrimSpace(payload.Type)
	lang := strings.TrimSpace(payload.Lang)
	if lang == "" {
		lang = "zh-CN"
	}
	if page == "" || typeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPageAndTypeRequired.Error()})
		return
	}
	if err := validateCMSPageKey(page); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.ContentJSON != "" && !json.Valid([]byte(payload.ContentJSON)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrContentJsonInvalid.Error()})
		return
	}
	// CustomHTML is sanitized for every block type so a non-custom_html block
	// cannot become an unsanitized storage vector for stored XSS.
	payload.CustomHTML = sanitizeHTML(payload.CustomHTML)
	visible := true
	if payload.Visible != nil {
		visible = *payload.Visible
	}
	block := domain.CMSBlock{Page: page, Type: typeName, Title: payload.Title, Subtitle: payload.Subtitle, ContentJSON: payload.ContentJSON, CustomHTML: payload.CustomHTML, Lang: lang, Visible: visible, SortOrder: payload.SortOrder}
	if err := h.cmsSvc.CreateBlock(c.Request.Context(), &block); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSBlockDTO(block))
}

func (h *Handler) AdminCMSBlockUpdate(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	block, err := h.cmsSvc.GetBlock(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
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
		writeBodyError(c, err)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrContentJsonInvalid.Error()})
			return
		}
		block.ContentJSON = *payload.ContentJSON
	}
	if payload.CustomHTML != nil {
		block.CustomHTML = sanitizeHTML(*payload.CustomHTML)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPageTypeLangRequired.Error()})
		return
	}
	if err := validateCMSPageKey(block.Page); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.cmsSvc.UpdateBlock(c.Request.Context(), block); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSBlockDTO(block))
}

func (h *Handler) AdminCMSBlockDelete(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.cmsSvc.DeleteBlock(c.Request.Context(), uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
