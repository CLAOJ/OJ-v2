'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Problem, PaginatedList, ProblemTaxonomy } from '@/types';
import { useAuth } from '@/components/providers/AuthProvider';
import { hasPerm } from '@/lib/permissions';
import { buildProblemListParams } from '@/lib/problemFilters';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link, useRouter } from '@/navigation';
import { useState, useEffect } from 'react';
import {
    Search,
    ChevronUp,
    ChevronDown,
    Flame,
    CheckCircle2,
    Filter,
    ArrowRight,
    Trophy,
    Gamepad2,
    RefreshCw,
    SlidersHorizontal,
    HelpCircle,
    Lightbulb
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { motion, AnimatePresence } from 'framer-motion';
import { PaginationBar, PAGE_SIZE_OPTIONS } from '@/components/ui/PaginationBar';

const TRENDING_LIMIT = 5;
// Slider ceiling, and the points value at which the upper bound is not sent at
// all: problems are authored with at most 100 points, and a hard `points_max`
// must never hide anything above the slider range.
const POINTS_MAX = 100;

export default function ProblemsPageContent() {
    const t = useTranslations('Problems');
    const tCommon = useTranslations('Common');
    const router = useRouter();
    const { user } = useAuth();
    const [search, setSearch] = useState('');
    const [status, setStatus] = useState<string>('all');
    const [group, setGroup] = useState<string>('');
    const [problemTypes, setProblemTypes] = useState<number[]>([]);
    const [hasEditorial, setHasEditorial] = useState(false);
    const [pointsMin, setPointsMin] = useState<number>(0);
    const [pointsMax, setPointsMax] = useState<number>(POINTS_MAX);
    const [sort, setSort] = useState<string>('code');
    const [order, setOrder] = useState<'asc' | 'desc'>('asc');
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState<number>(PAGE_SIZE_OPTIONS[1]);

    const { data: problemData, isLoading } = useQuery({
        queryKey: ['problems', search, status, group, problemTypes, hasEditorial, pointsMin, pointsMax, sort, order, page, pageSize],
        queryFn: async () => {
            const params = buildProblemListParams({
                search,
                status: status !== 'all' ? status : '',
                group,
                types: problemTypes,
                hasEditorial,
                pointsMin,
                pointsMax: pointsMax >= POINTS_MAX ? null : pointsMax,
                sort,
                order,
                page,
                pageSize,
            });
            const res = await api.get<PaginatedList<Problem>>(`/problems?${params.toString()}`);
            return res.data;
        }
    });

    // Filter vocabulary: v1 derives both selects from the full group/type tables.
    const { data: groups } = useQuery({
        // Distinct from the admin lists' ['problem-groups'] cache entry: this
        // one holds the public taxonomy array, not the admin paged response.
        queryKey: ['public-problem-groups'],
        queryFn: async () => {
            const res = await api.get<PaginatedList<ProblemTaxonomy>>('/problem-groups');
            return res.data.data;
        }
    });
    const { data: types } = useQuery({
        queryKey: ['public-problem-types'],
        queryFn: async () => {
            const res = await api.get<PaginatedList<ProblemTaxonomy>>('/problem-types');
            return res.data.data;
        }
    });

    const { data: hotProblems } = useQuery({
        queryKey: ['hot-problems'],
        queryFn: async () => {
            const res = await api.get<PaginatedList<Problem>>(
                `/problems?sort=user_count&order=desc&page_size=${TRENDING_LIMIT}`
            );
            return res.data.data.slice(0, TRENDING_LIMIT);
        }
    });

    const launchRandomProblem = async () => {
        try {
            const res = await api.get<{ code: string }>('/problems/random');
            router.push(`/problem/${res.data.code}`);
        } catch (err) {
            // Failed to launch random problem - user will see error state
        }
    };

    const toggleSort = (field: string) => {
        if (sort === field) {
            setOrder(order === 'asc' ? 'desc' : 'asc');
        } else {
            setSort(field);
            setOrder('asc');
        }
    };

    return (
        <div className="flex flex-col lg:flex-row gap-4 min-h-[calc(100vh-12rem)] mt-4">
            {/* Sidebar: Filters & Hot Problems.
                Deliberately not sticky: the column is taller than a typical
                viewport, so pinning it either overlaps the cards or clips the
                last one behind an inner scrollbar. */}
            <aside className="w-full lg:w-80 flex flex-col gap-4 shrink-0">
                {/* Advanced Filter Card */}
                <div className="p-4 rounded-md bg-card border shadow-sm space-y-4">
                    <div className="flex items-center gap-3">
                        <SlidersHorizontal size={20} className="text-muted-foreground" />
                        <h3 className="text-xs font-semibold text-primary">{t('advancedFilters')}</h3>
                    </div>

                    <div className="space-y-6">
                        {/* Search */}
                        <div className="space-y-2">
                            <label className="text-xs font-medium text-muted-foreground ml-1">{t('search')}</label>
                            <div className="relative">
                                <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/50" size={16} />
                                <input
                                    type="text"
                                    placeholder={t('searchPlaceholder')}
                                    className="w-full h-12 bg-muted/30 border border-muted-foreground/10 rounded-md pl-12 pr-4 text-sm font-medium focus:ring-2 focus:ring-primary/20 focus:border-primary/30 transition-all outline-none"
                                    value={search}
                                    onChange={(e) => setSearch(e.target.value)}
                                />
                            </div>
                        </div>

                        {/* Status */}
                        <div className="space-y-2">
                            <label className="text-xs font-medium text-muted-foreground ml-1">{t('solvedStatus')}</label>
                            <div className="grid grid-cols-3 gap-2">
                                {['all', 'unsolved', 'solved'].map((s) => (
                                    <button
                                        key={s}
                                        onClick={() => setStatus(s)}
                                        className={cn(
                                            "h-10 text-xs font-medium rounded-xl border transition-all",
                                            status === s
                                                ? "bg-primary text-primary-foreground border-primary"
                                                : "bg-muted/30 hover:bg-muted/50 border-transparent text-muted-foreground"
                                        )}
                                    >
                                        {s === 'all' ? t('statusAll') : s === 'solved' ? t('solved') : t('unsolved')}
                                    </button>
                                ))}
                            </div>
                        </div>

                        {/* Group */}
                        <div className="space-y-2">
                            <label className="text-xs font-medium text-muted-foreground ml-1">{t('problemGroup')}</label>
                            <select
                                className="w-full h-12 bg-muted/30 border border-transparent rounded-md px-4 text-sm font-medium focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none cursor-pointer"
                                value={group}
                                onChange={(e) => { setGroup(e.target.value); setPage(1); }}
                            >
                                <option value="">{t('allGroups')}</option>
                                {groups?.map(g => <option key={g.id} value={g.id}>{g.full_name}</option>)}
                            </select>
                        </div>

                        {/* Types */}
                        {types && types.length > 0 && (
                            <div className="space-y-2">
                                <label className="text-xs font-medium text-muted-foreground ml-1">{t('problemTypes')}</label>
                                <div className="flex flex-wrap gap-2">
                                    {types.map(pt => {
                                        const active = problemTypes.includes(pt.id);
                                        return (
                                            <button
                                                key={pt.id}
                                                onClick={() => {
                                                    setProblemTypes(active
                                                        ? problemTypes.filter(id => id !== pt.id)
                                                        : [...problemTypes, pt.id]);
                                                    setPage(1);
                                                }}
                                                className={cn(
                                                    "px-3 h-8 text-xs font-medium rounded-xl border transition-all",
                                                    active
                                                        ? "bg-primary text-primary-foreground border-primary"
                                                        : "bg-muted/30 hover:bg-muted/50 border-transparent text-muted-foreground"
                                                )}
                                            >
                                                {pt.full_name}
                                            </button>
                                        );
                                    })}
                                </div>
                            </div>
                        )}

                        {/* Editorial */}
                        <label className="flex items-center gap-3 ml-1 cursor-pointer">
                            <input
                                type="checkbox"
                                className="h-4 w-4 accent-primary cursor-pointer"
                                checked={hasEditorial}
                                onChange={(e) => { setHasEditorial(e.target.checked); setPage(1); }}
                            />
                            <span className="text-xs font-medium text-muted-foreground">{t('hasEditorial')}</span>
                        </label>

                        {/* Point Range */}
                        <div className="space-y-4">
                            <div className="flex justify-between items-center ml-1">
                                <label className="text-xs font-medium text-muted-foreground">{t('points')}</label>
                                <span className="text-xs font-medium text-primary">{pointsMin} — {pointsMax}</span>
                            </div>
                            <div className="space-y-6">
                                <input
                                    type="range"
                                    min="0"
                                    max={POINTS_MAX}
                                    step="1"
                                    value={pointsMax}
                                    onChange={(e) => setPointsMax(parseInt(e.target.value))}
                                    className="w-full accent-primary h-1.5 bg-muted rounded-full cursor-pointer"
                                />
                            </div>
                        </div>

                        <button
                            onClick={() => {
                                setSearch('');
                                setStatus('all');
                                setGroup('');
                                setProblemTypes([]);
                                setHasEditorial(false);
                                setPointsMin(0);
                                setPointsMax(POINTS_MAX);
                            }}
                            className="w-full h-9 rounded-md bg-muted/50 text-xs font-medium hover:bg-muted transition-all border border-dashed flex items-center justify-center gap-2"
                        >
                            <RefreshCw size={14} /> {t('resetFilters')}
                        </button>
                    </div>

                    {/* Random Problem Action */}
                    <div className="pt-6 border-t space-y-2">
                        <button
                            onClick={launchRandomProblem}
                            className="w-full h-10 rounded-md bg-foreground text-background font-medium flex items-center justify-center gap-2"
                        >
                            <Gamepad2 size={20} />
                            <span>{t('surpriseMe')}</span>
                        </button>
                        {/* v1 shows a "Suggest" tab in the problem list header
                            for holders of judge.suggest_new_problem. */}
                        {hasPerm(user, 'judge.suggest_new_problem') && (
                            <Link
                                href="/problems/suggest"
                                className="w-full h-10 rounded-md bg-muted/50 hover:bg-muted border border-dashed text-xs font-medium flex items-center justify-center gap-2 transition-all"
                            >
                                <Lightbulb size={16} />
                                <span>{t('suggestions')}</span>
                            </Link>
                        )}
                    </div>
                </div>

                {/* Hot Problems */}
                <div className="p-4 rounded-md bg-card border border-border space-y-4 overflow-hidden relative">
                    <div className="absolute top-0 right-0 p-12 opacity-5 pointer-events-none rotate-12">
                        <Flame size={120} className="text-muted-foreground" />
                    </div>

                    <h3 className="text-xs font-semibold text-muted-foreground flex items-center gap-3 relative z-10">
                        <Flame size={16} />
                        {t('trendingNow')}
                    </h3>

                    <div className="space-y-2 relative z-10">
                        {hotProblems?.map(hp => (
                            <Link
                                key={hp.code}
                                href={`/problem/${hp.code}`}
                                className="block p-4 rounded-md bg-muted/50 border border-border hover:bg-muted hover:border-rose-500/30 transition-all group"
                            >
                                <div className="flex flex-col gap-1">
                                    <span className="text-xs font-medium text-rose-500/70 ">{hp.code}</span>
                                    <span className="text-sm font-medium text-foreground group-hover:text-rose-400 transition-colors truncate">{hp.name}</span>
                                    <div className="flex items-center justify-between mt-1">
                                        <span className="text-xs font-medium text-muted-foreground">{t('usersSolved', { count: hp.user_count })}</span>
                                        <span className="text-xs font-medium text-muted-foreground">{t('acPercent', { rate: Math.round(hp.ac_rate) })}</span>
                                    </div>
                                </div>
                            </Link>
                        ))}
                    </div>
                </div>
            </aside>

            {/* Main Content: Problems Table */}
            <div className="flex-grow flex flex-col gap-6 min-w-0">
                <div className="bg-card border rounded-md overflow-hidden shadow-sm">
                    <div className="overflow-x-auto selection:bg-primary/10">
                        <table className="w-full text-left border-collapse">
                            <thead>
                                <tr className="bg-muted/30 border-b">
                                    <th className="px-10 py-6 text-xs font-medium text-muted-foreground w-20 text-center">{tCommon('status')}</th>
                                    <th className="px-6 py-6 text-xs font-medium text-muted-foreground cursor-pointer group" onClick={() => toggleSort('code')}>
                                        <div className="flex items-center gap-2">
                                            {t('code')} {sort === 'code' && (order === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />)}
                                        </div>
                                    </th>
                                    <th className="px-6 py-6 text-xs font-medium text-muted-foreground cursor-pointer group" onClick={() => toggleSort('name')}>
                                        <div className="flex items-center gap-2">
                                            {t('name')} {sort === 'name' && (order === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />)}
                                        </div>
                                    </th>
                                    <th className="px-6 py-6 text-xs font-medium text-muted-foreground text-center cursor-pointer group" onClick={() => toggleSort('points')}>
                                        <div className="flex items-center gap-2 justify-center">
                                            {t('points')} {sort === 'points' && (order === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />)}
                                        </div>
                                    </th>
                                    <th className="px-10 py-6 text-xs font-medium text-muted-foreground text-center cursor-pointer group" onClick={() => toggleSort('ac_rate')}>
                                        <div className="flex items-center gap-2 justify-center">
                                            {t('acRate')} {sort === 'ac_rate' && (order === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />)}
                                        </div>
                                    </th>
                                    <th className="px-10 py-6 text-xs font-medium text-muted-foreground text-center cursor-pointer group" onClick={() => toggleSort('user_count')}>
                                        <div className="flex items-center gap-2 justify-center">
                                            {t('acCount')} {sort === 'user_count' && (order === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />)}
                                        </div>
                                    </th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-muted/50">
                                {isLoading ? (
                                    Array.from({ length: Math.min(pageSize, 10) }).map((_, i) => (
                                        <tr key={i}><td colSpan={6} className="px-10 py-6"><Skeleton className="h-10 w-full rounded-md" /></td></tr>
                                    ))
                                ) : (
                                    problemData?.data.map(p => (
                                        <tr key={p.code} className="group hover:bg-muted/5 transition-colors">
                                            <td className="px-10 py-8 text-center">
                                                {p.is_solved ? (
                                                    <div className="inline-flex items-center justify-center p-2 rounded-full bg-emerald-500/10 text-emerald-500 shadow-sm border border-emerald-500/20">
                                                        <CheckCircle2 size={24} />
                                                    </div>
                                                ) : (
                                                    <div className="inline-flex items-center justify-center p-2 rounded-full bg-muted/30 text-muted-foreground/10 border border-muted-foreground/5">
                                                        <HelpCircle size={24} />
                                                    </div>
                                                )}
                                            </td>
                                            <td className="px-6 py-8 font-medium font-mono text-sm text-primary tracking-tighter">{p.code}</td>
                                            <td className="px-6 py-8">
                                                <Link
                                                    href={`/problem/${p.code}`}
                                                    className="font-semibold text-lg hover:text-primary transition-colors underline-offset-4 decoration-primary/20 hover:underline"
                                                >
                                                    {p.name}
                                                </Link>
                                                <div className="text-xs font-medium text-muted-foreground mt-1 opacity-40">{p.group}</div>
                                            </td>
                                            <td className="px-6 py-8 text-center">
                                                <span className="px-3 py-1.5 rounded-xl bg-primary/5 text-primary text-sm font-medium border border-primary/10">
                                                    {p.points}
                                                </span>
                                            </td>
                                            <td className="px-10 py-8 text-center">
                                                <div className="flex flex-col items-center gap-2">
                                                    <span className="text-xs font-medium text-muted-foreground">{Math.round(p.ac_rate)}%</span>
                                                    <div className="w-20 h-1.5 bg-muted rounded-full overflow-hidden shadow-inner">
                                                        <div
                                                            className="h-full bg-emerald-500"
                                                            style={{ width: `${p.ac_rate}%` }}
                                                        />
                                                    </div>
                                                </div>
                                            </td>
                                            <td className="px-10 py-8 text-center text-sm font-medium text-muted-foreground">{p.user_count}</td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>

                <PaginationBar
                    page={page}
                    onPageChange={setPage}
                    total={problemData?.total}
                    pageSize={pageSize}
                    onPageSizeChange={size => { setPageSize(size); setPage(1); }}
                />

                <div className="pb-8" />
            </div>
        </div>
    );
}
