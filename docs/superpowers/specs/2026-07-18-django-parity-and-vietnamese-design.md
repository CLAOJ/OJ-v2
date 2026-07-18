# OJ-v2: Django Schema/Permission Parity + Full Vietnamese Support — Design

**Date:** 2026-07-18
**Status:** Approved by user (all 4 sections)

## Background

OJ-v2 (Go backend `claoj` + Next.js frontend `claoj-web`) is a rewrite of the
Django/DMOJ-based CLAOJ site (`OJ/`). Both are intended to run **side-by-side
against the same live MySQL database**.

Problems today:

1. **Schema drift.** OJ-v2 ships 7 of its own migrations (`claoj/db/migrations/`)
   that create a parallel role/permission system (`judge_role`, `judge_permission`,
   `judge_role_permissions`, `judge_profile_roles`), add columns to Django-owned
   tables (`judge_contest.max_submissions`, `judge_problem.suggestion_*`), and
   create new tables (`audit_log`, `refresh_token`). Two of them (contest tags,
   webauthn) duplicate tables Django already has. Running OJ-v2 therefore
   requires mutating the production schema first — unacceptable for side-by-side.
2. **Permission divergence.** OJ-v2 invented its own permission vocabulary
   (`problems.view_hidden`, …) instead of Django's (`judge.see_private_problem`, …),
   so access control configured on one site does not apply to the other.
3. **Broken Vietnamese rendering.** The frontend loads the `Outfit` Google font
   with `subsets: ["latin", "latin-ext"]`. Outfit has **no Vietnamese glyphs**,
   so Vietnamese diacritics fall back to a system font → visibly mixed/broken text.
4. **Incomplete translation coverage.** Some UI strings are hardcoded in English.

## Goals

- OJ-v2 runs against the untouched Django database. **Zero DDL, zero migrations.**
  The schema is 100% owned by Django (`OJ/judge/migrations/`).
- Roles and permissions are Django's own (`auth_group`, `auth_permission`,
  `is_staff`, `is_superuser`) — one source of truth, live on both sites.
- All OJ-v2 functionality that doesn't require schema changes is preserved.
- Vietnamese renders perfectly everywhere; all UI strings are translatable and
  translated (en + vi).

## Non-goals

- Changing the Django site (`OJ/`) in any way.
- Making Vietnamese the default locale (stays as currently configured).
- Session sharing between the two sites (each keeps its own login mechanism;
  same user accounts/passwords work on both because `auth_user` is shared).
- Data migration/import tooling.

---

## Section 1 — Database: 100% Django-managed schema

**Rule: OJ-v2 performs only DML (rows). Never DDL (tables/columns/indexes).**

### Changes

1. **Delete** `claoj/db/migrations/` (all 7 files) and the migration runner /
   any startup hook that invokes it. `db.Connect()` already skips AutoMigrate;
   keep it that way and add a comment stating the DDL ban.
2. **Model audit.** Every GORM model in `claoj/models/` is verified
   column-by-column against the corresponding Django model in `OJ/judge/models/`
   (source of truth, including CLAOJ-fork customizations — not upstream DMOJ):
   - Column exists in Django schema → keep.
   - Column/table only in v2 migrations → remove the field and all code paths
     that read/write it.
   - Django column missing from the Go model that v2 features need → add it
     (reading existing schema is always safe).
3. **Reuse Django's existing tables** for features v2 duplicated:
   - Contest tags → `judge_contesttag`, `judge_contest_tags` (already in Django).
   - WebAuthn → `judge_webauthncredential`, `judge_profile.is_webauthn_enabled`
     (already in Django, migration 0105).
   The Go models point at the Django table shapes exactly.
4. **Removed tables' code**: `models/role.go`, `models/audit_log.go` (MySQL-backed
   part), refresh-token MySQL repository — deleted or re-homed per Sections 2–3.
5. **Optional cleanup script** `scripts/cleanup_v2_tables.sql` (manual, never
   auto-run) that drops the v2-only tables/columns from a DB where the old
   migrations were already applied:
   `judge_role`, `judge_permission`, `judge_role_permissions`,
   `judge_profile_roles`, `audit_log`, `refresh_token`,
   `judge_contest.max_submissions`, `judge_problem.suggestion_*` columns
   (+ their FK/indexes). Leaving them in place is also fine — Django ignores them.

### Verification

- Integration test boots the backend against a pristine Django-schema database
  (schema dump generated from `OJ` migrations) and asserts:
  - the app starts and serves,
  - **no DDL statement is ever emitted** (assert via SQL logger hook),
  - every GORM model can `SELECT` one row / `LIMIT 1` without unknown-column errors.

---

## Section 2 — Roles & permissions: Django-native adapter

**Rule: the Go backend enforces the same decisions the Django views make,
using Django's own auth tables and codenames.**

### Data model (new GORM models, read/write rows only)

| Go model | Table |
|---|---|
| `AuthGroup` | `auth_group` |
| `AuthPermission` | `auth_permission` (+ `django_content_type` for app labels) |
| `AuthUserGroup` | `auth_user_groups` |
| `AuthGroupPermission` | `auth_group_permissions` |
| `AuthUserPermission` | `auth_user_user_permissions` |

`AuthUser` (existing) keeps `is_staff`, `is_superuser`, `is_active`.

### Permission resolution

Effective permission check `Has(user, "app_label.codename")`:

1. `is_active == false` → deny all.
2. `is_superuser` → allow all.
3. Else: allow iff codename ∈ (user permissions ∪ permissions of user's groups),
   resolved by joining the tables above.

- Cached per-user in Redis (`perm:v{N}:{user_id}` → set of codenames, TTL 60s).
- When OJ-v2's admin UI modifies groups/permissions, it bumps the cache version
  key so changes apply immediately on OJ-v2; changes made from Django admin
  propagate within the TTL.
- JWT claims carry only `user_id`, `is_staff`, `is_superuser` — never the
  permission list (stale-claims risk).

### Codename mapping (v2 constant → Django behavior)

The mapping below replaces `auth/permissions.go`. Where Django distinguishes
scopes that v2 collapsed, the **Django semantics win**. Entries marked
*(verify)* must be confirmed against the actual `OJ` view code during
implementation — the Django view behavior is the source of truth, not this table.

| v2 code | Django check |
|---|---|
| `problems.create` | `judge.add_problem` |
| `problems.edit` | editability: author/curator + `judge.edit_own_problem`, OR `judge.edit_all_problem`, OR `judge.edit_public_problem` (public problems only) — mirror `Problem.is_editable_by` |
| `problems.delete` | `judge.delete_problem` *(verify)* |
| `problems.view_hidden` | `judge.see_private_problem` |
| `problems.edit_data` | same gate as problem editability (mirrors Django `ProblemData` views) |
| `contests.create` | `judge.add_contest` *(verify — plus `judge.create_private_contest` where relevant)* |
| `contests.edit` | author/curator + `judge.edit_own_contest`, OR `judge.edit_all_contest` — mirror `Contest.is_editable_by` |
| `contests.delete` | `judge.delete_contest` *(verify)* |
| `contests.view_hidden` | `judge.see_private_contest` |
| `contests.manage_problems` | same gate as contest editability |
| `submissions.rejudge` | `judge.rejudge_submission` (bulk: `judge.rejudge_submission_lot`) |
| `submissions.view_all` | `judge.view_all_submission` |
| `submissions.contest_access` | **removed as a permission** — contest submit rights are participation-based logic, as in Django |
| `users.ban` / `users.edit` / `users.delete` | `auth.change_user` / `auth.delete_user` *(verify how CLAOJ gates these views)* |
| `users.view_email` | `is_staff` *(verify)* |
| `organizations.*` | `judge.organization_admin`, `judge.edit_all_organization`, built-in add/delete *(verify)* |
| `comments.edit` / `comments.delete` / `comments.pin` | `judge.change_comment` / `judge.delete_comment` / comment-lock permission from CLAOJ `comment.py` Meta *(verify)* |
| `tickets.*` | CLAOJ ticket view gates *(verify)* |
| `blogs.edit` / `blogs.delete` | `judge.edit_all_post` / `judge.delete_blogpost` *(verify)* |
| `system.admin_panel` | `is_staff` |
| `system.moss` | `judge.moss_contest` |
| `system.stats` | `is_staff` *(verify)* |
| `system.manage_judges` | `judge.change_judge` |
| `system.manage_languages` | `judge.change_language` |
| `system.manage_announcements` | blog-post permissions *(verify)* |
| `problems.suggest` / `problems.manage_suggestions` | feature dropped (Section 3); Django's `judge.suggest_new_problem` remains available for future use |

Implementation note: enumerate every call site of the old constants
(`grep -r "Perm[A-Z]" claoj/`) and rewrite each against the mapped Django check.
The old `auth/permissions.go` constant file and `DefaultPermissionSets` are deleted
(no seeding — groups are whatever the production DB says they are).

### Admin UI: role manager → Django group manager

- Backend endpoints (`admin_users_roles.go` reworked): list/create/rename/delete
  groups; list all permissions (from `auth_permission`, grouped by content type);
  assign/remove permissions to a group; assign/remove users to a group; set
  `is_staff` / `is_superuser` / `is_active` on users (superuser-only operations).
- Frontend admin pages switch vocabulary from "roles" to "groups"; permission
  picker shows Django permission names (human-readable `name` column) grouped by
  app/model.
- All of this is row-level writes to Django's own tables — exactly what Django
  admin itself does; the Django site picks changes up instantly.

### Verification

- Unit tests: permission resolver against fixture rows (superuser, staff,
  group perms, direct user perms, inactive user).
- Parity test matrix: for each mapped endpoint, a table-driven test asserting
  the Go decision matches the documented Django gate.

---

## Section 3 — Infra re-homed to Redis; v2-only features dropped

### Refresh tokens → Redis

- Key: `rt:{sha256(token)}` → JSON {user_id, family_id, expires_at, revoked,
  user_agent, client_ip}, TTL = token expiry.
- Family index `rtfam:{family_id}` → set of token hashes (for rotation-reuse
  detection: on reuse of a rotated token, revoke the whole family — preserving
  the current design's semantics).
- The MySQL `refresh_token` repository is deleted; interface stays so handlers
  don't change shape.
- Trade-off accepted: a Redis flush logs everyone out (they just log in again).

### Audit log → Redis

- Redis Stream `audit:log` (XADD with MAXLEN ~100k) storing the same fields the
  MySQL table had; admin viewing API reads the stream (XRANGE, newest first,
  filter in code). No MySQL table.

### Dropped features (schema-dependent, not in Django)

- **Per-contest `max_submissions`**: remove the model field, API fields,
  enforcement logic, and admin/frontend inputs.
- **Problem suggestion workflow** (`suggestion_status`, `suggestion_notes`,
  `suggestion_reviewed_*`): remove Go endpoints, model fields, and the
  frontend pages/flows that use them.

---

## Section 4 — Full Vietnamese support

### Font (the rendering bug)

- Replace `Outfit` with **Be Vietnam Pro** in
  `claoj-web/src/app/[locale]/layout.tsx` via `next/font/google`:
  `subsets: ["latin", "vietnamese"]`, weights as needed (400/500/600/700),
  CSS variable renamed accordingly and Tailwind config updated.
- Sweep for any other font surfaces: `font-family` in CSS, Tailwind `fontFamily`
  config, Monaco editor font (code area uses **JetBrains Mono**, which has
  Vietnamese glyph coverage for comments/strings in code), KaTeX (math-only, unaffected), PDF/OG-image generation if any.
- No surface may load a font without Vietnamese coverage for UI text.

### Translation audit

- Sweep `claoj-web/src` for hardcoded user-visible English strings (JSX text,
  `placeholder=`, `title=`, `aria-label=`, toast messages, status label maps like
  `'IE': 'Internal Error'`, admin pages) and move them into `next-intl` messages.
- `en.json` and `vi.json` stay key-complete (same key set); Vietnamese
  translations written for every key.
- Locale routing, `<html lang={locale}>`, and date formatting (dayjs locale)
  verified for vi.

### Verification

- Visual check: page with the glyph test string `ạ ế ở ữ đ Ị Ổ ẵ Ứ` renders
  entirely in Be Vietnam Pro (no fallback), light + dark themes.
- CI-able check: script asserting `en.json` and `vi.json` have identical key sets.
- Manual pass over main pages in `vi` locale: no English leakage, no broken glyphs.

---

## Rollout

1. Land Sections 1–3 (backend parity) behind normal review; deploy OJ-v2 against
   a **copy** of the production DB first, run the no-DDL integration test, then
   point at the live DB.
2. Section 4 (frontend) is independent and can ship any time.
3. Optional: run `scripts/cleanup_v2_tables.sql` manually on databases that had
   the old v2 migrations applied.

## Risks

- **Mapping mistakes** could grant/deny incorrectly → mitigated by the *(verify)*
  pass against Django view code + parity test matrix; deny-by-default on any
  unmapped check.
- **Redis durability** for refresh tokens/audit log is weaker than MySQL →
  accepted trade-off (re-login; capped audit history). Enable Redis persistence
  (AOF) in deployment if history matters.
- **Existing v2 deployments** already using judge_role data: group memberships
  must be recreated as Django groups by an admin (one-time manual step; the old
  tables remain readable until cleanup).
