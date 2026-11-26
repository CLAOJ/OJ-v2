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
		ID         uint   `json:"id"`
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
		items[i] = Item{l.ID, l.Key, l.Name, short, l.CommonName, l.Extension}
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

	// v1 /status lists each judge's runtimes (judge_judge_runtimes).
	type Runtime struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	type Item struct {
		Name      string      `json:"name"`
		Online    bool        `json:"online"`
		Ping      *float64    `json:"ping"`
		Load      *float64    `json:"load"`
		StartTime interface{} `json:"start_time"`
		Runtimes  []Runtime   `json:"runtimes"`
	}
	ids := make([]uint, len(judges))
	for i, j := range judges {
		ids[i] = j.ID
	}
	var rows []struct {
		JudgeID uint
		Key     string
		Name    string
	}
	if len(ids) > 0 {
		if err := db.DB.Table("judge_judge_runtimes jr").
			Select("jr.judge_id, l.`key`, l.name").
			Joins("JOIN judge_language l ON l.id = jr.language_id").
			Where("jr.judge_id IN ?", ids).
			Order("l.name").Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}
	byJudge := map[uint][]Runtime{}
	for _, r := range rows {
		byJudge[r.JudgeID] = append(byJudge[r.JudgeID], Runtime{r.Key, r.Name})
	}
	items := make([]Item, len(judges))
	for i, j := range judges {
		rt := byJudge[j.ID]
		if rt == nil {
			rt = []Runtime{}
		}
		items[i] = Item{j.Name, j.Online, j.Ping, j.Load, j.StartTime, rt}
	}
	c.JSON(http.StatusOK, apiList(items))
}
