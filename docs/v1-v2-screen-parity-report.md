# CLAOJ v1 ↔ v2 screen parity — results and remaining gaps

**Date:** 2026-11-25 · **v1:** https://claoj.edu.vn · **v2:** https://beta.claoj.edu.vn
**Scope:** every page-level screen in v1 (public + admin) matched against v2, then the gap roadmap executed to completion.

## Method

1. **Inventory** — every route in v1 (`dmoj/urls.py`, `judge/admin/*.py`, templates) matched to a v2 page (`claoj-web/src/app/[locale]/**/page.tsx`) and API endpoint (`claoj/api/router.go`). 173 page-level screens: 122 public, 51 admin. A final exhaustive audit expanded this to **214 route rows** (including account subroutes, admin action URLs, widgets and feeds): 55 parity, 76 substitute, 79 not-a-screen, 4 gap. AJAX widgets, ops endpoints and feeds were classified separately (not screens).
2. **Execution** — gaps were fixed in eight roadmap phases (0, 1, 2, 3, 4a, 4b, 4c, 5) and a closing Phase 6 (7 tasks). Each task was implemented by a subagent, reviewed, and fixed; every phase ended with a cross-task review, full test suites, a CI deploy and live verification on beta.
3. **Verification** — `go test ./...`, jest, `tsc --noEmit`, `npm run build`, i18n checks, eslint baseline; then browser checks against production after each rollout. The full route-by-route table is in `docs/v1-v2-route-audit.md`.

## Result

| Metric | Before roadmap | After Phase 5 | After Phase 6 |
|---|---|---|---|
| v1 page-level screens with a v2 equivalent | 85 | 100 | **173** (all, or documented substitute) |
| Missing screens | 42 | 28 | **0** |
| v2 pages (`page.tsx` routes) | 65 | 72 | **93** |

Phases: **0** P0 bugs · **1** problem form · **2** contest form · **3** list UX · **4a** users/groups · **4b** orgs/blog/tickets/taxonomy/comments/licenses · **4c** submissions/impersonation/registrations/judges/languages/navbar/flat pages · **5** public pages · **6** the remaining 28.

## Phase 6 — what was closed

**Problem surfaces**
- `?best=1` on `/submissions`: one row per user (highest points, then fastest), zero-point and unlisted-profile rows dropped, total paginated correctly. This fixes the dead `&best=1` link that the problem page already had.
- `/admin/problems/[code]/diff`: setter diff matrix — the problem's latest 200 submissions × test cases with per-case status/points, username filter, pick two → the existing diff viewer. Behind `RequireProblemEdit`.
- `/problem/[code]/print?raw=1`: bare statement (title, limits, statement) replacing v1's `/problem/<code>/raw`.

**Contest submission surfaces**
- `?contest=<key>` on `/submissions` with a contest tab, honouring scoreboard visibility (non-participants see peers only when the scoreboard is visible), compatible with `user`/`problem`/`best` filters — covers v1's `/contest/<key>/submissions/<user>[/<problem>]`.
- `/contest/[key]/rank/[problem]`: best submission per user for one problem, ranked on contest points (v1 `ContestRankedSubmission`), linked from the scoreboard.
- Contest leave (`POST /contest/:key/leave` + button; detaches `current_contest_id` like v1, keeping rating/participation history) and owner abort on `/submission/[id]`.

**Proposals, license, account**
- `/problems/suggest/new`: reduced suggestion form gated by `judge.suggest_new_problem`, forcing suggester = caller, `is_public = false`, no authors.
- `/license/[key]` public page (markdown + source link); problem detail now carries its license and links it.
- Self-service email change (settings → account): password check, 10/hour throttle, single-use 1-hour token in the existing token store (no new table), mails the new address and notifies the old one, re-checks uniqueness on confirm.

**Organization self-service**
- Org-scoped creation of problems, contests and blog posts, gated by the v1 permissions (`judge.create_organization_problem`, `judge.create_private_contest`, `judge.edit_organization_post`) **and** org-admin membership (or edit-all), with the organization forced.
- Join-with-reason form for non-members, request state tabs (`?state=P|A|R|log`), request detail dialog with approve/reject.
- Public pickers for non-staff authors: org problem search, `/users?search=`, public `/contest-tags`, public `/licenses`.

**Admin leftovers and diagnostics**
- `/runtimes` and `/runtimes/matrix` (v1 `/runtimes/`, `/runtimes/matrix/`; linked from `/stats`, v2's `/status` substitute).
- `/admin/points-votes` (read-only ProblemPointsVote list, search/filter, no add — v1 is read-only too).
- `/admin/organization-requests` (read-only list; the org-manage request list also got an explicit user join).
- `/admin/classes`: full CRUD over the live columns (name/slug/org/active/description/access code/admins/members), contest picker kept working.
- `/admin/comment-locks`: CRUD plus enforcement on comment creation, with the `judge.override_comment_lock` bypass (v1 semantics: a lock row for the page string locks it).

**Leftover public screens**
- `/contributors`: unlisted-excluded profiles by contribution points with rank/rating/contribution columns (v1 ContribList); `/user/[username]/blog` lists that user's posts.
- Public blog authoring `/post/new` and `/post/[id]/edit` with the v1 gate (authenticated and solved-problem count ≥ 10, or superuser; org posts need org admin).
- Public organization creation `/organizations/create` (perm `judge.add_organization`, v1 admin-of limit, caller becomes org admin).
- Staff clone buttons surfaced for problems and contests (the admin problems list even defined a clone modal that was never rendered — now wired).
- Problems filter: problem types are now v1's shape — chosen types as removable tags above one closed "add" select — instead of an always-open chip cloud that pushed the filter panel past the viewport.

## Final audit closure (after the 214-route audit)

The exhaustive audit found three screens the earlier inventory had missed, plus four link/filter defects. All are closed:

- `/contests/tag/[name]` + `?tag=` on `GET /contests` — browse contests by tag (v1 `ContestTagDetail`).
- `/problem/[code]/tickets` + `?problem=` on the tickets API, gated like v1 (problem editors see the problem's tickets; other staff see their own/assigned).
- Resubmit: `GET /submission/:id/resubmit` (owner or `judge.resubmit_other`) prefills the submit form on `/problem/[code]?resubmit=<id>` with the previous source and language.
- Fixed the submission page's compare link, which pointed at a non-existent `/submission/<id>/diff` route (real route `/submissions/diff/[id]`).
- Contest participation tab supports `?user=` (v1 `/contest/<c>/participations/<user>`).
- Contest calendar reads/writes `?year=&month=` (v1 `/contests/<y>/<m>/`).
- `/users` and `/contributors` read `?search=` server-side (v1 `/users/find`, `/contributors/find`).

**Top header parity.** v1's nav is DB-driven (`judge_navigationbar`, rendered as `_(node.label)` in `templates/base.html:227`); v2's was hardcoded and missing Wiki, About, devlog, Server Status and Judges. v2 now serves `GET /navigation` (read-only against the shared table) and renders the header from it: v1 paths normalized to v2 routes (`/status/` → `/stats`), flat-page links kept, external links kept, v2-only entries (Ratings, Report) appended once, static fallback if the endpoint fails, active item from the row's regex with a path-prefix fallback. DB labels are translated through a catalog exactly like v1's gettext call (vi: Problems → Bài tập, Submissions → Các bài nộp, Leaderboard → Bảng xếp hạng, Organizations → Tổ chức, Contests → Các kỳ thi, About → Thông tin, Judges → Máy chấm); unknown/admin-added labels render verbatim. Nothing in v1 is written or modified.

**Filter shape parity (user request).** v1 uses multi-selects with the chosen values as tags; v2 had always-visible chip clouds. Both are now v1's shape: problem types on `/problems`, and verdict + language on `/submissions` — chosen values as removable tags above one closed "add" select that lists only the remaining options.

## Accepted gaps (documented, deliberate)

1. **Sites admin** — v2 is single-site by design (`CMSSiteID = 1`).
2. **Blog revision history / revert** — v1 uses django-reversion; v2 substitutes the request-level audit log (`/admin/audit-logs`, filterable by resource). No field-level diff or revert.
3. **`/status/` and `/status/oj/`** — replaced by `/stats` (judges online/load, queue, submission charts) plus `/runtimes`.
4. **Per-problem submission manager** (`/problem/<code>/manage/submission`) — replaced by `/admin/submissions` batch rejudge with preview, per-submission rejudge, and problem rescore-all.
5. **AJAX/ops helpers** — `/widgets/*`, `/judge-select2/*`, `/tasks/*`, `/mailgun/*`, `/newsletter/` (dead in v1). v2 uses JSON endpoints, inline previews and its own pickers.
6. **v1 Profile admin / SubmissionTestCase editor** — `/admin/users` and the submission detail's test-case panel + rejudge cover them (staff edit of individual test-case results is read-only in v2).
7. **Separate v1 list routes** — `/problem/<code>/submissions/`, `/submissions/user/<user>/`, `/user/<user>/submissions/` and the `/status/` listings are the same data as v2's filtered `/submissions` and `/stats` views; they exist as URLs in v1 only.

## Deliberate divergences

1. Statement "PDF" is a browser print view — no HTML→PDF engine and no new dependencies.
2. Solved list is AC-only (v1 listed any problem with points > 0).
3. Heatmap is UTC-keyed (v1 used the viewer's timezone).
4. Judge tier/capabilities are informational (v2 queue has no tier routing).
5. Impersonation is v2-native (signed claim + audit) vs `django-impersonate`.
6. Registrations act on v2 verification tokens; the v1 table is mapped read-only.
7. Contest clarifications are a v2 object model (v1 used comments/tickets).
8. Navbar is list + parent select, not drag-and-drop.
9. Language limits are a global CRUD page (v1 inline only).
10. Per-problem contest ranking withholds peers during a hidden scoreboard (stricter than v1).
11. Public blog editor reuses the markdown textarea but not the staff-only image upload.
12. Contest leave keeps participation history (v1 also leaves rating/submission rows; v2 does not delete).

## Post-deploy fixes found by live verification

1. **Profile activity grid and statistics returned zeros** for a user with 3,601 submissions, and the public daily-submission chart was empty. GORM mapped the SQL aliases `total_submissions`/`accepted_submissions` onto fields named `TotalSubs`/`AcceptedSubs` (never scanned), and `SELECT DATE(date)` returns `time.Time` under `parseTime=true`, which cannot scan into a string, so the activity queries errored silently. Fixed in `3c37824` (typed `day` scanner + renamed fields, also applied to `/stats/submissions/daily`). Live after deploy: 3,601 / 1,105 accepted, 194 submissions across 30 heatmap days, dated daily buckets.
2. **`/runtimes` returned 500 in production**: `ORDER BY key` is a syntax error in MySQL (`key` is reserved); SQLite accepts it, so tests passed. Qualified as `judge_language.key` (`ef55904`).
3. **The whole site returned 500 after the Phase 6 rollout**: Next.js refuses to serve any page when two dynamic segments under one parent use different names — the new `/post/[id]/edit` sat next to `/post/[slug]`. `next build` does not catch this; the server logs on `root@claoj` showed `You cannot use different slug names for the same dynamic path ('id' !== 'slug')`. Fixed in `6fedfd9` by moving the edit route to `/post/[slug]/edit` (same `<id>-<slug>` segment as the post page), plus a jest test that scans the app tree for conflicting segment names so it cannot regress.

## Known limits

See `docs/roadmap-tracker.md`. Highlights: impersonated refresh TTL 2 h; points backfill endpoint is synchronous (curl-only); submission files on local disk with no cleanup; solved list AC-only; org filter capped at 100; contest list groups status per page; diff matrix capped at the latest 200 submissions; email-change confirm consumes the token before the duplicate-address re-check (a taken address needs a new request).

## Verification evidence

- `go build ./...` exit 0; full `go test ./... -count=1` green on the merged Phase 6 tree (one pre-existing flake in `bridge` on a parallel first run; the package and the full suite pass on re-run).
- jest 44 suites / 369 tests; `tsc --noEmit` clean; `npm run build` succeeds (93 routes).
- i18n: 2,715 keys in both `en` and `vi`; 2,949 `t()` references resolve.
- eslint: 11 errors on the pre-Phase-6 baseline in the same six files, unchanged after Phase 6 (all pre-existing `no-explicit-any` / hook rules).
- CI run for the Phase 6 merge: see `docs/roadmap-tracker.md`; Watchtower rollout verified live on beta.

### Live checks on beta after the Phase 6 rollout

| Screen | Evidence |
|---|---|
| `/contributors` | "Contributors", columns Rank / User / Solved / Contribution, 20 rows/page |
| `/runtimes` | "Runtimes", ID / Name / Runtime Info, **15 rows** (v1 also 15), online-only note for non-staff |
| `/runtimes/matrix` | "Version matrix", 4 judge columns (Vịt Con, Vịt Graphics, Vịt Nhân Phẩm, Vịt Toàn Năng), 15 language rows, up-to-date/outdated legend |
| `/problems/suggest/new` | form fields code, name, description, points, time_limit, memory_limit + taxonomy/source/partial scoring |
| `/post/new` | Title, Publish on, Organization (org picker), Visible, Content |
| `/organizations/create` | Name, Slug, About, Logo override image |
| `/user/buituananh270908/blog` | "Blog posts from buituananh270908" + empty state |
| `/admin/organization-requests` | username / organization / state / time / reason, **417 rows** paginated |
| `/admin/points-votes`, `/admin/classes`, `/admin/comment-locks` | correct columns and empty states (no rows in the shared DB) |
| `/admin/problems/013_findn/diff` | **34 submissions × test cases** matrix with per-case AC 1/1 cells |
| `/submissions?problem=013_findn&best=1` | **8 rows, 8 distinct users** (one best per user) |
| `/submissions?contest=cplusplusnc` | **2,513** contest-scoped rows |
| `/contest/cplusplusnc/rank/arr01` | "Best solutions for In ngược dãy số in C++ nâng cao", 30 rows ranked by contest points then time |
| `/contest/cplusplusnc/submissions/tik25_ntnghia` | 117 rows, all that user's |
| `/organization/1` | member table with rating/solved/points/PP, 136 members |
| Problems filter (v1 shape) | `#problem-types` select with 78 options; choosing "2SAT" adds a removable tag, drops the option from the list and filters 20 rows → 2 |
| Submissions filter (v1 shape) | `#verdict-filter` (12 options) + `#language-filter` (16 options); WA + C tags → total **188,881 → 70**; removing the WA tag → **240** |
| `/license/[key]` | route + `GET /license/:key` live; the shared DB has **0 licenses** (v1 admin also shows "0 licenses"), so there is no data to display |