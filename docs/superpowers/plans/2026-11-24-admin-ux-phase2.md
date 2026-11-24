# Phase 2 — Contest form parity (v1 fields in v2)

**Spec:** `docs/admin-ux-gap-report.md` §2 + Roadmap Phase 2.
**Repo:** worktree `OJ-v2-wt/feat/admin-ux-phase2` (branch `feat/admin-ux-phase2`, from main e7f8541).
**Stack:** Next.js 16 web + Go/Gin API, shared MySQL, **no migrations**.

Current state: `models.Contest` maps most columns (rating floor/ceiling, rate_all, scoreboard_visibility, scoreboard_cache_timeout, push_announcements, hide_problem_authors, show_short_display, use_clarifications, is_organization_private, og_image, logo_override_image, problem_label_script, locked_after, points_precision, organizations, banned_users, private_contestants, tags, problems, staff). Admin create/update DTOs expose only a subset. Missing mappings confirmed in v1 (`OJ/judge/models/contest.py`): `view_contest_scoreboard` (M2M Profile), `limit_join_organizations` (bool), `join_organizations` (M2M Organization), `classes` (M2M Class — table `judge_class`), `performance_ceiling_override` (int). Verify v1's "No comments" admin field's actual column while implementing.

## Global constraints
- Keep v2 design; minimal diffs; no new dependencies; no migrations (map existing Django-owned columns only).
- i18n en+vi; tests for new behavior; commit per task; no push/deploy.

## Task 1 — Backend: contest fields (Go)
1. `models/contest.go`: add the missing mappings (`ViewContestScoreboard []Profile` m2m `judge_contest_view_contest_scoreboard` — verify through-table name in Django, `LimitJoinOrganizations bool`, `JoinOrganizations []Organization` m2m, `Classes []Class` m2m `judge_contest_classes` (add a minimal `Class` model mapping `judge_class` if absent), `PerformanceCeilingOverride *int`). Check `Oj/judge/models/contest.py` + migrations for exact column/table names; do not invent.
2. Admin create/update DTOs + service (`admin_contest.go`, `service/contest`): accept and persist all of: rating_floor, rating_ceiling, rate_all, scoreboard_visibility, scoreboard_cache_timeout, push_announcements, hide_problem_authors, show_short_display, use_clarifications (map v1's no-comments control only if a real column exists — otherwise skip and note), og_image, logo_override_image, problem_label_script, is_organization_private + organization_ids, private_contestant_ids, banned_user_ids, view_scoreboard_user_ids, limit_join_organizations + join_organization_ids, class_ids, performance_ceiling_override.
   - Follow Phase 1 patterns: pointer DTO for update, relations.Sync for m2m, validation of scoreboard visibility against v1 choices, length caps for image columns (150) and handle/label constraints.
3. Detail response (`AdminContestDetail`) must include every field above (ids + display refs where the form needs chips).
4. Tests: create+update round-trip, relation replace/clear, invalid scoreboard visibility → 400, length caps.

## Task 2 — Frontend: contest form sections (web)
Extend `ContestFormData`, `ContestForm`, `adminForms.ts` mappers and the create/edit pages:
- **Rating**: rated, rate all, rating floor, rating ceiling, performance ceiling override.
- **Scoreboard**: visibility select (v1 codes), cache timeout, points precision.
- **Access/Privacy**: private-to-organizations + org picker, private contestants picker, banned users picker, view-scoreboard users picker, limit-join-organizations + join organizations picker, classes picker.
- **Details**: og image, logo override, problem label script (textarea), existing format config stays.
- **Settings**: push announcements, hide problem authors, show short display, no-comments/use-clarifications (only if Task 1 mapped it).
- Keep Lock After on edit (already exists), format config on create+edit, staff/problems/tags sections unchanged.
- i18n en+vi; jest tests for the extended round-trip mapper.
- Acceptance: edit an existing contest, all v1 fields render and round-trip.

## Task 3 — Contest announcements admin (Go + web)
Model `ContestAnnouncement` exists. Add admin CRUD: list/create/update/delete under the contest admin routes, guarded like other contest admin endpoints. Web: announcements tab/page linked from contest edit. Resend is deferred (needs the notification pipeline) — note in the report.

## Acceptance
- Every v1 contest form field is present in v2 create/edit and persists (verified by API round-trip tests; one real edit checked on the deployed build).
- No schema changes; all mapped columns exist in Django.
- Announcements CRUD works; no announcement resend in this phase.
- All suites green; lint baseline unchanged.