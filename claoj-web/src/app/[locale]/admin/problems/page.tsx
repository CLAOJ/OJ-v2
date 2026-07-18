'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { AdminProblem } from '@/types';
import { adminProblemApi } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import {
    Search,
    Code2,
    Ban,
    Edit,
    Trash2,
    Database,
    Clock,
    Plus,
    Copy
} from 'lucide-react';
import { Dialog } from '@/components/ui/Dialog';

export default function AdminProblemPage() {
    const t = useTranslations('Admin');
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState('');
    const [cloneModalOpen, setCloneModalOpen] = useState(false);
    const [problemToClone, setProblemToClone] = useState<{code: string; name: string} | null>(null);

    const queryClient = useQueryClient();

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-problems', page, search],
        queryFn: async () => {
            const res = await api.get<{
                data: AdminProblem[];
                total: number;
                page: number;
                page_size: number;
            }>(`/admin/problems?page=${page}&page_size=50&search=${search}`);
            return res.data;
        }
    });

    const deleteMutation = useMutation({
        mutationFn: (code: string) => adminProblemApi.delete(code),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-problems'] });
        }
    });

    const problems = data?.data || [];

    const filteredProblems = problems.filter(p =>
        p.name.toLowerCase().includes(search.toLowerCase()) ||
        p.code.toLowerCase().includes(search.toLowerCase())
    );

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Code2 className="text-primary" size={32} />
                        {t('problems.title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">{t('problems.subtitle')}</p>
                </div>

                <div className="flex items-center gap-3">
                    <div className="relative w-full md:w-80">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                        <input
                            type="text"
                            placeholder={t('problems.searchPlaceholder')}
                            className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                        />
                    </div>
                    <Link
                        href="/admin/problems/create"
                        className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2 font-medium"
                    >
                        <Plus size={18} />
                        {t('common.create')}
                    </Link>
                </div>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-20 rounded-2xl" />)}
                </div>
            ) : (
                <div className="bg-card rounded-2xl border overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="w-full text-left">
                            <thead className="bg-muted/50 border-b">
                                <tr>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('problems.colProblem')}</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('problems.colStats')}</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('problems.colGroup')}</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground text-right">{t('common.actions')}</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {filteredProblems.length === 0 ? (
                                    <tr>
                                        <td colSpan={4} className="px-6 py-12 text-center text-muted-foreground">
                                            {t('problems.noProblemsFound')}
                                        </td>
                                    </tr>
                                ) : (
                                    filteredProblems.map((problem) => (
                                        <tr key={problem.code} className="hover:bg-muted/30 transition-colors">
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-3 mb-1">
                                                    <Link
                                                        href={`/problems/${problem.code}`}
                                                        className="font-bold text-sm hover:text-primary transition-colors"
                                                    >
                                                        {problem.name}
                                                    </Link>
                                                    {problem.partial && (
                                                        <Badge variant="secondary" className="text-[10px]">{t('problems.partialBadge')}</Badge>
                                                    )}
                                                </div>
                                                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                                    <span className="font-mono">{problem.code}</span>
                                                    {problem.is_public ? (
                                                        <Badge variant="success" className="text-[10px]">{t('problems.publicBadge')}</Badge>
                                                    ) : (
                                                        <Badge variant="destructive" className="text-[10px]">{t('problems.privateBadge')}</Badge>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex flex-col gap-1 text-sm">
                                                    <span className="text-muted-foreground font-bold">
                                                        {t('problems.pointsLabel', { count: problem.points.toFixed(1) })}
                                                    </span>
                                                    <span className="text-xs text-muted-foreground">
                                                        {t('problems.acRateLabel', { rate: Math.round(problem.ac_rate * 100) })}
                                                    </span>
                                                    <span className="text-xs text-muted-foreground">
                                                        {t('problems.usersLabel', { count: problem.user_count })}
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="text-sm text-muted-foreground">
                                                    {problem.group_name || t('problems.uncategorized')}
                                                </div>
                                                <div className="text-xs text-muted-foreground mt-1">
                                                    {problem.is_manually_managed ? t('problems.manuallyManaged') : t('problems.autoManaged')}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <div className="flex justify-end gap-2">
                                                    <Link
                                                        href={`/admin/problems/${problem.code}/edit`}
                                                        className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('problems.editTitle')}
                                                    >
                                                        <Edit size={18} />
                                                    </Link>
                                                    <Link
                                                        href={`/admin/problems/${problem.code}/data`}
                                                        className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('problems.manageDataTitle')}
                                                    >
                                                        <Database size={18} />
                                                    </Link>
                                                    <button
                                                        onClick={() => {
                                                            setProblemToClone({ code: problem.code, name: problem.name });
                                                            setCloneModalOpen(true);
                                                        }}
                                                        className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('problems.cloneTitle')}
                                                    >
                                                        <Copy size={18} />
                                                    </button>
                                                    <button
                                                        onClick={() => deleteMutation.mutate(problem.code)}
                                                        disabled={deleteMutation.isPending}
                                                        className="p-2 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                                        title={t('problems.deleteTitle')}
                                                    >
                                                        <Trash2 size={18} />
                                                    </button>
                                                </div>
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>

                    {/* Pagination */}
                    {filteredProblems.length > 0 && (
                        <div className="flex items-center justify-between px-6 py-4 border-t bg-muted/30">
                            <div className="text-sm text-muted-foreground">
                                {t('problems.showingCount', { shown: filteredProblems.length, total: data?.total || 0 })}
                            </div>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() => setPage(p => Math.max(1, p - 1))}
                                    disabled={page === 1}
                                    className="px-3 py-1.5 rounded-lg bg-card border disabled:opacity-50 hover:bg-muted transition-colors"
                                >
                                    {t('common.previous')}
                                </button>
                                <div className="px-3 py-1.5 rounded-lg bg-primary text-primary-foreground font-bold">
                                    {page}
                                </div>
                                <button
                                    onClick={() => setPage(p => p + 1)}
                                    disabled={filteredProblems.length < 50}
                                    className="px-3 py-1.5 rounded-lg bg-card border disabled:opacity-50 hover:bg-muted transition-colors"
                                >
                                    {t('common.next')}
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}


// Clone Problem Modal Component
function CloneProblemModal({ open, onOpenChange, problemCode, problemName, onSuccess }: {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    problemCode: string;
    problemName: string;
    onSuccess: () => void;
}) {
    const t = useTranslations('Admin');
    const [newCode, setNewCode] = useState('');
    const [newName, setNewName] = useState('');
    const [copyData, setCopyData] = useState(false);
    const [copyAuthors, setCopyAuthors] = useState(true);
    const [copySettings, setCopySettings] = useState(true);

    const cloneMutation = useMutation({
        mutationFn: () => adminProblemApi.clone(problemCode, {
            new_code: newCode,
            new_name: newName,
            copy_data: copyData,
            copy_authors: copyAuthors,
            copy_settings: copySettings,
        }),
        onSuccess: (data) => {
            onSuccess();
            onOpenChange(false);
            setNewCode('');
            setNewName('');
            alert(t('problems.cloneSuccess', { name: data.data.new_problem.name, code: data.data.new_problem.code }));
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || t('problems.cloneError'));
        },
    });

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        cloneMutation.mutate();
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4" />
            <div className="fixed inset-0 z-50 flex items-center justify-center p-4 pointer-events-none">
                <div className="bg-card rounded-2xl border shadow-2xl w-full max-w-md pointer-events-auto">
                    <form onSubmit={handleSubmit} className="space-y-6 p-6">
                        <div>
                            <h2 className="text-xl font-bold">{t('problems.cloneModalTitle')}</h2>
                            <p className="text-sm text-muted-foreground mt-1">
                                {t('problems.cloneModalSubtitle', { name: problemName })}
                            </p>
                        </div>

                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium mb-2">{t('problems.newCodeLabel')}</label>
                                <input
                                    type="text"
                                    value={newCode}
                                    onChange={(e) => setNewCode(e.target.value)}
                                    placeholder={t('problems.newCodePlaceholder')}
                                    className="w-full px-4 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                    required
                                />
                            </div>

                            <div>
                                <label className="block text-sm font-medium mb-2">{t('problems.newNameLabel')}</label>
                                <input
                                    type="text"
                                    value={newName}
                                    onChange={(e) => setNewName(e.target.value)}
                                    placeholder={t('problems.newNamePlaceholder')}
                                    className="w-full px-4 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                    required
                                />
                            </div>

                            <div className="space-y-2">
                                <label className="flex items-center gap-2 cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={copyAuthors}
                                        onChange={(e) => setCopyAuthors(e.target.checked)}
                                        className="rounded"
                                    />
                                    <span className="text-sm">{t('problems.copyAuthorsLabel')}</span>
                                </label>

                                <label className="flex items-center gap-2 cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={copySettings}
                                        onChange={(e) => setCopySettings(e.target.checked)}
                                        className="rounded"
                                    />
                                    <span className="text-sm">{t('problems.copySettingsLabel')}</span>
                                </label>

                                <label className="flex items-center gap-2 cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={copyData}
                                        onChange={(e) => setCopyData(e.target.checked)}
                                        className="rounded"
                                    />
                                    <span className="text-sm">{t('problems.copyDataLabel')}</span>
                                </label>
                            </div>
                        </div>

                        <div className="flex gap-3 pt-4">
                            <button
                                type="button"
                                onClick={() => onOpenChange(false)}
                                className="flex-1 px-4 py-2 rounded-lg bg-muted hover:bg-muted/80 transition-colors font-medium"
                            >
                                {t('common.cancel')}
                            </button>
                            <button
                                type="submit"
                                disabled={cloneMutation.isPending}
                                className="flex-1 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors font-medium disabled:opacity-50"
                            >
                                {cloneMutation.isPending ? t('common.cloning') : t('problems.cloneModalTitle')}
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </Dialog>
    );
}
