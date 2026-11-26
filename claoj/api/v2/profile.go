package v2

import (
	"net/http"
	"time"
	_ "time/tzdata" // validate timezones without relying on the image's zoneinfo

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

type ProfileUpdateRequest struct {
	About           *string `json:"about"`
	UsernameDisplay *string `json:"display_name"`
	Timezone        *string `json:"timezone"`
	LanguageID      *uint   `json:"language_id"`
	AceTheme        *string `json:"ace_theme"`
	SiteTheme       *string `json:"site_theme"`
}

// v1 judge/models/choices.py ACE_THEMES / SITE_THEMES (judge_profile is shared).
var aceThemes = map[string]bool{
	"auto": true, "ambiance": true, "chaos": true, "chrome": true, "clouds": true, "clouds_midnight": true,
	"cobalt": true, "crimson_editor": true, "dawn": true, "dreamweaver": true, "eclipse": true, "github": true,
	"idle_fingers": true, "katzenmilch": true, "kr_theme": true, "kuroir": true, "merbivore": true,
	"merbivore_soft": true, "mono_industrial": true, "monokai": true, "pastel_on_dark": true,
	"solarized_dark": true, "solarized_light": true, "terminal": true, "textmate": true, "tomorrow": true,
	"tomorrow_night": true, "tomorrow_night_blue": true, "tomorrow_night_bright": true,
	"tomorrow_night_eighties": true, "twilight": true, "vibrant_ink": true, "xcode": true,
}
var siteThemes = map[string]bool{"auto": true, "light": true, "dark": true}

// profilePrefErrors validates the preference fields; "" when all are fine.
func profilePrefErrors(req ProfileUpdateRequest) string {
	if req.Timezone != nil {
		if _, err := time.LoadLocation(*req.Timezone); err != nil || *req.Timezone == "" || len(*req.Timezone) > 50 {
			return "invalid timezone"
		}
	}
	if req.AceTheme != nil && !aceThemes[*req.AceTheme] {
		return "invalid editor theme"
	}
	if req.SiteTheme != nil && !siteThemes[*req.SiteTheme] {
		return "invalid site theme"
	}
	if req.LanguageID != nil {
		var n int64
		db.DB.Model(&models.Language{}).Where("id = ?", *req.LanguageID).Count(&n)
		if n == 0 {
			return "invalid language"
		}
	}
	return ""
}

// UpdateProfile – PATCH /api/v2/user/me
func UpdateProfile(c *gin.Context) {
	_, profile, ok := resolveUserProfile(c)
	if !ok {
		return
	}

	var req ProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiBindError(err))
		return
	}

	if msg := profilePrefErrors(req); msg != "" {
		c.JSON(http.StatusBadRequest, apiUserError(msg))
		return
	}

	updates := make(map[string]interface{})
	if req.Timezone != nil {
		updates["timezone"] = *req.Timezone
	}
	if req.LanguageID != nil {
		updates["language_id"] = *req.LanguageID
	}
	if req.AceTheme != nil {
		updates["ace_theme"] = *req.AceTheme
	}
	if req.SiteTheme != nil {
		updates["site_theme"] = *req.SiteTheme
	}
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
