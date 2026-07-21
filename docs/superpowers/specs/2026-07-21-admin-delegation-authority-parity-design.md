# Admin Delegation-of-Authority Parity (v2 ↔ v1)

**Date:** 2026-07-21
**Status:** Approved (design) — pending implementation plan
**Repo:** `OJ-v2` (backend: `claoj/`)
**Branch:** `feat/admin-delegation-parity` (off `dev`)

## 1. Problem

v2's admin endpoints do not enforce v1 (DMOJ)'s delegation-of-authority model.
Every `/api/admin/*` route stops at a route-level `is_staff` gate
(`AdminRequiredMiddleware`, `claoj/auth/middleware.go:132`); individual handlers
then apply an inconsistent second check, or none. The DMOJ-faithful per-object
checks already exist in `claoj/auth/access.go` (`CanEditProblem`,
`CanEditContest`, `CanRejudge`, …) but are wired into the **public** handlers,
not the **admin** ones.

### Verified live (2026-07-21, stack at `127.0.0.1:8090`)

Two opposite parity gaps reproduced end-to-end with throwaway accounts, then
confirmed non-destructive (target problem `01_02` unchanged):

| Gap | Account | Request | v2 today | v1 correct |
|---|---|---|---|---|
| A (under-restricts) | staff, not superuser, authors nothing, no `edit_all_problem` | `PATCH /api/admin/problem/01_02` (owned by another user) | **200 — allowed** | `403` |
| B (over-restricts) | superuser, `is_staff=0` | `GET /api/admin/problems` | **403 "admin access required"** | `200` (superuser bypasses all) |
| control | staff + superuser | `PATCH /api/admin/problem/01_02` | 200 | 200 |

Real accounts already mis-authorized on the shared DB: `nguyenkimngan` (794),
`btdat2506` (14) are staff without `edit_all_problem` → can currently edit/delete
**any** problem; `KhanhHoa_92` (30) is superuser-but-not-staff → wrongly locked
out of the admin API.

## 2. v1 model (target)

Authority in DMOJ is a two-layer AND:

1. a **capability permission floor** — `edit_own_problem` / `edit_own_contest`;
2. then either a **broad grant** (`edit_all_problem`, `edit_all_contest`,
   `edit_all_organization`) **or** **per-object membership** (problem
   authors/curators/testers, contest organizers/curators, organization admins).

`is_superuser` bypasses every check (Django `ModelBackend`). `is_staff` in v1
only gates the Django **admin site** — it is *not* what grants edit rights to a
specific problem/contest/org.

Reference implementations in v1: `Problem.is_editable_by` (`OJ/judge/models/problem.py:229`),
`Contest.is_editable_by` (`OJ/judge/models/contest.py:380`),
`Organization.is_admin` (`OJ/judge/models/profile.py:81`),
problem-data gate `is_superuser or problem.is_editable_by(user)`
(`OJ/judge/views/problem_data.py:131`).

## 3. Decisions (locked with stakeholder)

- **Model A — staff-gated surface + delegation enforced inside it.** The admin
  surface stays `is_staff`-gated (v2's whole admin UI is hidden unless
  `is_staff`: `claoj-web/.../AdminAccessComponents.tsx:117,292,556`). Within it,
  each delegated action additionally enforces v1's per-object rule. Superusers
  bypass the staff gate (fixes Gap B). *Not* Model B (v2 keeps a single
  staff-gated editing surface; non-staff authors continue to use v1 during
  migration — out of scope, YAGNI).
- **No permissions migration.** Ship the tightening as-is. Staff who lack
  `edit_all_*` and don't own an object will start receiving `403`s and must be
  granted the permission or added as author/curator. See §8.
- **Scope: delegated resources only** — Problems, Contests, Organizations, and
  the org-admin appointment path, plus the superuser-bypass gate fix. Pure
  site-config endpoints (languages, licenses, taxonomy, navigation bars,
  misc-configs, groups, contest-tag *catalog* CRUD) keep their current guards.

## 4. Architecture

Two layers, both required:

- **Route gate** (`AdminRequiredMiddleware`, `claoj/auth/middleware.go:132`):
  change from `is_staff` to **`is_staff OR is_superuser`**. Load both columns;
  allow if either is true. This is the only change that fixes Gap B. Everything
  else stays staff-gated exactly as today. (The `admin_user_id` set on context
  is retained.)
- **Per-handler authorization**: each delegated write **loads its object with
  the preloads the checker needs**, then calls the matching `auth.CanEdit*`
  helper before invoking the service. Deny → `403 {"error": "..."}`. This
  mirrors the one place v2 already does it right —
  `AdminSubmissionRescore` → `auth.CanRejudge(c, &sub.Problem)`
  (`claoj/api/v2/admin_submission.go:215`).

The admin handlers are thin (they hand a code/key straight to a service and do
not load the object), so the wiring is: **insert a `load-with-preloads →
authorize → 403-or-continue` block at the top of each targeted handler.** A
small shared helper per resource is acceptable (e.g. `loadProblemForEdit(code)`
returning the problem with `Authors`/`Curators` preloaded, or `(problem, ok)`).

No frontend changes. No new middleware.

## 5. Authorization matrix

Superuser bypass is already built into every `CanEdit*`/`HasPerm` helper.

| Endpoint(s) | New guard | v1 basis |
|---|---|---|
| `PATCH /admin/problem/:code`, `DELETE /admin/problem/:code` | load `Preload(Authors,Curators)` → `CanEditProblem` | `Problem.is_editable_by` |
| `POST /admin/problems` (create) | `HasPerm("judge.add_problem")` | `ProblemCreate` |
| `POST /admin/problem/:code/clone` | `HasPerm("judge.clone_problem")` | `ProblemClone` |
| All `/admin/problem/:code/data`, `/pdf`, `/data/testcase/...`, `/data/file/...`, `/data/reorder`, `/data/files` (the `api/v2/admin/` subpackage — incl. GET reads) | load `Preload(Authors,Curators)` → `CanEditProblem` | `problem_data.py:131` = `is_superuser or is_editable_by` |
| `POST /admin/problem/:code/clarification`, `DELETE /admin/problem/clarification/:id` | load → `CanEditProblem` *(judgment call §7)* | problem-scoped edit |
| `PATCH /admin/contest/:key`, `DELETE /admin/contest/:key` | load `Preload(Authors,Curators)` → `CanEditContest` | `Contest.is_editable_by` |
| `POST /admin/contest/:key/lock`; `.../participation/:id/disqualify` & `/undisqualify`; `.../tags/:tagId` add/remove | load → `CanEditContest` | contest-edit authority |
| `POST /admin/contests` (create) | `HasPerm("judge.add_contest")` | `ContestCreate` |
| `POST /admin/contest/:key/clone` | `HasPerm("judge.clone_contest")` | `ContestClone` |
| `PATCH /admin/organization/:id` | `CanEditOrganization(orgID)` *(new helper §6)* | org-edit mixins |
| `add_organization_admin` / `remove_organization_admin` in `PATCH /admin/user/:id` | for each org id → `CanEditOrganization(orgID)` | appointing admins = org-edit authority |
| `DELETE /admin/organization/:id` | `HasPerm("judge.edit_all_organization")` or superuser (not mere org-admin) *(judgment call §7)* | deletion is a heavier, admin-level act |
| `POST /admin/organizations` (create) | `HasPerm("judge.add_organization")` | org add permission |
| rejudge / rescore, blog, comments, tickets, MOSS | **no change** (already correct) | already matches |
| languages, licenses, taxonomy, nav-bars, misc-configs, groups, contest-tag *catalog* CRUD | **no change** (out of scope — site-config) | — |

## 6. New helper

Add to `claoj/auth/access.go`, mirroring v1 org-edit authority and the existing
membership-query pattern (`contestHasMember` in the same package). The
`judge_organization_admins` M2M is modelled at
`claoj/models/profile.go:25`.

```go
// CanEditOrganization mirrors OJ organization edit authority:
// superuser, OR judge.edit_all_organization, OR judge.organization_admin, OR being an org admin.
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

// userIsOrgAdmin: EXISTS on judge_organization_admins (organization_id, profile_id).
```

Do **not** import package `v2` from `auth` (the existing `isOrgAdmin` lives in
`api/v2/organization.go` and would create an import cycle) — implement the
membership query directly in `auth`, like `contestHasMember`.

## 7. Judgment calls (approved)

1. **Org delete** is gated harder than org edit: `edit_all_organization` or
   superuser, **not** mere org-admin — an org admin may edit their org but not
   delete it.
2. **Problem clarification** create/delete is treated as a problem-edit action
   (`CanEditProblem`).

## 8. Rollout consequence (no migration)

The change is a tightening. After deploy, staff without `edit_all_*` who don't
own an object receive `403`. To restore broad access intentionally, an operator
grants the relevant permission (via a group) — documented for operators, **run
by them, not by this change**:

```sql
-- Example only; operators run when ready. Grants edit_all_problem to a group.
-- INSERT INTO auth_group_permissions (group_id, permission_id)
-- SELECT <group_id>, id FROM auth_permission WHERE codename='edit_all_problem';
```

After any group/permission change, v2's permission cache must be invalidated
(`auth.BumpPermVersion()`); Django-side edits should bump the same
`perm:version` key or wait out the 60s TTL.

## 9. Testing

- **Unit** (`claoj/auth/access_test.go` style): `CanEditOrganization` truth table
  — superuser / `edit_all_organization` / `organization_admin` / org-admin
  member / outsider.
- **Integration** (follow `claoj/integration/organization_flow_test.go`): per
  resource, assert `staff-non-owner → 403`, `author|curator|org-admin → 200`,
  `superuser → 200`, and `superuser-not-staff` passes the route gate.
- **Live re-verify** at `:8090` with the throwaway accounts
  (`deleg_super` 5001, `deleg_staff` 5002, `deleg_superonly` 5003; created via
  Django `manage.py` in `claoj_site`): `deleg_staff` PATCH problem → now `403`;
  add `deleg_staff` as an author of that problem → `200`; `deleg_superonly`
  GET/PATCH → now `200`; `deleg_super` unaffected. **Delete the three test
  accounts afterward.**

## 10. Out of scope

- Model B (opening the admin surface to non-staff authors/org-admins) and any
  frontend changes.
- Normalizing site-config endpoint guards (languages, licenses, taxonomy,
  navigation bars, misc-configs, groups, contest-tag catalog) to their exact v1
  permissions.
- The `require_totp_for_admins` / TOTP login flow.
- Any change to the shared DB schema (owned by Django; v2 does rows-only).

## 11. Affected files (anticipated)

- `claoj/auth/middleware.go` — route gate `is_staff OR is_superuser`.
- `claoj/auth/access.go` (+ `access_test.go`) — new `CanEditOrganization` +
  `userIsOrgAdmin`.
- `claoj/api/v2/admin_problem.go`, `claoj/api/v2/admin.go` (clone) — problem
  guards.
- `claoj/api/v2/admin/*.go` — problem-data guards.
- `claoj/api/v2/admin_contest.go`, `claoj/api/v2/admin_contest_tag.go` — contest
  guards.
- `claoj/api/v2/admin_judge_organization.go` — org create/update/delete guards.
- `claoj/api/v2/admin_users_roles.go` — org-admin appointment guard.
- `claoj/api/v2/problem_clarification.go` (or wherever the admin clarification
  handlers live) — clarification guard.
- `claoj/integration/*` — new authorization tests.
