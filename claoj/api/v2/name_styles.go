package v2

import (
	"encoding/json"
	"net/http"
	"regexp"
	"unicode/utf8"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// Name styles: how a user's name looks per judge_profile.display_rank (v1
// hardcoded these in resources/ranks.scss). Stored as JSON in judge_miscconfig
// so admins can change them without a deploy.
const nameStylesKey = "display_rank_styles"

type nameStyle struct {
	Color  string `json:"color"`  // #rgb / #rrggbb, "" = keep the rating colour
	Bold   bool   `json:"bold"`   // bold name
	Strike bool   `json:"strike"` // struck-through name
	Italic bool   `json:"italic"` // italic name
	Prefix string `json:"prefix"` // emoji / short text before the name
}

// v1 ranks.scss.
var defaultNameStyles = map[string]nameStyle{
	"admin":   {Bold: true},
	"staff":   {Color: "#ff888a", Bold: true, Prefix: "🔥"},
	"teacher": {Prefix: "🧑‍🏫"},
	"banned":  {Strike: true, Prefix: "🚫"},
	"tst24":   {Color: "#fb9ec6", Prefix: "👑"},
}

var (
	nameStyleRank  = regexp.MustCompile(`^[a-z0-9_]{1,10}$`) // display_rank is varchar(10)
	nameStyleColor = regexp.MustCompile(`^(#[0-9a-fA-F]{3}|#[0-9a-fA-F]{6})?$`)
)

func loadNameStyles() map[string]nameStyle {
	var cfg models.MiscConfig
	styles := map[string]nameStyle{}
	if db.DB.Where("`key` = ?", nameStylesKey).First(&cfg).Error != nil || json.Unmarshal([]byte(cfg.Value), &styles) != nil {
		return defaultNameStyles
	}
	return styles
}

// validNameStyles returns a user-facing error, or "" when every entry is safe
// to drop into CSS (the frontend builds selectors/content from these).
func validNameStyles(styles map[string]nameStyle) string {
	if len(styles) > 30 {
		return "too many ranks"
	}
	for rank, s := range styles {
		if rank == "user" || !nameStyleRank.MatchString(rank) {
			return "invalid rank: " + rank
		}
		if !nameStyleColor.MatchString(s.Color) {
			return "invalid colour for " + rank
		}
		if utf8.RuneCountInString(s.Prefix) > 8 || !utf8.ValidString(s.Prefix) {
			return "prefix too long for " + rank
		}
	}
	return ""
}

// NameStyles - GET /api/v2/name-styles (public): the styles plus the users
// that have a styled rank, so every page can decorate their names.
func NameStyles(c *gin.Context) {
	styles := loadNameStyles()
	ranks := make([]string, 0, len(styles))
	for r := range styles {
		ranks = append(ranks, r)
	}
	users := map[string]string{}
	if len(ranks) > 0 {
		var rows []struct{ Username, DisplayRank string }
		db.DB.Table("judge_profile p").Select("au.username, p.display_rank").
			Joins("JOIN auth_user au ON au.id = p.user_id").
			Where("p.display_rank IN ?", ranks).Scan(&rows)
		for _, r := range rows {
			users[r.Username] = r.DisplayRank
		}
	}
	c.Header("Cache-Control", "no-cache") // admin edits must show up at once; React Query caches per tab
	c.JSON(http.StatusOK, gin.H{"styles": styles, "users": users})
}

// AdminNameStylesUpdate - PUT /api/v2/admin/name-styles
func AdminNameStylesUpdate(c *gin.Context) {
	var styles map[string]nameStyle
	if err := c.ShouldBindJSON(&styles); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid request body"))
		return
	}
	if msg := validNameStyles(styles); msg != "" {
		c.JSON(http.StatusBadRequest, apiUserError(msg))
		return
	}
	raw, _ := json.Marshal(styles)
	cfg := models.MiscConfig{Key: nameStylesKey}
	if err := db.DB.Where("`key` = ?", nameStylesKey).Assign(models.MiscConfig{Value: string(raw)}).FirstOrCreate(&cfg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"styles": styles})
}
