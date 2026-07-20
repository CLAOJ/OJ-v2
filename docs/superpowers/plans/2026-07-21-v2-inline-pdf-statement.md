# v2 Inline PDF Statements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a problem has a PDF statement (`pdf_url` set), render the PDF inline in the v2 (`claoj-web`) problem body instead of an empty box — at parity with v1.

**Architecture:** Frontend-only. A new client-only `PdfStatementViewer` component fetches the PDF as a blob through the existing authenticated axios client and renders it with react-pdf (pdf.js). The problem page branches to the viewer when `pdf_url` is present; the pre-existing sidebar button + modal are kept and re-point at the same viewer. The Go backend is unchanged — it already returns `pdf_url` and streams the file at `GET /api/problem/:code/pdf`.

**Tech Stack:** Next.js 16.1.6 (Turbopack) · React 19.2.3 · TypeScript · react-pdf v10 (pdf.js) · next-intl · @tanstack/react-query · Jest + ts-jest + React Testing Library (jsdom).

## Global Constraints

- Work on branch `feat/web-inline-pdf-statement` (already created). All paths below are relative to `claoj-web/` unless noted.
- No backend changes. Do not touch `claoj/` (the Go API).
- The PDF is fetched via the shared axios client `api` (default export of `@/lib/api`) with `{ responseType: 'blob' }` — never a second un-authenticated fetch path. The endpoint enforces `CanViewProblem` and caps at 10 MB.
- react-pdf's pdf.js worker uses `import.meta.url` and react-pdf ships `.css` files; both are isolated in a `pdfSetup` side-effect module so Jest (ts-jest, CommonJS) never transforms them.
- The viewer is imported into the (server-prerendered) page via `next/dynamic(..., { ssr: false })` — react-pdf must never run during SSR.
- User-facing strings use i18n keys under the `Problems` namespace; add both `en.json` and `vi.json` and keep them in parity (`npm run i18n:check`).
- Indentation: `.tsx` files use 4-space indent; `src/i18n/*.json` use 2-space indent.

---

## File Structure

| File | Responsibility |
|---|---|
| `src/components/ui/pdfSetup.ts` | **Create.** Side-effect module: configures the pdf.js worker + imports react-pdf layer CSS. Isolated so tests can mock it. |
| `src/components/ui/PdfStatementViewer.tsx` | **Create.** Client-only viewer: fetch blob → render pages, toolbar (zoom/download/open), loading + error-fallback states. |
| `__tests__/components/PdfStatementViewer.test.tsx` | **Create.** Loading / success / error-fallback tests with react-pdf, pdfSetup, and the api client mocked. |
| `src/app/[locale]/problems/[code]/ProblemPageContent.tsx` | **Modify.** Dynamic-import the viewer; branch the statement body on `pdf_url`; swap the modal `<iframe>` for the viewer; i18n the sidebar label. |
| `src/i18n/en.json`, `src/i18n/vi.json` | **Modify.** Add `Problems.pdfViewer` string block (parity). |
| `package.json` / `package-lock.json` | **Modify** via `npm install` (react-pdf + pinned pdfjs-dist). |
| `next.config.ts` | **Modify only if** the Turbopack build errors on pdfjs's optional `canvas` dep (Task 4 contingency). |

---

### Task 1: Add react-pdf dependency and pdf.js setup module

**Files:**
- Modify: `package.json`, `package-lock.json` (via npm)
- Create: `src/components/ui/pdfSetup.ts`

**Interfaces:**
- Produces: the `react-pdf` module (exports `Document`, `Page`, `pdfjs`) and a side-effect module `@/components/ui/pdfSetup` (no exports; sets `pdfjs.GlobalWorkerOptions.workerSrc` and imports layer CSS).

- [ ] **Step 1: Install react-pdf, then pin its matching pdfjs-dist**

Run (from `claoj-web/`):
```bash
npm install react-pdf@^10.1.0
npm ls pdfjs-dist
```
Note the resolved `pdfjs-dist` version printed by `npm ls` (react-pdf depends on it). Pin that exact version explicitly so the worker import path is stable:
```bash
npm install pdfjs-dist@<version-from-npm-ls>
```
Expected: `package.json` `dependencies` now lists both `react-pdf` and `pdfjs-dist`; `npm ls pdfjs-dist` shows a single deduped version.

- [ ] **Step 2: Create the pdf.js setup module**

Create `src/components/ui/pdfSetup.ts`:
```ts
// Side-effect module: configures pdf.js for react-pdf.
// Isolated from the viewer so Jest can mock it — the `import.meta.url` worker
// reference and the CSS imports below cannot be transformed by ts-jest (CommonJS).
import { pdfjs } from 'react-pdf';
import 'react-pdf/dist/Page/TextLayer.css';
import 'react-pdf/dist/Page/AnnotationLayer.css';

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
    'pdfjs-dist/build/pdf.worker.min.mjs',
    import.meta.url,
).toString();
```

- [ ] **Step 3: Typecheck**

Run: `npx tsc --noEmit`
Expected: PASS (no errors). If `tsc` reports a missing types package for react-pdf, it ships its own types — re-check the version installed; do not add `@types/react-pdf`.

- [ ] **Step 4: Commit**

```bash
git add package.json package-lock.json src/components/ui/pdfSetup.ts
git commit -m "feat(web): add react-pdf dependency and pdf.js worker setup"
```

---

### Task 2: PdfStatementViewer component (TDD)

**Files:**
- Create: `src/components/ui/PdfStatementViewer.tsx`
- Test: `__tests__/components/PdfStatementViewer.test.tsx`

**Interfaces:**
- Consumes: `api` (default export, `.get(url, { responseType: 'blob' })`) and `problemPdfApi.getPdfUrl(code)` from `@/lib/api`; `Document`, `Page` from `react-pdf`; the `@/components/ui/pdfSetup` side effect.
- Produces: `export default function PdfStatementViewer(props: { code: string; heightClass?: string }): JSX.Element`. Default export, default `heightClass = 'max-h-[80vh]'`.

- [ ] **Step 1: Write the failing test**

Create `__tests__/components/PdfStatementViewer.test.tsx`:
```tsx
import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import PdfStatementViewer from '@/components/ui/PdfStatementViewer';
import api from '@/lib/api';

// Isolate pdf.js worker + CSS side effects (import.meta.url / CSS can't be
// transformed by ts-jest).
jest.mock('@/components/ui/pdfSetup', () => ({}));

// Stub react-pdf: Document reports a 2-page load; Page renders a marker.
jest.mock('react-pdf', () => ({
    Document: ({ children, onLoadSuccess }: { children: React.ReactNode; onLoadSuccess?: (d: { numPages: number }) => void }) => {
        React.useEffect(() => { onLoadSuccess?.({ numPages: 2 }); }, [onLoadSuccess]);
        return <div data-testid="pdf-document">{children}</div>;
    },
    Page: ({ pageNumber }: { pageNumber: number }) => <div data-testid="pdf-page">page {pageNumber}</div>,
}));

// Mock the API client (default export) + the URL helper (named export).
jest.mock('@/lib/api', () => ({
    __esModule: true,
    default: { get: jest.fn() },
    problemPdfApi: { getPdfUrl: (code: string) => `http://test/api/problem/${code}/pdf` },
}));

const mockedGet = api.get as jest.Mock;

beforeAll(() => {
    // jsdom lacks object-URL APIs.
    (URL as unknown as { createObjectURL: unknown }).createObjectURL = jest.fn(() => 'blob:mock');
    (URL as unknown as { revokeObjectURL: unknown }).revokeObjectURL = jest.fn();
});

beforeEach(() => { mockedGet.mockReset(); });

test('shows a loading indicator while the PDF is being fetched', () => {
    mockedGet.mockReturnValue(new Promise(() => {})); // never resolves
    render(<PdfStatementViewer code="abc" />);
    expect(screen.getByText('pdfViewer.loading')).toBeInTheDocument();
});

test('renders the PDF pages after a successful fetch', async () => {
    mockedGet.mockResolvedValue({ data: new Blob(['%PDF'], { type: 'application/pdf' }) });
    render(<PdfStatementViewer code="abc" />);
    const pages = await screen.findAllByTestId('pdf-page');
    expect(pages).toHaveLength(2);
    expect(screen.getByText('pdfViewer.pageCount')).toBeInTheDocument();
});

test('shows a fallback with download + open links when the fetch fails', async () => {
    mockedGet.mockRejectedValue(new Error('403'));
    render(<PdfStatementViewer code="abc" />);
    await waitFor(() => expect(screen.getByText('pdfViewer.error')).toBeInTheDocument());
    const links = screen.getAllByRole('link');
    expect(links.length).toBeGreaterThanOrEqual(2);
    links.forEach((l) => expect(l).toHaveAttribute('href', 'http://test/api/problem/abc/pdf'));
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npm test -- PdfStatementViewer`
Expected: FAIL — cannot find module `@/components/ui/PdfStatementViewer`.

- [ ] **Step 3: Implement the viewer**

Create `src/components/ui/PdfStatementViewer.tsx`:
```tsx
'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { Document, Page } from 'react-pdf';
import { useTranslations } from 'next-intl';
import { Loader2, FileText, Download, ArrowUpRight, ZoomIn, ZoomOut } from 'lucide-react';
import api, { problemPdfApi } from '@/lib/api';
import '@/components/ui/pdfSetup';

interface PdfStatementViewerProps {
    code: string;
    heightClass?: string;
}

export default function PdfStatementViewer({ code, heightClass = 'max-h-[80vh]' }: PdfStatementViewerProps) {
    const t = useTranslations('Problems');
    const [objectUrl, setObjectUrl] = useState<string | null>(null);
    const [status, setStatus] = useState<'loading' | 'error' | 'ready'>('loading');
    const [numPages, setNumPages] = useState(0);
    const [zoom, setZoom] = useState(1);
    const [width, setWidth] = useState(0);
    const containerRef = useRef<HTMLDivElement>(null);

    const downloadUrl = problemPdfApi.getPdfUrl(code);

    // Fetch through the authenticated axios client (cookie session + CSRF/refresh
    // interceptors); the backend enforces CanViewProblem and caps at 10 MB.
    useEffect(() => {
        let created: string | null = null;
        let cancelled = false;
        setStatus('loading');
        setNumPages(0);
        setObjectUrl(null);
        api.get(`/problem/${code}/pdf`, { responseType: 'blob' })
            .then((res) => {
                if (cancelled) return;
                created = URL.createObjectURL(res.data as Blob);
                setObjectUrl(created);
            })
            .catch(() => {
                if (!cancelled) setStatus('error');
            });
        return () => {
            cancelled = true;
            if (created) URL.revokeObjectURL(created);
        };
    }, [code]);

    // Measure container width so pages fit; re-measure on window resize.
    useEffect(() => {
        const measure = () => setWidth(containerRef.current?.clientWidth ?? 0);
        measure();
        window.addEventListener('resize', measure);
        return () => window.removeEventListener('resize', measure);
    }, []);

    const file = useMemo(() => (objectUrl ? { url: objectUrl } : null), [objectUrl]);
    const pageWidth = width > 0 ? Math.max(200, (width - 32) * zoom) : undefined;

    if (status === 'error') {
        return (
            <div className="bg-card border rounded-3xl shadow-sm">
                <div className="flex flex-col items-center justify-center gap-4 p-10 text-center">
                    <FileText size={40} className="text-red-500" />
                    <p className="text-sm text-muted-foreground">{t('pdfViewer.error')}</p>
                    <div className="flex items-center gap-2">
                        <a href={downloadUrl} download className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg font-bold text-sm">
                            <Download size={16} /> {t('pdfViewer.download')}
                        </a>
                        <a href={downloadUrl} target="_blank" rel="noopener noreferrer" className="flex items-center gap-2 px-4 py-2 border rounded-lg font-bold text-sm">
                            <ArrowUpRight size={16} /> {t('pdfViewer.openNewTab')}
                        </a>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="bg-card border rounded-3xl shadow-sm overflow-hidden flex flex-col">
            <div className="flex items-center justify-between gap-2 border-b p-3">
                <span className="text-xs font-bold text-muted-foreground">
                    {status === 'ready' ? t('pdfViewer.pageCount', { count: numPages }) : t('pdfViewer.loading')}
                </span>
                <div className="flex items-center gap-1">
                    <button onClick={() => setZoom((z) => Math.max(0.5, z - 0.1))} aria-label={t('pdfViewer.zoomOut')} className="p-2 rounded-lg hover:bg-muted transition-colors"><ZoomOut size={16} /></button>
                    <button onClick={() => setZoom((z) => Math.min(2.5, z + 0.1))} aria-label={t('pdfViewer.zoomIn')} className="p-2 rounded-lg hover:bg-muted transition-colors"><ZoomIn size={16} /></button>
                    <a href={downloadUrl} download aria-label={t('pdfViewer.download')} className="p-2 rounded-lg hover:bg-muted transition-colors"><Download size={16} /></a>
                    <a href={downloadUrl} target="_blank" rel="noopener noreferrer" aria-label={t('pdfViewer.openNewTab')} className="p-2 rounded-lg hover:bg-muted transition-colors"><ArrowUpRight size={16} /></a>
                </div>
            </div>

            <div ref={containerRef} className={`overflow-auto bg-muted/30 ${heightClass}`}>
                {status === 'loading' && !file && (
                    <div className="flex items-center justify-center p-10"><Loader2 className="animate-spin text-primary" size={28} /></div>
                )}
                {file && (
                    <Document
                        file={file}
                        onLoadSuccess={({ numPages: n }) => { setNumPages(n); setStatus('ready'); }}
                        onLoadError={() => setStatus('error')}
                        loading={<div className="flex items-center justify-center p-10"><Loader2 className="animate-spin text-primary" size={28} /></div>}
                    >
                        <div className="flex flex-col items-center gap-4 py-4">
                            {Array.from({ length: numPages }, (_, i) => (
                                <Page key={i} pageNumber={i + 1} width={pageWidth} className="shadow-lg" />
                            ))}
                        </div>
                    </Document>
                )}
            </div>
        </div>
    );
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `npm test -- PdfStatementViewer`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add src/components/ui/PdfStatementViewer.tsx __tests__/components/PdfStatementViewer.test.tsx
git commit -m "feat(web): add PdfStatementViewer with loading/error/success states"
```

---

### Task 3: Integrate the viewer into the problem page + i18n

**Files:**
- Modify: `src/app/[locale]/problems/[code]/ProblemPageContent.tsx`
- Modify: `src/i18n/en.json`, `src/i18n/vi.json`

**Interfaces:**
- Consumes: `PdfStatementViewer` (default export from `@/components/ui/PdfStatementViewer`), `problemPdfApi.getPdfUrl` from `@/lib/api`, and `Problems.pdfViewer.*` i18n keys.

- [ ] **Step 1: Add the `pdfViewer` i18n block (English)**

In `src/i18n/en.json`, find (lines ~120-121):
```json
  "Problems": {
    "title": "Problems",
```
Replace with:
```json
  "Problems": {
    "title": "Problems",
    "pdfViewer": {
      "sidebarLabel": "PDF Statement",
      "loading": "Loading PDF…",
      "pageCount": "{count} pages",
      "error": "The PDF statement could not be loaded.",
      "download": "Download",
      "openNewTab": "Open in New Tab",
      "zoomIn": "Zoom in",
      "zoomOut": "Zoom out"
    },
```

- [ ] **Step 2: Add the `pdfViewer` block (Vietnamese)**

In `src/i18n/vi.json`, find (lines ~120-121):
```json
  "Problems": {
    "title": "Bài toán",
```
Replace with:
```json
  "Problems": {
    "title": "Bài toán",
    "pdfViewer": {
      "sidebarLabel": "Đề bài PDF",
      "loading": "Đang tải PDF…",
      "pageCount": "{count} trang",
      "error": "Không thể tải đề bài PDF.",
      "download": "Tải xuống",
      "openNewTab": "Mở trong tab mới",
      "zoomIn": "Phóng to",
      "zoomOut": "Thu nhỏ"
    },
```

- [ ] **Step 3: Verify i18n parity**

Run: `npm run i18n:check`
Expected: PASS (no missing/extra keys between en and vi).

- [ ] **Step 4: Dynamic-import the viewer and import the URL helper**

In `ProblemPageContent.tsx`, change the api import (line 6) from:
```tsx
import api, { problemClarificationApi } from '@/lib/api';
```
to:
```tsx
import api, { problemClarificationApi, problemPdfApi } from '@/lib/api';
```

Add `import dynamic from 'next/dynamic';` alongside the other imports (e.g. directly under line 5 `import { useTranslations } from 'next-intl';`).

Then, after the `formatMemoryLimit` helper (i.e. immediately before `export default function ProblemPageContent`), add the dynamic component:
```tsx
// react-pdf touches browser-only APIs — load it client-side only.
const PdfStatementViewer = dynamic(() => import('@/components/ui/PdfStatementViewer'), {
    ssr: false,
    loading: () => (
        <div className="flex items-center justify-center p-10 bg-card border rounded-3xl shadow-sm">
            <Loader2 className="animate-spin text-primary" size={28} />
        </div>
    ),
});
```

- [ ] **Step 5: Branch the statement body on `pdf_url`**

In `ProblemPageContent.tsx`, replace the statement body block (lines ~294-296):
```tsx
                        <div className="prose prose-sm dark:prose-invert max-w-none bg-card border rounded-3xl p-8 lg:p-10 shadow-sm leading-relaxed">
                            <MathRenderer content={problem.description} fullMarkup={problem.is_full_markup} />
                        </div>
```
with (renders the description box when there is a description **or** no PDF — preserving the current empty-box behavior for problems with neither — and appends the viewer whenever a PDF exists; no duplicated markup):
```tsx
                        {(problem.description?.trim() || !problem.pdf_url) && (
                            <div className="prose prose-sm dark:prose-invert max-w-none bg-card border rounded-3xl p-8 lg:p-10 shadow-sm leading-relaxed">
                                <MathRenderer content={problem.description} fullMarkup={problem.is_full_markup} />
                            </div>
                        )}
                        {problem.pdf_url && <PdfStatementViewer code={code} />}
```

Behavior: description-only → box only (unchanged); PDF + description → box then viewer; PDF-only → viewer only; neither → empty box (unchanged).

- [ ] **Step 6: i18n the sidebar button label**

In `ProblemPageContent.tsx`, replace (line ~203):
```tsx
                            <span className="text-sm font-bold">PDF Statement</span>
```
with:
```tsx
                            <span className="text-sm font-bold">{t('pdfViewer.sidebarLabel')}</span>
```
(The component already declares `const t = useTranslations('Problems');` at line 53.)

- [ ] **Step 7: Replace the modal iframe with the shared viewer**

In `ProblemPageContent.tsx`, in the PDF modal, replace the "Open in New Tab" href (line ~390) from `` `/api/problem/${code}/pdf` `` to `` `${problemPdfApi.getPdfUrl(code)}` ``:
```tsx
                            <a
                                href={problemPdfApi.getPdfUrl(code)}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:scale-105 transition-all font-bold shadow-lg"
                            >
```
Then replace the iframe container (lines ~407-414):
```tsx
                        {/* PDF iframe */}
                        <div className="w-full h-full bg-white rounded-lg overflow-hidden shadow-2xl">
                            <iframe
                                src={`/api/problem/${code}/pdf`}
                                className="w-full h-full"
                                title="PDF Statement"
                            />
                        </div>
```
with:
```tsx
                        {/* PDF viewer */}
                        <div className="w-full h-full overflow-hidden rounded-lg shadow-2xl">
                            <PdfStatementViewer code={code} heightClass="h-full" />
                        </div>
```

- [ ] **Step 8: Typecheck, lint, and run the full test suite**

Run: `npx tsc --noEmit && npm run lint && npm test`
Expected: PASS. `tsc` clean; lint clean (if lint flags the removed `X`/`FileText`/`iframe`-related imports as unused, remove only the now-unused ones — verify each is truly unused before deleting); all tests green.

- [ ] **Step 9: Commit**

```bash
git add src/app/[locale]/problems/[code]/ProblemPageContent.tsx src/i18n/en.json src/i18n/vi.json
git commit -m "feat(web): render PDF statements inline on the problem page (v1 parity)"
```

---

### Task 4: Build verification, contingencies, and manual check

**Files:**
- Modify (contingency only): `next.config.ts`

- [ ] **Step 1: Production build with Turbopack**

Run: `npm run build`
Expected: PASS. The build resolves `pdfjs-dist/build/pdf.worker.min.mjs` via `import.meta.url` and bundles the worker.

- [ ] **Step 2: Contingency — pdfjs `canvas` optional dependency**

Only if Step 1 fails with an error mentioning `canvas` (pdf.js's optional Node-only dependency), edit `next.config.ts` to alias it away. Change:
```ts
const nextConfig: NextConfig = {
  reactCompiler: false,
};
```
to:
```ts
const nextConfig: NextConfig = {
  reactCompiler: false,
  turbopack: {
    resolveAlias: {
      canvas: './src/lib/empty-module.ts',
    },
  },
};
```
Create `src/lib/empty-module.ts` with `export default {};`. Re-run `npm run build`. Then `git add next.config.ts src/lib/empty-module.ts && git commit -m "fix(web): alias pdfjs optional canvas dep for Turbopack build"`. If Step 1 already passed, skip this step entirely.

- [ ] **Step 3: Manual verification (drive the app)**

Start the app (`npm run dev`, or the project's compose setup that proxies `/api` to the Go backend) and confirm against a real PDF problem (e.g. code `01_02` "Chọn bi" from the screenshots) and a real markdown problem:

1. Open a **PDF problem** → the statement body shows the PDF pages inline (scrollable), not an empty box. Zoom +/- resize the pages. The sidebar "PDF Statement" button still opens the modal, which now shows the same viewer.
2. Click **Download** and **Open in New Tab** → both hit `/api/problem/<code>/pdf` and serve the PDF.
3. Open a **markdown-only problem** → body renders exactly as before (MathRenderer), no viewer, no empty extra box.
4. (If reachable) A problem with **both** a description and `pdf_url` → shows the description first, then the PDF viewer below.
5. Simulate a load failure (e.g. temporarily stop the backend, or open a problem whose PDF 404s) → the viewer shows the fallback card with working Download / Open-in-new-tab links, never a blank box.

- [ ] **Step 4: Final commit (if any config/doc changes remain)**

```bash
git status
# commit any remaining tracked changes with an appropriate message
```

---

## Self-Review

**1. Spec coverage** (against `docs/superpowers/specs/2026-07-21-v2-pdf-statement-inline-design.md`):
- §2 backend already supports PDF → no backend task (correct; verified in exploration). ✓
- §4.1 `PdfStatementViewer` (blob fetch via api client, all-pages, toolbar, loading/error fallback, worker+CSS setup) → Tasks 1-2. ✓
- §4.2 body branch + sidebar label + modal fold-in → Task 3 (steps 4-7). ✓
- §5 deps + worker config + `canvas` contingency → Tasks 1 & 4. ✓ (cmap contingency is runtime-only and listed as a manual note in the spec; surfaced in Task 4 Step 3 as glyph verification.)
- §6 i18n keys (en+vi) + tests → Task 3 (steps 1-3) & Task 2. ✓
- §6 "ProblemPageContent picks viewer vs MathRenderer" test → **intentionally covered by typecheck + manual verification (Task 4 Step 3), not an automated test.** The repo has no page-level render tests, and `ProblemPageContent` requires QueryClient, the React 19 `use(params)` promise, `useSearchParams`, Monaco, and Comments to mount — a full render test is disproportionate, and the branch itself (`problem.pdf_url ? … : …`) is trivial. Automated coverage stays on the self-contained viewer.
- §8 success criteria → Task 4 Step 3 checklist maps 1:1. ✓

**2. Placeholder scan:** No TBD/TODO; every code step shows complete code; the one conditional step (Task 4 Step 2) has an explicit trigger and full remedy code. ✓

**3. Type consistency:** `PdfStatementViewer` is a default export taking `{ code: string; heightClass?: string }` in Task 2 and consumed with exactly those props in Task 3 (`code={code}`, and `code={code} heightClass="h-full"`). `problemPdfApi.getPdfUrl(code: string)` matches `src/lib/api.ts:318`. `api.get(url, { responseType: 'blob' })` matches the axios default export. i18n keys used by the viewer (`pdfViewer.loading`, `.pageCount`, `.error`, `.download`, `.openNewTab`, `.zoomIn`, `.zoomOut`) and the page (`pdfViewer.sidebarLabel`) all exist in the Task 3 Step 1-2 blocks. ✓
