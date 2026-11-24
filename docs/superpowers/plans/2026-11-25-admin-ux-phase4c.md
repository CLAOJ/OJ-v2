# Phase 4c — Judging & System + v1-only admin areas

**Spec:** `docs/admin-ux-gap-report.md` §3.2 + Roadmap Phase 4 items 7–12.
**Repo:** worktree `OJ-v2-wt/feat/admin-ux-phase4c` (branch `feat/admin-ux-phase4c`, from main 573017e).
**Stack:** Go/Gin + Next.js 16, shared Django MySQL, no migrations (map existing columns only).

## Global constraints
- v2 design; minimal diffs; no new deps; no migrations; i18n en+vi; tests for new behavior; commit per task; no push.
- Id spaces: `judge_profile.id` for user-facing pickers, `auth_user.id` only where the column is an auth FK (verify per table).
- Permission gates follow v1 (`judge.*` codenames), not blanket superuser, unless v1 itself is superuser-only.
- No secrets (auth keys of judges are shown in v1 admin; keep them out of list payloads).

## Recon corrections (rulings)
1. **Impersonation has no v1 table.** v1 uses `django-impersonate` (superuser-only, logging disabled). Build a v2 mechanism + our own audit trail.
2. **Registration profiles** come from django-registration-redux; v2 verifies via one-time tokens. Ship a v2 pending/verified view + activate/resend using v2 tokens; map the v1 table read-only for display parity.
3. **`file_size_limit` unit is MB everywhere** (v1 model help, v1 upload check ×1024², judge receives raw int). Fix the v2 FE label to MB; do NOT convert values.
4. **`is_locked` is a v1 property** (`locked_after < now`), not a column — derive it. `is_archived` is a real column → map it.
5. **`judge_navigationbar.regex` is NOT NULL** and unmapped in v2 → creating nav items today can fail; mapping it is a bug fix.
6. Judge `disconnect`/`terminate` need an agent protocol v2 lacks → out of scope; tier is mapped and editable but documented as not used for queue routing.
7. Submission history: v1 uses django-reversion. v2 has no revision store → link to the v2 audit log filtered by object instead of inventing one.

## Task 1 — Submissions admin
Backend:
1. Map `Submission.IsArchived` (`judge_submission.is_archived`).
2. `GET /admin/submission/:id`: full detail — user/problem/language names, date, time, memory (KB→MB in FE), points, case_points/case_total, status/result, error (compile errors), judged_on (name), judged_date, rejudged_date, locked_after + derived is_locked, is_archived, contest_object_id, source text, testcases (id/status/time/memory/feedback), contest-submission link.
3. `PATCH /admin/submission/:id` gated `judge.change_submission`: language_id, status, result, points, case_points, case_total, judged_on_id, judged_date, error, locked_after (gate `judge.lock_submission`), is_archived (gate `judge.archive_submission`), contest_object_id (validated), source (via SubmissionSource). Validate enums/ids; no mass assignment of user/problem/date.
4. List: add id, problem code+name, language name, memory, total count (filtered); split status vs result filters; keep existing filters.
5. `POST /admin/submissions/bulk-delete` (gate `judge.delete_submission`, capped, confirm in UI).
6. Tests: detail shape, patch gates (lock/archive/change), source update, contest association validation, filtered total, bulk delete cap/permission.
Frontend:
7. New `/admin/submissions/[id]` page: read-only header + edit form for the writable fields (permission-aware), source viewer/editor, testcase table, links to audit log (object filter) and public submission.
8. List: ID/problem/language/memory columns, separate status/result filters, total, checkbox selection + bulk delete with confirmation.

## Task 2 — Impersonation (superuser-only)
1. `POST /admin/impersonate/:userId` (superuser): issue tokens for the target with an `act_as`/impersonator claim; store the real actor server-side (one-time/refresh store) so exit restores the original session. Reject: target superuser, self, inactive target, already impersonating.
2. `POST /admin/impersonate/exit`: restore the original actor's tokens, revoke the impersonated family.
3. Middleware: expose `impersonator_id`/`is_impersonating` on `/user/me`; audit every start/exit (actor + target).
4. FE: persistent banner "You are viewing as X — Exit" + "Login as" action on the users list (superuser only).
5. Tests: start/exit round-trip, target guards, audit rows, tokens of the original session restored, non-superuser 403.

## Task 3 — Registration / activation
1. Map `RegistrationProfile` (table `registration_registrationprofile`) read-only.
2. `GET /admin/registrations`: pending/verified list (auth_user.is_active + profile), search, total; show v1 activation state when present.
3. `POST /admin/registration/:userId/activate` (superuser): set is_active=true, mark v1 profile activated if the row exists, revoke outstanding verify tokens.
4. `POST /admin/registration/:userId/resend`: issue a v2 verification token + email (existing email helper); 400 if already active.
5. FE `/admin/registrations` page with filters + Activate/Resend actions; i18n.
6. Tests: activate/resend guards, idempotency, token issuance, list filters.

## Task 4 — Judges + Languages
Judges:
1. Map `Judge.Tier`; expose in list/detail/update; FE tier input (positive int, default 1).
2. Judge update: name, auth_key, tier, description, problem/runtime M2M pickers; keep online-edit 409 guards for name/key (v1 readonly while online).
3. Write `last_ip` on judge connect/status (bridge handler) so the field is not permanently empty.
4. FE: editable name/auth key/tier + capabilities pickers; document tier is informational in v2.
Languages:
5. FE label "File Size Limit (MB)" (ruling 3); expose template/description in list/detail/edit.
6. Gate update/delete with `judge.change_language`/`judge.delete_language` instead of superuser-only; implement v1's `include_in_problem` side effect (attach language to all problems on create/update when true).
7. Tests: tier round-trip, online guards, language perm gate, include_in_problem side effect, last_ip written.

## Task 5 — Navigation bars
1. Map `NavigationBar.Regex`; validate it compiles (Go regexp) on create/update; include in detail/list.
2. Parent picker (exclude self/descendants), order field; keep/repair MPTT insert/update/delete (rebuild subtree shifts).
3. FE: parent select + regex textarea with a live match preview against sample paths; i18n.
4. Tests: regex validation (invalid → 400), parent cycle rejection, MPTT consistency after create/move/delete, create succeeds with NOT NULL regex.

## Task 6 — Flat pages + Redirects
1. Map `django_flatpage` (+ `django_flatpage_sites`) and `django_redirect` models (no migrations).
2. Admin CRUD for both (superuser or `flatpages.change_flatpage` / `redirects.change_redirect`), list + search + total.
3. Public rendering: flat page route (`/page/<url>` and/or catch-all for known paths) rendering markdown with the existing renderer; redirect handling for `django_redirect` (server-side or Next middleware) with loop protection.
4. FE admin pages `/admin/flat-pages`, `/admin/redirects`; i18n.
5. Tests: CRUD, site scoping, public render, redirect match + loop guard.

## Acceptance
- Admin can inspect/edit a submission, impersonate (superuser) and exit, activate/resend registrations, manage judge tier/capabilities and language fields, build navbar hierarchy with highlight regex, and manage flat pages/redirects.
- Suites green; lint baseline unchanged; no migrations; no push.

## Out of scope
Judge disconnect/terminate (agent protocol), submission revision history (no v2 store), social-auth admin, sites admin.