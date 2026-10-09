# Admin panel: fix broken wiring, expose existing API, daily-use v1 parity

Date: 2026-10-09
Status: draft — awaiting review

## Goal

Staff can do their day-to-day admin work in the v2 admin panel without
opening v1's Django admin. "Day-to-day" is defined by the scope below; rarely
used v1 features are explicitly deferred.

## Constraints

- **v1 and v2 are both live on one MySQL database**, which Django owns. v2
  writes only existing tables/columns and writes them the way Django does
  (same join tables, same column semantics). No schema changes.
- **Recalculations reuse the Go ports of v1's formulas**
  (`scoring.CalculateProfilePoints`, `scoring.CalculateOrganizationPoints`,
  `contribution.UpdateProfileContributionPoints`,
  `contest_format.*.UpdateParticipation`) so both sites show the same numbers.
- Every admin write already passes the audit middleware; nothing new is needed
  for logging.
- Known, accepted limitations of running both sites:
  - v2 edits do not create django-reversion revisions; v1's "History" tab
    will not show them (the v2 audit log does).
  - v1's cached pages may serve old data until their cache entry expires.

## Audit summary (2026-10-09)

Broken (UI present, backend ignores it):

| Page | Defect |
|---|---|
| Problem edit | Sends `author_ids`, `type_ids`, `allowed_lang_ids`, `group_id`; `AdminProblemUpdate` only reads `add_*/remove_*` and has no group field. Edits are silently lost. |
| Contest edit | Sends staff ids, `tag_ids`, `problem_ids`, `format_name`, `format_config`, `time_limit`; `AdminContestUpdate` drops all of them. |
| Submissions | Sends `user/problem/result/language`; `AdminSubmissionList` reads no query params. |
| Problems, contests, organizations | Send `search=`; handlers ignore it. |
| MOSS | Calls `GET /admin/moss/results`, which does not exist. The Go MOSS client (`moss/moss.go`) is a stub that never contacts the MOSS server. |

API supported but not exposed: user edit fields beyond `is_active` /
`ban_reason` (email, display name, about, muted, unlisted, display rank,
organization membership and admin, `is_staff`, `is_superuser`).

Not in scope by design: editing a judge's problems/runtimes. v1 treats them
as read-only because the judge reports them on connect; manual edits would be
overwritten.

## Pattern: full-replacement lists (applies to every admin edit form)

Update handlers accept full lists for many-to-many relations:

- Field present (including `[]`) → replace the association with exactly that
  set via `Association(...).Replace()` on the existing Django join table.
- Field absent → association unchanged.
- Existing `add_*/remove_*` fields keep working; if both forms are sent for one
  relation, the full list wins.
- Each update runs in a single transaction; any failure rolls back the whole
  save and returns an error the form shows.

Precedent: `AdminJudgeUpdate` already uses this for `problem_ids` /
`runtime_ids`.

Exception, by necessity: per-object permission checks that must run per item
(organization admin changes on a user) stay as per-row add/remove actions.

---

## Sub-project A — fix broken wiring

### A1. Edits stop dropping data

`AdminProblemUpdate` accepts: `author_ids`, `curator_ids`, `tester_ids`,
`type_ids`, `allowed_lang_ids` (full lists), `group_id`.

`AdminContestUpdate` accepts: `author_ids`, `curator_ids`, `tester_ids`,
`tag_ids` (full lists), `format_name`, `format_config` (validated by
`contest_format.GetFormat`; invalid config → 400), `time_limit`, and

```json
"problems": [
  { "problem_id": 12, "points": 100, "partial": true, "is_pretested": false, "order": 1 }
]
```

which replaces the contest's `judge_contestproblem` rows: existing rows for
listed problems are updated in place (preserving their `id`, which
participations' submissions reference), unlisted rows are deleted, new
problems are inserted. Removing a problem that already has contest
submissions returns 409 with the problem code instead of deleting.

Frontend: the edit pages already send full state; adapt only the contest
problem list to the `problems` shape. The edit pages stay on the page after
save and show a success toast, instead of redirecting, so staff can confirm
what was saved.

### A2. Server-side search and filters

| Endpoint | New query params |
|---|---|
| `GET /admin/problems` | `search` (code, name), `is_public`, `group_id` |
| `GET /admin/contests` | `search` (key, name), `status` = `upcoming` \| `ongoing` \| `past` |
| `GET /admin/organizations` | `search` (name, short name, slug), pagination |
| `GET /admin/submissions` | `user` (username), `problem` (code), `result`, `status`, `language` (key) |

All keep returning `{ data, total }`. Search is `LIKE %term%` with the term
escaped. The four pages use the shared `PaginationBar` and reset to page 1 when
a filter changes; filter state lives in the URL query so links are shareable.

### A3. MOSS, matching v1's contest MOSS

- Implement the MOSS server protocol client in `moss/` (TCP to
  `moss.stanford.edu:7690`, same exchange as `mosspy`), replacing the stub.
  MOSS user id comes from misc config key `moss_user_id`; missing key → 400
  with a message naming the key.
- `POST /admin/contest/:key/moss` enqueues a background job that, like v1's
  `ContestMoss` flow, groups the contest's submissions by problem × MOSS
  language, sends each group, and writes one `judge_contestmoss` row per group
  (`contest`, `problem`, `language`, `submission_count`, `url`).
- `GET /admin/contest/:key/moss` lists those rows (including any v1 created).
- `DELETE /admin/contest/:key/moss` deletes them (v1's `contest_moss_delete`).
- Permission: `judge.moss_contest`, as today.
- Remove the per-submission `POST/GET /admin/submission/:id/moss` stub routes
  and the frontend calls to them.
- MOSS page: contest picker → table of problem, language, submission count,
  result link; "Run MOSS" (disabled while a job is running, polls for
  completion) and "Clear results" (confirm).

### A tests

- Go handler tests: replace set; omitted field unchanged; `[]` clears; full list
  beats add/remove; contest problem update preserves row ids; 409 on removing a
  problem with submissions; invalid `format_config` → 400; each search/filter
  param; search term escaping.
- MOSS client test against a fake TCP server; job test that writes the expected
  `judge_contestmoss` rows.

---

## Sub-project B — expose existing API: user detail page

New page `/admin/users/[id]`; the users list's edit action links to it and the
modal is removed.

Sections:

- **Profile:** email, display name, about, display rank, unlisted, muted. One
  Save; the request contains only changed fields.
- **Access:** active toggle; ban (reason + days) and unban via existing
  endpoints.
- **Organizations:** one row per membership with Member and Admin toggles that
  apply immediately (`add/remove_organization_ids`,
  `add/remove_organization_admin`); "Add to organization" picker. A 403 for one
  organization shows inline on that row.
- **Roles:** `is_staff`, `is_superuser` switches rendered only when the viewer
  is a superuser (backend already enforces); others see read-only badges.
  Group membership via existing `/admin/user/:id/groups` endpoints.
- **Danger zone:** delete user, confirmation requires typing the username.

Backend: `AdminUserDetail` returns every field the page edits (display name,
about, organizations with an `is_admin` flag, groups, `is_staff`,
`is_superuser`, `is_muted`, `is_unlisted`, `notes` for C3). Additive,
read-only.

### B tests

- Handler test: detail response contains the fields above.
- Frontend test: role switches hidden for a non-superuser viewer.
- Error path: a 403/400 from any section renders an inline error; nothing is
  reported as saved.

---

## Sub-project C — daily-use v1 parity

### C1. Problem

Edit form adds: curators, testers (full lists), `is_organization_private` +
organizations, license, `short_circuit`, publish `date`,
`submission_source_visibility_mode`, `testcase_visibility_mode`, banned users,
`summary`.

Problem list: row checkboxes and bulk **Make public**, **Make private**,
**Set publish date to now** (`POST /admin/problems/bulk` with
`{ codes, action }`), mirroring v1's `make_public`, `make_private`,
`update_publish_date`. Like v1's `_rescore`, the public/private actions then
enqueue a points recalculation for every user who has an accepted submission
on each changed problem (visibility affects points).

### C2. Contest

Edit form adds: `rate_all`, `rating_floor`, `rating_ceiling`, rate-excluded
users; `scoreboard_visibility`, `use_clarifications`, `hide_problem_authors`,
`points_precision`; `is_private` + private contestants; organizations; banned
users.

The Go `Contest` model does not yet map `rate_exclude`; add the many-to-many
association on the existing `judge_contest_rate_exclude` join table (model
change only, no schema change).

Problem table editor: per row points, partial, pretests; reorder by drag and
by up/down buttons (keyboard accessible); writes A1's `problems` list.

Participations page: **Recalculate results** → `POST
/admin/contest/:key/recalculate` enqueues a job running the contest format's
`UpdateParticipation` for every participation (v1 `recompute_results`); the
page shows progress and refreshes when done.

Contest list: row checkboxes and bulk **Show** / **Hide**
(`POST /admin/contests/bulk`, v1 `make_visible` / `make_hidden`).

### C3. User (extends B)

- Staff `notes` field in Profile.
- **Recalculate points** → `POST /admin/user/:id/recalculate`: profile points
  and contribution points.
- **Reset 2FA** → `POST /admin/user/:id/reset-2fa`: sets `is_totp_enabled`
  false, clears `totp_key`, `scratch_codes`, deletes the user's WebAuthn
  credentials, sets `is_webauthn_enabled` false. Superuser only; confirmation
  dialog.

### C4. Organization

Edit modal adds `slots` and admins (user picker, full list). **Recalculate
points** → `POST /admin/organization/:id/recalculate`.

### C tests

- Handler tests per new field and per action (permissions, 404, bulk with an
  unknown code).
- Recalculation parity: on a fixture copied from production, v2's recalculated
  profile points, organization points and one contest's results equal v1's.
- Reset 2FA: non-superuser → 403; afterwards login no longer asks for TOTP.

---

## Deferred (not in this round)

Problem translations, flat pages, comment locks, contest announcements, OG and
logo images, manual submission result edits, problem label scripts, judge
disconnect/terminate.

Judge disconnect/terminate is deferred because
`2026-10-09-judge-pull-queue-design.md` removes the bridge connection registry
it would use; under either design, the existing Block and Disable actions take
a judge out of service.

## Delivery order

A → B → C, each its own implementation plan. A first because it loses data
today. Within C: C3 (builds on B) and C2 before C1 and C4.
