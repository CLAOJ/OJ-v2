# CLAOJ v1 ↔ v2 final route audit (v2 = main @ 6fedfd9)

Source: full read of `OJ/dmoj/urls.py` (incl. `accounts/`, conditional `newsletter/`, `impersonate/`, favicon redirects), `OJ/judge/admin/*.py` registrations and `get_urls()` extras, mapped to `claoj-web/src/app/[locale]/**/page.tsx` (+ `*Content.tsx`) and `claoj/api/router.go`.
Verdicts: `parity` | `substitute` (tab/modal/filtered list/documented replacement) | `gap` | `not-a-screen` (AJAX, ops, feed, action-only POST, redirect).

## Summary

**214 v1 route rows: parity 55 · substitute 76 · not-a-screen 79 · gap 4.**
Public/page rows: 39 parity, 57 substitute, 73 not-a-screen, 3 gap. Admin rows: 16 parity, 19 substitute, 6 not-a-screen, 1 gap.

### Gap rows

| v1 URL | status | note |
|---|---|---|
| `/admin/sites/site/` | accepted, documented | v2 single-site (`CMSSiteID = 1`). |
| `/contests/tag/<name>` | real gap → being fixed | contests page read no `tag` param; only `GET /contest-tags` chips existed. |
| `/problem/<p>/tickets` | real gap → being fixed | no per-problem ticket list; tickets API had no `problem` filter. |
| `/problem/<p>/resubmit/<id>` | real gap → being fixed | no prefill of source/language from a previous submission. |

### Other defects found

1. Broken compare link: `submission/[id]/page.tsx:412` → `/submission/<id>/diff?compare=` (route does not exist; real route `/submissions/diff/[id]`).
2. Contest participation per-user view (`/contest/<c>/participations/<user>`) had no user filter.
3. Contest calendar month was client state only (v1 `/contests/<year>/<month>/`).
4. `/users/find`, `/contributors/find` had no server-side search deep link.
5. The draft report wrongly claimed `/contests?tag=` existed.

## 1. Home / flat pages / errors
| v1 | v2 page | v2 API | verdict |
|---|---|---|---|
| `/` (blog.PostList) | `/` dashboard | `/blogs`,`/contests`,`/users`,`/problems`,`/comments` | substitute (blog list moved to `/post`) |
| `/500/`, handler404/403/500 | `error.tsx`, `global-error.tsx` | — | parity / not-a-screen |
| flatpages (`/about/`) | `/[locale]/[...slug]` | `GET /page` | parity |
| favicon ×7 | `public/favicon*` | — | not-a-screen |

## 2. Accounts
| v1 | v2 | verdict |
|---|---|---|
| register / complete / closed | `/register` (+states) | parity / substitute |
| activate/<key>, activate/complete | `/verify-email?token=` | substitute |
| login | `/login` | parity |
| logout | `POST /auth/logout` | not-a-screen |
| password change(+done), reset(+confirm/complete/done) | `/settings` Account, `/forgot-password` | substitute (reset form parity) |
| social/error | inline on `/login` | substitute |
| email/change | `/settings` Account | substitute |
| email/change/activate/<key> | `/email/change/confirm?token=` | parity |
| 2fa (login/enable/edit/disable) | `/login` step, `/settings` Account | substitute |
| webauthn attest/assert/delete, scratchcode | `/settings` Passkey, `/login` | substitute / not-a-screen |
| api/token generate/remove | `/settings` API token | not-a-screen |
| social_django urls | `GET /auth/oauth/:provider` | not-a-screen |

## 3. Problems
| v1 | v2 | verdict |
|---|---|---|
| `/problems/` | `/problems` | parity |
| `/problems/random/` | `/problems/random` | not-a-screen (v1 redirect) |
| `/problems/create` | `/admin/problems/create` | substitute |
| `/problems/suggest_list/` | `/problems/suggest` | parity |
| `/problems/suggest` | `/problems/suggest/new` | parity |
| `/problem/<p>` | `/problem/[code]` | parity |
| `/problem/<p>/edit` | `/admin/problems/[code]/edit` | substitute |
| `/problem/<p>/editorial` | `/problem/[code]/editorial` | parity |
| `/problem/<p>/raw` | `/problem/[code]/print?raw=1` | substitute |
| `/problem/<p>/pdf[/<lang>]` | `/problem/[code]/print` | substitute |
| `/problem/<p>/clone` | admin clone modal | not-a-screen |
| `/problem/<p>/submit` | in-page submit form | substitute |
| `/problem/<p>/resubmit/<id>` | — | **gap** |
| `/problem/<p>/rank/` | `/submissions?problem=&best=1` | substitute |
| `/problem/<p>/submissions/[<user>/]` | `/submissions?problem=[&user=]` | substitute |
| `/problem/<p>/test_data[/init]` | `/admin/problems/[code]/data` | substitute / not-a-screen |
| `/problem/<p>/test_data/diff` | `/admin/problems/[code]/diff` | parity |
| `/problem/<p>/data/<path>` | data page Files | not-a-screen |
| `/problem/<p>/tickets` | — | **gap** |
| `/problem/<p>/tickets/new` | `/ticket/create?problem=` | substitute |
| `/problem/<p>/manage/submission` (+rejudge/preview/success/rescore) | `/admin/submissions` batch + rescore-all | substitute / not-a-screen |

## 4. Submissions
`/submissions/` → `/submissions` (parity); `/submissions/diff` → `/submissions/diff/[id]?compare=` (substitute); `/submissions/user/<u>/` → `?user=` (substitute); `/submission/<id>` → `/submission/[id]` (parity); abort/raw/src → in-page (not-a-screen).

## 5. Users
`/users/` → `/users` (parity); `/users/<page>`,`/users/find` → `?page=`/`?search=` (not-a-screen); `/user` → own profile (substitute); `/edit/profile/`, `/data/prepare|download` → `/settings` tabs (substitute); `/user/<u>` → `/user/[username]` (parity); `/user/<u>/ban` → admin dialog (substitute); `/user/<u>/blog/` → `/user/[username]/blog` (parity); `/user/<u>/solved[/ajax]` → Solved tab / pp-breakdown (substitute); `/user/<u>/submissions/` → `?user=` (not-a-screen).

## 6. Comments
upvote/downvote/hide/votes/history/render → widget actions (not-a-screen); `/comments/<id>/edit` → inline editor `PATCH /comment/:id` (substitute).

## 7. Contests
| v1 | v2 | verdict |
|---|---|---|
| `/contests/` | `/contests` | parity |
| `/contests/<y>/<m>/` | `/contests/calendar` | substitute (deep link being added) |
| `/contests/new/` | `/admin/contests/create` | substitute |
| `/contests/tag/<name>[/ajax]` | — | **gap** |
| `/contest/<c>` | `/contest/[key]` | parity |
| edit / moss(+delete) / announce / clone | admin pages | substitute / not-a-screen |
| `/contest/<c>/ranking/` | `?tab=scoreboard` | substitute |
| join / leave | buttons | not-a-screen |
| `/contest/<c>/stats` | `/contest/[key]/stats` | parity |
| `/contest/<c>/rank/<p>/` | `/contest/[key]/rank/[problem]` | parity |
| `/contest/<c>/submissions/<u>[/<p>]` | `/submissions?contest=&user=[&problem=]` | substitute |
| participations[/<u>] | `?tab=participation` | substitute (user filter being added) |
| participation/disqualify | admin participations | not-a-screen |

## 8. Contributors
`/contributors` → `/contributors` (parity, `GET /users?sort=contribution`); `/<page>`, `/find` → `?page=`/`?search=` (not-a-screen).

## 9. Organizations
list/create/home → parity; users/edit → tabs/manage (substitute); join/leave/kick → actions; problems/contests/submissions → tabs (substitute); problem-create/contest-create/post-new → dedicated pages (parity); request(+detail) and requests pending/log/approved/rejected → manage `?state=` + dialog (substitute).

## 10. Runtimes / status / stats
`/runtimes/` → `/runtimes` (parity); `/runtimes/matrix/` → `/runtimes/matrix` (parity); `/status/`, `/status/oj/` → `/stats` (substitute, accepted); `/stats/data/all/` → JSON (not-a-screen).

## 11. Blog
`/post/` → `/post` (parity); `/post/<id>-<slug>/` → `/post/[slug]` (parity); edit → `/post/[slug]/edit` (substitute); `/post/new` → `/post/new` (parity); upvote/downvote → actions.

## 12. License / tickets / widgets / feeds / ops
`/license/<key>` → `/license/[key]` (parity); tickets list/new/detail → parity; ticket notes → admin notes (substitute); ticket open/close/good/norm, ajax → actions; mailgun, widgets/* (rejudge, single_submission, submission_testcases, detect_timezone, status-table, template, select2, preview ×11, martor upload/search), judge-select2 ×6, feeds ×6, tasks/*, newsletter (dead in v1), impersonate, i18n, sitemap → not-a-screen.

## 13. Admin
parity: blogpost, comment, commentlock, contest, contesttag, flatpage, redirect, judge, language, license, miscconfig, class, organization, problem, submission, ticket, auth user, auth group.
substitute: admin index/auth, contestparticipation, logentry→audit-logs, navigationbar (list+parent select), organizationrequest & problempointsvote (read-only), problemgroup/problemtype→taxonomy, profile→/admin/users, registrationprofile→/admin/registrations, reversion history→audit log.
gap: sites (accepted, single-site).
not-a-screen: admin action URLs (rate all, judge disconnect/terminate/disable, submission judge, jsi18n, autocomplete/select2).

## v2-only pages
`/ratings`, `/notifications`, `/resend-verification`, `/problems/random`, `/admin/registrations`, `/admin/redirects`, `/admin/language-limits`, `/admin/moss`, `/admin/contests/[key]/clarifications`, `/admin/problems/[code]/{clarifications,editorial,translations}`, `/organization/[id]/blog`.