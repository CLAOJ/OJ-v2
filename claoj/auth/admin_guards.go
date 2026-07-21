package auth

import (
	"net/http"
	"strconv"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequirePerm aborts with 403 unless the request user holds the Django
// permission (superuser bypass is built into HasPerm).
func RequirePerm(codename string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !HasPerm(c, codename) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
		c.Next()
	}
}

// RequireProblemEdit loads the problem named by the :code path param with its
// author/curator associations and aborts unless the request user may edit it
// (auth.CanEditProblem). 404 if the problem does not exist.
func RequireProblemEdit() gin.HandlerFunc {
	return func(c *gin.Context) {
		var problem models.Problem
		if err := db.DB.Preload("Authors").Preload("Curators").
			Where("code = ?", c.Param("code")).First(&problem).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "problem not found"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if !CanEditProblem(c, &problem) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "you do not have permission to edit this problem"})
			return
		}
		c.Next()
	}
}

// RequireContestEdit loads the contest named by the :key path param with its
// author/curator associations and aborts unless the request user may edit it
// (auth.CanEditContest). 404 if the contest does not exist. The struct-condition
// Where quotes the reserved "key" column correctly per dialect.
func RequireContestEdit() gin.HandlerFunc {
	return func(c *gin.Context) {
		var contest models.Contest
		if err := db.DB.Preload("Authors").Preload("Curators").
			Where(&models.Contest{Key: c.Param("key")}).First(&contest).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "contest not found"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if !CanEditContest(c, &contest) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "you do not have permission to edit this contest"})
			return
		}
		c.Next()
	}
}

// RequireOrgEdit aborts unless the request user may edit the organization named
// by the :id path param (auth.CanEditOrganization).
func RequireOrgEdit() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
			return
		}
		if !CanEditOrganization(c, uint(orgID)) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "you do not have permission to edit this organization"})
			return
		}
		c.Next()
	}
}
