'use client';

import { useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { Link } from '@/navigation';
import { ArrowLeft, Printer } from 'lucide-react';
import MathRenderer from '@/components/ui/MathRenderer';
import { formatMemoryLimit } from '@/lib/utils';
import type { ProblemDetail } from '@/types';

/**
 * Print-optimized statement view. v1 rendered a PDF server-side with
 * wkhtmltopdf; v2 has no HTML→PDF engine (and no new deps), so "Download PDF"
 * opens this page and hands off to the browser's print dialog, where the user
 * picks "Save as PDF". The uploaded-PDF path is untouched.
 */
export default function ProblemPrintView({ problem, autoPrint }: { problem: ProblemDetail; autoPrint: boolean }) {
    const t = useTranslations('Problems');
    const tPrint = useTranslations('ProblemPrint');

    useEffect(() => {
        if (!autoPrint) return;
        let cancelled = false;
        const run = () => { if (!cancelled) window.print(); };
        // Webfonts and KaTeX finish laying out after first paint; printing
        // sooner snapshots a half-laid-out page. fonts.ready is that signal.
        if (typeof document !== 'undefined' && document.fonts) {
            void document.fonts.ready.then(run);
        } else {
            run();
        }
        return () => { cancelled = true; };
    }, [autoPrint]);

    return (
        <div className="problem-print mx-auto max-w-3xl bg-white p-2 print:max-w-none">
            <div className="mb-6 flex items-center justify-between gap-3 print:hidden">
                <Link
                    href={`/problem/${problem.code}`}
                    className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-primary transition-colors"
                >
                    <ArrowLeft size={16} />
                    {tPrint('back')}
                </Link>
                <button
                    type="button"
                    onClick={() => window.print()}
                    className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground"
                >
                    <Printer size={16} />
                    {tPrint('print')}
                </button>
            </div>

            <header className="space-y-3 border-b pb-4">
                <p className="text-xs font-mono text-muted-foreground">{problem.code}</p>
                <h1 className="text-3xl font-semibold tracking-tight">{problem.name}</h1>
                <dl className="grid grid-cols-2 gap-x-6 gap-y-1 text-sm sm:grid-cols-4">
                    <div>
                        <dt className="text-xs text-muted-foreground">{t('timeLimit')}</dt>
                        <dd className="font-medium">{problem.time_limit}s</dd>
                    </div>
                    <div>
                        <dt className="text-xs text-muted-foreground">{t('memoryLimit')}</dt>
                        <dd className="font-medium">{formatMemoryLimit(problem.memory_limit)}</dd>
                    </div>
                    <div>
                        <dt className="text-xs text-muted-foreground">{t('points')}</dt>
                        <dd className="font-medium">
                            {problem.points}
                            {problem.partial && <span className="ml-1 text-xs">({t('partialScoring')})</span>}
                        </dd>
                    </div>
                    <div>
                        <dt className="text-xs text-muted-foreground">{t('group')}</dt>
                        <dd className="truncate font-medium">{problem.group}</dd>
                    </div>
                </dl>
                {problem.authors.length > 0 && (
                    <p className="text-xs text-muted-foreground">
                        {t('authors')}: {problem.authors.map(a => `@${a.username}`).join(', ')}
                    </p>
                )}
            </header>

            <article className="pt-4">
                <MathRenderer content={problem.description} />
            </article>

            {problem.pdf_url && (
                <p className="mt-6 border-t pt-3 text-xs text-muted-foreground print:hidden">
                    {tPrint('uploadedPdfNote')}
                </p>
            )}
        </div>
    );
}