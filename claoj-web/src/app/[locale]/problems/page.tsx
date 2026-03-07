'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Problem, PaginatedList } from '@/types';
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
    HelpCircle
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { motion, AnimatePresence } from 'framer-motion';

export default function ProblemListPage() {
    const t = useTranslations('Problems');
    const router = useRouter();
    const [search, setSearch] = useState('');
    const [status, setStatus] = useState<string>('all');
    const [pointsMin, setPointsMin] = useState<number>(0);
    const [pointsMax, setPointsMax] = useState<number>(50);
    const [sort, setSort] = useState<string>('code');
    const [order, setOrder] = useState<'asc' | 'desc'>('asc');
    const [page, setPage] = useState(1);

    const { data: problemData, isLoading } = useQuery({
        queryKey: ['problems', search, status, pointsMin, pointsMax, sort, order, page],
        queryFn: async () => {
            const params = new URLSearchParams({
                search,
                status: status !== 'all' ? status : '',
                points_min: pointsMin.toString(),
                points_max: pointsMax.toString(),
                sort,
                order,
                page: page.toString(),
            });
            const res = await api.get<PaginatedList<Problem>>(`/problems?${params.toString()}`);
            return res.data;
        }
    });

    const { data: hotProblems } = useQuery({
        queryKey: ['hot-problems'],
        queryFn: async () => {
            const res = await api.get<PaginatedList<Problem>>('/problems?sort=user_count&order=desc&limit=5');
            return res.data.data;
        }
    });

    const launchRandomProblem = async () => {
        try {
            const res = await api.get<{ code: string }>('/problems/random');
            router.push(`/problems/${res.data.code}`);
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
        <div className="flex flex-col lg:flex-row gap-10 min-h-[calc(100vh-12rem)] animate-in fade-in duration-500 mt-4">
            {/* Sidebar: Filters & Hot Problems */}
            <aside className="w-full lg:w-80 flex flex-col gap-8 shrink-0">
                {/* Advanced Filter Card */}
                <div className="p-8 rounded-[3rem] bg-card border shadow-sm space-y-8 sticky top-4">
                    <div className="flex items-center gap-3">
                        <SlidersHorizontal size={20} className="text-primary" />
                        <h3 className="text-xs font-black uppercase tracking-[0.2em] text-primary">Advanced Filters</h3>
                    </div>

                    <div className="space-y-6">
                        {/* Search */}
                        <div className="space-y-2">
                            <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Search</label>
                            <div className="relative">
                                <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/50" size={16} />
                                <input
                                    type="text"
                                    placeholder="Code or Name..."
                                    className="w-full h-12 bg-muted/30 border border-muted-foreground/10 rounded-2xl pl-12 pr-4 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:border-primary/30 transition-all outline-none"
                                    value={search}
                                    onChange={(e) => setSearch(e.target.value)}
                                />
                            </div>
                        </div>

                        {/* Status */}
                        <div className="space-y-2">
                            <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Solved Status</label>
                            <div className="grid grid-cols-3 gap-2">
                                {['all', 'unsolved', 'solved'].map((s) => (
                                    <button
                                        key={s}
                                        onClick={() => setStatus(s)}
                                        className={cn(
                                            "h-10 text-[10px] font-black uppercase tracking-widest rounded-xl border transition-all",
                                            status === s
                                                ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                                : "bg-muted/30 hover:bg-muted/50 border-transparent text-muted-foreground"
                                        )}
                                    >
                                        {s}
                                    </button>
                                ))}
                            </div>
                        </div>

                        {/* Point Range */}
                        <div className="space-y-4">
                            <div className="flex justify-between items-center ml-1">
                                <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">Points</label>
                                <span className="text-xs font-black text-primary">{pointsMin} — {pointsMax}</span>
                            </div>
                            <div className="space-y-6">
                                <input
                                    type="range"
                                    min="0"
                                    max="50"
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
                                setPointsMin(0);
                                setPointsMax(50);
                            }}
                            className="w-full h-12 rounded-2xl bg-muted/50 text-[10px] font-black uppercase tracking-widest hover:bg-muted transition-all border border-dashed flex items-center justify-center gap-2"
                        >
                            <RefreshCw size={14} /> Reset Filters
                        </button>
                    </div>

                    {/* Random Problem Action */}
                    <div className="pt-6 border-t">
                        <button
                            onClick={launchRandomProblem}
                            className="w-full h-14 rounded-3xl bg-zinc-900 text-zinc-100 font-black flex items-center justify-center gap-3 group relative overflow-hidden transition-all hover:scale-[1.02] active:scale-95"
                        >
                            <div className="absolute inset-0 bg-gradient-to-r from-amber-500/20 to-rose-500/20 opacity-0 group-hover:opacity-100 transition-opacity" />
                            <Gamepad2 size={20} className="relative z-10 text-amber-500" />
                            <span className="relative z-10">Surprise Me!</span>
                        </button>
                    </div>
                </div>

                {/* Hot Problems */}
                <div className="p-8 rounded-[3rem] bg-zinc-900 border border-zinc-800 shadow-xl space-y-8 overflow-hidden relative">
                    <div className="absolute top-0 right-0 p-12 opacity-5 pointer-events-none rotate-12">
                        <Flame size={120} className="text-rose-500" />
                    </div>

                    <h3 className="text-xs font-black uppercase tracking-[0.2em] text-rose-500 flex items-center gap-3 relative z-10">
                        <Flame size={16} />
                        Trending Now
                    </h3>

                    <div className="space-y-2 relative z-10">
                        {hotProblems?.map(hp => (
                            <Link
                                key={hp.code}
                                href={`/problems/${hp.code}`}
                                className="block p-4 rounded-3xl bg-zinc-800/50 border border-zinc-700/50 hover:bg-zinc-800 hover:border-rose-500/30 transition-all group"
                            >
                                <div className="flex flex-col gap-1">
                                    <span className="text-[10px] font-black text-rose-500/70 uppercase tracking-widest">{hp.code}</span>
                                    <span className="text-sm font-black text-zinc-100 group-hover:text-rose-400 transition-colors truncate">{hp.name}</span>
                                    <div className="flex items-center justify-between mt-1">
                                        <span className="text-[10px] font-bold text-zinc-500">{hp.user_count} Users Solved</span>
                                        <span className="text-[10px] font-black text-zinc-400">{Math.round(hp.ac_rate)}% AC</span>
                                    </div>
                                </div>
                            </Link>
                        ))}
                    </div>
                </div>
            </aside>

            {/* Main Content: Problems Table */}
            <div className="flex-grow flex flex-col gap-6 min-w-0">
                <div className="bg-card border rounded-[3rem] overflow-hidden shadow-sm">
                    <div className="overflow-x-auto selection:bg-primary/10">
                        <table className="w-full text-left border-collapse">
                            <thead>
                                <tr className="bg-muted/30 border-b">
                                    <th className="px-10 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground w-20 text-center">Status</th>
                                    <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground cursor-pointer group" onClick={() => toggleSort('code')}>
                                        <div className="flex items-center gap-2">
                                            Code {sort === 'code' && (order === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />)}
                                        </div>
                                    </th>
                                    <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground cursor-pointer group" onClick={() => toggleSort('name')}>
                                        <div className="flex items-center gap-2">
                                            Name {sort === 'name' && (order === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />)}
                                        </div>
                                    </th>
                                    <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center cursor-pointer group" onClick={() => toggleSort('points')}>
                                        <div className="flex items-center gap-2 justify-center">
                                            Points {sort === 'points' && (order === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />)}
                                        </div>
                                    </th>
                                    <th className="px-10 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center cursor-pointer group" onClick={() => toggleSort('ac_rate')}>
                                        <div className="flex items-center gap-2 justify-center">
                                            AC Rate {sort === 'ac_rate' && (order === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />)}
                                        </div>
                                    </th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-muted/50">
                                {isLoading ? (
                                    [1, 2, 3, 4, 5, 6, 7, 8].map(i => (
                                        <tr key={i}><td colSpan={5} className="px-10 py-6"><Skeleton className="h-10 w-full rounded-2xl" /></td></tr>
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
                                            <td className="px-6 py-8 font-black font-mono text-sm text-primary tracking-tighter">{p.code}</td>
                                            <td className="px-6 py-8">
                                                <Link
                                                    href={`/problems/${p.code}`}
                                                    className="font-black text-lg hover:text-primary transition-colors underline-offset-4 decoration-primary/20 hover:underline"
                                                >
                                                    {p.name}
                                                </Link>
                                                <div className="text-[10px] uppercase font-black text-muted-foreground mt-1 tracking-widest opacity-40">{p.group}</div>
                                            </td>
                                            <td className="px-6 py-8 text-center">
                                                <span className="px-3 py-1.5 rounded-xl bg-primary/5 text-primary text-sm font-black border border-primary/10">
                                                    {p.points}
                                                </span>
                                            </td>
                                            <td className="px-10 py-8 text-center">
                                                <div className="flex flex-col items-center gap-2">
                                                    <span className="text-xs font-black text-muted-foreground">{Math.round(p.ac_rate)}%</span>
                                                    <div className="w-20 h-1.5 bg-muted rounded-full overflow-hidden shadow-inner">
                                                        <div
                                                            className="h-full bg-emerald-500 shadow-[0_0_10px_rgba(16,185,129,0.5)]"
                                                            style={{ width: `${p.ac_rate}%` }}
                                                        />
                                                    </div>
                                                </div>
                                            </td>
                                        </tr>
                                    ))
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>

                {/* Simple Pagination */}
                <div className="flex justify-center gap-4 py-8">
                    <button
                        onClick={() => setPage(p => Math.max(1, p - 1))}
                        disabled={page === 1}
                        className="px-8 h-12 rounded-2xl bg-card border font-black text-xs uppercase tracking-widest transition-all hover:bg-muted disabled:opacity-30 disabled:pointer-events-none"
                    >
                        Previous
                    </button>
                    <div className="h-12 flex items-center px-6 rounded-2xl bg-primary text-primary-foreground font-black text-xs">
                        Page {page}
                    </div>
                    <button
                        onClick={() => setPage(p => p + 1)}
                        className="px-8 h-12 rounded-2xl bg-card border font-black text-xs uppercase tracking-widest transition-all hover:bg-muted"
                    >
                        Next
                    </button>
                </div>
            </div>
        </div>
    );
}
