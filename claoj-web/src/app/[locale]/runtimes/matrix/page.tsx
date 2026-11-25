'use client';

import { useEffect, useMemo, useState } from 'react';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Link } from '@/navigation';
import { buildMatrixLookup, formatRuntimeVersions, runtimeCellState } from '@/lib/runtimes';
import { cn } from '@/lib/utils';
import type { RuntimeMatrix } from '@/types';

export default function RuntimeMatrixPage() {
    const t = useTranslations('Runtimes');
    const [matrix, setMatrix] = useState<RuntimeMatrix | null>(null);
    const [loading, setLoading] = useState(true);
    const [failed, setFailed] = useState(false);

    useEffect(() => {
        api.get<RuntimeMatrix>('/runtimes/matrix')
            .then((res) => setMatrix(res.data))
            .catch(() => setFailed(true))
            .finally(() => setLoading(false));
    }, []);

    const lookup = useMemo(() => buildMatrixLookup(matrix?.cells ?? []), [matrix]);

    return (
        <div className="container mx-auto px-4 py-8">
            <div className="flex flex-wrap items-end justify-between gap-4 mb-6">
                <div>
                    <h1 className="text-3xl font-bold">{t('matrixTitle')}</h1>
                    <p className="text-muted-foreground mt-1">{t('matrixSubtitle')}</p>
                </div>
                <nav className="flex gap-1 border-b w-full sm:w-auto" aria-label={t('title')}>
                    <Link href="/stats" className="px-3 py-2 text-sm text-muted-foreground hover:text-foreground">
                        {t('tabJudges')}
                    </Link>
                    <Link href="/runtimes" className="px-3 py-2 text-sm text-muted-foreground hover:text-foreground">
                        {t('tabRuntimes')}
                    </Link>
                    <span aria-current="page" className="px-3 py-2 text-sm font-semibold border-b-2 border-primary">
                        {t('tabVersions')}
                    </span>
                </nav>
            </div>

            {loading ? (
                <div className="flex justify-center py-16">
                    <div className="animate-spin rounded-full h-10 w-10 border-2 border-muted border-t-primary" />
                </div>
            ) : failed ? (
                <p className="text-center py-16 text-muted-foreground">{t('loadFailed')}</p>
            ) : !matrix || matrix.judges.length === 0 || matrix.languages.length === 0 ? (
                <p className="text-center py-16 text-muted-foreground">{t('matrixEmpty')}</p>
            ) : (
                <div className="border rounded-lg overflow-x-auto">
                    <table className="w-full text-sm">
                        <thead>
                            <tr className="border-b bg-muted/40">
                                <th scope="col" className="text-left font-semibold px-4 py-2" />
                                {matrix.judges.map((judge) => (
                                    <th key={judge} scope="col" className="text-left font-semibold px-4 py-2 font-mono">
                                        {judge}
                                    </th>
                                ))}
                            </tr>
                        </thead>
                        <tbody>
                            {matrix.languages.map((language) => (
                                <tr key={language.id} className="border-b last:border-0">
                                    <th scope="row" className="text-left font-medium px-4 py-2 whitespace-nowrap">
                                        {language.name}
                                    </th>
                                    {matrix.judges.map((judge) => {
                                        const cell = lookup.get(`${judge}|${language.id}`);
                                        const state = runtimeCellState(cell);
                                        return (
                                            <td
                                                key={judge}
                                                className={cn('px-4 py-2 align-top', {
                                                    'text-muted-foreground': state === 'blank',
                                                    'text-success': state === 'latest',
                                                    'text-warning': state === 'outdated',
                                                })}
                                            >
                                                {cell?.runtimes.length
                                                    ? cell.runtimes.map((runtime) => (
                                                        <div key={`${runtime.name}-${runtime.versions[0]}`}>
                                                            {formatRuntimeVersions([runtime])}
                                                        </div>
                                                    ))
                                                    : '—'}
                                            </td>
                                        );
                                    })}
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            <div className="mt-4 flex flex-wrap gap-4 text-xs text-muted-foreground">
                <span className="text-success">{t('legendLatest')}</span>
                <span className="text-warning">{t('legendOutdated')}</span>
                <span>{t('legendBlank')}</span>
            </div>
        </div>
    );
}