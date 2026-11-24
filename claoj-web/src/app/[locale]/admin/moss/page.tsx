'use client';

import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ScanSearch, Play, Trash2, ExternalLink, Search } from 'lucide-react';
import { adminContestApi, adminMossApi, type ContestMossResult } from '@/lib/adminApi';
import { contestPhase } from '@/lib/contestPhase';
import { useListParams } from '@/hooks/useListParams';
import { useDebounce } from '@/hooks/useDebounce';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';

const KEYS = ['contest'] as const;
const LIST_SIZE = 20;

type ApiError = { response?: { data?: { error?: string } } };
const errorText = (e: unknown) => (e as ApiError)?.response?.data?.error ?? String(e);

export default function AdminMossPage() {
    const t = useTranslations('Admin');
    const queryClient = useQueryClient();
    const { filters, setFilters } = useListParams(KEYS);
    const contestKey = filters.contest ?? '';
    const [term, setTerm] = useState('');
    const debounced = useDebounce(term.trim(), 250);

    const { data: contestList, isLoading: listLoading, isError: listError, error: listErrorValue } = useQuery({
        queryKey: ['moss-contest-list', debounced],
        queryFn: async () => (await adminContestApi.list(1, LIST_SIZE, { search: debounced })).data,
    });
    const contests = contestList?.data ?? [];

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

            <section className="rounded-md border bg-card">
                <header className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 border-b bg-secondary/60">
                    <h2 className="text-sm font-semibold">{t('moss.contestList')}</h2>
                    <div className="relative w-full sm:w-72">
                        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" size={14} />
                        <input
                            aria-label={t('moss.contestLabel')}
                            value={term}
                            onChange={e => setTerm(e.target.value)}
                            placeholder={t('moss.contestPlaceholder')}
                            autoComplete="off"
                            className="w-full h-8 pl-8 pr-3 rounded-md border bg-card text-sm focus:outline-none focus:ring-2 focus:ring-ring/40"
                        />
                    </div>
                </header>

                {listLoading ? (
                    <div className="p-4 space-y-2">{[1, 2, 3].map(i => <Skeleton key={i} className="h-6 w-full" />)}</div>
                ) : listError ? (
                    <p className="px-4 py-6 text-center text-sm text-destructive">{errorText(listErrorValue)}</p>
                ) : contests.length === 0 ? (
                    <p className="px-4 py-6 text-center text-sm text-muted-foreground">{t('moss.contestListEmpty')}</p>
                ) : (
                    <ul className="divide-y max-h-96 overflow-y-auto">
                        {contests.map(c => {
                            const selected = c.key === contestKey;
                            const phase = contestPhase(c.start_time, c.end_time);
                            return (
                                <li key={c.key}>
                                    <button
                                        type="button"
                                        onClick={() => setFilters({ contest: c.key })}
                                        aria-pressed={selected}
                                        className={`w-full flex items-center gap-3 px-4 py-2 text-left text-sm hover:bg-muted/50 ${selected ? 'bg-primary/10' : ''}`}
                                    >
                                        <span className={`truncate ${selected ? 'font-semibold text-primary' : 'font-medium'}`}>{c.name}</span>
                                        <span className="shrink-0 font-mono text-xs text-muted-foreground">{c.key}</span>
                                        <span className="ml-auto shrink-0 text-xs text-muted-foreground tabular-nums">
                                            {c.start_time ? new Date(c.start_time).toLocaleDateString() : '—'}
                                        </span>
                                        <span className={`shrink-0 w-24 text-right text-xs ${phase === 'ongoing' ? 'text-success' : 'text-muted-foreground'}`}>
                                            {t(`moss.phase.${phase}`)}
                                        </span>
                                    </button>
                                </li>
                            );
                        })}
                    </ul>
                )}
            </section>

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
