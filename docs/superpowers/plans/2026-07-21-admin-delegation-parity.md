# Admin Delegation-of-Authority Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make v2's admin endpoints enforce v1 (DMOJ)'s per-object delegation of authority, so a staff user can only edit the problems/contests/organizations they actually own (author/curator/org-admin) or hold `edit_all_*` for, and superusers are never wrongly locked out.

**Architecture:** Keep the `is_staff`-gated admin surface. Fix the route gate to also admit superusers (Gap B). Add small route-level **guard middlewares** in the `auth` package that load the target object with its author/curator associations and call the existing DMOJ-faithful `auth.CanEdit*` helpers; wire those guards onto the existing admin routes in `router.go`. Two guards that can't be a simple middleware (org-admin appointment, clarification-delete by id) are enforced inline.

**Tech Stack:** Go, Gin, GORM (MariaDB, schema owned by Django — rows only), testify. Existing auth helpers in `claoj/auth/access.go` + `claoj/auth/perms.go`.

## Global Constraints

- **Model A only.** Admin surface stays `is_staff`-gated; superusers bypass the staff gate. Do NOT open endpoints to non-staff users. No frontend changes.
- **Scope: delegated resources only** — problems, contests, organizations, org-admin appointment. Do NOT touch site-config endpoints (languages, licenses, problem-groups/types, navigation bars, misc-configs, groups, contest-tag *catalog* CRUD).
- **Reuse existing helpers verbatim** — `auth.CanEditProblem`, `auth.CanEditContest`, `auth.CanRejudge`, `auth.HasPerm`. Add exactly one new helper (`auth.CanEditOrganization`). Do NOT reimplement delegation logic in handlers.
- **`CanEditProblem`/`CanEditContest` require the object loaded with `Preload("Authors").Preload("Curators")`** or a legitimate author will be wrongly denied.
- **DB is rows-only.** No AutoMigrate/DDL in production code. Query Django tables by their existing names (`judge_organization_admins`, `judge_contest_authors`, …).
- **Exact Django permission strings:** `judge.add_problem`, `judge.clone_problem`, `judge.add_contest`, `judge.clone_contest`, `judge.add_organization`, `judge.edit_all_organization`, `judge.organization_admin`. (Superuser bypass is already built into `HasPerm`.)
- **No permissions migration.** The change intentionally tightens access; do not seed/grant permissions to existing accounts.
- **403 body shape:** `gin.H{"error": "<message>"}` (matches existing handlers).

---

### Task 1: Route gate admits superusers (fixes Gap B)

**Files:**
- Modify: `claoj/auth/middleware.go:131-156` (`AdminRequiredMiddleware`)
- Test: `claoj/auth/middleware_admin_gate_test.go` (create)

**Interfaces:**
- Consumes: `models.AuthUser{ID, IsStaff, IsSuperuser}`, `db.DB`.
- Produces: `AdminRequiredMiddleware()` now allows a request when the user is `is_staff` **OR** `is_superuser`.

- [ ] **Step 1: Write the failing test**

Create `claoj/auth/middleware_admin_gate_test.go`:

```go
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func runAdminGate(t *testing.T, userID uint, setUser bool) int {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if setUser {
		c.Set("user_id", userID)
	}
	AdminRequiredMiddleware()(c)
	return w.Code
}

func TestAdminGate_AllowsSuperuserNotStaff(t *testing.T) {
	setupPermsDB(t)
	u := models.AuthUser{Username: "superonly", IsActive: true, IsStaff: false, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&u).Error)
	require.Equal(t, http.StatusOK, runAdminGate(t, u.ID, true)) // not aborted -> recorder stays 200
}

func TestAdminGate_AllowsStaff(t *testing.T) {
	setupPermsDB(t)
	u := models.AuthUser{Username: "staff", IsActive: true, IsStaff: true, IsSuperuser: false}
	require.NoError(t, db.DB.Create(&u).Error)
	require.Equal(t, http.StatusOK, runAdminGate(t, u.ID, true))
}

func TestAdminGate_DeniesPlainUser(t *testing.T) {
	setupPermsDB(t)
	u := models.AuthUser{Username: "plain", IsActive: true, IsStaff: false, IsSuperuser: false}
	require.NoError(t, db.DB.Create(&u).Error)
	require.Equal(t, http.StatusForbidden, runAdminGate(t, u.ID, true))
}

func TestAdminGate_DeniesAnonymous(t *testing.T) {
	setupPermsDB(t)
	require.Equal(t, http.StatusUnauthorized, runAdminGate(t, 0, false))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd claoj && go test ./auth/ -run TestAdminGate -v`
Expected: `TestAdminGate_AllowsSuperuserNotStaff` FAILS (gets 403, because current gate checks only `is_staff`). Others pass.

- [ ] **Step 3: Implement the gate change**

In `claoj/auth/middleware.go`, replace the body of `AdminRequiredMiddleware` from the DB load through the staff check (currently lines 142-151):

```go
		var user models.AuthUser
		if err := db.DB.Select("id", "is_staff", "is_superuser").First(&user, userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		// v1 parity: is_staff opens the admin surface; a superuser bypasses the
		// staff requirement entirely (Django ModelBackend semantics).
		if !user.IsStaff && !user.IsSuperuser {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
```

Also update the doc comment on line 131 to: `// AdminRequiredMiddleware ensures the user is authenticated AND may use the admin surface (is_staff OR is_superuser).`

- [ ] **Step 4: Run test to verify it passes**

Run: `cd claoj && go test ./auth/ -run TestAdminGate -v`
Expected: all 4 PASS.

- [ ] **Step 5: Commit**

```bash
git add claoj/auth/middleware.go claoj/auth/middleware_admin_gate_test.go
git commit -m "fix(admin): admit superusers (not just is_staff) to admin gate — v1 parity"
```

---

### Task 2: `CanEditOrganization` helper

**Files:**
- Modify: `claoj/auth/access.go` (append helper + membership query)
- Test: `claoj/auth/access_org_test.go` (create)

**Interfaces:**
- Consumes: `GetAccess(c)`, `CurrentProfileID(c)`, `db.DB`, join table `judge_organization_admins(organization_id, profile_id)`.
- Produces:
  - `func CanEditOrganization(c *gin.Context, orgID uint) bool`
  - `func userIsOrgAdmin(orgID, profileID uint) bool`

- [ ] **Step 1: Write the failing test**

Create `claoj/auth/access_org_test.go`:

```go
package auth

import (
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
)

func TestCanEditOrganization(t *testing.T) {
	cases := []struct {
		name       string
		superuser  bool
		perms      []string
		isOrgAdmin bool
		want       bool
	}{
		{"outsider denied", false, nil, false, false},
		{"org admin allowed", false, nil, true, true},
		{"edit_all_organization allowed", false, []string{"edit_all_organization"}, false, true},
		{"organization_admin perm allowed", false, []string{"organization_admin"}, false, true},
		{"superuser allowed", true, nil, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupPermsDB(t)

			org := models.Organization{Name: "Org", Slug: "org", ShortName: "ORG"}
			require.NoError(t, db.DB.Create(&org).Error)

			user := models.AuthUser{Username: "u", IsActive: true, IsSuperuser: tc.superuser}
			require.NoError(t, db.DB.Create(&user).Error)
			profile := newProfileFor(t, user.ID)
			if tc.isOrgAdmin {
				require.NoError(t, db.DB.Model(&org).Association("Admins").Append(&profile))
			}
			grantPermsViaGroup(t, user.ID, tc.perms)

			c := ginContextForUser(user.ID)
			require.Equal(t, tc.want, CanEditOrganization(c, org.ID))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd claoj && go test ./auth/ -run TestCanEditOrganization -v`
Expected: FAIL to compile — `undefined: CanEditOrganization`.

- [ ] **Step 3: Implement the helper**

Append to `claoj/auth/access.go` (before the final line):

```go
// userIsOrgAdmin reports whether profileID is an administrator of the
// organization (membership in judge_organization_admins). joinTable columns
// are the Django defaults for Organization.admins.
func userIsOrgAdmin(orgID, profileID uint) bool {
	var count int64
	db.DB.Table("judge_organization_admins").
		Where("organization_id = ? AND profile_id = ?", orgID, profileID).
		Count(&count)
	return count > 0
}

// CanEditOrganization mirrors DMOJ organization-edit authority
// (OJ/judge/models/profile.py:81 + org mixins): superuser, OR
// judge.edit_all_organization, OR the judge.organization_admin bypass, OR being
// an administrator of this specific organization.
func CanEditOrganization(c *gin.Context, orgID uint) bool {
	access := GetAccess(c)
	if access.IsSuperuser && access.IsActive {
		return true
	}
	if access.HasPerm("judge.edit_all_organization") {
		return true
	}
	if access.HasPerm("judge.organization_admin") {
		return true
	}
	profileID, ok := CurrentProfileID(c)
	return ok && userIsOrgAdmin(orgID, profileID)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd claoj && go test ./auth/ -run TestCanEditOrganization -v`
Expected: all 5 subtests PASS.

- [ ] **Step 5: Commit**

```bash
git add claoj/auth/access.go claoj/auth/access_org_test.go
git commit -m "feat(auth): add CanEditOrganization helper (v1 org-edit delegation)"
```

---

### Task 3: Guard middlewares

**Files:**
- Create: `claoj/auth/admin_guards.go`
- Test: `claoj/auth/admin_guards_test.go`

**Interfaces:**
- Consumes: `HasPerm`, `CanEditProblem`, `CanEditContest`, `CanEditOrganization`, `db.DB`, `models.Problem`, `models.Contest`.
- Produces (all `gin.HandlerFunc` factories):
  - `RequirePerm(codename string) gin.HandlerFunc`
  - `RequireProblemEdit() gin.HandlerFunc` — reads `:code`
  - `RequireContestEdit() gin.HandlerFunc` — reads `:key`
  - `RequireOrgEdit() gin.HandlerFunc` — reads `:id`

- [ ] **Step 1: Write the failing test**

Create `claoj/auth/admin_guards_test.go`:

```go
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// runGuard invokes a guard on a context authenticated as userID with the given
// path params, returning the recorder status and whether the chain was aborted.
func runGuard(userID uint, params gin.Params, guard gin.HandlerFunc) (int, bool) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Params = params
	guard(c)
	return w.Code, c.IsAborted()
}

func TestRequirePerm(t *testing.T) {
	setupPermsDB(t)
	granted := newActiveUser(t, "granted")
	newProfileFor(t, granted.ID)
	grantPermsViaGroup(t, granted.ID, []string{"add_problem"})
	denied := newActiveUser(t, "denied")
	newProfileFor(t, denied.ID)

	code, aborted := runGuard(granted.ID, nil, RequirePerm("judge.add_problem"))
	require.False(t, aborted)
	require.Equal(t, http.StatusOK, code)

	code, aborted = runGuard(denied.ID, nil, RequirePerm("judge.add_problem"))
	require.True(t, aborted)
	require.Equal(t, http.StatusForbidden, code)
}

func TestRequireProblemEdit(t *testing.T) {
	setupPermsDB(t)
	prob := models.Problem{Code: "p1", Name: "P1", IsPublic: false}
	require.NoError(t, db.DB.Create(&prob).Error)

	// Author with edit_own_problem may edit.
	author := newActiveUser(t, "author")
	authorProfile := newProfileFor(t, author.ID)
	require.NoError(t, db.DB.Model(&prob).Association("Authors").Append(&authorProfile))
	grantPermsViaGroup(t, author.ID, []string{"edit_own_problem"})

	// Plain staff (no perms, not author) may not.
	outsider := newActiveUser(t, "outsider")
	newProfileFor(t, outsider.ID)

	p := gin.Params{{Key: "code", Value: "p1"}}
	code, aborted := runGuard(author.ID, p, RequireProblemEdit())
	require.False(t, aborted)
	require.Equal(t, http.StatusOK, code)

	code, aborted = runGuard(outsider.ID, p, RequireProblemEdit())
	require.True(t, aborted)
	require.Equal(t, http.StatusForbidden, code)

	// Missing problem -> 404.
	code, aborted = runGuard(outsider.ID, gin.Params{{Key: "code", Value: "nope"}}, RequireProblemEdit())
	require.True(t, aborted)
	require.Equal(t, http.StatusNotFound, code)
}

func TestRequireContestEdit(t *testing.T) {
	setupPermsDB(t)
	ct := models.Contest{Key: "c1", Name: "C1"}
	require.NoError(t, db.DB.Create(&ct).Error)

	author := newActiveUser(t, "cauthor")
	authorProfile := newProfileFor(t, author.ID)
	require.NoError(t, db.DB.Model(&ct).Association("Authors").Append(&authorProfile))
	grantPermsViaGroup(t, author.ID, []string{"edit_own_contest"})
	outsider := newActiveUser(t, "coutsider")
	newProfileFor(t, outsider.ID)

	p := gin.Params{{Key: "key", Value: "c1"}}
	code, aborted := runGuard(author.ID, p, RequireContestEdit())
	require.False(t, aborted)
	require.Equal(t, http.StatusOK, code)

	code, aborted = runGuard(outsider.ID, p, RequireContestEdit())
	require.True(t, aborted)
	require.Equal(t, http.StatusForbidden, code)
}

func TestRequireOrgEdit(t *testing.T) {
	setupPermsDB(t)
	org := models.Organization{Name: "O1", Slug: "o1", ShortName: "O1"}
	require.NoError(t, db.DB.Create(&org).Error)

	admin := newActiveUser(t, "oadmin")
	adminProfile := newProfileFor(t, admin.ID)
	require.NoError(t, db.DB.Model(&org).Association("Admins").Append(&adminProfile))
	outsider := newActiveUser(t, "ooutsider")
	newProfileFor(t, outsider.ID)

	p := gin.Params{{Key: "id", Value: itoa(org.ID)}}
	code, aborted := runGuard(admin.ID, p, RequireOrgEdit())
	require.False(t, aborted)
	require.Equal(t, http.StatusOK, code)

	code, aborted = runGuard(outsider.ID, p, RequireOrgEdit())
	require.True(t, aborted)
	require.Equal(t, http.StatusForbidden, code)
}

func itoa(u uint) string { return strconv.FormatUint(uint64(u), 10) }
```

Add `"strconv"` to the test file's imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd claoj && go test ./auth/ -run 'TestRequire' -v`
Expected: FAIL to compile — `undefined: RequirePerm` etc.

- [ ] **Step 3: Implement the middlewares**

Create `claoj/auth/admin_guards.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd claoj && go test ./auth/ -run 'TestRequire' -v`
Expected: all subtests PASS.

- [ ] **Step 5: Commit**

```bash
git add claoj/auth/admin_guards.go claoj/auth/admin_guards_test.go
git commit -m "feat(auth): add RequirePerm/RequireProblemEdit/RequireContestEdit/RequireOrgEdit guards"
```

---

### Task 4: Wire problem guards into the router

**Files:**
- Modify: `claoj/api/router.go:132-155` (problem routes) and the create/clone routes
- Test: `claoj/integration/admin_problem_authz_test.go` (create)

**Interfaces:**
- Consumes: `auth.RequireProblemEdit`, `auth.RequirePerm` (Task 3); integration harness (`integration.SetupIntegrationDB`, `integration.CreateTestUser`, `integration.TestRouter`, `integration.MakeRequest`, `loginToken` pattern).
- Produces: problem write routes gated per-object; `POST /admin/problems` requires `judge.add_problem`; clone requires `judge.clone_problem`.

- [ ] **Step 1: Write the failing test**

Create `claoj/integration/admin_problem_authz_test.go`:

```go
package integration_test

import (
	"fmt"
	"net/http"
	"testing"

	v2 "github.com/CLAOJ/claoj/api/v2"
	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/integration"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// makeStaff flips is_staff / is_superuser on a user row.
func makeStaff(t *testing.T, db *gorm.DB, userID uint, staff, super bool) {
	t.Helper()
	require.NoError(t, db.Model(&models.AuthUser{}).Where("id = ?", userID).
		Updates(map[string]interface{}{"is_staff": staff, "is_superuser": super}).Error)
}

// adminProblemRouter builds the real admin chain for the problem-update route.
func adminProblemRouter() *gin.Engine {
	g := integration.TestRouter()
	g.POST("/auth/login", authHandlers.Login)
	g.Use(auth.RequiredMiddleware())
	g.Use(auth.AdminRequiredMiddleware())
	g.PATCH("/admin/problem/:code", auth.RequireProblemEdit(), v2.AdminProblemUpdate)
	return g
}

func TestAdminProblemUpdate_Authz(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	prob := models.Problem{Code: "px", Name: "PX", IsPublic: true}
	require.NoError(t, testDB.DB.Create(&prob).Error)

	// Plain staff, not an author -> 403.
	staff := integration.CreateTestUser(testDB.DB, "pstaff", "Password123!", true)
	makeStaff(t, testDB.DB, staff.ID, true, false)
	staffToken := loginToken(t, "pstaff", "Password123!")

	// Superuser -> 200.
	su := integration.CreateTestUser(testDB.DB, "psuper", "Password123!", true)
	makeStaff(t, testDB.DB, su.ID, true, true)
	suToken := loginToken(t, "psuper", "Password123!")

	g := adminProblemRouter()

	resp := integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/problem/%s", prob.Code),
		Headers: map[string]string{"Authorization": "Bearer " + staffToken},
		Body:    map[string]interface{}{"is_public": true},
	})
	require.Equal(t, http.StatusForbidden, resp.Code)

	resp = integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/problem/%s", prob.Code),
		Headers: map[string]string{"Authorization": "Bearer " + suToken},
		Body:    map[string]interface{}{"is_public": true},
	})
	// Not 403: the guard admits the superuser. (Assert NotEqual rather than
	// ==200 to isolate the guard from the pre-existing update service.)
	require.NotEqual(t, http.StatusForbidden, resp.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd claoj && go test ./integration/ -run TestAdminProblemUpdate_Authz -v`
Expected: FAIL — the plain-staff request returns `200` (guard not wired yet), so the `403` assertion fails.

- [ ] **Step 3: Wire the guards in `router.go`**

In `claoj/api/router.go`, edit the Problems block (lines 132-155) so each write route gets its guard. Replace lines 134-155 with:

```go
			admin.POST("/admin/problems", auth.RequirePerm("judge.add_problem"), v2.AdminProblemCreate)
			admin.PATCH("/admin/problem/:code", auth.RequireProblemEdit(), v2.AdminProblemUpdate)
			admin.DELETE("/admin/problem/:code", auth.RequireProblemEdit(), v2.AdminProblemDelete)
			// Problem Clone
			admin.POST("/admin/problem/:code/clone", auth.RequirePerm("judge.clone_problem"), v2.AdminProblemClone)
			// Problem Clarifications
			admin.POST("/admin/problem/:code/clarification", auth.RequireProblemEdit(), v2.ProblemClarificationCreate)
			admin.DELETE("/admin/problem/clarification/:id", v2.ProblemClarificationDelete)
			// Problem Data
			admin.GET("/admin/problem/:code/data", auth.RequireProblemEdit(), adminHandlers.AdminProblemData)
			admin.POST("/admin/problem/:code/data", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataUpload)
			admin.DELETE("/admin/problem/:code/data/testcase/:id", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataDeleteTestCase)
			// Problem PDF
			admin.POST("/admin/problem/:code/pdf", auth.RequireProblemEdit(), adminHandlers.AdminProblemPdfUpload)
			admin.DELETE("/admin/problem/:code/pdf", auth.RequireProblemEdit(), adminHandlers.AdminProblemPdfDelete)
			// Problem Data - Reorder & File Operations
			admin.PATCH("/admin/problem/:code/data/reorder", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataReorder)
			admin.GET("/admin/problem/:code/data/files", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataFiles)
			admin.GET("/admin/problem/:code/data/file/*path", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataFileContent)
			admin.DELETE("/admin/problem/:code/data/file/*path", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataFileDelete)
			admin.GET("/admin/problem/:code/data/testcase/:id/content", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataTestCaseContent)
			admin.PATCH("/admin/problem/:code/data/testcase/:id", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataTestCaseUpdate)
```

Leave `GET /admin/problems` (list) and `GET /admin/problem/:code` (detail) unguarded — read access for any staff member is intentional. Leave `ProblemClarificationDelete` for Task 8. Verify `auth` is already imported in `router.go` (it is — the group uses `auth.AdminRequiredMiddleware()`).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd claoj && go test ./integration/ -run TestAdminProblemUpdate_Authz -v`
Expected: PASS (staff → 403, superuser → 200).

- [ ] **Step 5: Commit**

```bash
git add claoj/api/router.go claoj/integration/admin_problem_authz_test.go
git commit -m "feat(admin): enforce per-object delegation on problem write endpoints"
```

---

### Task 5: Wire contest guards into the router

**Files:**
- Modify: `claoj/api/router.go:110-129` (contest + contest-tag-on-contest routes)
- Test: `claoj/integration/admin_contest_authz_test.go` (create)

**Interfaces:**
- Consumes: `auth.RequireContestEdit`, `auth.RequirePerm`, `makeStaff`/`loginToken` helpers (defined in Task 4's test file, same `integration_test` package).
- Produces: contest write routes gated per-object; create requires `judge.add_contest`; clone requires `judge.clone_contest`.

- [ ] **Step 1: Write the failing test**

Create `claoj/integration/admin_contest_authz_test.go`:

```go
package integration_test

import (
	"fmt"
	"net/http"
	"testing"

	v2 "github.com/CLAOJ/claoj/api/v2"
	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/integration"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func adminContestRouter() *gin.Engine {
	g := integration.TestRouter()
	g.POST("/auth/login", authHandlers.Login)
	g.Use(auth.RequiredMiddleware())
	g.Use(auth.AdminRequiredMiddleware())
	g.PATCH("/admin/contest/:key", auth.RequireContestEdit(), v2.AdminContestUpdate)
	return g
}

func TestAdminContestUpdate_Authz(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	ct := models.Contest{Key: "cx", Name: "CX"}
	require.NoError(t, testDB.DB.Create(&ct).Error)

	staff := integration.CreateTestUser(testDB.DB, "cstaff", "Password123!", true)
	makeStaff(t, testDB.DB, staff.ID, true, false)
	staffToken := loginToken(t, "cstaff", "Password123!")

	su := integration.CreateTestUser(testDB.DB, "csuper", "Password123!", true)
	makeStaff(t, testDB.DB, su.ID, true, true)
	suToken := loginToken(t, "csuper", "Password123!")

	g := adminContestRouter()

	resp := integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/contest/%s", ct.Key),
		Headers: map[string]string{"Authorization": "Bearer " + staffToken},
		Body:    map[string]interface{}{"is_visible": true},
	})
	require.Equal(t, http.StatusForbidden, resp.Code)

	resp = integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/contest/%s", ct.Key),
		Headers: map[string]string{"Authorization": "Bearer " + suToken},
		Body:    map[string]interface{}{"is_visible": true},
	})
	// Not 403: the guard admits the superuser (isolates guard from the service).
	require.NotEqual(t, http.StatusForbidden, resp.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd claoj && go test ./integration/ -run TestAdminContestUpdate_Authz -v`
Expected: FAIL — plain staff returns `200`.

- [ ] **Step 3: Wire the guards in `router.go`**

Replace the Contests block (lines 113-120) and the two contest-tag-on-contest routes (lines 128-129):

```go
			admin.POST("/admin/contests", auth.RequirePerm("judge.add_contest"), v2.AdminContestCreate)
			admin.PATCH("/admin/contest/:key", auth.RequireContestEdit(), v2.AdminContestUpdate)
			admin.DELETE("/admin/contest/:key", auth.RequireContestEdit(), v2.AdminContestDelete)
			admin.POST("/admin/contest/:key/lock", auth.RequireContestEdit(), v2.AdminContestLock)
			admin.POST("/admin/contest/:key/clone", auth.RequirePerm("judge.clone_contest"), v2.AdminContestClone)
			// Contest Participation Disqualify
			admin.POST("/admin/contest/:key/participation/:id/disqualify", auth.RequireContestEdit(), v2.AdminContestParticipationDisqualify)
			admin.POST("/admin/contest/:key/participation/:id/undisqualify", auth.RequireContestEdit(), v2.AdminContestParticipationUndisqualify)
```

And lines 128-129:

```go
			admin.POST("/admin/contest/:key/tags/:tagId", auth.RequireContestEdit(), v2.AdminContestAddTag)
			admin.DELETE("/admin/contest/:key/tags/:tagId", auth.RequireContestEdit(), v2.AdminContestRemoveTag)
```

Leave `GET /admin/contests`, `GET /admin/contest/:key`, and the contest-tag *catalog* CRUD (`/admin/contest-tags`, `/admin/contest-tag/:id`) unchanged (out of scope).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd claoj && go test ./integration/ -run TestAdminContestUpdate_Authz -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add claoj/api/router.go claoj/integration/admin_contest_authz_test.go
git commit -m "feat(admin): enforce per-object delegation on contest write endpoints"
```

---

### Task 6: Wire organization guards into the router

**Files:**
- Modify: `claoj/api/router.go:174-177` (organization routes)
- Test: `claoj/integration/admin_org_authz_test.go` (create)

**Interfaces:**
- Consumes: `auth.RequireOrgEdit`, `auth.RequirePerm`, `makeStaff`/`loginToken`/`profileForUser` helpers.
- Produces: org update gated per-object; delete requires `judge.edit_all_organization`; create requires `judge.add_organization`.

- [ ] **Step 1: Write the failing test**

Create `claoj/integration/admin_org_authz_test.go`:

```go
package integration_test

import (
	"fmt"
	"net/http"
	"testing"

	v2 "github.com/CLAOJ/claoj/api/v2"
	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/integration"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func adminOrgUpdateRouter() *gin.Engine {
	g := integration.TestRouter()
	g.POST("/auth/login", authHandlers.Login)
	g.Use(auth.RequiredMiddleware())
	g.Use(auth.AdminRequiredMiddleware())
	g.PATCH("/admin/organization/:id", auth.RequireOrgEdit(), v2.AdminOrganizationUpdate)
	return g
}

func TestAdminOrgUpdate_Authz(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	org := models.Organization{Name: "OX", Slug: "ox", ShortName: "OX", IsOpen: true}
	require.NoError(t, testDB.DB.Create(&org).Error)

	// Staff who administers this org -> 200.
	orgAdmin := integration.CreateTestUser(testDB.DB, "oxadmin", "Password123!", true)
	makeStaff(t, testDB.DB, orgAdmin.ID, true, false)
	adminProfile := profileForUser(t, testDB.DB, orgAdmin.ID)
	require.NoError(t, testDB.DB.Model(&org).Association("Admins").Append(&adminProfile))
	adminToken := loginToken(t, "oxadmin", "Password123!")

	// Staff who does NOT administer this org -> 403.
	other := integration.CreateTestUser(testDB.DB, "oxother", "Password123!", true)
	makeStaff(t, testDB.DB, other.ID, true, false)
	otherToken := loginToken(t, "oxother", "Password123!")

	g := adminOrgUpdateRouter()

	resp := integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/organization/%d", org.ID),
		Headers: map[string]string{"Authorization": "Bearer " + otherToken},
		Body:    map[string]interface{}{"about": "x"},
	})
	require.Equal(t, http.StatusForbidden, resp.Code)

	resp = integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/organization/%d", org.ID),
		Headers: map[string]string{"Authorization": "Bearer " + adminToken},
		Body:    map[string]interface{}{"about": "x"},
	})
	// Not 403: the guard admits the org admin (isolates guard from the service).
	require.NotEqual(t, http.StatusForbidden, resp.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd claoj && go test ./integration/ -run TestAdminOrgUpdate_Authz -v`
Expected: FAIL — the non-admin staff returns `200`.

- [ ] **Step 3: Wire the guards in `router.go`**

Replace the Organizations block (lines 175-177):

```go
			admin.POST("/admin/organizations", auth.RequirePerm("judge.add_organization"), v2.AdminOrganizationCreate)
			admin.PATCH("/admin/organization/:id", auth.RequireOrgEdit(), v2.AdminOrganizationUpdate)
			admin.DELETE("/admin/organization/:id", auth.RequirePerm("judge.edit_all_organization"), v2.AdminOrganizationDelete)
```

(`DELETE` uses `RequirePerm("judge.edit_all_organization")` — superuser bypass is built in — so a mere org-admin cannot delete an org, per the approved judgment call. `GET /admin/organizations` list stays unguarded.)

- [ ] **Step 4: Run test to verify it passes**

Run: `cd claoj && go test ./integration/ -run TestAdminOrgUpdate_Authz -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add claoj/api/router.go claoj/integration/admin_org_authz_test.go
git commit -m "feat(admin): enforce delegation on organization write endpoints"
```

---

### Task 7: Org-admin appointment guard (inline)

**Files:**
- Modify: `claoj/api/v2/admin_users_roles.go` (`AdminUserUpdate`, before the `getUserService().UpdateUser` call ~line 200)
- Test: `claoj/integration/admin_orgadmin_appoint_test.go` (create)

**Interfaces:**
- Consumes: `auth.CanEditOrganization` (Task 2), the existing `input.AddOrganizationAdmin` / `input.RemoveOrganizationAdmin` fields (`[]uint`).
- Produces: appointing/removing an org admin requires `CanEditOrganization` for each targeted org id.

- [ ] **Step 1: Write the failing test**

Create `claoj/integration/admin_orgadmin_appoint_test.go`:

```go
package integration_test

import (
	"fmt"
	"net/http"
	"testing"

	v2 "github.com/CLAOJ/claoj/api/v2"
	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/integration"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func adminUserUpdateRouter() *gin.Engine {
	g := integration.TestRouter()
	g.POST("/auth/login", authHandlers.Login)
	g.Use(auth.RequiredMiddleware())
	g.Use(auth.AdminRequiredMiddleware())
	g.PATCH("/admin/user/:id", v2.AdminUserUpdate)
	return g
}

func TestAppointOrgAdmin_RequiresOrgAuthority(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	org := models.Organization{Name: "OA", Slug: "oa", ShortName: "OA"}
	require.NoError(t, testDB.DB.Create(&org).Error)

	// Target whose profile we will (attempt to) appoint as org admin.
	target := integration.CreateTestUser(testDB.DB, "target", "Password123!", true)
	targetProfile := profileForUser(t, testDB.DB, target.ID)

	// Staff who does NOT administer org -> appointment forbidden.
	staff := integration.CreateTestUser(testDB.DB, "plainstaff", "Password123!", true)
	makeStaff(t, testDB.DB, staff.ID, true, false)
	staffToken := loginToken(t, "plainstaff", "Password123!")

	g := adminUserUpdateRouter()
	resp := integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "PATCH",
		Path:    fmt.Sprintf("/admin/user/%d", targetProfile.ID),
		Headers: map[string]string{"Authorization": "Bearer " + staffToken},
		Body:    map[string]interface{}{"add_organization_admin": []uint{org.ID}},
	})
	require.Equal(t, http.StatusForbidden, resp.Code)

	// The appointment must NOT have happened.
	var count int64
	testDB.DB.Table("judge_organization_admins").
		Where("organization_id = ? AND profile_id = ?", org.ID, targetProfile.ID).Count(&count)
	require.Equal(t, int64(0), count)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd claoj && go test ./integration/ -run TestAppointOrgAdmin_RequiresOrgAuthority -v`
Expected: FAIL — currently returns `200` and the row is created.

- [ ] **Step 3: Implement the inline guard**

In `claoj/api/v2/admin_users_roles.go`, immediately after the input is bound (after the `ShouldBindJSON(&input)` block, before the is_staff/is_superuser block at ~line 165), insert:

```go
	// v1 parity: appointing or removing an organization administrator requires
	// authority over that organization (org admin, edit_all_organization, or
	// superuser). Checked per targeted org id.
	orgAdminTargets := append(append([]uint{}, input.AddOrganizationAdmin...), input.RemoveOrganizationAdmin...)
	for _, orgID := range orgAdminTargets {
		if !auth.CanEditOrganization(c, orgID) {
			c.JSON(http.StatusForbidden, apiError("you do not have permission to manage administrators of organization "+strconv.FormatUint(uint64(orgID), 10)))
			return
		}
	}
```

(`auth`, `http`, and `strconv` are already imported in this file.)

- [ ] **Step 4: Run test to verify it passes**

Run: `cd claoj && go test ./integration/ -run TestAppointOrgAdmin_RequiresOrgAuthority -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add claoj/api/v2/admin_users_roles.go claoj/integration/admin_orgadmin_appoint_test.go
git commit -m "feat(admin): require org authority to appoint/remove org admins"
```

---

### Task 8: Clarification-delete guard (inline)

**Files:**
- Modify: `claoj/api/v2/problem_clarification.go` (`ProblemClarificationDelete`, lines 109-128)
- Test: `claoj/integration/admin_clarification_authz_test.go` (create)

**Interfaces:**
- Consumes: `auth.CanEditProblem`, `models.ProblemClarification{ProblemID}`, `models.Problem`.
- Produces: deleting a clarification requires edit authority over its parent problem.

- [ ] **Step 1: Write the failing test**

Create `claoj/integration/admin_clarification_authz_test.go`:

```go
package integration_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	v2 "github.com/CLAOJ/claoj/api/v2"
	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/integration"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func adminClarificationDeleteRouter() *gin.Engine {
	g := integration.TestRouter()
	g.POST("/auth/login", authHandlers.Login)
	g.Use(auth.RequiredMiddleware())
	g.Use(auth.AdminRequiredMiddleware())
	g.DELETE("/admin/problem/clarification/:id", v2.ProblemClarificationDelete)
	return g
}

func TestClarificationDelete_Authz(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)
	// ProblemClarification is not in the harness AutoMigrate list — add it here.
	require.NoError(t, testDB.DB.AutoMigrate(&models.ProblemClarification{}))

	prob := models.Problem{Code: "pc", Name: "PC", IsPublic: true}
	require.NoError(t, testDB.DB.Create(&prob).Error)
	clar := models.ProblemClarification{ProblemID: prob.ID, Description: "d", Date: time.Now()}
	require.NoError(t, testDB.DB.Create(&clar).Error)

	staff := integration.CreateTestUser(testDB.DB, "clstaff", "Password123!", true)
	makeStaff(t, testDB.DB, staff.ID, true, false)
	staffToken := loginToken(t, "clstaff", "Password123!")

	g := adminClarificationDeleteRouter()
	resp := integration.MakeRequest(t, g, integration.HTTPRequest{
		Method:  "DELETE",
		Path:    fmt.Sprintf("/admin/problem/clarification/%d", clar.ID),
		Headers: map[string]string{"Authorization": "Bearer " + staffToken},
	})
	require.Equal(t, http.StatusForbidden, resp.Code)

	var count int64
	testDB.DB.Model(&models.ProblemClarification{}).Where("id = ?", clar.ID).Count(&count)
	require.Equal(t, int64(1), count) // not deleted
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd claoj && go test ./integration/ -run TestClarificationDelete_Authz -v`
Expected: FAIL — currently returns `200` and deletes the row.

- [ ] **Step 3: Implement the inline guard**

In `claoj/api/v2/problem_clarification.go`, in `ProblemClarificationDelete`, after the clarification is loaded (after line 120's error block, before the `db.DB.Delete`), insert:

```go
	// v1 parity: only editors of the parent problem may delete its clarifications.
	var problem models.Problem
	if err := db.DB.Preload("Authors").Preload("Curators").
		First(&problem, clarification.ProblemID).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("problem not found"))
		return
	}
	if !auth.CanEditProblem(c, &problem) {
		c.JSON(http.StatusForbidden, apiError("you do not have permission to edit this problem"))
		return
	}
```

(`auth`, `db`, `models`, `http` are already imported in this file.)

- [ ] **Step 4: Run test to verify it passes**

Run: `cd claoj && go test ./integration/ -run TestClarificationDelete_Authz -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add claoj/api/v2/problem_clarification.go claoj/integration/admin_clarification_authz_test.go
git commit -m "feat(admin): require problem-edit authority to delete clarifications"
```

---

### Task 9: Full-suite check + live end-to-end re-verify

**Files:**
- No source changes. Verification only.

**Interfaces:**
- Consumes: everything above; the running stack at `127.0.0.1:8090`; throwaway accounts `deleg_super` (5001), `deleg_staff` (5002), `deleg_superonly` (5003), password `DelegTest!2026`.

- [ ] **Step 1: Run the whole backend test suite**

Run: `cd claoj && go build ./... && go test ./auth/... ./integration/... -v`
Expected: all PASS. Investigate and fix any failure before proceeding.

- [ ] **Step 2: Rebuild the v2 backend container and confirm health**

```bash
cd "F:/Coding/CLAOJ/CLAOJ/claoj-docker/claoj"
MSYS_NO_PATHCONV=1 docker compose -f docker-compose.local.yml -f docker-compose.v2.yml -p claoj up -d --build v2_backend
docker ps --filter name=claoj_v2_backend --format '{{.Status}}'
```
Expected: container `Up ... (healthy)`.

- [ ] **Step 3: Re-run the live gap probes (should now be closed)**

```bash
API=http://127.0.0.1:8090/api
cd "$(mktemp -d)"
for a in deleg_super deleg_staff deleg_superonly; do
  curl -s -c "cj_$a.txt" -o /dev/null -X POST "$API/auth/login" -H 'Content-Type: application/json' -d "{\"username\":\"$a\",\"password\":\"DelegTest!2026\"}"
done
echo -n "staff PATCH non-owned problem (expect 403): "
curl -s -o /dev/null -w '%{http_code}\n' -b cj_deleg_staff.txt -X PATCH "$API/admin/problem/01_02" -H 'Content-Type: application/json' -d '{"is_public":true}'
echo -n "superuser-not-staff GET problems (expect 200): "
curl -s -o /dev/null -w '%{http_code}\n' -b cj_deleg_superonly.txt "$API/admin/problems?limit=1"
echo -n "superuser PATCH problem (expect 200): "
curl -s -o /dev/null -w '%{http_code}\n' -b cj_deleg_super.txt -X PATCH "$API/admin/problem/01_02" -H 'Content-Type: application/json' -d '{"is_public":true}'
```
Expected: `403`, `200`, `200`.

- [ ] **Step 4: Confirm the delegation positive path (author can edit)**

Make `deleg_staff` an author of `01_02`, then re-run its PATCH — expect `200`:

```bash
docker exec -i claoj_site python manage.py shell <<'PY'
from judge.models import Problem, Profile
from django.contrib.auth.models import User
p = Problem.objects.get(code='01_02')
prof = Profile.objects.get(user__username='deleg_staff')
p.authors.add(prof)
print('authors now:', list(p.authors.values_list('user__username', flat=True)))
PY
# perms cache TTL is 60s; author membership is read live, but CanEditProblem also
# needs edit_own_problem. Grant it to deleg_staff for a true positive check:
docker exec -i claoj_site python manage.py shell <<'PY'
from django.contrib.auth.models import User, Permission
u = User.objects.get(username='deleg_staff')
u.user_permissions.add(Permission.objects.get(codename='edit_own_problem'))
print('has edit_own_problem:', u.has_perm('judge.edit_own_problem'))
PY
```
The v2 perm cache has a 60s TTL and is NOT invalidated by Django-side permission grants, so wait ~65s (or clear it: `docker exec claoj_redis redis-cli --scan --pattern 'perm:*' | while read k; do docker exec claoj_redis redis-cli DEL "$k"; done`), then re-login `deleg_staff` for a fresh token and PATCH `01_02` → expect `200`.

- [ ] **Step 5: Clean up throwaway test accounts and revert the probe’s side effects**

```bash
docker exec -i claoj_site python manage.py shell <<'PY'
from django.contrib.auth.models import User
from judge.models import Problem, Profile
# remove deleg_staff from 01_02 authors (undo Step 4)
p = Problem.objects.get(code='01_02')
p.authors.remove(*Profile.objects.filter(user__username='deleg_staff'))
User.objects.filter(username__in=['deleg_super','deleg_staff','deleg_superonly']).delete()
print('cleanup done')
PY
```
Expected: `cleanup done`. (Deleting the users cascades their judge_profile rows.)

- [ ] **Step 6: Final commit (docs/ledger only, if any)** — no source changes in this task; nothing to commit unless notes were added.

---

## Notes for the executor

- Tasks 1-3 are pure `auth`-package units (fast, DB-backed via `setupPermsDB`). Tasks 4-8 are `integration_test` and require a reachable MariaDB (the running `claoj_db`); `integration.SetupIntegrationDB(t)` provisions/cleans the schema exactly as the existing `organization_flow_test.go` does.
- The helpers `makeStaff`, `loginToken`, `profileForUser` are defined once in the `integration_test` package (Task 4 adds `makeStaff`; `loginToken`/`profileForUser` already exist in `organization_flow_test.go`). Do not redefine them.
- Do NOT stage the 6 unrelated pre-existing dirty files (`contest_ranking.go`, `submission.go`, `contest_format/*.go`, `SubmissionSource.tsx`). Stage only the files named in each task.
