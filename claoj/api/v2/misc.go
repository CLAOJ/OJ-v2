package v2

import (
	"net/http"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// LanguageList – GET /api/v2/languages
func LanguageList(c *gin.Context) {
	var langs []models.Language
	if err := db.DB.Order("`key` ASC").Find(&langs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		Key        string `json:"key"`
		Name       string `json:"name"`
		ShortName  string `json:"short_name"`
		CommonName string `json:"common_name"`
		Extension  string `json:"extension"`
	}
	items := make([]Item, len(langs))
	for i, l := range langs {
		short := l.Key
		if l.ShortName != nil {
			short = *l.ShortName
		}
		items[i] = Item{l.Key, l.Name, short, l.CommonName, l.Extension}
	}
	c.JSON(http.StatusOK, apiList(items))
}

// JudgeList – GET /api/v2/judges
func JudgeList(c *gin.Context) {
	var judges []models.Judge
	if err := db.DB.
		Where("online = ?", true).
		Order("name ASC").
		Find(&judges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	type Item struct {
		Name      string      `json:"name"`
		Online    bool        `json:"online"`
		Ping      *float64    `json:"ping"`
		Load      *float64    `json:"load"`
		StartTime interface{} `json:"start_time"`
	}
	items := make([]Item, len(judges))
	for i, j := range judges {
		items[i] = Item{j.Name, j.Online, j.Ping, j.Load, j.StartTime}
	}
	c.JSON(http.StatusOK, apiList(items))
}
