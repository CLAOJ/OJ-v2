# Design: v2 URL parity with v1 (SEO preservation)

**Date:** 2026-07-21
**Component:** `claoj-web` (Next.js 16.1.6 + next-intl 4.8.3)
**Goal:** Make v2 (Next.js) URLs identical to v1 (DMOJ/Django) URLs so existing Google
rankings and analytics tracking transfer to v2 instead of being lost.

## Problem

v2 URLs diverge from v1 in two ways, both of which break the URL identity Google indexed
against v1:

1. **Locale prefix.** v2 uses next-intl with `defaultLocale: 'en'` and
   `localePrefix: 'as-needed'`, so English pages are bare (`/problems/aplusb`) and
   Vietnamese pages carry a prefix (`/vi/problems/aplusb`). v1 (DMOJ) has **no locale
   segment at all** — language is chosen by a cookie.
2. **Pluralized detail routes.** v1's convention is **plural for lists, singular for
   detail** (`/problems/` list, `/problem/<code>` detail). v2 made three detail routes
   plural: `/problems/[code]`, `/contests/[key]`, `/submissions/[id]`. The blog also
   moved from v1's `/post/<id>-<slug>/` to `/blog/[id]`.

## Decisions (locked with the user)

| Decision | Choice |
|---|---|
| Locale in URL | `localePrefix: 'never'`, `defaultLocale: 'vi'` — **no prefix for any language**, language via `NEXT_LOCALE` cookie (exactly like v1). |
| Old-v2 redirects | Not needed — v1 is the live/indexed site; v2 is not yet indexed. Goal is purely v2 URLs == v1 URLs. |
| Scope | Full route-by-route audit. |
| Blog | Full match: rename `blog` → `post`, detail path `/post/<id>-<slug>`. |
| Utility/auth pages | Leave as-is (noindex, no ranking value). |

## Route parity table

### Must change (indexed / SEO-critical)

| Page | v1 (target) | v2 now | Action |
|---|---|---|---|
| Problem detail | `/problem/<code>` | `/problems/[code]` | rename dir → `problem/[code]` (incl. child `editorial`) |
| Contest detail | `/contest/<key>` | `/contests/[key]` | rename dir → `contest/[key]` (incl. child `stats`) |
| Submission detail | `/submission/<id>` | `/submissions/[id]` | rename dir → `submission/[id]` |
| Blog list | `/post/` | `/blog` | rename dir → `post` |
| Blog post | `/post/<id>-<slug>/` | `/blog/[id]` | rename dir → `post/[slug]`, param `= "<id>-<slug>"`, parse leading int as id, fetch `/blog/{id}` |
| Organization detail | `/organization/<pk>-<slug>` | `/organization/[id]` | build `<id>-<slug>` links, param parse leading int as id, fetch `/organization/{id}` |

### Already correct — only the prefix disappears

`/` (home), `/problems` (list), `/problems/random`, `/contests` (list),
`/contests/calendar`, `/submissions` (list), `/users`, `/user/[username]`,
`/organizations`, `/tickets`, `/ticket/[id]`.

### Different but intentionally left as-is (utility / noindex / v2-new)

`/login` (v1 `/accounts/login/`), `/register`, `/settings` (v1 `/edit/profile/`),
`/ticket/create` (v1 `/tickets/new`), `/submissions/diff/[id]` (v1 `/submissions/diff?a=&b=`),
`/ratings` (v2-new), `/stats` (v2-new), all `/admin/*`.

### Slug storage (verified against the shared DB)

Both v1 and v2 point at the **same `claoj` MariaDB**. Slug is a stored column, not derived:

- `judge_blogpost.slug` — `varchar(50)` NOT NULL. v1 create view sets it as
  `''.join(x for x in title.lower() if x.isalpha() or x.isnumeric())` — lowercase,
  **all non-alphanumerics stripped (no hyphens), Unicode letters preserved** (e.g. title
  "Lấy tiền" → slug `lấytiền`). Some slugs are hand-edited (e.g. `ctqt`). Admin uses
  Django's hyphenated slugify. Public view looks up by **id only**; slug is decorative.
- `judge_organization.slug` — `varchar(128)` NOT NULL, short custom codes
  (`itcla`, `dtqg22`, `thpt_dh`). Public view looks up by **id only**.

**Implication:** v2 must emit the **stored slug from the API** (`post.slug`, `org.slug`) —
NOT a re-slugified title, which would produce a different string and break the indexed URL.
The stored slug is already returned by the public blog API and the DB is shared, so **no
backend change is required for blog**; organizations need the public org response to
include `slug` (verify — the column/model already has it). Both routes parse the leading
integer from the `<id>-<slug>` segment and fetch by id, exactly like v1.

> Minor drift noted (out of scope): v2's GORM model declares blog slug `size:100` while the
> live column is `varchar(50)`. Harmless for URLs; flag before any v2-driven schema migration.

## Change set (files)

1. **`src/navigation.ts`** — `defaultLocale: 'vi'`, `localePrefix: 'never'`.
2. **`src/i18n/request.ts`** — default fallback `'en'` → `'vi'`.
3. **`src/app/api/setlang/route.ts`** (+ its caller, the language switcher component) —
   stop building `/${locale}${path}`. Under `never` there is no prefix; set the
   `NEXT_LOCALE` cookie and return/keep the same prefix-less path.
4. **Route directory renames** under `src/app/[locale]/`:
   - `problems/[code]/` → `problem/[code]/` (keep `problems/` list + `problems/random/`)
   - `contests/[key]/` → `contest/[key]/` (keep `contests/` list + `contests/calendar/`)
   - `submissions/[id]/` → `submission/[id]/` (keep `submissions/` list + `submissions/diff/`)
   - `blog/` → `post/`, `blog/[id]/` → `post/[slug]/` (param holds `<id>-<slug>`, parse leading int → id)
   - `organization/[id]/` — keep dir; param now holds `<id>-<slug>`, parse leading int → id
     (also its children `organization/[id]/blog`, `organization/[id]/manage`)
5. **Internal links (~30 detail refs across ~25 files, + org links)** — rewrite
   `` `/problems/${…}` `` → `` `/problem/${…}` ``, contest, submission to singular.
   Blog links → `` `/post/${post.id}-${post.slug}` `` using the **stored** `post.slug` from
   the API (never a re-slugified title). Org links `` `/organization/${org.id}` `` →
   `` `/organization/${org.id}-${org.slug}` ``.
   **Must NOT touch** `/problems` (list), `/problems/random`, `/contests` (list),
   `/contests/calendar`, `/submissions` (list), `/submissions/diff`. Covers links from
   next-intl `Link`, `next/link`, and `next/navigation` router calls (`push`/`replace`/`redirect`).
6. **`src/lib/seo.ts`** —
   - `generateCanonicalUrl`: `${SITE_URL}/${locale}${path}` → `${SITE_URL}${path}`.
   - `generateContestJsonLd` url: `/contests/${key}` → `/contest/${key}`.
   - `generateArticleJsonLd` url + `mainEntityOfPage`: `/blog/${id}` → `/post/${id}-${slug}`.
   - `generateWebSiteJsonLd` search target: `/en/problems?…` → `/problems?…`.
7. **`src/app/sitemap.ts`** — emit a single prefix-less URL set (not per-locale) using the
   v1-style names (`/problem`… are dynamic, so static list stays; correct `/blog` → `/post`).
   Dynamic per-entity sitemap entries (problem/contest/user/post) are a **follow-up**, not
   this task.
8. **`src/app/[locale]/layout.tsx`** — audit `alternates`/`canonical`/`hreflang` metadata;
   drop locale-prefixed alternates (single-URL model like v1).

Note: the `[locale]` dynamic segment **stays** — next-intl with `localePrefix: 'never'`
still resolves locale through that segment via the middleware; only the visible URL loses
the prefix. There are **no** hardcoded `/en/` or `/vi/` string literals in the codebase
(verified), which keeps the sweep contained to the files above.

## Testing / verification

- `npm run build` (or `tsc`) passes — no broken route imports after renames.
- Manual/e2e: `/problem/aplusb`, `/contest/<key>`, `/submission/<id>`, `/post/<id>-<slug>`
  resolve and render; list pages `/problems`, `/contests`, `/submissions` unaffected.
- Language switch (VI↔EN) still works and **keeps the same URL** (cookie only), no `/vi/`
  or `/en/` appears in the address bar.
- `view-source` on a problem page: canonical + JSON-LD URLs are prefix-less and singular.
- `grep` sweep confirms no remaining `` `/problems/${ `` `` `/contests/${ `` `` `/submissions/${ ``
  `` `/blog/${ `` internal links.

## Out of scope

- 301 redirects from old v2 URLs (v2 not indexed).
- Dynamic sitemap entries for individual problems/contests/users/posts (follow-up).
- Aligning utility/auth page paths with v1.
