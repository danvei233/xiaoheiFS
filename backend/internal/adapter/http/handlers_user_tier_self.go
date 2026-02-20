package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) MeUserTier(c *gin.Context) {
	userID := getUserID(c)
	user, err := h.authSvc.GetUser(c, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	resp := gin.H{
		"group_id":  0,
		"expire_at": user.UserTierExpireAt,
	}
	if user.UserTierGroupID == nil || *user.UserTierGroupID <= 0 {
		c.JSON(http.StatusOK, resp)
		return
	}

	resp["group_id"] = *user.UserTierGroupID
	if h.userTierSvc == nil {
		c.JSON(http.StatusOK, resp)
		return
	}

	group, err := h.userTierSvc.GetGroup(c, *user.UserTierGroupID)
	if err != nil {
		c.JSON(http.StatusOK, resp)
		return
	}

	resp["group_name"] = group.Name
	resp["group_color"] = group.Color
	resp["group_icon"] = group.Icon
	resp["group_priority"] = group.Priority
	resp["is_default"] = group.IsDefault
	c.JSON(http.StatusOK, resp)
}
