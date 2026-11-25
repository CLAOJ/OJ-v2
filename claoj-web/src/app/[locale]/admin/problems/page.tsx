'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { adminProblemApi, adminProblemGroupApi } from '@/lib/adminApi';
import { Badge } from '@/components/ui/Badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import { useListParams } from '@/hooks/useListParams';
import { useDebounce } from '@/hooks/useDebounce';
import { PaginationBar } from '@/components/ui/PaginationBar';
import { SortableHeader } from '@/components/ui/SortableHeader';
import { formatShortDate } from '@/lib/date';
import { toggleSort, yearOptions, type ProblemSort, type SortOrder } from '@/lib/listSort';
import {
    Search,
    Code2,
    Edit,
    Trash2,
    Database,
    Plus,
    Copy
} from 'lucide-react';
import { CloneProblemModal } from '@/components/admin/CloneProblemModal';

const FILTER_KEYS = ['search', 'is_public', 'group_id', 'creator', 'year', 'sort', 'order'] as const;
const PAGE_SIZE = 50;
const DEFAULT_SORT: ProblemSort = 'date';

export default function AdminProblemPage() {
    const t = useTranslations('Admin');
    const [cloneModalOpen, setCloneModalOpen] = useState(false);
    const [problemToClone, setProblemToClone] = useState<{code: string; name: string} | null>(null);

    const queryClient = useQueryClient();

    const { page, filters, setFilters, setPage } = useListParams(FILTER_KEYS);
    const search = useDebounce(filters.search ?? '', 300);
    const creator = useDebounce(filters.creator ?? '', 300);
    const year = filters.year ? Number(filters.year) : undefined;
    const sort = (filters.sort ?? DEFAULT_SORT) as ProblemSort;
    const order: SortOrder = filters.order === 'asc' ? 'asc' : 'desc';

    const handleSort = (column: ProblemSort, firstOrder: SortOrder) => {
        const next = toggleSort({ sort, order }, column, firstOrder);
        setFilters({ sort: next.sort, order: next.order });
    };

    const { data, isLoading } = useQuery({
        queryKey: ['admin-problems', page, search, creator, filters.is_public, filters.group_id, year, sort, order],
        queryFn: async () => (await adminProblemApi.list(page, PAGE_SIZE, {
            search,
            creator,
            year,
            is_public: filters.is_public as 'true' | 'false' | undefined,
            group_id: filters.group_id ? Number(filters.group_id) : undefined,
            sort,
            order,
        })).data,
    });

    const { data: groups } = useQuery({
        queryKey: ['problem-groups'],
        queryFn: async () => (await adminProblemGroupApi.list(1, 500)).data,
    });

    const deleteMutation = useMutation({
        mutationFn: (code: string) => adminProblemApi.delete(code),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-problems'] });
        }
    });

    const problems = data?.data || [];

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

                <div className="flex flex-wrap items-center gap-3">
                    <div className="relative w-full md:w-80">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                        <input
                            type="text"
                            placeholder={t('problems.searchPlaceholder')}
                            className="w-full h-10 pl-10 pr-4 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                            value={filters.search ?? ''}
                            onChange={(e) => setFilters({ search: e.target.value })}
                        />
                    </div>
                    <select
                        aria-label={t('problems.visibilityFilter')}
                        className="h-10 px-3 rounded-xl bg-card border"
                        value={filters.is_public ?? ''}
                        onChange={(e) => setFilters({ is_public: e.target.value })}
                    >
                        <option value="">{t('problems.visibilityAll')}</option>
                        <option value="true">{t('problems.publicBadge')}</option>
                        <option value="false">{t('problems.privateBadge')}</option>
                    </select>
                    <select
                        aria-label={t('problems.groupFilter')}
                        className="h-10 px-3 rounded-xl bg-card border"
                        value={filters.group_id ?? ''}
                        onChange={(e) => setFilters({ group_id: e.target.value })}
                    >
                        <option value="">{t('problems.groupAll')}</option>
                        {groups?.data.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
                    </select>
                    <input
                        type="text"
                        aria-label={t('problems.creatorFilter')}
                        placeholder={t('problems.creatorPlaceholder')}
                        className="w-full md:w-44 h-10 px-3 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={filters.creator ?? ''}
                        onChange={(e) => setFilters({ creator: e.target.value })}
                    />
                    <select
                        aria-label={t('problems.yearFilter')}
                        className="h-10 px-3 rounded-xl bg-card border"
                        value={filters.year ?? ''}
                        onChange={(e) => setFilters({ year: e.target.value })}
                    >
                        <option value="">{t('problems.yearAll')}</option>
                        {yearOptions(new Date().getFullYear()).map(y => <option key={y} value={y}>{y}</option>)}
                    </select>
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
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-20 rounded-md" />)}
                </div>
            ) : (
                <div className="bg-card rounded-md border overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="w-full text-left text-sm">
                            <thead className="bg-muted/50 border-b">
                                <tr>
                                    <SortableHeader
                                        label={t('problems.colProblem')}
                                        active={sort === 'name'}
                                        order={order}
                                        onClick={() => handleSort('name', 'asc')}
                                    />
                                    <SortableHeader
                                        label={t('problems.colCode')}
                                        active={sort === 'code'}
                                        order={order}
                                        onClick={() => handleSort('code', 'asc')}
                                    />
                                    <SortableHeader
                                        label={t('problems.colPoints')}
                                        active={sort === 'points'}
                                        order={order}
                                        onClick={() => handleSort('points', 'desc')}
                                        className="text-right"
                                    />
                                    <SortableHeader
                                        label={t('problems.colDate')}
                                        active={sort === 'date'}
                                        order={order}
                                        onClick={() => handleSort('date', 'desc')}
                                    />
                                    <SortableHeader
                                        label={t('problems.colAcRate')}
                                        active={sort === 'ac_rate'}
                                        order={order}
                                        onClick={() => handleSort('ac_rate', 'desc')}
                                        className="text-right"
                                    />
                                    <SortableHeader
                                        label={t('problems.colUsers')}
                                        active={sort === 'user_count'}
                                        order={order}
                                        onClick={() => handleSort('user_count', 'desc')}
                                        className="text-right"
                                    />
                                    <th className="px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground">{t('problems.colAuthors')}</th>
                                    <th className="px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground">{t('problems.colGroup')}</th>
                                    <th className="px-4 py-2.5 text-xs font-bold uppercase text-muted-foreground text-right">{t('common.actions')}</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {problems.length === 0 ? (
                                    <tr>
                                        <td colSpan={9} className="px-4 py-12 text-center text-muted-foreground">
                                            {t('problems.noProblemsFound')}
                                        </td>
                                    </tr>
                                ) : (
                                    problems.map((problem) => (
                                        <tr key={problem.code} className="hover:bg-muted/30 transition-colors">
                                            <td className="px-4 py-2">
                                                <div className="flex items-center gap-2">
                                                    <Link
                                                        href={`/problem/${problem.code}`}
                                                        className="font-semibold text-sm hover:text-primary transition-colors"
                                                    >
                                                        {problem.name}
                                                    </Link>
                                                    {problem.partial && (
                                                        <Badge variant="secondary" className="text-[10px] py-0 px-1.5">{t('problems.partialBadge')}</Badge>
                                                    )}
                                                    {problem.is_public ? (
                                                        <Badge variant="success" className="text-[10px] py-0 px-1.5">{t('problems.publicBadge')}</Badge>
                                                    ) : (
                                                        <Badge variant="destructive" className="text-[10px] py-0 px-1.5">{t('problems.privateBadge')}</Badge>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-4 py-2 font-mono text-xs text-muted-foreground whitespace-nowrap">{problem.code}</td>
                                            <td className="px-4 py-2 text-sm text-right whitespace-nowrap">{problem.points.toFixed(1)}</td>
                                            <td className="px-4 py-2 text-xs text-muted-foreground whitespace-nowrap">{formatShortDate(problem.date)}</td>
                                            <td className="px-4 py-2 text-xs text-right whitespace-nowrap">{Math.round(problem.ac_rate * 100)}%</td>
                                            <td className="px-4 py-2 text-xs text-right whitespace-nowrap">{problem.user_count}</td>
                                            <td className="px-4 py-2 text-xs text-muted-foreground">
                                                <span
                                                    className="block max-w-[10rem] truncate"
                                                    title={problem.authors.map(a => a.username).join(', ')}
                                                >
                                                    {problem.authors.map(a => a.username).join(', ') || '—'}
                                                </span>
                                            </td>
                                            <td className="px-4 py-2 text-xs text-muted-foreground whitespace-nowrap">
                                                {problem.group_name || t('problems.uncategorized')} · {problem.is_manually_managed ? t('problems.manuallyManaged') : t('problems.autoManaged')}
                                            </td>
                                            <td className="px-4 py-2 text-right">
                                                <div className="flex justify-end gap-1">
                                                    <Link
                                                        href={`/admin/problems/${problem.code}/edit`}
                                                        className="p-1.5 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('problems.editTitle')}
                                                    >
                                                        <Edit size={16} />
                                                    </Link>
                                                    <Link
                                                        href={`/admin/problems/${problem.code}/data`}
                                                        className="p-1.5 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('problems.manageDataTitle')}
                                                    >
                                                        <Database size={16} />
                                                    </Link>
                                                    <button
                                                        onClick={() => {
                                                            setProblemToClone({ code: problem.code, name: problem.name });
                                                            setCloneModalOpen(true);
                                                        }}
                                                        className="p-1.5 hover:bg-primary/10 text-primary rounded-lg transition-colors"
                                                        title={t('problems.cloneTitle')}
                                                    >
                                                        <Copy size={16} />
                                                    </button>
                                                    <button
                                                        onClick={() => deleteMutation.mutate(problem.code)}
                                                        disabled={deleteMutation.isPending}
                                                        className="p-1.5 hover:bg-destructive/10 text-destructive rounded-lg transition-colors"
                                                        title={t('problems.deleteTitle')}
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

            {problemToClone && (
                <CloneProblemModal
                    open={cloneModalOpen}
                    onOpenChange={setCloneModalOpen}
                    problemCode={problemToClone.code}
                    problemName={problemToClone.name}
                    onSuccess={() => queryClient.invalidateQueries({ queryKey: ['admin-problems'] })}
                />
            )}

            <PaginationBar page={page} onPageChange={setPage} total={data?.total} pageSize={PAGE_SIZE} />
        </div>
    );
}
