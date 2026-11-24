# Phase 0 — UX bug fixes & quick wins

**Spec:** `docs/admin-ux-gap-report.md` → Executive summary + Roadmap Phase 0.
**Repo:** worktree `OJ-v2-wt/feat/admin-ux-phase0` (branch `feat/admin-ux-phase0`, from main 180598a).
**Stack:** web = Next.js 16 (`claoj-web`), API = Go/Gin (`claoj`). Shared MySQL with v1 — **no migrations**.

## Global constraints
- Keep v2's visual design. No redesigns. Minimal diffs.
- Do not touch unrelated files. No new dependencies unless unavoidable.
- All user-facing strings go through next-intl (`src/i18n/en.json` + `vi.json`).
- Tests: targeted jest tests for web changes, `go test` for Go changes. Run `npm run lint` in `claoj-web`; `go build ./...` in `claoj`.
- Commit per task, conventional message. Do not push, do not deploy.

## Task 1 — Public-page P0 bugs (claoj-web only)
1. **Problems >50 pts unreachable.** `src/app/[locale]/problems/ProblemsPageContent.tsx`: `pointsMax` defaults to 50 and the slider's max is 50, and search inherits the filter, so 19 problems (incl. 100-pt ones) are unfindable. Required: default to no points restriction (or the true maximum, ≥100), slider max ≥ highest problem points (100), and search must not silently apply the 50 cap. Acceptance: a 100-point problem found by search and by clearing filters; no upper bound below 100.
2. **Submission memory unit.** `src/app/[locale]/submission/[id]/page.tsx:323` prints `sub.memory.toFixed(1) + 'MB'` but the API value is KB → shows 3944.0MB for 3.85MB. Required: convert KB→MB with sensible precision (e.g. 3.9 MB / 3.85 MB), never ×1000 wrong. Also split the confusing header score `1/20` into tests passed and points the way v1 does (`20/20 (1.000/1.000 points)`), using data already present.
3. **Raw HTML in statements.** Problem statements show literal `<i>...</i>` (e.g. problem `swgame`). Required: inline HTML renders (as v1 does) in the problem statement renderer — sanitized against XSS. Find the renderer used by the problem page; reuse an existing approach/dependency (`rehype-raw` + sanitization) if present in the repo.
4. **Dead "View all past contests".** `src/app/[locale]/contests/ContestsPageContent.tsx:289` button does nothing. Required: it navigates to a working view of all past contests (paginated or an expanded/filtered list showing more than the 10 recent), matching existing contest list patterns. The API must already support the needed pagination — if not, note it in the report instead of inventing an endpoint.

## Task 2 — Admin quick wins + Language Limits crash (claoj-web + small Go)
1. **Language Limits page crash.** `src/app/[locale]/admin/language-limits/page.tsx` crashes to the error boundary on load / hangs at "0 of 0" though the API returns 200. Diagnose and fix the render crash. Also slim the API row payload: `claoj/api/v2/language_limit.go` currently embeds the full Problem object (incl. description) per row — return only what the table needs (problem id/code/name, language, limits). Keep response shape compatible with the frontend.
2. **Public contest description missing.** `claoj/api/v2/contest/*` DTO has no `description`, so the public contest page always shows the placeholder (`ContestPageContent.tsx:234` already renders `contest.description || placeholder`). Add `description` (and `summary` if trivially available) to the public contest detail response from the existing model fields. No FE change should be needed.
3. **Admin list pagination + totals.** Problems and Contests admin lists: show page numbers/total (API already returns `total`, `page`, `page_size`). Follow the existing pagination component used elsewhere in the admin.
4. **Accessible row actions.** Add `aria-label`/`title` to icon-only row action buttons on admin users, blog-posts, comments, taxonomy, languages, navigation-bars (edit/delete/ban/set-password etc.). Use existing i18n keys where possible.
5. **Breadcrumbs** on admin pages (Home › section › page), following the pattern used on public pages or in the admin layout.
6. **Orphaned flows.** Problem edit page (`admin/problems/[code]/edit`): add the missing Editorial link next to Manage Data / Clarifications. Contest create page (`admin/contests/create`): add "Lock After" and "Format Config" controls (API create/update already accepts `format_config`; lock uses the existing lock endpoint).

## Task 3 — Audit log recording (Go)
`/admin/audit-logs` and the dashboard "Recent changes" are always empty despite live admin API traffic. v2's audit model is v2-owned (`auditlog` package). Required: find why no entries are written, fix so successful admin mutations (create/update/delete) are recorded with user, action, resource, result; add/extend Go tests proving an admin API call produces a log row retrievable via the admin audit-log endpoint. Keep payloads small; do not log secrets (password bodies, auth keys).