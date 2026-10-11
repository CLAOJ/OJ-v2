# Admin CRUD Gaps Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the create/delete holes in the v2 admin API (judge, user, submission, ticket, clarifications) and stop raw `:id` path strings from reaching GORM as SQL.

**Architecture:** New gin handlers in `claoj/api/v2` next to the existing admin handlers, one new service method for submission deletion, routes registered in `claoj/api/router.go`. Deletes are hard deletes that remove the same child rows Django's ORM cascade would, inside one transaction. The admin frontend gets matching API client methods and buttons.

**Tech Stack:** Go 1.25, gin, GORM (MySQL in production, in-memory sqlite in tests), testify. Frontend: Next.js 16, React 19, TanStack Query, next-intl, jest.

**Spec:** `docs/superpowers/specs/2026-10-09-admin-panel-parity-design.md` (its Constraints section binds this plan). This plan covers CRUD holes that spec does not; the spec's sub-projects B and C (edit-form field parity) and its Deferred list stay out of scope.

## Global Constraints

- v1 (Django) and v2 share one MySQL database that Django owns. No schema changes: no new tables, no new columns, no `AutoMigrate` outside test files.
- v2 writes existing tables the way Django does (same join tables, same column semantics).
- Go module is `github.com/CLAOJ/claoj`. Run every Go command from the `claoj/` directory.
- Messages the client may see go through `apiUserError(msg)`. Internal errors go through `apiError(msg)`, which logs the message and returns a generic one. Never pass a raw `err.Error()` to `apiUserError`, except the fixed validation errors named in a task.
- A string that came from the request must never be passed to GORM as an inline condition (`db.First(&x, s)`): parse it to an unsigned integer first.
- Tests use testify `require`, an in-memory sqlite database assigned to `db.DB`, and restore the previous `db.DB` in `t.Cleanup`.
- No new dependencies (Go or npm).
- The existing soft "delete" endpoints for problem, contest, user and organization keep their current behaviour. Do not touch them.
- Out of scope: hard delete of problem / contest / organization / user, edit-form field parity, contest announcements, problem translations.
- Commits: conventional-commit subject. For a multi-line message use `git commit -F - <<'EOF' … EOF` (Git Bash heredoc), and end the message with the line `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Review Focus

1. A non-numeric `:id` (for example `1 OR 1=1`) must give 400 before any query runs — Task 1 tests.
2. Deleting something still in use (an online judge, a submission being judged) must give 409 and delete nothing — Task 2 and Task 4 tests.
3. A staff user who is not a superuser must not be able to set the password of a staff or superuser account — Task 3 test.
4. A duplicate username, email or judge name must give 409 and leave no half-created rows — Task 2 and Task 3 tests.
5. A delete must remove the object's child rows and nothing that belongs to a sibling object — Task 2, Task 4 and Task 5 tests.

---

### Task 1: Validate `:id` path params before GORM inline lookups

GORM treats a non-numeric string passed as an inline condition as raw SQL: `db.First(&org, "1 OR 1=1")` runs `WHERE 1 OR 1=1`. Several handlers pass `c.Param("id")` straight in, including public ones.

**Files:**
- Modify: `claoj/api/v2/helpers.go` (add `numericParam`)
- Modify: `claoj/api/v2/blog.go`, `claoj/api/v2/organization.go`, `claoj/api/v2/ticket.go`, `claoj/api/v2/ticket_reply.go`, `claoj/api/v2/problem_clarification.go`, `claoj/api/v2/admin_judge_organization.go`
- Test: `claoj/api/v2/id_param_test.go` (create)

**Interfaces:**
- Produces: `func numericParam(c *gin.Context, name string) (string, bool)` in package `v2`. Later tasks may use it.

- [ ] **Step 1: Write the failing test**

Create `claoj/api/v2/id_param_test.go`:

```go
package v2

import (
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupIDParamDB(t *testing.T) models.AuthUser {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.BlogPost{}, &models.Judge{}, &models.Ticket{}, &models.TicketMessage{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.ProblemClarification{},
	))
	db.DB = database

	u := models.AuthUser{Username: "alice", IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Profile{UserID: u.ID, Timezone: "UTC"}).Error)
	return u
}

// A non-numeric :id must be rejected before it can reach GORM, which would
// otherwise run it as a raw SQL condition.
func TestInlineIDLookups_RejectNonNumericID(t *testing.T) {
	u := setupIDParamDB(t)

	const evil = "1%20OR%201=1"
	cases := []struct {
		name    string
		method  string
		route   string
		target  string
		handler gin.HandlerFunc
	}{
		{"organization detail", http.MethodGet, "/organization/:id", "/organization/" + evil, OrganizationDetail},
		{"organization update", http.MethodPatch, "/organization/:id", "/organization/" + evil, UpdateOrganization},
		{"organization join", http.MethodPost, "/organization/:id/join", "/organization/" + evil + "/join", JoinOrganization},
		{"organization leave", http.MethodPost, "/organization/:id/leave", "/organization/" + evil + "/leave", LeaveOrganization},
		{"organization request", http.MethodPost, "/organization/:id/request", "/organization/" + evil + "/request", RequestJoinOrganization},
		{"organization kick", http.MethodPost, "/organization/:id/kick", "/organization/" + evil + "/kick", KickUser},
		{"blog detail", http.MethodGet, "/blog/:id", "/blog/" + evil, BlogDetail},
		{"blog vote", http.MethodPost, "/blog/:id/vote", "/blog/" + evil + "/vote", BlogVoteHandler},
		{"ticket detail", http.MethodGet, "/ticket/:id", "/ticket/" + evil, TicketDetail},
		{"ticket reply", http.MethodPost, "/ticket/:id/message", "/ticket/" + evil + "/message", TicketReply},
		{"admin ticket detail", http.MethodGet, "/admin/ticket/:id", "/admin/ticket/" + evil, AdminTicketDetail},
		{"judge block", http.MethodPost, "/admin/judge/:id/block", "/admin/judge/" + evil + "/block", AdminJudgeBlock},
		{"judge unblock", http.MethodPost, "/admin/judge/:id/unblock", "/admin/judge/" + evil + "/unblock", AdminJudgeUnblock},
		{"judge enable", http.MethodPost, "/admin/judge/:id/enable", "/admin/judge/" + evil + "/enable", AdminJudgeEnable},
		{"judge disable", http.MethodPost, "/admin/judge/:id/disable", "/admin/judge/" + evil + "/disable", AdminJudgeDisable},
		{"clarification delete", http.MethodDelete, "/admin/problem/clarification/:id", "/admin/problem/clarification/" + evil, ProblemClarificationDelete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := authedRouter(u.ID, tc.method, tc.route, tc.handler)
			body := ""
			if tc.method != http.MethodGet && tc.method != http.MethodDelete {
				body = `{}`
			}
			w := doRequest(router, tc.method, tc.target, body)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			require.Contains(t, w.Body.String(), "invalid id")
		})
	}
}

// A numeric id keeps working.
func TestAdminJudgeBlock_NumericIDStillWorks(t *testing.T) {
	u := setupIDParamDB(t)
	j := models.Judge{Name: "j1", AuthKey: "k", Created: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&j).Error)

	router := authedRouter(u.ID, http.MethodPost, "/admin/judge/:id/block", AdminJudgeBlock)
	w := doRequest(router, http.MethodPost, "/admin/judge/1/block", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.Judge
	require.NoError(t, db.DB.First(&got, j.ID).Error)
	require.True(t, got.IsBlocked)
}
```

`authedRouter` and `doRequest` already exist in `claoj/api/v2/admin_groups_test.go`; reuse them, do not redefine them.

- [ ] **Step 2: Run the test to verify it fails**

Run (from `claoj/`): `go test ./api/v2/ -run 'TestInlineIDLookups|TestAdminJudgeBlock_NumericID' -v`
Expected: `TestInlineIDLookups_RejectNonNumericID` sub-tests FAIL (status is not 400, or the body lacks `invalid id`). `TestAdminJudgeBlock_NumericIDStillWorks` passes already.

- [ ] **Step 3: Add the helper**

Append to `claoj/api/v2/helpers.go`, directly after the existing `parseUint` function:

```go
// numericParam returns the named path param when it is a base-10 unsigned
// integer; otherwise it writes a 400 and returns ok=false.
//
// GORM treats a non-numeric string passed as an inline condition
// (db.First(&x, s)) as raw SQL, so every :id that reaches such a call as a
// string must pass through here first.
func numericParam(c *gin.Context, name string) (string, bool) {
	s := c.Param(name)
	if _, err := strconv.ParseUint(s, 10, 64); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid "+name))
		return "", false
	}
	return s, true
}
```

- [ ] **Step 4: Use it at every raw-string site**

In each handler below, replace the line that reads the param (`id := c.Param("id")` or `ticketID := c.Param("id")`) with the validated form, keeping the variable name the handler already uses, and make it the **first statement of the handler** so nothing runs before the check:

```go
	id, ok := numericParam(c, "id")
	if !ok {
		return
	}
```

(for the ticket handlers the variable is `ticketID`). If the handler later declares `ok` with `:=` and the compiler reports "no new variables on left side of :=", change that later `:=` to `=`.

Sites, by handler:

| File | Handlers |
|---|---|
| `api/v2/organization.go` | `OrganizationDetail`, `UpdateOrganization`, `JoinOrganization`, `LeaveOrganization`, `RequestJoinOrganization`, `KickUser` |
| `api/v2/blog.go` | `BlogDetail`, `BlogVoteHandler` |
| `api/v2/ticket.go` | `TicketDetail`, `AdminTicketDetail` |
| `api/v2/ticket_reply.go` | `TicketReply` |
| `api/v2/problem_clarification.go` | `ProblemClarificationDelete` |
| `api/v2/admin_judge_organization.go` | `AdminJudgeBlock`, `AdminJudgeUnblock`, `AdminJudgeEnable`, `AdminJudgeDisable` |

Then sweep for sites this table missed. From `claoj/` run:

```bash
grep -rnE "\.(First|Take|Last|Find|Delete)\(&[A-Za-z_.]+, [A-Za-z_]+\)" --include=*.go api | grep -v _test.go
```

For every hit, look at how the second argument is declared. If it is a `string` that came from `c.Param` or `c.Query`, validate it the same way (in packages other than `v2`, inline the `strconv.ParseUint` check and return 400). If it is already a `uint`/`uint64`/`int` produced by a parse call, leave it. List every site you changed, and every hit you inspected and left alone, in your report.

- [ ] **Step 5: Run the tests**

Run: `go test ./api/v2/ -run 'TestInlineIDLookups|TestAdminJudgeBlock_NumericID' -v`
Expected: PASS.

Run: `go build ./... && go test ./api/... ./service/... ./auth/...`
Expected: build succeeds, all packages `ok`.

- [ ] **Step 6: Commit**

```bash
git add claoj/api
git commit -F - <<'EOF'
fix(api): reject non-numeric :id before it reaches GORM

GORM runs a non-numeric string passed as an inline condition as raw SQL,
so db.First(&x, c.Param("id")) was an injection path on public routes.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 2: Judge create, delete, rename and re-key

Judges can only be created by a manual SQL insert today. A judge agent authenticates with HTTP basic auth: judge name + `auth_key` (`claoj/bridge/http.go`).

**Files:**
- Create: `claoj/api/v2/admin_judge_crud.go`
- Modify: `claoj/api/v2/admin_judge_organization.go` (`AdminJudgeUpdate`)
- Modify: `claoj/api/router.go` (two routes)
- Test: `claoj/api/v2/admin_judge_crud_test.go` (create)

**Interfaces:**
- Produces: `POST /api/admin/judges` body `{"name": string, "auth_key"?: string, "description"?: string}` → 201 `{"id": number, "name": string, "auth_key": string}`.
- Produces: `DELETE /api/admin/judge/:id` → 200 `{"success": true}`; 409 `{"error": "judge is online; block it and wait for it to disconnect first"}`.
- Produces: `PATCH /api/admin/judge/:id` additionally accepts `"name"` and `"auth_key"`.

- [ ] **Step 1: Write the failing tests**

Create `claoj/api/v2/admin_judge_crud_test.go`:

```go
package v2

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupJudgeCrudDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{},
		&models.Judge{}, &models.RuntimeVersion{}, &models.Submission{},
	))
	db.DB = database
}

func seedJudge(t *testing.T, name string, online bool) models.Judge {
	t.Helper()
	j := models.Judge{Name: name, AuthKey: "key-" + name, Created: time.Now(), Online: online}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&j).Error)
	return j
}

func TestAdminJudgeCreate_GeneratesKey(t *testing.T) {
	setupJudgeCrudDB(t)
	router := authedRouter(1, http.MethodPost, "/admin/judges", AdminJudgeCreate)
	w := doRequest(router, http.MethodPost, "/admin/judges", `{"name":"judge1","description":"rack 3"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "judge1", body["name"])
	key, _ := body["auth_key"].(string)
	require.Len(t, key, 64)

	var j models.Judge
	require.NoError(t, db.DB.Where("name = ?", "judge1").First(&j).Error)
	require.Equal(t, key, j.AuthKey)
	require.Equal(t, "rack 3", j.Description)
	require.False(t, j.Created.IsZero())
	require.False(t, j.Online)
	require.Equal(t, float64(j.ID), body["id"])
}

func TestAdminJudgeCreate_KeepsSuppliedKey(t *testing.T) {
	setupJudgeCrudDB(t)
	router := authedRouter(1, http.MethodPost, "/admin/judges", AdminJudgeCreate)
	w := doRequest(router, http.MethodPost, "/admin/judges", `{"name":"judge2","auth_key":"my-secret-key"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var j models.Judge
	require.NoError(t, db.DB.Where("name = ?", "judge2").First(&j).Error)
	require.Equal(t, "my-secret-key", j.AuthKey)
}

func TestAdminJudgeCreate_Rejects(t *testing.T) {
	setupJudgeCrudDB(t)
	seedJudge(t, "taken", false)
	router := authedRouter(1, http.MethodPost, "/admin/judges", AdminJudgeCreate)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"missing name", `{}`, http.StatusBadRequest},
		{"blank name", `{"name":"   "}`, http.StatusBadRequest},
		{"name too long", `{"name":"` + strings.Repeat("a", 51) + `"}`, http.StatusBadRequest},
		{"key too long", `{"name":"ok","auth_key":"` + strings.Repeat("k", 101) + `"}`, http.StatusBadRequest},
		{"duplicate name", `{"name":"taken"}`, http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(router, http.MethodPost, "/admin/judges", tc.body)
			require.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}

	var n int64
	require.NoError(t, db.DB.Model(&models.Judge{}).Count(&n).Error)
	require.Equal(t, int64(1), n, "a rejected create must not insert a row")
}

func TestAdminJudgeUpdate_RenameAndRekey(t *testing.T) {
	setupJudgeCrudDB(t)
	a := seedJudge(t, "a", false)
	seedJudge(t, "b", false)
	router := authedRouter(1, http.MethodPatch, "/admin/judge/:id", AdminJudgeUpdate)
	target := "/admin/judge/1"
	require.Equal(t, uint(1), a.ID)

	w := doRequest(router, http.MethodPatch, target, `{"name":"a2","auth_key":"newkey"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var got models.Judge
	require.NoError(t, db.DB.First(&got, a.ID).Error)
	require.Equal(t, "a2", got.Name)
	require.Equal(t, "newkey", got.AuthKey)

	w = doRequest(router, http.MethodPatch, target, `{"name":"b"}`)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	w = doRequest(router, http.MethodPatch, target, `{"auth_key":""}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	w = doRequest(router, http.MethodPatch, target, `{"description":"only this"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, db.DB.First(&got, a.ID).Error)
	require.Equal(t, "a2", got.Name, "omitted fields stay unchanged")
	require.Equal(t, "newkey", got.AuthKey, "omitted fields stay unchanged")
	require.Equal(t, "only this", got.Description)
}

func TestAdminJudgeDelete_CascadesLikeDjango(t *testing.T) {
	setupJudgeCrudDB(t)
	j := seedJudge(t, "doomed", false)
	keep := seedJudge(t, "keep", false)

	lang := models.Language{Key: "py3", Name: "Python 3"}
	require.NoError(t, db.DB.Create(&lang).Error)
	p := models.Problem{Code: "p1", Name: "P1"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	for _, judge := range []models.Judge{j, keep} {
		judge := judge
		require.NoError(t, db.DB.Model(&judge).Association("Problems").Append(&p))
		require.NoError(t, db.DB.Model(&judge).Association("Runtimes").Append(&lang))
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.RuntimeVersion{
			LanguageID: lang.ID, JudgeID: judge.ID, Name: "python", Version: "3.12",
		}).Error)
	}
	sub := models.Submission{UserID: 1, ProblemID: p.ID, LanguageID: lang.ID, Date: time.Now(), Status: "D", JudgedOnID: &j.ID}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&sub).Error)

	router := authedRouter(1, http.MethodDelete, "/admin/judge/:id", AdminJudgeDelete)
	w := doRequest(router, http.MethodDelete, "/admin/judge/1", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	count := func(table, where string, arg any) int64 {
		var n int64
		require.NoError(t, db.DB.Table(table).Where(where, arg).Count(&n).Error)
		return n
	}
	require.Equal(t, int64(0), count("judge_judge", "id = ?", j.ID))
	require.Equal(t, int64(0), count("judge_runtimeversion", "judge_id = ?", j.ID))
	require.Equal(t, int64(0), count("judge_judge_problems", "judge_id = ?", j.ID))
	require.Equal(t, int64(0), count("judge_judge_runtimes", "judge_id = ?", j.ID))

	// The other judge keeps everything.
	require.Equal(t, int64(1), count("judge_judge", "id = ?", keep.ID))
	require.Equal(t, int64(1), count("judge_runtimeversion", "judge_id = ?", keep.ID))
	require.Equal(t, int64(1), count("judge_judge_problems", "judge_id = ?", keep.ID))
	require.Equal(t, int64(1), count("judge_judge_runtimes", "judge_id = ?", keep.ID))

	// Django: Submission.judged_on is SET_NULL, the submission survives.
	var gotSub models.Submission
	require.NoError(t, db.DB.First(&gotSub, sub.ID).Error)
	require.Nil(t, gotSub.JudgedOnID)
}

func TestAdminJudgeDelete_OnlineIs409(t *testing.T) {
	setupJudgeCrudDB(t)
	j := seedJudge(t, "busy", true)
	router := authedRouter(1, http.MethodDelete, "/admin/judge/:id", AdminJudgeDelete)
	w := doRequest(router, http.MethodDelete, "/admin/judge/1", "")
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	var n int64
	require.NoError(t, db.DB.Model(&models.Judge{}).Where("id = ?", j.ID).Count(&n).Error)
	require.Equal(t, int64(1), n)
}

func TestAdminJudgeDelete_Errors(t *testing.T) {
	setupJudgeCrudDB(t)
	router := authedRouter(1, http.MethodDelete, "/admin/judge/:id", AdminJudgeDelete)
	require.Equal(t, http.StatusNotFound, doRequest(router, http.MethodDelete, "/admin/judge/99", "").Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodDelete, "/admin/judge/abc", "").Code)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./api/v2/ -run 'TestAdminJudge(Create|Update|Delete)' -v`
Expected: compile error — `AdminJudgeCreate` and `AdminJudgeDelete` are undefined.

- [ ] **Step 3: Implement create and delete**

Create `claoj/api/v2/admin_judge_crud.go`:

```go
package v2

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	judgeNameMaxLen    = 50  // judge_judge.name varchar(50)
	judgeAuthKeyMaxLen = 100 // judge_judge.auth_key varchar(100)
)

// generateJudgeAuthKey returns a 64-character URL-safe random key.
func generateJudgeAuthKey() (string, error) {
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// judgeNameTaken reports whether another judge already uses name.
func judgeNameTaken(name string, exceptID uint) (bool, error) {
	var count int64
	err := db.DB.Model(&models.Judge{}).Where("name = ? AND id <> ?", name, exceptID).Count(&count).Error
	return count > 0, err
}

// AdminJudgeCreate - POST /api/v2/admin/judges
// Registers a judge. The agent authenticates with name + auth_key; when no
// key is supplied one is generated and returned in the response.
func AdminJudgeCreate(c *gin.Context) {
	var input struct {
		Name        string `json:"name"`
		AuthKey     string `json:"auth_key"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > judgeNameMaxLen {
		c.JSON(http.StatusBadRequest, apiUserError("name is required and must be at most 50 characters"))
		return
	}
	if len(input.AuthKey) > judgeAuthKeyMaxLen {
		c.JSON(http.StatusBadRequest, apiUserError("auth_key must be at most 100 characters"))
		return
	}

	taken, err := judgeNameTaken(name, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if taken {
		c.JSON(http.StatusConflict, apiUserError("a judge with this name already exists"))
		return
	}

	key := input.AuthKey
	if key == "" {
		if key, err = generateJudgeAuthKey(); err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}

	judge := models.Judge{
		Name:        name,
		Created:     time.Now(),
		AuthKey:     key,
		Description: input.Description,
	}
	if err := db.DB.Create(&judge).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       judge.ID,
		"name":     judge.Name,
		"auth_key": judge.AuthKey,
	})
}

// AdminJudgeDelete - DELETE /api/v2/admin/judge/:id
// Hard-deletes a judge the way Django's cascade does: runtime versions and
// the problem/runtime join rows go with it, and submissions it judged keep
// existing with judged_on cleared (Submission.judged_on is SET_NULL).
func AdminJudgeDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid id"))
		return
	}

	var judge models.Judge
	if err := db.DB.First(&judge, id).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("judge not found"))
		return
	}
	// Deleting a connected judge would strand whatever it is grading: its
	// next event fails authentication and the submission never finishes.
	if judge.Online {
		c.JSON(http.StatusConflict, apiUserError("judge is online; block it and wait for it to disconnect first"))
		return
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Submission{}).Where("judged_on_id = ?", judge.ID).
			Update("judged_on_id", gorm.Expr("NULL")).Error; err != nil {
			return err
		}
		if err := tx.Where("judge_id = ?", judge.ID).Delete(&models.RuntimeVersion{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&judge).Association("Problems").Clear(); err != nil {
			return err
		}
		if err := tx.Model(&judge).Association("Runtimes").Clear(); err != nil {
			return err
		}
		return tx.Delete(&judge).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
```

- [ ] **Step 4: Extend `AdminJudgeUpdate`**

In `claoj/api/v2/admin_judge_organization.go`, inside `AdminJudgeUpdate`, replace the `input` struct and the `updates` block (from `var input struct {` down to and including the `if len(updates) > 0 { … }` block) with:

```go
	var input struct {
		Name        *string `json:"name"`
		AuthKey     *string `json:"auth_key"`
		Description *string `json:"description"`
		ProblemIDs  []uint  `json:"problem_ids"`
		RuntimeIDs  []uint  `json:"runtime_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	updates := make(map[string]interface{})
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > judgeNameMaxLen {
			c.JSON(http.StatusBadRequest, apiUserError("name is required and must be at most 50 characters"))
			return
		}
		taken, err := judgeNameTaken(name, judge.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
		if taken {
			c.JSON(http.StatusConflict, apiUserError("a judge with this name already exists"))
			return
		}
		updates["name"] = name
	}
	if input.AuthKey != nil {
		if *input.AuthKey == "" || len(*input.AuthKey) > judgeAuthKeyMaxLen {
			c.JSON(http.StatusBadRequest, apiUserError("auth_key must be 1 to 100 characters"))
			return
		}
		updates["auth_key"] = *input.AuthKey
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&judge).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
			return
		}
	}
```

Leave the rest of the function (problem and runtime assignment, the final response) as it is.

- [ ] **Step 5: Register the routes**

In `claoj/api/router.go`, directly after the line `admin.PATCH("/admin/judge/:id", v2.AdminJudgeUpdate)`, add:

```go
			admin.POST("/admin/judges", auth.RequirePerm("judge.add_judge"), v2.AdminJudgeCreate)
			admin.DELETE("/admin/judge/:id", auth.RequirePerm("judge.delete_judge"), v2.AdminJudgeDelete)
```

- [ ] **Step 6: Run the tests**

Run: `go test ./api/v2/ -run 'TestAdminJudge' -v`
Expected: PASS.

Run: `go build ./... && go test ./api/...`
Expected: build succeeds, all packages `ok`.

- [ ] **Step 7: Commit**

```bash
git add claoj/api
git commit -F - <<'EOF'
feat(admin): create, delete, rename and re-key judges

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 3: Admin user create and set-password

Staff cannot create an account or reset a password from v2 today. `Register` (`claoj/api/v2/register.go`) already knows how to insert `auth_user` + `judge_profile`; this task extracts that into a shared function and reuses it.

In this handler family `:id` is the `judge_profile.id`, not the `auth_user.id`.

**Files:**
- Modify: `claoj/api/v2/register.go` (extract `createAccount`)
- Create: `claoj/api/v2/admin_user_create.go`
- Modify: `claoj/api/v2/admin_users_roles.go` (`AdminUserBan`: `day` becomes optional)
- Modify: `claoj/api/router.go` (two routes)
- Test: `claoj/api/v2/admin_user_create_test.go` (create)

**Interfaces:**
- Produces: `POST /api/admin/users` body `{"username", "email", "password", "full_name"?, "is_active"?: bool (default true)}` → 201 `{"success": true, "user": {"id": <profile id>, "username": string}}`; 409 on a taken username or email.
- Produces: `POST /api/admin/user/:id/password` body `{"password": string}` → 200 `{"success": true}`; 403 when the target is staff or superuser and the caller is not a superuser.
- Produces: `POST /api/admin/user/:id/ban` no longer requires `day`.

- [ ] **Step 1: Write the failing tests**

Create `claoj/api/v2/admin_user_create_test.go`:

```go
package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupAdminUserCreateDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
		&models.AuthUserPermission{}, &models.Profile{}, &models.Organization{}, &models.Language{},
	))
	db.DB = database
	userService = nil
	require.NoError(t, db.DB.Create(&models.Language{Key: "py3", Name: "Python 3"}).Error)
}

// seedAccount inserts an auth_user + profile and returns both.
func seedAccount(t *testing.T, username string, staff, superuser bool) (models.AuthUser, models.Profile) {
	t.Helper()
	u := models.AuthUser{
		Username: username, Email: username + "@example.com", Password: "old-hash",
		IsActive: true, IsStaff: staff, IsSuperuser: superuser,
	}
	require.NoError(t, db.DB.Create(&u).Error)
	p := models.Profile{UserID: u.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	return u, p
}

func TestAdminUserCreate_CreatesActiveUserAndProfile(t *testing.T) {
	setupAdminUserCreateDB(t)
	router := authedRouter(1, http.MethodPost, "/admin/users", AdminUserCreate)
	w := doRequest(router, http.MethodPost, "/admin/users",
		`{"username":"newbie","email":"newbie@example.com","password":"Passw0rdOK","full_name":"New Bie"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var u models.AuthUser
	require.NoError(t, db.DB.Where("username = ?", "newbie").First(&u).Error)
	require.True(t, u.IsActive, "admin-created accounts skip email verification")
	require.Equal(t, "newbie@example.com", u.Email)
	require.Equal(t, "New Bie", u.FirstName)
	require.NotEqual(t, "Passw0rdOK", u.Password)
	ok, err := auth.CheckPassword("Passw0rdOK", u.Password)
	require.NoError(t, err)
	require.True(t, ok)

	var p models.Profile
	require.NoError(t, db.DB.Where("user_id = ?", u.ID).First(&p).Error)
	require.Equal(t, "UTC", p.Timezone)
	require.NotZero(t, p.LanguageID)

	var body struct {
		Success bool `json:"success"`
		User    struct {
			ID       uint   `json:"id"`
			Username string `json:"username"`
		} `json:"user"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.Equal(t, p.ID, body.User.ID, "the admin API identifies users by profile id")
	require.Equal(t, "newbie", body.User.Username)
}

func TestAdminUserCreate_InactiveWhenAsked(t *testing.T) {
	setupAdminUserCreateDB(t)
	router := authedRouter(1, http.MethodPost, "/admin/users", AdminUserCreate)
	w := doRequest(router, http.MethodPost, "/admin/users",
		`{"username":"sleeper","email":"sleeper@example.com","password":"Passw0rdOK","is_active":false}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var u models.AuthUser
	require.NoError(t, db.DB.Where("username = ?", "sleeper").First(&u).Error)
	require.False(t, u.IsActive)
}

func TestAdminUserCreate_Rejects(t *testing.T) {
	setupAdminUserCreateDB(t)
	seedAccount(t, "taken", false, false)
	router := authedRouter(1, http.MethodPost, "/admin/users", AdminUserCreate)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"bad username", `{"username":"bad name!","email":"a@example.com","password":"Passw0rdOK"}`, http.StatusBadRequest},
		{"weak password", `{"username":"fine","email":"a@example.com","password":"short"}`, http.StatusBadRequest},
		{"bad email", `{"username":"fine","email":"nope","password":"Passw0rdOK"}`, http.StatusBadRequest},
		{"missing password", `{"username":"fine","email":"a@example.com"}`, http.StatusBadRequest},
		{"duplicate username", `{"username":"taken","email":"other@example.com","password":"Passw0rdOK"}`, http.StatusConflict},
		{"duplicate email", `{"username":"fine","email":"taken@example.com","password":"Passw0rdOK"}`, http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(router, http.MethodPost, "/admin/users", tc.body)
			require.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}

	var users, profiles int64
	require.NoError(t, db.DB.Model(&models.AuthUser{}).Count(&users).Error)
	require.NoError(t, db.DB.Model(&models.Profile{}).Count(&profiles).Error)
	require.Equal(t, int64(1), users, "a rejected create must not leave an auth_user row")
	require.Equal(t, int64(1), profiles, "a rejected create must not leave a profile row")
}

func TestAdminUserSetPassword_ChangesHash(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	target, targetProfile := seedAccount(t, "bob", false, false)

	router := authedRouter(root.ID, http.MethodPost, "/admin/user/:id/password", AdminUserSetPassword)
	w := doRequest(router, http.MethodPost, "/admin/user/"+strconv.Itoa(int(targetProfile.ID))+"/password", `{"password":"N3wPassword"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.AuthUser
	require.NoError(t, db.DB.First(&got, target.ID).Error)
	ok, err := auth.CheckPassword("N3wPassword", got.Password)
	require.NoError(t, err)
	require.True(t, ok)
}

// Setting a password is an account takeover primitive, so a staff user who is
// not a superuser must not be able to use it on a staff or superuser account.
func TestAdminUserSetPassword_StaffTargetNeedsSuperuser(t *testing.T) {
	setupAdminUserCreateDB(t)
	staff, _ := seedAccount(t, "staff", true, false)
	for _, tc := range []struct {
		name             string
		staff, superuser bool
	}{
		{"otherstaff", true, false},
		{"theroot", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target, targetProfile := seedAccount(t, tc.name, tc.staff, tc.superuser)
			router := authedRouter(staff.ID, http.MethodPost, "/admin/user/:id/password", AdminUserSetPassword)
			w := doRequest(router, http.MethodPost, "/admin/user/"+strconv.Itoa(int(targetProfile.ID))+"/password", `{"password":"N3wPassword"}`)
			require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

			var got models.AuthUser
			require.NoError(t, db.DB.First(&got, target.ID).Error)
			require.Equal(t, "old-hash", got.Password, "a refused request must not change the password")
		})
	}
}

func TestAdminUserSetPassword_Errors(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	_, targetProfile := seedAccount(t, "bob", false, false)
	router := authedRouter(root.ID, http.MethodPost, "/admin/user/:id/password", AdminUserSetPassword)
	id := strconv.Itoa(int(targetProfile.ID))

	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPost, "/admin/user/"+id+"/password", `{"password":"weak"}`).Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPost, "/admin/user/"+id+"/password", `{}`).Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPost, "/admin/user/abc/password", `{"password":"N3wPassword"}`).Code)
	require.Equal(t, http.StatusNotFound, doRequest(router, http.MethodPost, "/admin/user/999/password", `{"password":"N3wPassword"}`).Code)
}

// judge_profile has no ban-expiry column, so a ban cannot carry a duration;
// the endpoint must not demand one.
func TestAdminUserBan_DayIsOptional(t *testing.T) {
	setupAdminUserCreateDB(t)
	root, _ := seedAccount(t, "root", true, true)
	_, targetProfile := seedAccount(t, "spammer", false, false)

	router := authedRouter(root.ID, http.MethodPost, "/admin/user/:id/ban", AdminUserBan)
	w := doRequest(router, http.MethodPost, "/admin/user/"+strconv.Itoa(int(targetProfile.ID))+"/ban", `{"reason":"spam"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.Profile
	require.NoError(t, db.DB.First(&got, targetProfile.ID).Error)
	require.NotNil(t, got.BanReason)
	require.Equal(t, "spam", *got.BanReason)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./api/v2/ -run 'TestAdminUser(Create|SetPassword|Ban_Day)' -v`
Expected: compile error — `AdminUserCreate` and `AdminUserSetPassword` are undefined.

- [ ] **Step 3: Extract `createAccount` from `Register`**

In `claoj/api/v2/register.go`:

Add after the `RegisterRequest` type:

```go
// newAccount is what createAccount needs to insert an auth_user row and its
// judge_profile.
type newAccount struct {
	Username       string
	Email          string
	HashedPassword string
	FullName       string
	Language       string // judge_language.key; empty means the site default
	Timezone       string // empty means UTC
	IsActive       bool
}

var (
	errUsernameTaken = errors.New("username already taken")
	errEmailTaken    = errors.New("email already taken")
)

// createAccount inserts the auth_user and judge_profile rows inside tx and
// returns the new profile. Callers validate the username and password first.
func createAccount(tx *gorm.DB, a newAccount) (models.Profile, error) {
	var count int64
	tx.Model(&models.AuthUser{}).Where("username = ?", a.Username).Count(&count)
	if count > 0 {
		return models.Profile{}, errUsernameTaken
	}
	tx.Model(&models.AuthUser{}).Where("email = ?", a.Email).Count(&count)
	if count > 0 {
		return models.Profile{}, errEmailTaken
	}

	user := models.AuthUser{
		Username:   a.Username,
		Email:      a.Email,
		Password:   a.HashedPassword,
		FirstName:  a.FullName,
		IsActive:   a.IsActive,
		DateJoined: time.Now(),
	}
	if err := tx.Create(&user).Error; err != nil {
		return models.Profile{}, err
	}
	// GORM skips boolean zero values on columns with a default, so is_active
	// has to be written explicitly.
	if err := tx.Model(&models.AuthUser{}).Where("id = ?", user.ID).Update("is_active", a.IsActive).Error; err != nil {
		return models.Profile{}, err
	}

	langKey := a.Language
	if langKey == "" {
		langKey = config.C.App.DefaultLanguage
	}
	var lang models.Language
	if err := tx.Where("`key` = ?", langKey).First(&lang).Error; err != nil {
		// fallback to any
		tx.First(&lang)
	}

	tz := a.Timezone
	if tz == "" {
		tz = "UTC"
	}

	profile := models.Profile{
		UserID:     user.ID,
		LanguageID: lang.ID,
		Timezone:   tz,
		LastAccess: time.Now(),
		// judge_profile.math_engine is varchar(4) with Django choices
		// tex/svg/mml/jax/auto. "auto" is v1's MATHOID_DEFAULT_TYPE; the
		// old "mathjax" value overflowed the column (Error 1406) and broke
		// registration entirely.
		MathEngine: "auto",
	}
	if err := tx.Create(&profile).Error; err != nil {
		return models.Profile{}, err
	}
	return profile, nil
}
```

Then, in `Register`, replace everything from the line `var userID uint` down to and including the closing `}()` of the `go func() { … }()` block with:

```go
	var profile models.Profile
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var txErr error
		profile, txErr = createAccount(tx, newAccount{
			Username:       req.Username,
			Email:          req.Email,
			HashedPassword: hashedPassword,
			FullName:       req.FullName,
			Language:       req.Language,
			Timezone:       req.Timezone,
			IsActive:       false, // require email verification before activation
		})
		return txErr
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	// Send verification email asynchronously
	go func() {
		if err := SendVerificationEmailOnRegistration(profile.UserID, req.Email, req.Username); err != nil {
			// Log error but don't fail registration
		}
	}()
```

`Register` must behave exactly as before: same validation, same status codes, same response body.

- [ ] **Step 4: Implement the two handlers**

Create `claoj/api/v2/admin_user_create.go`:

```go
package v2

import (
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strconv"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminUserCreate - POST /api/v2/admin/users
// Creates an account on behalf of a user. Unlike self-registration the
// account is active immediately unless is_active is false.
func AdminUserCreate(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		FullName string `json:"full_name"`
		IsActive *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	if !usernameRegex.MatchString(input.Username) || len(input.Username) > 30 {
		c.JSON(http.StatusBadRequest, apiUserError("invalid username: must be 1-30 chars, alphanumeric or underscore"))
		return
	}
	if _, err := mail.ParseAddress(input.Email); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid email address"))
		return
	}
	if err := validatePasswordStrength(input.Password); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	hashedPassword, err := auth.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to hash password"))
		return
	}

	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}

	var profile models.Profile
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var txErr error
		profile, txErr = createAccount(tx, newAccount{
			Username:       input.Username,
			Email:          input.Email,
			HashedPassword: hashedPassword,
			FullName:       input.FullName,
			IsActive:       active,
		})
		return txErr
	})
	if err != nil {
		if errors.Is(err, errUsernameTaken) || errors.Is(err, errEmailTaken) {
			c.JSON(http.StatusConflict, apiUserError(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"user": gin.H{
			"id":       profile.ID,
			"username": input.Username,
		},
	})
}

// AdminUserSetPassword - POST /api/v2/admin/user/:id/password
// Sets a new password for the user with judge_profile.id = :id and signs
// them out everywhere.
func AdminUserSetPassword(c *gin.Context) {
	profileID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid user ID"))
		return
	}

	var input struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}
	if err := validatePasswordStrength(input.Password); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError(err.Error()))
		return
	}

	var profile models.Profile
	if err := db.DB.Preload("User").First(&profile, uint(profileID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, apiError("user not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Setting a password hands the caller the account, so only a superuser
	// may do it to a staff or superuser account.
	if (profile.User.IsStaff || profile.User.IsSuperuser) && !auth.GetAccess(c).IsSuperuser {
		c.JSON(http.StatusForbidden, apiUserError("superuser access required to set the password of a staff account"))
		return
	}

	hashedPassword, err := auth.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apiError("failed to hash password"))
		return
	}
	if err := db.DB.Model(&models.AuthUser{}).Where("id = ?", profile.UserID).
		Update("password", hashedPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// Sessions opened with the old password must not survive the reset.
	// RefreshStore is nil in tests.
	if authHandlers.RefreshStore != nil {
		if err := authHandlers.RefreshStore.RevokeAllForUser(profile.UserID); err != nil {
			// Best-effort: the password itself is already changed.
			log.Printf("admin: revoke sessions of user %d after password reset: %v", profile.UserID, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
```

- [ ] **Step 5: Make `day` optional on ban**

In `claoj/api/v2/admin_users_roles.go`, in `AdminUserBan`, replace the `input` struct with:

```go
	var input struct {
		Reason string `json:"reason" binding:"required"`
		// Accepted for older clients and ignored: judge_profile has no
		// ban-expiry column, so every ban lasts until an unban.
		Day int `json:"day"`
	}
```

- [ ] **Step 6: Register the routes**

In `claoj/api/router.go`, directly after the line `admin.POST("/admin/user/:id/unban", v2.AdminUserUnban)`, add:

```go
			admin.POST("/admin/users", auth.RequirePerm("auth.add_user"), v2.AdminUserCreate)
			admin.POST("/admin/user/:id/password", auth.RequirePerm("auth.change_user"), v2.AdminUserSetPassword)
```

- [ ] **Step 7: Run the tests**

Run: `go test ./api/v2/ -run 'TestAdminUser' -v`
Expected: PASS.

Run: `go build ./... && go test ./api/... ./service/... ./auth/...`
Expected: build succeeds, all packages `ok`.

- [ ] **Step 8: Commit**

```bash
git add claoj/api
git commit -F - <<'EOF'
feat(admin): create users and set passwords from the admin API

Register and the new admin create share one createAccount helper.
The ban endpoint no longer demands a day count it never honoured.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 4: Delete a submission

Deleting removes the submission with its source, test cases and contest link (what Django's cascade removes), then recalculates what the submission fed: the owner's points and problem count, the owner's organizations, and — for a contest submission — the participation's score.

Submission statuses: `QU`, `P`, `G` mean still being judged; `D`, `CE`, `IE`, `AB` are finished.

**Files:**
- Create: `claoj/service/submission/delete.go`
- Modify: `claoj/service/submission/errors.go` (one error)
- Modify: `claoj/api/v2/admin_submission.go` (handler)
- Modify: `claoj/api/router.go` (one route)
- Test: `claoj/service/submission/delete_test.go` (create)
- Test: `claoj/api/v2/admin_submission_delete_test.go` (create)

**Interfaces:**
- Produces: `func (s *SubmissionService) Delete(req DeleteRequest) error` with `type DeleteRequest struct{ SubmissionID uint }`; errors `ErrSubmissionNotFound`, `ErrSubmissionInProgress`.
- Produces: `DELETE /api/admin/submission/:id` → 200 `{"success": true, "message": "Submission deleted"}`; 404; 409 `{"error": "submission is still being judged; abort it first"}`.

- [ ] **Step 1: Write the failing service tests**

Create `claoj/service/submission/delete_test.go`:

```go
package submission

import (
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupDeleteDB(t *testing.T) models.Profile {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.ContestTag{},
		&models.Contest{}, &models.ContestParticipation{},
		&models.Submission{}, &models.SubmissionSource{}, &models.SubmissionTestCase{}, &models.ContestSubmission{},
	))
	db.DB = database

	u := models.AuthUser{Username: "alice", IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	prof := models.Profile{UserID: u.ID, Timezone: "UTC", Points: 150, ProblemCount: 2}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&prof).Error)
	return prof
}

// seedAC inserts a public problem worth pts and an accepted submission of it
// by prof, with a source row and two test cases.
func seedAC(t *testing.T, prof models.Profile, code string, pts float64) models.Submission {
	t.Helper()
	p := models.Problem{Code: code, Name: code, IsPublic: true, Points: pts}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	ac := "AC"
	s := models.Submission{
		UserID: prof.ID, ProblemID: p.ID, LanguageID: 1, Date: time.Now(),
		Status: "D", Result: &ac, Points: &pts, CasePoints: pts, CaseTotal: pts,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&s).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.SubmissionSource{SubmissionID: s.ID, Source: "print(1)"}).Error)
	for i := 1; i <= 2; i++ {
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.SubmissionTestCase{SubmissionID: s.ID, Case: i, Status: "AC"}).Error)
	}
	return s
}

func countWhere(t *testing.T, model any, where string, arg any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.DB.Model(model).Where(where, arg).Count(&n).Error)
	return n
}

func TestDelete_RemovesRowsAndRecalculatesPoints(t *testing.T) {
	prof := setupDeleteDB(t)
	doomed := seedAC(t, prof, "p1", 100)
	kept := seedAC(t, prof, "p2", 50)

	require.NoError(t, NewSubmissionService(nil).Delete(DeleteRequest{SubmissionID: doomed.ID}))

	require.Equal(t, int64(0), countWhere(t, &models.Submission{}, "id = ?", doomed.ID))
	require.Equal(t, int64(0), countWhere(t, &models.SubmissionSource{}, "submission_id = ?", doomed.ID))
	require.Equal(t, int64(0), countWhere(t, &models.SubmissionTestCase{}, "submission_id = ?", doomed.ID))

	// The user's other submission is untouched.
	require.Equal(t, int64(1), countWhere(t, &models.Submission{}, "id = ?", kept.ID))
	require.Equal(t, int64(1), countWhere(t, &models.SubmissionSource{}, "submission_id = ?", kept.ID))
	require.Equal(t, int64(2), countWhere(t, &models.SubmissionTestCase{}, "submission_id = ?", kept.ID))

	var got models.Profile
	require.NoError(t, db.DB.First(&got, prof.ID).Error)
	require.Equal(t, 50.0, got.Points, "points must be recalculated without the deleted submission")
	require.Equal(t, 1, got.ProblemCount)
}

func TestDelete_ContestSubmissionRecomputesParticipation(t *testing.T) {
	prof := setupDeleteDB(t)
	s := seedAC(t, prof, "p1", 100)

	contest := models.Contest{Key: "c1", Name: "C1", StartTime: time.Now().Add(-time.Hour), EndTime: time.Now().Add(time.Hour)}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&contest).Error)
	part := models.ContestParticipation{ContestID: contest.ID, UserID: prof.ID, RealStart: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&part).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.ContestSubmission{
		SubmissionID: s.ID, ProblemID: 1, ParticipationID: part.ID, Points: 100,
	}).Error)

	var gotID uint
	var gotKey string
	calls := 0
	old := recomputeParticipation
	t.Cleanup(func() { recomputeParticipation = old })
	recomputeParticipation = func(p *models.ContestParticipation) error {
		calls++
		gotID, gotKey = p.ID, p.Contest.Key
		return nil
	}

	require.NoError(t, NewSubmissionService(nil).Delete(DeleteRequest{SubmissionID: s.ID}))

	require.Equal(t, int64(0), countWhere(t, &models.ContestSubmission{}, "submission_id = ?", s.ID))
	require.Equal(t, int64(1), countWhere(t, &models.ContestParticipation{}, "id = ?", part.ID), "the participation itself stays")
	require.Equal(t, 1, calls)
	require.Equal(t, part.ID, gotID)
	require.Equal(t, "c1", gotKey, "the participation is handed over with its contest loaded")
}

func TestDelete_NonContestSubmissionSkipsParticipation(t *testing.T) {
	prof := setupDeleteDB(t)
	s := seedAC(t, prof, "p1", 100)

	calls := 0
	old := recomputeParticipation
	t.Cleanup(func() { recomputeParticipation = old })
	recomputeParticipation = func(*models.ContestParticipation) error { calls++; return nil }

	require.NoError(t, NewSubmissionService(nil).Delete(DeleteRequest{SubmissionID: s.ID}))
	require.Equal(t, 0, calls)
}

func TestDelete_InProgressIsRejected(t *testing.T) {
	for _, status := range []string{"QU", "P", "G"} {
		t.Run(status, func(t *testing.T) {
			prof := setupDeleteDB(t)
			s := seedAC(t, prof, "p1", 100)
			require.NoError(t, db.DB.Model(&models.Submission{}).Where("id = ?", s.ID).Update("status", status).Error)

			err := NewSubmissionService(nil).Delete(DeleteRequest{SubmissionID: s.ID})
			require.ErrorIs(t, err, ErrSubmissionInProgress)
			require.Equal(t, int64(1), countWhere(t, &models.Submission{}, "id = ?", s.ID))
			require.Equal(t, int64(1), countWhere(t, &models.SubmissionSource{}, "submission_id = ?", s.ID))
		})
	}
}

func TestDelete_NotFound(t *testing.T) {
	setupDeleteDB(t)
	require.ErrorIs(t, NewSubmissionService(nil).Delete(DeleteRequest{SubmissionID: 999}), ErrSubmissionNotFound)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./service/submission/ -run 'TestDelete_' -v`
Expected: compile error — `DeleteRequest`, `recomputeParticipation` and `ErrSubmissionInProgress` are undefined.

- [ ] **Step 3: Implement the service method**

In `claoj/service/submission/errors.go`, add to the `var (` block:

```go
	ErrSubmissionInProgress     = errors.New("submission: submission is still being judged")
```

Create `claoj/service/submission/delete.go`:

```go
package submission

import (
	"errors"
	"log"

	"github.com/CLAOJ/claoj/contest_format"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/scoring"
	"gorm.io/gorm"
)

// DeleteRequest holds the parameters for deleting a submission.
type DeleteRequest struct {
	SubmissionID uint
}

// recomputeParticipation re-runs the contest format over a participation
// (loaded with its Contest). It is a variable so tests can replace it: the
// contest formats use MySQL-only SQL.
var recomputeParticipation = func(p *models.ContestParticipation) error {
	cf := contest_format.GetFormat(p.Contest.FormatName, &p.Contest, p.Contest.FormatConfig)
	return cf.UpdateParticipation(p)
}

// Delete permanently removes a finished submission together with its source,
// test cases and contest link — the rows Django's cascade removes — and then
// recalculates what the submission fed: the owner's points, the owner's
// organizations and, for a contest submission, the participation's score.
//
// A submission that is still being judged is refused: the judge would keep
// writing results for a row that no longer exists.
func (s *SubmissionService) Delete(req DeleteRequest) error {
	var sub models.Submission
	if err := db.DB.First(&sub, req.SubmissionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSubmissionNotFound
		}
		return err
	}
	switch sub.Status {
	case "QU", "P", "G":
		return ErrSubmissionInProgress
	}

	var cSub models.ContestSubmission
	inContest := db.DB.Preload("Participation.Contest").
		Where("submission_id = ?", sub.ID).First(&cSub).Error == nil

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("submission_id = ?", sub.ID).Delete(&models.SubmissionTestCase{}).Error; err != nil {
			return err
		}
		if err := tx.Where("submission_id = ?", sub.ID).Delete(&models.SubmissionSource{}).Error; err != nil {
			return err
		}
		if err := tx.Where("submission_id = ?", sub.ID).Delete(&models.ContestSubmission{}).Error; err != nil {
			return err
		}
		return tx.Delete(&sub).Error
	})
	if err != nil {
		return err
	}

	// The submission is gone; a failed recalculation is logged, not returned,
	// because retrying the delete could not fix it.
	if err := scoring.CalculateProfilePoints(db.DB, sub.UserID); err != nil {
		log.Printf("submission: delete %d: recalculate points for profile %d: %v", sub.ID, sub.UserID, err)
	}
	var orgIDs []uint
	db.DB.Table("judge_profile_organizations").Where("profile_id = ?", sub.UserID).Pluck("organization_id", &orgIDs)
	for _, orgID := range orgIDs {
		if err := scoring.CalculateOrganizationPoints(db.DB, orgID); err != nil {
			log.Printf("submission: delete %d: recalculate points for organization %d: %v", sub.ID, orgID, err)
		}
	}
	if inContest {
		if err := recomputeParticipation(&cSub.Participation); err != nil {
			log.Printf("submission: delete %d: recompute participation %d: %v", sub.ID, cSub.ParticipationID, err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run the service tests**

Run: `go test ./service/submission/ -v`
Expected: PASS (the new `TestDelete_*` tests and the existing ones).

- [ ] **Step 5: Write the failing handler test**

Create `claoj/api/v2/admin_submission_delete_test.go`:

```go
package v2

import (
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupSubmissionDeleteDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.Submission{}))
	db.DB = database
}

func TestAdminSubmissionDelete_MapsServiceErrors(t *testing.T) {
	setupSubmissionDeleteDB(t)
	queued := models.Submission{UserID: 1, ProblemID: 1, LanguageID: 1, Date: time.Now(), Status: "QU"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&queued).Error)

	router := authedRouter(1, http.MethodDelete, "/admin/submission/:id", AdminSubmissionDelete)

	require.Equal(t, http.StatusNotFound, doRequest(router, http.MethodDelete, "/admin/submission/999", "").Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodDelete, "/admin/submission/abc", "").Code)

	w := doRequest(router, http.MethodDelete, "/admin/submission/1", "")
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "abort it first")

	var n int64
	require.NoError(t, db.DB.Model(&models.Submission{}).Count(&n).Error)
	require.Equal(t, int64(1), n)
}
```

Run: `go test ./api/v2/ -run TestAdminSubmissionDelete -v`
Expected: compile error — `AdminSubmissionDelete` is undefined.

- [ ] **Step 6: Implement the handler and route**

Append to `claoj/api/v2/admin_submission.go`:

```go
// AdminSubmissionDelete - DELETE /api/v2/admin/submission/:id
// Permanently deletes a finished submission and recalculates the points it fed.
func AdminSubmissionDelete(c *gin.Context) {
	var submissionID uint
	if err := parseUint(c.Param("id"), &submissionID); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid submission ID"))
		return
	}

	if err := getSubmissionService().Delete(submission.DeleteRequest{SubmissionID: submissionID}); err != nil {
		switch {
		case errors.Is(err, submission.ErrSubmissionNotFound):
			c.JSON(http.StatusNotFound, apiError("submission not found"))
		case errors.Is(err, submission.ErrSubmissionInProgress):
			c.JSON(http.StatusConflict, apiUserError("submission is still being judged; abort it first"))
		default:
			c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Submission deleted",
	})
}
```

In `claoj/api/router.go`, directly after the line `admin.POST("/admin/submission/:id/abort", v2.AdminSubmissionAbort)`, add:

```go
			admin.DELETE("/admin/submission/:id", auth.RequirePerm("judge.delete_submission"), v2.AdminSubmissionDelete)
```

- [ ] **Step 7: Run the tests**

Run: `go test ./api/v2/ -run TestAdminSubmissionDelete -v && go test ./service/submission/`
Expected: PASS.

Run: `go build ./... && go test ./api/... ./service/...`
Expected: build succeeds, all packages `ok`.

- [ ] **Step 8: Commit**

```bash
git add claoj/api claoj/service
git commit -F - <<'EOF'
feat(admin): delete a submission and recalculate what it fed

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 5: Ticket delete, clarification edit and delete, and the clarification-answer bug

Four small holes:

- `ContestClarificationAnswer` discards the parsed id (`var clarID int` is never assigned), so it always looks up clarification 0 and returns 404. Nobody can answer a clarification.
- A contest clarification cannot be deleted.
- A problem clarification cannot be edited.
- A ticket cannot be deleted.

**Files:**
- Modify: `claoj/api/v2/contest/contest_clarification.go`
- Modify: `claoj/api/v2/problem_clarification.go`
- Modify: `claoj/api/v2/ticket.go`
- Modify: `claoj/api/router.go` (three routes)
- Test: `claoj/api/v2/contest/contest_clarification_test.go` (create)
- Test: `claoj/api/v2/admin_small_gaps_test.go` (create)

**Interfaces:**
- Consumes: `numericParam` is not needed here; every id in this task is parsed to an integer.
- Produces: `DELETE /api/contest/:key/clarification/:id` → 200 `{"message": "clarification deleted"}`.
- Produces: `PATCH /api/admin/problem/clarification/:id` body `{"description": string}` → 200 `{"id", "problem_id", "description", "date"}`.
- Produces: `DELETE /api/admin/ticket/:id` → 200 `{"message": "ticket deleted"}`.

- [ ] **Step 1: Write the failing contest clarification tests**

Create `claoj/api/v2/contest/contest_clarification_test.go`:

```go
package contest

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type clarFixture struct {
	root, plain  models.AuthUser
	contest      models.Contest
	otherContest models.Contest
	clar         models.ContestClarification
}

func setupClarificationDB(t *testing.T) clarFixture {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
		&models.AuthUserPermission{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.ContestTag{},
		&models.Contest{}, &models.ContestClarification{},
	))
	db.DB = database

	var f clarFixture
	f.root = models.AuthUser{Username: "root", IsActive: true, IsStaff: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&f.root).Error)
	f.plain = models.AuthUser{Username: "bob", IsActive: true}
	require.NoError(t, db.DB.Create(&f.plain).Error)
	for _, u := range []models.AuthUser{f.root, f.plain} {
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Profile{UserID: u.ID, Timezone: "UTC"}).Error)
	}

	now := time.Now()
	f.contest = models.Contest{Key: "c1", Name: "C1", StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&f.contest).Error)
	f.otherContest = models.Contest{Key: "c2", Name: "C2", StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&f.otherContest).Error)

	f.clar = models.ContestClarification{ContestID: f.contest.ID, Question: "Is n positive?", CreateTime: now, AuthorID: 2}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&f.clar).Error)
	return f
}

func clarRequest(userID uint, method, route, target, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, route, func(c *gin.Context) {
		c.Set("user_id", userID)
		handler(c)
	})
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// Regression: the handler used to discard the parsed id and always look up
// clarification 0, so every answer returned 404.
func TestContestClarificationAnswer_FindsTheClarification(t *testing.T) {
	f := setupClarificationDB(t)
	id := strconv.Itoa(int(f.clar.ID))

	w := clarRequest(f.root.ID, http.MethodPost, "/contest/:key/clarification/:id/answer",
		"/contest/c1/clarification/"+id+"/answer", `{"answer":"Yes","is_public":true}`, ContestClarificationAnswer)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.ContestClarification
	require.NoError(t, db.DB.First(&got, f.clar.ID).Error)
	require.True(t, got.IsAnswered)
	require.NotNil(t, got.Answer)
	require.Equal(t, "Yes", *got.Answer)
}

func TestContestClarificationDelete(t *testing.T) {
	const route = "/contest/:key/clarification/:id"

	t.Run("editor deletes", func(t *testing.T) {
		f := setupClarificationDB(t)
		id := strconv.Itoa(int(f.clar.ID))
		w := clarRequest(f.root.ID, http.MethodDelete, route, "/contest/c1/clarification/"+id, "", ContestClarificationDelete)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var n int64
		require.NoError(t, db.DB.Model(&models.ContestClarification{}).Count(&n).Error)
		require.Equal(t, int64(0), n)
	})

	t.Run("non-editor is forbidden", func(t *testing.T) {
		f := setupClarificationDB(t)
		id := strconv.Itoa(int(f.clar.ID))
		w := clarRequest(f.plain.ID, http.MethodDelete, route, "/contest/c1/clarification/"+id, "", ContestClarificationDelete)
		require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

		var n int64
		require.NoError(t, db.DB.Model(&models.ContestClarification{}).Count(&n).Error)
		require.Equal(t, int64(1), n)
	})

	t.Run("clarification of another contest is not found", func(t *testing.T) {
		f := setupClarificationDB(t)
		id := strconv.Itoa(int(f.clar.ID))
		w := clarRequest(f.root.ID, http.MethodDelete, route, "/contest/c2/clarification/"+id, "", ContestClarificationDelete)
		require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

		var n int64
		require.NoError(t, db.DB.Model(&models.ContestClarification{}).Count(&n).Error)
		require.Equal(t, int64(1), n)
	})

	t.Run("bad ids", func(t *testing.T) {
		f := setupClarificationDB(t)
		require.Equal(t, http.StatusBadRequest,
			clarRequest(f.root.ID, http.MethodDelete, route, "/contest/c1/clarification/abc", "", ContestClarificationDelete).Code)
		require.Equal(t, http.StatusNotFound,
			clarRequest(f.root.ID, http.MethodDelete, route, "/contest/c1/clarification/999", "", ContestClarificationDelete).Code)
		require.Equal(t, http.StatusNotFound,
			clarRequest(f.root.ID, http.MethodDelete, route, "/contest/nope/clarification/1", "", ContestClarificationDelete).Code)
	})
}
```

- [ ] **Step 2: Write the failing problem clarification and ticket tests**

Create `claoj/api/v2/admin_small_gaps_test.go`:

```go
package v2

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupSmallGapsDB(t *testing.T) (root, plain models.AuthUser, plainProfile models.Profile) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
		&models.AuthUserPermission{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.ProblemClarification{},
		&models.Ticket{}, &models.TicketMessage{},
	))
	db.DB = database

	root = models.AuthUser{Username: "root", IsActive: true, IsStaff: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&root).Error)
	plain = models.AuthUser{Username: "bob", IsActive: true}
	require.NoError(t, db.DB.Create(&plain).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Profile{UserID: root.ID, Timezone: "UTC"}).Error)
	plainProfile = models.Profile{UserID: plain.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&plainProfile).Error)
	return root, plain, plainProfile
}

func TestProblemClarificationUpdate(t *testing.T) {
	root, plain, _ := setupSmallGapsDB(t)
	p := models.Problem{Code: "p1", Name: "P1"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	clar := models.ProblemClarification{ProblemID: p.ID, Description: "old text", Date: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&clar).Error)

	const route = "/admin/problem/clarification/:id"
	target := "/admin/problem/clarification/" + strconv.Itoa(int(clar.ID))
	load := func() models.ProblemClarification {
		var got models.ProblemClarification
		require.NoError(t, db.DB.First(&got, clar.ID).Error)
		return got
	}

	t.Run("non-editor is forbidden", func(t *testing.T) {
		w := doRequest(authedRouter(plain.ID, http.MethodPatch, route, ProblemClarificationUpdate), http.MethodPatch, target, `{"description":"hacked"}`)
		require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		require.Equal(t, "old text", load().Description)
	})

	router := authedRouter(root.ID, http.MethodPatch, route, ProblemClarificationUpdate)

	t.Run("empty description is rejected", func(t *testing.T) {
		w := doRequest(router, http.MethodPatch, target, `{"description":""}`)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		require.Equal(t, "old text", load().Description)
	})

	t.Run("bad ids", func(t *testing.T) {
		require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodPatch, "/admin/problem/clarification/abc", `{"description":"x"}`).Code)
		require.Equal(t, http.StatusNotFound, doRequest(router, http.MethodPatch, "/admin/problem/clarification/999", `{"description":"x"}`).Code)
	})

	t.Run("editor updates", func(t *testing.T) {
		w := doRequest(router, http.MethodPatch, target, `{"description":"new text"}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.Equal(t, "new text", load().Description)

		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Equal(t, "new text", body["description"])
		require.Equal(t, float64(clar.ID), body["id"])
	})
}

func TestAdminTicketDelete(t *testing.T) {
	root, _, plainProfile := setupSmallGapsDB(t)

	seed := func(title string) models.Ticket {
		tk := models.Ticket{Title: title, UserID: plainProfile.ID, Time: time.Now(), ContentTypeID: 1, ObjectID: 1}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&tk).Error)
		for i := 0; i < 2; i++ {
			require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.TicketMessage{
				TicketID: tk.ID, UserID: plainProfile.ID, Body: "hello", Time: time.Now(),
			}).Error)
		}
		require.NoError(t, db.DB.Model(&tk).Association("Assignees").Append(&plainProfile))
		return tk
	}
	doomed := seed("doomed")
	kept := seed("kept")

	count := func(table, where string, arg any) int64 {
		var n int64
		require.NoError(t, db.DB.Table(table).Where(where, arg).Count(&n).Error)
		return n
	}

	router := authedRouter(root.ID, http.MethodDelete, "/admin/ticket/:id", AdminTicketDelete)
	w := doRequest(router, http.MethodDelete, "/admin/ticket/"+strconv.Itoa(int(doomed.ID)), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, int64(0), count("judge_ticket", "id = ?", doomed.ID))
	require.Equal(t, int64(0), count("judge_ticketmessage", "ticket_id = ?", doomed.ID))
	require.Equal(t, int64(0), count("judge_ticket_assignees", "ticket_id = ?", doomed.ID))

	// The other ticket keeps its messages and assignee.
	require.Equal(t, int64(1), count("judge_ticket", "id = ?", kept.ID))
	require.Equal(t, int64(2), count("judge_ticketmessage", "ticket_id = ?", kept.ID))
	require.Equal(t, int64(1), count("judge_ticket_assignees", "ticket_id = ?", kept.ID))

	require.Equal(t, http.StatusNotFound, doRequest(router, http.MethodDelete, "/admin/ticket/999", "").Code)
	require.Equal(t, http.StatusBadRequest, doRequest(router, http.MethodDelete, "/admin/ticket/abc", "").Code)
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `go test ./api/v2/contest/ ./api/v2/ -run 'TestContestClarification|TestProblemClarificationUpdate|TestAdminTicketDelete' -v`
Expected: compile errors — `ContestClarificationDelete`, `ProblemClarificationUpdate` and `AdminTicketDelete` are undefined.

- [ ] **Step 4: Fix the answer bug and add the contest clarification delete**

In `claoj/api/v2/contest/contest_clarification.go`, inside `ContestClarificationAnswer`, replace:

```go
	clarificationID := c.Param("id")
	var clarID int
	if _, err := parseInt(clarificationID); err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid clarification id"))
		return
	}
```

with:

```go
	clarID, err := parseInt(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid clarification id"))
		return
	}
```

Then add this function directly after `ContestClarificationAnswer`:

```go
// ContestClarificationDelete – DELETE /api/v2/contest/:key/clarification/:id
// Contest editors may delete a clarification of their contest.
func ContestClarificationDelete(c *gin.Context) {
	if _, exists := c.Get("user_id"); !exists {
		c.JSON(http.StatusUnauthorized, apiError("unauthorized"))
		return
	}

	clarID, err := parseInt(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError("invalid clarification id"))
		return
	}

	var contest models.Contest
	if err := db.DB.Preload("Authors").Preload("Curators").Where("`key` = ?", c.Param("key")).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return
	}

	// Django parity: only contest editors may manage clarifications.
	if !auth.CanEditContest(c, &contest) {
		c.JSON(http.StatusForbidden, apiError("admin access required"))
		return
	}

	res := db.DB.Where("id = ? AND contest_id = ?", clarID, contest.ID).Delete(&models.ContestClarification{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError(res.Error.Error()))
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, apiError("clarification not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "clarification deleted",
	})
}
```

- [ ] **Step 5: Add the problem clarification update**

In `claoj/api/v2/problem_clarification.go`, add directly before `ProblemClarificationDelete`:

```go
// ProblemClarificationUpdate – PATCH /api/v2/admin/problem/clarification/:id
// Admin endpoint to edit the text of a problem clarification
func ProblemClarificationUpdate(c *gin.Context) {
	var id uint
	if err := parseUint(c.Param("id"), &id); err != nil {
		c.JSON(http.StatusBadRequest, apiUserError("invalid id"))
		return
	}

	var clarification models.ProblemClarification
	if err := db.DB.First(&clarification, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, apiError("clarification not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	// v1 parity: only editors of the parent problem may edit its clarifications.
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

	var req struct {
		Description string `json:"description" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}

	if err := db.DB.Model(&clarification).Update("description", req.Description).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          clarification.ID,
		"problem_id":  clarification.ProblemID,
		"description": req.Description,
		"date":        clarification.Date,
	})
}
```

- [ ] **Step 6: Add the ticket delete**

Append to `claoj/api/v2/ticket.go`:

```go
// AdminTicketDelete - DELETE /admin/ticket/:id
// Permanently deletes a ticket with its messages and assignee links (admin only)
func AdminTicketDelete(c *gin.Context) {
	ticketID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket ID"})
		return
	}

	var ticket models.Ticket
	if err := db.DB.First(&ticket, ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ticket_id = ?", ticket.ID).Delete(&models.TicketMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&ticket).Association("Assignees").Clear(); err != nil {
			return err
		}
		return tx.Delete(&ticket).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ticket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket deleted"})
}
```

- [ ] **Step 7: Register the routes**

In `claoj/api/router.go`:

Directly after `admin.DELETE("/admin/problem/clarification/:id", v2.ProblemClarificationDelete)` add:

```go
			admin.PATCH("/admin/problem/clarification/:id", v2.ProblemClarificationUpdate)
```

Directly after `admin.PATCH("/admin/ticket/:id/notes", v2.AdminTicketUpdateNotes)` add:

```go
			admin.DELETE("/admin/ticket/:id", auth.RequirePerm("judge.delete_ticket"), v2.AdminTicketDelete)
```

Directly after `protected.POST("/contest/:key/clarification/:id/answer", contestHandlers.ContestClarificationAnswer)` add:

```go
			protected.DELETE("/contest/:key/clarification/:id", contestHandlers.ContestClarificationDelete)
```

- [ ] **Step 8: Run the tests**

Run: `go test ./api/v2/contest/ ./api/v2/ -run 'TestContestClarification|TestProblemClarificationUpdate|TestAdminTicketDelete' -v`
Expected: PASS.

Run: `go build ./... && go test ./api/... ./service/... ./auth/...`
Expected: build succeeds, all packages `ok`.

- [ ] **Step 9: Commit**

```bash
git add claoj/api
git commit -F - <<'EOF'
feat(admin): delete tickets and clarifications, edit problem clarifications

Also fixes ContestClarificationAnswer, which discarded the parsed id and
therefore returned 404 for every clarification.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 6: Admin UI for judges and users

Expose Task 2 and Task 3 in the admin frontend. All work is in `claoj-web/`. Run `npm ci` there first (the worktree has no `node_modules`).

**Files:**
- Modify: `claoj-web/src/lib/adminApi.ts`
- Modify: `claoj-web/src/types/` (the file that declares `AdminJudgeUpdateRequest`)
- Modify: `claoj-web/src/app/[locale]/admin/judges/page.tsx`
- Modify: `claoj-web/src/app/[locale]/admin/users/page.tsx`
- Modify: `claoj-web/src/i18n/en.json`, `claoj-web/src/i18n/vi.json`
- Test: a jest test under `claoj-web/__tests__/lib/` for the new API client methods

**Interfaces:**
- Consumes (backend, already implemented):
  - `POST /admin/judges` body `{ name: string; auth_key?: string; description?: string }` → 201 `{ id: number; name: string; auth_key: string }`. 400 and 409 carry `{ error: string }` with a message safe to show.
  - `DELETE /admin/judge/:id` → 200 `{ success: true }`; 409 `{ error }` when the judge is online.
  - `PATCH /admin/judge/:id` additionally accepts `name?: string` and `auth_key?: string`.
  - `POST /admin/users` body `{ username: string; email: string; password: string; full_name?: string; is_active?: boolean }` → 201 `{ success: true; user: { id: number; username: string } }`. 400 and 409 carry `{ error: string }` safe to show.
  - `POST /admin/user/:id/password` body `{ password: string }` → 200 `{ success: true }`; 400 `{ error }` for a weak password; 403 `{ error }` when the target is staff and the viewer is not a superuser.
  - `POST /admin/user/:id/ban` body `{ reason: string }` — `day` is no longer required and is ignored; bans have no duration.
- Produces: `adminJudgeApi.create`, `adminJudgeApi.delete`, `adminUserApi.create`, `adminUserApi.setPassword`.

Password rule the backend enforces (show it as a hint under the password inputs): at least 8 characters with an uppercase letter, a lowercase letter and a digit.

- [ ] **Step 1: API client and types**

In the types file add:

```ts
export interface AdminJudgeCreateRequest {
    name: string;
    auth_key?: string;
    description?: string;
}

export interface AdminUserCreateRequest {
    username: string;
    email: string;
    password: string;
    full_name?: string;
    is_active?: boolean;
}
```

and add `name?: string;` and `auth_key?: string;` to `AdminJudgeUpdateRequest`.

In `src/lib/adminApi.ts` add to `adminJudgeApi`:

```ts
    create: (data: AdminJudgeCreateRequest) =>
        api.post<{ id: number; name: string; auth_key: string }>('/admin/judges', data),

    delete: (id: number) =>
        api.delete<{ success: boolean }>(`/admin/judge/${id}`),
```

add to `adminUserApi`:

```ts
    create: (data: AdminUserCreateRequest) =>
        api.post<{ success: boolean; user: { id: number; username: string } }>('/admin/users', data),

    setPassword: (id: number, password: string) =>
        api.post<{ success: boolean }>(`/admin/user/${id}/password`, { password }),
```

and change `adminUserApi.ban` to `(id: number, reason: string) => api.post(`/admin/user/${id}/ban`, { reason })`, updating its callers.

- [ ] **Step 2: Test the client methods first**

Look at the existing tests in `claoj-web/__tests__/lib/` for how `@/lib/api` is mocked, and add a test file for the four new methods and the changed `ban`, asserting the HTTP method, URL and payload of each. Run it and watch it fail before Step 1's code exists if you write the test first (preferred), or confirm it would fail by temporarily breaking one URL.

- [ ] **Step 3: Judges page**

In `src/app/[locale]/admin/judges/page.tsx`:

- Add an "Add judge" button in the header button row. It opens a dialog built from the existing `src/components/ui/Dialog.tsx` with: name (required, max 50), auth key (optional; placeholder text says a key is generated when left blank), description. Follow `src/components/admin/OrganizationEditModal.tsx` for how this codebase structures a modal with a mutation.
- On success, keep the dialog open in a "created" state that shows the returned `auth_key` in a read-only field with a copy button and one line saying the judge agent signs in with this name and key. Closing the dialog invalidates the `['admin-judges']` query.
- Add a delete action per judge row using `src/components/ui/ConfirmDialog.tsx`. Disable it while `judge.online` is true, with a `title` explaining the judge must be offline. On success invalidate `['admin-judges']`.
- Show the backend's `error` string from a 400/409 response inside the dialog; fall back to a generic translated message when the response has none.

- [ ] **Step 4: Users page**

In `src/app/[locale]/admin/users/page.tsx`:

- Add a "Create user" button that opens a dialog with username, email, password, full name and an "Active" switch (default on). On success invalidate the users list query and close.
- Add a "Set password" action for a user (next to the existing row or modal actions) that opens a small dialog with one password input; on success show the page's existing success feedback and close. Show a 403/400 `error` string inline.
- If the ban UI has a days/duration input, remove it together with its state and translation keys; send only the reason.

- [ ] **Step 5: Translations**

Every new user-visible string goes through `useTranslations('Admin')`. Add the keys to both `src/i18n/en.json` and `src/i18n/vi.json` under the existing `Admin.judges` and `Admin.users` objects, with real Vietnamese text in `vi.json` that matches the terminology already used in that file. Remove keys that are no longer used.

- [ ] **Step 6: Verify**

Run from `claoj-web/`:

```bash
npx tsc --noEmit
npm run lint
npm run i18n:check
npm run ui:check
npm test
```

Expected: all five succeed. If a check already fails on a file you did not touch, say so in your report with the output instead of fixing it.

- [ ] **Step 7: Commit**

```bash
git add claoj-web
git commit -F - <<'EOF'
feat(web): create and delete judges, create users and set passwords in admin

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 7: Admin UI for deleting submissions and tickets

Expose Task 4 and Task 5's ticket delete in the admin frontend. All work is in `claoj-web/` (`node_modules` is already installed by Task 6; run `npm ci` if it is missing).

**Files:**
- Modify: `claoj-web/src/lib/adminApi.ts`
- Modify: `claoj-web/src/app/[locale]/admin/submissions/page.tsx`
- Modify: `claoj-web/src/app/[locale]/admin/tickets/[id]/page.tsx`
- Modify: `claoj-web/src/i18n/en.json`, `claoj-web/src/i18n/vi.json`
- Test: extend the jest test file Task 6 added under `claoj-web/__tests__/lib/`

**Interfaces:**
- Consumes (backend, already implemented):
  - `DELETE /admin/submission/:id` → 200 `{ success: true; message: string }`; 404; 409 `{ error: "submission is still being judged; abort it first" }`.
  - `DELETE /admin/ticket/:id` → 200 `{ message: string }`; 404.
- Produces: `adminSubmissionApi.delete(id: number)`, `adminTicketApi.delete(id: number)`.

- [ ] **Step 1: API client, test first**

Add a failing jest test for the two methods (HTTP method and URL), then add to `adminSubmissionApi`:

```ts
    delete: (id: number) =>
        api.delete<{ success: boolean; message: string }>(`/admin/submission/${id}`),
```

and to `adminTicketApi`:

```ts
    delete: (id: number) =>
        api.delete<{ message: string }>(`/admin/ticket/${id}`),
```

- [ ] **Step 2: Submissions page**

In `src/app/[locale]/admin/submissions/page.tsx` add a delete action per row, next to the existing rejudge/abort actions and styled like them. It opens `src/components/ui/ConfirmDialog.tsx`; the confirmation text says the submission is removed permanently and the user's points are recalculated. On success refresh the list the same way the page's other mutations do. On a 409 show the backend's `error` string using the page's existing error feedback.

- [ ] **Step 3: Ticket detail page**

In `src/app/[locale]/admin/tickets/[id]/page.tsx` add a "Delete ticket" button, visually separated from the routine actions as a destructive action. It opens `ConfirmDialog`; on success navigate to the admin tickets list using the navigation helper the page already imports. Show a failure with the page's existing error feedback.

- [ ] **Step 4: Translations**

Add the new strings to both `src/i18n/en.json` and `src/i18n/vi.json` under the existing `Admin.submissions` and `Admin.tickets` objects (use whatever those objects are actually named in the file), with real Vietnamese text.

- [ ] **Step 5: Verify**

Run from `claoj-web/`:

```bash
npx tsc --noEmit
npm run lint
npm run i18n:check
npm run ui:check
npm test
```

Expected: all five succeed. If a check already fails on a file you did not touch, say so in your report with the output instead of fixing it.

- [ ] **Step 6: Commit**

```bash
git add claoj-web
git commit -F - <<'EOF'
feat(web): delete submissions and tickets from the admin panel

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```
