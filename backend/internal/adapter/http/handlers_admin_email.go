package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) AdminEmailTemplates(c *gin.Context) {
	items, err := h.adminSvc.ListEmailTemplates(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	if len(items) == 0 && h.adminSvc != nil {
		for _, tmpl := range defaultEmailTemplates() {
			cp := tmpl
			_ = h.adminSvc.UpsertEmailTemplate(c, getUserID(c), &cp)
		}
		items, _ = h.adminSvc.ListEmailTemplates(c)
	}
	c.JSON(http.StatusOK, gin.H{"items": toEmailTemplateDTOs(items)})
}

func (h *Handler) AdminEmailTemplateUpsert(c *gin.Context) {
	var payload EmailTemplateDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	var uri struct {
		ID int64 `uri:"id" binding:"required,gt=0"`
	}
	if strings.TrimSpace(c.Param("id")) != "" {
		if err := c.ShouldBindUri(&uri); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
			return
		}
		payload.ID = uri.ID
	}
	tmpl := emailTemplateDTOToDomain(payload)
	if err := h.adminSvc.UpsertEmailTemplate(c, getUserID(c), &tmpl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toEmailTemplateDTO(tmpl))
}

func (h *Handler) AdminEmailTemplateDelete(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if h.adminSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	if err := h.adminSvc.DeleteEmailTemplate(c, getUserID(c), uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
