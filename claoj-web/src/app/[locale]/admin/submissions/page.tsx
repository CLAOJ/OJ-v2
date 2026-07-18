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
import {
    Search,
    BrainCircuit,
    User,
    Hash,
    Clock,
    Filter,
    RefreshCw,
    ChevronLeft,
    ChevronRight,
    StopCircle,
    CheckCircle,
    AlertTriangle
} from 'lucide-react';
import { getStatusColor, getStatusVariant, cn } from '@/lib/utils';

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
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <BrainCircuit className="text-primary" size={32} />
                        {t('submissions.title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">{t('submissions.subtitle')}</p>
                </div>

                <div className="flex items-center gap-2">
                    <button
                        onClick={() => setShowBatchModal(true)}
                        className="px-4 py-2 rounded-xl bg-primary/10 text-primary hover:bg-primary/20 transition-colors text-sm font-bold flex items-center gap-2"
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
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
                    <div className="bg-card rounded-2xl p-6 w-full max-w-md border shadow-2xl">
                        <h2 className="text-xl font-bold mb-4 flex items-center gap-2">
                            <RefreshCw className={cn(isExecutingBatch ? 'animate-spin' : '')} size={24} />
                            {t('submissions.batchRejudgeButton')}
                        </h2>
                        <p className="text-muted-foreground text-sm mb-4">
                            {t('submissions.batchRejudgeDesc')}
                        </p>

                        {batchPreview && (
                            <div className={cn(
                                "p-4 rounded-xl mb-4 flex items-center gap-3",
                                batchPreview.count > 0 ? "bg-amber-500/10 text-amber-600" : "bg-muted"
                            )}>
                                <AlertTriangle size={20} />
                                <div>
                                    <div className="font-bold">{t('submissions.countSubmissions', { count: batchPreview.count })}</div>
                                    <div className="text-sm opacity-75">{batchPreview.message}</div>
                                </div>
                            </div>
                        )}

                        <div className="flex gap-2 mt-6">
                            <button
                                onClick={handleBatchPreview}
                                disabled={isExecutingBatch}
                                className="flex-1 px-4 py-2 rounded-xl bg-primary text-primary-foreground font-bold disabled:opacity-50"
                            >
                                {batchPreview ? t('submissions.refreshPreviewButton') : t('submissions.previewButton')}
                            </button>
                            {batchPreview && batchPreview.count > 0 && (
                                <button
                                    onClick={handleBatchExecute}
                                    disabled={isExecutingBatch}
                                    className="flex-1 px-4 py-2 rounded-xl bg-destructive text-destructive-foreground font-bold disabled:opacity-50"
                                >
                                    {isExecutingBatch ? t('submissions.executingButton') : t('submissions.executeButton')}
                                </button>
                            )}
                            <button
                                onClick={() => {
                                    setShowBatchModal(false);
                                    setBatchPreview(null);
                                }}
                                className="px-4 py-2 rounded-xl bg-muted font-bold"
                            >
                                {t('common.cancel')}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Filters */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 p-6 rounded-2xl bg-card border">
                <div className="space-y-2">
                    <label className="text-xs font-bold uppercase text-muted-foreground ml-1">{t('submissions.usernameLabel')}</label>
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
                    <label className="text-xs font-bold uppercase text-muted-foreground ml-1">{t('submissions.problemCodeLabel')}</label>
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
                    <label className="text-xs font-bold uppercase text-muted-foreground ml-1">{t('submissions.verdictLabel')}</label>
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
                    <label className="text-xs font-bold uppercase text-muted-foreground ml-1">{t('submissions.languageLabel')}</label>
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
            <div className="bg-card rounded-2xl border overflow-hidden shadow-2xl shadow-primary/5">
                <div className="overflow-x-auto">
                    <table className="w-full text-left">
                        <thead>
                            <tr className="bg-muted/30 border-b">
                                <th className="px-6 py-4 text-xs font-bold uppercase tracking-[0.2em] text-muted-foreground w-40">{t('submissions.colUser')}</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase tracking-[0.2em] text-muted-foreground">{t('submissions.colProblem')}</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase tracking-[0.2em] text-muted-foreground text-center w-32">{t('submissions.colStatus')}</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase tracking-[0.2em] text-muted-foreground text-center">{t('submissions.colScore')}</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase tracking-[0.2em] text-muted-foreground text-center w-24">{t('submissions.colTime')}</th>
                                <th className="px-6 py-4 text-xs font-bold uppercase tracking-[0.2em] text-muted-foreground text-right">{t('common.actions')}</th>
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
                                                <div className="w-9 h-9 rounded-xl bg-muted flex items-center justify-center text-muted-foreground font-bold text-xs">
                                                    {s.user[0]?.toUpperCase()}
                                                </div>
                                                <span className="font-bold text-sm">{s.user}</span>
                                            </Link>
                                        </td>
                                        <td className="px-6 py-4">
                                            <Link href={`/submissions/${s.id}`} className="inline-flex flex-col gap-0.5 outline-none hover:text-primary transition-colors">
                                                <span className="font-bold text-sm tracking-tight">{s.problem}</span>
                                                <span className="text-xs uppercase text-muted-foreground opacity-40">{t('submissions.problemCodeLabel')}</span>
                                            </Link>
                                        </td>
                                        <td className="px-6 py-4 text-center">
                                            <Badge variant={getStatusVariant(s.status)} className="px-4 py-1.5 rounded-full text-xs font-bold uppercase shadow-sm">
                                                {s.status}
                                            </Badge>
                                        </td>
                                        <td className="px-6 py-4 text-center">
                                            <div className="inline-flex items-center justify-center w-12 h-8 rounded-xl font-bold text-xs border">
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
                                                    className="px-3 py-1.5 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 transition-colors text-xs font-bold flex items-center gap-1"
                                                >
                                                    <RefreshCw size={14} className={rejudgeMutation.isPending ? 'animate-spin' : ''} />
                                                    {t('submissions.rejudgeButton')}
                                                </button>
                                                {(s.status === 'P' || s.status === 'G') && (
                                                    <button
                                                        onClick={() => {
                                                            if (confirm(t('submissions.abortConfirm'))) {
                                                                abortMutation.mutate(s.id);
                                                            }
                                                        }}
                                                        disabled={abortMutation.isPending}
                                                        className="px-3 py-1.5 rounded-lg bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors text-xs font-bold flex items-center gap-1"
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

                {/* Pagination */}
                {submissions.length > 0 && (
                    <div className="flex items-center justify-between px-6 py-4 border-t bg-muted/30">
                        <div className="text-sm text-muted-foreground">
                            {t('submissions.showingCount', { shown: submissions.length, total: data?.total || 0 })}
                        </div>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={() => setPage(p => Math.max(1, p - 1))}
                                disabled={page === 1}
                                className="w-10 h-10 rounded-xl bg-card border flex items-center justify-center hover:bg-muted disabled:opacity-50 transition-all"
                            >
                                <ChevronLeft size={18} />
                            </button>
                            <div className="h-10 px-4 rounded-xl bg-primary text-primary-foreground font-bold text-sm flex items-center shadow-lg shadow-primary/20">
                                {page}
                            </div>
                            <button
                                onClick={() => setPage(p => p + 1)}
                                disabled={submissions.length < 50}
                                className="w-10 h-10 rounded-xl bg-card border flex items-center justify-center hover:bg-muted disabled:opacity-50 transition-all"
                            >
                                <ChevronRight size={18} />
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}
