# Django Schema/Permission Parity + Full Vietnamese Support — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Go backend (`claoj/`) run against the untouched Django-managed MySQL database using Django's own roles/permissions, and make the Next.js frontend (`claoj-web/`) render and speak Vietnamese perfectly.

**Architecture:** The Go backend gets a Django-native permission adapter (reads `auth_group`/`auth_permission`/... tables, checks Django codenames, Redis-cached). The dead custom RBAC (`judge_role*`), all 7 v2 migrations, and schema-dependent v2 features are deleted. Refresh tokens, password-reset/email-verify tokens, and the audit log move to Redis. The frontend swaps Outfit → Be Vietnam Pro and gets full en/vi message coverage.

**Tech Stack:** Go 1.24, Gin, GORM (MySQL prod / SQLite in tests), go-redis v9, Next.js 16 App Router, next-intl 4, Tailwind CSS v4 (CSS `@theme`).

**Spec:** `docs/superpowers/specs/2026-07-18-django-parity-and-vietnamese-design.md`

## Global Constraints

- **No DDL, ever.** OJ-v2 only reads/writes rows. Schema is owned by Django migrations (`OJ/judge/migrations/`). No `AutoMigrate` outside `_test.go` files.
- **Django view behavior is the source of truth** for every permission gate. Deny by default for anything unmapped.
- **Retained-table exception (documented):** `notification`, `notification_preference`, `totp_device`, `backup_code` are pre-existing additive v2 tables holding live user data (2FA secrets, notifications). They stay. Everything else v2-only goes away.
- **Django reference gates** (verified from `OJ/` source — file:line refs):
  - `Problem.is_editable_by` — `OJ/judge/models/problem.py:229-238`: requires `judge.edit_own_problem`; then `judge.edit_all_problem` OR (`judge.edit_public_problem` AND public) OR profile ∈ editors (authors ∪ curators ∪ suggester, `problem.py:374-381`).
  - Hidden problem visibility: `judge.see_private_problem` (`problem.py:269`).
  - `Contest.is_editable_by` — `OJ/judge/models/contest.py:380-389`: `judge.edit_all_contest` OR (`judge.edit_own_contest` AND profile ∈ authors ∪ curators).
  - Private contest visibility: `judge.see_private_contest` OR `judge.edit_all_contest` (`contest.py:332-333`), or editor/tester.
  - Solution visibility — `problem.py:630-637`: public+published OR `judge.see_private_solution` OR problem editable.
  - Rejudge: `judge.rejudge_submission` AND problem editable (`problem.py:286-287`); bulk adds `judge.rejudge_submission_lot` + `is_staff`.
  - View any submission detail: `judge.view_all_submission` (`submission.py:146`).
  - Abort any submission: `judge.abort_any_submission` (`views/submission.py:261-267`).
  - Comments: hide/delete/edit-others `judge.change_comment` (`views/comment.py:186-188, 153-160`); lock override `judge.override_comment_lock`.
  - Tickets: `judge.change_ticket` OR owner OR assignee OR linked problem editable (`views/ticket.py:150-165`).
  - Blog: `judge.edit_all_post` OR author (`interface.py:116-125`).
  - MOSS: `judge.moss_contest` AND contest editable (`views/contests.py:895-903`).
  - Ban user: **is_superuser only** (`views/user.py:236-238`); ban = `ban_reason` + `display_rank='banned'` + `is_active=False` (`profile.py:316-324`).
  - Site stats data: is_superuser only (`views/stats.py:145-147`).
  - Problem/contest create: `judge.add_problem` / `judge.add_contest`.
  - Admin panel: `is_staff` (Django admin parity). Judges/languages management: admin-only → keep `is_staff` gate.
- **Frontend:** every user-visible string goes through next-intl; `en.json` and `vi.json` must have identical key sets; single UI font = Be Vietnam Pro (`latin` + `vietnamese` subsets); mono stays the existing system stack (`--font-mono` in globals.css — Consolas/Menlo cover Vietnamese; JetBrains Mono does NOT have a Vietnamese subset, spec is amended accordingly).
- Backend repo root for all Go paths: `f:/Coding/CLAOJ/OJ-v2/claoj`. Frontend root: `f:/Coding/CLAOJ/OJ-v2/claoj-web`. Commit after every task (both live in the `OJ-v2` git repo).
- Run Go tests with `go test ./...` from `claoj/`; frontend with `npm test` and `npm run build` from `claoj-web/`.

---

### Task 1: Django auth GORM models

**Files:**
- Create: `claoj/models/django_auth.go`
- Modify: `claoj/models/profile.go` (add `Groups` association to `AuthUser`)
- Test: `claoj/models/django_auth_test.go`

**Interfaces:**
- Consumes: existing `models.AuthUser` (table `auth_user`, fields `IsStaff`, `IsSuperuser`, `IsActive`).
- Produces: `models.AuthGroup`, `models.AuthPermission`, `models.DjangoContentType`, `models.AuthUserGroup`, `models.AuthGroupPermission`, `models.AuthUserPermission`, and `AuthUser.Groups []AuthGroup`. Later tasks (2, 6) rely on these exact names.

- [ ] **Step 1: Write the failing test**

```go
// claoj/models/django_auth_test.go
package models

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&AuthUser{}, &AuthGroup{}, &DjangoContentType{}, &AuthPermission{},
		&AuthUserGroup{}, &AuthGroupPermission{}, &AuthUserPermission{},
	))
	return database
}

func TestDjangoAuthTableNames(t *testing.T) {
	require.Equal(t, "auth_group", AuthGroup{}.TableName())
	require.Equal(t, "auth_permission", AuthPermission{}.TableName())
	require.Equal(t, "django_content_type", DjangoContentType{}.TableName())
	require.Equal(t, "auth_user_groups", AuthUserGroup{}.TableName())
	require.Equal(t, "auth_group_permissions", AuthGroupPermission{}.TableName())
	require.Equal(t, "auth_user_user_permissions", AuthUserPermission{}.TableName())
}

func TestGroupPermissionAssociations(t *testing.T) {
	database := setupAuthTestDB(t)
	ct := DjangoContentType{AppLabel: "judge", Model: "problem"}
	require.NoError(t, database.Create(&ct).Error)
	perm := AuthPermission{Name: "See hidden problems", Codename: "see_private_problem", ContentTypeID: ct.ID}
	require.NoError(t, database.Create(&perm).Error)
	group := AuthGroup{Name: "Editors"}
	require.NoError(t, database.Create(&group).Error)
	require.NoError(t, database.Model(&group).Association("Permissions").Append(&perm))

	var loaded AuthGroup
	require.NoError(t, database.Preload("Permissions").First(&loaded, group.ID).Error)
	require.Len(t, loaded.Permissions, 1)
	require.Equal(t, "see_private_problem", loaded.Permissions[0].Codename)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd f:/Coding/CLAOJ/OJ-v2/claoj && go test ./models/ -run TestDjangoAuth -v`
Expected: FAIL (compile error: `AuthGroup` undefined)

- [ ] **Step 3: Write the models**

```go
// claoj/models/django_auth.go
package models

// Django's built-in auth tables. OJ-v2 reads and writes ROWS in these tables
// (exactly like Django admin does) but never creates or alters them — the
// schema is owned by Django migrations in the OJ repository.

// AuthGroup is Django's auth_group — the "role" unit shared with the Django site.
type AuthGroup struct {
	ID          uint             `gorm:"primaryKey;column:id"`
	Name        string           `gorm:"column:name;size:150;uniqueIndex"`
	Permissions []AuthPermission `gorm:"many2many:auth_group_permissions;joinForeignKey:group_id;joinReferences:permission_id"`
}

func (AuthGroup) TableName() string { return "auth_group" }

// DjangoContentType is django_content_type; app_label+model qualify permission codenames.
type DjangoContentType struct {
	ID       uint   `gorm:"primaryKey;column:id"`
	AppLabel string `gorm:"column:app_label;size:100"`
	Model    string `gorm:"column:model;size:100"`
}

func (DjangoContentType) TableName() string { return "django_content_type" }

// AuthPermission is auth_permission. Full permission string = "{app_label}.{codename}".
type AuthPermission struct {
	ID            uint              `gorm:"primaryKey;column:id"`
	Name          string            `gorm:"column:name;size:255"`
	ContentTypeID uint              `gorm:"column:content_type_id"`
	Codename      string            `gorm:"column:codename;size:100"`
	ContentType   DjangoContentType `gorm:"foreignKey:ContentTypeID"`
}

func (AuthPermission) TableName() string { return "auth_permission" }

// AuthUserGroup is the auth_user_groups join row (user ↔ group).
type AuthUserGroup struct {
	ID      uint `gorm:"primaryKey;column:id"`
	UserID  uint `gorm:"column:user_id"`
	GroupID uint `gorm:"column:group_id"`
}

func (AuthUserGroup) TableName() string { return "auth_user_groups" }

// AuthGroupPermission is the auth_group_permissions join row (group ↔ permission).
type AuthGroupPermission struct {
	ID           uint `gorm:"primaryKey;column:id"`
	GroupID      uint `gorm:"column:group_id"`
	PermissionID uint `gorm:"column:permission_id"`
}

func (AuthGroupPermission) TableName() string { return "auth_group_permissions" }

// AuthUserPermission is the auth_user_user_permissions join row (user ↔ direct permission).
type AuthUserPermission struct {
	ID           uint `gorm:"primaryKey;column:id"`
	UserID       uint `gorm:"column:user_id"`
	PermissionID uint `gorm:"column:permission_id"`
}

func (AuthUserPermission) TableName() string { return "auth_user_user_permissions" }
```

In `claoj/models/profile.go`, add to the `AuthUser` struct (do not change any existing field):

```go
	Groups []AuthGroup `gorm:"many2many:auth_user_groups;joinForeignKey:user_id;joinReferences:group_id"`
```

- [ ] **Step 4: Run tests**

Run: `go test ./models/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add claoj/models/django_auth.go claoj/models/django_auth_test.go claoj/models/profile.go
git commit -m "feat(auth): add GORM models for Django's auth tables"
```

---

### Task 2: Django permission resolver with Redis cache

**Files:**
- Create: `claoj/auth/perms.go`
- Test: `claoj/auth/perms_test.go`

**Interfaces:**
- Consumes: Task 1 models; `db.DB`; `cache.Client` (`*redis.Client`, may be nil); gin context keys `"user_id"` (uint).
- Produces (used by Tasks 3, 4, 6):
  - `type PermSet map[string]struct{}` with method `Has(codename string) bool`
  - `type UserAccess struct { UserID uint; IsActive, IsStaff, IsSuperuser bool; Perms PermSet }`
  - `func LoadUserAccess(userID uint) (*UserAccess, error)` — DB+Redis-cached
  - `func GetAccess(c *gin.Context) *UserAccess` — per-request memo; returns anonymous (all-deny) access when unauthenticated
  - `func HasPerm(c *gin.Context, codename string) bool` — full Django semantics (inactive→false, superuser→true)
  - `func BumpPermVersion()` — invalidates the Redis permission cache

- [ ] **Step 1: Write the failing test**

```go
// claoj/auth/perms_test.go
package auth

import (
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPermsDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{},
		&models.AuthGroupPermission{}, &models.AuthUserPermission{},
	))
	db.DB = database
}

// seedPerm creates judge.<codename> and returns its ID.
func seedPerm(t *testing.T, codename string) uint {
	t.Helper()
	var ct models.DjangoContentType
	require.NoError(t, db.DB.FirstOrCreate(&ct, models.DjangoContentType{AppLabel: "judge", Model: "problem"}).Error)
	perm := models.AuthPermission{Name: codename, Codename: codename, ContentTypeID: ct.ID}
	require.NoError(t, db.DB.Create(&perm).Error)
	return perm.ID
}

func TestResolve_GroupAndDirectPerms(t *testing.T) {
	setupPermsDB(t)
	user := models.AuthUser{Username: "alice", IsActive: true}
	require.NoError(t, db.DB.Create(&user).Error)

	directID := seedPerm(t, "see_private_problem")
	groupPermID := seedPerm(t, "rejudge_submission")
	group := models.AuthGroup{Name: "Judges"}
	require.NoError(t, db.DB.Create(&group).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserPermission{UserID: user.ID, PermissionID: directID}).Error)
	require.NoError(t, db.DB.Create(&models.AuthGroupPermission{GroupID: group.ID, PermissionID: groupPermID}).Error)
	require.NoError(t, db.DB.Create(&models.AuthUserGroup{UserID: user.ID, GroupID: group.ID}).Error)

	access, err := LoadUserAccess(user.ID)
	require.NoError(t, err)
	require.True(t, access.Perms.Has("judge.see_private_problem"))
	require.True(t, access.Perms.Has("judge.rejudge_submission"))
	require.False(t, access.Perms.Has("judge.edit_all_problem"))
}

func TestResolve_InactiveDeniesAll_SuperuserAllowsAll(t *testing.T) {
	setupPermsDB(t)
	inactive := models.AuthUser{Username: "banned", IsActive: false}
	super := models.AuthUser{Username: "root", IsActive: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&inactive).Error)
	require.NoError(t, db.DB.Create(&super).Error)

	a1, err := LoadUserAccess(inactive.ID)
	require.NoError(t, err)
	require.False(t, a1.HasPerm("judge.see_private_problem"))

	a2, err := LoadUserAccess(super.ID)
	require.NoError(t, err)
	require.True(t, a2.HasPerm("judge.anything_at_all"))
}

func TestAnonymousAccessDeniesAll(t *testing.T) {
	a := AnonymousAccess()
	require.False(t, a.HasPerm("judge.see_private_problem"))
	require.False(t, a.IsStaff)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./auth/ -run "TestResolve|TestAnonymous" -v`
Expected: FAIL (compile error: `LoadUserAccess` undefined)

- [ ] **Step 3: Implement the resolver**

```go
// claoj/auth/perms.go
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
	permCacheTTL    = 60 * time.Second
	permVersionKey  = "perm:version"
	accessCtxKey    = "django_access"
)

// PermSet is a set of Django permission strings ("app_label.codename").
type PermSet map[string]struct{}

func (s PermSet) Has(codename string) bool { _, ok := s[codename]; return ok }

// UserAccess is a user's resolved Django auth state.
// HasPerm applies Django's ModelBackend semantics: inactive users have no
// permissions, superusers have all of them.
type UserAccess struct {
	UserID      uint    `json:"user_id"`
	IsActive    bool    `json:"is_active"`
	IsStaff     bool    `json:"is_staff"`
	IsSuperuser bool    `json:"is_superuser"`
	Perms       PermSet `json:"-"`
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
```

- [ ] **Step 4: Run tests**

Run: `go test ./auth/ -v`
Expected: PASS (existing auth tests must also still pass)

- [ ] **Step 5: Commit**

```bash
git add claoj/auth/perms.go claoj/auth/perms_test.go
git commit -m "feat(auth): Django-native permission resolver with Redis cache"
```

---

### Task 3: Django-semantics access helpers

**Files:**
- Create: `claoj/auth/access.go`
- Test: `claoj/auth/access_test.go`

**Interfaces:**
- Consumes: Task 2 (`GetAccess`, `HasPerm`); existing `models.Problem` (assocs `Authors`, `Curators`, `Testers` — []Profile; field `SuggesterID *uint`), `models.Contest` (assocs `Authors`, `Curators`, `Testers`), `models.Profile` (field `UserID`), `models.Solution`.
- Produces (used by Tasks 4, 5, 6, and any future handler):
  - `func CurrentProfileID(c *gin.Context) (uint, bool)` — profile ID for the request user
  - `func CanEditProblem(c *gin.Context, problem *models.Problem) bool`
  - `func CanViewProblem(c *gin.Context, problem *models.Problem) bool`
  - `func CanEditContest(c *gin.Context, contest *models.Contest) bool`
  - `func CanViewContest(c *gin.Context, contest *models.Contest) bool`
  - `func CanViewSolution(c *gin.Context, solution *models.Solution, problem *models.Problem) bool`
  - `func CanRejudge(c *gin.Context, problem *models.Problem) bool`

- [ ] **Step 1: Write the failing tests**

Write table-driven tests in `claoj/auth/access_test.go`. Reuse `setupPermsDB`/`seedPerm` from Task 2's test file (same package). Seed an `AuthUser` + `Profile` (AutoMigrate `models.Profile` too in `setupPermsDB` — add it to the AutoMigrate list) and a `Problem` with the profile as author. Build a gin context with `c.Set("user_id", user.ID)` via `gin.CreateTestContext(httptest.NewRecorder())`. Cases (each row: grant perms → expect result):

```go
func TestCanEditProblem_DjangoSemantics(t *testing.T) {
	// mirrors OJ/judge/models/problem.py:229-238
	cases := []struct {
		name     string
		perms    []string // judge.* codenames granted via a group
		isAuthor bool
		isPublic bool
		want     bool
	}{
		{"no perms at all", nil, true, false, false},                                        // author but lacks edit_own_problem
		{"edit_own only, not author", []string{"edit_own_problem"}, false, false, false},
		{"edit_own + author", []string{"edit_own_problem"}, true, false, true},
		{"edit_all, not author", []string{"edit_own_problem", "edit_all_problem"}, false, false, true},
		{"edit_public on public", []string{"edit_own_problem", "edit_public_problem"}, false, true, true},
		{"edit_public on private", []string{"edit_own_problem", "edit_public_problem"}, false, false, false},
	}
	// ... build user/profile/problem per case, grant perms, assert CanEditProblem
}
```

Also write: `TestCanViewProblem` (public→anyone incl. anonymous; hidden→false for plain user, true with `see_private_problem`, true for author with `edit_own_problem`, true for tester), `TestCanEditContest` (edit_all_contest→true; edit_own_contest+author→true; edit_own_contest alone→false), `TestCanViewContest` (visible→anyone; hidden→`see_private_contest` or `edit_all_contest` or editor/tester), `TestCanViewSolution` (public+published→true; else `see_private_solution`→true; else problem-editable→true; else false), `TestCanRejudge` (`rejudge_submission`+editable→true; either alone→false; superuser→true). Write the full table bodies — every row above becomes a real subtest.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./auth/ -run "TestCan" -v`
Expected: FAIL (compile error: `CanEditProblem` undefined)

- [ ] **Step 3: Implement the helpers**

```go
// claoj/auth/access.go
package auth

import (
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// CurrentProfileID resolves the request user's judge_profile id.
func CurrentProfileID(c *gin.Context) (uint, bool) {
	userID, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}
	if v, ok := c.Get("current_profile_id"); ok {
		return v.(uint), true
	}
	var profile models.Profile
	if err := db.DB.Select("id").Where("user_id = ?", userID.(uint)).First(&profile).Error; err != nil {
		return 0, false
	}
	c.Set("current_profile_id", profile.ID)
	return profile.ID, true
}

func profileInList(profiles []models.Profile, profileID uint) bool {
	for _, p := range profiles {
		if p.ID == profileID {
			return true
		}
	}
	return false
}

// problemEditorIDs mirrors Problem.editor_ids (OJ problem.py:374-381):
// authors ∪ curators ∪ suggester.
func isProblemEditor(problem *models.Problem, profileID uint) bool {
	if profileInList(problem.Authors, profileID) || profileInList(problem.Curators, profileID) {
		return true
	}
	return problem.SuggesterID != nil && *problem.SuggesterID == profileID
}

// CanEditProblem mirrors Problem.is_editable_by (OJ problem.py:229-238).
// The problem must be loaded with Preload("Authors").Preload("Curators").
func CanEditProblem(c *gin.Context, problem *models.Problem) bool {
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	if !access.HasPerm("judge.edit_own_problem") {
		return false
	}
	if access.HasPerm("judge.edit_all_problem") {
		return true
	}
	if access.HasPerm("judge.edit_public_problem") && problem.IsPublic {
		return true
	}
	profileID, ok := CurrentProfileID(c)
	return ok && isProblemEditor(problem, profileID)
}

// CanViewProblem mirrors the codename-relevant parts of Problem.is_accessible_by
// (OJ problem.py:240-284): public problems are visible; hidden ones require
// see_private_problem, editability, or being a tester.
// Load with Preload("Authors").Preload("Curators").Preload("Testers").
func CanViewProblem(c *gin.Context, problem *models.Problem) bool {
	if problem.IsPublic {
		return true
	}
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	if access.HasPerm("judge.see_private_problem") {
		return true
	}
	if CanEditProblem(c, problem) {
		return true
	}
	profileID, ok := CurrentProfileID(c)
	return ok && profileInList(problem.Testers, profileID)
}

// contestEditorIDs mirrors Contest.editor_ids: authors ∪ curators.
func isContestEditor(contest *models.Contest, profileID uint) bool {
	return profileInList(contest.Authors, profileID) || profileInList(contest.Curators, profileID)
}

// CanEditContest mirrors Contest.is_editable_by (OJ contest.py:380-389).
// Load with Preload("Authors").Preload("Curators").
func CanEditContest(c *gin.Context, contest *models.Contest) bool {
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	if access.HasPerm("judge.edit_all_contest") {
		return true
	}
	if !access.HasPerm("judge.edit_own_contest") {
		return false
	}
	profileID, ok := CurrentProfileID(c)
	return ok && isContestEditor(contest, profileID)
}

// CanViewContest mirrors the codename-relevant parts of Contest.access_check
// (OJ contest.py:321-370). Load with Preload("Authors").Preload("Curators").Preload("Testers").
func CanViewContest(c *gin.Context, contest *models.Contest) bool {
	if contest.IsVisible {
		return true
	}
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	if access.HasPerm("judge.see_private_contest") || access.HasPerm("judge.edit_all_contest") {
		return true
	}
	profileID, ok := CurrentProfileID(c)
	if !ok {
		return false
	}
	return isContestEditor(contest, profileID) || profileInList(contest.Testers, profileID)
}

// CanViewSolution mirrors Solution.is_accessible_by (OJ problem.py:630-637).
func CanViewSolution(c *gin.Context, solution *models.Solution, problem *models.Problem) bool {
	if solution.IsPublic && solution.PublishOn != nil && solution.PublishOn.Before(time.Now()) {
		return true
	}
	if HasPerm(c, "judge.see_private_solution") {
		return true
	}
	return CanEditProblem(c, problem)
}

// CanRejudge mirrors Problem.is_rejudgeable_by (OJ problem.py:286-287).
func CanRejudge(c *gin.Context, problem *models.Problem) bool {
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	return access.HasPerm("judge.rejudge_submission") && CanEditProblem(c, problem)
}
```

Note: check the exact field names on `models.Solution` (`IsPublic`, `PublishOn`) in `claoj/models/problem.go` before compiling — if `PublishOn` is `time.Time` (not pointer), drop the nil check.

- [ ] **Step 4: Run tests**

Run: `go test ./auth/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add claoj/auth/access.go claoj/auth/access_test.go
git commit -m "feat(auth): access helpers mirroring Django is_editable_by/is_accessible_by"
```

---

### Task 4: Wire Django gates into handlers

**Files:**
- Modify: `claoj/api/v2/solution.go:340-370` (the one real role consumer)
- Modify: every handler with an inline `is_admin` / `IsStaff` fine-grained decision (list below)
- Test: extend `claoj/integration/` flows only if a gate change alters an existing test's expectation

**Interfaces:**
- Consumes: Task 3 helpers, Task 2 `HasPerm`.
- Produces: no new API; behavioral parity with Django gates.

- [ ] **Step 1: Replace the solution.go role check**

`claoj/api/v2/solution.go:354-360` currently does `Preload("Roles")` on Profile and iterates `profile.Roles` to decide solution visibility. Replace that whole block with:

```go
	if !auth.CanViewSolution(c, &solution, &problem) {
		c.JSON(http.StatusNotFound, gin.H{"error": "solution not found"})
		return
	}
```

(Load `problem` with `Preload("Authors").Preload("Curators")` if the handler doesn't already. Delete the now-unused `Preload("Roles")` query.)

- [ ] **Step 2: Sweep remaining inline privilege checks**

Run: `grep -rn "is_admin\|IsStaff\|IsSuperuser" claoj/api/v2/ claoj/jobs/` and classify every hit with this decision table:

| Current check means | Replace with |
|---|---|
| "may enter admin area" (route-level, binary) | keep `is_staff` (Django-admin parity) |
| "may see hidden problem" | `auth.CanViewProblem(c, &problem)` |
| "may edit problem / its data" | `auth.CanEditProblem(c, &problem)` |
| "may see/edit hidden contest" | `auth.CanViewContest` / `auth.CanEditContest` |
| "may rejudge" (incl. admin submissions rejudge/rescore) | `auth.CanRejudge(c, &problem)`; bulk additionally `auth.HasPerm(c, "judge.rejudge_submission_lot")` |
| "may view any submission detail" | `auth.HasPerm(c, "judge.view_all_submission")` (fallback: own submission / problem editable, per OJ submission.py:139-167) |
| "may abort any submission" | `auth.HasPerm(c, "judge.abort_any_submission")` (own+unjudged still allowed) |
| "may moderate comments" (hide/delete/edit others, incl. admin_comment.go) | `auth.HasPerm(c, "judge.change_comment")` |
| "may edit any blog post" (admin_blog_post.go) | `auth.HasPerm(c, "judge.edit_all_post")` |
| "may run MOSS" (moss.go) | `auth.HasPerm(c, "judge.moss_contest")` AND `auth.CanEditContest` |
| "may ban user" | `GetAccess(c).IsSuperuser` (OJ user.py:236-238 — superuser only) |
| "may manage tickets" | `auth.HasPerm(c, "judge.change_ticket")` OR owner OR assignee OR problem editable |

Known files from research: `api/v2/moss.go`, `api/v2/me.go`, `api/v2/submission.go`, `api/v2/problem.go` (lines ~42/182/242), `api/v2/comment.go`, `api/v2/admin_comment.go`, `api/v2/admin_blog_post.go`, `api/v2/contest/contest_clarification.go`, `api/v2/auth/oauth_helpers.go`, `jobs/export.go`. Apply the table to each hit; where a check is route-level admin gating, leave it.

- [ ] **Step 3: Build and run all tests**

Run: `go build ./... && go test ./...`
Expected: PASS. If an integration test asserted old behavior (e.g. staff could see everything), update the test to seed the matching Django permission instead.

- [ ] **Step 4: Commit**

```bash
git add -A claoj/
git commit -m "feat(auth): enforce Django permission gates in handlers"
```

---

### Task 5: Delete the custom RBAC and orphaned migrations

**Files:**
- Delete: `claoj/db/migrations/` (entire directory, 7 files)
- Delete: `claoj/auth/permissions.go`, `claoj/auth/permissions_test.go`, `claoj/auth/authorization.go`
- Delete: `claoj/service/role/` (entire package)
- Delete: `claoj/models/role.go`
- Modify: `claoj/auth/middleware.go` (remove `PermissionRequiredMiddleware` :159-207 and `RoleRequiredMiddleware` :209-257)
- Modify: `claoj/middleware/user_auth.go` (remove `RequirePermission` :170-228)
- Modify: `claoj/models/profile.go:68` (remove `Roles []Role` field)
- Modify: `claoj/api/v2/admin_users_roles.go` (remove role handlers :281-533 — keep the user handlers :19-279; drop the `resp.Roles` block at :297-298)
- Modify: `claoj/api/router.go:182-189` (remove role/permission route registrations)
- Modify: `claoj/integration/helpers.go` (drop `&models.Role{}` from AutoMigrate)
- Modify: `claoj/db/db.go` (strengthen comment)

**Interfaces:**
- Consumes: Tasks 2-4 must be merged first (they replace the only real consumer).
- Produces: `/api/admin/roles*` and `/api/admin/permissions` endpoints are GONE until Task 6 recreates groups endpoints. Frontend roles page breaks until Task 7 — acceptable mid-plan; Tasks 6-7 land in the same release.

- [ ] **Step 1: Delete files and edit call sites** (as listed above). In `db/db.go` replace the comment at lines 14-16 with:

```go
// Connect opens the GORM connection to MySQL using the configured DSN.
// The schema is 100% owned by Django migrations (OJ repo). OJ-v2 must NEVER
// execute DDL — no AutoMigrate, no CREATE/ALTER/DROP. Rows only.
```

- [ ] **Step 2: Build, fix stragglers, run tests**

Run: `go build ./... 2>&1 | head -40` then `go test ./...`
Expected: compile errors point at any missed reference (fix by deleting the dead usage); then PASS.

- [ ] **Step 3: Verify no DDL sources remain**

Run: `grep -rn "AutoMigrate\|CREATE TABLE\|ALTER TABLE" claoj/ --include="*.go" | grep -v "_test.go"`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add -A claoj/
git commit -m "refactor(auth)!: delete custom RBAC, orphaned migrations, and dead permission middleware"
```

---

### Task 6: Django groups admin API

**Files:**
- Create: `claoj/service/group/group_service.go`, `claoj/service/group/types.go`, `claoj/service/group/errors.go`
- Create: `claoj/api/v2/admin_groups.go`
- Modify: `claoj/api/router.go` (register new routes where role routes were)
- Modify: `claoj/api/v2/admin_users_roles.go` `AdminUserUpdate` (add `is_staff`/`is_superuser` fields, superuser-only)
- Test: `claoj/service/group/group_service_test.go`

**Interfaces:**
- Consumes: Task 1 models, Task 2 `BumpPermVersion`.
- Produces REST API (Task 7 frontend consumes exactly these):
  - `GET    /api/admin/groups` → `{data: [{id, name, user_count, permission_count}]}`
  - `GET    /api/admin/group/:id` → `{id, name, permission_ids: [uint], users: [{id, username}]}`
  - `POST   /api/admin/groups` body `{name string, permission_ids []uint}` → 201 `{id, name}`
  - `PATCH  /api/admin/group/:id` body `{name?, permission_ids?}` → 200
  - `DELETE /api/admin/group/:id` → 204
  - `GET    /api/admin/permissions` → `{data: [{id, codename ("judge.edit_all_problem"), name, app_label, model}]}`
  - `POST   /api/admin/user/:id/groups` body `{group_id uint}` → 204
  - `DELETE /api/admin/user/:id/groups/:groupId` → 204
- Service signatures: `ListGroups() ([]GroupSummary, error)`, `GetGroup(id uint) (*GroupDetail, error)`, `CreateGroup(name string, permissionIDs []uint) (*GroupSummary, error)`, `UpdateGroup(id uint, name *string, permissionIDs *[]uint) error`, `DeleteGroup(id uint) error`, `ListPermissions() ([]PermissionInfo, error)`, `AddUserToGroup(userID, groupID uint) error`, `RemoveUserFromGroup(userID, groupID uint) error`.

- [ ] **Step 1: Write failing service tests** — sqlite in-memory (same pattern as Task 2 test), covering: create group with permissions → list shows counts; update replaces permission set; delete removes join rows; add/remove user membership; duplicate name → `ErrGroupNameExists`; permission listing returns `app_label.codename` form.

- [ ] **Step 2: Run to verify fail** — `go test ./service/group/ -v` → compile error.

- [ ] **Step 3: Implement service + handlers + routes.** Service methods write rows via the Task 1 models only (`Association("Permissions").Replace`, `Create`/`Delete` on join models). EVERY mutating method's success path ends with `auth.BumpPermVersion()`. Handlers follow the existing style of `admin_users_roles.go` user handlers (bind → service → JSON). Routes replace the removed block at `router.go:182-189`:

```go
		// Django group (role) management — writes rows in Django's own auth tables
		admin.GET("/groups", apiv2.AdminGroupList)
		admin.GET("/group/:id", apiv2.AdminGroupDetail)
		admin.POST("/groups", apiv2.AdminGroupCreate)
		admin.PATCH("/group/:id", apiv2.AdminGroupUpdate)
		admin.DELETE("/group/:id", apiv2.AdminGroupDelete)
		admin.GET("/permissions", apiv2.AdminPermissionList)
		admin.POST("/user/:id/groups", apiv2.AdminUserAddGroup)
		admin.DELETE("/user/:id/groups/:groupId", apiv2.AdminUserRemoveGroup)
```

In `AdminUserUpdate`, add optional body fields `is_staff *bool`, `is_superuser *bool`; applying either requires `auth.GetAccess(c).IsSuperuser` (403 otherwise) and ends with `auth.BumpPermVersion()`.

- [ ] **Step 4: Run tests** — `go test ./...` → PASS.

- [ ] **Step 5: Commit**

```bash
git add claoj/service/group claoj/api/v2/admin_groups.go claoj/api/router.go claoj/api/v2/admin_users_roles.go
git commit -m "feat(admin): Django group management API (replaces custom roles)"
```

---

### Task 7: Frontend — groups admin UI replaces roles UI

**Files:**
- Create: `claoj-web/src/app/[locale]/admin/groups/page.tsx`
- Delete: `claoj-web/src/app/[locale]/admin/roles/page.tsx`
- Modify: `claoj-web/src/lib/adminApi.ts` (:416-457 — replace `adminRolesApi`/role types with `adminGroupsApi`/group types)
- Modify: `claoj-web/src/types/index.ts` (:594-614 — replace `Permission`/`Role`/`RoleWithUserCount`; :106 — remove `roles` from `UserDetail`)
- Modify: `claoj-web/src/app/[locale]/admin/layout.tsx` (:103-107 — sidebar link `/admin/roles` → `/admin/groups`)
- Modify: any component rendering `UserDetail.roles` badges (run `grep -rn "\.roles" claoj-web/src --include="*.tsx"` and remove those render blocks)
- Modify: `claoj-web/src/i18n/en.json`, `claoj-web/src/i18n/vi.json` (new `AdminGroups` namespace)

**Interfaces:**
- Consumes: Task 6 REST API exactly as specified.
- Produces: types `Group {id, name, user_count, permission_count}`, `GroupDetail {id, name, permission_ids, users}`, `PermissionInfo {id, codename, name, app_label, model}`; `adminGroupsApi.{list, detail, create, update, delete, permissions, addUser, removeUser}`.

- [ ] **Step 1: Replace API wrapper and types** per the interface block above (mirror the old `adminRolesApi` axios call style at `adminApi.ts:433-457`).

- [ ] **Step 2: Build the groups page** — structure mirrors the old roles page minus color/display-name/is_default: header + "Create group" button; group cards (name, member count, permission count); detail panel with permission picker **grouped by `app_label`/`model`** and a member list with add/remove (username search can reuse the admin user list endpoint). ALL strings via `useTranslations('AdminGroups')` — no hardcoded English. Add the full key set to BOTH `en.json` and `vi.json` in the same commit, e.g.:

```json
"AdminGroups": {
  "title": "Groups & Permissions",
  "subtitle": "Groups are shared with the Django site — changes apply to both.",
  "createGroup": "Create group",
  "groupName": "Group name",
  "members": "Members",
  "permissions": "Permissions",
  "addMember": "Add member",
  "removeMember": "Remove",
  "deleteGroup": "Delete group",
  "confirmDelete": "Delete group \"{name}\"? Members lose its permissions.",
  "saved": "Group saved",
  "deleted": "Group deleted"
}
```

with the Vietnamese equivalents (`"title": "Nhóm & Quyền hạn"`, `"subtitle": "Nhóm được dùng chung với trang Django — thay đổi áp dụng cho cả hai."`, `"createGroup": "Tạo nhóm"`, `"groupName": "Tên nhóm"`, `"members": "Thành viên"`, `"permissions": "Quyền hạn"`, `"addMember": "Thêm thành viên"`, `"removeMember": "Xóa"`, `"deleteGroup": "Xóa nhóm"`, `"confirmDelete": "Xóa nhóm \"{name}\"? Thành viên sẽ mất các quyền của nhóm."`, `"saved": "Đã lưu nhóm"`, `"deleted": "Đã xóa nhóm"`).

- [ ] **Step 3: Verify** — `cd claoj-web && npx tsc --noEmit && npm run build`
Expected: clean build; `grep -rn "adminRolesApi\|RoleWithUserCount" src/` → no output.

- [ ] **Step 4: Commit**

```bash
git add -A claoj-web/
git commit -m "feat(admin-ui): Django groups manager replaces roles page"
```

---

### Task 8: Refresh tokens → Redis

**Files:**
- Create: `claoj/auth/tokenstore/tokenstore.go` (interface + Redis impl + memory impl)
- Test: `claoj/auth/tokenstore/tokenstore_test.go`
- Modify: `claoj/api/v2/auth/auth.go` (`Login` :174-188, `Refresh` :216-273, `Logout` :303-338, `RevokeAllSessions` :349-359)
- Modify: `claoj/models/runtime.go:106-120` (delete `RefreshToken` model)
- Modify: `claoj/main.go` (initialize store)
- Modify tests: `claoj/api/v2/auth_login_test.go`, `claoj/integration/auth_flow_test.go`, `claoj/integration/registration_flow_test.go`, `claoj/integration/helpers.go`, `claoj/service/user/user_service_test.go`

**Interfaces:**
- Produces:

```go
package tokenstore

type Entry struct {
	UserID    uint      `json:"user_id"`
	FamilyID  string    `json:"family_id"`
	Revoked   bool      `json:"revoked"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UserAgent string    `json:"user_agent,omitempty"`
	ClientIP  string    `json:"client_ip,omitempty"`
}

type Store interface {
	Save(token string, e Entry) error
	Get(token string) (*Entry, bool, error) // (entry, found, err)
	Revoke(token string) error              // marks revoked, keeps entry until TTL (rotation-reuse detection)
	RevokeFamily(familyID string) error
	RevokeAllForUser(userID uint) error
}

func NewRedisStore(client *redis.Client) Store
func NewMemoryStore() Store // for tests and Redis-less dev
```

- Redis layout: value `rt:{sha256hex(token)}` → JSON Entry with TTL to `ExpiresAt`; index sets `rtfam:{familyID}` and `rtuser:{userID}` holding token hashes (TTL refreshed to the longest member's expiry). `Revoke` rewrites the value with `Revoked:true` preserving remaining TTL. `RevokeFamily`/`RevokeAllForUser` iterate the index set and revoke each.
- Package-level wiring in `api/v2/auth/auth.go`: `var RefreshStore tokenstore.Store`; `main.go` sets `authapi.RefreshStore = tokenstore.NewRedisStore(cache.Client)` when Redis is up, else `NewMemoryStore()` with a startup log warning ("refresh sessions are in-memory; logins reset on restart").

- [ ] **Step 1: Write failing store tests** against `NewMemoryStore()` (Save→Get roundtrip; Get unknown → found=false; Revoke → entry found with Revoked=true; RevokeFamily revokes both members; RevokeAllForUser; expired entry → found=false).
- [ ] **Step 2: Run** — `go test ./auth/tokenstore/ -v` → compile FAIL.
- [ ] **Step 3: Implement** both stores (memory: `map[string]Entry` + `sync.Mutex` + expiry check on Get; Redis as per layout). Run → PASS.
- [ ] **Step 4: Rewrite the four auth handlers** replacing every `models.RefreshToken` DB access:
  - `Login`: `RefreshStore.Save(refreshToken, tokenstore.Entry{UserID, FamilyID, ExpiresAt, CreatedAt: time.Now(), UserAgent, ClientIP})`.
  - `Refresh`: `Get` → not found or expired → 401; **found && Revoked → token-reuse: `RevokeFamily(entry.FamilyID)` + 401**; else `Revoke(old)` + `Save(new)` with same FamilyID.
  - `Logout`: `Revoke(token)` (both branches that used `Update("revoked_at", ...)`).
  - `RevokeAllSessions`: fix the bug — read `c.Get("user_id")` (NOT `c.GetUint("userID")`), then `RevokeAllForUser(userID)`.
- [ ] **Step 5: Delete the model** (`models/runtime.go:106-120`) and update tests: in each listed test file replace `models.RefreshToken` AutoMigrate/queries with assertions against a `NewMemoryStore()` injected into the auth package (`authapi.RefreshStore = tokenstore.NewMemoryStore()` in test setup). Integration `helpers.go:336` — delete the `DELETE FROM refresh_token` cleanup line.
- [ ] **Step 6: Run everything** — `go test ./...` → PASS.
- [ ] **Step 7: Commit**

```bash
git add -A claoj/
git commit -m "feat(auth): move refresh tokens to Redis with rotation-reuse detection"
```

---

### Task 9: Password-reset + email-verification tokens → Redis

**Files:**
- Create: `claoj/auth/tokenstore/onetime.go` (+ tests in `onetime_test.go`)
- Modify: `claoj/api/v2/auth/password.go`, `claoj/api/v2/verify.go`, `claoj/api/v2/auth/auth.go` (wherever `models.PasswordResetToken` / `models.EmailVerificationToken` are used — run `grep -rn "PasswordResetToken\|EmailVerificationToken" claoj/ --include="*.go"` for the authoritative list)
- Modify: `claoj/models/runtime.go` (delete both models)
- Modify tests: `claoj/api/v2/verify_test.go`, `claoj/integration/registration_flow_test.go`, `claoj/integration/helpers.go` (drop from AutoMigrate)

**Interfaces:**
- Produces:

```go
// OneTimeStore issues and atomically consumes single-use tokens (password
// reset, email verification). Backed by Redis GETDEL in production.
type OneTimeStore interface {
	Issue(kind string, token string, userID uint, ttl time.Duration) error
	Consume(kind string, token string) (userID uint, ok bool, err error) // deletes on read
	Invalidate(kind string, userID uint) error // revoke outstanding tokens for a user
}
func NewRedisOneTime(client *redis.Client) OneTimeStore
func NewMemoryOneTime() OneTimeStore
```

Kinds: `"pwreset"`, `"emailverify"`. Redis keys `ott:{kind}:{sha256hex(token)}` → userID with TTL; reverse index `ottuser:{kind}:{userID}` for `Invalidate`.

- [ ] **Step 1: Failing tests** (memory impl): Issue→Consume returns userID once, second Consume ok=false; expired → ok=false; Invalidate kills outstanding token.
- [ ] **Step 2: Implement both impls; run** → PASS.
- [ ] **Step 3: Swap the handlers.** Keep TTLs identical to the current DB rows' expiry logic (read the current code for the exact durations — reset tokens and verification tokens each have an expiry constant/field; reuse those values). Wire package var + main.go init exactly like Task 8.
- [ ] **Step 4: Delete both models; update the listed tests; `go test ./...`** → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(auth): one-time tokens (password reset, email verify) moved to Redis"`

---

### Task 10: Audit log → Redis Stream

**Files:**
- Create: `claoj/auditlog/store.go` (+ `store_test.go`) — new package replacing `service/auditlog`
- Delete: `claoj/service/auditlog/`, `claoj/models/audit_log.go`
- Modify: `claoj/middleware/audit.go`, `claoj/api/v2/admin_config_audit.go` (:169-260), `claoj/middleware/user_auth_test.go` (drop `models.AuditLog` AutoMigrate)

**Interfaces:**
- Produces:

```go
package auditlog

type Entry struct {
	ID        string    `json:"id"` // Redis stream ID, e.g. "1721300000000-0"
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	ResourceID string   `json:"resource_id"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Details   string    `json:"details"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Store interface {
	Write(e Entry) error
	List(f Filters, page, pageSize int) ([]Entry, int64, error) // newest first
	Get(id string) (*Entry, error)
}
type Filters struct{ Action, Resource, Status string; UserID uint; Start, End *time.Time }
func NewRedisStore(client *redis.Client) Store // XADD "audit:log" MAXLEN ~100000; List via XRevRange + in-code filter
func NewMemoryStore() Store
```

- **API change (document in commit):** audit entry `id` becomes a string (stream ID). `admin_config_audit.go` response field types change accordingly; no frontend consumer exists (verified — no audit wrapper in `adminApi.ts`).

- [ ] **Step 1: Failing store tests** (memory): Write 3 → List newest-first; filter by action; Get by ID; pagination.
- [ ] **Step 2: Implement; run** → PASS.
- [ ] **Step 3: Rewrite `middleware/audit.go`:** DELETE the broken path guard (`strings.HasPrefix(path, "/api/v2/admin")` at :20 — the middleware is only mounted on the admin group, so no guard is needed) and write via the store (async, as now). Keep skipping GET requests if the current code does. Rewrite the two handlers in `admin_config_audit.go` on the store. Wire package var + main.go init like Tasks 8-9.
- [ ] **Step 4: `go test ./...`** → PASS. Manual check: `curl -s localhost:8080/api/audit-logs` (with admin cookie) returns entries after performing an admin POST.
- [ ] **Step 5: Commit** — `git commit -m "feat(audit): audit log moved to Redis stream; fix middleware path guard"`

---

### Task 11: Drop contest-level max_submissions (KEEP per-problem)

`judge_contest.max_submissions` is a v2-only column; `judge_contestproblem.max_submissions` is a REAL Django column (`OJ/judge/models/contest.py:565`) and must stay.

**Files:**
- Modify (backend): `claoj/models/contest.go:76` (delete `Contest.MaxSubmissions` — keep `ContestProblem.MaxSubmissions` :148), `claoj/service/contest/types.go` (:45/:70/:96 — delete only the CONTEST-level DTO fields; inspect each to confirm which struct it belongs to), `claoj/service/contest/contest_service.go` (:117, :204-205, :511), `claoj/api/v2/admin_contest.go` (:116, :157, :202, :226), `claoj/api/v2/submit.go` (:145-153 delete contest-level enforcement; KEEP :170-178 per-problem enforcement)
- Modify (frontend): `claoj-web/src/components/admin/contest-form/FormatSection.tsx` (:7, :33, :60-72 — remove the Max Submissions input block), `BasicInfoSection.tsx:36`, `ScheduleSection.tsx:34`, `claoj-web/src/components/admin/ContestForm.tsx` (:30, :81, :179), `claoj-web/src/app/[locale]/admin/contests/[key]/edit/page.tsx:94`, `claoj-web/src/types/index.ts` (:344, :446)

- [ ] **Step 1: Backend removal** — delete field + all listed uses. `go build ./... && go test ./...` → PASS.
- [ ] **Step 2: Frontend removal** — delete input + type plumbing. `npx tsc --noEmit && npm run build` → PASS. `grep -rn "max_submissions" claoj-web/src claoj/ --include="*.go" --include="*.ts*"` → only `ContestProblem`/per-problem hits remain.
- [ ] **Step 3: Commit** — `git commit -m "refactor!: drop v2-only contest-level max_submissions (per-problem limit kept)"`

---

### Task 12: Drop the problem-suggestion workflow

Django's real columns `suggester_id` (and permission `judge.suggest_new_problem`) STAY. The v2-only columns (`suggestion_status`, `suggestion_notes`, `suggestion_reviewed_at`, `suggestion_reviewed_by_id`) and the approve/reject workflow GO.

**Files:**
- Delete (backend): `claoj/service/problemsuggestion/` (whole package), `claoj/api/v2/problem_suggest.go`
- Modify (backend): `claoj/models/problem.go` (:72-75 delete `SuggestionStatus`, `SuggestionNotes`, `SuggestionReviewedAt`, `SuggestionReviewedBy`; :81 delete `ReviewedBy` relation; KEEP `SuggesterID` :71 and `Suggester` :80), `claoj/service/problem/problem_service.go` (:41, :508), `claoj/api/v2/admin.go` (:22, :34, :66-71 unwire suggestion service), `claoj/api/router.go` (:246-250 admin suggestion routes; :408-409 `POST /problems/suggest`, `GET /my-suggestions`), `claoj/contribution/service.go` (:67-75)
- Delete (frontend): `claoj-web/src/app/[locale]/problems/suggest/page.tsx`, `claoj-web/src/components/problems/ProblemSuggestForm.tsx`, `claoj-web/src/app/[locale]/admin/problem-suggestions/page.tsx`, `claoj-web/src/app/[locale]/admin/problem-suggestions/[id]/page.tsx`
- Modify (frontend): `claoj-web/src/lib/api.ts` (:235-273 delete `problemSuggestionApi`), `claoj-web/src/app/[locale]/admin/layout.tsx` (remove the Problem Suggestions sidebar link), plus `grep -rn "suggest" claoj-web/src --include="*.tsx" -il` for any remaining links (e.g. a "Suggest a problem" button on the problems page)

- [ ] **Step 1: Backend.** Contribution query at `contribution/service.go:67-75` becomes Django-semantics "suggested problems that got published":

```go
	// Suggested problems that were accepted (suggester set, problem now public).
	// Mirrors Django CLAOJ: a suggestion-in-progress has suggester != NULL and
	// is_public = false (Problem.is_suggesting, OJ problem.py:226-227).
	var approvedSuggestionCount int64
	err = db.DB.Model(&models.Problem{}).
		Where("suggester_id = ? AND is_public = ?", profileID, true).
		Count(&approvedSuggestionCount).Error
```

Delete everything else listed. `go build ./... && go test ./...` → PASS.
- [ ] **Step 2: Frontend.** Delete pages/components/API wrapper + links. `npx tsc --noEmit && npm run build` → PASS. `grep -rn "problemSuggestionApi\|problem-suggestions\|suggestion_status" claoj-web/src claoj/ --include="*.go" --include="*.ts*"` → no output.
- [ ] **Step 3: Commit** — `git commit -m "refactor!: drop v2-only problem suggestion workflow (Django suggester_id kept)"`

---

### Task 13: Schema audit, DDL guard, and cleanup scripts

**Files:**
- Create: `claoj/db/ddl_guard.go` (+ `ddl_guard_test.go`)
- Create: `claoj/integration/schema_parity_test.go`
- Create: `OJ-v2/scripts/cleanup_v2_tables.sql`, `OJ-v2/scripts/v2_runtime_tables.sql`
- Create: `OJ-v2/docs/schema-audit.md`
- Modify: `claoj/db/db.go` (register guard), spec file (amendments)

- [ ] **Step 1: DDL guard (failing test first).**

```go
// claoj/db/ddl_guard_test.go
package db

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDDLGuardBlocksDDLAllowsDML(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// table created BEFORE the guard is installed
	require.NoError(t, database.Exec("CREATE TABLE t (id INTEGER)").Error)
	RegisterDDLGuard(database)
	require.NoError(t, database.Exec("INSERT INTO t (id) VALUES (1)").Error)
	require.NoError(t, database.Exec("SELECT * FROM t").Error)
	require.Panics(t, func() { database.Exec("ALTER TABLE t ADD COLUMN x INTEGER") })
	require.Panics(t, func() { database.Exec("CREATE TABLE u (id INTEGER)") })
	require.Panics(t, func() { database.Exec("DROP TABLE t") })
}
```

Implementation:

```go
// claoj/db/ddl_guard.go
package db

import (
	"regexp"

	"gorm.io/gorm"
)

var ddlRe = regexp.MustCompile(`(?is)^\s*(CREATE|ALTER|DROP|TRUNCATE|RENAME)\b`)

// RegisterDDLGuard panics on any DDL statement. The schema of the shared
// database is owned by Django migrations; OJ-v2 must only touch rows.
func RegisterDDLGuard(g *gorm.DB) {
	check := func(tx *gorm.DB) {
		if ddlRe.MatchString(tx.Statement.SQL.String()) {
			panic("DDL blocked (schema is Django-owned): " + tx.Statement.SQL.String())
		}
	}
	_ = g.Callback().Raw().Before("gorm:raw").Register("claoj:ddl_guard", check)
}
```

Call `RegisterDDLGuard(DB)` at the end of `db.Connect()`. Run test → PASS.

- [ ] **Step 2: Column-by-column audit.** For each Go model file, diff fields against the Django source (`OJ/judge/models/problem.py`, `contest.py`, `profile.py`, `submission.py`, `comment.py`, `interface.py`, `ticket.py`, `runtime.py`, `problem_data.py`). Record results in `docs/schema-audit.md` as a table (`Go model | Django model | verdict | note`). Expected findings already handled by earlier tasks: Roles field (T5), Contest.MaxSubmissions (T11), suggestion columns (T12), RefreshToken/AuditLog/Role models (T5/8/10). Fix any NEW drift found (wrong column name/type → correct the gorm tag; v2-only column → delete field + code paths). Retained-exception tables get their own table section (`notification`, `notification_preference`, `totp_device`, `backup_code`).

- [ ] **Step 3: Schema-parity smoke test** (env-gated — runs only when a Django-schema MySQL DSN is provided):

```go
// claoj/integration/schema_parity_test.go
package integration

import (
	"os"
	"testing"

	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestSchemaParity verifies every GORM model SELECTs cleanly against a real
// Django-migrated database. Provision one with:
//   cd OJ && python manage.py migrate   (against an empty MySQL db)
// then: CLAOJ_DJANGO_DB_DSN="user:pass@tcp(127.0.0.1:3306)/claoj_schema?parseTime=true" go test ./integration/ -run TestSchemaParity
func TestSchemaParity(t *testing.T) {
	dsn := os.Getenv("CLAOJ_DJANGO_DB_DSN")
	if dsn == "" {
		t.Skip("CLAOJ_DJANGO_DB_DSN not set")
	}
	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	djangoModels := []interface{}{
		&models.AuthUser{}, &models.AuthGroup{}, &models.AuthPermission{},
		&models.DjangoContentType{}, &models.AuthUserGroup{},
		&models.AuthGroupPermission{}, &models.AuthUserPermission{},
		&models.Profile{}, &models.Organization{}, &models.OrganizationRequest{},
		&models.WebAuthnCredential{},
		&models.Problem{}, &models.ProblemGroup{}, &models.ProblemType{},
		&models.License{}, &models.ProblemTranslation{}, &models.ProblemClarification{},
		&models.LanguageLimit{}, &models.Solution{},
		&models.ProblemData{}, &models.ProblemTestCase{},
		&models.Contest{}, &models.ContestTag{}, &models.ContestAnnouncement{},
		&models.ContestClarification{}, &models.ContestParticipation{},
		&models.ContestProblem{}, &models.ContestSubmission{}, &models.Rating{},
		&models.Submission{}, &models.SubmissionSource{}, &models.SubmissionTestCase{},
		&models.Comment{}, &models.CommentVote{}, &models.CommentLock{}, &models.CommentRevision{},
		&models.GeneralIssue{}, &models.Ticket{}, &models.TicketMessage{},
		&models.BlogPost{}, &models.BlogVote{}, &models.MiscConfig{}, &models.NavigationBar{},
		&models.Language{}, &models.Judge{}, &models.RuntimeVersion{},
	}
	for _, m := range djangoModels {
		// Select every mapped column of one row; unknown columns error out.
		err := database.Limit(1).Find(m).Error
		require.NoErrorf(t, err, "model %T does not match Django schema", m)
	}
}
```

(Adjust the list to the final model inventory after the audit; every Django-table model must be present.)

- [ ] **Step 4: Cleanup scripts.**

`OJ-v2/scripts/cleanup_v2_tables.sql` (MANUAL, optional — for databases where old v2 migrations were applied):

```sql
-- Removes tables/columns created by the deleted OJ-v2 migrations 001-007.
-- Run MANUALLY, once, after backing up. Safe to skip: Django ignores these.
DROP TABLE IF EXISTS judge_profile_roles;
DROP TABLE IF EXISTS judge_role_permissions;
DROP TABLE IF EXISTS judge_permission;
DROP TABLE IF EXISTS judge_role;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS refresh_token;
DROP TABLE IF EXISTS password_reset_token;
DROP TABLE IF EXISTS email_verification_token;
ALTER TABLE judge_contest DROP COLUMN max_submissions;
ALTER TABLE judge_problem
  DROP FOREIGN KEY fk_problem_suggestion_reviewed_by;
ALTER TABLE judge_problem
  DROP COLUMN suggestion_status,
  DROP COLUMN suggestion_notes,
  DROP COLUMN suggestion_reviewed_at,
  DROP COLUMN suggestion_reviewed_by_id;
```

`OJ-v2/scripts/v2_runtime_tables.sql` (MANUAL, fresh installs only — the four retained additive tables; write `CREATE TABLE IF NOT EXISTS` statements matching the GORM models in `models/notification.go` and `models/runtime.go` for `notification`, `notification_preference`, `totp_device`, `backup_code`, utf8mb4, InnoDB).

- [ ] **Step 5: Amend the spec** (`docs/superpowers/specs/2026-07-18-django-parity-and-vietnamese-design.md`): add a "## Amendments (implementation findings)" section stating: (a) retained-table exception (the four tables above, reason: live 2FA/notification data, additive-only); (b) mono font stays the system stack — JetBrains Mono lacks a Vietnamese subset on Google Fonts and system monos (Consolas/Menlo) cover Vietnamese; (c) password-reset/email-verify tokens also moved to Redis (same principle as refresh tokens).

- [ ] **Step 6: Commit** — `git commit -m "feat(db): DDL guard, schema audit, parity test, cleanup scripts"`

---

### Task 14: Vietnamese font — Be Vietnam Pro

**Files:**
- Modify: `claoj-web/src/app/[locale]/layout.tsx` (:4, :15-18, :117)
- Modify: `claoj-web/src/app/[locale]/globals.css` (:140)

- [ ] **Step 1: Swap the font.** In `layout.tsx`:

```tsx
import { Be_Vietnam_Pro } from "next/font/google";

const beVietnamPro = Be_Vietnam_Pro({
  subsets: ["latin", "vietnamese"],
  weight: ["300", "400", "500", "600", "700"],
  variable: "--font-be-vietnam-pro",
});
```

Body class (:117): `${beVietnamPro.variable} font-sans antialiased min-h-screen flex flex-col`.
In `globals.css:140`: `--font-sans: var(--font-be-vietnam-pro), ui-sans-serif, system-ui;`
Leave `--font-mono` (:141) untouched.

- [ ] **Step 2: Verify no other font surface exists.**
Run: `grep -rn "font-outfit\|Outfit" claoj-web/src` → no output. `grep -rn "fonts.googleapis\|@font-face" claoj-web/src` → no output.

- [ ] **Step 3: Visual glyph check.** `npm run dev`, open `/vi`, paste into any visible text (e.g. search box) the string `ạ ế ở ữ đ Ị Ổ ẵ Ứ` and confirm in devtools (Rendered Fonts panel) every glyph renders in "Be Vietnam Pro" — zero fallback fonts. Check dark theme too.

- [ ] **Step 4: Commit** — `git commit -m "fix(i18n): Be Vietnam Pro font — full Vietnamese glyph coverage"`

---

### Task 15: i18n plumbing — one middleware, key-parity gate, EN catch-up

**Files:**
- Delete: `claoj-web/src/i18n.ts` (dead duplicate config), `claoj-web/middleware.ts` (root)
- Modify: `claoj-web/src/proxy.ts` (make it the single intl middleware, importing shared routing)
- Create: `claoj-web/scripts/check-i18n.mjs`
- Modify: `claoj-web/package.json` (add script), `claoj-web/src/i18n/en.json` (add the 203 missing keys / 4 namespaces)

- [ ] **Step 1: Unify middleware.** Rewrite `src/proxy.ts` (Next 16 convention) to:

```ts
import createMiddleware from 'next-intl/middleware';
import { routing } from './navigation';

export default createMiddleware(routing);

export const config = {
    matcher: ['/((?!api|static|_next|_vercel|.*\\..*).*)']
};
```

Delete root `middleware.ts` and `src/i18n.ts`. Run `npm run dev`, verify `/` and `/vi` both render and the language switcher works.

- [ ] **Step 2: Parity script (write it failing first — EN currently misses 203 keys).**

```js
// claoj-web/scripts/check-i18n.mjs
import { readFileSync } from 'node:fs';

const load = (p) => JSON.parse(readFileSync(new URL(p, import.meta.url), 'utf8'));
const flatten = (obj, prefix = '') =>
    Object.entries(obj).flatMap(([k, v]) =>
        typeof v === 'object' && v !== null ? flatten(v, `${prefix}${k}.`) : [`${prefix}${k}`]);

const en = new Set(flatten(load('../src/i18n/en.json')));
const vi = new Set(flatten(load('../src/i18n/vi.json')));
const missingInEn = [...vi].filter(k => !en.has(k));
const missingInVi = [...en].filter(k => !vi.has(k));

if (missingInEn.length || missingInVi.length) {
    if (missingInEn.length) console.error(`Missing in en.json (${missingInEn.length}):\n  ` + missingInEn.join('\n  '));
    if (missingInVi.length) console.error(`Missing in vi.json (${missingInVi.length}):\n  ` + missingInVi.join('\n  '));
    process.exit(1);
}
console.log(`i18n OK: ${en.size} keys in both locales.`);
```

`package.json` scripts: `"i18n:check": "node scripts/check-i18n.mjs"`. Run it → FAILS listing 203 keys.

- [ ] **Step 3: Fill `en.json`** — add every missing key with the proper English string (translate from the Vietnamese value's meaning; namespaces `Organization`, `Ticket`, `Settings`, `Notification` plus the per-namespace stragglers). Run `npm run i18n:check` → PASS.

- [ ] **Step 4: Commit** — `git commit -m "fix(i18n): single intl middleware, en/vi key parity + check script"`

---

### Task 16: Localize submission status maps

**Files:**
- Modify: `claoj-web/src/components/submission/TestCaseResults.tsx`, `claoj-web/src/components/submission/SingleSubmissionWidget.tsx`, `claoj-web/src/app/[locale]/submissions/[id]/page.tsx`
- Create: `claoj-web/src/lib/submissionStatus.ts`
- Modify: `claoj-web/src/i18n/en.json` + `vi.json` (namespace `Submissions.status`)

- [ ] **Step 1: Add the shared status key helper + messages.**

```ts
// claoj-web/src/lib/submissionStatus.ts
// Verdict codes come from the judge; labels are translated via
// the Submissions.status.* namespace. Use: t(`status.${statusKey(code)}`)
const KNOWN = new Set([
    'AC', 'WA', 'TLE', 'MLE', 'OLE', 'IR', 'RTE', 'CE', 'IE', 'AB',
    'QU', 'P', 'G', 'D', 'CP', 'SC',
]);
export const statusKey = (code: string): string => (KNOWN.has(code) ? code : 'unknown');
```

`en.json` (mirror in `vi.json` with the Vietnamese strings shown):

```json
"Submissions": {
  "status": {
    "AC": "Accepted",            // vi: "Chấp nhận"
    "WA": "Wrong Answer",        // vi: "Sai kết quả"
    "TLE": "Time Limit Exceeded",// vi: "Quá thời gian"
    "MLE": "Memory Limit Exceeded", // vi: "Quá bộ nhớ"
    "OLE": "Output Limit Exceeded", // vi: "Quá giới hạn xuất"
    "IR": "Invalid Return",      // vi: "Trả về không hợp lệ"
    "RTE": "Runtime Error",      // vi: "Lỗi thực thi"
    "CE": "Compile Error",       // vi: "Lỗi biên dịch"
    "IE": "Internal Error",      // vi: "Lỗi hệ thống"
    "AB": "Aborted",             // vi: "Đã hủy"
    "QU": "Queued",              // vi: "Đang chờ"
    "P": "Processing",           // vi: "Đang xử lý"
    "G": "Grading",              // vi: "Đang chấm"
    "D": "Completed",            // vi: "Hoàn thành"
    "CP": "Compiling",           // vi: "Đang biên dịch"
    "SC": "Short Circuited",     // vi: "Dừng sớm"
    "unknown": "Unknown"         // vi: "Không rõ"
  }
}
```

(JSON has no comments — the `// vi:` notes above show the values to put in `vi.json`.)

- [ ] **Step 2: Rewrite the three files** to translate labels at render time — keep colors/icons in the local maps, but replace every hardcoded label string with `t(\`status.${statusKey(code)}\`)` (`useTranslations('Submissions')`). Also translate the surrounding hardcoded strings found in research: "Test Cases", "passed", "Total Time", "Max Memory", "Result/Time/Memory/Score" headers, "Feedback", "Output", "No test cases available yet.", "Submission #", "Live", "Submission not found", StatBox labels — add keys under `Submissions` in BOTH locales.
- [ ] **Step 3: Verify** — `npm run i18n:check && npx tsc --noEmit && npm run build` → PASS. Open a submission page under `/vi`: verdicts display in Vietnamese.
- [ ] **Step 4: Commit** — `git commit -m "fix(i18n): translate submission status labels and detail pages"`

---

### Task 17: Localize the admin area

**Files:** the 24 admin files with hardcoded strings (checklist below), `claoj-web/src/i18n/en.json` + `vi.json` (new `Admin` namespace tree).

Checklist (tick each file when it contains zero hardcoded user-visible strings):

- [ ] `admin/layout.tsx` (sidebar labels `adminLinks` :32-118, "Admin Panel", "Logged in as", "Logout")
- [ ] `admin/users/page.tsx`  — [ ] `admin/contests/page.tsx` — [ ] `admin/contests/create/page.tsx`
- [ ] `admin/contests/[key]/edit/page.tsx` — [ ] `admin/contests/[key]/participations/page.tsx` — [ ] `admin/contests/tags/page.tsx`
- [ ] `admin/problems/page.tsx` — [ ] `admin/problems/create/page.tsx` — [ ] `admin/problems/[code]/edit/page.tsx` — [ ] `admin/problems/[code]/data/page.tsx`
- [ ] `admin/judges/page.tsx` — [ ] `admin/judges/[id]/page.tsx`
- [ ] `admin/organizations/page.tsx` — [ ] `admin/submissions/page.tsx`
- [ ] `admin/languages/page.tsx` — [ ] `admin/language-limits/page.tsx` — [ ] `admin/licenses/page.tsx`
- [ ] `admin/taxonomy/page.tsx` — [ ] `admin/blog-posts/page.tsx` — [ ] `admin/misc-configs/page.tsx`
- [ ] `admin/navigation-bars/page.tsx` — [ ] `admin/comments/page.tsx` — [ ] `admin/tickets/[id]/page.tsx`
- [ ] shared admin components: `components/admin/contest-form/FormatSection.tsx` (+ siblings), `components/admin/testcases/*`, `components/admin/Admin*.tsx` banners/sidebar

Method (same for every file):
1. Add `const t = useTranslations('Admin');` (client) or `getTranslations` (server).
2. Move every heading, table header, button label, `placeholder=`, `title=`, `aria-label=`, `toast.*(...)`, `confirm(...)`, `prompt(...)` string into `Admin.<area>.<key>` in BOTH locale files (English + Vietnamese written together).
3. Key naming: `Admin.users.title`, `Admin.users.banConfirm`, `Admin.common.save`, `Admin.common.delete`, `Admin.common.previous`, `Admin.common.next`, etc. Shared strings go under `Admin.common`.

Acceptance gates after the sweep:

```bash
npm run i18n:check                                            # PASS
npx tsc --noEmit && npm run build                             # PASS
grep -rn 'placeholder="[A-Za-z]' src/app/\[locale\]/admin     # no output
grep -rnE "toast\.(success|error)\('[A-Z]" src/app/\[locale\]/admin  # no output
grep -rnE "(confirm|prompt|alert)\('[A-Z]" src/app/\[locale\]/admin  # no output
```

- [ ] **Final step: Commit** — `git commit -m "fix(i18n): fully localize admin area (en/vi)"` (commit per 4-6 files as you go, not one giant commit).

---

### Task 18: Localize remaining user-facing pages

**Files:** `claoj-web/src/app/[locale]/settings/page.tsx`, `verify-email/page.tsx`, `resend-verification/page.tsx`, `notifications/page.tsx`, `components/settings/account/*`, plus any file surfaced by the final gate greps. Locale files: extend the existing `Settings`, `Notification`, `Auth` namespaces (they already exist in `vi.json` — Task 15 added them to `en.json`).

- [ ] **Step 1: Sweep** with the same method as Task 17.
- [ ] **Step 2: Gates:**

```bash
npm run i18n:check && npx tsc --noEmit && npm run build       # PASS
grep -rn 'placeholder="[A-Za-z]' src/app src/components | grep -v admin   # no output
grep -rnE "(alert|confirm)\('[A-Z]" src/app src/components | grep -v admin # no output
```

- [ ] **Step 3: Manual vi pass** — visit under `/vi`: home, problems list, one problem, submit, submissions, one submission detail, contests, one contest, login, register, settings, notifications, user profile. Zero English leakage, zero broken glyphs.
- [ ] **Step 4: Commit** — `git commit -m "fix(i18n): localize settings, verification, notification pages"`

---

### Task 19: Final verification

- [ ] **Backend:** `cd claoj && go vet ./... && go build ./... && go test ./...` → all PASS.
- [ ] **No-DDL proof:** `grep -rn "AutoMigrate" claoj/ --include="*.go" | grep -v "_test.go"` → no output. DDL-guard test green.
- [ ] **Schema parity (if a Django-schema MySQL is available):** `CLAOJ_DJANGO_DB_DSN=... go test ./integration/ -run TestSchemaParity -v` → PASS.
- [ ] **Frontend:** `cd claoj-web && npm run i18n:check && npx tsc --noEmit && npm test && npm run build` → all PASS.
- [ ] **End-to-end smoke (dev):** boot backend + frontend; login as a non-staff user with group "Editors" (create the group via the new admin UI as a superuser, grant `judge.edit_own_problem`); verify: the user can edit their own problem, cannot edit others'; revoking the group's permission takes effect within 60s (or immediately after another admin write); Vietnamese pages render fully in Be Vietnam Pro.
- [ ] **Commit any fixes**, then: `git log --oneline` sanity review of the task commits.
