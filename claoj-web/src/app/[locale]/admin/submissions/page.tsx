'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { adminSubmissionApi } from '@/lib/adminApi';
import { AdminSubmission } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import { toast } from 'sonner';
import { PaginationBar } from '@/components/ui/PaginationBar';
import {
    BrainCircuit,
    User,
    Hash,
    RefreshCw,
    StopCircle,
    Calculator,
    AlertTriangle
} from 'lucide-react';
import { getStatusVariant, cn } from '@/lib/utils';

export default function AdminSubmissionPage() {
    const t = useTranslations('Admin');
    const [page, setPage] = useState(1);
    const [userFilter, setUserFilter] = useState('');
    const [problemFilter, setProblemFilter] = useState('');
    const [resultFilter, setResultFilter] = useState('');
    const [langFilter, setLangFilter] = useState('');
    const [showBatchModal, setShowBatchModal] = useState(false);
    const [batchPreview, setBatchPreview] = useState<{ count: number; message: string } | null>(null);
    const [isExecutingBatch, setIsExecutingBatch] = useState(false);
    const queryClient = useQueryClient();

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-submissions', page, userFilter, problemFilter, resultFilter, langFilter],
        queryFn: async () => {
            const res = await api.get<{
                data: AdminSubmission[];
                total: number;
                page: number;
                page_size: number;
            }>(`/admin/submissions?page=${page}&page_size=50&user=${userFilter}&problem=${problemFilter}&result=${resultFilter}&language=${langFilter}`);
            return res.data;
        }
    });

    const rejudgeMutation = useMutation({
        mutationFn: (id: number) => adminSubmissionApi.rejudge(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-submissions'] });
        }
    });

    const abortMutation = useMutation({
        mutationFn: (id: number) => adminSubmissionApi.abort(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-submissions'] });
        }
    });

    const rescoreMutation = useMutation({
        mutationFn: (id: number) => adminSubmissionApi.rescore(id),
        onSuccess: () => {
            toast.success(t('submissions.rescoreSuccess'));
            queryClient.invalidateQueries({ queryKey: ['admin-submissions'] });
        },
        onError: () => toast.error(t('submissions.rescoreError')),
    });

    // Recomputes scores from stored per-case results (e.g. after changing
    // point values) without sending anything back to the judges.
    const rescoreAllMutation = useMutation({
        mutationFn: (code: string) => adminSubmissionApi.rescoreAll(code),
        onSuccess: (res) => {
            toast.success(t('submissions.rescoreAllSuccess', { count: res.data.rescored, total: res.data.total }));
            queryClient.invalidateQueries({ queryKey: ['admin-submissions'] });
        },
        onError: () => toast.error(t('submissions.rescoreError')),
    });

    const submissions = data?.data || [];

    const results = ['AC', 'WA', 'TLE', 'MLE', 'OLE', 'RE', 'IE', 'CE', 'AB', 'QU', 'P', 'G'];

    // Batch rejudge preview
    const handleBatchPreview = async () => {
        setIsExecutingBatch(false);
        const filters: any = {};

        // For text-based filters (username, problem code), we need to resolve to IDs first
        // For now, pass them as string filters and let backend handle resolution
        if (userFilter) filters.username = userFilter;
        if (problemFilter) filters.problem_code = problemFilter;
        if (resultFilter) filters.result = resultFilter;
        if (langFilter) filters.language = langFilter;

        try {
            const res = await adminSubmissionApi.batchRejudge({
                filters,
                dry_run: true
            });
            setBatchPreview({ count: res.data.count, message: res.data.message });
        } catch (err: any) {
            setBatchPreview({ count: 0, message: err.response?.data?.error || t('submissions.previewError') });
        }
    };

    // Execute batch rejudge
    const handleBatchExecute = async () => {
        if (!batchPreview || batchPreview.count === 0) return;
        if (!confirm(t('submissions.batchRejudgeConfirm', { count: batchPreview.count }))) return;

        setIsExecutingBatch(true);
        const filters: any = {};
        if (userFilter) filters.username = userFilter;
        if (problemFilter) filters.problem_code = problemFilter;
        if (resultFilter) filters.result = resultFilter;
        if (langFilter) filters.language = langFilter;

        try {
            await adminSubmissionApi.batchRejudge({
                filters,
                dry_run: false
            });
            alert(t('submissions.batchRejudgeSuccess', { count: batchPreview.count }));
            setShowBatchModal(false);
            setBatchPreview(null);
            refetch();
        } catch (err: any) {
            alert(`${t('submissions.errorPrefix')}${err.response?.data?.error || t('submissions.batchRejudgeError')}`);
        } finally {
            setIsExecutingBatch(false);
        }
    };

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-semibold flex items-center gap-3">
                        <BrainCircuit className="text-primary" size={32} />
                        {t('submissions.title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">{t('submissions.subtitle')}</p>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                    <button
                        onClick={() => {
                            if (confirm(t('submissions.rescoreAllConfirm', { problem: problemFilter }))) {
                                rescoreAllMutation.mutate(problemFilter);
                            }
                        }}
                        disabled={!problemFilter || rescoreAllMutation.isPending}
                        title={problemFilter ? undefined : t('submissions.rescoreAllHint')}
                        className="px-4 py-2 rounded-xl border bg-card hover:bg-muted transition-colors text-sm font-medium flex items-center gap-2 disabled:opacity-50"
                    >
                        <Calculator size={18} /> {t('submissions.rescoreAllButton')}
                    </button>
                    <button
                        onClick={() => setShowBatchModal(true)}
                        className="px-4 py-2 rounded-xl bg-primary/10 text-primary hover:bg-primary/20 transition-colors text-sm font-medium flex items-center gap-2"
                    >
                        <RefreshCw size={18} /> {t('submissions.batchRejudgeButton')}
                    </button>
                    <button
                        onClick={() => {
                            setUserFilter('');
                            setProblemFilter('');
                            setResultFilter('');
                            setLangFilter('');
                            setPage(1);
                            refetch();
                        }}
                        className="px-3 py-2 rounded-xl bg-muted/50 hover:bg-muted transition-colors text-sm font-medium flex items-center gap-2"
                    >
                        <RefreshCw size={16} /> {t('submissions.resetFiltersButton')}
                    </button>
                </div>
            </div>

            {/* Batch Rejudge Modal */}
            {showBatchModal && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 ">
                    <div className="bg-card rounded-md p-6 w-full max-w-md border shadow-lg">
                        <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
                            <RefreshCw className={cn(isExecutingBatch ? 'animate-spin' : '')} size={24} />
                            {t('submissions.batchRejudgeButton')}
                        </h2>
                        <p className="text-muted-foreground text-sm mb-4">
                            {t('submissions.batchRejudgeDesc')}
                        </p>

                        {batchPreview && (
                            <div className={cn(
                                "p-4 rounded-xl mb-4 flex items-center gap-3",
                                batchPreview.count > 0 ? "bg-warning/10 text-warning" : "bg-muted"
                            )}>
                                <AlertTriangle size={20} />
                                <div>
                                    <div className="font-medium">{t('submissions.countSubmissions', { count: batchPreview.count })}</div>
                                    <div className="text-sm opacity-75">{batchPreview.message}</div>
                                </div>
                            </div>
                        )}

                        <div className="flex gap-2 mt-6">
                            <button
                                onClick={handleBatchPreview}
                                disabled={isExecutingBatch}
                                className="flex-1 px-4 py-2 rounded-xl bg-primary text-primary-foreground font-medium disabled:opacity-50"
                            >
                                {batchPreview ? t('submissions.refreshPreviewButton') : t('submissions.previewButton')}
                            </button>
                            {batchPreview && batchPreview.count > 0 && (
                                <button
                                    onClick={handleBatchExecute}
                                    disabled={isExecutingBatch}
                                    className="flex-1 px-4 py-2 rounded-xl bg-destructive text-destructive-foreground font-medium disabled:opacity-50"
                                >
                                    {isExecutingBatch ? t('submissions.executingButton') : t('submissions.executeButton')}
                                </button>
                            )}
                            <button
                                onClick={() => {
                                    setShowBatchModal(false);
                                    setBatchPreview(null);
                                }}
                                className="px-4 py-2 rounded-xl bg-muted font-medium"
                            >
                                {t('common.cancel')}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Filters */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 p-6 rounded-md bg-card border">
                <div className="space-y-2">
                    <label className="text-xs font-medium uppercase text-muted-foreground ml-1">{t('submissions.usernameLabel')}</label>
                    <div className="relative">
                        <User className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={16} />
                        <input
                            type="text"
                            placeholder={t('submissions.searchUserPlaceholder')}
                            className="w-full h-10 pl-10 rounded-xl bg-muted/30 border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={userFilter}
                            onChange={(e) => { setUserFilter(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-xs font-medium uppercase text-muted-foreground ml-1">{t('submissions.problemCodeLabel')}</label>
                    <div className="relative">
                        <Hash className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={16} />
                        <input
                            type="text"
                            placeholder={t('submissions.problemCodePlaceholder')}
                            className="w-full h-10 pl-10 rounded-xl bg-muted/30 border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={problemFilter}
                            onChange={(e) => { setProblemFilter(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-xs font-medium uppercase text-muted-foreground ml-1">{t('submissions.verdictLabel')}</label>
                    <select
                        className="w-full h-10 px-3 rounded-xl bg-muted/30 border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={resultFilter}
                        onChange={(e) => { setResultFilter(e.target.value); setPage(1); }}
                    >
                        <option value="">{t('submissions.allResultsOption')}</option>
                        {results.map(r => <option key={r} value={r}>{r}</option>)}
                    </select>
                </div>

                <div className="space-y-2">
                    <label className="text-xs font-medium uppercase text-muted-foreground ml-1">{t('submissions.languageLabel')}</label>
                    <input
                        type="text"
                        placeholder={t('submissions.languageFilterPlaceholder')}
                        className="w-full h-10 px-3 rounded-xl bg-muted/30 border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={langFilter}
                        onChange={(e) => { setLangFilter(e.target.value); setPage(1); }}
                    />
                </div>
            </div>

            {/* Submission Table */}
            <div className="bg-card rounded-md border overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="w-full text-left">
                        <thead>
                            <tr className="bg-muted/30 border-b">
                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground w-40">{t('submissions.colUser')}</th>
                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground">{t('submissions.colProblem')}</th>
                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-32">{t('submissions.colStatus')}</th>
                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center">{t('submissions.colScore')}</th>
                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-24">{t('submissions.colTime')}</th>
                                <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-right">{t('common.actions')}</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-muted/50">
                            {isLoading ? (
                                Array.from({ length: 15 }).map((_, i) => (
                                    <tr key={i}>
                                        <td colSpan={6} className="px-6 py-6"><Skeleton className="h-10 w-full rounded-xl" /></td>
                                    </tr>
                                ))
                            ) : (
                                submissions.map((s) => (
                                    <tr key={s.id} className="hover:bg-muted/10 transition-colors">
                                        <td className="px-6 py-4">
                                            <Link href={`/user/${s.user}`} className="flex items-center gap-3 group outline-none">
                                                <div className="w-9 h-9 rounded-xl bg-muted flex items-center justify-center text-muted-foreground font-medium text-xs">
                                                    {s.user[0]?.toUpperCase()}
                                                </div>
                                                <span className="font-medium text-sm">{s.user}</span>
                                            </Link>
                                        </td>
                                        <td className="px-6 py-4">
                                            <Link href={`/submission/${s.id}`} className="inline-flex flex-col gap-0.5 outline-none hover:text-primary transition-colors">
                                                <span className="font-medium text-sm tracking-tight">{s.problem}</span>
                                                <span className="text-xs uppercase text-muted-foreground opacity-40">{t('submissions.problemCodeLabel')}</span>
                                            </Link>
                                        </td>
                                        <td className="px-6 py-4 text-center">
                                            <Badge variant={getStatusVariant(s.result || s.status)} className="px-4 py-1.5 rounded-full text-xs font-medium uppercase shadow-sm">
                                                {s.result || s.status}
                                            </Badge>
                                        </td>
                                        <td className="px-6 py-4 text-center">
                                            <div className="inline-flex items-center justify-center w-12 h-8 rounded-xl font-medium text-xs border">
                                                {s.score !== null ? Math.round(s.score) : '-'}
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 text-center">
                                            {s.time !== null ? (
                                                <span className="text-xs font-mono">{(s.time * 1000).toFixed(0)}ms</span>
                                            ) : (
                                                <span className="text-xs text-muted-foreground">-</span>
                                            )}
                                        </td>
                                        <td className="px-6 py-4 text-right">
                                            <div className="flex gap-2">
                                                <button
                                                    onClick={() => {
                                                        if (confirm(t('submissions.rejudgeConfirm'))) {
                                                            rejudgeMutation.mutate(s.id);
                                                        }
                                                    }}
                                                    disabled={rejudgeMutation.isPending}
                                                    className="px-3 py-1.5 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 transition-colors text-xs font-medium flex items-center gap-1"
                                                >
                                                    <RefreshCw size={14} className={rejudgeMutation.isPending && rejudgeMutation.variables === s.id ? 'animate-spin' : ''} />
                                                    {t('submissions.rejudgeButton')}
                                                </button>
                                                <button
                                                    onClick={() => rescoreMutation.mutate(s.id)}
                                                    disabled={rescoreMutation.isPending && rescoreMutation.variables === s.id}
                                                    title={t('submissions.rescoreHint')}
                                                    className="px-3 py-1.5 rounded-lg border hover:bg-muted transition-colors text-xs font-medium flex items-center gap-1 disabled:opacity-50"
                                                >
                                                    <Calculator size={14} />
                                                    {t('submissions.rescoreButton')}
                                                </button>
                                                {(s.status === 'QU' || s.status === 'P' || s.status === 'G') && (
                                                    <button
                                                        onClick={() => {
                                                            if (confirm(t('submissions.abortConfirm'))) {
                                                                abortMutation.mutate(s.id);
                                                            }
                                                        }}
                                                        disabled={abortMutation.isPending}
                                                        className="px-3 py-1.5 rounded-lg bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors text-xs font-medium flex items-center gap-1"
                                                    >
                                                        <StopCircle size={14} className={abortMutation.isPending ? 'animate-spin' : ''} />
                                                        {t('submissions.abortButton')}
                                                    </button>
                                                )}
                                            </div>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>

            </div>

            <PaginationBar page={page} onPageChange={setPage} total={data?.total} pageSize={50} />
        </div>
    );
}
