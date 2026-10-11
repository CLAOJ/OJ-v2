'use client';

import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ScanSearch, Play, Trash2, ExternalLink } from 'lucide-react';
import { adminMossApi, type ContestMossResult } from '@/lib/adminApi';
import { searchContests } from '@/lib/adminSearch';
import { useListParams } from '@/hooks/useListParams';
import { useDebounce } from '@/hooks/useDebounce';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';

const KEYS = ['contest'] as const;

type ApiError = { response?: { data?: { error?: string } } };
const errorText = (e: unknown) => (e as ApiError)?.response?.data?.error ?? String(e);

export default function AdminMossPage() {
    const t = useTranslations('Admin');
    const queryClient = useQueryClient();
    const { filters, setFilters } = useListParams(KEYS);
    const contestKey = filters.contest ?? '';
    const [term, setTerm] = useState('');
    const debounced = useDebounce(term.trim(), 250);

    const { data: matches = [] } = useQuery({
        queryKey: ['moss-contest-search', debounced],
        queryFn: () => searchContests(debounced),
        enabled: debounced.length > 0,
    });

    const { data, isLoading, isError, error } = useQuery({
        queryKey: ['admin-contest-moss', contestKey],
        queryFn: async () => (await adminMossApi.list(contestKey)).data,
        enabled: contestKey !== '',
        refetchInterval: q => (q.state.data?.running ? 5000 : false),
    });

    const run = useMutation({
        mutationFn: () => adminMossApi.run(contestKey),
        onSuccess: () => {
            toast.success(t('moss.queued'));
            queryClient.invalidateQueries({ queryKey: ['admin-contest-moss', contestKey] });
        },
        onError: e => toast.error(errorText(e)),
    });

    const clear = useMutation({
        mutationFn: () => adminMossApi.clear(contestKey),
        onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin-contest-moss', contestKey] }),
        onError: e => toast.error(errorText(e)),
    });

    // Group rows by problem so each problem reads as one block.
    const grouped = useMemo(() => {
        const map = new Map<string, { name: string; rows: ContestMossResult[] }>();
        for (const r of data?.results ?? []) {
            if (!map.has(r.problem_code)) map.set(r.problem_code, { name: r.problem_name, rows: [] });
            map.get(r.problem_code)!.rows.push(r);
        }
        return [...map.entries()];
    }, [data]);

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-2xl font-bold flex items-center gap-2.5">
                    <ScanSearch className="text-primary" size={26} />
                    {t('moss.title')}
                </h1>
                <p className="text-sm text-muted-foreground mt-1">{t('moss.subtitle')}</p>
            </div>

            <div className="relative max-w-md">
                <label htmlFor="moss-contest" className="text-xs text-muted-foreground block mb-1">{t('moss.contestLabel')}</label>
                <input
                    id="moss-contest"
                    value={term}
                    onChange={e => setTerm(e.target.value)}
                    placeholder={contestKey || t('moss.contestPlaceholder')}
                    autoComplete="off"
                    className="w-full h-9 px-3 rounded-md border bg-card text-sm focus:outline-none focus:ring-2 focus:ring-ring/40"
                />
                {debounced && matches.length > 0 && (
                    <ul className="absolute z-20 mt-1 w-full rounded-md border bg-popover shadow-lg max-h-60 overflow-y-auto">
                        {matches.map(m => (
                            <li key={m.id}>
                                <button
                                    type="button"
                                    onClick={() => { setFilters({ contest: m.label }); setTerm(''); }}
                                    className="w-full text-left px-3 py-2 text-sm hover:bg-muted flex gap-2 items-baseline"
                                >
                                    <span className="font-medium">{m.label}</span>
                                    <span className="text-xs text-muted-foreground truncate">{m.detail}</span>
                                </button>
                            </li>
                        ))}
                    </ul>
                )}
            </div>

            {!contestKey ? (
                <p className="text-sm text-muted-foreground">{t('moss.pickContest')}</p>
            ) : (
                <section className="rounded-md border bg-card">
                    <header className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 border-b bg-secondary/60">
                        <h2 className="text-sm font-semibold">
                            <Link href={`/contest/${contestKey}`} className="hover:underline">{contestKey}</Link>
                            {data?.running && <span className="ml-2 text-xs font-normal text-warning">{t('moss.running')}</span>}
                        </h2>
                        <div className="flex gap-2">
                            <button
                                onClick={() => run.mutate()}
                                disabled={run.isPending || data?.running}
                                className="inline-flex items-center gap-1.5 h-8 px-3 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-50"
                            >
                                <Play size={14} /> {t('moss.run')}
                            </button>
                            <button
                                onClick={() => { if (confirm(t('moss.clearConfirm'))) clear.mutate(); }}
                                disabled={clear.isPending || !data?.results.length}
                                className="inline-flex items-center gap-1.5 h-8 px-3 rounded-md border text-sm hover:bg-muted disabled:opacity-50"
                            >
                                <Trash2 size={14} /> {t('moss.clear')}
                            </button>
                        </div>
                    </header>

                    {data?.error && (
                        <p className="px-4 py-2 text-sm text-destructive bg-destructive/10 border-b">
                            {t('moss.errorLabel')} {data.error}
                        </p>
                    )}

                    {isLoading ? (
                        <div className="p-4 space-y-2">{[1, 2, 3].map(i => <Skeleton key={i} className="h-5 w-full" />)}</div>
                    ) : isError ? (
                        <p className="px-4 py-8 text-center text-sm text-destructive">{errorText(error)}</p>
                    ) : grouped.length === 0 ? (
                        <p className="px-4 py-8 text-center text-sm text-muted-foreground">{t('moss.noResults')}</p>
                    ) : (
                        <table className="w-full text-sm">
                            <thead className="text-left text-xs text-muted-foreground">
                                <tr>
                                    <th className="px-4 py-2 font-semibold">{t('moss.colProblem')}</th>
                                    <th className="px-4 py-2 font-semibold">{t('moss.colLanguage')}</th>
                                    <th className="px-4 py-2 font-semibold text-right">{t('moss.colSubmissions')}</th>
                                    <th className="px-4 py-2 font-semibold">{t('moss.colReport')}</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {grouped.flatMap(([code, g]) => g.rows.map((r, i) => (
                                    <tr key={`${code}-${r.language}`}>
                                        <td className="px-4 py-2">{i === 0 && <><span className="font-medium">{code}</span> <span className="text-muted-foreground">{g.name}</span></>}</td>
                                        <td className="px-4 py-2">{r.language}</td>
                                        <td className="px-4 py-2 text-right tabular-nums">{r.submission_count}</td>
                                        <td className="px-4 py-2">
                                            {r.url ? (
                                                <a href={r.url} target="_blank" rel="noopener noreferrer" className="text-accent hover:underline inline-flex items-center gap-1">
                                                    {t('moss.viewReport')} <ExternalLink size={12} />
                                                </a>
                                            ) : <span className="text-muted-foreground">—</span>}
                                        </td>
                                    </tr>
                                )))}
                            </tbody>
                        </table>
                    )}
                </section>
            )}
        </div>
    );
}
