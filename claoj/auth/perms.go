package auth

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

const (
	permCacheTTL   = 60 * time.Second
	permVersionKey = "perm:version"
	accessCtxKey   = "django_access"
)

// PermSet is a set of Django permission strings ("app_label.codename").
type PermSet map[string]struct{}

func (s PermSet) Has(codename string) bool { _, ok := s[codename]; return ok }

// UserAccess is a user's resolved Django auth state.
// HasPerm applies Django's ModelBackend semantics: inactive users have no
// permissions, superusers have all of them.
type UserAccess struct {
	UserID      uint     `json:"user_id"`
	IsActive    bool     `json:"is_active"`
	IsStaff     bool     `json:"is_staff"`
	IsSuperuser bool     `json:"is_superuser"`
	Perms       PermSet  `json:"-"`
	PermList    []string `json:"perms"` // JSON-serializable form for the Redis cache
}

func (a *UserAccess) HasPerm(codename string) bool {
	if a == nil || !a.IsActive {
		return false
	}
	if a.IsSuperuser {
		return true
	}
	return a.Perms.Has(codename)
}

// AnonymousAccess is the all-deny access for unauthenticated requests.
func AnonymousAccess() *UserAccess { return &UserAccess{} }

func permQuery(joinTable, userCol string, userID uint) ([]string, error) {
	var rows []struct {
		AppLabel string
		Codename string
	}
	err := db.DB.
		Table("auth_permission").
		Select("django_content_type.app_label AS app_label, auth_permission.codename AS codename").
		Joins("JOIN django_content_type ON django_content_type.id = auth_permission.content_type_id").
		Joins(joinTable).
		Where(userCol+" = ?", userID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.AppLabel+"."+r.Codename)
	}
	return out, nil
}

// resolveFromDB computes effective permissions the way Django's ModelBackend
// does: direct user permissions ∪ permissions of the user's groups.
func resolveFromDB(userID uint) (*UserAccess, error) {
	var user models.AuthUser
	if err := db.DB.Select("id", "is_active", "is_staff", "is_superuser").First(&user, userID).Error; err != nil {
		return nil, err
	}
	direct, err := permQuery(
		"JOIN auth_user_user_permissions ON auth_user_user_permissions.permission_id = auth_permission.id",
		"auth_user_user_permissions.user_id", userID)
	if err != nil {
		return nil, err
	}
	viaGroups, err := permQuery(
		"JOIN auth_group_permissions ON auth_group_permissions.permission_id = auth_permission.id "+
			"JOIN auth_user_groups ON auth_user_groups.group_id = auth_group_permissions.group_id",
		"auth_user_groups.user_id", userID)
	if err != nil {
		return nil, err
	}
	access := &UserAccess{
		UserID:      user.ID,
		IsActive:    user.IsActive,
		IsStaff:     user.IsStaff,
		IsSuperuser: user.IsSuperuser,
		Perms:       make(PermSet, len(direct)+len(viaGroups)),
	}
	for _, p := range append(direct, viaGroups...) {
		access.Perms[p] = struct{}{}
	}
	access.PermList = make([]string, 0, len(access.Perms))
	for p := range access.Perms {
		access.PermList = append(access.PermList, p)
	}
	return access, nil
}

func permCacheKey(userID uint) string {
	version := "1"
	if cache.Client != nil {
		if v, err := cache.Client.Get(cache.Ctx, permVersionKey).Result(); err == nil {
			version = v
		}
	}
	return fmt.Sprintf("perm:v%s:%d", version, userID)
}

// LoadUserAccess returns the user's resolved access, via Redis when available.
func LoadUserAccess(userID uint) (*UserAccess, error) {
	if cache.Client == nil {
		return resolveFromDB(userID)
	}
	key := permCacheKey(userID)
	if raw, err := cache.Client.Get(cache.Ctx, key).Result(); err == nil {
		var access UserAccess
		if json.Unmarshal([]byte(raw), &access) == nil {
			access.Perms = make(PermSet, len(access.PermList))
			for _, p := range access.PermList {
				access.Perms[p] = struct{}{}
			}
			return &access, nil
		}
	}
	access, err := resolveFromDB(userID)
	if err != nil {
		return nil, err
	}
	if raw, err := json.Marshal(access); err == nil {
		cache.Client.Set(cache.Ctx, key, raw, permCacheTTL)
	}
	return access, nil
}

// BumpPermVersion invalidates all cached permission sets. Call after any write
// to groups / group-permissions / user-groups / user staff flags.
func BumpPermVersion() {
	if cache.Client != nil {
		cache.Client.Incr(cache.Ctx, permVersionKey)
	}
}

// GetAccess returns the request user's access, memoized on the gin context.
// Unauthenticated requests get AnonymousAccess (all deny).
func GetAccess(c *gin.Context) *UserAccess {
	if v, ok := c.Get(accessCtxKey); ok {
		return v.(*UserAccess)
	}
	userID, ok := c.Get("user_id")
	if !ok {
		a := AnonymousAccess()
		c.Set(accessCtxKey, a)
		return a
	}
	access, err := LoadUserAccess(userID.(uint))
	if err != nil {
		access = AnonymousAccess()
	}
	c.Set(accessCtxKey, access)
	return access
}

// HasPerm reports whether the request user holds the Django permission,
// e.g. HasPerm(c, "judge.see_private_problem").
func HasPerm(c *gin.Context, codename string) bool {
	return GetAccess(c).HasPerm(codename)
}
