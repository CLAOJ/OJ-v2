'use client';

import { useState } from 'react';
import { Dialog } from '@/components/ui/Dialog';
import { Button } from '@/components/ui/Button';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { adminContestApi } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import { useListParams } from '@/hooks/useListParams';
import { useDebounce } from '@/hooks/useDebounce';
import { PaginationBar } from '@/components/ui/PaginationBar';
import { SortableHeader } from '@/components/ui/SortableHeader';
import { toggleSort, type ContestSort, type SortOrder } from '@/lib/listSort';
import {
    Search,
    Tag,
    Trophy,
    Ban,
    Edit,
    Trash2,
    RefreshCw,
    Play,
    Copy as CopyIcon
} from 'lucide-react';

const FILTER_KEYS = ['search', 'status', 'sort', 'order'] as const;
const PAGE_SIZE = 50;
const DEFAULT_SORT: ContestSort = 'start_time';

export default function AdminContestPage() {
    const t = useTranslations('Admin');
    const [cloneModalOpen, setCloneModalOpen] = useState(false);
    const [selectedContest, setSelectedContest] = useState<{ key: string; name: string } | null>(null);

    const queryClient = useQueryClient();
    const router = useRouter();

    const { page, filters, setFilters, setPage } = useListParams(FILTER_KEYS);
    const search = useDebounce(filters.search ?? '', 300);
    const sort = (filters.sort ?? DEFAULT_SORT) as ContestSort;
    const order: SortOrder = filters.order === 'asc' ? 'asc' : 'desc';

    const handleSort = (column: ContestSort, firstOrder: SortOrder) => {
        const next = toggleSort({ sort, order }, column, firstOrder);
        setFilters({ sort: next.sort, order: next.order });
    };

    const { data, isLoading } = useQuery({
        queryKey: ['admin-contests', page, search, filters.status, sort, order],
        queryFn: async () => (await adminContestApi.list(page, PAGE_SIZE, {
            search,
            status: filters.status as 'upcoming' | 'ongoing' | 'past' | undefined,
            sort,
            order,
        })).data,
    });

    const deleteMutation = useMutation({
        mutationFn: (key: string) => adminContestApi.delete(key),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-contests'] });
        }
    });

    const contests = data?.data || [];

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

                <div className="flex flex-wrap items-center gap-3">
                    <div className="relative w-full md:w-80">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                        <input
                            type="text"
                            placeholder={t('contests.searchPlaceholder')}
                            className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={filters.search ?? ''}
                            onChange={(e) => setFilters({ search: e.target.value })}
                        />
                    </div>
                    <select
                        aria-label={t('contests.statusFilter')}
                        className="h-10 px-3 rounded-xl bg-card border"
                        value={filters.status ?? ''}
                        onChange={(e) => setFilters({ status: e.target.value })}
                    >
                        <option value="">{t('contests.statusAll')}</option>
                        <option value="upcoming">{t('contests.statusUpcoming')}</option>
                        <option value="ongoing">{t('contests.statusOngoing')}</option>
                        <option value="past">{t('contests.statusPast')}</option>
                    </select>
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
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-20 rounded-md" />)}
                </div>
            ) : (
                <div className="bg-card rounded-md border overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="w-full text-left text-sm">
                            <thead className="bg-muted/50 border-b">
                                <tr>
                                    <SortableHeader
                                        label={t('contests.colContest')}
                                        active={sort === 'name'}
                                        order={order}
                                        onClick={() => handleSort('name', 'asc')}
                                    />
                                    <SortableHeader
                                        label={t('contests.colStart')}
                                        active={sort === 'start_time'}
                                        order={order}
                                        onClick={() => handleSort('start_time', 'desc')}
                                    />
                                    <SortableHeader
                                        label={t('contests.colEnd')}
                                        active={sort === 'end_time'}
                                        order={order}
                                        onClick={() => handleSort('end_time', 'desc')}
                                    />
                                    <SortableHeader
                                        label={t('contests.colUsers')}
                                        active={sort === 'user_count'}
                                        order={order}
                                        onClick={() => handleSort('user_count', 'desc')}
                                        className="text-right"
                                    />
                                    <th className="px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground">{t('contests.colSettings')}</th>
                                    <th className="px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground text-right">{t('common.actions')}</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {contests.length === 0 ? (
                                    <tr>
                                        <td colSpan={6} className="px-4 py-12 text-center text-muted-foreground">
                                            {t('contests.noContestsFound')}
                                        </td>
                                    </tr>
                                ) : (
                                    contests.map((contest) => (
                                        <tr key={contest.key} className="hover:bg-muted/30 transition-colors">
                                            <td className="px-4 py-2">
                                                <div className="flex items-center gap-2">
                                                    <span className="font-semibold text-sm">{contest.name}</span>
                                                    {contest.is_rated && (
                                                        <Badge variant="warning" className="text-[10px] py-0 px-1.5">{t('contests.rated')}</Badge>
                                                    )}
                                                    {contest.is_organization_private && (
                                                        <Badge variant="secondary" className="text-[10px] py-0 px-1.5">{t('contests.private')}</Badge>
                                                    )}
                                                </div>
                                                <div className="text-xs text-muted-foreground font-mono">{contest.key}</div>
                                            </td>
                                            <td className="px-4 py-2 text-xs text-muted-foreground whitespace-nowrap">
                                                {contest.start_time && new Date(contest.start_time).toLocaleDateString()}
                                            </td>
                                            <td className="px-4 py-2 text-xs text-muted-foreground whitespace-nowrap">
                                                {contest.end_time && new Date(contest.end_time).toLocaleDateString()}
                                            </td>
                                            <td className="px-4 py-2 text-xs text-right whitespace-nowrap">{contest.user_count}</td>
                                            <td className="px-4 py-2">
                                                {contest.is_visible ? (
                                                    <Badge variant="success" className="text-[10px] py-0 px-1.5">{t('contests.visible')}</Badge>
                                                ) : (
                                                    <Badge variant="destructive" className="text-[10px] py-0 px-1.5">{t('contests.hidden')}</Badge>
                                                )}
                                            </td>
                                            <td className="px-4 py-2 text-right">
                                                <div className="flex justify-end gap-1">
                                                    <button
                                                        onClick={() => {
                                                            setSelectedContest({ key: contest.key, name: contest.name });
                                                            setCloneModalOpen(true);
                                                        }}
                                                        className="p-1.5 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('contests.cloneTitle')}
                                                    >
                                                        <CopyIcon size={16} />
                                                    </button>
                                                    <Link
                                                        href={`/admin/contests/${contest.key}/edit`}
                                                        className="p-1.5 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('contests.editTitle')}
                                                    >
                                                        <Edit size={16} />
                                                    </Link>
                                                    <button
                                                        onClick={() => deleteMutation.mutate(contest.key)}
                                                        disabled={deleteMutation.isPending}
                                                        className="p-1.5 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                                        title={t('contests.deleteTitle')}
                                                    >
                                                        <Trash2 size={16} />
                                                    </button>
                                                </div>
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>

                </div>
            )}
            
            <PaginationBar page={page} onPageChange={setPage} total={data?.total} pageSize={PAGE_SIZE} />

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
            <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4" />
            <div className="fixed inset-0 z-50 flex items-center justify-center p-4 pointer-events-none">
                <div className="bg-card rounded-md border shadow-md w-full max-w-md pointer-events-auto">
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
