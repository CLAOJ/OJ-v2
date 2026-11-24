# Task 1 - Organizations: report

Worktree `OJ-v2-wt/feat/admin-ux-phase4b`. Original task report was not present in the
worktree, so this file records the review fix pass; the implemented scope is the Task 1
section of `2026-11-25-admin-ux-phase4b.md`.

## Delivered

- `GET /admin/organization/:id`: flat detail with about, slots, access_code,
  logo_override_image, creation_date, member_count, performance_points, admins
  `[{id, username}]`.
- Create/update accept `logo_override_image` and `admins` (profile ids, validated,
  replace m2m); `creation_date` is never accepted.
- `judge.organization_admin` gate on create and on changed `is_open`/`slots`/`admins`
  (echoed values pass).
- `POST /admin/organization/:id/recalculate` returns the new points.
- Unified create/edit modal with all fields, admins EntityPicker, Recalculate, View on
  site, i18n en+vi.

## Review fix (2026-11-25, HEAD before fix f17e7e7)

1. **Response shape.** POST and PATCH returned `{success, organization: <Go struct>}`
   with PascalCase keys, no json tags and a `MemberIDs` list of every member profile.
   All three handlers now share `organizationDetailBody`, so create/update return the
   same flat body as GET, and the frontend `create`/`update` types are
   `AdminOrganizationDetail`.
2. **Save-path cost.** `GetOrganization` no longer preloads `Members`; `member_count`
   comes from the row Django maintains. The dead `MemberIDs` field and its computation
   were removed.
3. **Regression tests.**
   - `TestAdminOrganizationUpdate_OmittingAdminsKeepsMembership`: 2 seeded admins survive
     a name-only patch, and the response carries both.
   - `TestAdminOrganizationCreate_RestrictedFieldsGate`: non-`judge.organization_admin`
     POST with `is_open`/`slots`/`admins` is 403 and inserts nothing; a plain create by
     the same user succeeds.
   - Explicit `admins: []` clearing was already covered by
     `TestAdminOrganizationCreateUpdate_RoundTrip`.
4. **Modal error state.** A failed detail fetch now shows `organizations.loadFailed`
   instead of an empty live form, so Delete/Save are never offered for data that never
   loaded.
5. **Slots clearing (optional, done).** `PATCH {"clear_slots": true}` sets `slots` to
   NULL; it is gated like any other slots change. The modal sends it when the slots
   input is emptied.

## Verification

- `go build ./...` - ok
- `go test ./api/v2/ -count=1` - ok
- `npx tsc --noEmit` - ok
- `npm test -- --silent` - 25 suites / 272 tests passed
- `npx eslint` on the touched frontend files - only pre-existing warnings in
  `adminApi.ts` unused imports and pre-existing `no-empty-object-type` errors in
  `types/index.ts` (file not modified by this fix).