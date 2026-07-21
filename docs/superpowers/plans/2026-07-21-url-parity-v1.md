# v2 URL Parity with v1 (SEO Preservation) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make v2 (Next.js) public URLs byte-identical to v1 (DMOJ) URLs — no locale prefix, singular detail routes, v1-style blog/org slugs — so existing Google rankings and analytics transfer instead of being lost.

**Architecture:** Switch next-intl to `localePrefix: 'never'` with `defaultLocale: 'vi'` (language via `NEXT_LOCALE` cookie, exactly like v1); rename three detail route folders from plural to singular; move `blog` → `post/[slug]` and re-slug organizations using the `<id>-<slug>` form; rewrite only *navigation* links (never API calls); and correct the SEO surface (canonical, JSON-LD, sitemap, hreflang).

**Tech Stack:** Next.js 16.1.6 (App Router, `proxy.ts` middleware), next-intl 4.8.3, React 19, TypeScript, Jest. Backend (Go/Gin) and MariaDB are **unchanged**.

## Global Constraints

- Keep the `src/app/[locale]/` dynamic segment — next-intl `localePrefix: 'never'` still resolves locale through it via the middleware; only the *visible* URL loses the prefix.
- **Never modify API calls.** `api.get/post/put/patch/delete(...)` and every path in `src/lib/api.ts` target the Go backend (whose routes are already singular and unchanged). Only rewrite *navigation*: `href=`, `<Link href=`, `router.push(`, `router.replace(`, `redirect(`.
- **Never touch `/admin/*` routes** — internal, noindex, out of scope.
- **Preserve list & static sub-routes:** `/problems`, `/problems/random`, `/contests`, `/contests/calendar`, `/submissions`, `/submissions/diff`, `/users`, `/organizations`, `/tickets`.
- Blog & org slugs are **stored DB values** (`judge_blogpost.slug`, `judge_organization.slug`), already returned by the public API as `slug`. Always build URLs from the API's `slug` field — never re-slugify a title (v1 slugs are alphanumeric-only, no hyphens, Unicode-preserving, sometimes hand-edited).
- Blog & org detail pages receive a `<id>-<slug>` route param but must call the API with the **leading integer only** (the backend parses numeric ids, not `93-lấytiền`).
- Branch off **`dev`** (the integration branch), e.g. `feat/url-parity-v1` — **not** `main` (`main` is reserved for stable CI/CD image builds). Commit after each task; merge back into `dev`.
- Spec: `docs/superpowers/specs/2026-07-21-url-parity-v1-design.md`.

---

## File Structure

**New files**
- `src/utils/route.ts` — `parseLeadingId(segment)` helper (extract numeric id from `<id>-<slug>`).
- `src/utils/route.test.ts` — unit tests for the helper.

**Renamed directories** (use `git mv` to preserve history)
- `src/app/[locale]/problems/[code]/`  → `src/app/[locale]/problem/[code]/`  (incl. child `editorial/`)
- `src/app/[locale]/contests/[key]/`   → `src/app/[locale]/contest/[key]/`   (incl. child `stats/`)
- `src/app/[locale]/submissions/[id]/` → `src/app/[locale]/submission/[id]/`
- `src/app/[locale]/blog/`             → `src/app/[locale]/post/` ; `post/[id]/` → `post/[slug]/`

**Modified (config / i18n / SEO)**
- `src/navigation.ts`, `src/i18n/request.ts`, `src/components/navbar/LanguageSwitcher.tsx`, `src/app/api/setlang/route.ts`
- `src/lib/seo.ts`, `src/app/[locale]/layout.tsx`, `src/app/sitemap.ts`, `src/app/robots.ts`

**Modified (navigation link strings)** — enumerated per task.

**Untouched by design:** everything under `src/app/[locale]/admin/`, `src/lib/api.ts`, and the Go backend.

---

## Task 1: Switch i18n to prefix-less, Vietnamese-default

**Files:**
- Modify: `src/navigation.ts`
- Modify: `src/i18n/request.ts`

**Interfaces:**
- Produces: `routing` with `defaultLocale: 'vi'`, `localePrefix: 'never'`. After this task, every `Link`/`useRouter`/`usePathname` from `@/navigation` emits prefix-less URLs, and the middleware (`src/proxy.ts`, unchanged) resolves locale from the `NEXT_LOCALE` cookie / `Accept-Language`.

- [ ] **Step 1: Edit `src/navigation.ts`**

Replace the `defineRouting` call:

```ts
export const routing = defineRouting({
    locales: ['en', 'vi'],
    defaultLocale: 'vi',
    localePrefix: 'never'
});
```

- [ ] **Step 2: Edit `src/i18n/request.ts` default fallback**

Change the invalid/undefined fallback from `'en'` to `'vi'`:

```ts
  // Default to 'vi' if locale is undefined or invalid
  if (!locale || !locales.includes(locale)) {
    locale = 'vi';
  }
```

- [ ] **Step 3: Build to verify config compiles**

Run: `npm run build`
Expected: build completes with no type/route errors.

- [ ] **Step 4: Manual smoke test**

Run: `npm run dev`, open `http://localhost:3000/problems`.
Expected: page renders in Vietnamese by default; the address bar stays `/problems` (no `/vi`, no `/en`); no redirect loop.

- [ ] **Step 5: Commit**

```bash
git add src/navigation.ts src/i18n/request.ts
git commit -m "feat(i18n): prefix-less URLs, Vietnamese default (localePrefix never)"
```

---

## Task 2: Fix language switching for the no-prefix model

**Files:**
- Modify: `src/components/navbar/LanguageSwitcher.tsx:16-34`
- Modify: `src/app/api/setlang/route.ts`

**Interfaces:**
- Consumes: `routing` from Task 1 (`localePrefix: 'never'`).
- Produces: language switch that sets `NEXT_LOCALE` and reloads the **same** unprefixed URL.

Under `localePrefix: 'never'` there is no `/en` or `/vi` path to navigate to — navigating to `/${newLocale}${path}` would 404. The switch is now: set the cookie, then hard-reload the current path so the server re-renders in the new locale.

- [ ] **Step 1: Replace `handleLanguageChange` in `LanguageSwitcher.tsx`**

Replace the whole function body (lines 16-34) with:

```ts
    const handleLanguageChange = (newLocale: string) => {
        if (newLocale === locale) return;
        // localePrefix is 'never': locale lives only in the NEXT_LOCALE cookie.
        // Set it, then hard-navigate to the SAME unprefixed path so the server
        // re-renders in the new language. usePathname() drops the query/hash,
        // so re-attach them to preserve filters like /submissions?user=x.
        document.cookie = `NEXT_LOCALE=${newLocale}; path=/; max-age=31536000; SameSite=Lax`;
        const bare = pathname === '/' ? '/' : pathname;
        window.location.assign(`${bare}${window.location.search}${window.location.hash}`);
    };
```

- [ ] **Step 2: Update `setlang` route to be prefix-less (defensive; route is otherwise unused)**

In `src/app/api/setlang/route.ts`, replace the prefixed-path construction with a cookie set + same-path echo:

```ts
        // localePrefix 'never': keep the same path, switch via cookie.
        const currentPath = path || '/';
        const response = NextResponse.json({
            success: true,
            locale,
            redirect: currentPath,
        });
        response.cookies.set('NEXT_LOCALE', locale, {
            path: '/',
            maxAge: 31536000,
            sameSite: 'lax',
        });
        return response;
```

(Delete the old `newPath = /${locale}...` line.)

- [ ] **Step 3: Build**

Run: `npm run build`
Expected: passes.

- [ ] **Step 4: Manual test the switcher**

With `npm run dev`: on `/problems`, click VI↔EN.
Expected: content language flips, the URL stays `/problems` (no prefix), and it does **not** revert on the next navigation. Repeat on `/submissions?user=foo` — the query string survives.

- [ ] **Step 5: Commit**

```bash
git add src/components/navbar/LanguageSwitcher.tsx src/app/api/setlang/route.ts
git commit -m "fix(i18n): switch language via cookie + same-path reload"
```

---

## Task 3: Add the leading-id helper (TDD)

**Files:**
- Create: `src/utils/route.ts`
- Create: `src/utils/route.test.ts`

**Interfaces:**
- Produces: `export function parseLeadingId(segment: string): string` — returns the substring before the first `-`, else the whole string. Used by the blog (`post/[slug]`) and organization detail pages to derive the numeric API id from an `<id>-<slug>` route param.

- [ ] **Step 1: Write the failing test — `src/utils/route.test.ts`**

```ts
import { parseLeadingId } from './route';

describe('parseLeadingId', () => {
  it('extracts the numeric id from an <id>-<slug> segment', () => {
    expect(parseLeadingId('93-lấytiền')).toBe('93');
  });
  it('handles a slug containing extra hyphens', () => {
    expect(parseLeadingId('1-thpt-dh')).toBe('1');
  });
  it('returns the whole segment when there is no slug', () => {
    expect(parseLeadingId('42')).toBe('42');
  });
  it('returns an empty string for an empty segment', () => {
    expect(parseLeadingId('')).toBe('');
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npx jest src/utils/route.test.ts`
Expected: FAIL — cannot find module `./route` / `parseLeadingId is not a function`.

- [ ] **Step 3: Implement `src/utils/route.ts`**

```ts
/**
 * Extract the leading numeric id from a route segment of the form `<id>-<slug>`.
 * v1 URLs look like `/post/93-lấytiền` and `/organization/1-itcla`; the backend
 * API is keyed by the numeric id only, so pages parse it out with this helper.
 */
export function parseLeadingId(segment: string): string {
  const dash = segment.indexOf('-');
  return dash === -1 ? segment : segment.slice(0, dash);
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `npx jest src/utils/route.test.ts`
Expected: PASS (4 passing).

- [ ] **Step 5: Commit**

```bash
git add src/utils/route.ts src/utils/route.test.ts
git commit -m "feat(routing): add parseLeadingId helper for <id>-<slug> params"
```

---

## Task 4: Problem detail route `problems/[code]` → `problem/[code]`

**Files:**
- Rename: `src/app/[locale]/problems/[code]/` → `src/app/[locale]/problem/[code]/` (moves `page.tsx`, `ProblemPageContent.tsx`, and child `editorial/page.tsx`)
- Modify (navigation strings — all are nav, no API collision): `src/app/[locale]/HomePageContent.tsx`, `src/app/[locale]/admin/judges/[id]/page.tsx`, `src/app/[locale]/admin/problems/[code]/editorial/page.tsx`, `src/app/[locale]/admin/problems/page.tsx`, `src/app/[locale]/contests/[key]/ContestPageContent.tsx`, `src/app/[locale]/problems/ProblemsPageContent.tsx`, `src/app/[locale]/problem/[code]/ProblemPageContent.tsx` (after rename), `src/app/[locale]/problem/[code]/editorial/page.tsx` (after rename), `src/app/[locale]/problems/random/page.tsx`, `src/app/[locale]/submissions/[id]/page.tsx`, `src/app/[locale]/submissions/page.tsx`, `src/app/[locale]/ticket/[id]/page.tsx`, `src/app/[locale]/user/[username]/UserProfilePageContent.tsx`, `src/components/submission/diff/SubmissionInfoCard.tsx`

**Interfaces:**
- Consumes: nothing new.
- Produces: problem detail served at `/problem/<code>`; all internal links point there.

- [ ] **Step 1: Rename the directory**

```bash
git mv "src/app/[locale]/problems/[code]" "src/app/[locale]/problem/[code]"
```

Note: `src/app/[locale]/problems/` still exists (list `page.tsx` + `ProblemsPageContent.tsx` + `random/`). Only the `[code]` subtree moves.

- [ ] **Step 2: Rewrite navigation links `` `/problems/${ `` → `` `/problem/${ `` **

Every occurrence of the template-literal prefix `` `/problems/${ `` is a navigation link (the backend uses `/problem/:code`, so no `api.*` call uses this plural form). Replace across `src/` — but NOT the literal `"/problems/random"` (no `${`) and NOT the list path `/problems`:

```bash
grep -rlE '`/problems/\$\{' src --include='*.tsx' --include='*.ts' \
  | xargs sed -i 's#`/problems/\${#`/problem/${#g'
```

- [ ] **Step 3: Verify no plural problem-detail links remain, and the list/random survive**

Run:
```bash
grep -rnE '`/problems/\$\{' src --include='*.tsx' --include='*.ts'   # expect: no output
grep -rnE '/problems/random|href="/problems"|`/problems`' src --include='*.tsx' --include='*.ts' | head   # expect: still present
```
Expected: first grep empty; second still shows the list + random links intact.

- [ ] **Step 4: Build**

Run: `npm run build`
Expected: passes; no “module not found” for the moved route.

- [ ] **Step 5: Manual check**

`npm run dev` → open `/problems` (list works), click a problem → URL is `/problem/<code>`, page renders; open `/problem/<code>/editorial` → renders.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat(routes): problem detail at /problem/<code> (singular, v1 parity)"
```

---

## Task 5: Contest detail route `contests/[key]` → `contest/[key]`

**Files:**
- Rename: `src/app/[locale]/contests/[key]/` → `src/app/[locale]/contest/[key]/` (moves `page.tsx`, `ContestPageContent.tsx`, child `stats/page.tsx`)
- Modify (navigation strings — all nav): `src/app/[locale]/HomePageContent.tsx`, `src/app/[locale]/blog/page.tsx`, `src/app/[locale]/contests/ContestsPageContent.tsx`, `src/app/[locale]/contest/[key]/ContestPageContent.tsx` (after rename), `src/app/[locale]/contest/[key]/stats/page.tsx` (after rename), `src/app/[locale]/contests/calendar/page.tsx`, `src/app/[locale]/ticket/[id]/page.tsx`, `src/app/[locale]/user/[username]/UserProfilePageContent.tsx`

**Interfaces:**
- Produces: contest detail at `/contest/<key>`; `/contests` list and `/contests/calendar` unchanged.

- [ ] **Step 1: Rename the directory**

```bash
git mv "src/app/[locale]/contests/[key]" "src/app/[locale]/contest/[key]"
```

`src/app/[locale]/contests/` still holds the list `page.tsx`, `ContestsPageContent.tsx`, and `calendar/`.

- [ ] **Step 2: Rewrite navigation links `` `/contests/${ `` → `` `/contest/${ `` **

All plural template-literal contest links are navigation (backend is `/contest/:key`). Does not affect `"/contests/calendar"` (no `${`) or the `/contests` list:

```bash
grep -rlE '`/contests/\$\{' src --include='*.tsx' --include='*.ts' \
  | xargs sed -i 's#`/contests/\${#`/contest/${#g'
```

- [ ] **Step 3: Verify**

Run:
```bash
grep -rnE '`/contests/\$\{' src --include='*.tsx' --include='*.ts'   # expect: no output
grep -rnE '/contests/calendar|href="/contests"|`/contests`' src --include='*.tsx' --include='*.ts' | head   # expect: still present
```

- [ ] **Step 4: Build**

Run: `npm run build`
Expected: passes.

- [ ] **Step 5: Manual check**

`/contests` list works; click a contest → `/contest/<key>`; `/contest/<key>/stats` renders; `/contests/calendar` still works.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat(routes): contest detail at /contest/<key> (singular, v1 parity)"
```

---

## Task 6: Submission detail route `submissions/[id]` → `submission/[id]`

**Files:**
- Rename: `src/app/[locale]/submissions/[id]/` → `src/app/[locale]/submission/[id]/`
- Modify (navigation only — exact sites): `src/app/[locale]/admin/moss/page.tsx:162`, `src/app/[locale]/admin/submissions/page.tsx:299`, `src/app/[locale]/problem/[code]/ProblemPageContent.tsx:95` (moved in Task 4), `src/app/[locale]/submissions/page.tsx:211`, `src/app/[locale]/submission/[id]/page.tsx:360` (after rename), `src/components/submission/SingleSubmissionWidget.tsx:123,154`
- **Do NOT modify:** `src/lib/api.ts:362` (`api.get(\`/submissions/${id1}/diff/${id2}\`)` — API call).

**Interfaces:**
- Produces: submission detail at `/submission/<id>`; `/submissions` list and `/submissions/diff` unchanged.

- [ ] **Step 1: Rename the directory**

```bash
git mv "src/app/[locale]/submissions/[id]" "src/app/[locale]/submission/[id]"
```

`src/app/[locale]/submissions/` still holds the list `page.tsx` and `diff/`.

- [ ] **Step 2: Rewrite navigation links, excluding `src/lib/api.ts`**

The only non-navigation match is in `src/lib/api.ts`; exclude it explicitly:

```bash
grep -rlE '`/submissions/\$\{' src --include='*.tsx' --include='*.ts' \
  | grep -v 'src/lib/api.ts' \
  | xargs sed -i 's#`/submissions/\${#`/submission/${#g'
```

This also rewrites the diff link in `submission/[id]/page.tsx:360` from `` `/submissions/${id}/diff?...` `` to `` `/submission/${id}/diff?...` ``, keeping it consistent with the detail route. (The `submissions/diff/[id]` route itself is intentionally left plural per spec.)

- [ ] **Step 3: Verify API call untouched, nav links renamed**

Run:
```bash
grep -rnE '`/submissions/\$\{' src --include='*.tsx' --include='*.ts'
```
Expected: exactly one line remains — `src/lib/api.ts:362` (the API diff call). No `.tsx` navigation matches.

- [ ] **Step 4: Build**

Run: `npm run build`
Expected: passes.

- [ ] **Step 5: Manual check**

`/submissions` list works; click a row → `/submission/<id>` renders; submitting a solution redirects to `/submission/<id>`.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat(routes): submission detail at /submission/<id> (singular, v1 parity)"
```

---

## Task 7: Blog → `/post/<id>-<slug>`

**Files:**
- Rename: `src/app/[locale]/blog/` → `src/app/[locale]/post/` ; then `post/[id]/` → `post/[slug]/`
- Modify: `src/app/[locale]/post/[slug]/page.tsx` (param `id`→`slug`, parse leading id, OG url)
- Modify: `src/app/[locale]/post/[slug]/BlogPageContent.tsx` (param usage; keep `api.get(\`/blog/${id}\`)` with parsed id; breadcrumb `href="/blog"`→`"/post"`)
- Modify (nav links `/blog/...` → `/post/<id>-<slug>`): `src/app/[locale]/post/page.tsx` (list, after rename), `src/app/[locale]/HomePageContent.tsx:250,292,332`, `src/app/[locale]/organization/[id]/blog/page.tsx:159`
- **Do NOT modify** any `api.get(\`/blog/${...}\`)` endpoint string or `src/lib/api.ts:275` (`/blog/${blogId}/vote`).

**Interfaces:**
- Consumes: `parseLeadingId` (Task 3).
- Produces: blog list at `/post`, blog post at `/post/<id>-<slug>`; API still called by numeric id.

- [ ] **Step 1: Rename directories**

```bash
git mv "src/app/[locale]/blog" "src/app/[locale]/post"
git mv "src/app/[locale]/post/[id]" "src/app/[locale]/post/[slug]"
```

- [ ] **Step 2: Update `post/[slug]/page.tsx` — param, API id, OG url**

Change the param type from `{ locale: string; id: string }` to `{ locale: string; slug: string }` in **both** `generateMetadata` and the default export, then derive the id. Replace the destructure + fetch usage:

In `generateMetadata`:
```ts
import { parseLeadingId } from '@/utils/route';
// ...
  const { slug } = await params;
  const id = parseLeadingId(slug);
  const post = await fetchBlogPost(id);
```
And change the OG url to the prefix-less canonical:
```ts
        url: `${SITE_URL}/post/${slug}`,
```
In the default `BlogPage` export, replace `const { id } = await params;` with:
```ts
  const { slug } = await params;
  const id = parseLeadingId(slug);
```
(`fetchBlogPost(id)` and `<ArticleJsonLd article={post} />` stay as-is.)

- [ ] **Step 3: Update `post/[slug]/BlogPageContent.tsx`**

Wherever it reads the route param for the API call, parse the leading id (keep the `api.get(\`/blog/${id}\`)` call — only the *source* of `id` changes). Update the "back to list" breadcrumb `href="/blog"` → `href="/post"`. Concretely:

```ts
import { parseLeadingId } from '@/utils/route';
// where params/{ id } was consumed:
const { slug } = await params;          // or use(params) in a client component
const id = parseLeadingId(slug);
const res = await api.get<BlogPostDetail>(`/blog/${id}`);   // endpoint unchanged
```
```tsx
<Link href="/post"> ... </Link>   // was href="/blog"
```

- [ ] **Step 4: Update navigation links to posts (list, home, org-blog)**

These build post links from a post object that includes `id` and `slug` (public blog API returns both). Rewrite each `` `/blog/${post.id}` `` navigation to `` `/post/${post.id}-${post.slug}` ``:
- `src/app/[locale]/post/page.tsx` (list card link, was `blog/page.tsx:131`)
- `src/app/[locale]/HomePageContent.tsx` lines 292 & 332 (post links); line 250 `href="/blog"` → `href="/post"`
- `src/app/[locale]/organization/[id]/blog/page.tsx:159`

Example transformation:
```tsx
// before
href={`/blog/${post.id}`}
// after
href={`/post/${post.id}-${post.slug}`}
```
And the section “View all” link:
```tsx
<Link href="/post"> ... </Link>   // was "/blog"
```

- [ ] **Step 5: Verify — no navigation `/blog` links remain; API calls intact**

Run:
```bash
grep -rnE 'href=.?[`"]/blog|push\(`/blog' src --include='*.tsx' --include='*.ts'   # expect: no output
grep -rnE 'api\.(get|post)\(`/blog/' src --include='*.tsx' --include='*.ts'        # expect: still present (unchanged)
```
Expected: first grep empty; second still shows the API calls.

- [ ] **Step 6: Build**

Run: `npm run build`
Expected: passes.

- [ ] **Step 7: Manual check**

`/post` lists posts; click one → `/post/<id>-<slug>` renders the correct post (verify a Unicode-slug post, e.g. id 93 → `/post/93-lấytiền`); voting still works.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "feat(routes): blog at /post/<id>-<slug> (v1 parity, stored slug)"
```

---

## Task 8: Organization detail `/organization/<id>-<slug>`

**Files:**
- Modify: `src/app/[locale]/organization/[id]/page.tsx` (parse leading id for API/mutations)
- Modify: `src/app/[locale]/organization/[id]/blog/page.tsx`, `src/app/[locale]/organization/[id]/manage/page.tsx` (parse leading id for API; nav to detail keeps slug)
- Modify (nav links → add `-<slug>`): `src/app/[locale]/organizations/page.tsx:115`, `src/app/[locale]/users/page.tsx:278`, `src/app/[locale]/organization/[id]/page.tsx:221` (`/organization/${org.id}/blog`)
- Keep folder name `organization/[id]` (param now carries `<id>-<slug>`).

**Interfaces:**
- Consumes: `parseLeadingId` (Task 3). Public org API returns `slug` (`organization.go` → `json:"slug"`).
- Produces: org detail at `/organization/<id>-<slug>`; API keyed by numeric id.

- [ ] **Step 1: Parse leading id in `organization/[id]/page.tsx`**

At the top of the component, split the id-with-slug param into a numeric id used for every API call. Replace:
```ts
const { id } = use(params);
```
with:
```ts
import { parseLeadingId } from '@/utils/route';
// ...
const { id: idParam } = use(params);
const id = parseLeadingId(idParam);
```
All existing `api.get(\`/organization/${id}\`)`, `/join`, `/leave`, and `queryKey: ['organization', id]` usages now correctly use the numeric id with no further edits.

- [ ] **Step 2: Same parse in `organization/[id]/blog/page.tsx` and `organization/[id]/manage/page.tsx`**

Apply the identical `parseLeadingId` treatment so their API calls (`/organization/${id}/...`) and query keys use the numeric id. Where these pages link back to the org detail (`href={\`/organization/${params.id}\`}` / `${id}` / `${organization.id}`), point them at the slugged URL when the org object is available:
```tsx
href={`/organization/${organization.id}-${organization.slug}`}
```
If only the numeric id is available in that scope (no org object), keep numeric (`/organization/${id}`) — the backend/page resolves by leading id either way; these management breadcrumbs are noindex.

- [ ] **Step 3: Add slug to outgoing org-detail navigation links**

In the list/user pages, the org object includes `slug`. Rewrite:
- `src/app/[locale]/organizations/page.tsx:115` — `href={\`/organization/${org.id}\`}` → `href={\`/organization/${org.id}-${org.slug}\`}`
- `src/app/[locale]/users/page.tsx:278` — same transformation (`org.id` → `org.id}-${org.slug`)
- `src/app/[locale]/organization/[id]/page.tsx:221` — `href={\`/organization/${org.id}/blog\`}` → `href={\`/organization/${org.id}-${org.slug}/blog\`}`

- [ ] **Step 4: Verify slug is present on the list items**

Run: `grep -rnE "interface .*Organization|slug" src/types` and confirm the org list/summary type used by `organizations/page.tsx` and `users/page.tsx` includes `slug`. If missing, add `slug: string;` to that type (the API already returns it). Do not change the API layer.

- [ ] **Step 5: Build**

Run: `npm run build`
Expected: passes (no TS error about `org.slug` being unknown).

- [ ] **Step 6: Manual check**

`/organizations` → click an org → URL `/organization/<id>-<slug>` (e.g. `/organization/1-itcla`), page loads the right org; join/leave still work; visiting `/organization/1` (bare id) also still resolves.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat(routes): organization detail at /organization/<id>-<slug> (v1 parity)"
```

---

## Task 9: Correct the SEO surface (canonical, JSON-LD, hreflang, sitemap, robots)

**Files:**
- Modify: `src/lib/seo.ts`
- Modify: `src/app/[locale]/layout.tsx:30-36`
- Modify: `src/app/sitemap.ts`
- Modify: `src/app/robots.ts`

**Interfaces:**
- Consumes: the new route shapes from Tasks 4-8.
- Produces: prefix-less, v1-matching canonical URLs, JSON-LD `url`s, sitemap entries, and single-URL hreflang.

- [ ] **Step 1: `src/lib/seo.ts` — prefix-less canonical + singular/slug JSON-LD urls**

- `generateCanonicalUrl`:
```ts
export function generateCanonicalUrl(path: string, locale: string): string {
  return `${SITE_URL}${path}`;   // no locale prefix (single URL per page, like v1)
}
```
- `generateContestJsonLd` url (was `/contests/${contest.key}`):
```ts
    url: `${SITE_URL}/contest/${contest.key}`,
```
- `generateArticleJsonLd` — accept a slug and emit `/post/<id>-<slug>`. Extend the arg type with `slug: string;` and set:
```ts
    url: `${SITE_URL}/post/${article.id}-${article.slug}`,
    mainEntityOfPage: {
      '@type': 'WebPage',
      '@id': `${SITE_URL}/post/${article.id}-${article.slug}`,
    },
```
(The `ArticleJsonLd` component is fed the full `post` object, which already includes `slug`.)
- `generateWebSiteJsonLd` search target (was `/en/problems`):
```ts
        urlTemplate: `${SITE_URL}/problems?q={search_term_string}`,
```

- [ ] **Step 2: `src/app/[locale]/layout.tsx` — drop per-locale hreflang**

With one URL per page (language via cookie), distinct `/en` `/vi` alternates are wrong. Replace the `alternates` block (lines ~30-36):
```ts
  alternates: {
    canonical: siteUrl,
  },
```
(Remove the `languages: { 'en-US': '/en', 'vi-VN': '/vi' }` map.)

- [ ] **Step 3: `src/app/sitemap.ts` — one prefix-less URL set, v1 names**

Replace the per-locale loop with a single prefix-less set, and rename `/blog` → `/post`:
```ts
const STATIC_ROUTES = [
  { route: '', priority: 1.0, changefreq: 'daily' as const },
  { route: '/problems', priority: 0.9, changefreq: 'daily' as const },
  { route: '/contests', priority: 0.9, changefreq: 'hourly' as const },
  { route: '/contests/calendar', priority: 0.7, changefreq: 'hourly' as const },
  { route: '/users', priority: 0.8, changefreq: 'daily' as const },
  { route: '/ratings', priority: 0.8, changefreq: 'daily' as const },
  { route: '/organizations', priority: 0.7, changefreq: 'weekly' as const },
  { route: '/submissions', priority: 0.7, changefreq: 'hourly' as const },
  { route: '/post', priority: 0.8, changefreq: 'daily' as const },
  { route: '/stats', priority: 0.6, changefreq: 'daily' as const },
  { route: '/login', priority: 0.3, changefreq: 'monthly' as const },
  { route: '/register', priority: 0.3, changefreq: 'monthly' as const },
];

function generateSitemapEntries(): MetadataRoute.Sitemap {
  return STATIC_ROUTES.map(({ route, priority, changefreq }) => ({
    url: `${SITE_URL}${route}`,
    lastModified: new Date(),
    changeFrequency: changefreq,
    priority,
  }));
}
```
(Remove the `LOCALES` loop and the `/${locale}` prefix.)

- [ ] **Step 4: `src/app/robots.ts` — align allow list**

Change the `/blog` entry to `/post` in the `allow` array; leave the rest (already prefix-less).

- [ ] **Step 5: Build**

Run: `npm run build`
Expected: passes.

- [ ] **Step 6: Verify the SEO output**

Run: `npm run dev`, then:
```bash
curl -s http://localhost:3000/sitemap.xml | grep -E '/(en|vi)/' | head   # expect: no output (no locale-prefixed urls)
curl -s http://localhost:3000/sitemap.xml | grep -E '/post|/problems' | head   # expect: prefix-less v1-style urls
```
Also view-source a problem page: canonical + JSON-LD `url` are prefix-less and singular; an article page's JSON-LD `url` is `/post/<id>-<slug>`.

- [ ] **Step 7: Commit**

```bash
git add src/lib/seo.ts "src/app/[locale]/layout.tsx" src/app/sitemap.ts src/app/robots.ts
git commit -m "feat(seo): prefix-less canonical/JSON-LD/sitemap, v1 URL parity"
```

---

## Task 10: Final verification sweep

**Files:** none (verification only).

- [ ] **Step 1: No stale plural/blog navigation links anywhere**

Run:
```bash
grep -rnE '`/problems/\$\{|`/contests/\$\{' src --include='*.tsx' --include='*.ts'   # expect: empty
grep -rnE '`/submissions/\$\{' src --include='*.tsx' --include='*.ts'                # expect: only src/lib/api.ts
grep -rnE 'href=.?[`"]/blog|push\(`/blog' src --include='*.tsx' --include='*.ts'     # expect: empty
grep -rnE '`/organization/\$\{[a-zA-Z.]+\}`' src --include='*.tsx' --include='*.ts'  # review: bare-id org links (management breadcrumbs only)
```

- [ ] **Step 2: No leftover locale-prefixed URL literals or `/en`,`/vi` assumptions**

Run:
```bash
grep -rnE "['\"\`]/(en|vi)/" src --include='*.tsx' --include='*.ts'   # expect: empty
```

- [ ] **Step 3: Full production build + tests**

Run: `npm run build && npx jest`
Expected: build succeeds; `parseLeadingId` tests pass.

- [ ] **Step 4: End-to-end route parity checklist (dev server)**

Confirm each resolves with **no** locale prefix and renders:
`/` · `/problems` · `/problem/<code>` · `/problem/<code>/editorial` · `/problems/random` · `/contests` · `/contest/<key>` · `/contest/<key>/stats` · `/contests/calendar` · `/submissions` · `/submission/<id>` · `/users` · `/user/<name>` · `/organizations` · `/organization/<id>-<slug>` · `/post` · `/post/<id>-<slug>` · `/tickets` · `/ticket/<id>`.
Language toggle keeps the URL unchanged and does not revert.

- [ ] **Step 5: Final commit (if the sweep required fixes)**

```bash
git add -A
git commit -m "chore(routes): final URL-parity verification fixes"
```

---

## Self-Review Notes

- **Spec coverage:** locale prefix (Task 1-2), problem/contest/submission singular (Tasks 4-6), blog `/post/<id>-<slug>` with stored slug + id lookup (Task 7), organization `<id>-<slug>` (Task 8), setlang/switcher (Task 2), `seo.ts`/`layout`/`sitemap`/`robots` (Task 9), API/admin/list-route preservation (Global Constraints + per-task excludes). All spec items map to a task.
- **No 301s / dynamic sitemap entries:** intentionally out of scope per spec (v2 not indexed; dynamic sitemap is a follow-up).
- **Risk note:** the `submissions/[id]/diff` vs `submissions/diff/[id]` inconsistency is pre-existing in v2 and left as-is; only the `/submissions/` → `/submission/` prefix on that link is normalized.
