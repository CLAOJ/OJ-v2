'use client';

import { useState } from 'react';
import { Dialog } from '@/components/ui/Dialog';
import { Button } from '@/components/ui/Button';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { AdminContest } from '@/types';
import { adminContestApi } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import {
    Search,
    Tag,
    Trophy,
    Globe,
    Ban,
    Edit,
    Trash2,
    Calendar,
    Clock,
    RefreshCw,
    Play,
    Copy as CopyIcon
} from 'lucide-react';

export default function AdminContestPage() {
    const t = useTranslations('Admin');
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState('');
    const [cloneModalOpen, setCloneModalOpen] = useState(false);
    const [selectedContest, setSelectedContest] = useState<{ key: string; name: string } | null>(null);

    const queryClient = useQueryClient();
    const router = useRouter();

    const { data, isLoading, refetch } = useQuery({
        queryKey: ['admin-contests', page, search],
        queryFn: async () => {
            const res = await api.get<{
                data: AdminContest[];
                total: number;
                page: number;
                page_size: number;
            }>(`/admin/contests?page=${page}&page_size=50&search=${search}`);
            return res.data;
        }
    });

    const deleteMutation = useMutation({
        mutationFn: (key: string) => adminContestApi.delete(key),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-contests'] });
        }
    });

    const contests = data?.data || [];

    const filteredContests = contests.filter(c =>
        c.name.toLowerCase().includes(search.toLowerCase()) ||
        c.key.toLowerCase().includes(search.toLowerCase())
    );

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Trophy className="text-primary" size={32} />
                        {t('contests.title')}
                    </h1>
                    <p className="text-muted-foreground mt-1">{t('contests.subtitle')}</p>
                </div>

                <div className="flex items-center gap-3">
                    <div className="relative w-full md:w-80">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                        <input
                            type="text"
                            placeholder={t('contests.searchPlaceholder')}
                            className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                        />
                    </div>
                    <div className="flex gap-2">
                        <Link
                            href="/admin/contests/tags"
                            className="px-4 py-2 rounded-xl bg-muted hover:bg-muted/80 transition-colors flex items-center gap-2 font-medium"
                        >
                            <Tag size={18} />
                            {t('contests.tagsLink')}
                        </Link>
                        <Link
                            href="/admin/contests/create"
                            className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2 font-medium"
                        >
                            <Play size={18} />
                            {t('common.create')}
                        </Link>
                    </div>
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
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('contests.colContest')}</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('contests.colTime')}</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground">{t('contests.colSettings')}</th>
                                    <th className="px-6 py-4 text-xs font-bold uppercase text-muted-foreground text-right">{t('common.actions')}</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {filteredContests.length === 0 ? (
                                    <tr>
                                        <td colSpan={4} className="px-6 py-12 text-center text-muted-foreground">
                                            {t('contests.noContestsFound')}
                                        </td>
                                    </tr>
                                ) : (
                                    filteredContests.map((contest) => (
                                        <tr key={contest.key} className="hover:bg-muted/30 transition-colors">
                                            <td className="px-6 py-4">
                                                <div className="font-bold text-sm mb-1">{contest.name}</div>
                                                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                                    <span className="font-mono">{contest.key}</span>
                                                    {contest.is_rated && (
                                                        <Badge variant="warning" className="text-[10px]">{t('contests.rated')}</Badge>
                                                    )}
                                                    {contest.is_organization_private && (
                                                        <Badge variant="secondary" className="text-[10px]">{t('contests.private')}</Badge>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex flex-col gap-1 text-sm">
                                                    <span className="text-muted-foreground">
                                                        {contest.start_time && new Date(contest.start_time).toLocaleDateString()}
                                                    </span>
                                                    <span className="text-xs text-muted-foreground flex items-center gap-1">
                                                        <Clock size={12} />
                                                        {contest.end_time && new Date(contest.end_time).toLocaleDateString()}
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-2">
                                                    {contest.is_visible ? (
                                                        <Badge variant="success" className="flex items-center gap-1 text-xs">
                                                            {t('contests.visible')}
                                                        </Badge>
                                                    ) : (
                                                        <Badge variant="destructive" className="flex items-center text-xs">
                                                            {t('contests.hidden')}
                                                        </Badge>
                                                    )}
                                                    <span className="text-xs text-muted-foreground">
                                                        {t('contests.userCount', { count: contest.user_count })}
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <div className="flex justify-end gap-2">
                                                    <button
                                                        onClick={() => {
                                                            setSelectedContest({ key: contest.key, name: contest.name });
                                                            setCloneModalOpen(true);
                                                        }}
                                                        className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('contests.cloneTitle')}
                                                    >
                                                        <CopyIcon size={18} />
                                                    </button>
                                                    <Link
                                                        href={`/admin/contests/${contest.key}/edit`}
                                                        className="p-2 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('contests.editTitle')}
                                                    >
                                                        <Edit size={18} />
                                                    </Link>
                                                    <button
                                                        onClick={() => deleteMutation.mutate(contest.key)}
                                                        disabled={deleteMutation.isPending}
                                                        className="p-2 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                                        title={t('contests.deleteTitle')}
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
                    {filteredContests.length > 0 && (
                        <div className="flex items-center justify-between px-6 py-4 border-t bg-muted/30">
                            <div className="text-sm text-muted-foreground">
                                {t('contests.showingCount', { shown: filteredContests.length, total: data?.total || 0 })}
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
                                    disabled={filteredContests.length < 50}
                                    className="px-3 py-1.5 rounded-lg bg-card border disabled:opacity-50 hover:bg-muted transition-colors"
                                >
                                    {t('common.next')}
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            )}
            
            {/* Clone Modal */}
            {selectedContest && (
                <CloneContestModal
                    open={cloneModalOpen}
                    onOpenChange={setCloneModalOpen}
                    contestKey={selectedContest.key}
                    contestName={selectedContest.name}
                    onSuccess={() => {
                        queryClient.invalidateQueries({ queryKey: ['admin-contests'] });
                    }}
                />
            )}
        </div>
    );
}


interface CloneContestModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    contestKey: string;
    contestName: string;
    onSuccess: () => void;
}

function CloneContestModal({ open, onOpenChange, contestKey, contestName, onSuccess }: CloneContestModalProps) {
    const t = useTranslations('Admin');
    const [newKey, setNewKey] = useState('');
    const [newName, setNewName] = useState('');
    const [copyProblems, setCopyProblems] = useState(true);
    const [copySettings, setCopySettings] = useState(true);

    const cloneMutation = useMutation({
        mutationFn: () => adminContestApi.clone(contestKey, {
            new_key: newKey,
            new_name: newName,
            copy_problems: copyProblems,
            copy_settings: copySettings,
        }),
        onSuccess: (data) => {
            onSuccess();
            onOpenChange(false);
            setNewKey('');
            setNewName('');
            alert(t('contests.cloneSuccess', { name: data.data.new_contest.name, key: data.data.new_contest.key }));
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || t('contests.cloneError'));
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
                            <h2 className="text-xl font-bold">{t('contests.cloneModalTitle')}</h2>
                            <p className="text-sm text-muted-foreground mt-1">
                                {t('contests.cloneModalSubtitle', { name: contestName })}
                            </p>
                        </div>

                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium mb-2">{t('contests.newKeyLabel')}</label>
                                <input
                                    type="text"
                                    value={newKey}
                                    onChange={(e) => setNewKey(e.target.value)}
                                    placeholder={t('contests.newKeyPlaceholder')}
                                    className="w-full px-4 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                    required
                                />
                            </div>

                            <div>
                                <label className="block text-sm font-medium mb-2">{t('contests.newNameLabel')}</label>
                                <input
                                    type="text"
                                    value={newName}
                                    onChange={(e) => setNewName(e.target.value)}
                                    placeholder={t('contests.newNamePlaceholder')}
                                    className="w-full px-4 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                                    required
                                />
                            </div>

                            <div className="space-y-2">
                                <label className="flex items-center gap-3 p-3 rounded-lg border cursor-pointer hover:bg-muted/30">
                                    <input
                                        type="checkbox"
                                        checked={copyProblems}
                                        onChange={(e) => setCopyProblems(e.target.checked)}
                                        className="w-4 h-4"
                                    />
                                    <div>
                                        <div className="text-sm font-medium">{t('contests.copyProblemsLabel')}</div>
                                        <div className="text-xs text-muted-foreground">{t('contests.copyProblemsDesc')}</div>
                                    </div>
                                </label>

                                <label className="flex items-center gap-3 p-3 rounded-lg border cursor-pointer hover:bg-muted/30">
                                    <input
                                        type="checkbox"
                                        checked={copySettings}
                                        onChange={(e) => setCopySettings(e.target.checked)}
                                        className="w-4 h-4"
                                    />
                                    <div>
                                        <div className="text-sm font-medium">{t('contests.copySettingsLabel')}</div>
                                        <div className="text-xs text-muted-foreground">{t('contests.copySettingsDesc')}</div>
                                    </div>
                                </label>
                            </div>
                        </div>

                        <div className="flex items-center gap-3 pt-4 border-t">
                            <Button
                                type="button"
                                variant="secondary"
                                onClick={() => onOpenChange(false)}
                                className="flex-1"
                            >
                                {t('common.cancel')}
                            </Button>
                            <Button
                                type="submit"
                                variant="default"
                                disabled={cloneMutation.isPending}
                                className="flex-1"
                            >
                                {cloneMutation.isPending ? t('common.cloning') : t('contests.cloneModalTitle')}
                            </Button>
                        </div>
                    </form>
                </div>
            </div>
        </Dialog>
    );
}
