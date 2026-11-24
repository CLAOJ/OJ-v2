# Phase 3 — Admin list UX (problems & contests)

**Spec:** `docs/admin-ux-gap-report.md` §1.3, §2.3, Roadmap Phase 3.
**Repo:** worktree `OJ-v2-wt/feat/admin-ux-phase3` (branch `feat/admin-ux-phase3`, from main b4d3b6f).
**Stack:** Next.js 16 web + Go/Gin API, no migrations.

**Scope ruling (controller):** this phase covers the two most-used admin lists (problems, contests). Bulk destructive actions (multi-delete/rejudge) are explicitly deferred: they need their own design/confirmation flow and are the riskiest item in the roadmap. Everything else from the roadmap item is in scope: server-side sorting, creator/year filters (problems), author display, denser rows, and consistent totals (already returned by both APIs).

## Global constraints
- Keep v2 design; minimal diffs; no new dependencies; i18n en+vi; tests for new behavior; commit per task; no push/deploy.

## Task 1 — Backend: sorting/filtering/list fields (Go)
1. Problems list (`GET /api/admin/problems`, `service/problem.ListProblems`):
   - `sort` param restricted to code, name, points, date, ac_rate, user_count (+ existing default date); `order` = asc|desc. Whitelist → column map, never interpolate input.
   - `creator` filter: match problems whose authors include a profile whose user username equals the value (join `judge_problem_authors` + `auth_user`), mirroring v1's "By creator".
   - `year` filter: problems whose `date` falls in that calendar year (v1's year links). Semantics: date NULL excluded when year is set.
   - List items gain `authors: [{id, username}]` (preload/joins; avoid N+1).
   - Tests: each sort direction, creator filter (match + non-match), year filter, authors present in items.
2. Contests list (`GET /api/admin/contests`): add `sort` (name, start_time, user_count, end_time) + `order` whitelist; keep search/status. Tests.
3. No migrations, no response-breaking changes (additive fields only).

## Task 2 — Frontend: sortable tables, filters, density (web)
1. Problems admin page:
   - Sortable column headers (code/name/points/date/AC rate/users) with asc/desc indicators; clicking toggles order, uses Task 1 params.
   - Filters: creator text input (debounced) + year select (derive years from current year down to 2021, or a static range matching v1), in addition to search/visibility/group.
   - Authors column (username list, truncate with title).
   - Denser rows (compact padding/typography) so ~12–15 rows fit; keep actions.
2. Contests admin page: sortable headers (name/start/users), denser rows; keep search/status.
3. Pure helpers (sort-param toggling, year list) unit-tested; i18n en+vi.
4. Keep pagination totals and per-page selector behaviour intact.

## Acceptance
- Clicking a header sorts server-side and survives pagination; filters combine; authors visible; rows noticeably denser; both lists still paginate with correct totals.
- Suites green; lint baseline unchanged.