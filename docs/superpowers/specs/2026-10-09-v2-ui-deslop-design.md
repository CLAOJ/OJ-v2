# v2 UI De-slop Pass — Design

**Date:** 2026-10-09
**Scope:** `claoj-web/src` — all public and admin pages, styling only
**Direction:** dense judge tool (v1 / DMOJ / Codeforces feel)

## Goal

Remove the generic "AI-generated" visual language from claoj-web v2 so it reads
as a dense, utilitarian online judge consistent with v1. The v1-derived color
tokens in `globals.css` are already correct; the slop lives in per-page inline
Tailwind classes.

### Baseline (measured 2026-10-09, `src/`)

| Marker | Files | Hits |
|---|---|---|
| `font-black` | 28 | 326 |
| `text-[8-10px]` | 25 | 182 |
| `uppercase` | 41 | 252 |
| `tracking-widest` / `tracking-[0.Nem]` | 25 | 166 |
| `rounded-2xl` / `rounded-3xl` | 64 | 209 |
| `rounded-[Nrem]` | 19 | 77 |
| `shadow-lg/xl/2xl` | 43 | 87 |
| `hover:scale` / `hover:-translate` | 11 | 22 |
| `backdrop-blur` | 7 | 7 |
| `bg-gradient` | 6 | 7 |

Worst files: `UserProfilePageContent.tsx` (39), `ProblemPageContent.tsx` (34),
`admin/tickets/page.tsx` (22), `ContestPageContent.tsx` (21), `post/page.tsx` (18),
`submission/[id]/page.tsx` (17).

## Constraints

- Styling only: no layout restructuring, no changes to routes, component props,
  i18n keys, or behavior.
- Keep the v1 color tokens, fonts, and both light/dark themes.
- Semantic color stays: rating colors, verdict colors (AC/WA/RE/TLE…),
  difficulty colors, admin tag palette, status dots.
- Untouched: skeletons, spinners, live-verdict animations, Monaco, KaTeX, PDF
  viewer, Recharts data series colors, anything inside `.site-nav`.

## Approach

Rule-based class sweep (approach A). Unambiguous swaps are applied
mechanically; context-dependent ones (weight on titles vs. cells, decorative vs.
semantic icon color, spacing) are edited by hand. Shared primitives
(`Panel`/`StatItem`) are explicitly out of scope; revisit if repeated markup
remains painful after this pass.

## Rules

### Shared components and tokens

| Target | Change |
|---|---|
| `globals.css` `--radius` | `0.5rem` → `0.375rem` |
| `components/ui/Badge.tsx` | base `rounded-full px-2.5` → `rounded px-1.5` |
| `components/ui/Table.tsx` | head/cell `h-12 px-4` → `h-9 px-3` |
| `components/ui/Button.tsx` | default `h-10` → `h-9`; `lg` `h-11 px-8` → `h-10 px-5` |
| `components/ui/Input.tsx` | `h-10` → `h-9` |
| `components/ui/Dialog.tsx` | enter/exit animation: opacity only (drop `scale` and `y`) |

### Page class mapping

| Slop | Replacement | Mode |
|---|---|---|
| `font-black` | `font-semibold` on h1–h3 and headline numbers; `font-medium` elsewhere | hand |
| `font-bold` on body text, labels, buttons | `font-medium` | hand |
| `text-[8-10px]` + `uppercase` + `tracking-wider/widest` labels | `text-xs text-muted-foreground`, sentence case | mechanical + eyeball |
| `tracking-tighter` / `tracking-tight` on numbers | remove; add `tabular-nums` where numbers align in columns | mechanical |
| `rounded-2xl` / `rounded-3xl` / `rounded-[Nrem]` | `rounded-md` (avatars included) | mechanical |
| `rounded-full` on non-avatar, non-dot elements | `rounded` | hand |
| `shadow-md/lg/xl/2xl` on static surfaces | remove (border carries it); keep on popover, dropdown, dialog, toast | hand |
| colored shadows (`shadow-primary/20` etc.) | remove | mechanical |
| `bg-gradient-*` tiles/overlays | flat `bg-muted`, or remove overlay | hand |
| `hover:scale-*`, `group-hover:scale-*`, `active:scale-*`, `hover:-translate-*` | remove; hover = color change only | mechanical |
| `backdrop-blur*` with `bg-card/80` | solid `bg-card`; modal scrims become plain `bg-black/50` | mechanical |
| page-level `animate-in fade-in duration-*` | remove | hand |
| decorative per-icon colors (`text-amber-500` next to a label) | `text-muted-foreground` | hand |
| `p-8` / `p-10` / `gap-8` / `space-y-8` on cards and sections | one step down (`p-4`/`p-5`, `gap-4`, `space-y-4`) | hand |

Rule of thumb for icon color: if removing the color loses information (verdict,
rating, status), it stays; otherwise it goes muted.

## Batches

One commit per batch, in traffic order, on branch `ui/de-slop`. The regression guard (batch 7 below) is built first in the implementation plan so it can serve as each batch's red/green check.

1. **Shared base** — token + Badge, Table, Button, Input, Dialog. Screenshot
   checkpoint before continuing.
2. **Core judge flow** — home, problems list, problem page, submissions list,
   submission detail, submission diff.
3. **Contests** — contests list, calendar, contest page, contest stats, ratings.
4. **People & orgs** — user profile, users, organizations, organization page,
   organization blog/manage.
5. **Content, auth, misc** — posts, login, register, forgot-password,
   resend-verification, verify-email, settings, notifications, tickets, stats;
   shared layout pieces (Footer, MobileMenu, NotificationBell, comments).
6. **Admin** — all `admin/**` pages and `components/admin/**`.
7. **Regression guard** — `claoj-web/scripts/check-ui-slop.mjs` + `npm run
   ui:check`, modeled on `check-i18n.mjs`. Greps `src/**/*.tsx` for the banned
   patterns below; exits non-zero on any hit. A line containing
   `ui-check-allow` is skipped.

### Banned patterns for `ui:check`

```
font-black
text-\[(8|9|10)px\]
tracking-(widest|\[0\.\d+em\])
rounded-(2xl|3xl|\[[\d.]+rem\])
(hover|active):-?(scale|translate)-      (also matches group-hover:)
bg-(gradient|linear|radial|conic)-
backdrop-blur
shadow-(2xl|[a-z]+(-\d+)?/\d+)
animate-in
```

(`uppercase` is not banned outright: it stays legitimate for verdict codes like
`AC`/`WA` and short table headers. `shadow-lg/xl` is not banned because floating
layers keep it; reviewed by hand instead.)

## Verification

Per batch:

- `npm run lint`, `npm test`, `npm run build` in `claoj-web` — all must pass.
- Marker grep over the batch's files — banned patterns at zero (or allowlisted).
- Playwright screenshots of the batch's key pages against `next dev`, light +
  dark, 1440px and 390px, before and after, saved to the session scratchpad.
  Review for density, leftover decoration, clipped text, broken spacing, and
  contrast.

Final key-page set (13): home, problems list, a problem, submissions list, a
submission, contests list, a contest, ratings, a user profile, an organization,
login, admin dashboard, one admin table page.

If the API configured in `.env.local` is unreachable locally, pages are checked
in their empty/skeleton states and the report names which pages lacked real data.

## Done criteria

- `npm run ui:check` passes on all of `src`.
- Lint, tests, and build pass.
- Before/after screenshots exist for the 13 key pages in both themes.
- The diff touches only Tailwind class strings (in `className` or in style-class
  literals such as `color: 'text-amber-500'` in stat arrays), the five shared components,
  `globals.css`, `package.json` (script entry), and the new check script.

## Out of scope

- Layout restructuring (card grids → tables, sidebar redesign).
- New shared primitives (`Panel`, `SectionHeader`, `StatItem`).
- Palette, font, or icon-set changes.
