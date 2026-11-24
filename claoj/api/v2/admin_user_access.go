package v2

import (
	"errors"
	"net/http"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// ADMIN USER ACCESS (GROUPS, DIRECT PERMISSIONS, 2FA) API
//
// Writes rows into Django's own auth_user_groups / auth_user_user_permissions
// tables and judge_profile's security columns, exactly like Django admin does.
// :id is a judge_profile.id (the convention of this handler family).
// ============================================================

// AdminUserReplaceGroups - PUT /api/v2/admin/user/:id/groups
// Replaces the user's full group membership with the supplied id list
// (an empty list clears it).
func AdminUserReplaceGroups(c *gin.Context) {
	if !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("superuser access required"))
		return
	}

	var profileID uint
	if err := parseUint(c.Param("id"), &profileID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user id"))
		return
	}

	var input struct {
		GroupIDs *[]uint `json:"group_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}
	if input.GroupIDs == nil {
		c.JSON(http.StatusBadRequest, apiUserError("group_ids is required"))
		return
	}

	groupIDs := dedupeIDs(*input.GroupIDs)
	if len(groupIDs) > 0 {
		var count int64
		if err := db.DB.Model(&models.AuthGroup{}).Where("id IN ?", groupIDs).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
		if int(count) != len(groupIDs) {
			c.JSON(http.StatusBadRequest, apiUserError("one or more group IDs do not exist"))
			return
		}
	}

	authUserID, ok := resolveAuthUserID(c, profileID)
	if !ok {
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", authUserID).Delete(&models.AuthUserGroup{}).Error; err != nil {
			return err
		}
		for _, gid := range groupIDs {
			if err := tx.Create(&models.AuthUserGroup{UserID: authUserID, GroupID: gid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	auth.BumpPermVersion()
	c.JSON(http.StatusOK, gin.H{"success": true, "group_ids": groupIDs})
}

// AdminUserReplacePermissions - PUT /api/v2/admin/user/:id/permissions
// Replaces the user's direct auth_user_user_permissions rows with the
// supplied permission id list (an empty list clears them).
func AdminUserReplacePermissions(c *gin.Context) {
	if !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("superuser access required"))
		return
	}

	var profileID uint
	if err := parseUint(c.Param("id"), &profileID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user id"))
		return
	}

	var input struct {
		PermissionIDs *[]uint `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}
	if input.PermissionIDs == nil {
		c.JSON(http.StatusBadRequest, apiUserError("permission_ids is required"))
		return
	}

	permissionIDs := dedupeIDs(*input.PermissionIDs)
	if len(permissionIDs) > 0 {
		var count int64
		if err := db.DB.Model(&models.AuthPermission{}).Where("id IN ?", permissionIDs).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
		if int(count) != len(permissionIDs) {
			c.JSON(http.StatusBadRequest, apiUserError("one or more permission IDs do not exist"))
			return
		}
	}

	authUserID, ok := resolveAuthUserID(c, profileID)
	if !ok {
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", authUserID).Delete(&models.AuthUserPermission{}).Error; err != nil {
			return err
		}
		for _, pid := range permissionIDs {
			if err := tx.Create(&models.AuthUserPermission{UserID: authUserID, PermissionID: pid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	auth.BumpPermVersion()
	c.JSON(http.StatusOK, gin.H{"success": true, "permission_ids": permissionIDs})
}

// AdminUserSecurity - POST /api/v2/admin/user/:id/security
//
// Disable-only: clearing totp_enabled wipes the encrypted TOTP key and
// scratch codes, and clearing webauthn_enabled deletes the user's WebAuthn
// credentials. Enabling either is refused -- it must go through the user's
// own setup flow, which proves possession of the second factor.
func AdminUserSecurity(c *gin.Context) {
	if !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiError("superuser access required"))
		return
	}

	var profileID uint
	if err := parseUint(c.Param("id"), &profileID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid user id"))
		return
	}

	var input struct {
		TotpEnabled     *bool `json:"totp_enabled"`
		WebauthnEnabled *bool `json:"webauthn_enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}
	if input.TotpEnabled == nil && input.WebauthnEnabled == nil {
		c.JSON(http.StatusBadRequest, apiUserError("totp_enabled or webauthn_enabled is required"))
		return
	}
	if (input.TotpEnabled != nil && *input.TotpEnabled) || (input.WebauthnEnabled != nil && *input.WebauthnEnabled) {
		c.JSON(http.StatusBadRequest, apiUserError("2FA can only be disabled here; the user must enable it through their own setup flow"))
		return
	}

	var profile models.Profile
	if err := db.DB.First(&profile, profileID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("user not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if input.TotpEnabled != nil {
			if err := tx.Model(&models.Profile{}).Where("id = ?", profile.ID).Updates(map[string]interface{}{
				"is_totp_enabled": false,
				"totp_key":        nil,
				"scratch_codes":   nil,
			}).Error; err != nil {
				return err
			}
		}
		if input.WebauthnEnabled != nil {
			if err := tx.Where("user_id = ?", profile.ID).Delete(&models.WebAuthnCredential{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Profile{}).Where("id = ?", profile.ID).
				Update("is_webauthn_enabled", false).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	var credCount int64
	if err := db.DB.Model(&models.WebAuthnCredential{}).Where("user_id = ?", profile.ID).Count(&credCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	var updated models.Profile
	if err := db.DB.Select("is_totp_enabled", "is_webauthn_enabled").First(&updated, profile.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":                   true,
		"is_totp_enabled":           updated.IsTotpEnabled,
		"is_webauthn_enabled":       updated.IsWebauthnEnabled,
		"webauthn_credential_count": credCount,
	})
}

// dedupeIDs drops duplicate ids, keeping first-seen order. Django's
// auto-created through tables have a unique (user_id, x_id) constraint, so a
// duplicate input id would otherwise fail the insert on the second row.
func dedupeIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
