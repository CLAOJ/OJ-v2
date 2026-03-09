package v2

import (
	"net/http"

	"github.com/CLAOJ/claoj/db"
	"github.com/gin-gonic/gin"
)

type ProfileUpdateRequest struct {
	About           *string `json:"about"`
	UsernameDisplay *string `json:"display_name"`
}

// UpdateProfile – PATCH /api/v2/user/me
func UpdateProfile(c *gin.Context) {
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var req ProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	updates := make(map[string]interface{})
	if req.About != nil {
		updates["about"] = *req.About
	}
	if req.UsernameDisplay != nil {
		updates["username_display_override"] = *req.UsernameDisplay
	}

	if len(updates) > 0 {
		if err := db.DB.Model(profile).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}
