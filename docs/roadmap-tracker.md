# CLAOJ v2 → v1 parity roadmap tracker

Source of truth for gaps: `docs/admin-ux-gap-report.md`. This file tracks execution status and evidence.
Live v2: https://beta.claoj.edu.vn · Live v1: https://claoj.edu.vn · Deploy: push `main` → GHCR images → Watchtower (~120 s).

| Phase | Scope | Status | Merge commit | Evidence |
|---|---|---|---|---|
| 0 | P0 bugs (points cap, memory unit, raw HTML, dead contest button, contest description API), language-limits crash, audit log, breadcrumbs, aria | Done | in main | live-verified in session |
| 1 | Problem form parity (people/publishing/access/social), translations CRUD, MOSS contest list | Done | in main | live-verified |
| 2 | Contest form parity (rating/scoreboard/access/details/settings, classes, orgs, permission gates), announcements CRUD | Done | in main | live-verified |
| 3 | List UX: sortable headers, creator/year filters, authors column, dense rows | Done | in main | live-verified |
| 4a | Users + Groups admin (tabbed editor, roles/permissions/security, session revocation, group membership) | Done | `8938ce0` | go test exit 0; jest 272; CI 45454748538 green; live 4-tab editor + group members |
| 4b | Organizations, Blog (+markdown image upload), Tickets, Taxonomy/Tags, Comments, Licenses | Done | `573017e` | go test exit 0; jest 277; CI 45475795661 green; live checks on all five admin pages |
| 4c | Submissions admin detail/edit, Impersonation, Registration activate/resend, Judges, Languages, Navbar, Flat pages/Redirects | Done | `722d368` | go test exit 0; jest 28 suites / 297 tests; `npm run build` ok; 6 task reviews + final cross-task review; CSRF added to admin group |
| 5 | Public pages: contest content/announcements/scoreboard, problem discovery filters, org lists, profile, problem page (auto-PDF, file submit), polish | Done | `22749b2` | go test exit 0; jest 37 suites / 343 tests; `npm run build` ok; 6 task reviews + cross-task final review; fixed contest-submission linkage, scoreboard PDF bypass, announcement gating, grading-end point recalculation |
| 6 | Closing screens: best-solutions list, setter diff matrix, raw statement, contest submissions/ranking/leave/abort, suggestion form, public license, email change, org self-service creates/requests/join, runtimes + PointsVote/OrgRequest/Class/CommentLock admin, contributors/user blog, public blog authoring, public org create, clone buttons | Done | `a548fac` | 7 tasks; go test exit 0; jest 44 suites / 369 tests; `tsc` clean; `npm run build` (93 routes); i18n 2,715 keys both locales; eslint baseline unchanged; CI run 45524928616 |
| R | Full v1↔v2 screen comparison (public + admin) and gap/result report | Done | `23a0777` | report: `docs/v1-v2-screen-parity-report.md` — 214-route audit: 55 parity / 76 substitute / 79 not-a-screen / 0 open gaps (Sites + blog revision history documented) |
| 7 | Final audit closure: contest tag page/filter, per-problem tickets, resubmit prefill, compare-link fix, participation `?user=`, calendar deep link, `?search=` deep links, DB-driven header with v1 gettext labels, problems/submissions filters as multi-select tags | Done | `23a0777` | go test exit 0; jest 46 suites / 405 tests; `tsc` clean; `npm run build`; i18n 2,740 keys both locales; routing guard green; `next start` smoke 200s |

## Known limits carried forward
- Comment hide does not cascade to replies (v1 bulk actions do not either).
- `access_code` on organizations is outside the org field gate.
- Markdown images live on the v2 data volume; no cleanup/retention.
- Problem statement PDF: v1 renders one for every problem with wkhtmltopdf. v2 has no HTML→PDF engine (no new deps allowed), so a problem without an uploaded PDF gets a print-optimized view (`/problem/<code>/print`) and a "Download PDF" button that opens it with the browser's print dialog. Uploaded-PDF problems keep the existing viewer.
- File-only submissions are stored under `data/submissions/<16hex>.<ext>` and served by the public `/api/submission-file/:name` route (server-chosen names; the upload is validated against the language's configured extension, and serving pins a safe Content-Type — `text/plain` for text-ish, `application/zip` for zip, `application/octet-stream` otherwise — with `Content-Disposition: attachment` and `nosniff`). `language.file_size_limit = 0` means uploads are disabled (v1 parity), and the body is hard-capped at 32 MiB.
- Latent GORM `belongsTo` relation shape on Submission/Rating/Participation models (untouched; fixed only on Ticket).
- Contest/problem clone flows still drop some new settings.
- Phase 4c: judge tier/capabilities are informational (v2 queue has no tier routing); impersonated refresh family lives 2 h; exit restores a standard 7-day actor family (remember-me not preserved); registrations resend has no rate limit; flat-page `template_name`/`enable_comments` are mapped but not editable; `/api/redirects-map` is a public path-only dump; nav parent picker offers only the loaded page.
- Phase 5: statement "PDF" is a browser print view (no HTML→PDF engine, no new deps); submission files live on local disk with no cleanup and are public by random name; solved list is AC-only (v1 lists points>0); heatmap is UTC-keyed (v1 uses the viewer's tz); profile blogs tab hides the owner's drafts; `POST /admin/recalculate-points` is synchronous/curl-only (504 risk on large installs, idempotent re-run); contest list groups upcoming/ongoing/past per page; 32 MiB upload body cap; org filter dropdown capped at 100 without search.
- The points backfill (`POST /api/v2/admin/recalculate-points`, superuser) runs synchronously inside the request and has no admin button: on a large install it can outlive the proxy timeout, so run it once with curl (or the API client) from the server itself and check the JSON counts in the response.
- Phase 6: public blog authoring (`/post/new`, `/post/<slug>/edit`) reuses the markdown textarea but not the admin image-upload button (the upload endpoint is staff-only); title-derived slugs get a `-2`, `-3` suffix when taken; the blog problem-count gate (10) and the organization admin-of limit (3) are constants mirroring the v1 settings, not v2 config.
- Route changes need a runtime check, not just `next build`: Next.js 500s the whole site when two dynamic segments under one parent use different names (`/post/[id]/edit` beside `/post/[slug]` did exactly that in production). `__tests__/routing/dynamicSegments.test.ts` guards it; also smoke-test `npm run start` on a few routes before pushing. Server logs: `ssh root@claoj "docker logs --tail 60 claoj_v2_web"`.
- MySQL reserved words are not quoted by GORM's raw `Order(...)`: qualify them (`judge_language.key`, not `key`). SQLite tests accept the bare name, so only the live deploy catches it.

## Deliberate scope rulings
- No org-specific 2FA gate (v1 has none; global middleware only).
- No license file field (v1 license text is markdown; image upload added instead).
- No organization member manager (v1 admin exposes admins + read-only member count).
- ProblemGroup "included problems" is read-only (v1 field is a broken FK reverse set).
- Comment parent cannot be reparented in v2 (no safe MPTT helper).
- v1's Django Profile admin (`/admin/judge/profile/`) is replaced by `/admin/users` and the tabbed user editor: the same profile, role and security fields, with v1-only admin columns read-only.
- v1's SubmissionTestCase editor (`/admin/judge/submissiontestcase/`) is replaced by the read-only testcase panel on the admin submission page plus the rejudge action; testcase rows belong to the judge and are not hand-edited.
- v1 Profile admin's per-object permission matrix is edited per user in the user editor, not as a standalone changelist.