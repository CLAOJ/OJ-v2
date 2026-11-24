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
| 4c | Submissions admin detail/edit, Impersonation, Registration activate/resend, Judges, Languages, Navbar, Flat pages/Redirects | In progress | — | worktree `OJ-v2-wt/feat/admin-ux-phase4c` |
| 5 | Public pages: contest content/announcements/scoreboard, problem discovery filters, org lists, profile, problem page (auto-PDF, file submit), polish | Pending | — | — |
| R | Full v1↔v2 screen comparison (public + admin) and gap/result report | Pending | — | report will be `docs/v1-v2-screen-parity-report.md` |

## Known limits carried forward
- Comment hide does not cascade to replies (v1 bulk actions do not either).
- `access_code` on organizations is outside the org field gate.
- Markdown images live on the v2 data volume; no cleanup/retention.
- Latent GORM `belongsTo` relation shape on Submission/Rating/Participation models (untouched; fixed only on Ticket).
- Contest/problem clone flows still drop some new settings.

## Deliberate scope rulings
- No org-specific 2FA gate (v1 has none; global middleware only).
- No license file field (v1 license text is markdown; image upload added instead).
- No organization member manager (v1 admin exposes admins + read-only member count).
- ProblemGroup "included problems" is read-only (v1 field is a broken FK reverse set).
- Comment parent cannot be reparented in v2 (no safe MPTT helper).