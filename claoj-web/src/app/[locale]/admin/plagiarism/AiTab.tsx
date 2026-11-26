'use client';

import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ExternalLink, Plus, RefreshCw, Trash2 } from 'lucide-react';
import { adminContestApi, adminMossApi, adminErrorMessage, type AIProblem } from '@/lib/adminApi';
import { useConfirm } from '@/components/providers/ConfirmProvider';
import { Skeleton } from '@/components/ui/Skeleton';
import { Textarea } from '@/components/ui/Textarea';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/Dialog';
import { Link } from '@/navigation';

type ApiError = { response?: { status?: number; data?: { error?: string } } };
const errorText = (e: unknown) => (e as ApiError)?.response?.data?.error ?? String(e);

export default function AiTab({ contestKey, languages, problems }: { contestKey: string; languages: string[]; problems: string[] }) {
    const t = useTranslations('Admin');
    const confirm = useConfirm();
    const queryClient = useQueryClient();
    const [filter, setFilter] = useState('');
    const [adding, setAdding] = useState(false);
    const [problem, setProblem] = useState('');
    const [lang, setLang] = useState('');
    const [source, setSource] = useState('');

    const report = useQuery({
        queryKey: ['admin-contest-moss-ai', contestKey],
        queryFn: async () => (await adminMossApi.aiReport(contestKey)).data,
        retry: false, // 404 = no run yet
    });
    const refs = useQuery({
        queryKey: ['admin-contest-moss-refs', contestKey],
        queryFn: async () => (await adminMossApi.references(contestKey)).data.references,
    });
    // Contest problems exist before any run, so a PDF-only problem can get a reference first.
    const contest = useQuery({
        queryKey: ['admin-contest-detail-problems', contestKey],
        queryFn: async () => (await adminContestApi.detail(contestKey)).data,
    });
    const refsChanged = () => {
        queryClient.invalidateQueries({ queryKey: ['admin-contest-moss-refs', contestKey] });
        toast.success(t('moss.ai.appliesNext'));
    };

    const add = useMutation({
        mutationFn: (p: { problem_code: string; language: string; source: string }) => adminMossApi.addReference(contestKey, p),
        onSuccess: () => { setAdding(false); setSource(''); refsChanged(); },
        onError: e => toast.error(adminErrorMessage(e, errorText(e))),
    });
    const del = useMutation({
        mutationFn: (id: number) => adminMossApi.deleteReference(contestKey, id),
        onSuccess: refsChanged,
        onError: e => toast.error(errorText(e)),
    });
    const regen = useMutation({
        mutationFn: (code: string) => adminMossApi.regenerate(contestKey, code),
        onSuccess: refsChanged,
        onError: e => toast.error(errorText(e)),
    });

    const rows = useMemo(() => {
        const q = filter.trim().toLowerCase();
        return (report.data?.rows ?? []).filter(r => !q || r.username.toLowerCase().includes(q) || r.problem_code.toLowerCase().includes(q));
    }, [report.data, filter]);

    // Problem x language entries: report problems plus any problem that only has references.
    const entries = useMemo(() => {
        const map = new Map<string, AIProblem>();
        for (const p of report.data?.problems ?? []) map.set(`${p.problem_code}/${p.language}`, p);
        for (const r of refs.data ?? []) {
            const k = `${r.problem_code}/${r.language}`;
            if (!map.has(k)) map.set(k, { problem_code: r.problem_code, language: r.language, generated: 0, admin: 0, contestants: 0, matched: 0, trivial: false, note: '' });
        }
        return [...map.values()].sort((a, b) => a.problem_code.localeCompare(b.problem_code) || a.language.localeCompare(b.language));
    }, [report.data, refs.data]);
    const problemCodes = [...new Set([...(contest.data?.problems ?? []).map(p => p.code), ...problems, ...entries.map(e => e.problem_code)])].sort();
    const langOptions = [...new Set([...languages, ...entries.map(e => e.language)])].sort();

    // Codes may arrive after the dialog opens: fall back to the first one.
    const selected = problem || problemCodes[0] || '';

    const noteText = (note: string) => {
        if (!note) return null;
        const [kind, ...detail] = note.split(': ');
        return t.has(`moss.ai.note.${kind}`) ? `${t(`moss.ai.note.${kind}`)}${detail.length ? `: ${detail.join(': ')}` : ''}` : note;
    };

    const noRun = (report.error as ApiError | null)?.response?.status === 404;

    return (
        <div className="space-y-6">
            <section>
                <header className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 border-b bg-secondary/60">
                    <h2 className="text-sm font-semibold">{t('moss.ai.suspects')}</h2>
                    <input
                        aria-label={t('moss.ai.filter')}
                        placeholder={t('moss.ai.filter')}
                        value={filter}
                        onChange={e => setFilter(e.target.value)}
                        className="w-full sm:w-64 h-8 px-3 rounded-md border bg-card text-sm focus:outline-none focus:ring-2 focus:ring-ring/40"
                    />
                </header>
                <p className="px-4 py-2 text-xs text-warning bg-warning/15 border-b">{t('moss.ai.disclaimer')}</p>
                {report.isLoading ? (
                    <div className="p-4 space-y-2">{[1, 2, 3].map(i => <Skeleton key={i} className="h-5 w-full" />)}</div>
                ) : noRun ? (
                    <p className="px-4 py-8 text-center text-sm text-muted-foreground">{t('moss.ai.noRun')}</p>
                ) : report.isError ? (
                    <p className="px-4 py-8 text-center text-sm text-destructive">{errorText(report.error)}</p>
                ) : !(report.data?.rows ?? []).length ? (
                    <p className="px-4 py-8 text-center text-sm text-muted-foreground">{t('moss.ai.noSuspects')}</p>
                ) : rows.length === 0 ? (
                    <p className="px-4 py-8 text-center text-sm text-muted-foreground">{t('moss.ai.noMatches')}</p>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="w-full text-sm">
                            <thead className="text-left text-xs text-muted-foreground">
                                <tr>
                                    <th className="px-4 py-2 font-semibold">{t('moss.ai.colUser')}</th>
                                    <th className="px-4 py-2 font-semibold">{t('moss.ai.colProblem')}</th>
                                    <th className="px-4 py-2 font-semibold">{t('moss.ai.colLang')}</th>
                                    <th className="px-4 py-2 font-semibold">{t('moss.ai.colMatch')}</th>
                                    <th className="px-4 py-2 font-semibold">{t('moss.ai.colRewrites')}</th>
                                    <th className="px-4 py-2 font-semibold">{t('moss.ai.colOutlier')}</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {rows.map(r => (
                                    <tr key={`${r.username}-${r.problem_code}-${r.language}`}>
                                        <td className="px-4 py-2 font-medium">{r.username}</td>
                                        <td className="px-4 py-2">{r.problem_code}</td>
                                        <td className="px-4 py-2">{r.language}</td>
                                        <td className="px-4 py-2">
                                            {r.ai_match ? (
                                                <a href={adminMossApi.reportUrl(contestKey, r.ai_match.report_path, `match_${r.ai_match.match}.html`)} target="_blank" rel="noopener noreferrer" className="text-accent hover:underline inline-flex items-center gap-1 tabular-nums">
                                                    {Math.round(r.ai_match.pct)}% <ExternalLink size={12} />
                                                </a>
                                            ) : <span className="text-muted-foreground">—</span>}
                                        </td>
                                        <td className="px-4 py-2">
                                            {r.rewrites > 0 ? (
                                                <Link href={`/submissions?user=${encodeURIComponent(r.username)}&problem=${encodeURIComponent(r.problem_code)}&contest=${encodeURIComponent(contestKey)}`} className="text-accent hover:underline">
                                                    {t('moss.ai.viewRewrites', { n: r.rewrites })}
                                                </Link>
                                            ) : <span className="text-muted-foreground">—</span>}
                                        </td>
                                        <td className="px-4 py-2 text-xs space-y-0.5">
                                            {r.above_history && <div className="text-warning">{t('moss.ai.aboveHistory', { pts: r.prev_max_points ?? 0 })}</div>}
                                            {r.speed_inversion && <div className="text-warning">{t('moss.ai.speedInversion')}</div>}
                                            {r.rating_before != null && <div className="text-muted-foreground">{t('moss.ai.ratingBefore', { r: r.rating_before })}</div>}
                                            {!r.above_history && !r.speed_inversion && r.rating_before == null && <span className="text-muted-foreground">—</span>}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </section>

            <section>
                <header className="px-4 py-3 border-b bg-secondary/60">
                    <h2 className="text-sm font-semibold">{t('moss.ai.refs')}</h2>
                    <p className="text-xs text-muted-foreground mt-0.5">{t('moss.ai.refsHint')}</p>
                </header>
                {entries.length === 0 ? (
                    <p className="px-4 py-6 text-center text-sm text-muted-foreground">{t('moss.ai.noRefs')}</p>
                ) : (
                    <ul className="divide-y">
                        {entries.map(e => {
                            const mine = (refs.data ?? []).filter(r => r.problem_code === e.problem_code && r.language === e.language);
                            const note = noteText(e.note);
                            return (
                                <li key={`${e.problem_code}/${e.language}`} className="px-4 py-3 text-sm space-y-1.5">
                                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                                        <span className="font-medium">{e.problem_code}</span>
                                        <span className="text-muted-foreground">{e.language}</span>
                                        <span className="text-xs text-muted-foreground">
                                            {t('moss.ai.refCounts', { generated: mine.filter(r => r.origin === 'generated').length, admin: mine.filter(r => r.origin !== 'generated').length })}
                                        </span>
                                        <span className="text-xs text-muted-foreground tabular-nums">
                                            {t('moss.ai.matched', { matched: e.matched, total: e.contestants })}
                                        </span>
                                        {e.trivial && (
                                            <span className="text-xs rounded-md border border-amber-500/40 bg-amber-500/10 px-1.5 py-0.5 text-amber-700 dark:text-amber-400">
                                                {t('moss.ai.trivial')}
                                            </span>
                                        )}
                                        <button
                                            type="button"
                                            onClick={async () => { if (await confirm(t('moss.ai.regenerateConfirm'), { variant: 'danger' })) regen.mutate(e.problem_code); }}
                                            disabled={regen.isPending}
                                            className="ml-auto inline-flex items-center gap-1 h-7 px-2 rounded-md border text-xs hover:bg-muted disabled:opacity-50"
                                        >
                                            <RefreshCw size={12} /> {t('moss.ai.regenerate')}
                                        </button>
                                    </div>
                                    {note && <p className="text-xs text-warning">{note}</p>}
                                    {mine.length > 0 && (
                                        <ul className="space-y-1">
                                            {mine.map(r => (
                                                <li key={r.id} className="flex items-center gap-2 text-xs">
                                                    <span className="px-1.5 py-0.5 rounded bg-muted text-muted-foreground">{r.origin}</span>
                                                    <span className="font-mono text-muted-foreground">{r.model || '—'}</span>
                                                    <button
                                                        type="button"
                                                        aria-label={t('moss.ai.deleteRef')}
                                                        onClick={async () => { if (await confirm(t('moss.ai.deleteConfirm'), { variant: 'danger' })) del.mutate(r.id); }}
                                                        className="ml-auto text-destructive hover:bg-destructive/10 rounded p-1"
                                                    >
                                                        <Trash2 size={13} />
                                                    </button>
                                                </li>
                                            ))}
                                        </ul>
                                    )}
                                </li>
                            );
                        })}
                    </ul>
                )}
                <div className="px-4 py-3 border-t">
                    <button
                        type="button"
                        onClick={() => { setAdding(true); setProblem(''); setLang(langOptions[0] ?? ''); setSource(''); }}
                        className="inline-flex items-center gap-1 h-8 px-3 rounded-md border text-sm hover:bg-muted"
                    >
                        <Plus size={14} /> {t('moss.ai.add')}
                    </button>
                </div>
            </section>

            <Dialog open={adding} onOpenChange={o => { if (!o) setAdding(false); }}>
                <DialogContent>
                    <DialogHeader><DialogTitle>{t('moss.ai.addTitle')}</DialogTitle></DialogHeader>
                    <div className="space-y-3 px-1">
                        <label className="block text-sm">
                            <span className="text-muted-foreground">{t('moss.ai.colProblem')}</span>
                            <select value={selected} onChange={e => setProblem(e.target.value)} className="mt-1 w-full h-9 px-2 rounded-md border bg-card text-sm">
                                {problemCodes.map(c => <option key={c} value={c}>{c}</option>)}
                            </select>
                        </label>
                        <label className="block text-sm">
                            <span className="text-muted-foreground">{t('moss.ai.language')}</span>
                            <select value={lang} onChange={e => setLang(e.target.value)} className="mt-1 w-full h-9 px-2 rounded-md border bg-card text-sm">
                                {langOptions.map(l => <option key={l} value={l}>{l}</option>)}
                            </select>
                        </label>
                        <label className="block text-sm">
                            <span className="text-muted-foreground">{t('moss.ai.source')}</span>
                            <Textarea value={source} onChange={e => setSource(e.target.value)} rows={12} className="mt-1 font-mono" spellCheck={false} />
                        </label>
                    </div>
                    <DialogFooter>
                        <button type="button" onClick={() => setAdding(false)} className="h-8 px-3 rounded-md border text-sm hover:bg-muted">{t('moss.ai.cancel')}</button>
                        <button
                            type="button"
                            disabled={add.isPending || !selected || !lang || !source.trim()}
                            onClick={() => add.mutate({ problem_code: selected, language: lang, source })}
                            className="h-8 px-3 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-50"
                        >
                            {t('moss.ai.save')}
                        </button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
