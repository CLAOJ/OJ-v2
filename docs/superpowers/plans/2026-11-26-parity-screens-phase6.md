# Phase 6 — Close the remaining v1↔v2 screen gaps

**Goal:** v2 must have a screen for every v1 screen. Ops/AJAX helpers are not screens and are documented as replaced.
**Repo:** worktree `OJ-v2-wt/feat/parity-screens-phase6` (branch `feat/parity-screens-phase6`, from main 3c37824).
**Constraints:** v2 design; no new deps; no migrations (map existing columns only); i18n en+vi; tests for new behavior; no push.

## Accepted gaps (not screens)
- `django.contrib.sites` admin — v2 is single-site by design (`CMSSiteID = 1`).
- Blog revision history/revert — v1 uses django-reversion; v2 substitutes the audit log (no field diff). Documented.
- `widgets/*`, `tasks/*`, `mailgun/*`, `newsletter/` — AJAX/ops helpers; v2 uses JSON endpoints, websocket/toasts, and its own pickers. `newsletter` is dead in v1.
- `/status/`, `/status/oj/` — replaced by v2 `/stats` (judges online/load, queue, charts).
- v1's Django Profile admin (`/admin/judge/profile/`) — replaced by `/admin/users` plus the tabbed user editor (the same profile, roles and security fields; v1-only admin columns stay read-only).
- v1's SubmissionTestCase editor (`/admin/judge/submissiontestcase/`) — replaced by the read-only testcase panel on the admin submission page plus the rejudge action; judge-owned testcase rows are not hand-edited.

## Task 1 — Problem surfaces
1. `?best=1` on the submissions list: one best submission per user (max points, then min time), `points > 0`, excludes unlisted profiles; honour it in the submissions page (the problem page already links `?best=1`).
2. Setter diff matrix at `/admin/problems/[code]/diff`: rows = submissions, columns = testcases, per-case status/points, select two submissions → link to the existing diff viewer. Permission: problem editors (same gate as the data page).
3. `?raw=1` bare statement mode on `/problem/[code]/print` (no chrome) for the v1 `/problem/<code>/raw` use case.
4. Tests: best-mode dedupe/order/visibility; diff page permission; raw mode renders.

## Task 2 — Contest submission surfaces
1. `?contest=<key>` filter on the public submissions list (access check via the existing contest visibility helper; non-participants only see what the scoreboard rules allow).
2. Contest page tab/link to the contest's submissions (respecting the same visibility rules), optionally scoped by user/problem.
3. Per-problem contest ranking at `/contest/[key]/rank/[problem]`: best submission per user for that problem ranked by contest points, plus a link from the scoreboard problem header.
4. Contest leave (`POST /contest/:key/leave`, v1 `/contest/<key>/leave`) + button, and abort of your own in-flight submission (`/submission/<id>/abort` for the owner) on the submission page.
5. Tests: contest filter access (participant/non-participant/hidden scoreboard); per-problem ranking order and exclusions; leave and abort permissions.

## Task 3 — Proposals, license, account email
1. Problem suggestion form (`/problems/suggest/new`): reduced create form gated by `judge.suggest_new_problem`, forces `suggester = caller`, `is_public = false`, no authors; returns to the suggest queue. Reuse the admin problem form sections where practical.
2. Public license page `/license/[key]` rendering the markdown text + link; add license id/name to the problem detail payload and link from the problem page.
3. Self-service email change: request (new email, notify both addresses, signed one-time token, expiry, uniqueness re-check) + confirm endpoint/page; settings entry. Tests: token expiry/reuse, duplicate email, notification hooks.
4. i18n.

## Task 4 — Organization self-service
1. Non-staff org-scoped create endpoints for problems, contests, and blog posts, gated by `judge.create_organization_problem`, `judge.create_private_contest`, `judge.edit_organization_post` respectively (or org-admin membership), forcing the organization onto the object.
2. Org pages: create-problem / create-contest / create-blog forms (reuse admin forms) linked from the org page for permitted members.
3. Join-with-reason request form for open/closed orgs (the API exists; no UI).
4. Request status tabs (pending/approved/rejected/log) on the manage page via `?state=` + a request detail modal; fix the API to return the requested state (currently hardcodes pending).
5. Tests for each gate, forced-org enforcement, request state filtering.

## Task 5 — Admin leftovers + diagnostics
1. `/runtimes` and `/runtimes/matrix`: public runtime list (online-only for non-staff) and the version matrix from the mapped `RuntimeVersion` data.
2. `ProblemPointsVote`: map the existing table, read-only admin list with search/filter (no add).
3. `OrganizationRequest` admin list (read-only; fold into a superuser view if cheaper).
4. `Class`: map the remaining columns (description/access_code/admins/members) and add CRUD with v1 scoping.
5. `CommentLock`: CRUD + enforce the lock on comment creation (gate `judge.override_comment_lock`).
6. Tests for gates, mapping, and lock enforcement.

## Task 7 — Author-facing content screens (v1 pages that were API-only in v2)
1. `/post/new` and `/post/[id]/edit`: public blog authoring. v1 gate: authenticated and (`problem_count >= CLAOJ_BLOG_MIN_PROBLEM_COUNT` or superuser or org admin). v2 only has the staff page `/admin/blog-posts` + admin endpoints; add non-admin create/update endpoints with the v1 gate (reusing the admin handler logic) and pages reusing the existing form/markdown editor. Keep the staff pages working.
2. `/organizations/create`: public organization creation, perm `judge.add_organization`, with the v1 admin-of limit check. v2 only has the staff modal.
3. Staff clone actions for problems and contests (v1 had clone buttons; v2 has APIs only) — buttons on the admin problem/contest pages.
4. Tests for the blog gate (below threshold, at threshold, superuser), org-create perm/limit, and clone buttons' wiring.

## Task 6 — Leftover public screens
1. `/contributors`: v1 lists unlisted-excluded profiles sorted by contribution points (rank, user, organizations, rating, contribution points; 100/page). v2 `/users` API already sorts (`?sort=contribution`) and returns those fields but the page exposes no sort. Add the contributors view (reuse the users list with a contribution sort selector/column) and keep the v1 URL working.
2. `/user/[username]/blog`: v1 pages that user's posts. v2 only has the profile Blogs tab; add the route rendering the same `GET /blogs?author=` list with pagination (link from the profile).
3. Document `/admin/users` as the v1 Profile admin screen substitute and the read-only testcase panel + rejudge as the v1 SubmissionTestCase editor substitute.

## Acceptance
- Every v1 page-level screen has a v2 equivalent or a documented accepted gap.
- `go test ./...`, jest, `tsc`, `npm run build` green; no migrations; live verification after deploy.