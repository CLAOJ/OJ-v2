# claoj-web

Next.js 16 (App Router, React 19, TypeScript, Tailwind 4) front end for CLAOJ
v2. It talks to the Go API in `../claoj` over same-origin `/api`; there is no
separate API host in the browser.

## Run

```bash
npm install
npm run dev        # http://localhost:3000/vi
```

| Script | |
|---|---|
| `npm run dev` / `build` / `start` | Next dev / production build / serve |
| `npm test` | Jest (`__tests__/`) |
| `npm run lint` | ESLint (advisory in CI) |
| `npm run i18n:check` | every key present in both locales |
| `npm run i18n:usage` | unused/missing translation keys |
| `npm run ui:check` | banned decorative UI patterns |

Environment variables are optional and documented in the root
[README](../README.md#frontend-environment).

## Layout

```
src/app/[locale]/     locale-prefixed pages (/vi, /en); admin/ is the admin app
src/components/       shared UI, admin editors, layout
src/lib/api.ts        axios client: base URL, cookie auth, CSRF header, retries
src/lib/              API modules, server fetch helpers, SEO
src/i18n/             en.json / vi.json and next-intl config
src/hooks/            WebSocket and data hooks
__tests__/            Jest suites, including routing/dynamicSegments.test.ts
```

All user-visible strings come from `src/i18n/{en,vi}.json` — add both.
Authenticated pages depend on httpOnly cookies; mutations are blocked without
the `csrf_token` cookie echoed as `X-CSRF-Token` (handled in `lib/api.ts`).

## Public pages

`/` home feed, `/problems` + `/problem/<code>`, `/contests` +
`/contest/<key>` (ranking, submissions, clarifications), `/submissions`,
`/submission/<id>`, `/users` + `/user/<username>`, `/organizations`,
`/ratings`, `/contributors`, `/runtimes`, `/status`, `/post/<slug>`,
`/ticket`, `/settings`, auth flows (`/login`, `/register`,
`/forgot-password`, email verification), and a CMS catch-all.

## Admin (`/admin`)

Users, groups/roles, problems, contests, submissions, organizations +
requests, judges, languages + limits, comments + locks, blog posts, tickets,
taxonomy, licenses, flat pages, redirects, navigation bars, name styles,
classes, audit logs, plagiarism, points votes, registrations, misc config.
Most editors mirror the Django admin fields; the API enforces Django
permission codenames.

## Notes

- Route segments under one parent must use one dynamic name (`[id]` vs
  `[slug]` breaks the production build). The guard is
  `__tests__/routing/dynamicSegments.test.ts`; still smoke-test `npm run start`
  on changed routes.
- `npm run build` alone is not a deploy check — dynamic routes can pass the
  build and 500 at runtime.
- Date rendering is pinned to Vietnam time to keep SSR and hydration equal.