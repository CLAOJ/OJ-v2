# Admin Sub-project A (Fix Broken Wiring) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the v2 admin problem/contest edit forms, list search/filters, and MOSS page actually work end to end.

**Architecture:** Backend (Go, gin + GORM over the shared Django MySQL schema) gets full-replacement many-to-many updates through one small `relations.Sync` helper, snake_case flat detail responses, real list filters, and a real MOSS TCP client plus a contest MOSS background job writing v1's `judge_contestmoss` table. Frontend (Next.js 16, React Query) gets pure mapper functions between API shapes and form data, a search-as-you-type `EntityPicker`, URL-backed list filters, and a rewritten MOSS page.

**Tech Stack:** Go 1.25, gin, GORM (MySQL prod / SQLite in tests), asynq, go-redis; Next.js 16, React 19, TypeScript, @tanstack/react-query, react-hook-form, next-intl, Jest + Testing Library.

**Spec:** `docs/superpowers/specs/2026-10-09-admin-panel-parity-design.md` (sub-project A, plus the "Pattern: full-replacement lists" section).

## Global Constraints

- v1 (Django) and v2 share one MySQL database that Django owns. **No schema changes**; only existing tables/columns are read or written, the way Django writes them.
- Many-to-many update rule: field present (including `[]`) → replace with exactly that set; field absent or `null` → unchanged; legacy `add_*/remove_*` keep working; if both are sent for one relation the full list wins; unknown ids → 400 and nothing is saved.
- Every multi-step update runs in one DB transaction.
- `judge_contest.time_limit` is microseconds (Django `DurationField`). The admin API speaks **seconds**; the contest form speaks **minutes**.
- `judge_contestsubmission.problem_id` references `judge_contestproblem.id`; existing contest-problem rows are updated in place, never deleted and re-created.
- LIKE searches escape `%`, `_`, `!` and use `ESCAPE '!'` (works identically in MySQL and SQLite).
- Admin list endpoints keep returning `{ data, total, page, page_size }`.
- All user-visible strings go through next-intl in both `src/i18n/en.json` and `src/i18n/vi.json`; `npm run i18n:check` and `npm run i18n:usage` must pass.
- Go tests use in-memory SQLite. Every new test DB calls `sqlDB.SetMaxOpenConns(1)` (transactions on `:memory:` otherwise get a fresh empty database on a second connection). Contest test DBs must drop and recreate `judge_contestproblem` after `AutoMigrate` (the `Contest.Problems` many2many otherwise leaves a bare join table without `id`/`points`).
- Commit messages: imperative, prefixed `feat(admin):` / `fix(admin):` / `test(admin):`, ending with the attribution line `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. In Git Bash use `git commit -F - <<'EOF'` heredocs, never PowerShell here-strings. Verify with `git log -1 --format=%s`.

## Review Focus

1. **Saving a contest after removing a problem that already has contest submissions** must return 409 naming the problem and change nothing (not orphan `judge_contestsubmission` rows). → Task 4 test `TestReplaceContestProblems_RefusesRemovingProblemWithSubmissions`.
2. **Opening an edit form and saving without touching relations** must not wipe authors/types/languages/staff: the form must be seeded with exactly the ids the detail returned. → Task 7 tests `problemToFormData keeps every relation id` and `contestToFormData keeps every relation id and shows minutes`; Task 8 Step 7 renders each edit form only after its detail has loaded, so `defaultValues` are never seeded from `undefined`.
3. **Search terms containing `%` or `_`** (e.g. problem code `a_b`) must match literally, not as wildcards. → Task 1 test `TestContainsPattern_EscapesWildcards` and Task 5 test `TestListProblems_SearchIsLiteral`.
4. **Contest times round-trip in the staff member's local time zone**: a value typed into `datetime-local`, saved, and reloaded shows the same wall-clock time. → Task 7 test `local datetime round-trips through ISO`.
5. **MOSS server rejects a language or drops the connection mid-run**: the run fails visibly and the previous `judge_contestmoss` rows stay intact. → Task 10 test `TestSubmit_LanguageRejected` and Task 11 test `TestRunContestMoss_FailureKeepsOldRows`.

## Spec deltas found while planning (already reflected in the tasks)

- The problem and contest **detail** endpoints currently return PascalCase JSON nested under `problem` / `contest`, so both edit forms load empty and `PdfTab` never sees `pdf_url`. A1 fixes the detail contract (flat, snake_case).
- `UpdateContest` accepts `start_time`/`end_time` but never applies them; contest **create** fails for every request because the form sends `datetime-local` strings to an RFC 3339 parser; the time-limit input posts a string to an `int64` field and is unit-mismatched (seconds vs microseconds). All fixed in A1.
- Form pickers fetch `page_size=1000` but services cap pages at 100, so only the first 100 users/problems are selectable. A1 replaces them with a search-as-you-type `EntityPicker`.
- Contest problem order is the position in the submitted `problems` list (no separate `order` field to conflict).

## File Structure

Backend (`OJ-v2/claoj/`):

| File | Responsibility |
|---|---|
| `db/like.go` (new) | `ContainsPattern` — escaped LIKE pattern |
| `service/relations/relations.go` (new) | `Sync[T]` full-replace / legacy add-remove; `UserRef`, `UserRefs` |
| `service/problem/problem_service.go` | JSON tags, detail preloads, update with full lists + group, list filters |
| `service/problem/errors.go` | `ErrInvalidGroup` |
| `service/contest/types.go` | JSON tags, `time_limit` seconds, update fields, flat detail type |
| `service/contest/contest_service.go` | update rewrite (transaction, schedule, format, lists), list filters |
| `service/contest/contest_problem_service.go` | `ReplaceContestProblems`, legacy merge, preload fix |
| `service/contest/errors.go` | new errors |
| `contest_format/registry.go` | `IsRegistered` |
| `service/organization/organization_service.go` | `Search` filter |
| `service/submission/submission_service.go` | user/problem/result/status/language filters |
| `api/v2/admin_problem.go`, `api/v2/admin_contest.go`, `api/v2/admin_judge_organization.go`, `api/v2/admin_submission.go` | bind new fields / query params |
| `moss/moss.go` (replaced) | MOSS TCP protocol client |
| `models/contest.go` | `ContestMoss` model |
| `jobs/moss.go` (new) | `RunContestMoss`, enqueue, running flag, task handler |
| `jobs/server.go` | register task |
| `api/v2/admin_contest_moss.go` (new), `api/v2/moss.go` (deleted) | contest MOSS endpoints |
| `api/router.go` | routes |

Frontend (`OJ-v2/claoj-web/src/`):

| File | Responsibility |
|---|---|
| `types/index.ts` | `UserRef`, `AdminProblemDetail`, `AdminContestDetail`, `AdminContestProblemRow`, `ContestProblemInput`, `AdminContestUpdateRequest` |
| `lib/adminApi.ts` | typed detail/update, list filters, `adminMossApi` |
| `lib/adminForms.ts` (new) | pure mappers: detail → form data, form → request, local datetime ↔ ISO |
| `lib/adminSearch.ts` (new) | `searchUsers`, `searchProblems`, `searchContests` for pickers |
| `hooks/useListParams.ts` (new) | URL-backed page + filters |
| `components/admin/EntityPicker.tsx` (new) | search-as-you-type multi-select with chips |
| `components/admin/ProblemForm.tsx`, `problem-form/AuthorsSection.tsx` | author picker |
| `components/admin/ContestForm.tsx`, `contest-form/StaffSection.tsx`, `contest-form/ProblemsSection.tsx`, `contest-form/ScheduleSection.tsx` | pickers, ISO + minutes conversion |
| `app/[locale]/admin/problems/[code]/edit/page.tsx`, `app/[locale]/admin/contests/[key]/edit/page.tsx`, `app/[locale]/admin/contests/create/page.tsx` | use mappers, stay on page with toast |
| `app/[locale]/admin/{problems,contests,organizations,submissions}/page.tsx` | server filters + `PaginationBar` |
| `app/[locale]/admin/layout.tsx` | `<Suspense>` around children (for `useSearchParams`) |
| `app/[locale]/admin/moss/page.tsx` (rewritten) | contest MOSS page |
| `i18n/en.json`, `i18n/vi.json` | new keys |

Tests: `db/like_test.go`, `service/relations/relations_test.go`, `service/problem/problem_service_test.go` (extended), `service/contest/contest_admin_test.go` (new), `service/organization/organization_search_test.go` (new), `service/submission/submission_list_test.go` (new), `api/v2/admin_detail_test.go` (new), `moss/moss_test.go` (new), `jobs/moss_test.go` (new), `api/v2/admin_contest_moss_test.go` (new); frontend `__tests__/lib/adminForms.test.ts`, `__tests__/hooks/listParams.test.ts`, `__tests__/components/EntityPicker.test.tsx`.

## Prerequisite

- [ ] **Step 0: Start from a clean branch.** `main` in `OJ-v2` currently has uncommitted UI work from the previous session (theme + admin shell). Ask the user whether to commit it first; then create the work branch:

```bash
cd /f/Coding/CLAOJ/OJ-v2
git status --short          # must be empty (or only the plan/spec docs) before branching
git switch -c feat/admin-a-broken-wiring
```

Commands below assume `cd /f/Coding/CLAOJ/OJ-v2/claoj` for Go and `cd /f/Coding/CLAOJ/OJ-v2/claoj-web` for the web app.

---

### Task 1: Shared backend helpers (`ContainsPattern`, `relations.Sync`)

**Files:**
- Create: `claoj/db/like.go`, `claoj/db/like_test.go`
- Create: `claoj/service/relations/relations.go`, `claoj/service/relations/relations_test.go`

**Interfaces:**
- Produces: `db.ContainsPattern(term string) string`; `relations.Sync[T any](tx *gorm.DB, owner any, name string, full *[]uint, add, remove []uint) error`; `relations.ErrUnknownID`; `relations.UserRef{ID uint; Username string}` (JSON `id`, `username`); `relations.UserRefs(profiles []models.Profile) []relations.UserRef`.

- [ ] **Step 1: Write the failing `ContainsPattern` test**

`claoj/db/like_test.go`:

```go
package db

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestContainsPattern_EscapesWildcards(t *testing.T) {
	require.Equal(t, "%abc%", ContainsPattern("abc"))
	require.Equal(t, "%a!_b%", ContainsPattern("a_b"))
	require.Equal(t, "%100!%%", ContainsPattern("100%"))
	require.Equal(t, "%x!!y%", ContainsPattern("x!y"))

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.Exec("CREATE TABLE t (code TEXT)").Error)
	require.NoError(t, database.Exec("INSERT INTO t (code) VALUES ('a_b'), ('axb'), ('100%'), ('1000')").Error)

	var got []string
	require.NoError(t, database.Raw("SELECT code FROM t WHERE code LIKE ? ESCAPE '!' ORDER BY code", ContainsPattern("a_b")).Scan(&got).Error)
	require.Equal(t, []string{"a_b"}, got)

	got = nil
	require.NoError(t, database.Raw("SELECT code FROM t WHERE code LIKE ? ESCAPE '!' ORDER BY code", ContainsPattern("100%")).Scan(&got).Error)
	require.Equal(t, []string{"100%"}, got)
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./db/ -run TestContainsPattern -count=1`
Expected: FAIL — `undefined: ContainsPattern`.

- [ ] **Step 3: Implement `ContainsPattern`**

`claoj/db/like.go`:

```go
package db

import "strings"

var likeEscaper = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")

// ContainsPattern returns a LIKE pattern that matches term anywhere, with
// LIKE wildcards in term matched literally. Always pair it with
// `LIKE ? ESCAPE '!'`: '!' (not '\') is the escape character that behaves
// the same in MySQL and in the SQLite test database.
func ContainsPattern(term string) string {
	return "%" + likeEscaper.Replace(term) + "%"
}
```

- [ ] **Step 4: Run it to verify it passes**

Run: `go test ./db/ -run TestContainsPattern -count=1`
Expected: PASS.

- [ ] **Step 5: Write the failing `relations.Sync` test**

`claoj/service/relations/relations_test.go`:

```go
package relations

import (
	"testing"

	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupDB(t *testing.T) (*gorm.DB, models.Problem, []models.ProblemType) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.ProblemType{}, &models.Problem{}))

	p := models.Problem{Code: "p1", Name: "P1"}
	require.NoError(t, database.Omit(clause.Associations).Create(&p).Error)
	types := []models.ProblemType{{Name: "dp"}, {Name: "graph"}, {Name: "math"}}
	for i := range types {
		require.NoError(t, database.Create(&types[i]).Error)
	}
	return database, p, types
}

func typeIDs(t *testing.T, database *gorm.DB, p models.Problem) []uint {
	t.Helper()
	var loaded models.Problem
	require.NoError(t, database.Preload("Types").First(&loaded, p.ID).Error)
	ids := []uint{}
	for _, ty := range loaded.Types {
		ids = append(ids, ty.ID)
	}
	return ids
}

func TestSync_FullListReplaces(t *testing.T) {
	database, p, types := setupDB(t)
	full := []uint{types[0].ID, types[1].ID}
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", &full, nil, nil))
	require.ElementsMatch(t, full, typeIDs(t, database, p))

	full = []uint{types[2].ID, types[2].ID} // duplicates are collapsed
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", &full, nil, nil))
	require.Equal(t, []uint{types[2].ID}, typeIDs(t, database, p))
}

func TestSync_EmptyListClears(t *testing.T) {
	database, p, types := setupDB(t)
	full := []uint{types[0].ID}
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", &full, nil, nil))
	empty := []uint{}
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", &empty, nil, nil))
	require.Empty(t, typeIDs(t, database, p))
}

func TestSync_NilFullUsesLegacyAddRemove(t *testing.T) {
	database, p, types := setupDB(t)
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", nil, []uint{types[0].ID, types[1].ID}, nil))
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", nil, nil, []uint{types[0].ID}))
	require.Equal(t, []uint{types[1].ID}, typeIDs(t, database, p))
}

func TestSync_FullListBeatsAddRemove(t *testing.T) {
	database, p, types := setupDB(t)
	full := []uint{types[2].ID}
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", &full, []uint{types[0].ID}, nil))
	require.Equal(t, []uint{types[2].ID}, typeIDs(t, database, p))
}

func TestSync_UnknownIDIsAnErrorAndChangesNothing(t *testing.T) {
	database, p, types := setupDB(t)
	full := []uint{types[0].ID}
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", &full, nil, nil))

	bad := []uint{types[1].ID, 9999}
	err := database.Transaction(func(tx *gorm.DB) error {
		return Sync[models.ProblemType](tx, &p, "Types", &bad, nil, nil)
	})
	require.ErrorIs(t, err, ErrUnknownID)
	require.Equal(t, []uint{types[0].ID}, typeIDs(t, database, p))
}

func TestUserRefs(t *testing.T) {
	refs := UserRefs([]models.Profile{
		{ID: 4, User: models.AuthUser{Username: "haruxne"}},
		{ID: 9, User: models.AuthUser{Username: "NK"}},
	})
	require.Equal(t, []UserRef{{ID: 4, Username: "haruxne"}, {ID: 9, Username: "NK"}}, refs)
	require.NotNil(t, UserRefs(nil)) // marshals as [] not null
}
```

- [ ] **Step 6: Run it to verify it fails**

Run: `go test ./service/relations/ -count=1`
Expected: FAIL — package has no non-test Go files / `undefined: Sync`.

- [ ] **Step 7: Implement `relations`**

`claoj/service/relations/relations.go`:

```go
// Package relations applies many-to-many edits for the admin update
// handlers and renders related users as {id, username} refs.
package relations

import (
	"errors"
	"fmt"

	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// ErrUnknownID is returned when a full list names an id that has no row,
// so a stale form or typo cannot silently drop a relation.
var ErrUnknownID = errors.New("unknown id")

// Sync updates owner's association `name` (rows of type T).
//
//   - full != nil: replace the association with exactly those ids; an
//     empty list clears it. add/remove are ignored.
//   - full == nil: apply the legacy add/remove lists.
//
// Run it inside the caller's transaction.
func Sync[T any](tx *gorm.DB, owner any, name string, full *[]uint, add, remove []uint) error {
	if full != nil {
		ids := unique(*full)
		if len(ids) == 0 {
			return tx.Model(owner).Association(name).Clear()
		}
		rows, err := find[T](tx, ids)
		if err != nil {
			return err
		}
		if len(rows) != len(ids) {
			return fmt.Errorf("%w in %s", ErrUnknownID, name)
		}
		return tx.Model(owner).Association(name).Replace(&rows)
	}
	if len(add) > 0 {
		rows, err := find[T](tx, unique(add))
		if err != nil {
			return err
		}
		if err := tx.Model(owner).Association(name).Append(&rows); err != nil {
			return err
		}
	}
	if len(remove) > 0 {
		rows, err := find[T](tx, unique(remove))
		if err != nil {
			return err
		}
		if err := tx.Model(owner).Association(name).Delete(&rows); err != nil {
			return err
		}
	}
	return nil
}

func find[T any](tx *gorm.DB, ids []uint) ([]T, error) {
	var rows []T
	err := tx.Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}

func unique(ids []uint) []uint {
	seen := make(map[uint]bool, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// UserRef is a related profile as the admin forms display it.
type UserRef struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// UserRefs maps profiles (preloaded with User) to refs; never nil.
func UserRefs(profiles []models.Profile) []UserRef {
	refs := make([]UserRef, len(profiles))
	for i, p := range profiles {
		refs[i] = UserRef{ID: p.ID, Username: p.User.Username}
	}
	return refs
}
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test ./db/ ./service/relations/ -count=1`
Expected: PASS (all tests).

- [ ] **Step 9: Commit**

```bash
git add claoj/db/like.go claoj/db/like_test.go claoj/service/relations
git commit -F - <<'EOF'
feat(admin): add escaped LIKE pattern and many-to-many sync helpers

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 2: Problem detail contract and full-list update (backend)

**Files:**
- Modify: `claoj/service/problem/problem_service.go` (ProblemProfile 14-46, UpdateProblemRequest 66-83, GetProblem 191-206, UpdateProblem 259-345, problemToProfile 474-512)
- Modify: `claoj/service/problem/errors.go`
- Modify: `claoj/api/v2/admin_problem.go` (AdminProblemDetail, AdminProblemUpdate)
- Test: `claoj/service/problem/problem_service_test.go` (extend), `claoj/api/v2/admin_detail_test.go` (new)

**Interfaces:**
- Consumes: `relations.Sync`, `relations.UserRef`, `relations.UserRefs` (Task 1).
- Produces: `GET /admin/problem/:code` → flat JSON `ProblemProfile` with keys `id, code, name, description, pdf_url, group_id, group_name, time_limit, memory_limit, short_circuit, points, partial, is_public, is_manually_managed, date, license_id, og_image, summary, user_count, ac_rate, is_full_markup, submission_source_visibility_mode, testcase_visibility_mode, is_organization_private, suggester_id, author_ids, curator_ids, tester_ids, type_ids, allowed_lang_ids, organization_ids, authors, curators, testers` (`authors|curators|testers` are `[{id, username}]`; all arrays are `[]` never `null`). `PATCH /admin/problem/:code` accepts `group_id`, `author_ids`, `curator_ids`, `tester_ids`, `type_ids`, `allowed_lang_ids` (full lists) in addition to existing fields; unknown ids / group → 400.

- [ ] **Step 1: Write failing service tests**

In `claoj/service/problem/problem_service_test.go`, change `SetupTest` so the DB uses one connection and migrates `AuthUser`:

```go
func (s *ProblemServiceTestSuite) SetupTest() {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s.Require().NoError(err)
	sqlDB, err := database.DB()
	s.Require().NoError(err)
	sqlDB.SetMaxOpenConns(1)

	s.database = database
	db.DB = database

	// Migrate schema
	err = database.AutoMigrate(
		&models.AuthUser{},
		&models.Problem{},
		&models.ProblemGroup{},
		&models.ProblemType{},
		&models.Profile{},
		&models.Language{},
		&models.Organization{},
		&models.ProblemClarification{},
	)
	s.Require().NoError(err)

	s.service = NewProblemService()
}
```

Append these tests to the same file:

```go
func (s *ProblemServiceTestSuite) TestUpdateProblem_FullListsReplace() {
	group := s.createTestGroup("g1")
	p := s.createTestProblem("FULL1", "Full", group.ID)
	a1 := s.createTestProfile("alice")
	a2 := s.createTestProfile("bob")
	ty := s.createTestType("dp")
	lang := s.createTestLanguage("C++17", "CPP17")

	authors := []uint{a1.ID, a2.ID}
	types := []uint{ty.ID}
	langs := []uint{lang.ID}
	curators := []uint{a2.ID}
	_, err := s.service.UpdateProblem(UpdateProblemRequest{
		ProblemCode: p.Code, AuthorIDs: &authors, TypeIDs: &types, AllowedLangIDs: &langs, CuratorIDs: &curators,
	})
	s.Require().NoError(err)

	got, err := s.service.GetProblem(GetProblemRequest{ProblemCode: p.Code})
	s.Require().NoError(err)
	s.ElementsMatch(authors, got.AuthorIDs)
	s.Equal(types, got.TypeIDs)
	s.Equal(langs, got.AllowedLangIDs)
	s.Equal(curators, got.CuratorIDs)
	s.ElementsMatch([]string{"alice", "bob"}, []string{got.Authors[0].Username, got.Authors[1].Username})

	authors = []uint{a2.ID}
	_, err = s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: p.Code, AuthorIDs: &authors})
	s.Require().NoError(err)
	got, _ = s.service.GetProblem(GetProblemRequest{ProblemCode: p.Code})
	s.Equal([]uint{a2.ID}, got.AuthorIDs)
	s.Equal(types, got.TypeIDs, "omitted type_ids must stay unchanged")
	s.Equal(langs, got.AllowedLangIDs, "omitted allowed_lang_ids must stay unchanged")
}

func (s *ProblemServiceTestSuite) TestUpdateProblem_EmptyListClears() {
	group := s.createTestGroup("g2")
	p := s.createTestProblem("CLR1", "Clear", group.ID)
	a := s.createTestProfile("carol")
	authors := []uint{a.ID}
	_, err := s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: p.Code, AuthorIDs: &authors})
	s.Require().NoError(err)

	none := []uint{}
	_, err = s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: p.Code, AuthorIDs: &none})
	s.Require().NoError(err)
	got, _ := s.service.GetProblem(GetProblemRequest{ProblemCode: p.Code})
	s.Empty(got.AuthorIDs)
	s.NotNil(got.AuthorIDs)
}

func (s *ProblemServiceTestSuite) TestUpdateProblem_UnknownIDRollsBackEverything() {
	group := s.createTestGroup("g3")
	p := s.createTestProblem("RB1", "Before", group.ID)
	name := "After"
	bad := []uint{424242}
	_, err := s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: p.Code, Name: &name, AuthorIDs: &bad})
	s.Require().Error(err)

	got, _ := s.service.GetProblem(GetProblemRequest{ProblemCode: p.Code})
	s.Equal("Before", got.Name, "a failed relation must roll back the scalar update too")
}

func (s *ProblemServiceTestSuite) TestUpdateProblem_GroupID() {
	g1 := s.createTestGroup("g4")
	g2 := s.createTestGroup("g5")
	p := s.createTestProblem("GRP1", "Group", g1.ID)

	_, err := s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: p.Code, GroupID: &g2.ID})
	s.Require().NoError(err)
	got, _ := s.service.GetProblem(GetProblemRequest{ProblemCode: p.Code})
	s.Equal(g2.ID, got.GroupID)

	missing := uint(98765)
	_, err = s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: p.Code, GroupID: &missing})
	s.ErrorIs(err, ErrInvalidGroup)
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./service/problem/ -count=1`
Expected: FAIL to compile — `unknown field AuthorIDs in struct literal of type UpdateProblemRequest` (and `CuratorIDs`, `ErrInvalidGroup`).

- [ ] **Step 3: Implement the service changes**

In `claoj/service/problem/errors.go` add to the `var (...)` block:

```go
	ErrInvalidGroup          = errors.New("problem: group does not exist")
```

In `claoj/service/problem/problem_service.go`:

Add `"github.com/CLAOJ/claoj/service/relations"` to the imports.

Replace the `ProblemProfile` struct with:

```go
// ProblemProfile is the admin view of a problem. JSON is flat snake_case;
// the admin edit form and PdfTab read it directly.
type ProblemProfile struct {
	ID                             uint                 `json:"id"`
	Code                           string               `json:"code"`
	Name                           string               `json:"name"`
	Source                         string               `json:"source"`
	Description                    string               `json:"description"`
	PdfURL                         string               `json:"pdf_url"`
	GroupID                        uint                 `json:"group_id"`
	GroupName                      string               `json:"group_name"`
	TimeLimit                      float64              `json:"time_limit"`
	MemoryLimit                    uint                 `json:"memory_limit"`
	ShortCircuit                   bool                 `json:"short_circuit"`
	Points                         float64              `json:"points"`
	Partial                        bool                 `json:"partial"`
	IsPublic                       bool                 `json:"is_public"`
	IsManuallyManaged              bool                 `json:"is_manually_managed"`
	Date                           *time.Time           `json:"date"`
	LicenseID                      *uint                `json:"license_id"`
	OgImage                        string               `json:"og_image"`
	Summary                        string               `json:"summary"`
	UserCount                      int                  `json:"user_count"`
	AcRate                         float64              `json:"ac_rate"`
	IsFullMarkup                   bool                 `json:"is_full_markup"`
	SubmissionSourceVisibilityMode string               `json:"submission_source_visibility_mode"`
	TestcaseVisibilityMode         string               `json:"testcase_visibility_mode"`
	IsOrganizationPrivate          bool                 `json:"is_organization_private"`
	SuggesterID                    *uint                `json:"suggester_id"`
	AuthorIDs                      []uint               `json:"author_ids"`
	CuratorIDs                     []uint               `json:"curator_ids"`
	TesterIDs                      []uint               `json:"tester_ids"`
	TypeIDs                        []uint               `json:"type_ids"`
	AllowedLangIDs                 []uint               `json:"allowed_lang_ids"`
	OrganizationIDs                []uint               `json:"organization_ids"`
	Authors                        []relations.UserRef  `json:"authors"`
	Curators                       []relations.UserRef  `json:"curators"`
	Testers                        []relations.UserRef  `json:"testers"`
}
```

Replace `UpdateProblemRequest` with:

```go
// UpdateProblemRequest holds the parameters for updating a problem.
// Pointer-to-slice fields are full replacement lists (nil = unchanged);
// the Add*/Remove* slices are the legacy incremental form.
type UpdateProblemRequest struct {
	ProblemCode       string
	Name              *string
	Description       *string
	Points            *float64
	Partial           *bool
	IsPublic          *bool
	TimeLimit         *float64
	MemoryLimit       *uint
	IsManuallyManaged *bool
	PdfURL            *string
	GroupID           *uint
	AuthorIDs         *[]uint
	CuratorIDs        *[]uint
	TesterIDs         *[]uint
	TypeIDs           *[]uint
	AllowedLangIDs    *[]uint
	AddTypeIDs        []uint
	RemoveTypeIDs     []uint
	AddAuthorIDs      []uint
	RemoveAuthorIDs   []uint
	AddLangIDs        []uint
	RemoveLangIDs     []uint
}
```

Replace the body of `GetProblem` with:

```go
func (s *ProblemService) GetProblem(req GetProblemRequest) (*ProblemProfile, error) {
	var problem models.Problem
	if err := db.DB.Preload("Group").
		Preload("Types").
		Preload("Authors.User").
		Preload("Curators.User").
		Preload("Testers.User").
		Preload("AllowedLangs").
		Preload("Organizations").
		Where("code = ?", req.ProblemCode).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrProblemNotFound
		}
		return nil, err
	}

	profile := problemToProfile(problem, problem.Group.Name)
	return &profile, nil
}
```

Replace `UpdateProblem` entirely with:

```go
// UpdateProblem updates an existing problem in one transaction.
func (s *ProblemService) UpdateProblem(req UpdateProblemRequest) (*ProblemProfile, error) {
	var problem models.Problem
	if err := db.DB.Where("code = ?", req.ProblemCode).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrProblemNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = sanitization.SanitizeTitle(*req.Name)
	}
	if req.Description != nil {
		updates["description"] = sanitization.SanitizeProblemContent(*req.Description)
	}
	if req.Points != nil {
		updates["points"] = *req.Points
	}
	if req.Partial != nil {
		updates["partial"] = *req.Partial
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}
	if req.TimeLimit != nil {
		updates["time_limit"] = *req.TimeLimit
	}
	if req.MemoryLimit != nil {
		updates["memory_limit"] = *req.MemoryLimit
	}
	if req.IsManuallyManaged != nil {
		updates["is_manually_managed"] = *req.IsManuallyManaged
	}
	if req.PdfURL != nil {
		updates["pdf_url"] = *req.PdfURL
	}
	if req.GroupID != nil {
		var count int64
		if err := db.DB.Model(&models.ProblemGroup{}).Where("id = ?", *req.GroupID).Count(&count).Error; err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, ErrInvalidGroup
		}
		updates["group_id"] = *req.GroupID
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&problem).Updates(updates).Error; err != nil {
				return err
			}
		}
		if err := relations.Sync[models.ProblemType](tx, &problem, "Types", req.TypeIDs, req.AddTypeIDs, req.RemoveTypeIDs); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &problem, "Authors", req.AuthorIDs, req.AddAuthorIDs, req.RemoveAuthorIDs); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &problem, "Curators", req.CuratorIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &problem, "Testers", req.TesterIDs, nil, nil); err != nil {
			return err
		}
		return relations.Sync[models.Language](tx, &problem, "AllowedLangs", req.AllowedLangIDs, req.AddLangIDs, req.RemoveLangIDs)
	})
	if err != nil {
		return nil, err
	}

	return s.GetProblem(GetProblemRequest{ProblemCode: problem.Code})
}
```

In `problemToProfile`, add the three new id lists and the ref lists. Replace the first four lines of the function body and add the new fields to the literal:

```go
	authorIDs := getProfileIDs(p.Authors)
	curatorIDs := getProfileIDs(p.Curators)
	testerIDs := getProfileIDs(p.Testers)
	typeIDs := getTypeIDs(p.Types)
	langIDs := getLangIDs(p.AllowedLangs)
	orgIDs := getOrgIDs(p.Organizations)
```

and in the returned `ProblemProfile{...}` literal, after `AuthorIDs: authorIDs,` add:

```go
		CuratorIDs:                     curatorIDs,
		TesterIDs:                      testerIDs,
		Authors:                        relations.UserRefs(p.Authors),
		Curators:                       relations.UserRefs(p.Curators),
		Testers:                        relations.UserRefs(p.Testers),
```

(`getProfileIDs`, `getTypeIDs`, `getLangIDs`, `getOrgIDs` already return non-nil slices via `make`.)

- [ ] **Step 4: Run the service tests to verify they pass**

Run: `go test ./service/problem/ -count=1`
Expected: PASS (new and pre-existing tests).

- [ ] **Step 5: Write the failing handler test for the flat detail**

`claoj/api/v2/admin_detail_test.go`:

```go
package v2

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// setupAdminDetailDB migrates everything the problem and contest admin
// detail/update handlers touch. judge_contestproblem is recreated because
// Contest's many2many leaves a bare join table without id/points.
func setupAdminDetailDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.ContestTag{},
		&models.Contest{}, &models.ContestParticipation{}, &models.Submission{}, &models.ContestSubmission{},
	))
	require.NoError(t, database.Migrator().DropTable("judge_contestproblem"))
	require.NoError(t, database.Migrator().CreateTable(&models.ContestProblem{}))
	db.DB = database
	problemService = nil
	contestService = nil
}

func TestAdminProblemDetail_IsFlatSnakeCase(t *testing.T) {
	setupAdminDetailDB(t)
	u := models.AuthUser{Username: "alice", IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	prof := models.Profile{UserID: u.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&prof).Error)
	p := models.Problem{Code: "aplusb", Name: "A+B", PdfURL: "/pdf/aplusb.pdf"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	require.NoError(t, db.DB.Model(&p).Association("Authors").Append(&prof))

	router := authedRouter(u.ID, http.MethodGet, "/admin/problem/:code", AdminProblemDetail)
	w := doRequest(router, http.MethodGet, "/admin/problem/aplusb", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "aplusb", body["code"])
	require.Equal(t, "/pdf/aplusb.pdf", body["pdf_url"])
	require.Equal(t, []any{float64(prof.ID)}, body["author_ids"])
	require.Equal(t, []any{map[string]any{"id": float64(prof.ID), "username": "alice"}}, body["authors"])
	require.Equal(t, []any{}, body["type_ids"])
	require.NotContains(t, body, "problem", "detail must not be nested")
}

func TestAdminProblemUpdate_UnknownAuthorIs400(t *testing.T) {
	setupAdminDetailDB(t)
	p := models.Problem{Code: "p2", Name: "P2"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)

	router := authedRouter(1, http.MethodPatch, "/admin/problem/:code", AdminProblemUpdate)
	w := doRequest(router, http.MethodPatch, "/admin/problem/p2", `{"author_ids":[777]}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

var _ gin.HandlerFunc = AdminProblemDetail
```

- [ ] **Step 6: Run it to verify it fails**

Run: `go test ./api/v2/ -run 'TestAdminProblem' -count=1`
Expected: FAIL — `body["code"]` is nil because the response is `{"problem": {...}}`.

- [ ] **Step 7: Update the handlers**

In `claoj/api/v2/admin_problem.go`:

In `AdminProblemDetail`, replace the success response

```go
	c.JSON(http.StatusOK, gin.H{
		"problem": profile,
	})
```

with

```go
	c.JSON(http.StatusOK, profile)
```

(keep whatever else the function returns only if it is inside `gin.H` with `problem` — the whole response becomes `profile`).

In `AdminProblemUpdate`, add to the `input` struct:

```go
		GroupID           *uint    `json:"group_id"`
		AuthorIDs         *[]uint  `json:"author_ids"`
		CuratorIDs        *[]uint  `json:"curator_ids"`
		TesterIDs         *[]uint  `json:"tester_ids"`
		TypeIDs           *[]uint  `json:"type_ids"`
		AllowedLangIDs    *[]uint  `json:"allowed_lang_ids"`
```

and pass them through in the `problem.UpdateProblemRequest{...}` literal:

```go
		GroupID:           input.GroupID,
		AuthorIDs:         input.AuthorIDs,
		CuratorIDs:        input.CuratorIDs,
		TesterIDs:         input.TesterIDs,
		TypeIDs:           input.TypeIDs,
		AllowedLangIDs:    input.AllowedLangIDs,
```

(The existing error mapping already returns 404 for `ErrProblemNotFound` and 400 for everything else, which covers `relations.ErrUnknownID` and `problem.ErrInvalidGroup`.)

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test ./api/v2/ -run 'TestAdminProblem' -count=1 && go build ./...`
Expected: PASS; build succeeds.

- [ ] **Step 9: Commit**

```bash
git add claoj/service/problem claoj/api/v2/admin_problem.go claoj/api/v2/admin_detail_test.go
git commit -F - <<'EOF'
fix(admin): return flat problem detail and apply full relation lists on update

The edit form read fields off a nested PascalCase response and loaded
empty; authors, types, languages and group edits were silently dropped.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 3: Contest detail contract, schedule, format, staff and tags (backend)

**Files:**
- Modify: `claoj/service/contest/types.go` (ContestProfile, UpdateContestRequest, ContestDetailResponse)
- Modify: `claoj/service/contest/contest_service.go` (GetContest 73-98, UpdateContest 163-243, contestToProfile 466-513)
- Modify: `claoj/service/contest/contest_problem_service.go` (ContestProblemInfo 19-27, GetContestProblems 30-50)
- Modify: `claoj/service/contest/errors.go`
- Modify: `claoj/contest_format/registry.go`
- Modify: `claoj/api/v2/admin_contest.go` (AdminContestDetail, AdminContestCreate time limit, AdminContestUpdate)
- Test: `claoj/service/contest/contest_admin_test.go` (new), `claoj/api/v2/admin_detail_test.go` (extend)

**Interfaces:**
- Consumes: `relations.Sync`, `relations.UserRefs` (Task 1); `setupAdminDetailDB` (Task 2).
- Produces:
  - `GET /admin/contest/:key` → flat contest JSON (`id, key, name, description, summary, start_time, end_time, time_limit` **seconds or null**, `is_visible, is_rated, scoreboard_visibility, use_clarifications, push_announcements, rating_floor, rating_ceiling, rate_all, is_private, hide_problem_tags, hide_problem_authors, run_pretests_only, show_short_display, is_organization_private, access_code, format_name, format_config, locked_after, points_precision, user_count, author_ids, curator_ids, tester_ids, tag_ids, organization_ids, authors, curators, testers`) plus `problems: [{id, problem_id, code, name, points, partial, is_pretested, order}]`.
  - `PATCH /admin/contest/:key` accepts `start_time`, `end_time` (RFC 3339, now actually applied), `time_limit` (seconds; `0` clears), `format_name` (must be registered), `format_config` (JSON object string; `""` clears), `author_ids`, `curator_ids`, `tester_ids`, `tag_ids` (full lists). `update_time_limit` is removed.
  - `POST /admin/contests` `time_limit` is now seconds.
  - Service: `UpdateContestRequest` fields `FormatName *string`, `FormatConfig *string`, `AuthorIDs/CuratorIDs/TesterIDs/TagIDs *[]uint`, `TimeLimit *int64` (microseconds; `<=0` clears); errors `ErrInvalidTime`, `ErrInvalidSchedule`, `ErrUnknownFormat`.
  - `contest_format.IsRegistered(name string) bool`.
  - Test helper `setupContestDB(t) *gorm.DB` in package `contest`.

- [ ] **Step 1: Write failing service tests**

`claoj/service/contest/contest_admin_test.go`:

```go
package contest

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

func setupContestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.ContestTag{},
		&models.Contest{}, &models.ContestParticipation{}, &models.Submission{}, &models.ContestSubmission{},
	))
	require.NoError(t, database.Migrator().DropTable("judge_contestproblem"))
	require.NoError(t, database.Migrator().CreateTable(&models.ContestProblem{}))
	db.DB = database
	return database
}

func newContest(t *testing.T, key string) models.Contest {
	t.Helper()
	start := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	c := models.Contest{Key: key, Name: key, StartTime: start, EndTime: start.Add(3 * time.Hour), FormatName: "icpc"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)
	return c
}

func newProfile(t *testing.T, username string) models.Profile {
	t.Helper()
	u := models.AuthUser{Username: username, IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	p := models.Profile{UserID: u.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	return p
}

func newProblem(t *testing.T, code string) models.Problem {
	t.Helper()
	p := models.Problem{Code: code, Name: code}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	return p
}

func strp(s string) *string { return &s }
func i64p(v int64) *int64  { return &v }

func TestUpdateContest_AppliesSchedule(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "sched")
	svc := NewContestService()

	_, err := svc.UpdateContest(UpdateContestRequest{
		ContestKey: c.Key,
		StartTime:  strp("2026-11-01T09:00:00+07:00"),
		EndTime:    strp("2026-11-01T12:00:00+07:00"),
	})
	require.NoError(t, err)
	got, err := svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.NoError(t, err)
	require.True(t, got.StartTime.Equal(time.Date(2026, 11, 1, 2, 0, 0, 0, time.UTC)))
	require.True(t, got.EndTime.Equal(time.Date(2026, 11, 1, 5, 0, 0, 0, time.UTC)))

	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, EndTime: strp("2026-10-31T00:00:00Z")})
	require.ErrorIs(t, err, ErrInvalidSchedule)

	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, StartTime: strp("2026-11-01T09:00")})
	require.ErrorIs(t, err, ErrInvalidTime)
}

func TestUpdateContest_TimeLimitMicrosecondsAndClear(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "tl")
	svc := NewContestService()

	_, err := svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, TimeLimit: i64p(3 * 3600 * 1_000_000)})
	require.NoError(t, err)
	got, _ := svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.NotNil(t, got.TimeLimitSeconds)
	require.Equal(t, int64(10800), *got.TimeLimitSeconds)

	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, TimeLimit: i64p(0)})
	require.NoError(t, err)
	got, _ = svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.Nil(t, got.TimeLimitSeconds)
}

func TestUpdateContest_FormatValidation(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "fmt")
	svc := NewContestService()

	_, err := svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, FormatName: strp("ioi16"), FormatConfig: strp(`{"cumtime": true}`)})
	require.NoError(t, err)
	got, _ := svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.Equal(t, "ioi16", got.FormatName)
	require.Equal(t, true, got.FormatConfig["cumtime"])

	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, FormatName: strp("nope")})
	require.ErrorIs(t, err, ErrUnknownFormat)
	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, FormatConfig: strp(`[1,2]`)})
	require.ErrorIs(t, err, ErrInvalidFormatConfig)
}

func TestUpdateContest_StaffAndTagsFullLists(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "staff")
	a := newProfile(t, "alice")
	b := newProfile(t, "bob")
	tag := models.ContestTag{Name: "div2", Color: "#000000"}
	require.NoError(t, db.DB.Create(&tag).Error)
	svc := NewContestService()

	authors := []uint{a.ID, b.ID}
	testers := []uint{b.ID}
	tags := []uint{tag.ID}
	_, err := svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, AuthorIDs: &authors, TesterIDs: &testers, TagIDs: &tags})
	require.NoError(t, err)
	got, _ := svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.ElementsMatch(t, authors, got.AuthorIDs)
	require.Equal(t, testers, got.TesterIDs)
	require.Equal(t, tags, got.TagIDs)
	require.Equal(t, "bob", got.Testers[0].Username)

	name := "Renamed"
	bad := []uint{99999}
	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, Name: &name, CuratorIDs: &bad})
	require.Error(t, err)
	got, _ = svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.Equal(t, "staff", got.Name, "failed relation must roll back the rename")
}

func TestGetContest_ProblemsCarryCodeAndName(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "probs")
	p := newProblem(t, "aplusb")
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.ContestProblem{ContestID: c.ID, ProblemID: p.ID, Points: 100, Partial: true, Order: 1}).Error)

	got, err := NewContestService().GetContest(GetContestRequest{ContestKey: c.Key})
	require.NoError(t, err)
	require.Len(t, got.Problems, 1)
	require.Equal(t, "aplusb", got.Problems[0].Code)
}
```

(`models.ContestTag` fields: check the struct in `models/contest.go` around line 31 and set every non-pointer field the table requires; `Name` and `Color` are the common ones.)

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./service/contest/ -count=1`
Expected: FAIL to compile — unknown fields `FormatName`, `AuthorIDs`, `TimeLimitSeconds`, undefined `ErrInvalidSchedule`, `ErrInvalidTime`, `ErrUnknownFormat`.

- [ ] **Step 3: Add errors and the format registry check**

`claoj/service/contest/errors.go` — add inside `var (...)`:

```go
	ErrInvalidTime                  = errors.New("contest: start_time and end_time must be RFC 3339 timestamps")
	ErrInvalidSchedule              = errors.New("contest: end_time must be after start_time")
	ErrUnknownFormat                = errors.New("contest: unknown contest format")
	ErrUnknownProblem               = errors.New("contest: unknown or duplicate problem in problems list")
	ErrInvalidPoints                = errors.New("contest: problem points must not be negative")
	ErrContestProblemHasSubmissions = errors.New("contest: problem has contest submissions and cannot be removed")
```

(`ErrUnknownProblem`, `ErrInvalidPoints`, `ErrContestProblemHasSubmissions` are used in Task 4; declaring them now keeps the error list in one edit.)

`claoj/contest_format/registry.go` — add:

```go
// IsRegistered reports whether name is a known format. GetFormat silently
// falls back to "default", so validation has to ask explicitly.
func IsRegistered(name string) bool {
	_, ok := registry[name]
	return ok
}
```

- [ ] **Step 4: Update the contest types**

In `claoj/service/contest/types.go`:

Add imports `"github.com/CLAOJ/claoj/service/relations"`.

Replace `ContestProfile` with:

```go
// ContestProfile is the admin view of a contest (flat snake_case JSON).
// TimeLimit is the stored microseconds and never leaves the service;
// the API exposes TimeLimitSeconds as "time_limit".
type ContestProfile struct {
	ID                     uint                `json:"id"`
	Key                    string              `json:"key"`
	Name                   string              `json:"name"`
	Description            string              `json:"description"`
	Summary                string              `json:"summary"`
	StartTime              time.Time           `json:"start_time"`
	EndTime                time.Time           `json:"end_time"`
	TimeLimit              *int64              `json:"-"`
	TimeLimitSeconds       *int64              `json:"time_limit"`
	IsVisible              bool                `json:"is_visible"`
	IsRated                bool                `json:"is_rated"`
	ScoreboardVisibility   string              `json:"scoreboard_visibility"`
	ScoreboardCacheTimeout uint                `json:"scoreboard_cache_timeout"`
	UseClarifications      bool                `json:"use_clarifications"`
	PushAnnouncements      bool                `json:"push_announcements"`
	RatingFloor            *int                `json:"rating_floor"`
	RatingCeiling          *int                `json:"rating_ceiling"`
	RateAll                bool                `json:"rate_all"`
	IsPrivate              bool                `json:"is_private"`
	HideProblemTags        bool                `json:"hide_problem_tags"`
	HideProblemAuthors     bool                `json:"hide_problem_authors"`
	RunPretestsOnly        bool                `json:"run_pretests_only"`
	ShowShortDisplay       bool                `json:"show_short_display"`
	IsOrganizationPrivate  bool                `json:"is_organization_private"`
	OgImage                string              `json:"og_image"`
	LogoOverrideImage      string              `json:"logo_override_image"`
	UserCount              int                 `json:"user_count"`
	VirtualCount           int                 `json:"virtual_count"`
	AccessCode             string              `json:"access_code"`
	FormatName             string              `json:"format_name"`
	FormatConfig           models.JSONField    `json:"format_config"`
	ProblemLabelScript     string              `json:"problem_label_script"`
	LockedAfter            *time.Time          `json:"locked_after"`
	PointsPrecision        int                 `json:"points_precision"`
	AuthorIDs              []uint              `json:"author_ids"`
	CuratorIDs             []uint              `json:"curator_ids"`
	TesterIDs              []uint              `json:"tester_ids"`
	TagIDs                 []uint              `json:"tag_ids"`
	OrganizationIDs        []uint              `json:"organization_ids"`
	Authors                []relations.UserRef `json:"authors"`
	Curators               []relations.UserRef `json:"curators"`
	Testers                []relations.UserRef `json:"testers"`
}
```

Replace `UpdateContestRequest` with:

```go
// UpdateContestRequest holds the parameters for updating a contest.
// Pointer-to-slice fields are full replacement lists (nil = unchanged).
// TimeLimit is microseconds; a value <= 0 clears it.
type UpdateContestRequest struct {
	ContestKey            string
	Name                  *string
	Description           *string
	Summary               *string
	StartTime             *string
	EndTime               *string
	IsVisible             *bool
	IsRated               *bool
	AccessCode            *string
	HideProblemTags       *bool
	RunPretestsOnly       *bool
	IsOrganizationPrivate *bool
	TimeLimit             *int64
	FormatName            *string
	FormatConfig          *string
	AuthorIDs             *[]uint
	CuratorIDs            *[]uint
	TesterIDs             *[]uint
	TagIDs                *[]uint
	Problems              *[]ContestProblemInput
	AddProblemIDs         []uint
	RemoveProblemIDs      []uint
	AddTagIDs             []uint
	RemoveTagIDs          []uint
}
```

Replace `ContestDetailResponse` with:

```go
// ContestDetailResponse is the flat contest profile plus its problems.
type ContestDetailResponse struct {
	ContestProfile
	Problems []ContestProblemInfo `json:"problems"`
}
```

- [ ] **Step 5: Update the contest problem info and its loader**

In `claoj/service/contest/contest_problem_service.go` replace `ContestProblemInfo` and `GetContestProblems`:

```go
// ContestProblemInfo is one contest problem row as the admin sees it.
type ContestProblemInfo struct {
	ID          uint   `json:"id"`
	ProblemID   uint   `json:"problem_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Points      int    `json:"points"`
	Partial     bool   `json:"partial"`
	IsPretested bool   `json:"is_pretested"`
	Order       uint   `json:"order"`
}

// ContestProblemInput is one row of a full contest problem list. Its
// position in the list is its order. Omitted fields keep the existing
// row's value, or the defaults (100 points, partial) for a new row.
type ContestProblemInput struct {
	ProblemID   uint  `json:"problem_id"`
	Points      *int  `json:"points"`
	Partial     *bool `json:"partial"`
	IsPretested *bool `json:"is_pretested"`
}

// GetContestProblems retrieves all problems for a contest, in order.
func (s *ContestProblemService) GetContestProblems(contestID uint) ([]ContestProblemInfo, error) {
	var contestProblems []models.ContestProblem
	if err := db.DB.Preload("Problem").Where("contest_id = ?", contestID).Order("`order` ASC").Find(&contestProblems).Error; err != nil {
		return nil, err
	}

	problems := make([]ContestProblemInfo, len(contestProblems))
	for i, cp := range contestProblems {
		problems[i] = ContestProblemInfo{
			ID:          cp.ID,
			ProblemID:   cp.ProblemID,
			Code:        cp.Problem.Code,
			Name:        cp.Problem.Name,
			Points:      cp.Points,
			Partial:     cp.Partial,
			IsPretested: cp.IsPretested,
			Order:       cp.Order,
		}
	}
	return problems, nil
}
```

- [ ] **Step 6: Update `GetContest`, `UpdateContest`, `contestToProfile`**

In `claoj/service/contest/contest_service.go`:

Add imports `"strings"`, `"github.com/CLAOJ/claoj/contest_format"`, `"github.com/CLAOJ/claoj/service/relations"`.

Replace `GetContest` with:

```go
// GetContest retrieves a contest by key with full details.
func (s *ContestService) GetContest(req GetContestRequest) (*ContestDetailResponse, error) {
	var contest models.Contest
	if err := db.DB.Preload("Authors.User").
		Preload("Curators.User").
		Preload("Testers.User").
		Preload("Tags").
		Preload("Organizations").
		Where("`key` = ?", req.ContestKey).First(&contest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	problems, err := s.problems.GetContestProblems(contest.ID)
	if err != nil {
		return nil, err
	}

	return &ContestDetailResponse{ContestProfile: contestToProfile(contest), Problems: problems}, nil
}
```

Replace `UpdateContest` with (Task 4 fills in `applyProblems`; for now it must compile, so add the method stub at the end of this step as shown):

```go
// UpdateContest updates an existing contest in one transaction.
func (s *ContestService) UpdateContest(req UpdateContestRequest) (*ContestProfile, error) {
	var contest models.Contest
	if err := db.DB.Where("`key` = ?", req.ContestKey).First(&contest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = sanitization.SanitizeTitle(*req.Name)
	}
	if req.Description != nil {
		updates["description"] = sanitization.SanitizeProblemContent(*req.Description)
	}
	if req.Summary != nil {
		updates["summary"] = sanitization.SanitizeBlogSummary(*req.Summary)
	}
	if req.IsVisible != nil {
		updates["is_visible"] = *req.IsVisible
	}
	if req.IsRated != nil {
		updates["is_rated"] = *req.IsRated
	}
	if req.AccessCode != nil {
		updates["access_code"] = *req.AccessCode
	}
	if req.HideProblemTags != nil {
		updates["hide_problem_tags"] = *req.HideProblemTags
	}
	if req.RunPretestsOnly != nil {
		updates["run_pretests_only"] = *req.RunPretestsOnly
	}
	if req.IsOrganizationPrivate != nil {
		updates["is_organization_private"] = *req.IsOrganizationPrivate
	}
	if req.TimeLimit != nil {
		if *req.TimeLimit <= 0 {
			updates["time_limit"] = gorm.Expr("NULL")
		} else {
			updates["time_limit"] = *req.TimeLimit
		}
	}

	start, end := contest.StartTime, contest.EndTime
	if req.StartTime != nil {
		t, err := time.Parse(time.RFC3339, *req.StartTime)
		if err != nil {
			return nil, ErrInvalidTime
		}
		start = t
		updates["start_time"] = t
	}
	if req.EndTime != nil {
		t, err := time.Parse(time.RFC3339, *req.EndTime)
		if err != nil {
			return nil, ErrInvalidTime
		}
		end = t
		updates["end_time"] = t
	}
	if !end.After(start) {
		return nil, ErrInvalidSchedule
	}

	if req.FormatName != nil {
		if !contest_format.IsRegistered(*req.FormatName) {
			return nil, ErrUnknownFormat
		}
		updates["format_name"] = *req.FormatName
	}
	if req.FormatConfig != nil {
		if strings.TrimSpace(*req.FormatConfig) == "" {
			updates["format_config"] = gorm.Expr("NULL")
		} else {
			var cfg models.JSONField
			if err := cfg.Scan(*req.FormatConfig); err != nil {
				return nil, ErrInvalidFormatConfig
			}
			updates["format_config"] = cfg
		}
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&contest).Updates(updates).Error; err != nil {
				return err
			}
		}
		if err := relations.Sync[models.Profile](tx, &contest, "Authors", req.AuthorIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &contest, "Curators", req.CuratorIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.Profile](tx, &contest, "Testers", req.TesterIDs, nil, nil); err != nil {
			return err
		}
		if err := relations.Sync[models.ContestTag](tx, &contest, "Tags", req.TagIDs, req.AddTagIDs, req.RemoveTagIDs); err != nil {
			return err
		}
		return s.applyProblems(tx, contest.ID, req)
	})
	if err != nil {
		return nil, err
	}

	detail, err := s.GetContest(GetContestRequest{ContestKey: contest.Key})
	if err != nil {
		return nil, err
	}
	return &detail.ContestProfile, nil
}

// applyProblems is completed in Task 4 (full problems list + legacy merge).
func (s *ContestService) applyProblems(tx *gorm.DB, contestID uint, req UpdateContestRequest) error {
	return nil
}
```

In `contestToProfile`, after the existing id helpers add the seconds conversion and the refs. Replace the line `TimeLimit: c.TimeLimit,` in the literal with:

```go
		TimeLimit:              c.TimeLimit,
		TimeLimitSeconds:       microsToSeconds(c.TimeLimit),
```

and after `OrganizationIDs: orgIDs,` add:

```go
		Authors:                relations.UserRefs(c.Authors),
		Curators:               relations.UserRefs(c.Curators),
		Testers:                relations.UserRefs(c.Testers),
```

Add at the bottom of the file:

```go
// microsToSeconds converts Django DurationField microseconds to seconds.
func microsToSeconds(us *int64) *int64 {
	if us == nil {
		return nil
	}
	s := *us / 1_000_000
	return &s
}
```

Check the existing `getProfileIDs`, `getTagIDs`, `getOrgIDs` helpers in this file build their result with `make([]uint, len(x))` (non-nil). If any uses `var ids []uint`, change it to `make([]uint, 0, len(x))` so empty lists marshal as `[]`.

- [ ] **Step 7: Run the service tests to verify they pass**

Run: `go test ./service/contest/ -count=1`
Expected: PASS.

- [ ] **Step 8: Write the failing handler tests**

Append to `claoj/api/v2/admin_detail_test.go`:

```go
func TestAdminContestDetail_IsFlatWithProblemsAndSeconds(t *testing.T) {
	setupAdminDetailDB(t)
	limit := int64(5400 * 1_000_000)
	start := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	c := models.Contest{Key: "tik25", Name: "TiK25", StartTime: start, EndTime: start.Add(5 * time.Hour), TimeLimit: &limit}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)
	p := models.Problem{Code: "knap", Name: "Knapsack"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.ContestProblem{ContestID: c.ID, ProblemID: p.ID, Points: 100, Order: 1}).Error)

	router := authedRouter(1, http.MethodGet, "/admin/contest/:key", AdminContestDetail)
	w := doRequest(router, http.MethodGet, "/admin/contest/tik25", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "tik25", body["key"])
	require.Equal(t, float64(5400), body["time_limit"])
	problems := body["problems"].([]any)
	require.Len(t, problems, 1)
	require.Equal(t, "knap", problems[0].(map[string]any)["code"])
	require.NotContains(t, body, "contest")
}

func TestAdminContestUpdate_TimeLimitSecondsAndBadFormat(t *testing.T) {
	setupAdminDetailDB(t)
	start := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	c := models.Contest{Key: "c2", Name: "C2", StartTime: start, EndTime: start.Add(time.Hour)}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)

	router := authedRouter(1, http.MethodPatch, "/admin/contest/:key", AdminContestUpdate)
	w := doRequest(router, http.MethodPatch, "/admin/contest/c2", `{"time_limit": 1800}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var stored models.Contest
	require.NoError(t, db.DB.First(&stored, c.ID).Error)
	require.Equal(t, int64(1800*1_000_000), *stored.TimeLimit)

	w = doRequest(router, http.MethodPatch, "/admin/contest/c2", `{"format_name": "nope"}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
```

Add `"time"` to that file's imports.

- [ ] **Step 9: Run them to verify they fail**

Run: `go test ./api/v2/ -run 'TestAdminContest' -count=1`
Expected: FAIL — `body["key"]` nil (response still nested) and `time_limit` stored raw.

- [ ] **Step 10: Update the contest handlers**

In `claoj/api/v2/admin_contest.go`:

Replace the body of `AdminContestDetail` after the error handling (the `problems := make(...)` loop and the `c.JSON(...)`) with:

```go
	c.JSON(http.StatusOK, resp)
```

Add near the top of the file:

```go
// secondsToMicros converts the admin API's seconds to Django's
// DurationField microseconds. nil stays nil; 0 is passed through so the
// update path can clear the limit.
func secondsToMicros(seconds *int64) *int64 {
	if seconds == nil {
		return nil
	}
	us := *seconds * 1_000_000
	return &us
}
```

In `AdminContestCreate`, change the `TimeLimit:` line passed into `contest.CreateContestRequest{...}` to:

```go
		TimeLimit:             positiveOrNil(secondsToMicros(input.TimeLimit)),
```

and add:

```go
func positiveOrNil(v *int64) *int64 {
	if v == nil || *v <= 0 {
		return nil
	}
	return v
}
```

In `AdminContestUpdate`, replace the `input` struct with:

```go
	var input struct {
		Name                  *string                        `json:"name"`
		Description           *string                        `json:"description"`
		Summary               *string                        `json:"summary"`
		StartTime             *string                        `json:"start_time"`
		EndTime               *string                        `json:"end_time"`
		TimeLimit             *int64                         `json:"time_limit"` // seconds; 0 clears
		IsVisible             *bool                          `json:"is_visible"`
		IsRated               *bool                          `json:"is_rated"`
		AccessCode            *string                        `json:"access_code"`
		HideProblemTags       *bool                          `json:"hide_problem_tags"`
		RunPretestsOnly       *bool                          `json:"run_pretests_only"`
		IsOrganizationPrivate *bool                          `json:"is_organization_private"`
		FormatName            *string                        `json:"format_name"`
		FormatConfig          *string                        `json:"format_config"`
		AuthorIDs             *[]uint                        `json:"author_ids"`
		CuratorIDs            *[]uint                        `json:"curator_ids"`
		TesterIDs             *[]uint                        `json:"tester_ids"`
		TagIDs                *[]uint                        `json:"tag_ids"`
		Problems              *[]contest.ContestProblemInput `json:"problems"`
		AddProblemIDs         []uint                         `json:"add_problem_ids"`
		RemoveProblemIDs      []uint                         `json:"remove_problem_ids"`
		AddTagIDs             []uint                         `json:"add_tag_ids"`
		RemoveTagIDs          []uint                         `json:"remove_tag_ids"`
	}
```

Replace the `contest.UpdateContestRequest{...}` literal and the error handling after it with:

```go
	profile, err := getContestService().UpdateContest(contest.UpdateContestRequest{
		ContestKey:            key,
		Name:                  input.Name,
		Description:           input.Description,
		Summary:               input.Summary,
		StartTime:             input.StartTime,
		EndTime:               input.EndTime,
		TimeLimit:             secondsToMicros(input.TimeLimit),
		IsVisible:             input.IsVisible,
		IsRated:               input.IsRated,
		AccessCode:            input.AccessCode,
		HideProblemTags:       input.HideProblemTags,
		RunPretestsOnly:       input.RunPretestsOnly,
		IsOrganizationPrivate: input.IsOrganizationPrivate,
		FormatName:            input.FormatName,
		FormatConfig:          input.FormatConfig,
		AuthorIDs:             input.AuthorIDs,
		CuratorIDs:            input.CuratorIDs,
		TesterIDs:             input.TesterIDs,
		TagIDs:                input.TagIDs,
		Problems:              input.Problems,
		AddProblemIDs:         input.AddProblemIDs,
		RemoveProblemIDs:      input.RemoveProblemIDs,
		AddTagIDs:             input.AddTagIDs,
		RemoveTagIDs:          input.RemoveTagIDs,
	})
	if err != nil {
		switch {
		case errors.Is(err, contest.ErrContestNotFound):
			c.JSON(http.StatusNotFound, apiError("contest not found"))
		case errors.Is(err, contest.ErrContestProblemHasSubmissions):
			c.JSON(http.StatusConflict, apiError(err.Error()))
		default:
			c.JSON(http.StatusBadRequest, apiError(err.Error()))
		}
		return
	}
```

(keep the existing success `c.JSON(http.StatusOK, gin.H{"success": true, "contest": profile})`).

- [ ] **Step 11: Run all affected tests and build**

Run: `go build ./... && go test ./service/contest/ ./api/v2/ -count=1`
Expected: build OK; PASS. If `go build` reports another use of the removed `UpdateContestRequest.TimeLimit` semantics or `resp.Contest`, fix that call site to use `resp.ContestProfile`.

- [ ] **Step 12: Commit**

```bash
git add claoj/service/contest claoj/contest_format/registry.go claoj/api/v2/admin_contest.go claoj/api/v2/admin_detail_test.go
git commit -F - <<'EOF'
fix(admin): return flat contest detail; apply schedule, format, staff and tags

Start/end times were accepted but never written, the time limit was
stored in the wrong unit, and staff/tag/format edits were dropped.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 4: Contest problems as an ordered full list (backend)

**Files:**
- Modify: `claoj/service/contest/contest_problem_service.go` (add `ReplaceContestProblems`, `mergeLegacyProblemEdits`)
- Modify: `claoj/service/contest/contest_service.go` (`applyProblems` body)
- Test: `claoj/service/contest/contest_admin_test.go` (extend), `claoj/api/v2/admin_detail_test.go` (extend)

**Interfaces:**
- Consumes: `ContestProblemInput`, errors from Task 3, `setupContestDB`, `newContest`, `newProblem`, `newProfile`.
- Produces: `(*ContestProblemService).ReplaceContestProblems(tx *gorm.DB, contestID uint, items []ContestProblemInput) error`; `PATCH /admin/contest/:key` with `problems` → rows updated in place / inserted / deleted; 409 `contest: problem has contest submissions and cannot be removed: <code>` when removing a problem with submissions; legacy `add_problem_ids`/`remove_problem_ids` go through the same path.

- [ ] **Step 1: Write failing tests**

Append to `claoj/service/contest/contest_admin_test.go`:

```go
func ip(v int) *int    { return &v }
func bp(v bool) *bool  { return &v }

func contestRows(t *testing.T, contestID uint) []models.ContestProblem {
	t.Helper()
	var rows []models.ContestProblem
	require.NoError(t, db.DB.Where("contest_id = ?", contestID).Order("`order`").Find(&rows).Error)
	return rows
}

func TestReplaceContestProblems_UpdatesInPlaceInsertsAndOrders(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "rp")
	p1, p2, p3 := newProblem(t, "p1"), newProblem(t, "p2"), newProblem(t, "p3")
	existing := models.ContestProblem{ContestID: c.ID, ProblemID: p1.ID, Points: 100, Partial: true, Order: 1}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&existing).Error)

	items := []ContestProblemInput{
		{ProblemID: p2.ID, Points: ip(250)},
		{ProblemID: p1.ID, Partial: bp(false)},
		{ProblemID: p3.ID},
	}
	require.NoError(t, db.DB.Transaction(func(tx *gorm.DB) error {
		return NewContestProblemService().ReplaceContestProblems(tx, c.ID, items)
	}))

	rows := contestRows(t, c.ID)
	require.Len(t, rows, 3)
	require.Equal(t, []uint{p2.ID, p1.ID, p3.ID}, []uint{rows[0].ProblemID, rows[1].ProblemID, rows[2].ProblemID})
	require.Equal(t, 250, rows[0].Points)
	require.Equal(t, existing.ID, rows[1].ID, "existing row must keep its id")
	require.Equal(t, 100, rows[1].Points, "omitted points keep the existing value")
	require.False(t, rows[1].Partial)
	require.Equal(t, 100, rows[2].Points, "new rows default to 100 points")
	require.True(t, rows[2].Partial)
}

func TestReplaceContestProblems_RefusesRemovingProblemWithSubmissions(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "subs")
	p1, p2 := newProblem(t, "keep"), newProblem(t, "solved")
	cp1 := models.ContestProblem{ContestID: c.ID, ProblemID: p1.ID, Points: 100, Order: 1}
	cp2 := models.ContestProblem{ContestID: c.ID, ProblemID: p2.ID, Points: 100, Order: 2}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&cp1).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&cp2).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.ContestSubmission{SubmissionID: 1, ProblemID: cp2.ID, ParticipationID: 1}).Error)

	_, err := NewContestService().UpdateContest(UpdateContestRequest{
		ContestKey: c.Key,
		Name:       strp("changed"),
		Problems:   &[]ContestProblemInput{{ProblemID: p1.ID}},
	})
	require.ErrorIs(t, err, ErrContestProblemHasSubmissions)
	require.Contains(t, err.Error(), "solved")
	require.Len(t, contestRows(t, c.ID), 2, "nothing may be deleted")
	var stored models.Contest
	require.NoError(t, db.DB.First(&stored, c.ID).Error)
	require.Equal(t, "subs", stored.Name, "the whole update must roll back")
}

func TestReplaceContestProblems_RejectsUnknownDuplicateAndNegative(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "bad")
	p := newProblem(t, "only")
	svc := NewContestProblemService()
	run := func(items []ContestProblemInput) error {
		return db.DB.Transaction(func(tx *gorm.DB) error { return svc.ReplaceContestProblems(tx, c.ID, items) })
	}
	require.ErrorIs(t, run([]ContestProblemInput{{ProblemID: 404}}), ErrUnknownProblem)
	require.ErrorIs(t, run([]ContestProblemInput{{ProblemID: p.ID}, {ProblemID: p.ID}}), ErrUnknownProblem)
	require.ErrorIs(t, run([]ContestProblemInput{{ProblemID: p.ID, Points: ip(-1)}}), ErrInvalidPoints)
	require.Empty(t, contestRows(t, c.ID))
}

func TestUpdateContest_LegacyAddRemoveProblems(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "legacy")
	p1, p2 := newProblem(t, "l1"), newProblem(t, "l2")
	svc := NewContestService()

	_, err := svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, AddProblemIDs: []uint{p1.ID, p2.ID}})
	require.NoError(t, err)
	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, RemoveProblemIDs: []uint{p1.ID}})
	require.NoError(t, err)
	rows := contestRows(t, c.ID)
	require.Len(t, rows, 1)
	require.Equal(t, p2.ID, rows[0].ProblemID)
	require.Equal(t, uint(1), rows[0].Order)
}
```

Append to `claoj/api/v2/admin_detail_test.go`:

```go
func TestAdminContestUpdate_RemovingSolvedProblemIs409(t *testing.T) {
	setupAdminDetailDB(t)
	start := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	c := models.Contest{Key: "c409", Name: "C", StartTime: start, EndTime: start.Add(time.Hour)}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)
	p := models.Problem{Code: "hard", Name: "Hard"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	cp := models.ContestProblem{ContestID: c.ID, ProblemID: p.ID, Points: 100, Order: 1}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&cp).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.ContestSubmission{SubmissionID: 1, ProblemID: cp.ID, ParticipationID: 1}).Error)

	router := authedRouter(1, http.MethodPatch, "/admin/contest/:key", AdminContestUpdate)
	w := doRequest(router, http.MethodPatch, "/admin/contest/c409", `{"problems": []}`)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "hard")
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./service/contest/ ./api/v2/ -run 'Replace|Legacy|409' -count=1`
Expected: FAIL — `ReplaceContestProblems` undefined; legacy test sees no rows (stub `applyProblems`).

- [ ] **Step 3: Implement `ReplaceContestProblems` and the legacy merge**

Append to `claoj/service/contest/contest_problem_service.go` (add `"fmt"` and `"gorm.io/gorm"` to imports if missing):

```go
const defaultContestProblemPoints = 100

// ReplaceContestProblems makes the contest's problem rows exactly items,
// in that order. Existing rows are updated in place (their ids are what
// judge_contestsubmission references); unlisted rows are deleted unless
// they already have contest submissions. Run inside the caller's tx.
func (s *ContestProblemService) ReplaceContestProblems(tx *gorm.DB, contestID uint, items []ContestProblemInput) error {
	listed := make(map[uint]bool, len(items))
	ids := make([]uint, 0, len(items))
	for _, it := range items {
		if listed[it.ProblemID] {
			return fmt.Errorf("%w: problem %d is listed twice", ErrUnknownProblem, it.ProblemID)
		}
		if it.Points != nil && *it.Points < 0 {
			return ErrInvalidPoints
		}
		listed[it.ProblemID] = true
		ids = append(ids, it.ProblemID)
	}
	if len(ids) > 0 {
		var n int64
		if err := tx.Model(&models.Problem{}).Where("id IN ?", ids).Count(&n).Error; err != nil {
			return err
		}
		if int(n) != len(ids) {
			return ErrUnknownProblem
		}
	}

	var existing []models.ContestProblem
	if err := tx.Preload("Problem").Where("contest_id = ?", contestID).Find(&existing).Error; err != nil {
		return err
	}
	byProblem := make(map[uint]models.ContestProblem, len(existing))
	for _, cp := range existing {
		byProblem[cp.ProblemID] = cp
	}

	for _, cp := range existing {
		if listed[cp.ProblemID] {
			continue
		}
		var subs int64
		if err := tx.Model(&models.ContestSubmission{}).Where("problem_id = ?", cp.ID).Count(&subs).Error; err != nil {
			return err
		}
		if subs > 0 {
			return fmt.Errorf("%w: %s", ErrContestProblemHasSubmissions, cp.Problem.Code)
		}
		if err := tx.Delete(&models.ContestProblem{}, cp.ID).Error; err != nil {
			return err
		}
	}

	for i, it := range items {
		cp, ok := byProblem[it.ProblemID]
		if !ok {
			cp = models.ContestProblem{
				ContestID: contestID,
				ProblemID: it.ProblemID,
				Points:    defaultContestProblemPoints,
				Partial:   true,
			}
		}
		cp.Order = uint(i + 1)
		if it.Points != nil {
			cp.Points = *it.Points
		}
		if it.Partial != nil {
			cp.Partial = *it.Partial
		}
		if it.IsPretested != nil {
			cp.IsPretested = *it.IsPretested
		}
		cp.Problem = models.Problem{}
		cp.Contest = models.Contest{}
		if err := tx.Omit("Problem", "Contest").Save(&cp).Error; err != nil {
			return err
		}
	}
	return nil
}

// mergeLegacyProblemEdits turns add/remove id lists into a full ordered
// list (current order, removals dropped, additions appended) so the legacy
// form gets the same submission protection as the full list.
func (s *ContestProblemService) mergeLegacyProblemEdits(tx *gorm.DB, contestID uint, add, remove []uint) ([]ContestProblemInput, error) {
	var current []models.ContestProblem
	if err := tx.Where("contest_id = ?", contestID).Order("`order` ASC").Find(&current).Error; err != nil {
		return nil, err
	}
	removed := make(map[uint]bool, len(remove))
	for _, id := range remove {
		removed[id] = true
	}
	present := make(map[uint]bool, len(current)+len(add))
	items := make([]ContestProblemInput, 0, len(current)+len(add))
	for _, cp := range current {
		if removed[cp.ProblemID] {
			continue
		}
		present[cp.ProblemID] = true
		items = append(items, ContestProblemInput{ProblemID: cp.ProblemID})
	}
	for _, id := range add {
		if !present[id] && !removed[id] {
			present[id] = true
			items = append(items, ContestProblemInput{ProblemID: id})
		}
	}
	return items, nil
}
```

Also change the hard-coded `Points: 100,` in `AddProblemsToContest` to `Points: defaultContestProblemPoints,` so create and update share the default.

- [ ] **Step 4: Fill in `applyProblems`**

In `claoj/service/contest/contest_service.go` replace the stub with:

```go
// applyProblems writes the contest problem list: the full list when given,
// otherwise the legacy add/remove lists folded into one.
func (s *ContestService) applyProblems(tx *gorm.DB, contestID uint, req UpdateContestRequest) error {
	if req.Problems != nil {
		return s.problems.ReplaceContestProblems(tx, contestID, *req.Problems)
	}
	if len(req.AddProblemIDs) == 0 && len(req.RemoveProblemIDs) == 0 {
		return nil
	}
	items, err := s.problems.mergeLegacyProblemEdits(tx, contestID, req.AddProblemIDs, req.RemoveProblemIDs)
	if err != nil {
		return err
	}
	return s.problems.ReplaceContestProblems(tx, contestID, items)
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./service/contest/ ./api/v2/ -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add claoj/service/contest claoj/api/v2/admin_detail_test.go
git commit -F - <<'EOF'
feat(admin): replace contest problems as an ordered list, protecting solved rows

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 5: Problem and contest list filters (backend)

**Files:**
- Modify: `claoj/service/problem/problem_service.go` (`ListProblemsRequest` 103-106, `ListProblems` 142-188)
- Modify: `claoj/service/contest/types.go` (`ListContestsRequest`), `claoj/service/contest/contest_service.go` (`ListContests` 31-70), `claoj/service/contest/errors.go`
- Modify: `claoj/api/v2/admin_problem.go` (`AdminProblemList`), `claoj/api/v2/admin_contest.go` (`AdminContestList`)
- Test: `claoj/service/problem/problem_service_test.go`, `claoj/service/contest/contest_admin_test.go`

**Interfaces:**
- Consumes: `db.ContainsPattern` (Task 1).
- Produces: `GET /admin/problems?search=&is_public=true|false&group_id=` and `GET /admin/contests?search=&status=upcoming|ongoing|past` (invalid status → 400); `total` counts the filtered set. Service: `ListProblemsRequest{Page, PageSize int; Search string; IsPublic *bool; GroupID uint}`, `ListContestsRequest{Page, PageSize int; Search, Status string; Now time.Time}`, `contest.ErrInvalidStatus`.

- [ ] **Step 1: Write failing tests**

Append to `claoj/service/problem/problem_service_test.go`:

```go
func (s *ProblemServiceTestSuite) TestListProblems_FiltersAndCountsFilteredTotal() {
	g1 := s.createTestGroup("ga")
	g2 := s.createTestGroup("gb")
	s.createTestProblem("dijkstra", "Shortest Path", g1.ID)
	s.createTestProblem("knapsack", "Knapsack", g2.ID)
	priv := s.createTestProblem("secret", "Hidden Dijkstra", g1.ID)
	s.Require().NoError(s.database.Model(&priv).Update("is_public", false).Error)

	resp, err := s.service.ListProblems(ListProblemsRequest{Page: 1, PageSize: 10, Search: "dijkstra"})
	s.Require().NoError(err)
	s.Equal(int64(2), resp.Total, "matches code or name, case-insensitively")

	public := true
	resp, _ = s.service.ListProblems(ListProblemsRequest{Page: 1, PageSize: 10, Search: "dijkstra", IsPublic: &public})
	s.Equal(int64(1), resp.Total)
	s.Equal("dijkstra", resp.Problems[0].Code)

	resp, _ = s.service.ListProblems(ListProblemsRequest{Page: 1, PageSize: 10, GroupID: g2.ID})
	s.Equal(int64(1), resp.Total)
	s.Equal("knapsack", resp.Problems[0].Code)
}

func (s *ProblemServiceTestSuite) TestListProblems_SearchIsLiteral() {
	g := s.createTestGroup("gl")
	s.createTestProblem("a_b", "Underscore", g.ID)
	s.createTestProblem("axb", "Other", g.ID)

	resp, err := s.service.ListProblems(ListProblemsRequest{Page: 1, PageSize: 10, Search: "a_b"})
	s.Require().NoError(err)
	s.Equal(int64(1), resp.Total)
	s.Equal("a_b", resp.Problems[0].Code)
}
```

Append to `claoj/service/contest/contest_admin_test.go`:

```go
func TestListContests_SearchAndStatus(t *testing.T) {
	setupContestDB(t)
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	mk := func(key, name string, start time.Time) {
		c := models.Contest{Key: key, Name: name, StartTime: start, EndTime: start.Add(2 * time.Hour)}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)
	}
	mk("past1", "Halloween Contest", now.Add(-48*time.Hour))
	mk("live1", "Live Round", now.Add(-time.Hour))
	mk("next1", "Next Halloween", now.Add(24*time.Hour))
	svc := NewContestService()

	resp, err := svc.ListContests(ListContestsRequest{Page: 1, PageSize: 10, Search: "halloween", Now: now})
	require.NoError(t, err)
	require.Equal(t, int64(2), resp.Total)

	for status, key := range map[string]string{"past": "past1", "ongoing": "live1", "upcoming": "next1"} {
		resp, err = svc.ListContests(ListContestsRequest{Page: 1, PageSize: 10, Status: status, Now: now})
		require.NoError(t, err)
		require.Equal(t, int64(1), resp.Total, status)
		require.Equal(t, key, resp.Contests[0].Key, status)
	}

	_, err = svc.ListContests(ListContestsRequest{Page: 1, PageSize: 10, Status: "soon"})
	require.ErrorIs(t, err, ErrInvalidStatus)
}
```

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./service/problem/ ./service/contest/ -run 'Filter|Literal|SearchAndStatus' -count=1`
Expected: FAIL to compile — unknown fields `Search`, `IsPublic`, `GroupID`, `Status`, `Now`; undefined `ErrInvalidStatus`.

- [ ] **Step 3: Implement problem list filters**

In `claoj/service/problem/problem_service.go` replace `ListProblemsRequest` with:

```go
// ListProblemsRequest holds the parameters for listing problems.
type ListProblemsRequest struct {
	Page     int
	PageSize int
	Search   string // code or name contains (literal)
	IsPublic *bool
	GroupID  uint
}
```

In `ListProblems`, replace everything from `var problems []struct {` through the `Scan(&problems)` error check with:

```go
	var problems []struct {
		models.Problem
		GroupName string `gorm:"column:group_name"`
	}

	filter := func(q *gorm.DB) *gorm.DB {
		if req.Search != "" {
			p := db.ContainsPattern(req.Search)
			q = q.Where("(judge_problem.code LIKE ? ESCAPE '!' OR judge_problem.name LIKE ? ESCAPE '!')", p, p)
		}
		if req.IsPublic != nil {
			q = q.Where("judge_problem.is_public = ?", *req.IsPublic)
		}
		if req.GroupID != 0 {
			q = q.Where("judge_problem.group_id = ?", req.GroupID)
		}
		return q
	}

	var total int64
	if err := filter(db.DB.Table("judge_problem")).Count(&total).Error; err != nil {
		return nil, err
	}

	if err := filter(db.DB.Table("judge_problem").
		Joins("LEFT JOIN judge_problemgroup ON judge_problemgroup.id = judge_problem.group_id").
		Select("judge_problem.*, judge_problemgroup.name as group_name")).
		Order("judge_problem.date DESC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Scan(&problems).Error; err != nil {
		return nil, err
	}
```

(Case-insensitivity: MySQL's default collation and SQLite's ASCII LIKE are both case-insensitive for ASCII; no `LOWER()` needed.)

- [ ] **Step 4: Implement contest list filters**

`claoj/service/contest/errors.go` — add:

```go
	ErrInvalidStatus                = errors.New("contest: status must be upcoming, ongoing or past")
```

`claoj/service/contest/types.go` — replace `ListContestsRequest` with:

```go
// ListContestsRequest holds the parameters for listing contests.
type ListContestsRequest struct {
	Page     int
	PageSize int
	Search   string    // key or name contains (literal)
	Status   string    // "", "upcoming", "ongoing", "past"
	Now      time.Time // zero = time.Now(); tests pin it
}
```

`claoj/service/contest/contest_service.go` — in `ListContests`, replace from `var contests []models.Contest` through the paginated `Find` error check with:

```go
	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}
	filter := func(q *gorm.DB) (*gorm.DB, error) {
		if req.Search != "" {
			p := db.ContainsPattern(req.Search)
			q = q.Where("(`key` LIKE ? ESCAPE '!' OR name LIKE ? ESCAPE '!')", p, p)
		}
		switch req.Status {
		case "":
		case "upcoming":
			q = q.Where("start_time > ?", now)
		case "ongoing":
			q = q.Where("start_time <= ? AND end_time > ?", now, now)
		case "past":
			q = q.Where("end_time <= ?", now)
		default:
			return nil, ErrInvalidStatus
		}
		return q, nil
	}

	countQ, err := filter(db.DB.Model(&models.Contest{}))
	if err != nil {
		return nil, err
	}
	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, err
	}

	listQ, _ := filter(db.DB.Model(&models.Contest{}))
	var contests []models.Contest
	if err := listQ.Order("start_time DESC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&contests).Error; err != nil {
		return nil, err
	}
```

- [ ] **Step 5: Bind the query params in the handlers**

In `claoj/api/v2/admin_problem.go` `AdminProblemList` (add `"strconv"` import), replace the `ListProblems` call with:

```go
	var isPublic *bool
	if v := c.Query("is_public"); v != "" {
		b := v == "true" || v == "1"
		isPublic = &b
	}
	groupID, _ := strconv.ParseUint(c.Query("group_id"), 10, 32)

	resp, err := getProblemService().ListProblems(problem.ListProblemsRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   strings.TrimSpace(c.Query("search")),
		IsPublic: isPublic,
		GroupID:  uint(groupID),
	})
```

(add `"strings"` import too).

In `claoj/api/v2/admin_contest.go` `AdminContestList`, replace the `ListContests` call and its error handling with:

```go
	resp, err := getContestService().ListContests(contest.ListContestsRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   strings.TrimSpace(c.Query("search")),
		Status:   c.Query("status"),
	})
	if err != nil {
		if errors.Is(err, contest.ErrInvalidStatus) {
			c.JSON(http.StatusBadRequest, apiError(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
```

(add `"strings"` import).

- [ ] **Step 6: Run the tests and build**

Run: `go build ./... && go test ./service/problem/ ./service/contest/ ./api/v2/ -count=1`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add claoj/service/problem claoj/service/contest claoj/api/v2/admin_problem.go claoj/api/v2/admin_contest.go
git commit -F - <<'EOF'
feat(admin): server-side search and filters for problem and contest lists

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 6: Organization search and submission filters (backend)

**Files:**
- Modify: `claoj/service/organization/organization_service.go` (`ListOrganizationsRequest` 64-67, `ListOrganizations` 104-140)
- Modify: `claoj/service/submission/submission_service.go` (`ListSubmissionsRequest` 95-98, `ListSubmissions` 131-180)
- Modify: `claoj/api/v2/admin_judge_organization.go` (`AdminOrganizationList`), `claoj/api/v2/admin_submission.go` (`AdminSubmissionList`)
- Test: `claoj/service/organization/organization_search_test.go` (new), `claoj/service/submission/submission_list_test.go` (new)

**Interfaces:**
- Consumes: `db.ContainsPattern`.
- Produces: `GET /admin/organizations?search=&page=&page_size=`; `GET /admin/submissions?user=<username>&problem=<code>&result=<code>&status=<code>&language=<key or name>` (exact matches); filtered `total`. Service: `ListOrganizationsRequest.Search string`; `ListSubmissionsRequest{Page, PageSize int; Username, ProblemCode, Result, Status, Language string}`.

- [ ] **Step 1: Write failing tests**

`claoj/service/organization/organization_search_test.go`:

```go
package organization

import (
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestListOrganizations_Search(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := database.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.AuthUser{}, &models.Profile{}, &models.Organization{}))
	db.DB = database
	for _, o := range []models.Organization{
		{Name: "CLB ABA", Slug: "clb-aba", ShortName: "ABA"},
		{Name: "Long An HSGS", Slug: "la-hsgs", ShortName: "HSGS"},
	} {
		o := o
		require.NoError(t, database.Omit(clause.Associations).Create(&o).Error)
	}

	resp, err := NewOrganizationService().ListOrganizations(ListOrganizationsRequest{Page: 1, PageSize: 20, Search: "hsgs"})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.Total)
	require.Equal(t, "Long An HSGS", resp.Organizations[0].Name)

	resp, _ = NewOrganizationService().ListOrganizations(ListOrganizationsRequest{Page: 1, PageSize: 20})
	require.Equal(t, int64(2), resp.Total)
}
```

`claoj/service/submission/submission_list_test.go`:

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

func TestListSubmissions_Filters(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := database.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.AuthUser{}, &models.Profile{}, &models.Problem{}, &models.Language{}, &models.Submission{}))
	db.DB = database

	mkUser := func(name string) models.Profile {
		u := models.AuthUser{Username: name, IsActive: true}
		require.NoError(t, database.Create(&u).Error)
		p := models.Profile{UserID: u.ID, Timezone: "UTC"}
		require.NoError(t, database.Omit(clause.Associations).Create(&p).Error)
		return p
	}
	alice, bob := mkUser("alice"), mkUser("bob")
	pa := models.Problem{Code: "aplusb", Name: "A+B"}
	pk := models.Problem{Code: "knap", Name: "Knapsack"}
	require.NoError(t, database.Omit(clause.Associations).Create(&pa).Error)
	require.NoError(t, database.Omit(clause.Associations).Create(&pk).Error)
	cpp := models.Language{Key: "CPP17", Name: "C++17", CommonName: "C++"}
	py := models.Language{Key: "PY3", Name: "Python 3", CommonName: "Python"}
	require.NoError(t, database.Create(&cpp).Error)
	require.NoError(t, database.Create(&py).Error)

	ac, wa := "AC", "WA"
	for _, s := range []models.Submission{
		{UserID: alice.ID, ProblemID: pa.ID, LanguageID: cpp.ID, Status: "D", Result: &ac, Date: time.Now()},
		{UserID: alice.ID, ProblemID: pk.ID, LanguageID: py.ID, Status: "D", Result: &wa, Date: time.Now()},
		{UserID: bob.ID, ProblemID: pa.ID, LanguageID: cpp.ID, Status: "QU", Date: time.Now()},
	} {
		s := s
		require.NoError(t, database.Omit(clause.Associations).Create(&s).Error)
	}

	svc := &SubmissionService{}
	cases := []struct {
		req  ListSubmissionsRequest
		want int64
	}{
		{ListSubmissionsRequest{}, 3},
		{ListSubmissionsRequest{Username: "alice"}, 2},
		{ListSubmissionsRequest{ProblemCode: "aplusb"}, 2},
		{ListSubmissionsRequest{Result: "AC"}, 1},
		{ListSubmissionsRequest{Status: "QU"}, 1},
		{ListSubmissionsRequest{Language: "PY3"}, 1},
		{ListSubmissionsRequest{Language: "C++17"}, 2},
		{ListSubmissionsRequest{Username: "alice", ProblemCode: "aplusb"}, 1},
	}
	for _, tc := range cases {
		tc.req.Page, tc.req.PageSize = 1, 20
		resp, err := svc.ListSubmissions(tc.req)
		require.NoError(t, err)
		require.Equal(t, tc.want, resp.Total, "%+v", tc.req)
		require.Len(t, resp.Submissions, int(tc.want), "%+v", tc.req)
	}
}
```

(If `SubmissionService` has required fields, construct it the way `NewSubmissionService(nil)` does — check its constructor signature in `submission_service.go` and use `NewSubmissionService(nil)` instead of `&SubmissionService{}` if a nil bridge is accepted.)

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./service/organization/ ./service/submission/ -run 'Search|Filters' -count=1`
Expected: FAIL to compile — unknown fields `Search`, `Username`, `ProblemCode`, `Result`, `Status`, `Language`.

- [ ] **Step 3: Implement organization search**

`organization_service.go` — add `Search string` to `ListOrganizationsRequest`:

```go
type ListOrganizationsRequest struct {
	Page     int
	PageSize int
	Search   string // name, short name or slug contains (literal)
}
```

In `ListOrganizations`, replace from `var organizations []models.Organization` through the paginated `Find` error check with:

```go
	filter := func(q *gorm.DB) *gorm.DB {
		if req.Search != "" {
			p := db.ContainsPattern(req.Search)
			q = q.Where("(name LIKE ? ESCAPE '!' OR short_name LIKE ? ESCAPE '!' OR slug LIKE ? ESCAPE '!')", p, p, p)
		}
		return q
	}

	var total int64
	if err := filter(db.DB.Model(&models.Organization{})).Count(&total).Error; err != nil {
		return nil, err
	}

	var organizations []models.Organization
	if err := filter(db.DB.Model(&models.Organization{})).Order("name ASC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&organizations).Error; err != nil {
		return nil, err
	}
```

(add `"gorm.io/gorm"` import if the file lacks it).

- [ ] **Step 4: Implement submission filters**

`submission_service.go` — replace `ListSubmissionsRequest` with:

```go
// ListSubmissionsRequest holds the parameters for listing submissions.
// All filters are exact matches; empty means no filter.
type ListSubmissionsRequest struct {
	Page        int
	PageSize    int
	Username    string
	ProblemCode string
	Result      string
	Status      string
	Language    string // judge_language.key or judge_language.name
}
```

In `ListSubmissions`, replace from the `query := db.DB.Table("judge_submission")...` assignment through the `Scan(&submissions)` error check with:

```go
	// judge_submission.user_id references judge_profile.id, so the username
	// join must go through judge_profile.
	joined := func() *gorm.DB {
		return db.DB.Table("judge_submission").
			Joins("JOIN judge_profile ON judge_profile.id = judge_submission.user_id").
			Joins("JOIN auth_user ON auth_user.id = judge_profile.user_id").
			Joins("JOIN judge_problem ON judge_problem.id = judge_submission.problem_id").
			Joins("JOIN judge_language ON judge_language.id = judge_submission.language_id")
	}
	filter := func(q *gorm.DB) *gorm.DB {
		if req.Username != "" {
			q = q.Where("auth_user.username = ?", req.Username)
		}
		if req.ProblemCode != "" {
			q = q.Where("judge_problem.code = ?", req.ProblemCode)
		}
		if req.Result != "" {
			q = q.Where("judge_submission.result = ?", req.Result)
		}
		if req.Status != "" {
			q = q.Where("judge_submission.status = ?", req.Status)
		}
		if req.Language != "" {
			q = q.Where("(judge_language.key = ? OR judge_language.name = ?)", req.Language, req.Language)
		}
		return q
	}
	filtered := req.Username != "" || req.ProblemCode != "" || req.Result != "" || req.Status != "" || req.Language != ""

	// Unfiltered counts skip the joins: judge_submission is the largest table.
	var total int64
	countQ := db.DB.Model(&models.Submission{})
	if filtered {
		countQ = filter(joined())
	}
	if err := countQ.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := filter(joined()).
		Select("judge_submission.*, auth_user.username, judge_problem.code as problem_code, judge_language.name as language_name").
		Order("judge_submission.date DESC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Scan(&submissions).Error; err != nil {
		return nil, err
	}
```

(keep the `var submissions []struct{...}` declaration above it.)

- [ ] **Step 5: Bind the params in the handlers**

`api/v2/admin_judge_organization.go` `AdminOrganizationList`:

```go
	resp, err := getOrgService().ListOrganizations(organization.ListOrganizationsRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   strings.TrimSpace(c.Query("search")),
	})
```

`api/v2/admin_submission.go` `AdminSubmissionList`:

```go
	resp, err := getSubmissionService().ListSubmissions(submission.ListSubmissionsRequest{
		Page:        page,
		PageSize:    pageSize,
		Username:    strings.TrimSpace(c.Query("user")),
		ProblemCode: strings.TrimSpace(c.Query("problem")),
		Result:      c.Query("result"),
		Status:      c.Query("status"),
		Language:    strings.TrimSpace(c.Query("language")),
	})
```

(add `"strings"` imports where missing).

- [ ] **Step 6: Run the tests and build**

Run: `go build ./... && go test ./service/... ./api/v2/ -count=1`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add claoj/service/organization claoj/service/submission claoj/api/v2/admin_judge_organization.go claoj/api/v2/admin_submission.go
git commit -F - <<'EOF'
feat(admin): organization search and working submission filters

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 7: Frontend API contract and pure form mappers

**Files:**
- Modify: `claoj-web/src/types/index.ts`
- Modify: `claoj-web/src/lib/adminApi.ts`
- Create: `claoj-web/src/lib/adminForms.ts`
- Test: `claoj-web/__tests__/lib/adminForms.test.ts`

**Interfaces:**
- Consumes: backend contracts from Tasks 2–6.
- Produces (TypeScript):
  - Types `UserRef`, `AdminProblemDetail`, `AdminContestProblemRow`, `AdminContestDetail`, `ContestProblemInput`, `AdminContestUpdateRequest`, `AdminProblemUpdateRequest`.
  - `adminProblemApi.list(page, pageSize, filters?: { search?: string; is_public?: 'true' | 'false'; group_id?: number })`, `adminContestApi.list(page, pageSize, filters?: { search?: string; status?: 'upcoming' | 'ongoing' | 'past' })`, `adminUserApi.list(page, pageSize, search?: string)`, `adminOrganizationApi.list(page, pageSize, search?: string)`, `adminSubmissionApi.list(page, pageSize, filters?: { user?: string; problem?: string; result?: string; status?: string; language?: string })`; `adminProblemApi.detail → AdminProblemDetail`; `adminContestApi.detail → AdminContestDetail`; `adminContestApi.update(key, data: AdminContestUpdateRequest)`; `adminProblemApi.update(code, data: AdminProblemUpdateRequest)`.
  - `adminForms.ts`: `toLocalInput(iso: string): string`, `fromLocalInput(local: string): string`, `problemToFormData(p: AdminProblemDetail): ProblemFormData`, `contestToFormData(c: AdminContestDetail): ContestFormData`, `contestFormToUpdate(data: ContestFormData, existing: AdminContestProblemRow[]): AdminContestUpdateRequest`, `contestFormToCreate(data: ContestFormData): AdminContestCreateRequest`.

- [ ] **Step 1: Write the failing mapper tests**

`claoj-web/__tests__/lib/adminForms.test.ts`:

```ts
import {
    toLocalInput,
    fromLocalInput,
    problemToFormData,
    contestToFormData,
    contestFormToUpdate,
    contestFormToCreate,
} from '@/lib/adminForms';
import type { AdminProblemDetail, AdminContestDetail } from '@/types';

const problem: AdminProblemDetail = {
    id: 1, code: 'aplusb', name: 'A+B', description: 'd', points: 5, partial: false, is_public: true,
    time_limit: 1.5, memory_limit: 262144, group_id: 3, is_manually_managed: false, pdf_url: '/x.pdf',
    author_ids: [4, 9], curator_ids: [2], tester_ids: [], type_ids: [7], allowed_lang_ids: [1, 2],
    authors: [{ id: 4, username: 'a' }, { id: 9, username: 'b' }], curators: [{ id: 2, username: 'c' }], testers: [],
};

const contest: AdminContestDetail = {
    id: 1, key: 'tik25', name: 'TiK25', description: 'd', summary: 's',
    start_time: '2026-11-01T02:00:00Z', end_time: '2026-11-01T05:00:00Z', time_limit: 5400,
    is_visible: true, is_rated: false, format_name: 'ioi16', format_config: { cumtime: true },
    access_code: '', hide_problem_tags: false, run_pretests_only: false, is_organization_private: false,
    locked_after: null, author_ids: [4], curator_ids: [], tester_ids: [9], tag_ids: [3],
    authors: [{ id: 4, username: 'a' }], curators: [], testers: [{ id: 9, username: 'b' }],
    problems: [
        { id: 11, problem_id: 20, code: 'p20', name: 'P20', points: 250, partial: false, is_pretested: true, order: 1 },
        { id: 12, problem_id: 21, code: 'p21', name: 'P21', points: 100, partial: true, is_pretested: false, order: 2 },
    ],
};

describe('adminForms', () => {
    it('local datetime round-trips through ISO', () => {
        const local = '2026-11-01T09:30';
        const iso = fromLocalInput(local);
        expect(iso).toMatch(/Z$/);
        expect(toLocalInput(iso)).toBe(local);
        expect(new Date(iso).getTime()).toBe(new Date(2026, 10, 1, 9, 30).getTime());
    });

    it('problemToFormData keeps every relation id', () => {
        const f = problemToFormData(problem);
        expect(f.author_ids).toEqual([4, 9]);
        expect(f.type_ids).toEqual([7]);
        expect(f.allowed_lang_ids).toEqual([1, 2]);
        expect(f.group_id).toBe(3);
        expect(f.pdf_url).toBe('/x.pdf');
    });

    it('contestToFormData keeps every relation id and shows minutes', () => {
        const f = contestToFormData(contest);
        expect(f.author_ids).toEqual([4]);
        expect(f.tester_ids).toEqual([9]);
        expect(f.curator_ids).toEqual([]);
        expect(f.tag_ids).toEqual([3]);
        expect(f.problem_ids).toEqual([20, 21]);
        expect(f.time_limit).toBe(90);
        expect(f.format_config).toBe('{"cumtime":true}');
        expect(f.start_time).toBe(toLocalInput(contest.start_time));
    });

    it('contestFormToUpdate preserves per-problem settings and appends new problems', () => {
        const f = { ...contestToFormData(contest), problem_ids: [21, 20, 30] };
        const req = contestFormToUpdate(f, contest.problems);
        expect(req.problems).toEqual([
            { problem_id: 21, points: 100, partial: true, is_pretested: false },
            { problem_id: 20, points: 250, partial: false, is_pretested: true },
            { problem_id: 30 },
        ]);
        expect(req.time_limit).toBe(5400);
        expect(req.start_time).toBe(fromLocalInput(f.start_time));
        expect('key' in req).toBe(false);
        expect('problem_ids' in req).toBe(false);
    });

    it('an emptied time limit clears it on update and is omitted on create', () => {
        const f = { ...contestToFormData(contest), time_limit: undefined };
        expect(contestFormToUpdate(f, contest.problems).time_limit).toBe(0);
        expect(contestFormToCreate(f).time_limit).toBeUndefined();
    });
});
```

- [ ] **Step 2: Run them to verify they fail**

Run: `npx jest __tests__/lib/adminForms.test.ts`
Expected: FAIL — `Cannot find module '@/lib/adminForms'`.

- [ ] **Step 3: Add the types**

In `claoj-web/src/types/index.ts`, after the existing `AdminContest` interface, add:

```ts
// ============================================================
// ADMIN DETAIL CONTRACTS (flat snake_case, see api/v2/admin_problem.go,
// api/v2/admin_contest.go)
// ============================================================

export interface UserRef {
    id: number;
    username: string;
}

export interface AdminProblemDetail {
    id: number;
    code: string;
    name: string;
    description: string;
    points: number;
    partial: boolean;
    is_public: boolean;
    time_limit: number;
    memory_limit: number;
    group_id: number;
    is_manually_managed: boolean;
    pdf_url: string;
    author_ids: number[];
    curator_ids: number[];
    tester_ids: number[];
    type_ids: number[];
    allowed_lang_ids: number[];
    authors: UserRef[];
    curators: UserRef[];
    testers: UserRef[];
}

export interface AdminProblemUpdateRequest {
    name?: string;
    description?: string;
    points?: number;
    partial?: boolean;
    is_public?: boolean;
    time_limit?: number;
    memory_limit?: number;
    is_manually_managed?: boolean;
    pdf_url?: string;
    group_id?: number;
    author_ids?: number[];
    curator_ids?: number[];
    tester_ids?: number[];
    type_ids?: number[];
    allowed_lang_ids?: number[];
}

export interface AdminContestProblemRow {
    id: number;
    problem_id: number;
    code: string;
    name: string;
    points: number;
    partial: boolean;
    is_pretested: boolean;
    order: number;
}

export interface AdminContestDetail {
    id: number;
    key: string;
    name: string;
    description: string;
    summary: string;
    start_time: string;
    end_time: string;
    /** Per-participant window in seconds; null = whole contest. */
    time_limit: number | null;
    is_visible: boolean;
    is_rated: boolean;
    format_name: string;
    format_config: Record<string, unknown> | null;
    access_code: string;
    hide_problem_tags: boolean;
    run_pretests_only: boolean;
    is_organization_private: boolean;
    locked_after: string | null;
    author_ids: number[];
    curator_ids: number[];
    tester_ids: number[];
    tag_ids: number[];
    authors: UserRef[];
    curators: UserRef[];
    testers: UserRef[];
    problems: AdminContestProblemRow[];
}

/** One row of the ordered full problem list sent on contest update. */
export interface ContestProblemInput {
    problem_id: number;
    points?: number;
    partial?: boolean;
    is_pretested?: boolean;
}

export interface AdminContestUpdateRequest {
    name?: string;
    description?: string;
    summary?: string;
    start_time?: string;
    end_time?: string;
    /** Seconds; 0 clears. */
    time_limit?: number;
    is_visible?: boolean;
    is_rated?: boolean;
    format_name?: string;
    format_config?: string;
    access_code?: string;
    hide_problem_tags?: boolean;
    run_pretests_only?: boolean;
    is_organization_private?: boolean;
    author_ids?: number[];
    curator_ids?: number[];
    tester_ids?: number[];
    tag_ids?: number[];
    problems?: ContestProblemInput[];
}
```

Check `AdminContestCreateRequest` in the same file: if it lacks `tag_ids?: number[]`, add it.

- [ ] **Step 4: Update `adminApi.ts`**

In `claoj-web/src/lib/adminApi.ts`:

Add `AdminProblemDetail, AdminProblemUpdateRequest, AdminContestDetail, AdminContestUpdateRequest` to the type import list.

Add near the top (after the imports):

```ts
/** Builds a query string from non-empty params. */
function qs(params: Record<string, string | number | undefined>): string {
    const sp = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
        if (v !== undefined && v !== '') sp.set(k, String(v));
    }
    return sp.toString();
}
```

Replace these members:

```ts
// adminUserApi
    list: (page: number = 1, pageSize: number = 50, search?: string) =>
        api.get<AdminUserListResponse>(`/admin/users?${qs({ page, page_size: pageSize, search })}`),

// adminContestApi
    list: (page: number = 1, pageSize: number = 50, filters: { search?: string; status?: 'upcoming' | 'ongoing' | 'past' } = {}) =>
        api.get<AdminContestListResponse>(`/admin/contests?${qs({ page, page_size: pageSize, ...filters })}`),

    detail: (key: string) =>
        api.get<AdminContestDetail>(`/admin/contest/${key}`),

    update: (key: string, data: AdminContestUpdateRequest) =>
        api.patch(`/admin/contest/${key}`, data),

// adminProblemApi
    list: (page: number = 1, pageSize: number = 50, filters: { search?: string; is_public?: 'true' | 'false'; group_id?: number } = {}) =>
        api.get<AdminProblemListResponse>(`/admin/problems?${qs({ page, page_size: pageSize, ...filters })}`),

    detail: (code: string) =>
        api.get<AdminProblemDetail>(`/admin/problem/${code}`),

    update: (code: string, data: AdminProblemUpdateRequest) =>
        api.patch(`/admin/problem/${code}`, data),

// adminOrganizationApi
    list: (page: number = 1, pageSize: number = 50, search?: string) =>
        api.get<AdminOrganizationListResponse>(`/admin/organizations?${qs({ page, page_size: pageSize, search })}`),

// adminSubmissionApi
    list: (page: number = 1, pageSize: number = 50, filters: { user?: string; problem?: string; result?: string; status?: string; language?: string } = {}) =>
        api.get<AdminSubmissionListResponse>(`/admin/submissions?${qs({ page, page_size: pageSize, ...filters })}`),
```

Delete `ContestProblem`, `ProblemAuthor`, `ProblemTypeRef` interfaces at the top of the file if nothing else references them after this change (`npx tsc` in Step 6 will tell you).

- [ ] **Step 5: Write `adminForms.ts`**

`claoj-web/src/lib/adminForms.ts`:

```ts
// Pure mappers between the admin API detail shapes and the edit forms.
// Kept free of React so the round-trips are unit-tested.
import type {
    AdminProblemDetail,
    AdminContestDetail,
    AdminContestProblemRow,
    AdminContestUpdateRequest,
    AdminContestCreateRequest,
    ContestProblemInput,
} from '@/types';
import type { ProblemFormData } from '@/components/admin/ProblemForm';
import type { ContestFormData } from '@/components/admin/ContestForm';

const pad = (n: number) => String(n).padStart(2, '0');

/** ISO timestamp → `datetime-local` value in the browser's time zone. */
export function toLocalInput(iso: string): string {
    if (!iso) return '';
    const d = new Date(iso);
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** `datetime-local` value (browser time zone) → RFC 3339 UTC. */
export function fromLocalInput(local: string): string {
    return new Date(local).toISOString();
}

export function problemToFormData(p: AdminProblemDetail): ProblemFormData {
    return {
        code: p.code,
        name: p.name,
        description: p.description,
        points: p.points,
        partial: p.partial,
        is_public: p.is_public,
        time_limit: p.time_limit,
        memory_limit: p.memory_limit,
        group_id: p.group_id,
        type_ids: [...p.type_ids],
        author_ids: [...p.author_ids],
        allowed_lang_ids: [...p.allowed_lang_ids],
        is_manually_managed: p.is_manually_managed,
        pdf_url: p.pdf_url,
    };
}

export function contestToFormData(c: AdminContestDetail): ContestFormData {
    return {
        key: c.key,
        name: c.name,
        description: c.description,
        summary: c.summary,
        start_time: toLocalInput(c.start_time),
        end_time: toLocalInput(c.end_time),
        time_limit: c.time_limit ? Math.round(c.time_limit / 60) : undefined,
        is_visible: c.is_visible,
        is_rated: c.is_rated,
        format_name: c.format_name || 'default',
        format_config: c.format_config ? JSON.stringify(c.format_config) : '',
        access_code: c.access_code || '',
        hide_problem_tags: c.hide_problem_tags,
        run_pretests_only: c.run_pretests_only,
        is_organization_private: c.is_organization_private,
        author_ids: [...c.author_ids],
        curator_ids: [...c.curator_ids],
        tester_ids: [...c.tester_ids],
        tag_ids: [...c.tag_ids],
        problem_ids: c.problems.map(p => p.problem_id),
    };
}

/** Minutes from the form → seconds for the API (undefined when empty). */
const minutesToSeconds = (m: number | undefined) =>
    m && Number.isFinite(m) && m > 0 ? Math.round(m * 60) : undefined;

function sharedContestFields(data: ContestFormData) {
    return {
        name: data.name,
        description: data.description,
        summary: data.summary,
        start_time: fromLocalInput(data.start_time),
        end_time: fromLocalInput(data.end_time),
        is_visible: data.is_visible,
        is_rated: data.is_rated,
        format_name: data.format_name,
        format_config: data.format_config ?? '',
        access_code: data.access_code ?? '',
        hide_problem_tags: data.hide_problem_tags,
        run_pretests_only: data.run_pretests_only,
        is_organization_private: data.is_organization_private,
        author_ids: data.author_ids ?? [],
        curator_ids: data.curator_ids ?? [],
        tester_ids: data.tester_ids ?? [],
        tag_ids: data.tag_ids ?? [],
    };
}

export function contestFormToUpdate(data: ContestFormData, existing: AdminContestProblemRow[]): AdminContestUpdateRequest {
    const byProblem = new Map(existing.map(row => [row.problem_id, row]));
    const problems: ContestProblemInput[] = (data.problem_ids ?? []).map(id => {
        const row = byProblem.get(id);
        return row
            ? { problem_id: id, points: row.points, partial: row.partial, is_pretested: row.is_pretested }
            : { problem_id: id };
    });
    return {
        ...sharedContestFields(data),
        time_limit: minutesToSeconds(data.time_limit) ?? 0,
        problems,
    };
}

export function contestFormToCreate(data: ContestFormData): AdminContestCreateRequest {
    return {
        ...sharedContestFields(data),
        key: data.key,
        time_limit: minutesToSeconds(data.time_limit),
        problem_ids: data.problem_ids ?? [],
    } as AdminContestCreateRequest;
}
```

- [ ] **Step 6: Run the tests and the type check**

Run: `npx jest __tests__/lib/adminForms.test.ts && npx tsc --noEmit -p . 2>&1 | grep '^src/' | head -40`
Expected: tests PASS. `tsc` will now report errors in the two edit pages, `PdfTab.tsx`, and anywhere using the old `detail` shapes — those are fixed in Task 8; there must be no errors in `src/lib` or `src/types`.

- [ ] **Step 7: Commit**

```bash
git add claoj-web/src/types/index.ts claoj-web/src/lib/adminApi.ts claoj-web/src/lib/adminForms.ts claoj-web/__tests__/lib/adminForms.test.ts
git commit -F - <<'EOF'
feat(admin): typed admin detail contracts and pure form mappers

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 8: Search-as-you-type pickers and working edit/create forms

**Files:**
- Create: `claoj-web/src/components/admin/EntityPicker.tsx`, `claoj-web/src/lib/adminSearch.ts`
- Modify: `claoj-web/src/components/admin/ProblemForm.tsx`, `problem-form/AuthorsSection.tsx`
- Modify: `claoj-web/src/components/admin/ContestForm.tsx`, `contest-form/StaffSection.tsx`, `contest-form/ProblemsSection.tsx`, `contest-form/ScheduleSection.tsx`
- Modify: `claoj-web/src/app/[locale]/admin/problems/[code]/edit/page.tsx`, `claoj-web/src/app/[locale]/admin/contests/[key]/edit/page.tsx`, `claoj-web/src/app/[locale]/admin/contests/create/page.tsx`
- Modify: `claoj-web/src/i18n/en.json`, `claoj-web/src/i18n/vi.json`
- Test: `claoj-web/__tests__/components/EntityPicker.test.tsx`

**Interfaces:**
- Consumes: Task 7 API + mappers.
- Produces: `EntityPicker` props `{ id: string; label: string; value: number[]; onChange(ids: number[]): void; search(term: string): Promise<EntityOption[]>; queryKey: string; initialOptions?: EntityOption[]; placeholder?: string }`, `EntityOption { id: number; label: string; detail?: string }`; `searchUsers`, `searchProblems`, `searchContests` (each `(term: string) => Promise<EntityOption[]>`). `ContestFormProps` gains `initialOptions?: { problems?: EntityOption[]; authors?: EntityOption[]; curators?: EntityOption[]; testers?: EntityOption[] }`; `ProblemFormProps` gains `initialAuthors?: EntityOption[]`. `ContestForm.onSubmit` keeps receiving raw `ContestFormData`; pages convert with the mappers.

- [ ] **Step 1: Write the failing picker test**

`claoj-web/__tests__/components/EntityPicker.test.tsx`:

```tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { EntityPicker, EntityOption } from '@/components/admin/EntityPicker';

function renderPicker(props: Partial<React.ComponentProps<typeof EntityPicker>> = {}) {
    const onChange = jest.fn();
    const search = jest.fn(async (term: string): Promise<EntityOption[]> =>
        [{ id: 7, label: `${term}-match`, detail: 'Seven' }]);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
        <QueryClientProvider client={client}>
            <EntityPicker
                id="authors"
                label="Authors"
                value={[4]}
                onChange={onChange}
                search={search}
                queryKey="users"
                initialOptions={[{ id: 4, label: 'alice' }]}
                {...props}
            />
        </QueryClientProvider>
    );
    return { onChange, search };
}

describe('EntityPicker', () => {
    it('shows labels for preselected ids', () => {
        renderPicker();
        expect(screen.getByText('alice')).toBeInTheDocument();
    });

    it('searches after typing and adds the clicked option', async () => {
        const { onChange, search } = renderPicker();
        fireEvent.change(screen.getByRole('combobox'), { target: { value: 'bo' } });
        const option = await screen.findByRole('option', { name: /bo-match/ });
        expect(search).toHaveBeenCalledWith('bo');
        fireEvent.click(option);
        expect(onChange).toHaveBeenCalledWith([4, 7]);
    });

    it('removes a chip', () => {
        const { onChange } = renderPicker();
        fireEvent.click(screen.getByRole('button', { name: 'picker.remove' }));
        expect(onChange).toHaveBeenCalledWith([]);
    });

    it('does not search for an empty term', async () => {
        const { search } = renderPicker();
        fireEvent.change(screen.getByRole('combobox'), { target: { value: '   ' } });
        await waitFor(() => expect(search).not.toHaveBeenCalled());
    });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `npx jest __tests__/components/EntityPicker.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `EntityPicker` and the search helpers**

`claoj-web/src/components/admin/EntityPicker.tsx`:

```tsx
'use client';

import { useMemo, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { X } from 'lucide-react';
import { useDebounce } from '@/hooks/useDebounce';
import { useOnClickOutside } from '@/hooks/useOnClickOutside';
import { cn } from '@/lib/utils';

export interface EntityOption {
    id: number;
    label: string;
    detail?: string;
}

interface EntityPickerProps {
    id: string;
    label: string;
    value: number[];
    onChange: (ids: number[]) => void;
    search: (term: string) => Promise<EntityOption[]>;
    /** Cache namespace for results, e.g. "users". */
    queryKey: string;
    /** Labels for ids already selected when the form opens. */
    initialOptions?: EntityOption[];
    placeholder?: string;
}

/**
 * Multi-select that searches the server as you type, so it works for
 * thousands of users/problems (list endpoints cap a page at 100). Selected
 * items render as removable chips in selection order.
 */
export function EntityPicker({ id, label, value, onChange, search, queryKey, initialOptions = [], placeholder }: EntityPickerProps) {
    const t = useTranslations('Admin');
    const [term, setTerm] = useState('');
    const [open, setOpen] = useState(false);
    const [learned, setLearned] = useState<Record<number, EntityOption>>({});
    const debounced = useDebounce(term.trim(), 250);
    const boxRef = useRef<HTMLDivElement>(null);
    useOnClickOutside(boxRef, () => setOpen(false));

    const known = useMemo(() => {
        const map: Record<number, EntityOption> = {};
        for (const o of initialOptions) map[o.id] = o;
        return { ...map, ...learned };
    }, [initialOptions, learned]);

    const { data: results = [], isFetching } = useQuery({
        queryKey: ['entity-picker', queryKey, debounced],
        queryFn: () => search(debounced),
        enabled: open && debounced.length > 0,
        staleTime: 30_000,
    });
    const visible = results.filter(o => !value.includes(o.id));

    const add = (o: EntityOption) => {
        setLearned(prev => ({ ...prev, [o.id]: o }));
        onChange([...value, o.id]);
        setTerm('');
    };

    return (
        <div ref={boxRef} className="space-y-2">
            <label htmlFor={id} className="text-sm font-medium text-muted-foreground block">{label}</label>
            {value.length > 0 && (
                <ul className="flex flex-wrap gap-1.5">
                    {value.map(v => {
                        const o = known[v];
                        const text = o?.label ?? `#${v}`;
                        return (
                            <li key={v} className="inline-flex items-center gap-1 rounded-md border bg-muted/50 pl-2 pr-1 py-0.5 text-sm">
                                <span>{text}</span>
                                {o?.detail && <span className="text-xs text-muted-foreground">{o.detail}</span>}
                                <button
                                    type="button"
                                    aria-label={t('picker.remove', { label: text })}
                                    onClick={() => onChange(value.filter(x => x !== v))}
                                    className="p-0.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
                                >
                                    <X size={14} />
                                </button>
                            </li>
                        );
                    })}
                </ul>
            )}
            <div className="relative">
                <input
                    id={id}
                    role="combobox"
                    aria-expanded={open && debounced.length > 0}
                    aria-controls={`${id}-listbox`}
                    aria-autocomplete="list"
                    autoComplete="off"
                    value={term}
                    placeholder={placeholder ?? t('picker.searchPlaceholder')}
                    onChange={e => { setTerm(e.target.value); setOpen(true); }}
                    onFocus={() => setOpen(true)}
                    onKeyDown={e => {
                        if (e.key === 'Escape') setOpen(false);
                        if (e.key === 'Enter') {
                            e.preventDefault();
                            if (visible[0]) add(visible[0]);
                        }
                    }}
                    className="w-full h-9 px-3 rounded-md border bg-card text-sm focus:outline-none focus:ring-2 focus:ring-ring/40"
                />
                {open && debounced.length > 0 && (
                    <ul
                        id={`${id}-listbox`}
                        role="listbox"
                        className="absolute z-20 mt-1 w-full max-h-60 overflow-y-auto rounded-md border bg-popover shadow-lg"
                    >
                        {isFetching && visible.length === 0 ? (
                            <li className="px-3 py-2 text-sm text-muted-foreground">{t('picker.searching')}</li>
                        ) : visible.length === 0 ? (
                            <li className="px-3 py-2 text-sm text-muted-foreground">{t('picker.noResults')}</li>
                        ) : visible.map(o => (
                            <li
                                key={o.id}
                                role="option"
                                aria-selected={false}
                                onMouseDown={e => e.preventDefault()}
                                onClick={() => add(o)}
                                className={cn('px-3 py-2 text-sm cursor-pointer hover:bg-muted flex items-baseline gap-2')}
                            >
                                <span className="font-medium">{o.label}</span>
                                {o.detail && <span className="text-xs text-muted-foreground truncate">{o.detail}</span>}
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </div>
    );
}
```

`claoj-web/src/lib/adminSearch.ts`:

```ts
// Server-side search functions for EntityPicker.
import { adminUserApi, adminProblemApi, adminContestApi } from '@/lib/adminApi';
import type { EntityOption } from '@/components/admin/EntityPicker';

const LIMIT = 20;

export async function searchUsers(term: string): Promise<EntityOption[]> {
    const res = await adminUserApi.list(1, LIMIT, term);
    return res.data.data.map(u => ({ id: u.id, label: u.username }));
}

export async function searchProblems(term: string): Promise<EntityOption[]> {
    const res = await adminProblemApi.list(1, LIMIT, { search: term });
    return res.data.data.map(p => ({ id: p.id, label: p.code, detail: p.name }));
}

export async function searchContests(term: string): Promise<EntityOption[]> {
    const res = await adminContestApi.list(1, LIMIT, { search: term });
    return res.data.data.map(c => ({ id: c.id, label: c.key, detail: c.name }));
}
```

Add i18n keys (`Admin.picker`) to both locales:

| key | en | vi |
|---|---|---|
| `Admin.picker.searchPlaceholder` | `Type to search…` | `Nhập để tìm…` |
| `Admin.picker.searching` | `Searching…` | `Đang tìm…` |
| `Admin.picker.noResults` | `No matches` | `Không có kết quả` |
| `Admin.picker.remove` | `Remove {label}` | `Bỏ {label}` |

- [ ] **Step 4: Run the picker test**

Run: `npx jest __tests__/components/EntityPicker.test.tsx`
Expected: PASS. (The `next-intl` mock returns the key, so the remove button's name is `picker.remove`.)

- [ ] **Step 5: Use the picker in the problem form**

Replace `claoj-web/src/components/admin/problem-form/AuthorsSection.tsx` with:

```tsx
'use client';

import { useTranslations } from 'next-intl';
import { EntityPicker, EntityOption } from '../EntityPicker';
import { searchUsers } from '@/lib/adminSearch';

interface Language {
    id: number;
    name: string;
    key: string;
}

interface AuthorsSectionProps {
    languages?: { data: Language[] };
    selectedAuthors: number[];
    initialAuthors?: EntityOption[];
    selectedLangs: number[];
    onAuthorsChange: (ids: number[]) => void;
    onLangToggle: (langId: number, checked: boolean) => void;
}

export function AuthorsSection({ languages, selectedAuthors, initialAuthors, selectedLangs, onAuthorsChange, onLangToggle }: AuthorsSectionProps) {
    const t = useTranslations('Problems');

    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">{t('authorsAndLanguages')}</h3>

            <EntityPicker
                id="problem-authors"
                label={t('authors')}
                value={selectedAuthors}
                onChange={onAuthorsChange}
                search={searchUsers}
                queryKey="users"
                initialOptions={initialAuthors}
            />

            <div>
                <label htmlFor="allowed-languages" className="text-sm font-medium text-muted-foreground block mb-2">
                    {t('allowedLanguages')}
                </label>
                <div id="allowed-languages" className="grid grid-cols-2 md:grid-cols-3 gap-2" role="group" aria-labelledby="allowed-languages">
                    {languages?.data.map(lang => (
                        <label key={lang.id} className="flex items-center gap-2 p-2 rounded-lg border cursor-pointer hover:bg-muted/30">
                            <input
                                type="checkbox"
                                checked={selectedLangs.includes(lang.id)}
                                onChange={(e) => onLangToggle(lang.id, e.target.checked)}
                                className="rounded"
                            />
                            <span className="text-sm">{lang.name}</span>
                        </label>
                    ))}
                </div>
            </div>
        </div>
    );
}
```

In `claoj-web/src/components/admin/ProblemForm.tsx`:
- Add `import type { EntityOption } from './EntityPicker';` and add `initialAuthors?: EntityOption[];` to `ProblemFormProps`; destructure it: `export default function ProblemForm({ initialData, initialAuthors, onSubmit, isLoading }: ProblemFormProps)`.
- In `defaultValues`, add `group_id: initialData?.group_id, type_ids: initialData?.type_ids ?? [], author_ids: initialData?.author_ids ?? [], allowed_lang_ids: initialData?.allowed_lang_ids ?? [],` (today they are not seeded, which is why `watch()` starts empty).
- Delete the `UserProfile` interface and the whole `const { data: users } = useQuery(...)` block.
- Replace the `<AuthorsSection ... />` element with:

```tsx
            <AuthorsSection
                languages={languages}
                selectedAuthors={selectedAuthors}
                initialAuthors={initialAuthors}
                selectedLangs={selectedLangs}
                onAuthorsChange={(ids) => setValue('author_ids', ids)}
                onLangToggle={(langId, checked) => handleMultiSelect('allowed_lang_ids', langId, checked)}
            />
```

- [ ] **Step 6: Use the pickers in the contest form, fix schedule units**

Replace `claoj-web/src/components/admin/contest-form/StaffSection.tsx` with:

```tsx
'use client';

import { useTranslations } from 'next-intl';
import { EntityPicker, EntityOption } from '../EntityPicker';
import { searchUsers } from '@/lib/adminSearch';

type StaffField = 'author_ids' | 'curator_ids' | 'tester_ids';

interface StaffSectionProps {
    selected: Record<StaffField, number[]>;
    initialOptions?: Partial<Record<StaffField, EntityOption[]>>;
    onChange: (field: StaffField, ids: number[]) => void;
}

export function StaffSection({ selected, initialOptions = {}, onChange }: StaffSectionProps) {
    const t = useTranslations('Admin');
    const fields: { field: StaffField; label: string }[] = [
        { field: 'author_ids', label: t('contestForm.authorsLabel') },
        { field: 'curator_ids', label: t('contestForm.curatorsLabel') },
        { field: 'tester_ids', label: t('contestForm.testersLabel') },
    ];
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">{t('contestForm.staffTitle')}</h3>
            {fields.map(({ field, label }) => (
                <EntityPicker
                    key={field}
                    id={`contest-${field}`}
                    label={label}
                    value={selected[field]}
                    onChange={ids => onChange(field, ids)}
                    search={searchUsers}
                    queryKey="users"
                    initialOptions={initialOptions[field]}
                />
            ))}
        </div>
    );
}
```

Replace `claoj-web/src/components/admin/contest-form/ProblemsSection.tsx` with:

```tsx
'use client';

import { useTranslations } from 'next-intl';
import { EntityPicker, EntityOption } from '../EntityPicker';
import { searchProblems } from '@/lib/adminSearch';

interface ProblemsSectionProps {
    selectedProblems: number[];
    initialOptions?: EntityOption[];
    onChange: (ids: number[]) => void;
}

/** Selection order is contest problem order. */
export function ProblemsSection({ selectedProblems, initialOptions, onChange }: ProblemsSectionProps) {
    const t = useTranslations('Admin');
    return (
        <div className="bg-card rounded-2xl border p-6 space-y-4">
            <h3 className="text-lg font-bold">{t('contestForm.problemsTitle')}</h3>
            <p className="text-sm text-muted-foreground">{t('contestForm.problemsDesc')}</p>
            <EntityPicker
                id="contest-problems"
                label={t('contestForm.problemsTitle')}
                value={selectedProblems}
                onChange={onChange}
                search={searchProblems}
                queryKey="problems"
                initialOptions={initialOptions}
            />
            {selectedProblems.length > 0 && (
                <div className="text-sm text-muted-foreground">
                    {t('contestForm.selectedProblemsCount', { count: selectedProblems.length })}
                </div>
            )}
        </div>
    );
}
```

In `claoj-web/src/components/admin/contest-form/ScheduleSection.tsx`, replace the time-limit `<input .../>` with:

```tsx
                <input
                    type="number"
                    min="1"
                    step="1"
                    inputMode="numeric"
                    className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                    placeholder={t('contestForm.timeLimitPlaceholder')}
                    {...register('time_limit', { setValueAs: (v: string) => (v === '' || v == null ? undefined : Number(v)) })}
                />
```

In `claoj-web/src/components/admin/ContestForm.tsx`:
- Add `import type { EntityOption } from './EntityPicker';` and extend `ContestFormProps` with `initialOptions?: { problems?: EntityOption[]; author_ids?: EntityOption[]; curator_ids?: EntityOption[]; tester_ids?: EntityOption[] };`; destructure it.
- Change `format_name: initialData?.format_name || 'icpc',` to `format_name: initialData?.format_name || 'default',`, and add to `defaultValues`: `author_ids: initialData?.author_ids ?? [], curator_ids: initialData?.curator_ids ?? [], tester_ids: initialData?.tester_ids ?? [], problem_ids: initialData?.problem_ids ?? [], tag_ids: initialData?.tag_ids ?? [],`.
- Delete the `Problem` and `UserProfile` interfaces, the `problems` and `users` `useQuery` blocks, `handleMultiSelect`, and `handleProblemSelect`. Keep the tags query (tags are few) and `handleTagSelect`.
- Replace the `<ProblemsSection .../>` and `<StaffSection .../>` elements with:

```tsx
            <ProblemsSection
                selectedProblems={selectedProblems}
                initialOptions={initialOptions?.problems}
                onChange={(ids) => setValue('problem_ids', ids)}
            />
```

```tsx
            <StaffSection
                selected={{ author_ids: selectedAuthors, curator_ids: selectedCurators, tester_ids: selectedTesters }}
                initialOptions={initialOptions}
                onChange={(field, ids) => setValue(field, ids)}
            />
```

- Change `selectedFormat={selectedFormat || 'icpc'}` to `selectedFormat={selectedFormat || 'default'}`.

Update i18n (both locales):

| key | en | vi |
|---|---|---|
| `Admin.contestForm.timeLimitLabel` | `Time window per participant (minutes, optional)` | `Thời gian làm bài cho mỗi thí sinh (phút, không bắt buộc)` |
| `Admin.contestForm.timeLimitPlaceholder` | `Empty: everyone has the whole contest` | `Để trống: thí sinh làm trong toàn bộ thời gian kỳ thi` |
| `Admin.problems.saved` | `Problem saved` | `Đã lưu bài tập` |
| `Admin.problems.saveFailed` | `Could not save: {error}` | `Không lưu được: {error}` |
| `Admin.contests.saved` | `Contest saved` | `Đã lưu kỳ thi` |
| `Admin.contests.saveFailed` | `Could not save: {error}` | `Không lưu được: {error}` |

- [ ] **Step 7: Wire the edit and create pages**

Problem edit page `claoj-web/src/app/[locale]/admin/problems/[code]/edit/page.tsx`:
- Imports: add `import { toast } from 'sonner';`, `import { problemToFormData } from '@/lib/adminForms';`, and `import { useQueryClient } from '@tanstack/react-query';` (merge into the existing react-query import).
- Replace the `updateMutation` with:

```tsx
    const queryClient = useQueryClient();
    const updateMutation = useMutation({
        mutationFn: ({ code, data }: { code: string; data: ProblemFormData }) =>
            adminProblemApi.update(code, data),
        onSuccess: () => {
            toast.success(t('problems.saved'));
            queryClient.invalidateQueries({ queryKey: ['admin-problem-detail', code] });
        },
        onError: (err: unknown) => {
            const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? String(err);
            toast.error(t('problems.saveFailed', { error: msg }));
        },
    });
```

  (Rename the detail `queryKey` to `['admin-problem-detail', code]` so it shares the cache with `PdfTab`.) Remove the now-unused `router` / `useRouter` if nothing else uses it.
- Replace the whole `const initialData = problemData ? {...} : undefined;` block with:

```tsx
    const initialData = problemData ? problemToFormData(problemData) : undefined;
    const initialAuthors = problemData?.authors.map(a => ({ id: a.id, label: a.username }));
```

  and pass `initialAuthors={initialAuthors}` to `<ProblemForm>`. Make sure the form only renders once `problemData` exists (render the loading state while `isLoadingProblem || !problemData`), so `defaultValues` are seeded from real data.

Contest edit page `claoj-web/src/app/[locale]/admin/contests/[key]/edit/page.tsx`:
- Imports: `import { toast } from 'sonner';`, `import { contestToFormData, contestFormToUpdate } from '@/lib/adminForms';`.
- Replace `updateMutation` with:

```tsx
    const updateMutation = useMutation({
        mutationFn: (data: ContestFormData) =>
            adminContestApi.update(key, contestFormToUpdate(data, contestData?.problems ?? [])),
        onSuccess: () => {
            toast.success(t('contests.saved'));
            queryClient.invalidateQueries({ queryKey: ['admin-contest-detail', key] });
        },
        onError: (err: unknown) => {
            const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? String(err);
            toast.error(t('contests.saveFailed', { error: msg }));
        },
    });

    const handleSubmit = async (data: ContestFormData) => {
        await updateMutation.mutateAsync(data);
    };
```

- Replace the `const contest = contestData; const problems = ...; const initialData = contest ? {...} : undefined;` block with:

```tsx
    const initialData = contestData ? contestToFormData(contestData) : undefined;
    const initialOptions = contestData ? {
        problems: contestData.problems.map(p => ({ id: p.problem_id, label: p.code, detail: p.name })),
        author_ids: contestData.authors.map(u => ({ id: u.id, label: u.username })),
        curator_ids: contestData.curators.map(u => ({ id: u.id, label: u.username })),
        tester_ids: contestData.testers.map(u => ({ id: u.id, label: u.username })),
    } : undefined;
```

  and pass `initialOptions={initialOptions}` to `<ContestForm>`. Delete the local `formatDateTimeLocal` helper at the bottom of the file and any remaining reference to `contest`/`problems` locals (use `contestData`). Render the loading state while `isLoadingContest || !contestData`.

Contest create page `claoj-web/src/app/[locale]/admin/contests/create/page.tsx`:
- Import `contestFormToCreate` from `@/lib/adminForms` and change the mutation to `mutationFn: (data: ContestFormData) => adminContestApi.create(contestFormToCreate(data)),`.

Problem create page needs no change (create already uses full lists).

- [ ] **Step 8: Type check, lint, tests, i18n**

Run:

```bash
npx tsc --noEmit -p . 2>&1 | grep '^src/' ; npx jest ; npm run -s i18n:check ; npm run -s i18n:usage
npx eslint src/components/admin src/lib/adminForms.ts src/lib/adminSearch.ts "src/app/[locale]/admin/problems" "src/app/[locale]/admin/contests"
```

Expected: no `src/` type errors; all jest suites pass; both i18n checks OK; no new ESLint errors in the touched files (pre-existing `no-explicit-any` in untouched code is acceptable).

- [ ] **Step 9: Manual check against the dev backend**

With the backend running (`go run .` in `claoj/` against a dev DB) and `npm run dev`, as a superuser:
1. Open an existing problem's edit page: all fields, authors (by username), types, languages and group are pre-filled; the PDF tab shows the current PDF.
2. Add an author found by search (any user, not just the first 100), save: toast "Problem saved"; reload: author persists.
3. Create a contest with start/end times and a 90-minute window: it saves (it returned 400 before). Edit it: times show the same wall-clock values; window shows 90.
4. In a contest with submissions, remove a solved problem and save: toast shows the 409 message naming the problem; reload: nothing changed.

- [ ] **Step 10: Commit**

```bash
git add claoj-web/src claoj-web/__tests__/components/EntityPicker.test.tsx
git commit -F - <<'EOF'
fix(admin): working problem/contest edit and create forms with searchable pickers

Forms now load from the flat detail contract, keep per-problem contest
settings, send RFC 3339 times and minute-based windows, and pick from
every user/problem instead of the first 100.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 9: URL-backed list filters on problems, contests, organizations, submissions

**Files:**
- Create: `claoj-web/src/hooks/useListParams.ts`
- Modify: `claoj-web/src/app/[locale]/admin/layout.tsx` (Suspense)
- Modify: `claoj-web/src/app/[locale]/admin/{problems,contests,organizations,submissions}/page.tsx`
- Modify: `claoj-web/src/i18n/en.json`, `claoj-web/src/i18n/vi.json`
- Test: `claoj-web/__tests__/hooks/listParams.test.ts`

**Interfaces:**
- Consumes: Task 7 `adminProblemApi.list` / `adminContestApi.list` / `adminOrganizationApi.list` / `adminSubmissionApi.list` filter signatures; `PaginationBar` (`src/components/ui/PaginationBar.tsx`, props `page, onPageChange, total, pageSize`).
- Produces: `readListParams(sp: URLSearchParams, keys: readonly string[]): { page: number; filters: Record<string, string> }`, `buildListQuery(page: number, filters: Record<string, string>): string`, `useListParams(keys: readonly string[]): { page; filters; setFilters(patch: Record<string, string>): void; setPage(p: number): void; reset(): void }`.

- [ ] **Step 1: Write the failing test**

`claoj-web/__tests__/hooks/listParams.test.ts`:

```ts
import { readListParams, buildListQuery } from '@/hooks/useListParams';

describe('list params', () => {
    it('reads page and only the named filters', () => {
        const sp = new URLSearchParams('page=3&search=dp&junk=1&status=');
        expect(readListParams(sp, ['search', 'status'])).toEqual({ page: 3, filters: { search: 'dp' } });
    });

    it('clamps a bad page to 1', () => {
        expect(readListParams(new URLSearchParams('page=-2'), []).page).toBe(1);
        expect(readListParams(new URLSearchParams('page=abc'), []).page).toBe(1);
    });

    it('builds a query omitting empty filters and page 1', () => {
        expect(buildListQuery(1, { search: 'a b', status: '' })).toBe('?search=a+b');
        expect(buildListQuery(2, {})).toBe('?page=2');
        expect(buildListQuery(1, {})).toBe('');
    });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `npx jest __tests__/hooks/listParams.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement the hook**

`claoj-web/src/hooks/useListParams.ts`:

```ts
'use client';

import { useCallback, useMemo } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';

export type ListFilters = Record<string, string>;

/** Pure: page (>= 1) and the named, non-empty filters from a query string. */
export function readListParams(sp: URLSearchParams, keys: readonly string[]): { page: number; filters: ListFilters } {
    const page = Math.max(1, parseInt(sp.get('page') ?? '1', 10) || 1);
    const filters: ListFilters = {};
    for (const k of keys) {
        const v = sp.get(k);
        if (v) filters[k] = v;
    }
    return { page, filters };
}

/** Pure: query string for page + non-empty filters; page 1 is implied. */
export function buildListQuery(page: number, filters: ListFilters): string {
    const sp = new URLSearchParams();
    for (const [k, v] of Object.entries(filters)) if (v) sp.set(k, v);
    if (page > 1) sp.set('page', String(page));
    const s = sp.toString();
    return s ? `?${s}` : '';
}

/**
 * List page + filters kept in the URL so a filtered view is linkable and
 * survives reload. Changing a filter resets to page 1. Pass a module-level
 * constant for `keys`.
 */
export function useListParams(keys: readonly string[]) {
    const router = useRouter();
    const pathname = usePathname();
    const searchParams = useSearchParams();
    const { page, filters } = useMemo(
        () => readListParams(new URLSearchParams(searchParams.toString()), keys),
        [searchParams, keys]
    );

    const go = useCallback(
        (p: number, f: ListFilters) => router.replace(pathname + buildListQuery(p, f), { scroll: false }),
        [router, pathname]
    );
    const setFilters = useCallback((patch: ListFilters) => go(1, { ...filters, ...patch }), [go, filters]);
    const setPage = useCallback((p: number) => go(p, filters), [go, filters]);
    const reset = useCallback(() => go(1, {}), [go]);

    return { page, filters, setFilters, setPage, reset };
}
```

In `claoj-web/src/app/[locale]/admin/layout.tsx`, add `Suspense` to the React import (`import { Suspense, useEffect, useState } from 'react';`) and wrap the page:

```tsx
                <main className="p-4 lg:p-8 max-w-[1400px]">
                    <Suspense fallback={null}>{children}</Suspense>
                </main>
```

- [ ] **Step 4: Run the hook test**

Run: `npx jest __tests__/hooks/listParams.test.ts`
Expected: PASS.

- [ ] **Step 5: Problems page**

In `claoj-web/src/app/[locale]/admin/problems/page.tsx`:
- Imports: replace `import api from '@/lib/api';` with `import { adminProblemGroupApi } from '@/lib/adminApi';` merged into the existing `adminApi` import; add `import { useListParams } from '@/hooks/useListParams';`, `import { useDebounce } from '@/hooks/useDebounce';`, `import { PaginationBar } from '@/components/ui/PaginationBar';`. Remove `AdminProblem` from `@/types` if unused afterwards.
- Above the component: `const FILTER_KEYS = ['search', 'is_public', 'group_id'] as const; const PAGE_SIZE = 50;`
- Replace `const [page, setPage] = useState(1); const [search, setSearch] = useState('');` and the `useQuery` with:

```tsx
    const { page, filters, setFilters, setPage } = useListParams(FILTER_KEYS);
    const search = useDebounce(filters.search ?? '', 300);

    const { data, isLoading } = useQuery({
        queryKey: ['admin-problems', page, search, filters.is_public, filters.group_id],
        queryFn: async () => (await adminProblemApi.list(page, PAGE_SIZE, {
            search,
            is_public: filters.is_public as 'true' | 'false' | undefined,
            group_id: filters.group_id ? Number(filters.group_id) : undefined,
        })).data,
    });

    const { data: groups } = useQuery({
        queryKey: ['problem-groups'],
        queryFn: async () => (await adminProblemGroupApi.list(1, 500)).data,
    });
```

- Replace `const problems = data?.data || []; const filteredProblems = problems.filter(...);` with `const problems = data?.data || [];` and rename every later `filteredProblems` to `problems`.
- Replace the search `<input ... value={search} onChange={(e) => setSearch(e.target.value)} />` with `value={filters.search ?? ''} onChange={(e) => setFilters({ search: e.target.value })}`, and add after the search box's wrapping `div`:

```tsx
                    <select
                        aria-label={t('problems.visibilityFilter')}
                        className="h-10 px-3 rounded-xl bg-card border"
                        value={filters.is_public ?? ''}
                        onChange={(e) => setFilters({ is_public: e.target.value })}
                    >
                        <option value="">{t('problems.visibilityAll')}</option>
                        <option value="true">{t('problems.publicBadge')}</option>
                        <option value="false">{t('problems.privateBadge')}</option>
                    </select>
                    <select
                        aria-label={t('problems.groupFilter')}
                        className="h-10 px-3 rounded-xl bg-card border"
                        value={filters.group_id ?? ''}
                        onChange={(e) => setFilters({ group_id: e.target.value })}
                    >
                        <option value="">{t('problems.groupAll')}</option>
                        {groups?.data.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
                    </select>
```

  Change the header's right-hand container `className="flex items-center gap-3"` to `className="flex flex-wrap items-center gap-3"`.
- Replace the whole `{/* Pagination */} {filteredProblems.length > 0 && (...)}` block with nothing, and after the closing `)}` of the `isLoading ? … : (…)` block add:

```tsx
            <PaginationBar page={page} onPageChange={setPage} total={data?.total} pageSize={PAGE_SIZE} />
```

- Remove `refetch` from destructuring if unused; remove `useState` import only if nothing else uses it (the clone modal does).

- [ ] **Step 6: Contests page**

Same pattern in `claoj-web/src/app/[locale]/admin/contests/page.tsx`:
- `const FILTER_KEYS = ['search', 'status'] as const; const PAGE_SIZE = 50;`
- State/query:

```tsx
    const { page, filters, setFilters, setPage } = useListParams(FILTER_KEYS);
    const search = useDebounce(filters.search ?? '', 300);

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-contests', page, search, filters.status],
        queryFn: async () => (await adminContestApi.list(page, PAGE_SIZE, {
            search,
            status: filters.status as 'upcoming' | 'ongoing' | 'past' | undefined,
        })).data,
    });
```

  (keep `refetch` if the clone flow uses it.)
- `const contests = data?.data || [];` and rename `filteredContests` → `contests`.
- Search input bound to `filters.search` / `setFilters({ search })`; add the status select after the search box:

```tsx
                    <select
                        aria-label={t('contests.statusFilter')}
                        className="h-10 px-3 rounded-xl bg-card border"
                        value={filters.status ?? ''}
                        onChange={(e) => setFilters({ status: e.target.value })}
                    >
                        <option value="">{t('contests.statusAll')}</option>
                        <option value="upcoming">{t('contests.statusUpcoming')}</option>
                        <option value="ongoing">{t('contests.statusOngoing')}</option>
                        <option value="past">{t('contests.statusPast')}</option>
                    </select>
```

- Replace the `{/* Pagination */}` block with nothing and add `<PaginationBar page={page} onPageChange={setPage} total={data?.total} pageSize={PAGE_SIZE} />` after the list block (before `{/* Clone Modal */}`).
- Remove `import api from '@/lib/api';` if unused.

- [ ] **Step 7: Organizations page**

In `claoj-web/src/app/[locale]/admin/organizations/page.tsx`:
- `const FILTER_KEYS = ['search'] as const; const PAGE_SIZE = 50;`
- Replace `const [search, setSearch] = useState('');` and the `useQuery` with:

```tsx
    const { page, filters, setFilters, setPage } = useListParams(FILTER_KEYS);
    const search = useDebounce(filters.search ?? '', 300);

    const { data, isLoading } = useQuery({
        queryKey: ['admin-organizations', page, search],
        queryFn: async () => (await adminOrganizationApi.list(page, PAGE_SIZE, search)).data,
    });
```

- `const organizations = data?.data || [];` and rename `filteredOrgs` → `organizations`.
- Search input → `value={filters.search ?? ''} onChange={(e) => setFilters({ search: e.target.value })}`.
- Insert `<PaginationBar page={page} onPageChange={setPage} total={data?.total} pageSize={PAGE_SIZE} />` immediately before `{/* Create Modal */}`.
- Remove `import api from '@/lib/api';` if unused; keep `refetch` only if still used.

- [ ] **Step 8: Submissions page**

In `claoj-web/src/app/[locale]/admin/submissions/page.tsx`:
- `const FILTER_KEYS = ['user', 'problem', 'result', 'status', 'language'] as const; const PAGE_SIZE = 50; const STATUS_CODES = ['QU', 'P', 'G'];`
- Replace the five `useState` filter/page lines and the `useQuery` with:

```tsx
    const { page, filters, setFilters, setPage, reset } = useListParams(FILTER_KEYS);
    const debounced = useDebounce(filters, 300);

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-submissions', page, debounced],
        queryFn: async () => (await adminSubmissionApi.list(page, PAGE_SIZE, debounced)).data,
    });
    const userFilter = filters.user ?? '';
    const problemFilter = filters.problem ?? '';
    const langFilter = filters.language ?? '';
    const verdictFilter = filters.result ?? filters.status ?? '';
```

- Change the `results` array to `const results = ['AC', 'WA', 'TLE', 'MLE', 'OLE', 'IR', 'RTE', 'CE', 'IE', 'AB', 'QU', 'P', 'G'];` (DMOJ's runtime-error code is `RTE`, not `RE`).
- In both batch-rejudge handlers, replace the four `if (...) filters.x = ...` lines (rename their local `filters` variable to `batch` to avoid shadowing) with:

```tsx
        const batch: Record<string, string> = {};
        if (userFilter) batch.username = userFilter;
        if (problemFilter) batch.problem_code = problemFilter;
        if (filters.result) batch.result = filters.result;
        if (filters.status) batch.status = filters.status;
        if (langFilter) batch.language = langFilter;
```

  and pass `filters: batch` to `adminSubmissionApi.batchRejudge(...)`.
- Inputs: user → `onChange={(e) => setFilters({ user: e.target.value })}`; problem → `setFilters({ problem: e.target.value })`; language → `setFilters({ language: e.target.value })`; the verdict select uses `value={verdictFilter}` and:

```tsx
                        onChange={(e) => {
                            const v = e.target.value;
                            setFilters(STATUS_CODES.includes(v) ? { status: v, result: '' } : { result: v, status: '' });
                        }}
```

- The reset button's `onClick` becomes `onClick={() => reset()}`.
- Remove `import api from '@/lib/api';` and `useState` for the removed state if now unused (the batch modal still uses `useState`).

- [ ] **Step 9: i18n keys**

Add to both locales:

| key | en | vi |
|---|---|---|
| `Admin.problems.visibilityFilter` | `Visibility` | `Hiển thị` |
| `Admin.problems.visibilityAll` | `All problems` | `Tất cả bài tập` |
| `Admin.problems.groupFilter` | `Group` | `Nhóm` |
| `Admin.problems.groupAll` | `All groups` | `Tất cả nhóm` |
| `Admin.contests.statusFilter` | `Status` | `Trạng thái` |
| `Admin.contests.statusAll` | `All contests` | `Tất cả kỳ thi` |
| `Admin.contests.statusUpcoming` | `Upcoming` | `Sắp diễn ra` |
| `Admin.contests.statusOngoing` | `Ongoing` | `Đang diễn ra` |
| `Admin.contests.statusPast` | `Past` | `Đã kết thúc` |

Remove `Admin.problems.showingCount` and `Admin.contests.showingCount` if no code references them anymore (`npm run i18n:usage` only checks the other direction; grep for `showingCount` before deleting).

- [ ] **Step 10: Verify**

```bash
npx tsc --noEmit -p . 2>&1 | grep '^src/' ; npx jest ; npm run -s i18n:check ; npm run -s i18n:usage
npx eslint src/hooks/useListParams.ts "src/app/[locale]/admin"
npx next build 2>&1 | tail -15
```

Expected: no `src/` type errors; tests pass; i18n OK; `next build` completes without a "useSearchParams() should be wrapped in a suspense boundary" error.

Manual: on `/admin/problems` type a code that is not on page 1 → it is found; reload the page → filters persist in the URL; filtered `total` drives the pagination bar. On `/admin/submissions?user=<name>&result=AC` only that user's ACs show.

- [ ] **Step 11: Commit**

```bash
git add claoj-web/src claoj-web/__tests__/hooks/listParams.test.ts
git commit -F - <<'EOF'
feat(admin): URL-backed server filters and shared pagination on admin lists

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 10: MOSS protocol client

**Files:**
- Replace: `claoj/moss/moss.go`
- Delete: `claoj/api/v2/moss.go` (the stub handlers; done in Task 11 together with their routes — in this task only the `moss` package changes, so first check `grep -rn "moss\." claoj --include=*.go` to see which call sites break; if `api/v2/moss.go` uses the old types, delete that file and the two routes in `api/router.go` lines `admin.POST("/admin/submission/:id/moss", ...)` and `admin.GET("/admin/submission/:id/moss", ...)` in this task so the build stays green)
- Test: `claoj/moss/moss_test.go`

**Interfaces:**
- Produces: `moss.Client{UserID, Addr string; MaxMatches, Show int; Timeout time.Duration}`, `moss.File{Name string; Content []byte}`, `(*moss.Client).Submit(lang, comment string, files []moss.File) (string, error)`, `moss.ErrLanguageRejected`, `moss.DefaultAddr = "moss.stanford.edu:7690"`.

- [ ] **Step 1: Write the failing test**

`claoj/moss/moss_test.go`:

```go
package moss

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type received struct {
	header []string
	files  map[string]string
}

// fakeServer speaks the MOSS server side once and records what it got.
func fakeServer(t *testing.T, acceptLanguage bool) (string, <-chan received) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	out := make(chan received, 1)
	go func() {
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		got := received{files: map[string]string{}}
		for i := 0; i < 6; i++ { // moss, directory, X, maxmatches, show, language
			line, _ := r.ReadString('\n')
			got.header = append(got.header, strings.TrimSpace(line))
		}
		if !acceptLanguage {
			fmt.Fprint(conn, "no\n")
			out <- got
			return
		}
		fmt.Fprint(conn, "yes\n")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "file ") {
				var id, size int
				var lang, name string
				fmt.Sscanf(line, "file %d %s %d %s", &id, &lang, &size, &name)
				buf := make([]byte, size)
				_, _ = io.ReadFull(r, buf)
				got.files[name] = string(buf)
				continue
			}
			if strings.HasPrefix(line, "query 0 ") {
				fmt.Fprint(conn, "http://moss.stanford.edu/results/1/2345\n")
				continue
			}
			if line == "end" {
				break
			}
		}
		out <- got
	}()
	return ln.Addr().String(), out
}

func TestSubmit_SpeaksTheProtocol(t *testing.T) {
	addr, got := fakeServer(t, true)
	c := &Client{UserID: "987654", Addr: addr, Show: 200}
	url, err := c.Submit("cc", "tik25 - knap", []File{
		{Name: "alice", Content: []byte("int main(){}\n")},
		{Name: "bob smith", Content: []byte("x")},
	})
	require.NoError(t, err)
	require.Equal(t, "http://moss.stanford.edu/results/1/2345", url)

	r := <-got
	require.Equal(t, []string{"moss 987654", "directory 0", "X 0", "maxmatches 10", "show 200", "language cc"}, r.header)
	require.Equal(t, "int main(){}\n", r.files["alice"])
	require.Equal(t, "x", r.files["bob_smith"], "spaces in names become underscores")
}

func TestSubmit_LanguageRejected(t *testing.T) {
	addr, _ := fakeServer(t, false)
	_, err := (&Client{UserID: "1", Addr: addr}).Submit("klingon", "c", []File{{Name: "a", Content: []byte("x")}})
	require.ErrorIs(t, err, ErrLanguageRejected)
}

func TestSubmit_ConnectionRefused(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	ln.Close()
	_, err = (&Client{UserID: "1", Addr: addr}).Submit("cc", "c", []File{{Name: "a", Content: []byte("x")}})
	require.Error(t, err)
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./moss/ -count=1`
Expected: FAIL to compile — `undefined: Client`, `File`, `ErrLanguageRejected` (old stub types differ).

- [ ] **Step 3: Replace `moss/moss.go`**

```go
// Package moss is a client for the MOSS (Measure Of Software Similarity)
// server protocol, the same exchange as the official moss script, mosspy,
// and the DMOJ moss library v1 uses.
package moss

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

const DefaultAddr = "moss.stanford.edu:7690"

// ErrLanguageRejected means the server does not support the language.
var ErrLanguageRejected = errors.New("moss: language not accepted by server")

// Client submits one batch of files per Submit call.
type Client struct {
	UserID     string        // MOSS account id (numeric string)
	Addr       string        // default DefaultAddr
	MaxMatches int           // -m: ignore code appearing in more files; default 10
	Show       int           // -n: matching file pairs to report; default 250
	Timeout    time.Duration // whole exchange; default 5 minutes
}

// File is one program; Name is how it appears in the report.
type File struct {
	Name    string
	Content []byte
}

// Submit uploads files in MOSS language lang (e.g. "cc", "java") and
// returns the report URL.
func (c *Client) Submit(lang, comment string, files []File) (string, error) {
	addr := c.Addr
	if addr == "" {
		addr = DefaultAddr
	}
	maxMatches, show, timeout := c.MaxMatches, c.Show, c.Timeout
	if maxMatches == 0 {
		maxMatches = 10
	}
	if show == 0 {
		show = 250
	}
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("moss: connect %s: %w", addr, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)

	fmt.Fprintf(w, "moss %s\n", c.UserID)
	fmt.Fprintf(w, "directory 0\n")
	fmt.Fprintf(w, "X 0\n")
	fmt.Fprintf(w, "maxmatches %d\n", maxMatches)
	fmt.Fprintf(w, "show %d\n", show)
	fmt.Fprintf(w, "language %s\n", lang)
	if err := w.Flush(); err != nil {
		return "", fmt.Errorf("moss: send header: %w", err)
	}
	reply, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("moss: read language reply: %w", err)
	}
	if strings.TrimSpace(reply) == "no" {
		fmt.Fprint(w, "end\n")
		_ = w.Flush()
		return "", fmt.Errorf("%w: %s", ErrLanguageRejected, lang)
	}

	for i, f := range files {
		name := strings.ReplaceAll(f.Name, " ", "_")
		fmt.Fprintf(w, "file %d %s %d %s\n", i+1, lang, len(f.Content), name)
		if _, err := w.Write(f.Content); err != nil {
			return "", fmt.Errorf("moss: upload %s: %w", name, err)
		}
	}
	fmt.Fprintf(w, "query 0 %s\n", comment)
	if err := w.Flush(); err != nil {
		return "", fmt.Errorf("moss: send files: %w", err)
	}

	url, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("moss: read result: %w", err)
	}
	url = strings.TrimSpace(url)
	fmt.Fprint(w, "end\n")
	_ = w.Flush()
	if !strings.HasPrefix(url, "http") {
		return "", fmt.Errorf("moss: unexpected reply %q", url)
	}
	return url, nil
}
```

- [ ] **Step 4: Remove the stub per-submission endpoints if the build breaks**

Run: `go build ./... 2>&1 | head`
If it reports errors in `api/v2/moss.go`, delete that file and remove these two lines from `claoj/api/router.go`:

```go
			admin.POST("/admin/submission/:id/moss", v2.AdminSubmissionMossAnalysis)
			admin.GET("/admin/submission/:id/moss", v2.AdminSubmissionMossResults)
```

Rerun `go build ./...` until clean (also remove any now-unused `MossAnalysisRequest` type references).

- [ ] **Step 5: Run the tests**

Run: `go test ./moss/ -count=1 && go build ./...`
Expected: PASS; build OK.

- [ ] **Step 6: Commit**

```bash
git add -A claoj/moss claoj/api/v2/moss.go claoj/api/router.go
git commit -F - <<'EOF'
feat(admin): real MOSS protocol client replacing the instructions stub

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 11: Contest MOSS job and endpoints

**Files:**
- Modify: `claoj/models/contest.go` (add `ContestMoss`)
- Create: `claoj/jobs/moss.go`, `claoj/jobs/moss_test.go`
- Modify: `claoj/jobs/server.go` (register handler)
- Create: `claoj/api/v2/admin_contest_moss.go`, `claoj/api/v2/admin_contest_moss_test.go`
- Modify: `claoj/api/router.go`

**Interfaces:**
- Consumes: `moss.Client`, `moss.File` (Task 10).
- Produces:
  - `models.ContestMoss{ID, ContestID, ProblemID uint; Language string; SubmissionCount uint; URL *string}` (table `judge_contestmoss`).
  - `jobs.MossLanguages` (`[]struct{CommonName, Moss string}` = C/c, C++/cc, Java/java, Python/python, Pascal/pascal), `jobs.MossSubmitter` interface `Submit(lang, comment string, files []moss.File) (string, error)`, `jobs.RunContestMoss(tx *gorm.DB, contestID uint, sub MossSubmitter) (int, error)`, `jobs.MossUserID(tx *gorm.DB) (string, error)`, `jobs.ErrMossNotConfigured`, `jobs.EnqueueContestMoss(contestID uint) error`, `jobs.ErrMossAlreadyRunning`, `jobs.IsContestMossRunning(contestID uint) bool`, `jobs.TypeContestMoss`.
  - `GET /admin/contest/:key/moss` → `{ running: bool, results: [{ problem_code, problem_name, language, submission_count, url }] }` (ordered by contest problem order, then language order); `POST` → 202 `{ queued: true }`, 400 when `moss_user_id` is not configured, 409 when already running; `DELETE` → 200 `{ deleted: n }`. All require `judge.moss_contest` and contest edit rights.

- [ ] **Step 1: Add the model**

Append to `claoj/models/contest.go`:

```go
// ContestMoss mirrors judge_contestmoss (v1 ContestMoss): one MOSS report
// per contest problem × language.
type ContestMoss struct {
	ID              uint    `gorm:"primaryKey;column:id"`
	ContestID       uint    `gorm:"column:contest_id;not null;index"`
	ProblemID       uint    `gorm:"column:problem_id;not null;index"`
	Language        string  `gorm:"column:language;size:10;not null"`
	SubmissionCount uint    `gorm:"column:submission_count;not null;default:0"`
	URL             *string `gorm:"column:url;size:200"`
}

func (ContestMoss) TableName() string { return "judge_contestmoss" }
```

- [ ] **Step 2: Write the failing job tests**

`claoj/jobs/moss_test.go`:

```go
package jobs

import (
	"errors"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/moss"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type call struct {
	lang, comment string
	files         []moss.File
}

type fakeSubmitter struct {
	calls []call
	err   error
}

func (f *fakeSubmitter) Submit(lang, comment string, files []moss.File) (string, error) {
	f.calls = append(f.calls, call{lang, comment, files})
	if f.err != nil {
		return "", f.err
	}
	return "http://moss.stanford.edu/results/1/" + lang, nil
}

func setupMossDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := database.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Language{}, &models.Problem{}, &models.Contest{},
		&models.ContestParticipation{}, &models.Submission{}, &models.SubmissionSource{}, &models.ContestSubmission{},
		&models.ContestMoss{}, &models.MiscConfig{},
	))
	require.NoError(t, database.Migrator().DropTable("judge_contestproblem"))
	require.NoError(t, database.Migrator().CreateTable(&models.ContestProblem{}))
	return database
}

// seed builds contest "tik25" with problem "knap" and C++ submissions:
// alice 100pt + alice 40pt (same user, lower), bob 70pt (live), carol (virtual, excluded).
func seed(t *testing.T, database *gorm.DB) models.Contest {
	t.Helper()
	c := models.Contest{Key: "tik25", Name: "TiK25", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}
	require.NoError(t, database.Omit(clause.Associations).Create(&c).Error)
	p := models.Problem{Code: "knap", Name: "Knapsack"}
	require.NoError(t, database.Omit(clause.Associations).Create(&p).Error)
	cp := models.ContestProblem{ContestID: c.ID, ProblemID: p.ID, Points: 100, Order: 1}
	require.NoError(t, database.Omit(clause.Associations).Create(&cp).Error)
	cpp := models.Language{Key: "CPP17", Name: "C++17", CommonName: "C++"}
	require.NoError(t, database.Create(&cpp).Error)

	add := func(username string, virtual int, points float64, source string) {
		var u models.AuthUser
		database.Where("username = ?", username).First(&u)
		if u.ID == 0 {
			u = models.AuthUser{Username: username, IsActive: true}
			require.NoError(t, database.Create(&u).Error)
		}
		var prof models.Profile
		database.Where("user_id = ?", u.ID).First(&prof)
		if prof.ID == 0 {
			prof = models.Profile{UserID: u.ID, Timezone: "UTC"}
			require.NoError(t, database.Omit(clause.Associations).Create(&prof).Error)
		}
		part := models.ContestParticipation{ContestID: c.ID, UserID: prof.ID, RealStart: time.Now(), Virtual: virtual}
		require.NoError(t, database.Omit(clause.Associations).Create(&part).Error)
		sub := models.Submission{UserID: prof.ID, ProblemID: p.ID, LanguageID: cpp.ID, Status: "D", Points: &points, Date: time.Now(), ContestObjectID: &c.ID}
		require.NoError(t, database.Omit(clause.Associations).Create(&sub).Error)
		require.NoError(t, database.Omit(clause.Associations).Create(&models.SubmissionSource{SubmissionID: sub.ID, Source: source}).Error)
		require.NoError(t, database.Omit(clause.Associations).Create(&models.ContestSubmission{SubmissionID: sub.ID, ProblemID: cp.ID, ParticipationID: part.ID}).Error)
	}
	add("alice", 0, 100, "best alice")
	add("alice", 0, 40, "worse alice")
	add("bob", -1, 70, "bob code")
	add("carol", 1, 100, "virtual carol")
	return c
}

func TestRunContestMoss_MirrorsV1(t *testing.T) {
	database := setupMossDB(t)
	c := seed(t, database)
	sub := &fakeSubmitter{}

	n, err := RunContestMoss(database, c.ID, sub)
	require.NoError(t, err)
	require.Equal(t, len(MossLanguages), n, "one row per problem x language, even when empty")

	require.Len(t, sub.calls, 1, "only C++ has submissions")
	require.Equal(t, "cc", sub.calls[0].lang)
	require.Equal(t, "tik25 - knap", sub.calls[0].comment)
	require.Equal(t, []moss.File{{Name: "alice", Content: []byte("best alice")}, {Name: "bob", Content: []byte("bob code")}}, sub.calls[0].files)

	var rows []models.ContestMoss
	require.NoError(t, database.Where("contest_id = ?", c.ID).Find(&rows).Error)
	require.Len(t, rows, len(MossLanguages))
	for _, r := range rows {
		if r.Language == "C++" {
			require.Equal(t, uint(2), r.SubmissionCount)
			require.Equal(t, "http://moss.stanford.edu/results/1/cc", *r.URL)
		} else {
			require.Zero(t, r.SubmissionCount)
			require.Nil(t, r.URL)
		}
	}
}

func TestRunContestMoss_FailureKeepsOldRows(t *testing.T) {
	database := setupMossDB(t)
	c := seed(t, database)
	old := "http://old/report"
	require.NoError(t, database.Create(&models.ContestMoss{ContestID: c.ID, ProblemID: 1, Language: "C++", SubmissionCount: 5, URL: &old}).Error)

	_, err := RunContestMoss(database, c.ID, &fakeSubmitter{err: errors.New("connection reset")})
	require.Error(t, err)

	var rows []models.ContestMoss
	require.NoError(t, database.Where("contest_id = ?", c.ID).Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, old, *rows[0].URL)
}

func TestMossUserID(t *testing.T) {
	database := setupMossDB(t)
	_, err := MossUserID(database)
	require.ErrorIs(t, err, ErrMossNotConfigured)
	require.NoError(t, database.Create(&models.MiscConfig{Key: "moss_user_id", Value: " 12345 "}).Error)
	id, err := MossUserID(database)
	require.NoError(t, err)
	require.Equal(t, "12345", id)
}
```

- [ ] **Step 3: Run them to verify they fail**

Run: `go test ./jobs/ -run 'Moss' -count=1`
Expected: FAIL to compile — `undefined: RunContestMoss`, `MossLanguages`, `MossUserID`, `ErrMossNotConfigured`.

- [ ] **Step 4: Implement the job**

`claoj/jobs/moss.go`:

```go
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/moss"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

const TypeContestMoss = "contest_moss"

// MossLanguages is v1's ContestMoss.LANG_MAPPING: Language.common_name →
// MOSS language.
var MossLanguages = []struct{ CommonName, Moss string }{
	{"C", "c"},
	{"C++", "cc"},
	{"Java", "java"},
	{"Python", "python"},
	{"Pascal", "pascal"},
}

var (
	ErrMossNotConfigured  = errors.New(`moss: set misc config "moss_user_id" to your MOSS account id`)
	ErrMossAlreadyRunning = errors.New("moss: a run for this contest is already queued or running")
)

// MossSubmitter is the part of moss.Client the job needs (faked in tests).
type MossSubmitter interface {
	Submit(lang, comment string, files []moss.File) (string, error)
}

// MossUserID reads the MOSS account id from judge_miscconfig.
func MossUserID(tx *gorm.DB) (string, error) {
	var cfg models.MiscConfig
	err := tx.Where("`key` = ?", "moss_user_id").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrMossNotConfigured
	}
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(cfg.Value)
	if id == "" {
		return "", ErrMossNotConfigured
	}
	return id, nil
}

// RunContestMoss mirrors v1 judge.tasks.contest.run_moss: for every contest
// problem × MOSS language, send each live/spectating contestant's
// best-scored source and record the report. Rows are replaced only after
// every call succeeded, so a failed run keeps the previous results.
func RunContestMoss(tx *gorm.DB, contestID uint, sub MossSubmitter) (int, error) {
	var contest models.Contest
	if err := tx.First(&contest, contestID).Error; err != nil {
		return 0, err
	}
	var problems []models.Problem
	if err := tx.Table("judge_problem").
		Select("judge_problem.*").
		Joins("JOIN judge_contestproblem cp ON cp.problem_id = judge_problem.id").
		Where("cp.contest_id = ?", contestID).
		Order("cp.`order` ASC").
		Find(&problems).Error; err != nil {
		return 0, err
	}

	results := make([]models.ContestMoss, 0, len(problems)*len(MossLanguages))
	for _, p := range problems {
		for _, lang := range MossLanguages {
			row := models.ContestMoss{ContestID: contestID, ProblemID: p.ID, Language: lang.CommonName}

			var subs []struct {
				Username string
				Source   string
			}
			if err := tx.Table("judge_submission s").
				Select("u.username AS username, src.source AS source").
				Joins("JOIN judge_submissionsource src ON src.submission_id = s.id").
				Joins("JOIN judge_language l ON l.id = s.language_id").
				Joins("JOIN judge_profile pr ON pr.id = s.user_id").
				Joins("JOIN auth_user u ON u.id = pr.user_id").
				Joins("JOIN judge_contestsubmission cs ON cs.submission_id = s.id").
				Joins("JOIN judge_contestparticipation part ON part.id = cs.participation_id").
				Where("s.contest_object_id = ? AND s.problem_id = ? AND l.common_name = ?", contestID, p.ID, lang.CommonName).
				Where("part.virtual IN ?", []int{0, -1}). // LIVE, SPECTATE
				Order("s.points DESC").
				Scan(&subs).Error; err != nil {
				return 0, err
			}

			seen := make(map[string]bool, len(subs))
			files := make([]moss.File, 0, len(subs))
			for _, s := range subs {
				if seen[s.Username] {
					continue
				}
				seen[s.Username] = true
				files = append(files, moss.File{Name: s.Username, Content: []byte(s.Source)})
			}
			if len(files) > 0 {
				url, err := sub.Submit(lang.Moss, fmt.Sprintf("%s - %s", contest.Key, p.Code), files)
				if err != nil {
					return 0, fmt.Errorf("moss %s/%s: %w", p.Code, lang.CommonName, err)
				}
				row.URL = &url
				row.SubmissionCount = uint(len(files))
			}
			results = append(results, row)
		}
	}

	err := tx.Transaction(func(t *gorm.DB) error {
		if err := t.Where("contest_id = ?", contestID).Delete(&models.ContestMoss{}).Error; err != nil {
			return err
		}
		if len(results) == 0 {
			return nil
		}
		return t.Create(&results).Error
	})
	if err != nil {
		return 0, err
	}
	return len(results), nil
}

type contestMossPayload struct {
	ContestID uint `json:"contest_id"`
}

func mossRunningKey(contestID uint) string { return fmt.Sprintf("claoj:moss:running:%d", contestID) }

// IsContestMossRunning reports whether a run is queued or in progress.
func IsContestMossRunning(contestID uint) bool {
	if cache.Client == nil {
		return false
	}
	n, err := cache.Client.Exists(cache.Ctx, mossRunningKey(contestID)).Result()
	return err == nil && n > 0
}

// EnqueueContestMoss queues one run per contest; a second request while one
// is pending returns ErrMossAlreadyRunning.
func EnqueueContestMoss(contestID uint) error {
	payload, err := json.Marshal(contestMossPayload{ContestID: contestID})
	if err != nil {
		return err
	}
	task := asynq.NewTask(TypeContestMoss, payload,
		asynq.MaxRetry(0),
		asynq.Timeout(30*time.Minute),
		asynq.TaskID(fmt.Sprintf("contest_moss:%d", contestID)),
	)
	if _, err := Client.Enqueue(task, asynq.Queue("low")); err != nil {
		if errors.Is(err, asynq.ErrTaskIDConflict) {
			return ErrMossAlreadyRunning
		}
		return err
	}
	if cache.Client != nil {
		cache.Client.Set(cache.Ctx, mossRunningKey(contestID), "1", 30*time.Minute)
	}
	return nil
}

// HandleContestMossTask runs a queued contest MOSS job.
func HandleContestMossTask(ctx context.Context, t *asynq.Task) error {
	var p contestMossPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	defer func() {
		if cache.Client != nil {
			cache.Client.Del(cache.Ctx, mossRunningKey(p.ContestID))
		}
	}()
	userID, err := MossUserID(db.DB)
	if err != nil {
		return err
	}
	n, err := RunContestMoss(db.DB, p.ContestID, &moss.Client{UserID: userID, Show: 200})
	if err != nil {
		log.Printf("jobs: MOSS for contest %d failed: %v", p.ContestID, err)
		return err
	}
	log.Printf("jobs: MOSS for contest %d wrote %d rows", p.ContestID, n)
	return nil
}
```

(`Show: 200` mirrors v1's `matching_file_limit=200`.)

In `claoj/jobs/server.go`, add after the other `mux.HandleFunc` lines:

```go
	mux.HandleFunc(TypeContestMoss, HandleContestMossTask)
```

- [ ] **Step 5: Run the job tests**

Run: `go test ./jobs/ -run 'Moss' -count=1`
Expected: PASS.

- [ ] **Step 6: Write the failing handler tests**

`claoj/api/v2/admin_contest_moss_test.go`:

```go
package v2

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupContestMossDB(t *testing.T) (root, staff models.AuthUser, contest models.Contest) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := database.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{}, &models.AuthPermission{},
		&models.AuthUserGroup{}, &models.AuthGroupPermission{}, &models.AuthUserPermission{},
		&models.Profile{}, &models.Problem{}, &models.Contest{}, &models.ContestMoss{}, &models.MiscConfig{},
	))
	// The list handler LEFT JOINs judge_contestproblem for ordering.
	require.NoError(t, database.Migrator().DropTable("judge_contestproblem"))
	require.NoError(t, database.Migrator().CreateTable(&models.ContestProblem{}))
	db.DB = database
	root = models.AuthUser{Username: "root", IsActive: true, IsStaff: true, IsSuperuser: true}
	staff = models.AuthUser{Username: "staff", IsActive: true, IsStaff: true}
	require.NoError(t, database.Create(&root).Error)
	require.NoError(t, database.Create(&staff).Error)
	contest = models.Contest{Key: "tik25", Name: "TiK25", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}
	require.NoError(t, database.Omit(clause.Associations).Create(&contest).Error)
	return
}

func TestContestMoss_RequiresPermission(t *testing.T) {
	_, staff, _ := setupContestMossDB(t)
	router := authedRouter(staff.ID, http.MethodGet, "/admin/contest/:key/moss", AdminContestMossList)
	w := doRequest(router, http.MethodGet, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestContestMoss_ListReturnsRows(t *testing.T) {
	root, _, c := setupContestMossDB(t)
	p := models.Problem{Code: "knap", Name: "Knapsack"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	url := "http://moss/report"
	require.NoError(t, db.DB.Create(&models.ContestMoss{ContestID: c.ID, ProblemID: p.ID, Language: "C++", SubmissionCount: 4, URL: &url}).Error)

	router := authedRouter(root.ID, http.MethodGet, "/admin/contest/:key/moss", AdminContestMossList)
	w := doRequest(router, http.MethodGet, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Running bool `json:"running"`
		Results []struct {
			ProblemCode     string  `json:"problem_code"`
			Language        string  `json:"language"`
			SubmissionCount uint    `json:"submission_count"`
			URL             *string `json:"url"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.False(t, body.Running)
	require.Len(t, body.Results, 1)
	require.Equal(t, "knap", body.Results[0].ProblemCode)
	require.Equal(t, url, *body.Results[0].URL)
}

func TestContestMoss_RunNeedsConfigAndEnqueues(t *testing.T) {
	root, _, c := setupContestMossDB(t)
	var enqueued []uint
	enqueueContestMoss = func(id uint) error { enqueued = append(enqueued, id); return nil }
	t.Cleanup(func() { enqueueContestMoss = jobs.EnqueueContestMoss })

	router := authedRouter(root.ID, http.MethodPost, "/admin/contest/:key/moss", AdminContestMossRun)
	w := doRequest(router, http.MethodPost, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "moss_user_id")

	require.NoError(t, db.DB.Create(&models.MiscConfig{Key: "moss_user_id", Value: "123"}).Error)
	w = doRequest(router, http.MethodPost, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	require.Equal(t, []uint{c.ID}, enqueued)

	enqueueContestMoss = func(uint) error { return jobs.ErrMossAlreadyRunning }
	w = doRequest(router, http.MethodPost, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestContestMoss_DeleteClearsRows(t *testing.T) {
	root, _, c := setupContestMossDB(t)
	require.NoError(t, db.DB.Create(&models.ContestMoss{ContestID: c.ID, ProblemID: 1, Language: "C"}).Error)
	router := authedRouter(root.ID, http.MethodDelete, "/admin/contest/:key/moss", AdminContestMossDelete)
	w := doRequest(router, http.MethodDelete, "/admin/contest/tik25/moss", "")
	require.Equal(t, http.StatusOK, w.Code)
	var n int64
	db.DB.Model(&models.ContestMoss{}).Count(&n)
	require.Zero(t, n)
}
```

- [ ] **Step 7: Run them to verify they fail**

Run: `go test ./api/v2/ -run 'ContestMoss' -count=1`
Expected: FAIL to compile — undefined handlers and `enqueueContestMoss`.

- [ ] **Step 8: Implement the handlers and routes**

`claoj/api/v2/admin_contest_moss.go`:

```go
package v2

import (
	"errors"
	"net/http"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
)

// enqueueContestMoss is swapped in tests.
var enqueueContestMoss = jobs.EnqueueContestMoss

// contestForMoss loads the contest and enforces v1's judge.moss_contest
// permission; it writes the error response and returns false on failure.
func contestForMoss(c *gin.Context) (models.Contest, bool) {
	var contest models.Contest
	if !auth.HasPerm(c, "judge.moss_contest") {
		c.JSON(http.StatusForbidden, apiError("you are not allowed to run MOSS"))
		return contest, false
	}
	if err := db.DB.Where("`key` = ?", c.Param("key")).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, apiError("contest not found"))
		return contest, false
	}
	return contest, true
}

// AdminContestMossList - GET /api/v2/admin/contest/:key/moss
func AdminContestMossList(c *gin.Context) {
	contest, ok := contestForMoss(c)
	if !ok {
		return
	}
	type row struct {
		ProblemCode     string  `json:"problem_code"`
		ProblemName     string  `json:"problem_name"`
		Language        string  `json:"language"`
		SubmissionCount uint    `json:"submission_count"`
		URL             *string `json:"url"`
	}
	var rows []row
	if err := db.DB.Table("judge_contestmoss m").
		Select("p.code AS problem_code, p.name AS problem_name, m.language, m.submission_count, m.url").
		Joins("JOIN judge_problem p ON p.id = m.problem_id").
		Joins("LEFT JOIN judge_contestproblem cp ON cp.problem_id = m.problem_id AND cp.contest_id = m.contest_id").
		Where("m.contest_id = ?", contest.ID).
		Order("cp.`order` ASC, m.id ASC").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	if rows == nil {
		rows = []row{}
	}
	c.JSON(http.StatusOK, gin.H{"running": jobs.IsContestMossRunning(contest.ID), "results": rows})
}

// AdminContestMossRun - POST /api/v2/admin/contest/:key/moss
func AdminContestMossRun(c *gin.Context) {
	contest, ok := contestForMoss(c)
	if !ok {
		return
	}
	if _, err := jobs.MossUserID(db.DB); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}
	if err := enqueueContestMoss(contest.ID); err != nil {
		if errors.Is(err, jobs.ErrMossAlreadyRunning) {
			c.JSON(http.StatusConflict, apiError(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, apiError(err.Error()))
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"queued": true})
}

// AdminContestMossDelete - DELETE /api/v2/admin/contest/:key/moss
func AdminContestMossDelete(c *gin.Context) {
	contest, ok := contestForMoss(c)
	if !ok {
		return
	}
	res := db.DB.Where("contest_id = ?", contest.ID).Delete(&models.ContestMoss{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, apiError(res.Error.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}
```

In `claoj/api/router.go`, next to the other `/admin/contest/:key/...` routes:

```go
			admin.GET("/admin/contest/:key/moss", auth.RequireContestEdit(), v2.AdminContestMossList)
			admin.POST("/admin/contest/:key/moss", auth.RequireContestEdit(), v2.AdminContestMossRun)
			admin.DELETE("/admin/contest/:key/moss", auth.RequireContestEdit(), v2.AdminContestMossDelete)
```

- [ ] **Step 9: Run tests and build**

Run: `go build ./... && go test ./jobs/ ./api/v2/ ./moss/ -count=1`
Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add claoj/models/contest.go claoj/jobs claoj/api/v2/admin_contest_moss.go claoj/api/v2/admin_contest_moss_test.go claoj/api/router.go
git commit -F - <<'EOF'
feat(admin): contest MOSS runs writing v1's judge_contestmoss results

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 12: MOSS page (frontend)

**Files:**
- Modify: `claoj-web/src/lib/adminApi.ts` (add `adminMossApi`, remove `mossAnalyze` / `mossResults` and `MossAnalysisResult` if unused)
- Replace: `claoj-web/src/app/[locale]/admin/moss/page.tsx`
- Modify: `claoj-web/src/i18n/en.json`, `claoj-web/src/i18n/vi.json` (replace `Admin.moss`)

**Interfaces:**
- Consumes: Task 11 endpoints; `useListParams` (Task 9); `searchContests` (Task 8).
- Produces: `adminMossApi.list(key) → { running: boolean; results: ContestMossResult[] }`, `adminMossApi.run(key)`, `adminMossApi.clear(key)`; `ContestMossResult { problem_code: string; problem_name: string; language: string; submission_count: number; url: string | null }`.

- [ ] **Step 1: API client**

Append to `claoj-web/src/lib/adminApi.ts`:

```ts
// ============================================================
// ADMIN CONTEST MOSS API (v1 ContestMoss parity)
// ============================================================

export interface ContestMossResult {
    problem_code: string;
    problem_name: string;
    language: string;
    submission_count: number;
    url: string | null;
}

export const adminMossApi = {
    list: (key: string) =>
        api.get<{ running: boolean; results: ContestMossResult[] }>(`/admin/contest/${key}/moss`),
    run: (key: string) =>
        api.post<{ queued: boolean }>(`/admin/contest/${key}/moss`),
    clear: (key: string) =>
        api.delete<{ deleted: number }>(`/admin/contest/${key}/moss`),
};
```

Delete `mossAnalyze`, `mossResults` from `adminSubmissionApi` and the `MossAnalysisResult` interface (grep first: the old MOSS page is the only user and is replaced below).

- [ ] **Step 2: Replace the page**

`claoj-web/src/app/[locale]/admin/moss/page.tsx`:

```tsx
'use client';

import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ScanSearch, Play, Trash2, ExternalLink } from 'lucide-react';
import { adminMossApi, type ContestMossResult } from '@/lib/adminApi';
import { searchContests } from '@/lib/adminSearch';
import { useListParams } from '@/hooks/useListParams';
import { useDebounce } from '@/hooks/useDebounce';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';

const KEYS = ['contest'] as const;

type ApiError = { response?: { data?: { error?: string } } };
const errorText = (e: unknown) => (e as ApiError)?.response?.data?.error ?? String(e);

export default function AdminMossPage() {
    const t = useTranslations('Admin');
    const queryClient = useQueryClient();
    const { filters, setFilters } = useListParams(KEYS);
    const contestKey = filters.contest ?? '';
    const [term, setTerm] = useState('');
    const debounced = useDebounce(term.trim(), 250);

    const { data: matches = [] } = useQuery({
        queryKey: ['moss-contest-search', debounced],
        queryFn: () => searchContests(debounced),
        enabled: debounced.length > 0,
    });

    const { data, isLoading, isError, error } = useQuery({
        queryKey: ['admin-contest-moss', contestKey],
        queryFn: async () => (await adminMossApi.list(contestKey)).data,
        enabled: contestKey !== '',
        refetchInterval: q => (q.state.data?.running ? 5000 : false),
    });

    const run = useMutation({
        mutationFn: () => adminMossApi.run(contestKey),
        onSuccess: () => {
            toast.success(t('moss.queued'));
            queryClient.invalidateQueries({ queryKey: ['admin-contest-moss', contestKey] });
        },
        onError: e => toast.error(errorText(e)),
    });

    const clear = useMutation({
        mutationFn: () => adminMossApi.clear(contestKey),
        onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin-contest-moss', contestKey] }),
        onError: e => toast.error(errorText(e)),
    });

    // Group rows by problem so each problem reads as one block.
    const grouped = useMemo(() => {
        const map = new Map<string, { name: string; rows: ContestMossResult[] }>();
        for (const r of data?.results ?? []) {
            if (!map.has(r.problem_code)) map.set(r.problem_code, { name: r.problem_name, rows: [] });
            map.get(r.problem_code)!.rows.push(r);
        }
        return [...map.entries()];
    }, [data]);

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-2xl font-bold flex items-center gap-2.5">
                    <ScanSearch className="text-primary" size={26} />
                    {t('moss.title')}
                </h1>
                <p className="text-sm text-muted-foreground mt-1">{t('moss.subtitle')}</p>
            </div>

            <div className="relative max-w-md">
                <label htmlFor="moss-contest" className="text-xs text-muted-foreground block mb-1">{t('moss.contestLabel')}</label>
                <input
                    id="moss-contest"
                    value={term}
                    onChange={e => setTerm(e.target.value)}
                    placeholder={contestKey || t('moss.contestPlaceholder')}
                    autoComplete="off"
                    className="w-full h-9 px-3 rounded-md border bg-card text-sm focus:outline-none focus:ring-2 focus:ring-ring/40"
                />
                {debounced && matches.length > 0 && (
                    <ul className="absolute z-20 mt-1 w-full rounded-md border bg-popover shadow-lg max-h-60 overflow-y-auto">
                        {matches.map(m => (
                            <li key={m.id}>
                                <button
                                    type="button"
                                    onClick={() => { setFilters({ contest: m.label }); setTerm(''); }}
                                    className="w-full text-left px-3 py-2 text-sm hover:bg-muted flex gap-2 items-baseline"
                                >
                                    <span className="font-medium">{m.label}</span>
                                    <span className="text-xs text-muted-foreground truncate">{m.detail}</span>
                                </button>
                            </li>
                        ))}
                    </ul>
                )}
            </div>

            {!contestKey ? (
                <p className="text-sm text-muted-foreground">{t('moss.pickContest')}</p>
            ) : (
                <section className="rounded-md border bg-card">
                    <header className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 border-b bg-secondary/60">
                        <h2 className="text-sm font-semibold">
                            <Link href={`/contest/${contestKey}`} className="hover:underline">{contestKey}</Link>
                            {data?.running && <span className="ml-2 text-xs font-normal text-warning">{t('moss.running')}</span>}
                        </h2>
                        <div className="flex gap-2">
                            <button
                                onClick={() => run.mutate()}
                                disabled={run.isPending || data?.running}
                                className="inline-flex items-center gap-1.5 h-8 px-3 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-50"
                            >
                                <Play size={14} /> {t('moss.run')}
                            </button>
                            <button
                                onClick={() => { if (confirm(t('moss.clearConfirm'))) clear.mutate(); }}
                                disabled={clear.isPending || !data?.results.length}
                                className="inline-flex items-center gap-1.5 h-8 px-3 rounded-md border text-sm hover:bg-muted disabled:opacity-50"
                            >
                                <Trash2 size={14} /> {t('moss.clear')}
                            </button>
                        </div>
                    </header>

                    {isLoading ? (
                        <div className="p-4 space-y-2">{[1, 2, 3].map(i => <Skeleton key={i} className="h-5 w-full" />)}</div>
                    ) : isError ? (
                        <p className="px-4 py-8 text-center text-sm text-destructive">{errorText(error)}</p>
                    ) : grouped.length === 0 ? (
                        <p className="px-4 py-8 text-center text-sm text-muted-foreground">{t('moss.noResults')}</p>
                    ) : (
                        <table className="w-full text-sm">
                            <thead className="text-left text-xs text-muted-foreground">
                                <tr>
                                    <th className="px-4 py-2 font-semibold">{t('moss.colProblem')}</th>
                                    <th className="px-4 py-2 font-semibold">{t('moss.colLanguage')}</th>
                                    <th className="px-4 py-2 font-semibold text-right">{t('moss.colSubmissions')}</th>
                                    <th className="px-4 py-2 font-semibold">{t('moss.colReport')}</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {grouped.flatMap(([code, g]) => g.rows.map((r, i) => (
                                    <tr key={`${code}-${r.language}`}>
                                        <td className="px-4 py-2">{i === 0 && <><span className="font-medium">{code}</span> <span className="text-muted-foreground">{g.name}</span></>}</td>
                                        <td className="px-4 py-2">{r.language}</td>
                                        <td className="px-4 py-2 text-right tabular-nums">{r.submission_count}</td>
                                        <td className="px-4 py-2">
                                            {r.url ? (
                                                <a href={r.url} target="_blank" rel="noopener noreferrer" className="text-accent hover:underline inline-flex items-center gap-1">
                                                    {t('moss.viewReport')} <ExternalLink size={12} />
                                                </a>
                                            ) : <span className="text-muted-foreground">—</span>}
                                        </td>
                                    </tr>
                                )))}
                            </tbody>
                        </table>
                    )}
                </section>
            )}
        </div>
    );
}
```

- [ ] **Step 3: Replace the `Admin.moss` i18n block in both locales**

Delete every existing key under `Admin.moss` and add exactly:

| key | en | vi |
|---|---|---|
| `title` | `MOSS` | `MOSS` |
| `subtitle` | `Plagiarism reports per contest problem and language, as in v1.` | `Báo cáo đạo văn theo từng bài và ngôn ngữ của kỳ thi, giống v1.` |
| `contestLabel` | `Contest` | `Kỳ thi` |
| `contestPlaceholder` | `Search a contest by key or name` | `Tìm kỳ thi theo mã hoặc tên` |
| `pickContest` | `Pick a contest to see or run its MOSS reports.` | `Chọn một kỳ thi để xem hoặc chạy MOSS.` |
| `run` | `Run MOSS` | `Chạy MOSS` |
| `running` | `Running… results refresh automatically` | `Đang chạy… kết quả tự cập nhật` |
| `queued` | `MOSS run queued` | `Đã đưa MOSS vào hàng đợi` |
| `clear` | `Clear results` | `Xoá kết quả` |
| `clearConfirm` | `Delete every MOSS result for this contest?` | `Xoá toàn bộ kết quả MOSS của kỳ thi này?` |
| `noResults` | `No MOSS results yet. Run MOSS to create them.` | `Chưa có kết quả MOSS. Hãy chạy MOSS để tạo.` |
| `colProblem` | `Problem` | `Bài` |
| `colLanguage` | `Language` | `Ngôn ngữ` |
| `colSubmissions` | `Contestants` | `Thí sinh` |
| `colReport` | `Report` | `Báo cáo` |
| `viewReport` | `Open report` | `Mở báo cáo` |

- [ ] **Step 4: Verify**

```bash
npx tsc --noEmit -p . 2>&1 | grep '^src/' ; npx jest ; npm run -s i18n:check ; npm run -s i18n:usage
npx eslint "src/app/[locale]/admin/moss" src/lib/adminApi.ts
```

Expected: clean. Manual (dev backend + Redis + worker, `moss_user_id` set in Misc Config): pick a contest that v1 already ran MOSS for → its existing reports appear; Run MOSS → button disables, "Running…" shows, table refreshes when done; a second Run while running → toast with the 409 message; without `moss_user_id` → toast naming the key.

- [ ] **Step 5: Commit**

```bash
git add claoj-web/src
git commit -F - <<'EOF'
feat(admin): contest MOSS page with run, clear and live progress

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```

---

### Task 13: Final verification and spec sync

**Files:**
- Modify: `docs/superpowers/specs/2026-10-09-admin-panel-parity-design.md` (record the planning deltas)

- [ ] **Step 1: Full backend suite**

Run: `cd claoj && gofmt -l . | grep -v vendor ; go vet ./... && go test ./... -count=1`
Expected: `gofmt -l` prints nothing for touched files; vet clean; all tests PASS.

- [ ] **Step 2: Full frontend suite**

Run:

```bash
cd ../claoj-web
npx tsc --noEmit -p . 2>&1 | grep -c '^src/'      # expect 0
npx jest
npm run -s i18n:check && npm run -s i18n:usage
npx next build 2>&1 | tail -20
```

Expected: 0 `src/` type errors; jest all PASS; i18n OK; build succeeds.

- [ ] **Step 3: Record the spec deltas**

In the spec, under "A1. Edits stop dropping data", append:

```markdown
Planning found three more breaks this sub-project fixes: the problem and
contest detail endpoints returned nested PascalCase JSON (edit forms and
PdfTab loaded empty); `UpdateContest` ignored start/end times and contest
create rejected every `datetime-local` value; the time limit was sent as a
string in the wrong unit. The admin API now speaks seconds for
`time_limit` (the form shows minutes). Contest problem order is the
position in the `problems` list. Form pickers search the server
(`EntityPicker`) because list pages cap at 100 rows.
```

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/specs/2026-10-09-admin-panel-parity-design.md
git commit -F - <<'EOF'
docs(admin): record sub-project A planning findings in the spec

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format=%s
```
