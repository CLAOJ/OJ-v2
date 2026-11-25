package v2

import (
	"net/http"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// PublicLicenseDetail - GET /api/v2/license/:key
// @Description A license's markdown text and source link (v1 LicenseDetail).
// @Tags Problems
// @Summary License detail
// @Produce json
// @Param key path string true "License key"
// @Success 200 {object} map[string]interface{} "License"
// @Failure 404 {object} map[string]interface{} "License not found"
// @Router /license/{key} [get]
func PublicLicenseDetail(c *gin.Context) {
	var license models.License
	if err := db.DB.Where("`key` = ?", c.Param("key")).First(&license).Error; err != nil {
		c.JSON(http.StatusNotFound, apiUserError("license not found"))
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{
		"id":      license.ID,
		"key":     license.Key,
		"name":    license.Name,
		"link":    license.Link,
		"display": license.Display,
		"icon":    license.Icon,
		"text":    license.Text,
	})
}
