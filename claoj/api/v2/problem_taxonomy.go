package v2

import (
	"net/http"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// problemTaxonomyItem is the public shape of a problem group or type. Both
// tables hold only display metadata, so the whole row is safe to expose.
type problemTaxonomyItem struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// ProblemGroupList - GET /api/v2/problem-groups
// @Description List problem categories for the public problem filter (v1's
//
//	Category select is ProblemGroup.objects.all()).
//
// @Tags Problems
// @Summary List problem groups
// @Produce json
// @Success 200 {object} map[string]interface{} "All problem groups"
// @Router /problem-groups [get]
func ProblemGroupList(c *gin.Context) {
	var groups []models.ProblemGroup
	if err := db.DB.Order("full_name").Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	items := make([]problemTaxonomyItem, len(groups))
	for i, g := range groups {
		items[i] = problemTaxonomyItem{ID: g.ID, Name: g.Name, FullName: g.FullName}
	}
	// The tables change a few times a year; let browsers and the proxy keep
	// the answer instead of re-querying on every problem-list load.
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, apiList(items))
}

// ProblemTypeList - GET /api/v2/problem-types
// @Description List problem types for the public problem filter.
// @Tags Problems
// @Summary List problem types
// @Produce json
// @Success 200 {object} map[string]interface{} "All problem types"
// @Router /problem-types [get]
func ProblemTypeList(c *gin.Context) {
	var types []models.ProblemType
	if err := db.DB.Order("full_name").Find(&types).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	items := make([]problemTaxonomyItem, len(types))
	for i, t := range types {
		items[i] = problemTaxonomyItem{ID: t.ID, Name: t.Name, FullName: t.FullName}
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, apiList(items))
}
