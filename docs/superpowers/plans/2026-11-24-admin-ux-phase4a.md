# Phase 4a — Users & Groups admin parity

**Spec:** `docs/admin-ux-gap-report.md` §3.1 (Users, Groups) + Roadmap Phase 4 items 1.
**Repo:** worktree `OJ-v2-wt/feat/admin-ux-phase4a` (branch `feat/admin-ux-phase4a`, from main d42907e).
**Stack:** Next.js 16 + Go/Gin, shared MySQL (Django `auth_user`, `auth_user_groups`, `auth_user_user_permissions`, `judge_profile`), no migrations.

Current state: v2 can create users, set password, ban/status, delete, and add/remove a group via `POST /admin/user/:id/groups` + `DELETE .../:groupId`. No identity edit, no staff/superuser flags, no group/permission assignment on the user, no profile editing. Profile model already maps: About, Timezone, LanguageID, AceTheme, SiteTheme, DisplayRank, Mute, IsUnlisted, BanReason, UserScript, UsernameDisplayOverride, Notes, TOTP/WebAuthn flags. Group membership has no management UI.

## Global constraints
- Keep v2 design; minimal diffs; no new deps; no migrations; i18n en+vi; tests for new behavior; commit per task; no push/deploy.
- Security: never expose password hashes, TOTP secrets, scratch codes, API tokens or internal encrypted blobs in API responses; JSON admin responses are for staff only.

## Task 1 — Backend: user edit, groups, filters
1. `GET /admin/user/:id` (detail): identity (id, username, email, first/last name, is_active/staff/superuser, is_superuser, date_joined, last_login), profile fields (about, timezone, language_id, ace_theme, site_theme, display_rank, mute, is_unlisted, ban_reason, username_display_override, notes, user_script), group ids/names, user permission ids/codenames, security status (is_totp_enabled, is_webauthn_enabled, webauthn credential count). NO secrets.
2. `PATCH /admin/user/:id`: username (validate uniqueness, Django-compatible charset), email, first/last name, is_active, is_staff, is_superuser, and the profile fields above except computed/secure ones. Prevent self-demotion/self-deactivation hazards: cannot set is_staff/is_superuser false on yourself, cannot deactivate yourself (return 400).
3. `PUT /admin/user/:id/groups`: replace full list (validate ids exist). `PUT /admin/user/:id/permissions`: replace direct user_permissions.
4. Security admin: `POST /admin/user/:id/security` with `{totp_enabled: bool, webauthn_enabled: bool}` — disabling clears the encrypted TOTP key/scratch codes and deletes WebAuthn credentials for the user (v1 admin can disable 2FA); enabling is NOT allowed here (must go through the user's own setup flow) → 400.
5. Users list: filters `staff`, `superuser`, `active`, `group_id`; `sort` (username, date_joined, last_login, points) + order; include profile points/problem_count/rating and group names (avoid N+1).
6. Tests for each endpoint (happy path, validation, self-demotion guard, group replace, secrets absent from JSON, filters/sort).

## Task 2 — Frontend: user management UI
1. Replace the single "Manage User" modal with a tabbed page/modal: Identity (username/email/names/active/staff/superuser), Groups & Permissions (group multi-select + direct permissions search), Profile (about, timezone, language, themes, display rank, mute, unlisted, notes, display-name override, ban reason, user script), Security (2FA/WebAuthn status + disable).
2. Users list: filters (role/status/group) + sortable headers per Task 1; keep ban/set-password/delete actions.
3. Editing yourself: disable the staff/superuser/active controls per backend guard with an explanatory hint.
4. i18n en+vi; jest tests for new pure mapping/validation helpers.

## Task 3 — Groups membership
1. Backend: `GET /admin/group/:id/members` (paginated, search) and `PUT /admin/group/:id/members` (replace) or add/remove endpoints; guard by the same admin middleware; tests.
2. Frontend: group edit modal gains a Members section (search users, add/remove, count refresh); group cards' member counts stay accurate.
3. i18n.

## Acceptance
- Admin can fully manage a user (identity, roles, groups, permissions, profile, 2FA disable) and group membership without v1.
- No secret field ever appears in any admin JSON (test-enforced); self-demotion guarded.
- Suites green; lint baseline unchanged.