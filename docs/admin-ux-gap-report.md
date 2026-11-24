# CLAOJ v2 UX Gap Report & Roadmap

**Date:** 2026-11-24
**Method:** live side-by-side comparison of v1 (`claoj.edu.vn`, Django/DMOJ) and v2 (`beta.claoj.edu.vn`, Next.js + Go), logged in as superuser `bachtam2001`. Admin forms, list pages and flows inspected in the browser; gaps cross-checked against the Go API (`claoj/api/v2/admin_*.go`) and GORM models (`claoj/models/*.go`). Normal-user (public) pages audited in parallel.
**Constraint from owner:** keep v2's visual design. Fix UX only: missing fields, missing flow steps, list-page usability.

---

## Executive summary

v2's design is fine; its **admin is a subset of v1's functionality** and its **public pages have a handful of live data bugs**. Most admin gaps are cheap: the Go models already map the shared Django schema, so fixes are API DTO + form work, not migrations.

**Live bugs to fix first (Phase 0):**
1. Problems >50 points unfindable by normal users (`points_max=50` default + capped slider) — 19 problems hidden.
2. Submission memory shown ×1000 wrong (KB labeled MB).
3. Contest descriptions render as placeholder; announcements missing; "View all past contests" dead.
4. Language Limits admin page crashes; Audit Log records 0 entries.
5. Raw HTML leaks in problem statements.

**Biggest functional gaps:** admin Users (no identity/permission/profile editing, no group membership anywhere), no submission admin detail, no impersonation, contest/problem create forms missing ~15 fields each, no bulk actions/sorting/weak pagination on every admin list. Public: no group/type filters on problems, no org problem list, scoreboard/profile feature cuts.

**Roadmap:** Phase 0 bugs+quick wins → 1 problem form parity → 2 contest form parity → 3 list UX → 4 remaining admin sections → 5 public-page gaps. Details below.

---

## 0. Architecture context (why most gaps are cheap)

- v2 = Next.js 16 frontend (`claoj-web`, react-hook-form + zod + TanStack Query) + Go/Gin backend (`claoj`).
- **v2 reads the same MySQL schema as v1** (`docs/schema-audit.md`: zero drift, Django models are the source of truth). Every v1 admin field already exists as a DB column.
- The Go models already map almost all of them (`Problem`: suggester, curators, testers, banned users, license, date, og_image, summary, short_circuit, visibility modes, org-private, per-language limits, translations, solution; `Contest`: rating floor/ceiling, rate_all, scoreboard visibility/cache, precision, push announcements, hide authors, logo override, label script, locked_after, private contestants, orgs, banned users, announcements).
- **The gap is in three layers, in increasing cost:**
  - **FE** — API already accepts/returns the field; only the React form/list is missing it.
  - **API** — Go model has the column; the admin request/response DTOs and handlers don't expose it. Add DTO fields + assignment + form controls. No migration.
  - **MAP** — column exists in the shared DB but is not mapped in the Go model at all (a few contest access/rating fields). Add gorm field + API + FE. Still no migration.
  - **NEW** — genuinely new behavior (bulk-action endpoints, announcement resend).

---

## 1. Admin — Problems

### 1.1 Create & edit form (edit page has the *same* fields as create)

v2 today: code, name, description, points, time/memory limit, group, types, authors, allowed languages, public, partial, manually managed, PDF URL.

| Missing vs v1 | Go model | Admin API | Tier |
|---|---|---|---|
| Suggester | `SuggesterID` ✓ | ✗ | API |
| Date of publishing | `Date` ✓ | ✗ | API |
| Curators, Testers | ✓ | update ✓ (`curator_ids`,`tester_ids`), create ✗, UI ✗ | FE + API(create) |
| License | `LicenseID` ✓ | ✗ | API |
| Problem source | `Source` ✓ | ✗ | API |
| OpenGraph image, Problem summary | `OgImage`,`Summary` ✓ | ✗ | API |
| Short circuit | `ShortCircuit` ✓ | ✗ | API |
| Allow full markdown | `IsFullMarkup` ✓ | ✗ | API |
| Submission source visibility | ✓ | ✗ | API |
| Testcase visibility | ✓ | ✗ | API |
| Private to organizations + org picker | `IsOrganizationPrivate`,`Organizations` ✓ | ✗ | API |
| Banned users (personae non gratae) | `BannedUsers` ✓ | ✗ | API |
| Per-language time/memory limits | `ProblemLanguageLimit` ✓ | check admin CRUD | API/FE |
| Problem translations | `ProblemTranslation` ✓ | no admin UI found | API + FE page |
| Edit reason | v2 has Audit Log page | verify it records admin edits | FE/verify |

### 1.2 Flow gaps

- **Editorial page exists** (`/admin/problems/[code]/editorial`) but the edit page only links **Manage Data** and **Clarifications** — editorial is orphaned. v1 has solution inline on the problem form. → FE: add link/tab (and surface solution visibility/publish date/authors there).
- Clarifications page exists and is linked ✓ (v1 parity OK), but not reachable from create (v1 allows inline on add) — minor.
- Test cases: v2 Data page (single + ZIP batch, files, configuration, PDF) ≈ v1 flow ✓.

### 1.3 List page (`/admin/problems`)

| v1 feature | v2 status | Tier |
|---|---|---|
| ~20 dense rows/screen | ~6 tall rows | FE |
| Checkboxes + bulk actions + "n of N selected" | none | NEW (bulk endpoints) + FE |
| Sortable columns (code, name…) | none | API (sort param) + FE |
| Filters: visibility ✓, group ✓, creator ✗, year ✗ | partial | API + FE |
| Search | ✓ | — |
| Numbered pagination + total count | API returns `total/page/page_size`; UI doesn't show it | FE |
| Authors column | not in list DTO | API + FE |
| "View on site" per row | none | FE |
| Row actions labeled | icon-only (edit/data/clone/delete) | FE (labels/tooltips) |

---

## 2. Admin — Contests

### 2.1 Create form

v2 today: key, name, description, summary, start/end, time window, format, access code, visible, rated, hide tags, pretests only, org-private toggle, problems, tags, authors/curators/testers.

| Missing vs v1 | Go model | Admin API | Tier |
|---|---|---|---|
| Contest lock (`locked_after`) | ✓ | separate endpoint ✓; **edit form only**, create ✗ | FE |
| Format configuration | ✓ | create+update ✓ (`format_config`), UI ✗ | FE |
| Problem label script | ✓ | ✗ | API |
| Rating floor / ceiling / rate all | ✓ | ✗ (checking "Rated" reveals nothing) | API |
| Scoreboard visibility / cache timeout / points precision | ✓ | ✗ | API |
| Push announcements / hide problem authors / show short display | ✓ | ✗ | API |
| "No comments" / comment control | `UseClarifications` ✓ (verify exact v1 mapping) | ✗ | API |
| OpenGraph image / logo override | ✓ | ✗ | API |
| Private contestants (user picker) | ✓ | ✗ | API |
| Organizations picker (toggle reveals nothing) | ✓ | ✗ | API |
| Banned users | ✓ | ✗ | API |
| Classes, join-organizations, view-scoreboard / see-submissions permissions | **not mapped** | ✗ | MAP |
| Exclude from ratings / performance ceiling override | **not mapped** (check Django rating fields) | ✗ | MAP |
| Announcements: create + resend | `ContestAnnouncement` ✓ | no admin endpoints found | NEW + FE |

### 2.2 Flow gaps

- Edit page links **Participations** only; clarifications page exists but is unlinked; announcements have no admin surface at all.
- "Private to organization" toggle renders no organization picker (dead toggle).
- "Rated contest" toggle renders no rating parameters (dead toggle).

### 2.3 List page

Same gaps as problems: no bulk actions, no sorting, no pagination/total UI, icon-only actions, tall rows. Status filter ✓, search ✓.

### 2.4 Deferred in Phase 2

- **Public contest-page rendering of announcements** and **push-on-create / resend** need the notification pipeline and the public consumption path; both land in Phase 5. Phase 2 ships the admin CRUD surface only.
- **Contest access/rating field authz is still v2-wide coarse**: `is_private`, `private_contestants`, `organizations`, `rating_floor`/`rating_ceiling`/`rate_all` remain behind the pre-existing `RequireContestEdit` gate rather than v1's `create_private_contest` / `contest_rating` permissions. The two v1-gated fields that would otherwise be newly writable with no check — `problem_label_script` (`judge.contest_problem_label`) and `performance_ceiling_override` (`judge.override_performance_ceiling`) — are now checked on create and update.

---

## 3. Admin — remaining sections

### 3.1 Content & Community (audited)

**Users — biggest gap in the whole admin.** v2 can only create, set password, ban/status (Activate/Deactivate/Hide/Unhide + ban reason), delete. Missing vs v1 (`auth/user` + `judge/profile`, both fully mapped in Go per schema-audit → **API+FE**):
- Edit username / email / full name after creation (create-only today).
- Staff / superuser flags, groups assignment, per-user permissions (Role column read-only).
- Entire profile surface: organization, display rank, self-description, location, preferred language, ace theme, comment mute, unlisted, internal notes, display-name override, current contest, user script.
- TOTP 2FA / scratch codes / WebAuthn credential management; last login / date joined view.
- List: no staff/superuser/active/groups filters; 5,694 rows at 50/page with Prev/Next only.

**Groups.** Permission editor is *better* than v1 (grouped checkboxes vs flat 291-select). But **membership can't be managed anywhere** — v1 does it via the user form, which v2 lacks. → API+FE (m2m tables mapped).

**Organizations.** Missing: logo override, max size, creation date, administrators M2M, member management (count is read-only), Recalculate scores action, View-on-site. Create modal (Name/Slug/Short Name) is poorer than edit (adds About/Open/Unlisted) — create→edit inconsistency. **Security:** v1 has no org-specific 2FA gate (only the global middleware), so v2 needs none. → API+FE (model mapped).

**Blog posts.** Missing: Authors + Organization fields (Authors column shows "-" for all 68 posts), markdown image upload, status filter. Otherwise good parity + quick Show/Sticky toggles. → API+FE (`BlogPost` in `models/interface.go`).

**Tickets.** v2 has better filters/reply/internal notes/assignees, and Phase 4b closed the gaps: the linked problem/contest is shown and editable, messages can be edited/deleted, and title/creator are editable with creation time read-only.

**Taxonomy & Contest Tags.** "Included problems/contests" M2M relations missing on types/groups/tags; taxonomy has no search across 77 types (4 blind page flips). → API+FE.

**Licenses.** Parity except v1 uploads license text as a file, v2 requires pasting. → FE (+upload endpoint).

**Comments.** v2 adds type filter + edit-reason audit, but can't edit author/page/parent/time/votes; no bulk hide/unhide; full body in rows makes them very tall. → API+FE.

---

### 3.2 Judging & System (audited)

**🐛 Language Limits page is BROKEN (P0).** Crashes to the error boundary on ~4 of 5 loads; otherwise stuck at "Loading… 0 of 0" while the API returns 200 with data. API embeds the full Problem (incl. description) per row — payload bloat, likely crash cause. Add/edit unreachable. v1 manages these inline on the problem form. → FE fix + slim API payload.

**🐛 Audit Log records nothing (P0 verify).** 0 entries despite heavy admin API traffic; dashboard "No changes recorded yet". v1 has 4,972 Django LogEntries (not migrated, different model). v2's log adds IP/result/date-range but has no user filter. → backend: verify/fix audit middleware writes; FE: user filter + object links.

**Submissions.** No admin detail at all — `/admin/submissions/<id>` 404s; detail is the read-only public page + Rejudge/Compare. Missing vs v1: edit language/status/result/points/judged-on/submission lock/is archived/compile errors, source editing, contest-submission association, HISTORY. List: no total count, no ID/problem-name/memory/language columns, status+result filters merged into one Verdict, no bulk delete/rejudge/rescore (per-row actions + Batch Rejudge exist). → API+FE (model mapped).

**Judges.** Missing: **judge tier** (HA routing unmanageable), edit name/auth key after create, capabilities M2M editing. 1,861 assigned problems render unpaginated. Extras: status cards, search, refresh, language versions. → API+FE.

**Languages.** Missing: runtime info override. **File-size limit unit changed MB → bytes** — semantics/migration risk, verify what the judge actually reads. Icon-only actions unlabeled. → API+FE + unit decision.

**Navigation Bars.** List shows PARENT/LEVEL but the form can't set **parent item** or **highlight regex** — hierarchy unmanageable. → API+FE (check model mapping).

**Misc Configs.** Parity (0 rows both); v2 adds search + immutable-key guard. ✔

**MOSS.** Parity, not an extra — v1 has it at `/contest/<key>/moss`; v2 centralizes + adds Clear results. ✔

**v1-only areas with no v2 equivalent:**
| Area | Impact |
|---|---|
| Impersonate (~13.8k log entries in v1) | heavily used support flow, gone entirely |
| Registration profiles (5,688; Activate users / Re-send activation emails) | manual activation flow gone |
| Flat pages | static pages (/about/…) unmanageable |
| Redirects | redirect rules unmanageable |
| Sites / Social auth admin | likely intentional (single site, empty data); no social-unlink flow |

## 4. Normal user view (public pages, audited)

Severity: **H** = core function/info missing or wrong, **M** = useful feature weaker, **L** = polish.

### 4.1 Problems list — H
- **>50-point problems unreachable:** default `points_max=50` and slider capped at 50 hide 19 problems (incl. the 100-pt `[I LOVE BUG]` series); search applies the same filter, so they're unfindable except by direct URL. **Live bug affecting all users.**
- No group/category filter (~30 groups in v1), no problem-type filter (~80 types), no "has editorial" filter/column, no #AC column, no curated Suggest list.
- L: AC rate integer-rounded; default sort hides newest problems.
- v2 adds: 3-state solved filter, per-page + total, Trending, Surprise Me.

### 4.2 Contests — H
- **Descriptions not rendered** — placeholder text even when v1 has a real description.
- **Announcements missing entirely** (verified a real `tet26` announcement absent in v2).
- **"View all past contests" button is dead**; only 10 recent past contests vs v1's ~250/13 pages.
- M: no live countdowns, no Participation tab, scoreboard lacks CSV export / name-org / virtual toggles / Total AC row, minute-only times, format rules incomplete ("Infinite Window" shown for fixed-window contests), tabs not deep-linkable.
- v2 adds: Join/Virtual-join in list, Rating+Window columns, statistics page.

### 4.3 Submission detail — H (data bug)
- **Memory header shows `3944.0MB`** — KB value labeled MB (×1000 wrong); correct value in test table.
- M: header score `1/20` mixes points/test-count (v1: `20/20 (1.000/1.000 points)`); no Resubmit.
- L: raw language codes, problem code instead of name, no judge name.

### 4.4 Problem detail — M
- No auto-PDF for problems without an uploaded PDF (v1 generates for every problem); no file-upload submit; no problem-scoped Report/clarification link (generic ticket form instead).
- **Raw HTML leaks in statements** (`<i>…</i>` shown literally on `swgame`) — rendering bug.
- L: no sample Copy buttons, no comment Preview/markdown guide, no allowed-languages display, no partial indicator, editorial not deep-linkable.
- v2 adds: inline submit box, AC rate, solved badge, KaTeX works.

### 4.5 Organizations — H
- **No org problem list** (org-scoped practice problems); M: no org contest list, no org submissions; L: no points ranking, no "your organizations", member list lacks points/solved.

### 4.6 Profile — M
- No Blogs tab, no activity heatmap, no "View submissions" link; solved list codes-only (no names/dates/language/tests).
- **PP breakdown disagrees with header** (668/332.80 vs 611/324) — data inconsistency to verify.
- v2 adds: PP breakdown, rating chart/table, orgs tab, joined/seen.

### 4.7 Submissions list / users / home — M/L
- Submissions: no time/memory columns, no org filter, single-select verdict/language, raw language codes.
- Users/leaderboard: no display name/org inline, no solved column.
- Home: no comment stream, no top contributors; **Wiki/About 404**; countdown raw hours (`890:49:45`); home top-rating widget filters users that /ratings shows.
- v2-only pages (no gap): /ratings, /stats (L: unformatted judge load floats).

### 4.8 Cross-cutting
- Empty/skeleton flashes and wrong-value counters on every page (1–3 s); tabs don't change URL (no deep links/back button); raw identifiers shown to users; several widget-vs-page data inconsistencies.

---

## 5. Cross-cutting UX issues

1. **Auth flash + spinner:** every admin navigation briefly renders a logged-out navbar ("Log in / Sign up") and a full-page spinner before hydration. Feels broken/slow. → FE: persist session state (cookie/SSR) and render skeleton inside the admin shell, not a logged-out page.
2. **Sidebar is a short fixed-height scroll card** instead of a full-height persistent nav; sub-items (e.g. problem groups/types under Problems in v1) are separate top-level entries. → FE layout.
3. **No breadcrumbs** on admin pages (v1: Home › Online Judge › Problems). → FE.
4. **Icon-only row actions** everywhere (edit/clone/delete unlabeled). → FE labels/tooltips; destructive actions need confirmation + undo where possible.
5. **Low density:** tall rows show ~6 items/screen vs v1's ~20. → FE table density pass.
6. **Modals instead of pages:** most v2 create/edit forms are modals over the list — no URL per record, no browser history, not deep-linkable (v1 has dedicated add/change pages). → FE: route-based forms or at least URL-synced modals.
7. **Weak pagination:** Prev/Next only; users = 5,694 rows at 50/page with no page numbers/jump. → FE (API already returns totals).
8. **Accessibility:** row action icons on users/blog/comments/taxonomy have no aria-label/title. → FE.
9. **2FA gate:** none needed — v1 has no org-specific gate (global middleware only).

---

## 6. Roadmap

Effort key: S ≤ half day, M ≤ 1–2 days, L ≤ 3–5 days (one dev, includes tests).
Tiers from §0. Every phase keeps v2 design; acceptance = repeat the v1 flow in v2 browser and diff fields.

### Phase 0 — Bugs + FE quick wins (S)
- **Fix Language Limits crash** (FE error boundary + slim the API row payload; API change allowed here).
- **Verify/fix audit-log recording** (0 entries despite traffic) — backend middleware check.
- **Fix `points_max=50` default + slider cap** — 19 problems currently unfindable by normal users (FE, maybe API default).
- **Fix submission memory unit** (KB shown as MB, ×1000) and confusing `1/20` score header (FE).
- **Fix contest description placeholder** (render real description) and **dead "View all past contests" button** (FE).
- **Fix raw HTML leak** in problem statements (`<i>` shown literally) — sanitization/renderer config (FE).
- Link Editorial / Clarifications / (Announcements once built) from problem & contest edit pages; add tab bar like the Data page.
- Contest create: add Lock After + Format config (API already supports).
- Show pagination + total on problems/contests lists (API already returns them).
- Label row actions; add "View on site"; add breadcrumbs.
- Fix auth flash / logged-out skeleton.
Files: `claoj-web/src/app/[locale]/admin/{problems,contests}/**`, admin shell components.

### Phase 1 — Problem form parity (M–L, API+FE)
- Extend `admin_problem.go` create/update DTOs + handlers: suggester, date, license, source, og_image, summary, short_circuit, is_full_markup, both visibility modes, org-private + organizations, banned users, curators/testers on create; per-language limits CRUD; translations admin endpoints.
- Rebuild form into v1-equivalent sections (Basic / People / Publishing / Access / Scoring & Limits / Social / Content extras) with v2 styling; add Translations tab.
- Go handler tests + jest form tests; verify each field round-trips against shared DB (read back via v1 admin).

### Phase 2 — Contest form parity (M–L, API+FE, small MAP)
- Extend `admin_contest.go` DTOs: rating params, scoreboard visibility/cache/precision, push announcements, hide authors, show short display, og_image, logo override, label script, private contestants, organizations, banned users.
- MAP missing columns after confirming them in `OJ/judge/models/contest.py` (classes, join orgs, scoreboard/submission view perms, rating exclusions) — gorm fields only, no migration.
- Announcements admin: list/create/resend endpoints (model exists) + UI tab.
- Wire dead toggles (rated → rating params; org-private → org picker).

### Phase 3 — List UX (M, API+FE+NEW)
- Sort + creator/year filters + authors in list DTOs (problems, contests, submissions).
- Bulk endpoints (delete/visibility/etc.) + checkbox selection + action bar.
- Dense table variant shared across admin lists.

### Phase 4 — Remaining admin sections (prioritized from §3)
1. **Users + Groups (L):** user edit (identity, staff/superuser, groups, permissions) + profile tab (organization, rank, mute, notes, display override, 2FA/WebAuthn admin); group member editor; user list filters + page numbers. Closes the "can't make someone staff / add to group / set org" dead ends.
2. **Organizations (M):** admins M2M, logo, max size, creation date, member management, recalculate scores; unify create/edit fields (no 2FA gate: v1 has none beyond the global middleware).
3. **Blog posts (S–M):** authors + organization fields, fix Authors column, image upload.
4. **Tickets (M):** show/edit linked item, per-message edit/remove, editable title/creator/time.
5. **Taxonomy / Tags (S–M):** included-problems/contests M2M, taxonomy search.
6. **Comments / Licenses (S):** comment metadata editing + bulk hide/unhide; license file upload.
7. **Submissions admin detail (M):** edit status/result/language/points/judged-on/lock/archived, source + contest association, history; list total + columns + split status/result filters + bulk rejudge/rescore/delete.
8. **Impersonation (M):** impersonate flow + log (heavily used in v1) — new backend session mechanism + FE.
9. **Registration profiles (S–M):** activate users / re-send activation emails.
10. **Judges (S):** tier field, name/auth-key edit, paginate assigned problems.
11. **Navbar / Languages (S):** parent + highlight regex; runtime info override; decide MB-vs-bytes unit (check judge consumer).
12. **Flat pages / Redirects (S–M):** admin CRUD for static content and redirect rules.

### Phase 4b known limits (accepted, not implemented)
- Hiding a comment does not cascade to its replies; v1's bulk actions don't either.
- An organization's `access_code` stays outside the org field gate.
- No media cleanup / ephemeral-volume handling.

### Phase 5 — Normal user view gaps (prioritized from §4)
1. **Contest content (M):** render descriptions, announcements UI (pairs with Phase 2 announcement admin), fix dead "View all past contests" + paginate past contests, live countdowns, deep-linkable tabs.
2. **Problem discovery (M):** group + type + editorial filters, #AC column, Suggest list; raise/remove the 50-point cap (bug fixed in Phase 0).
3. **Org pages (M):** problem/contest/submission lists.
4. **Scoreboard (S–M):** CSV export, name/org + virtual toggles, HH:MM:SS times, Total AC row.
5. **Profile (S–M):** Blogs tab, heatmap, View submissions link, richer solved list; verify PP breakdown math.
6. **Problem page (S–M):** auto-PDF generation, file-upload submit, problem-scoped report link, fix raw-HTML sanitization, sample Copy buttons.
7. **Polish (S):** friendly language names everywhere, time/memory columns + org filter on submissions, Resubmit, display names on leaderboard, Wiki/About pages, comment stream on home, consistent widget filtering, loading-state pass (skeletons with correct totals, no wrong-value flashes).

---

## Appendix A — audit raw notes

- `audit-admin-content.md`, `audit-admin-judging.md`, `audit-user-view.md` (session temp) — full per-section field lists from the parallel browser audits.