'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { adminLanguageApi, adminSubmissionApi, adminErrorMessage } from '@/lib/adminApi';
import { Link } from '@/navigation';
import { useAuth } from '@/components/providers/AuthProvider';
import {
    buildSubmissionUpdatePayload,
    toSubmissionEditValues,
    type SubmissionEditValues,
} from '@/lib/adminSubmission';
import { formatMemoryMb } from '@/lib/submissionDetail';
import { formatDate } from '@/lib/date';
import { hasPerm } from '@/lib/permissions';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { toast } from 'sonner';
import { getStatusVariant } from '@/lib/utils';
import {
    ArrowLeft,
    BrainCircuit,
    ExternalLink,
    History,
    Lock,
    Archive,
} from 'lucide-react';

const STATUS_CODES = ['QU', 'P', 'G', 'D', 'IE', 'CE', 'AB'];
const RESULT_CODES = ['AC', 'WA', 'TLE', 'MLE', 'OLE', 'IR', 'RTE', 'CE', 'IE', 'SC', 'AB'];
const field = 'h-9 px-3 rounded-md border bg-card text-sm focus:outline-none focus:ring-2 focus:ring-ring/40';

export default function AdminSubmissionDetailPage() {
    const t = useTranslations('Admin');
    const params = useParams();
    const router = useRouter();
    const id = Number(params.id);
    const queryClient = useQueryClient();
    const { user } = useAuth();
    // Every edit below is gated by judge.change_submission; lock/archive keep
    // their own Django permissions. Superusers hold all of them. The backend
    // enforces the same checks.
    const canEdit = hasPerm(user, 'judge.change_submission');
    const canLock = hasPerm(user, 'judge.lock_submission');
    const canArchive = hasPerm(user, 'judge.archive_submission');

    const { data: sub, isLoading } = useQuery({
        queryKey: ['admin-submission', id],
        queryFn: async () => (await adminSubmissionApi.detail(id)).data,
        enabled: Number.isFinite(id),
    });
    const { data: languages } = useQuery({
        queryKey: ['admin-languages', 'all'],
        queryFn: async () => (await adminLanguageApi.list(1, 100)).data.data,
    });

    const [values, setValues] = useState<SubmissionEditValues | null>(null);
    // Seed the form once the first response arrives and re-seed when the route
    // id changes; later refetches keep the user's edits instead of
    // overwriting them.
    const [seededId, setSeededId] = useState<number | null>(null);
    if (sub && seededId !== id) {
        setSeededId(id);
        setValues(toSubmissionEditValues(sub));
    }

    const updateMutation = useMutation({
        mutationFn: (payload: Record<string, unknown>) => adminSubmissionApi.update(id, payload),
        onSuccess: () => {
            toast.success(t('submissionDetail.saved'));
            queryClient.invalidateQueries({ queryKey: ['admin-submission', id] });
            queryClient.invalidateQueries({ queryKey: ['admin-submissions'] });
        },
        onError: (err) => toast.error(adminErrorMessage(err, t('submissionDetail.saveFailed'))),
    });

    if (isLoading) {
        return (
            <div className="space-y-6">
                <Skeleton className="h-10 w-48" />
                <Skeleton className="h-64 rounded-md" />
                <Skeleton className="h-96 rounded-md" />
            </div>
        );
    }

    if (!sub || !values) {
        return (
            <div className="text-center py-12">
                <BrainCircuit size={48} className="mx-auto text-muted-foreground opacity-20" />
                <h2 className="text-xl font-semibold mt-4">{t('submissionDetail.notFoundTitle')}</h2>
                <Link href="/admin/submissions" className="text-accent hover:underline mt-2 inline-block">
                    {t('submissionDetail.backToList')}
                </Link>
            </div>
        );
    }

    const set = <K extends keyof SubmissionEditValues>(key: K, value: SubmissionEditValues[K]) =>
        setValues(v => (v ? { ...v, [key]: value } : v));
    const payload = buildSubmissionUpdatePayload(sub, values);

    return (
        <div className="space-y-6">
            <div className="flex flex-wrap items-center gap-4">
                <button onClick={() => router.back()} className="p-2 rounded-lg hover:bg-card border" aria-label={t('submissionDetail.back')}>
                    <ArrowLeft size={20} />
                </button>
                <div className="flex-1">
                    <h1 className="text-3xl font-semibold flex items-center gap-3">
                        <BrainCircuit className="text-primary" size={30} />
                        {t('submissionDetail.title', { id: sub.id })}
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        {t('submissionDetail.subtitle')}
                    </p>
                </div>
                <div className="flex flex-wrap gap-2">
                    <Badge variant={getStatusVariant(sub.result || sub.status)} className="uppercase">
                        {sub.result || sub.status}
                    </Badge>
                    {sub.is_locked && (
                        <Badge variant="warning" className="flex items-center gap-1">
                            <Lock size={14} /> {t('submissionDetail.locked')}
                        </Badge>
                    )}
                    {sub.is_archived && (
                        <Badge variant="secondary" className="flex items-center gap-1">
                            <Archive size={14} /> {t('submissionDetail.archived')}
                        </Badge>
                    )}
                </div>
            </div>

            {/* Read-only header */}
            <div className="bg-card rounded-md border p-6 space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4 text-sm">
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.user')}</div>
                        <Link href={`/user/${sub.user}`} className="font-medium text-accent hover:underline">{sub.user}</Link>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.problem')}</div>
                        <Link href={`/problem/${sub.problem}`} className="font-medium text-accent hover:underline">
                            {sub.problem} · {sub.problem_name}
                        </Link>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.language')}</div>
                        <div className="font-medium">{sub.language_name} <span className="text-muted-foreground font-mono text-xs">{sub.language}</span></div>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.judgedOn')}</div>
                        <div className="font-medium">{sub.judged_on || t('common.notAvailable')}</div>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.status')}</div>
                        <div className="font-mono font-medium">{sub.status}</div>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.result')}</div>
                        <div className="font-mono font-medium">{sub.result || t('common.notAvailable')}</div>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.points')}</div>
                        <div className="font-medium">{sub.points ?? t('common.notAvailable')} · {sub.case_points}/{sub.case_total}</div>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.timeMemory')}</div>
                        <div className="font-medium">
                            {sub.time !== null ? `${sub.time.toFixed(2)}s` : '-'} · {formatMemoryMb(sub.memory)}
                        </div>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.date')}</div>
                        <div className="font-medium">{formatDate(sub.date, 'datetime')}</div>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.judgedDate')}</div>
                        <div className="font-medium">{sub.judged_date ? formatDate(sub.judged_date, 'datetime') : t('common.notAvailable')}</div>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.rejudgedDate')}</div>
                        <div className="font-medium">{sub.rejudged_date ? formatDate(sub.rejudged_date, 'datetime') : t('common.notAvailable')}</div>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.lockedAfter')}</div>
                        <div className="font-medium">{sub.locked_after ? formatDate(sub.locked_after, 'datetime') : t('common.notAvailable')}</div>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.contest')}</div>
                        <div className="font-medium">
                            {sub.contest_object_id ? (
                                sub.contest_key ? (
                                    <Link href={`/contest/${sub.contest_key}`} className="text-accent hover:underline">
                                        {sub.contest_key}{sub.contest_name ? ` · ${sub.contest_name}` : ''}
                                    </Link>
                                ) : (
                                    `#${sub.contest_object_id}`
                                )
                            ) : t('common.notAvailable')}
                        </div>
                    </div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('submissionDetail.pretested')}</div>
                        <div className="font-medium">{sub.is_pretested ? t('common.yes') : t('common.no')}</div>
                    </div>
                </div>

                {sub.error && (
                    <pre className="text-xs font-mono whitespace-pre-wrap break-all rounded border bg-destructive/5 text-destructive p-3 max-h-60 overflow-auto">
                        {sub.error}
                    </pre>
                )}

                <div className="flex flex-wrap gap-3 pt-2">
                    <Link href={`/submission/${sub.id}`} className="inline-flex items-center gap-2 text-sm text-accent hover:underline">
                        <ExternalLink size={14} /> {t('submissionDetail.viewPublic')}
                    </Link>
                    <Link href="/admin/audit-logs?resource=Submission" className="inline-flex items-center gap-2 text-sm text-accent hover:underline">
                        <History size={14} /> {t('submissionDetail.viewAudit')}
                    </Link>
                </div>
            </div>

            {/* Editable fields */}
            <div className="bg-card rounded-md border p-6 space-y-4">
                <h2 className="font-semibold text-lg">{t('submissionDetail.editTitle')}</h2>

                {!canEdit ? (
                    <p className="text-sm text-muted-foreground">{t('submissionDetail.editHint')}</p>
                ) : (<>
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                    <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                        {t('submissionDetail.language')}
                        <select className={field} value={values.language_id} onChange={e => set('language_id', Number(e.target.value))}>
                            {languages?.map(l => <option key={l.id} value={l.id}>{l.name} ({l.key})</option>)}
                            {!languages?.some(l => l.id === values.language_id) && (
                                <option value={values.language_id}>{sub.language_name}</option>
                            )}
                        </select>
                    </label>
                    <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                        {t('submissionDetail.status')}
                        <select className={field} value={values.status} onChange={e => set('status', e.target.value)}>
                            {STATUS_CODES.map(code => <option key={code} value={code}>{code}</option>)}
                        </select>
                    </label>
                    <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                        {t('submissionDetail.result')}
                        <select className={field} value={values.result} onChange={e => set('result', e.target.value)}>
                            <option value="">{t('submissionDetail.noResult')}</option>
                            {RESULT_CODES.map(code => <option key={code} value={code}>{code}</option>)}
                        </select>
                    </label>
                    <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                        {t('submissionDetail.points')}
                        <input type="number" step="any" className={field} value={values.points} onChange={e => set('points', e.target.value)} />
                    </label>
                    <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                        {t('submissionDetail.casePoints')}
                        <input type="number" step="any" className={field} value={values.case_points} onChange={e => set('case_points', e.target.value)} />
                    </label>
                    <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                        {t('submissionDetail.caseTotal')}
                        <input type="number" step="any" className={field} value={values.case_total} onChange={e => set('case_total', e.target.value)} />
                    </label>
                    <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                        {t('submissionDetail.judgedOnId')}
                        <input type="number" min="1" className={field} value={values.judged_on_id} onChange={e => set('judged_on_id', e.target.value)} />
                    </label>
                    <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                        {t('submissionDetail.judgedDate')}
                        <input type="datetime-local" className={field} value={values.judged_date} onChange={e => set('judged_date', e.target.value)} />
                    </label>
                    <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                        {t('submissionDetail.contestId')}
                        <input type="number" min="1" className={field} value={values.contest_object_id} onChange={e => set('contest_object_id', e.target.value)} />
                    </label>
                    <label className="flex flex-col gap-1 text-xs text-muted-foreground md:col-span-2 xl:col-span-1">
                        {t('submissionDetail.lockedAfter')}
                        <input
                            type="datetime-local"
                            className={field}
                            value={values.locked_after}
                            disabled={!canLock}
                            onChange={e => set('locked_after', e.target.value)}
                        />
                    </label>
                </div>

                {!canLock && (
                    <p className="text-xs text-muted-foreground">{t('submissionDetail.lockHint')}</p>
                )}

                <label className="flex items-center gap-2 text-sm">
                    <input
                        type="checkbox"
                        checked={values.is_archived}
                        disabled={!canArchive}
                        onChange={e => set('is_archived', e.target.checked)}
                    />
                    {t('submissionDetail.isArchived')}
                </label>

                <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                    {t('submissionDetail.error')}
                    <textarea
                        rows={4}
                        className="w-full rounded-md border bg-background p-3 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-ring/40"
                        value={values.error}
                        onChange={e => set('error', e.target.value)}
                    />
                </label>

                <label className="flex flex-col gap-1 text-xs text-muted-foreground">
                    {t('submissionDetail.source')}
                    <textarea
                        rows={14}
                        className="w-full rounded-md border bg-background p-3 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-ring/40"
                        value={values.source}
                        onChange={e => set('source', e.target.value)}
                    />
                </label>

                <div className="flex items-center justify-end gap-3">
                    <button
                        onClick={() => setValues(toSubmissionEditValues(sub))}
                        disabled={updateMutation.isPending || Object.keys(payload).length === 0}
                        className="h-9 px-4 rounded-md border text-sm font-medium hover:bg-muted disabled:opacity-50"
                    >
                        {t('common.cancel')}
                    </button>
                    <button
                        onClick={() => updateMutation.mutate(payload)}
                        disabled={updateMutation.isPending || Object.keys(payload).length === 0}
                        className="h-9 px-4 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-50"
                    >
                        {updateMutation.isPending ? t('common.saving') : t('common.saveChanges')}
                    </button>
                </div>
                </>)}
            </div>

            {/* Test cases */}
            <div className="bg-card rounded-md border overflow-hidden">
                <h2 className="font-semibold text-lg px-6 pt-6 pb-3">{t('submissionDetail.testCases', { count: sub.test_cases.length })}</h2>
                <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                        <thead className="bg-secondary/60 text-left text-xs text-muted-foreground">
                            <tr>
                                <th className="px-4 py-2.5 font-semibold">#</th>
                                <th className="px-4 py-2.5 font-semibold">{t('submissionDetail.status')}</th>
                                <th className="px-4 py-2.5 font-semibold">{t('submissionDetail.time')}</th>
                                <th className="px-4 py-2.5 font-semibold">{t('submissionDetail.memory')}</th>
                                <th className="px-4 py-2.5 font-semibold">{t('submissionDetail.points')}</th>
                                <th className="px-4 py-2.5 font-semibold">{t('submissionDetail.feedback')}</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y">
                            {sub.test_cases.length === 0 ? (
                                <tr><td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">{t('submissionDetail.noTestCases')}</td></tr>
                            ) : sub.test_cases.map(tc => (
                                <tr key={tc.id}>
                                    <td className="px-4 py-2 tabular-nums">{tc.case}</td>
                                    <td className="px-4 py-2 font-mono">{tc.status}</td>
                                    <td className="px-4 py-2 tabular-nums">{tc.time !== null ? `${tc.time.toFixed(3)}s` : '-'}</td>
                                    <td className="px-4 py-2 tabular-nums">{formatMemoryMb(tc.memory)}</td>
                                    <td className="px-4 py-2 tabular-nums">{tc.points ?? '-'}{tc.total !== null ? ` / ${tc.total}` : ''}</td>
                                    <td className="px-4 py-2 font-mono text-xs">{tc.feedback || '-'}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    );
}