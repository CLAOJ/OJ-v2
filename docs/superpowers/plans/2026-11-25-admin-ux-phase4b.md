# Phase 4b — Organizations, Blog, Tickets, Taxonomy, Comments, Licenses

**Spec:** `docs/admin-ux-gap-report.md` §3.1 + Roadmap Phase 4 items 2–6.
**Repo:** worktree `OJ-v2-wt/feat/admin-ux-phase4b` (branch `feat/admin-ux-phase4b`, from main 8938ce0).
**Stack:** Next.js 16 + Go/Gin, shared MySQL (Django schema), no migrations.

## Global constraints
- Keep v2 design; minimal diffs; no new deps; no migrations; i18n en+vi; tests for new behavior; commit per task; no push.
- Secrets never in admin JSON. Bulk destructive actions need explicit confirmation.
- Reuse `components/admin/EntityPicker.tsx` for all multi-selects; reuse existing list endpoints for search.

## Rulings (from recon of v1 code)
1. **Org 2FA gate on create is NOT a v1 feature.** v1 has a global 2FA middleware for 2FA-enabled users; there is no org-specific gate. Do not add one. (Gap report claim corrected.)
2. **License file upload is NOT a v1 feature.** v1 `LicenseAdmin` is a plain text field rendered with the Martor markdown editor; the upload is the *editor's* image upload, not a license file. Implement shared markdown image upload only.
3. **Org member management is not in v1 admin.** v1 `OrganizationAdmin` exposes name/slug/short/about/is_open/is_unlisted/logo/slots/creation_date/admins; member count is read-only. Keep member count read-only; do not build a member manager.
4. **ProblemGroup "included problems" is a v1 bug.** `Problem.group` is a FK (one group per problem); `taxon.py` calls `obj.problem_set.set(...)`, which is not valid for a reverse FK. Implement included-problems for **ProblemType** (real M2M) and included-contests for **ContestTag** (real M2M); for ProblemGroup show a read-only count + link to the filtered problem list.
5. `creation_date` is `auto_now_add` → read-only everywhere (API returns, update ignores/rejects).
6. Ticket `time` stays read-only (v1 `readonly_fields=('time',)`); title and creator are editable.

## Task 1 — Organizations
Backend (`admin_judge_organization.go`, `service/organization`):
1. `GET /admin/organization/:id`: all existing list fields + about, slots, access_code, logo_override_image, creation_date, performance_points, admins `[{id, username}]`, member_count. Gate: staff read.
2. Create/update accept `logo_override_image`, `admins` (profile ids, replace m2m `judge_organization_admins`, validate ids). `creation_date` never accepted.
3. Field gate: changing `admins` / `is_open` / `slots` requires `judge.organization_admin` (v1 makes them readonly otherwise) → 403 on change, unchanged values pass.
4. `POST /admin/organization/:id/recalculate`: recompute `performance_points` via existing `scoring.CalculateOrganizationPoints`, return new points. Gate `RequireOrgEdit`.
5. Tests: detail fields, create/update round-trip incl. logo + admins, creation_date ignored, field gate 403 for non-`organization_admin`, recalc updates points, 404s.

Frontend (`admin/organizations/page.tsx`, `OrganizationEditModal.tsx`, `lib/adminApi.ts`, types):
6. One unified create/edit modal with all fields (name, slug, short_name, about, is_open, is_unlisted, slots, access_code, logo_override_image, admins EntityPicker over `/admin/users`); edit also shows creation_date read-only, member_count/performance_points, Recalculate button, View on site link. Fields 3 gates disabled with a hint when the viewer lacks `judge.organization_admin` (viewer perms from `/me`).
7. List cards keep current look; add logo/PP if cheap. i18n en+vi.

## Task 2 — Blog posts + markdown image upload (shared with Licenses)
Backend:
1. Blog list: populate `author_names` + `organization_name` (fix the declared-but-empty DTO); add `search` (title/slug) and `visible` filter + filtered total.
2. New `POST /admin/markdown/image` (staff): multipart `image`, allow png/jpg/jpeg/gif/webp ≤ 2 MB, save under the existing data dir (`data/media/<random>.<ext>`), return `{url}`. Add the static serving route if the data dir is not already served (check first; do not invent a new storage system). Tests: type/size rejection, happy path URL.
3. License: no backend change (text already stored).
Frontend:
4. Blog list: Authors column from API, Organization column, search box actually sent, status filter (visible/all/hidden). Detail/edit form gains Authors EntityPicker (`/admin/users`) + Organization select (`/admin/organizations`), image-upload button next to content/summary inserting markdown at cursor.
5. License dialog: same image-upload button on `text`; no file field (ruling 2).
6. Shared small component (`components/admin/MarkdownImageButton.tsx`) so blog + license reuse it. i18n.

## Task 3 — Tickets
Backend (`ticket.go`):
1. Admin detail returns linked item: `linked_type` (`problem`|`contest`|`""`), `linked_id`, `linked_label` (code + name / key + name), resolved by content-type model (`judge.problem`/`judge.contest`), not by hardcoded ids.
2. `PATCH /admin/ticket/:id` (gated `judge.change_ticket`): `title`, `user_id` (profile id → auth_user id), `linked_type`+`linked_id` (validated both-or-neither; empty clears both). `time` unchanged.
3. `PATCH /admin/ticket/:id/message/:messageId` (body, user_id) and `DELETE .../message/:messageId`, same gate as reply (`judge.change_ticket` cross-user).
4. Admin list `total` must honour the filters (currently ignores them).
5. Tests: linked item resolution + set/clear, title/creator patch, message edit/delete + 404s/403s, filtered total.
Frontend:
6. Ticket detail: title and creator editable, linked-item row with type select + EntityPicker (problems/contests search) + public link; message bubbles get edit (inline) and delete; filtered total shown. i18n.

## Task 4 — Taxonomy / tags included lists
Backend:
1. ProblemType create/update accept `problem_ids` (replace `judge_problem_types` association), detail returns included ids, list returns `problem_count`. Validate ids.
2. ContestTag create/update accept `contest_ids` (replace rows in `judge_contest_tags` for this tag only), detail returns included ids, list returns `contest_count`.
3. ProblemGroup detail returns `problem_count` (read-only; ruling 4).
4. List endpoints get `search` (name/full_name; tags already have it).
5. Tests: replace semantics (add+remove), count, invalid ids, group read-only.
Frontend (`admin/taxonomy/page.tsx`, `contests/tags/page.tsx`):
6. Search box for groups/types; type dialog gains Included problems EntityPicker (`/admin/problems`); group dialog shows read-only count + link to `/admin/problems?group_id=`; tag dialog gains Included contests EntityPicker (`/admin/contests`). i18n.

## Task 5 — Comments
Backend (`models/comment.go`, `admin_comment.go`):
1. Map `Revisions` (`judge_comment.revisions`) and return it.
2. List: actually apply `page_type` filter (`page LIKE 'problem:%'` etc.); keep filtered total.
3. PATCH accepts `author_id`, `page`, `parent_id` in addition to body/hidden/reason; validate author/parent; `time`/`score` remain read-only.
4. `POST /admin/comments/bulk-hide` `{ids[], hidden}` (gated `judge.change_comment`), audit-logged.
5. Tests: metadata patch, page_type filter, bulk hide/unhide, revisions round-trip.
Frontend (`admin/comments/page.tsx`):
6. Compact rows (truncate body, expand on click), row checkboxes + bulk hide/unhide bar with confirmation, dialog gains author (EntityPicker, single), page, parent id; hide/unhide per row kept. i18n.

## Acceptance
- Org admin can create/edit everything v1 can (logo string, slots, admins, creation date read-only) and recalculate scores; blog lists authors/org and filters; tickets show/edit linked item and messages; taxonomy/tags manage included problems/contests; comments edit metadata + bulk hide; licenses/blog can upload markdown images.
- Suites green (`go test ./...`, jest, tsc, lint baseline unchanged); no migrations; no push.

## Explicitly out of scope (Phase 4c/5)
Submissions admin detail/edit, impersonation, registration activate/resend, judges, navbar/languages, flat pages/redirects, public org/blog/contest pages, bulk destructive actions, org member management, class admin.