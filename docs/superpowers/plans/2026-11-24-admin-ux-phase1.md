# Phase 1 — Problem form parity (v1 fields in v2)

**Spec:** `docs/admin-ux-gap-report.md` §1 + Roadmap Phase 1.
**Repo:** worktree `OJ-v2-wt/feat/admin-ux-phase1` (branch `feat/admin-ux-phase1`, from main 1ec0db2).
**Stack:** web = Next.js 16 (`claoj-web`), API = Go/Gin (`claoj`). Shared MySQL with v1 — **no migrations**; every column already exists (schema-audit: zero drift).

Current state: `ProblemProfile` (GET detail) already includes source, short_circuit, license_id, og_image, summary, is_full_markup, both visibility modes, suggester_id, curators, testers, organizations, banned users. Missing: create/update service+handler fields for most of them, and the whole form surface. Update already accepts curators/testers via `*[]uint`.

## Global constraints
- Keep v2 design. Minimal diffs. No new dependencies. No migrations.
- All user-facing strings via next-intl (`src/i18n/en.json` + `vi.json`).
- No unrelated refactors. Commit per task, conventional messages. No push/deploy.
- Tests: Go handler/service tests for API changes; jest for new web logic; run `go build ./...`, targeted `go test`, `tsc --noEmit`, `npm run lint` (report pre-existing only).

## Task 1 — Backend: problem create/update fields (Go)
Goal: the admin API accepts and persists every v1 problem field the model already maps.

1. `service/problem/problem_service.go`:
   - `CreateProblemRequest`: add `Source string`, `Date *time.Time`, `LicenseID *uint`, `OgImage string`, `Summary string`, `ShortCircuit bool`, `IsFullMarkup bool`, `SubmissionSourceVisibilityMode string`, `TestcaseVisibilityMode string`, `IsOrganizationPrivate bool`, `OrganizationIDs []uint`, `BannedUserIDs []uint`, `SuggesterID *uint`, `CuratorIDs []uint`, `TesterIDs []uint`.
   - `UpdateProblemRequest`: add pointer forms for the same scalar fields plus `OrganizationIDs`, `BannedUserIDs`, `SuggesterID`, and `IsOrganizationPrivate`; relations sync via `relations.Sync` (mirror Curators/Testers).
   - Apply values in create/update; sync Organizations/BannedUsers associations. Validate visibility modes against v1's allowed values (inspect `OJ/judge/models/problem.py` choices: submission source + testcase visibility — replicate the same single-char codes and reject unknown values with a 400).
   - `problemToProfile` already returns these; no response change needed except confirm banned users/organizations are serialized for the edit form (add to profile if absent).
2. `api/v2/admin_problem.go`: extend create/update request structs and pass-throughs. Create must accept curators/testers too.
3. Tests: service test that creates a problem with all new fields and reads them back; update test for scalar changes + org/banned sync; invalid visibility mode → error/400.

## Task 2 — Frontend: full problem form (web)
1. `ProblemFormData` + `components/admin/ProblemForm.tsx`:
   - **People**: suggester + curators + testers pickers (reuse `EntityPicker`, as AuthorsSection does); keep authors.
   - **Publishing**: date of publishing (datetime-local → ISO/null), license select (admin license API), problem source (text), submission source visibility (select), testcase visibility (select), allow full markdown (checkbox).
   - **Access**: private-to-organizations checkbox + organization multi-picker, banned users picker.
   - **Scoring/limits**: short circuit checkbox; keep points/partial/time/memory; add a link to the problem's Language Limits (`/admin/language-limits` filtered by problem if supported, else the page).
   - **Social**: og_image URL + summary textarea.
   - Types in `src/types/index.ts` and `src/lib/adminApi.ts` updated to match the extended DTO.
2. `admin/problems/[code]/edit/page.tsx` (+ create page): map the new problem detail fields into `initialData` (including curators/testers/suggester/orgs/banned from the profile shape).
3. i18n keys in en+vi. Keep the existing 4-section layout pattern; add sections rather than restyle.
4. Tests: any new pure mapping helpers get jest tests; no component tests for ESM-only renderers.

## Task 3 — Problem translations admin (Go + web)
1. API: mirror `api/v2/problem_clarification.go` for `judge_problemtranslation` (model `ProblemTranslation`: problem_id, language, name, description): admin list/create/update/delete under `/admin/problem/:code/translations` (respect existing route/auth patterns).
2. Web: new page `admin/problems/[code]/translations/page.tsx` in the style of the clarifications page; link it from the problem edit tab row (next to Manage Data / Clarifications / Editorial); i18n en+vi.
3. Tests: Go handler test for create/list round-trip; jest for any helper.

## Acceptance
- Create a problem in v2 admin filling every v1 field; read it back in the v2 edit form and confirm values persist (and are visible in v1 admin where applicable).
- Language-specific limits: reachable from the problem edit page and pre-filtered to the problem.
- Translations: create/edit/delete a translation and see it persisted (model table unchanged).
- All suites green; lint baseline unchanged.
## Task 0 — MOSS page: visible contest list (web, owner request)
/admin/moss currently shows only a floating autocomplete for contests. Add a visible contest list on the page:
- Search input + list of contests from the existing dminContestApi.list(page, pageSize, { search }) (no new endpoint), showing name, key and start date, with status/ended indication; default view = first 20 contests, newest first as the API returns them.
- Clicking a row selects that contest (same selection the autocomplete does), highlights the selected row, and keeps the existing results/run/clear section below unchanged.
- Status filter (ongoing/upcoming/past) is optional if cheap; the list itself is the requirement.
- Keep v2 design; i18n keys en+vi; tests only for new pure helpers; commit.
