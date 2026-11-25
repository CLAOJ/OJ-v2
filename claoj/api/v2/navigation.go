package v2

import (
	"net/http"

	"github.com/CLAOJ/claoj/service/navigation"
	"github.com/gin-gonic/gin"
)

// publicNavigationNode is the JSON shape of one navbar node.
type publicNavigationNode struct {
	Key      string                 `json:"key"`
	Label    string                 `json:"label"`
	Path     string                 `json:"path"`
	Regex    string                 `json:"regex"`
	Children []publicNavigationNode `json:"children"`
}

func toPublicNavigationNodes(items []navigation.PublicNavigationItem) []publicNavigationNode {
	nodeList := make([]publicNavigationNode, len(items))
	for i, item := range items {
		nodeList[i] = publicNavigationNode{
			Key:      item.Key,
			Label:    item.Label,
			Path:     item.Path,
			Regex:    item.Regex,
			Children: toPublicNavigationNodes(item.Children),
		}
	}
	return nodeList
}

// PublicNavigation - GET /api/navigation
// The site header's items: v1's judge_navigationbar table, no auth required.
func PublicNavigation(c *gin.Context) {
	items, err := getNavigationService().PublicNavigation()
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, apiList(toPublicNavigationNodes(items)))
}