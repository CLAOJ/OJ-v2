# v2 Inline PDF Statements — Design

**Date:** 2026-07-21
**Status:** Approved design, pending spec review
**Scope:** Render PDF-based problem statements inline on the v2 (`claoj-web`) problem page, at parity with v1. Frontend-only.

---

## 1. Goal

When a problem's statement is a PDF, the v2 problem page currently shows an **empty
body box** — the statement never appears inline. v1 embeds the PDF right where the
statement goes (an `<object type="application/pdf">` with a download fallback).

This work brings that behavior to v2: **when a problem has a PDF, render it inline in
the statement body**, using a themed react-pdf/pdf.js viewer, with a graceful
download/open-in-new-tab fallback.

Put plainly: **a PDF problem in v2 shows its statement in the body, not a blank box.**

---

## 2. Current state (context)

The Go backend **already supports PDF problems** — no backend change is needed:

- `Problem.PdfURL` (`claoj/models/problem.go:52`, DB column `pdf_url`) holds the PDF
  pointer. A problem "has a PDF" when `pdf_url` is non-empty.
- `GET /api/v2/problem/:code` (`claoj/api/v2/problem.go:155-237`) returns `pdf_url`
  in its JSON payload (line 235), alongside the raw markdown `description`.
- `GET /api/v2/problem/:code/pdf` (`claoj/api/v2/problem.go:241-306`) streams the
  file `application/pdf; inline`, enforcing `auth.CanViewProblem` (403 otherwise),
  path-traversal-guarded, with a 10 MB cap.

The gap is entirely in the Next.js frontend:

- The statement body at
  `claoj-web/src/app/[locale]/problems/[code]/ProblemPageContent.tsx:294-296`
  **unconditionally** renders `problem.description` as Markdown via `MathRenderer`.
  For PDF problems `description` is empty, so the box is blank.
- A sidebar "PDF Statement" button + modal already exist
  (`ProblemPageContent.tsx:56`, `:195-207`, `:383-417`). The modal shows a raw
  `<iframe src="/api/problem/${code}/pdf">`. It is the only way to read a PDF problem
  today, and the PDF is never shown inline in the body.
- Helpers already exist but are unused: `problemPdfApi.getPdfUrl(code)` /
  `hasPdf(problem)` at `claoj-web/src/lib/api.ts:316-322`.
- No PDF library is installed. Stack: **Next.js 16.1.6 (Turbopack) + React 19.2.3**.

---

## 3. Decisions

Each row was chosen deliberately during brainstorming; the rationale is the
trade-off that won.

| Decision | Choice | Why |
|---|---|---|
| Presentation | **Inline in the statement body** (v1 parity), keep the sidebar button + modal | Matches v1; the body is where readers look. |
| Render engine | **react-pdf v10 (pdf.js)** | User choice: themed viewer chrome + page controls, consistent with the dark UI (vs. the browser's native PDF chrome). |
| Reading UX | **Continuous scroll of all pages** | Closest to v1's single scrollable embed; simplest to read. |
| PDF fetch | **Blob via the axios `api` client** (`responseType: 'blob'`) | Reuses auth/CSRF/refresh interceptors + cookie session (endpoint enforces `CanViewProblem`); avoids a second un-authenticated fetch path. PDFs are ≤10 MB, so in-memory is fine. |
| Modal | **Fold the existing modal into the shared viewer** | One PDF code path instead of two divergent renderers (iframe vs react-pdf). |
| SSR | **`next/dynamic(..., { ssr: false })`** | react-pdf/pdf.js touch browser-only globals; never run it during SSR. |
| Backend | **No change** | `pdf_url` + `/pdf` endpoint already exist and work. |

---

## 4. Components & data flow

### 4.1 New: `PdfStatementViewer`

`claoj-web/src/components/ui/PdfStatementViewer.tsx` — a self-contained, client-only
(`'use client'`) viewer, consumed via `next/dynamic(..., { ssr: false })`.

- **What it does:** given a problem `code`, fetches its PDF and renders the pages.
- **How you use it:** `<PdfStatementViewer code={problem.code} />`.
- **Depends on:** the axios `api` client, `problemPdfApi`, `react-pdf`.

Responsibilities:

1. Fetch PDF as a blob: `api.get('/problem/${code}/pdf', { responseType: 'blob' })`
   → `URL.createObjectURL(blob)`. Memoize the `file={{ url }}` object (react-pdf
   re-renders if the reference changes). Revoke the object URL on unmount.
2. Render `<Document file={file}>` with all pages stacked (`<Page pageNumber={n} />`
   for `1..numPages`), fit-to-width via a container ref + `ResizeObserver`, inside a
   scrollable container with a dark gutter behind the (white) PDF pages.
3. Toolbar: page count (`Page X of Y` or total), zoom in/out, **Download** and
   **Open in new tab** — both `href = problemPdfApi.getPdfUrl(code)`.
4. States, none of which is ever a blank box:
   - **loading** → spinner/skeleton;
   - **error** (blob fetch 403/404/network, or react-pdf `onLoadError`, or corrupt)
     → fallback card with Download + Open-in-new-tab links and a short message.
5. Module-level setup: pdf.js worker
   `pdfjs.GlobalWorkerOptions.workerSrc = new URL('pdfjs-dist/build/pdf.worker.min.mjs', import.meta.url).toString()`,
   plus `import 'react-pdf/dist/Page/TextLayer.css'` and `AnnotationLayer.css`.

### 4.2 Integration in `ProblemPageContent`

At the body box (`ProblemPageContent.tsx:294-296`), branch like v1:

- Render the `MathRenderer` description **only when `problem.description` is
  non-empty**.
- When `problem.pdf_url` is set, render `<PdfStatementViewer code={code} />` — below
  the description if both exist, alone if `description` is empty.
- When neither is present, keep the existing (empty) render / a "no statement" note.

The sidebar button + modal are kept, but the modal's raw `<iframe>` is replaced with
`<PdfStatementViewer />` so there is a single PDF code path.

### 4.3 Data flow

```
ProblemPageContent
  └─ GET /problem/:code  ──►  { description, pdf_url, ... }
        pdf_url set?
          ├─ no  → MathRenderer(description)                (unchanged)
          └─ yes → <PdfStatementViewer code>
                     └─ api.get('/problem/:code/pdf', blob) ──► objectURL
                          └─ <Document file={{url}}> → <Page> × N
                     (Download / Open-in-new-tab → problemPdfApi.getPdfUrl(code))
```

---

## 5. Dependencies & build config

- Add `react-pdf@^10` (+ its `pdfjs-dist@^5` peer). Pin exact versions at install.
- Worker via `import.meta.url` (Turbopack-compatible on Next 16); import react-pdf's
  `TextLayer.css` + `AnnotationLayer.css`.
- **Contingencies (apply only if the build surfaces them):**
  - If Turbopack errors on pdfjs's optional `canvas` dep, alias `canvas → false` in
    `next.config.ts` (Turbopack `resolveAlias` / webpack `resolve.alias`).
  - If Vietnamese glyphs render as boxes (non-embedded fonts), ship pdf.js cmaps to
    `public/` and set react-pdf `options.cMapUrl`. Vietnamese PDFs usually embed
    fonts, so this is likely unneeded.

---

## 6. i18n & tests

- Add viewer strings to `src/i18n/en.json` + `vi.json` (loading, error/fallback,
  download, open in new tab, page indicator, zoom). Replace the hardcoded
  `"PDF Statement"` string (`ProblemPageContent.tsx:203`) with an i18n key
  (an existing `Problems…pdfTab` key already reads "PDF Statement" / "Đề bài PDF").
- Jest + React Testing Library, with `react-pdf` and the `api` client **mocked**
  (pdf.js cannot render a canvas in jsdom):
  - viewer: loading → pages on success; fallback card (correct download/open hrefs)
    on fetch error;
  - `ProblemPageContent`: renders the viewer when `pdf_url` is set, `MathRenderer`
    when only `description` is present.

---

## 7. Assumptions & non-goals

**Assumptions**

- `/api` is reverse-proxied to the Go backend in every running environment. The app
  already depends on this (the problem detail itself loads via `/api`), and the PDF
  endpoint sits under the same prefix. Where a dev setup lacks the proxy, the blob
  fetch fails and the viewer shows its fallback card rather than a blank box.
- Statement PDFs are within the backend's 10 MB cap.

**Non-goals (YAGNI)**

- No in-PDF text search, thumbnails, annotation editing, or printing UI beyond the
  browser default.
- No server-side PDF rendering (v1's disabled `ProblemPdfView`/puppeteer path is not
  reproduced).
- No backend changes.

---

## 8. Success criteria

- Opening a problem whose `pdf_url` is set shows the PDF inline in the statement
  body (pages visible, scrollable), not an empty box.
- Download and Open-in-new-tab work and point at `/api/problem/:code/pdf`.
- A problem with only a markdown `description` is unchanged.
- A problem with both shows the description then the PDF.
- When the PDF can't be fetched, the viewer shows a fallback with working links, not
  a blank box.
- `next build` (Turbopack) succeeds; new + existing tests pass.
