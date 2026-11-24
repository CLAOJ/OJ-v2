# Phase 5 — Public / normal-user parity

**Spec:** `docs/admin-ux-gap-report.md` §4 + Roadmap Phase 5.
**Stack:** Go/Gin + Next.js 16, shared Django MySQL, no migrations, no new deps.

## Global constraints
- Keep v2 design; reuse existing helpers (`MathRenderer`, `Markdown`, `PaginationBar`, `lib/contestPhase`, `lib/listSort`, `SortableHeader`, `Comments`).
- i18n en+vi; tests for new pure helpers and new API behavior; commit per task; no push.
- Public endpoints must not leak private data (org-private problems/contests, hidden comments, unpublished posts).

## Recon corrections / rulings
1. Already fixed, do not redo: contest description rendering, "view all past contests" toggle, problem points cap, raw-HTML rendering, submission memory/score header, editorial deep-link, home comment stream.
2. **Contest list pagination bug:** FE requests `page_size=1000`, API caps at 100 and returns no `total` → only a 50-row window. Fix API total + FE paging.
3. **`contest.time_limit` is microseconds** (Django DurationField); FE divides by 60 and labels minutes, and shows "Infinite Window" for 0. Fix the unit conversion in FE (and any other consumer).
4. **PP disagreement is an API bug:** both `scoring.CalculateProfilePoints` and `/user/:u/pp-breakdown` omit v1's `is_organization_private = 0` / `is_archived = 0` filters and use different ordering. Fix once in the shared scoring path and make the breakdown call it.
5. **Statement PDF:** v1 renders HTML with wkhtmltopdf; v2 has no HTML→PDF engine and no new deps are allowed. Ship a print-optimized statement view + "Download PDF" (browser print) for problems without an uploaded PDF; keep the uploaded-PDF path unchanged. Documented divergence.
6. **File-upload submit:** the judge already supports `file_only` (jobs/judge.go:64 sends `SiteFullURL + source`). Add a multipart submit path that stores the file under `data/submissions/<hex>.<ext>` and serves it from a public, name-whitelisted route; plain-text files are read client-side into the editor.
7. Wiki/About 404s are closed by Phase 4c Task 6 (flat pages) + restoring the footer links; Phase 5 only wires the links.

## Task 1 — Contest public pages
API: public announcements read path (respect visibility/push state); contest list `total` + honest page cap; ranking: include virtual participants behind a toggle, return display name/org short name; participations endpoint usable publicly (currently filters virtual=0).
FE: announcements section on the contest page; real pagination for past contests; live countdowns in the contest list; Participation tab; scoreboard CSV export, Total AC row, HH:MM:SS times, name/org + virtual toggles; correct time-limit/window rules; deep-linkable tabs (`?tab=`).

## Task 2 — Problem discovery
API: `type` filter, `has_editorial` filter, Suggest list endpoint (`is_public=false AND suggester IS NOT NULL`, gated `judge.suggest_new_problem`).
FE: group filter control (API already supports `group`), type filter, has-editorial filter, #AC column (`user_count` already returned), Suggest page/list.

## Task 3 — Organization public pages
API: `/organization/:id/problems`, `/contests`, `/submissions` (scoped, privacy-respecting, paginated); member DTO gains points/problem_count/rating/performance_points.
FE: org tabs for problems/contests/submissions, member ranking columns, PP column on the org list, "your organizations" shortcut.

## Task 4 — Profile
API: blog list by author; yearly activity heatmap (365 days, per-day counts); richer solved-problem rows (name, date, language, tests); PP fix per ruling 4 (single shared calculation).
FE: Blogs tab, heatmap, "View submissions" link, richer solved list; verify header == breakdown after the fix.

## Task 5 — Problem detail
API: multipart file submit for `file_only` languages + public file serving route (ruling 6); statement print/PDF path (ruling 5).
FE: file-upload submit UI, problem-scoped report/clarification link (`/ticket/create?problem=<code>` honoured), sample Copy buttons, allowed-languages display, partial-scoring indicator, Download PDF/print button.

## Task 6 — Submissions / users / home polish
API: submissions org filter + multi-value verdict/language filters; users list `display_name` + organization.
FE: submissions time/memory columns, friendly language names, multi-select filters; users solved column + display name/org inline; home top contributors + human-readable countdown; restore Wiki/About links once flat pages exist.

## Acceptance
- Every §4 gap is either closed or explicitly documented as a divergence with a reason.
- Suites green; lint baseline unchanged; no migrations/new deps; no push.