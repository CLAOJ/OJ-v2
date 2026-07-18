# Schema Audit — OJ-v2 GORM models vs. Django (`OJ/judge/models/*.py`)

**Date:** 2026-07-19
**Scope:** every GORM model in `claoj/models/*.go`, plus GORM models declared
outside that package (`api/v2/auth/oauth_helpers.go`, `api/v2/moss.go`,
`api/v2/comment.go`), diffed column-by-column against the Django models that
are the single source of truth for the shared MySQL schema
(`F:/Coding/CLAOJ/OJ/judge/models/{problem,contest,profile,submission,comment,
interface,ticket,runtime,problem_data}.py`).

**Verdicts**
- **OK** — every mapped column/table exists in Django with a matching name/type.
- **RETAINED-V2** — an additive v2-only table or column, kept on purpose
  (decision recorded 2026-07-19, see below). Django never creates, reads, or
  writes these; OJ-v2 owns them entirely.
- **DRIFT** — a column/table that shouldn't exist, or a wrong name/type.

**Result: zero DRIFT.** Every object in this audit is either OK (pure Django
parity) or RETAINED-V2 (an explicit, documented exception). See "Decision
record" below for how the RETAINED-V2 set was arrived at.

---

## 1. Already-removed v2 stuff — confirmed absent

Grepped `claoj/models/*.go` and the full `claoj/` tree for the following;
none are present:

| Removed in | Item | Status |
|---|---|---|
| Task 5 | `Role`, `RefreshToken` models | absent |
| Task 8 | `AuditLog` model | absent |
| Task 10 | `PasswordResetToken`, `EmailVerificationToken` models | absent |
| Task 11 | `Contest.MaxSubmissions` field | absent (Go `Contest` struct has no such field; `judge_contestproblem.max_submissions` is legitimate — see row below) |
| Task 12 | `Problem.Suggestion*` fields (`suggestion_status`, `suggestion_notes`, `suggestion_reviewed_at`, `suggestion_reviewed_by_id`) | absent (Go `Problem` struct has none of these; `Problem.SuggesterID`/`suggester_id` is a *different*, legitimate Django field — see row below) |

---

## 2. OK — Django-table models with full column parity

| Go model | Django model | Verdict | Note |
|---|---|---|---|
| `AuthGroup` | `django.contrib.auth.models.Group` (`auth_group`) | OK | Standard Django auth table |
| `DjangoContentType` | `django.contrib.contenttypes.models.ContentType` (`django_content_type`) | OK | |
| `AuthPermission` | `django.contrib.auth.models.Permission` (`auth_permission`) | OK | |
| `AuthUserGroup` | `auth_user_groups` (implicit M2M through table) | OK | |
| `AuthGroupPermission` | `auth_group_permissions` (implicit M2M through table) | OK | |
| `AuthUserPermission` | `auth_user_user_permissions` (implicit M2M through table) | OK | |
| `AuthUser` | `django.contrib.auth.models.User` (`auth_user`) | OK | |
| `Organization` | `Organization` (`judge_organization`, `profile.py`) | OK | |
| `Profile` | `Profile` (`judge_profile`, `profile.py`) | OK | Verified field-for-field, incl. TOTP/WebAuthn/api_token/notes columns — all real Django fields, not v2 additions |
| `OrganizationRequest` | `OrganizationRequest` (`judge_organizationrequest`) | OK | |
| `WebAuthnCredential` | `WebAuthnCredential` (`judge_webauthncredential`) | OK | |
| `ProblemGroup` | `ProblemGroup` (`judge_problemgroup`) | OK | |
| `ProblemType` | `ProblemType` (`judge_problemtype`) | OK | |
| `License` | `License` (`judge_license`) | OK | |
| `Problem` | `Problem` (`judge_problem`, `problem.py`) | OK | `SuggesterID`/`suggester_id` matches Django's real `suggester` FK (problem.py:201), distinct from the removed `suggestion_*` review-workflow columns |
| `ProblemTranslation` | `ProblemTranslation` (`judge_problemtranslation`) | OK | |
| `ProblemClarification` | `ProblemClarification` (`judge_problemclarification`) | OK | Real Django model — do not confuse with `ContestClarification` (RETAINED-V2, see below) |
| `LanguageLimit` | `LanguageLimit` (`judge_languagelimit`) | OK | |
| `Solution` (base columns) | `Solution` (`judge_solution`) | OK | `id, problem_id, pdf_url, content, is_public, publish_on, authors` all match Django exactly. **4 extra columns are RETAINED-V2 — see §3.** |
| `ProblemData` | `ProblemData` (`judge_problemdata`) | OK | |
| `ProblemTestCase` | `ProblemTestCase` (`judge_problemtestcase`) | OK | |
| `ContestTag` | `ContestTag` (`judge_contesttag`) | OK | |
| `Contest` | `Contest` (`judge_contest`, `contest.py`) | OK | Verified all 27 fields; no `max_submissions` (that field belongs to `ContestProblem`, not `Contest`) |
| `ContestAnnouncement` | `ContestAnnouncement` (`judge_contestannouncement`) | OK | |
| `ContestParticipation` | `ContestParticipation` (`judge_contestparticipation`) | OK | `RealStart`/`start` db_column matches Django's `db_column='start'` |
| `ContestProblem` | `ContestProblem` (`judge_contestproblem`) | OK | `MaxSubmissions`/`max_submissions` matches Django's real field on **ContestProblem** (contest.py:565), not the removed `Contest.max_submissions` |
| `ContestSubmission` | `ContestSubmission` (`judge_contestsubmission`) | OK | |
| `Rating` | `Rating` (`judge_rating`) | OK | |
| `Submission` | `Submission` (`judge_submission`) | OK | Incl. `locked_after` (real Django field, submission.py:89) |
| `SubmissionSource` | `SubmissionSource` (`judge_submissionsource`) | OK | |
| `SubmissionTestCase` | `SubmissionTestCase` (`judge_submissiontestcase`) | OK | |
| `Comment` | `Comment` (`judge_comment`) | OK | |
| `CommentVote` | `CommentVote` (`judge_commentvote`) | OK | |
| `CommentLock` | `CommentLock` (`judge_commentlock`) | OK | |
| `BlogPost` | `BlogPost` (`judge_blogpost`) | OK | |
| `BlogVote` | `BlogVote` (`judge_blogvote`) | OK | |
| `MiscConfig` | `MiscConfig` (`judge_miscconfig`) | OK | |
| `NavigationBar` | `NavigationBar` (`judge_navigationbar`) | OK | |
| `GeneralIssue` | `GeneralIssue` (`judge_generalissue`) | OK | |
| `Ticket` | `Ticket` (`judge_ticket`) | OK | |
| `TicketMessage` | `TicketMessage` (`judge_ticketmessage`) | OK | |
| `Language` | `Language` (`judge_language`) | OK | |
| `Judge` | `Judge` (`judge_judge`) | OK | |
| `RuntimeVersion` | `RuntimeVersion` (`judge_runtimeversion`) | OK | |

Not mapped on the Go side (missing, not DRIFT — DRIFT is only extra/wrong,
never absent): Django's `ContestMoss` (`judge_contestmoss`). OJ-v2 has its
own, non-overlapping `moss_result` table instead (see §3) — noted here for
completeness, not actionable.

---

## 3. RETAINED-V2 — additive v2-only schema (decision record 2026-07-19)

During this audit, two classes of v2-only additions were found beyond the
four already-known runtime tables. **Decision: keep all of them.** They
become permanent, documented additive schema. Django ignores them entirely;
`scripts/v2_runtime_tables.sql` is the one-time script that provisions them
on a Django-migrated database for a side-by-side OJ-v2 deployment.

| Object | Kind | Backing feature | Additive vs. alter | Collision risk |
|---|---|---|---|---|
| `notification` | table | In-app notifications (`models/notification.go`) | additive table | none — not `judge_`-prefixed |
| `notification_preference` | table | Per-user notification settings (`models/notification.go`) | additive table | none — not `judge_`-prefixed |
| `totp_device` | table | TOTP 2FA secret storage (`models/runtime.go`) | additive table | none — not `judge_`-prefixed |
| `backup_code` | table | 2FA backup codes (`models/runtime.go`) | additive table | none — not `judge_`-prefixed |
| `oauth_user_link` | table | Google/GitHub OAuth login linking (`api/v2/auth/oauth_helpers.go`, routes `GET/POST /auth/oauth/:provider[/callback]`) | additive table | none — not `judge_`-prefixed |
| `moss_result` | table | Cached MOSS plagiarism-check results (`api/v2/moss.go`, routes `POST/GET /admin/submission/:id/moss`) | additive table | none — not `judge_`-prefixed. Distinct from Django's own `judge_contestmoss` (`ContestMoss` model), which OJ-v2 does not use |
| `judge_commentrevision` | table | Comment edit-history (`models/comment.go` `CommentRevision`, route `GET /comment/:id/revisions`, written on every comment edit in `api/v2/comment.go:356`) | additive table | **yes** — uses the `judge_` prefix Django reserves for its own migrations. Django's real edit-history mechanism is the generic `django-reversion` package (`reversion_version`/`reversion_revision`), not a per-app table; this name is not currently claimed by any Django migration, but a future Django migration could theoretically claim it. Kept as-is per explicit decision (not renamed) |
| `judge_contestclarification` | table | Contest clarification Q&A (`models/contest.go` `ContestClarification`, routes `GET/POST /contest/:key/clarifications`, `POST /contest/:key/clarification/:id/answer`) | additive table | **yes** — same `judge_` prefix caveat as above. Django's own contest "clarifications" concept is just `ProblemClarification` rows filtered by the contest's problems (`use_clarifications` is only a display toggle in `views/contests.py`); `ContestClarification` is a separate, v2-only, contest-scoped Q&A feature. Kept as-is per explicit decision (not renamed) |
| `judge_solution.is_official` | column (ALTER on Django table) | Editorial "official" badge (`api/v2/solution.go`) | additive column on Django-owned table | table is Django-owned; column itself does not collide with any current Django migration |
| `judge_solution.valid_until` | column (ALTER on Django table) | Editorial expiry date (`api/v2/solution.go`) | additive column on Django-owned table | same as above |
| `judge_solution.summary` | column (ALTER on Django table) | Editorial summary text (`api/v2/solution.go`) | additive column on Django-owned table | found during this audit in addition to the 3 columns named in the original decision request; included here for completeness since Django's `Solution` model has no `summary` field either — see "Note" below |
| `judge_solution.language` | column (ALTER on Django table) | Editorial language selection (`api/v2/solution.go`) | additive column on Django-owned table | same as above |

**Note on `judge_solution`:** the original decision request named 3 extra
columns (`is_official`, `valid_until`, `language`). Re-checking Django's
`Solution` model (`problem.py:608-644`) during finalization turned up a 4th:
`summary` is also not a Django field (Django's `Solution` model is exactly
`problem, pdf_url, is_public, publish_on, authors, content`). Since the
decision was "keep the feature, no DRIFT to delete," `summary` is included
in `scripts/v2_runtime_tables.sql` alongside the other three so the feature
is actually complete after provisioning — flagging this correction
explicitly rather than silently dropping a column from the deploy script.

**Also observed (not a schema issue):** `api/v2/comment.go` declares a
second, unused `CommentRevision` struct (lines 17–29) that duplicates
`models.CommentRevision` and maps to the same `judge_commentrevision` table.
The file actually uses `models.CommentRevision` at the call sites (lines
356, 400). This is dead/duplicate code, not a schema drift — left as-is
since no schema objects are affected and the task scope is schema, not
general code cleanup.

---

## 4. Summary

- **OK:** 44 Django-table models (auth: 7, profile: 5, problem: 8, contest: 7,
  submission: 3, comment: 3, interface: 4, ticket: 3, runtime: 3) + the base
  columns of `Solution` (counted above).
- **RETAINED-V2:** 8 tables (`notification`, `notification_preference`,
  `totp_device`, `backup_code`, `oauth_user_link`, `moss_result`,
  `judge_commentrevision`, `judge_contestclarification`) + 4 additive columns
  on `judge_solution`.
- **DRIFT:** 0.
