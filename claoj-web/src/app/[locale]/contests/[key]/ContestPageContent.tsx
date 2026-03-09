'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { ContestDetail, RankingResponse } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { useState, useEffect, use } from 'react';
import {
    Trophy,
    Clock,
    LayoutDashboard,
    ListOrdered,
    BarChart3,
    Lock,
    Play,
    CheckCircle2,
    HelpCircle,
    Info,
    Activity,
    Wifi
} from 'lucide-react';
import { cn, getRankColor, getRankBadgeColor, getRatingChangeColor, formatRatingChange } from '@/lib/utils';
import dayjs from 'dayjs';
import duration from 'dayjs/plugin/duration';
import relativeTime from 'dayjs/plugin/relativeTime';
import { motion, AnimatePresence } from 'framer-motion';
import { Link } from '@/navigation';
import Comments from '@/components/common/Comments';
import { useWebSocketContext } from '@/contexts/WebSocketContext';

dayjs.extend(duration);
dayjs.extend(relativeTime);

export default function ContestPageContent({ params }: { params: Promise<{ key: string }> }) {
    const { key } = use(params);
    const t = useTranslations('Contest');
    const [activeTab, setActiveTab] = useState<'dashboard' | 'problems' | 'scoreboard'>('dashboard');
    const [timeLeft, setTimeLeft] = useState<string>('');
    const queryClient = useQueryClient();
    const { subscribe, unsubscribe, status } = useWebSocketContext();
    const [lastLiveUpdate, setLastLiveUpdate] = useState<Date | null>(null);

    const { data: contest, isLoading: isFetching } = useQuery({
        queryKey: ['contest', key],
        queryFn: async () => {
            const res = await api.get<ContestDetail>(`/contest/${key}`);
            return res.data;
        }
    });

    const { data: ranking, isLoading: isRankingFetching } = useQuery({
        queryKey: ['contest-ranking', key],
        queryFn: async () => {
            const res = await api.get<RankingResponse>(`/contest/${key}/ranking`);
            return res.data;
        },
        enabled: activeTab === 'scoreboard',
        refetchInterval: (query) => {
            if (activeTab !== 'scoreboard') return false;
            const contestData = queryClient.getQueryData<ContestDetail>(['contest', key]);
            if (!contestData) return false;
            const now = dayjs();
            const isRunning = now.isAfter(contestData.start_time) && now.isBefore(contestData.end_time);
            return isRunning ? 10000 : false;
        }
    });

    const { mutate: joinContest, isPending: isJoining } = useMutation({
        mutationFn: async () => {
            await api.post(`/contest/${key}/join`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['contest', key] });
        }
    });

    useEffect(() => {
        if (!contest) return;

        const interval = setInterval(() => {
            const now = dayjs();
            const start = dayjs(contest.start_time);
            const end = dayjs(contest.end_time);

            if (now.isBefore(start)) {
                const diff = dayjs.duration(start.diff(now));
                setTimeLeft(`${t('startIn')} ${Math.floor(diff.asHours())}h ${diff.minutes()}m ${diff.seconds()}s`);
            } else if (now.isBefore(end)) {
                const diff = dayjs.duration(end.diff(now));
                setTimeLeft(`${t('endsIn')} ${Math.floor(diff.asHours())}h ${diff.minutes()}m ${diff.seconds()}s`);
            } else {
                setTimeLeft(t('ended'));
                clearInterval(interval);
            }
        }, 1000);

        return () => clearInterval(interval);
    }, [contest, t]);

    useEffect(() => {
        if (!key) return;

        const contestChannel = `contest_${key}`;
        subscribe(contestChannel);

        return () => {
            unsubscribe(contestChannel);
        };
    }, [key, subscribe, unsubscribe]);

    useEffect(() => {
        if (activeTab === 'scoreboard' && status === 'connected') {
            setLastLiveUpdate(new Date());
        }
    }, [ranking, status, activeTab]);

    if (isFetching) return <div className="p-8 max-w-7xl mx-auto"><Skeleton className="h-[60vh] w-full rounded-[3rem]" /></div>;
    if (!contest) return <div className="p-8 text-center text-muted-foreground">Contest not found.</div>;

    const isRunning = dayjs().isAfter(contest.start_time) && dayjs().isBefore(contest.end_time);
    const isPast = dayjs().isAfter(contest.end_time);

    return (
        <div className="max-w-7xl mx-auto space-y-8 pb-20 animate-in fade-in duration-700 mt-4">
            {/* Contest Header */}
            <div className="relative overflow-hidden rounded-[3rem] border bg-card shadow-2xl shadow-primary/5">
                <div className="absolute top-0 right-0 p-16 opacity-5 pointer-events-none rotate-12">
                    <Trophy size={240} className="text-primary" />
                </div>

                <div className="p-10 md:p-14 md:pb-8 space-y-8 relative">
                    <div className="space-y-4">
                        <div className="flex flex-wrap items-center gap-4">
                            <Badge variant="outline" className="px-4 py-1.5 rounded-full border-primary/20 bg-primary/5 text-primary text-[10px] font-black uppercase tracking-widest shadow-sm">
                                {contest.format} Format
                            </Badge>
                            <div className="flex items-center gap-2 text-xs font-black text-muted-foreground bg-muted/30 px-4 py-1.5 rounded-full border border-dashed tracking-wide">
                                <Clock size={14} className="text-primary" />
                                {timeLeft}
                            </div>
                        </div>
                        <h1 className="text-4xl md:text-6xl font-black tracking-tighter leading-none">
                            {contest.name}
                        </h1>
                    </div>

                    <div className="flex flex-wrap items-center gap-4 pt-4">
                        {isRunning && (
                            <button
                                onClick={() => joinContest()}
                                disabled={isJoining || contest.is_joined}
                                className={cn(
                                    "px-10 h-14 rounded-2xl font-black transition-all shadow-xl flex items-center gap-3",
                                    contest.is_joined
                                        ? "bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 cursor-default"
                                        : "bg-primary text-primary-foreground hover:scale-[1.02] active:scale-95 shadow-primary/20"
                                )}
                            >
                                {contest.is_joined ? <CheckCircle2 size={20} /> : <Play size={20} fill="currentColor" />}
                                {contest.is_joined ? 'Participating' : t('joinContest')}
                            </button>
                        )}
                        <div className="flex items-center gap-2 px-6 h-14 rounded-2xl bg-card border font-black text-sm shadow-sm group">
                            <BarChart3 size={18} className="text-primary group-hover:scale-110 transition-transform" />
                            {contest.is_rated ? t('rated') : 'Unrated'}
                        </div>
                    </div>
                </div>

                {/* Navigation Tabs */}
                <div className="px-10 md:px-14 border-t bg-muted/10">
                    <div className="flex gap-10 overflow-x-auto no-scrollbar whitespace-nowrap">
                        {[
                            { id: 'dashboard', label: t('dashboard'), icon: LayoutDashboard, href: null },
                            { id: 'problems', label: t('problems'), icon: ListOrdered, href: null },
                            { id: 'scoreboard', label: t('scoreboard'), icon: BarChart3, href: null },
                            { id: 'stats', label: t('statistics'), icon: Activity, href: `/contests/${key}/stats`, external: true }
                        ].map((tab) => (
                            <button
                                key={tab.id}
                                onClick={() => tab.href ? window.location.href = tab.href : setActiveTab(tab.id as any)}
                                className={cn(
                                    "relative py-6 text-xs font-black uppercase tracking-[0.2em] flex items-center gap-3 transition-all cursor-pointer",
                                    activeTab === tab.id ? "text-primary scale-105" : "text-muted-foreground hover:text-foreground"
                                )}
                            >
                                <tab.icon size={16} />
                                {tab.label}
                                {activeTab === tab.id && !tab.href && (
                                    <motion.div layoutId="tab-underline-contest" className="absolute bottom-0 left-0 right-0 h-1.5 bg-primary rounded-t-full shadow-[0_-4px_10px_rgba(var(--primary),0.3)]" />
                                )}
                            </button>
                        ))}
                    </div>
                </div>
            </div>

            {/* Main Content */}
            <AnimatePresence mode="wait">
                {activeTab === 'dashboard' && (
                    <motion.div
                        key="dashboard"
                        initial={{ opacity: 0, scale: 0.98 }}
                        animate={{ opacity: 1, scale: 1 }}
                        exit={{ opacity: 0, scale: 0.98 }}
                        className="grid grid-cols-1 lg:grid-cols-3 gap-8"
                    >
                        <div className="lg:col-span-2 space-y-8">
                            <section className="p-10 rounded-[3rem] border bg-card shadow-sm space-y-8 min-h-[40vh]">
                                <div className="space-y-4">
                                    <h2 className="text-3xl font-black tracking-tight">{t('dashboard')}</h2>
                                    <div className="h-1.5 w-20 bg-primary rounded-full" />
                                </div>

                                {contest.summary && (
                                    <div className="bg-muted/30 border border-dashed rounded-[2rem] p-8 text-sm font-medium leading-relaxed italic text-muted-foreground">
                                        {contest.summary}
                                    </div>
                                )}

                                <div className="prose prose-zinc dark:prose-invert max-w-none text-muted-foreground leading-relaxed">
                                    {contest.description || "The contest description will be displayed here."}
                                </div>
                            </section>

                            <section className="pt-10">
                                <Comments page={`c/${key}`} />
                            </section>
                        </div>

                        <div className="space-y-6">
                            <section className="p-8 rounded-[3rem] border bg-card shadow-sm space-y-8">
                                <h3 className="text-xs font-black uppercase tracking-widest text-primary flex items-center gap-3">
                                    <Info size={16} />
                                    Rules & Format
                                </h3>
                                <div className="space-y-4">
                                    <div className="space-y-1.5">
                                        <p className="text-[10px] uppercase font-black text-muted-foreground tracking-widest">Contest System</p>
                                        <p className="text-sm font-black">{contest.format} (DMOJ)</p>
                                    </div>
                                    <div className="space-y-1.5">
                                        <p className="text-[10px] uppercase font-black text-muted-foreground tracking-widest">Time Constraint</p>
                                        <p className="text-sm font-black">{contest.time_limit ? `${contest.time_limit / 60} Minutes` : 'Infinite Window'}</p>
                                    </div>
                                    <div className="pt-4 border-t space-y-4">
                                        <div className="flex items-center justify-between text-xs font-bold">
                                            <span className="text-muted-foreground">Scoreboard</span>
                                            <span className="text-emerald-500">Public</span>
                                        </div>
                                        <div className="flex items-center justify-between text-xs font-bold">
                                            <span className="text-muted-foreground">Clars</span>
                                            <span className="text-blue-500">Enabled</span>
                                        </div>
                                    </div>
                                </div>
                            </section>

                            <section className="p-8 rounded-[3rem] bg-zinc-900 text-zinc-100 border border-zinc-800 shadow-xl space-y-8">
                                <h3 className="text-xs font-black uppercase tracking-widest text-amber-500 flex items-center gap-3">
                                    <Clock size={16} />
                                    Timeline
                                </h3>
                                <div className="space-y-6 relative pl-4 border-l border-zinc-700">
                                    <div className="relative">
                                        <div className="absolute -left-[21px] top-1 w-2.5 h-2.5 rounded-full bg-emerald-500 shadow-[0_0_10px_rgba(16,185,129,0.5)]" />
                                        <p className="text-[10px] font-black uppercase text-zinc-500 tracking-widest">Start Time</p>
                                        <p className="text-sm font-black">{dayjs(contest.start_time).format('HH:mm, DD MMM YYYY')}</p>
                                    </div>
                                    <div className="relative">
                                        <div className="absolute -left-[21px] top-1 w-2.5 h-2.5 rounded-full bg-rose-500" />
                                        <p className="text-[10px] font-black uppercase text-zinc-500 tracking-widest">End Time</p>
                                        <p className="text-sm font-black">{dayjs(contest.end_time).format('HH:mm, DD MMM YYYY')}</p>
                                    </div>
                                </div>
                            </section>
                        </div>
                    </motion.div>
                )}

                {activeTab === 'problems' && (
                    <motion.div
                        key="problems"
                        initial={{ opacity: 0, y: 30 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: -30 }}
                        className="space-y-4"
                    >
                        {contest.problems.length > 0 ? (
                            <div className="bg-card border rounded-[3rem] overflow-hidden shadow-sm">
                                <table className="w-full text-left border-collapse">
                                    <thead>
                                        <tr className="bg-muted/30 border-b">
                                            <th className="px-10 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground w-24 text-center">#</th>
                                            <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">Problem</th>
                                            <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center w-32">Status</th>
                                            <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center w-32">Points</th>
                                            <th className="px-10 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center w-32">AC Rate</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y">
                                        {contest.problems.map((prob, idx) => (
                                            <tr key={prob.code} className="group hover:bg-muted/10 transition-colors">
                                                <td className="px-10 py-6 text-center">
                                                    <div className="w-10 h-10 rounded-xl bg-primary/5 flex items-center justify-center font-black text-sm text-primary group-hover:bg-primary group-hover:text-primary-foreground transition-all">
                                                        {String.fromCharCode(65 + idx)}
                                                    </div>
                                                </td>
                                                <td className="px-6 py-6">
                                                    <Link
                                                        href={`/problems/${prob.code}?contest=${key}`}
                                                        className="block space-y-1 outline-none"
                                                    >
                                                        <h4 className="text-lg font-black group-hover:text-primary transition-colors">{prob.name}</h4>
                                                        <div className="text-[10px] font-mono font-bold text-muted-foreground uppercase opacity-50">{prob.code}</div>
                                                    </Link>
                                                </td>
                                                <td className="px-6 py-6 text-center">
                                                    {prob.is_solved ? (
                                                        <div className="inline-flex items-center justify-center p-2 rounded-full bg-emerald-500/10 text-emerald-500">
                                                            <CheckCircle2 size={24} />
                                                        </div>
                                                    ) : (
                                                        <div className="inline-flex items-center justify-center p-2 rounded-full bg-muted/30 text-muted-foreground/20">
                                                            <HelpCircle size={24} />
                                                        </div>
                                                    )}
                                                </td>
                                                <td className="px-6 py-6 text-center">
                                                    <span className="text-sm font-black">{prob.points}</span>
                                                </td>
                                                <td className="px-10 py-6 text-center">
                                                    <div className="flex flex-col items-center gap-1.5">
                                                        <span className="text-xs font-black">{Math.round(prob.ac_rate)}%</span>
                                                        <div className="w-16 h-1 bg-muted rounded-full overflow-hidden">
                                                            <div className="h-full bg-primary" style={{ width: `${prob.ac_rate}%` }} />
                                                        </div>
                                                    </div>
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        ) : (
                            <div className="p-24 text-center rounded-[3rem] border border-dashed bg-card flex flex-col items-center gap-4">
                                <Lock size={64} className="text-primary opacity-10 animate-pulse" />
                                <div className="space-y-1">
                                    <p className="text-lg font-black tracking-tight">Access Restricted</p>
                                    <p className="text-sm font-medium text-muted-foreground">Problems will be visible once the contest begins.</p>
                                </div>
                            </div>
                        )}
                    </motion.div>
                )}

                {activeTab === 'scoreboard' && (
                    <motion.div
                        key="ranking"
                        initial={{ opacity: 0, scale: 0.95 }}
                        animate={{ opacity: 1, scale: 1 }}
                        exit={{ opacity: 0, scale: 0.95 }}
                        className="rounded-[3rem] border bg-card shadow-2xl shadow-primary/5 overflow-hidden"
                    >
                        {/* Scoreboard Header with Live Indicator */}
                        <div className="flex items-center justify-between px-8 py-4 border-b bg-muted/30">
                            <div className="flex items-center gap-3">
                                <BarChart3 size={20} className="text-primary" />
                                <h3 className="text-sm font-black uppercase tracking-widest">Scoreboard</h3>
                            </div>
                            <div className="flex items-center gap-2">
                                {isRunning && status === 'connected' && (
                                    <span className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 text-emerald-500 text-xs font-bold animate-pulse">
                                        <span className="w-2 h-2 rounded-full bg-emerald-500" />
                                        <Wifi size={14} />
                                        <span className="hidden sm:inline">Live</span>
                                    </span>
                                )}
                                {lastLiveUpdate && (
                                    <span className="text-[10px] text-muted-foreground font-medium">
                                        Updated {dayjs(lastLiveUpdate).fromNow()}
                                    </span>
                                )}
                            </div>
                        </div>
                        <div className="overflow-x-auto custom-scrollbar">
                            <table className="w-full text-left border-collapse">
                                <thead>
                                    <tr className="bg-muted/30 border-b">
                                        <th className="px-10 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground w-20 text-center">Rank</th>
                                        <th className="px-8 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">Competitor</th>
                                        <th className="px-6 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center w-24">Rating</th>
                                        <th className="px-8 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center">Total</th>
                                        {ranking?.problems.map((p, i) => (
                                            <th key={i} className="px-4 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center w-28">
                                                {p.label}
                                            </th>
                                        ))}
                                        <th className="px-10 py-6 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center">Window</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-muted/50">
                                    {ranking?.rankings.map((row) => (
                                        <tr key={row.username} className="hover:bg-muted/5 transition-colors">
                                            <td className="px-10 py-8 text-center">
                                                <span className={cn(
                                                    "inline-flex items-center justify-center w-9 h-9 rounded-xl font-black text-sm",
                                                    row.rank === 1 ? "bg-amber-500 text-white shadow-[0_5px_15px_rgba(245,158,11,0.4)]" :
                                                        row.rank === 2 ? "bg-zinc-400 text-white" :
                                                            row.rank === 3 ? "bg-orange-600 text-white" :
                                                                "bg-muted text-muted-foreground"
                                                )}>
                                                    {row.rank}
                                                </span>
                                            </td>
                                            <td className="px-8 py-8">
                                                <Link href={`/user/${row.username}`} className="font-black text-base hover:text-primary transition-colors">
                                                    {row.username}
                                                </Link>
                                            </td>
                                            <td className="px-6 py-8 text-center">
                                                <div className="flex flex-col items-center gap-1">
                                                    {row.rating ? (
                                                        <>
                                                            <span className={cn(
                                                                "text-xs font-black px-2 py-0.5 rounded-md",
                                                                getRankBadgeColor(row.rating),
                                                                "text-white"
                                                            )}>
                                                                {row.rating}
                                                            </span>
                                                            {row.rating_change !== undefined && row.rating_change !== null && (
                                                                <span className={cn(
                                                                    "text-[10px] font-black",
                                                                    getRatingChangeColor(row.rating_change)
                                                                )}>
                                                                    {formatRatingChange(row.rating_change)}
                                                                </span>
                                                            )}
                                                        </>
                                                    ) : (
                                                        <span className="text-[10px] text-muted-foreground font-black">N/A</span>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-8 py-8 text-center font-black text-primary text-xl tracking-tighter">
                                                {Math.round(row.score)}
                                            </td>
                                            {row.breakdown.map((b: any, i) => (
                                                <td key={i} className="px-4 py-8 text-center">
                                                    {b ? (
                                                        <div className="space-y-1.5">
                                                            <div className={cn(
                                                                "text-sm font-black text-white px-2 py-1 rounded-lg inline-block min-w-[50px] shadow-sm",
                                                                b.points > 0 ? "bg-emerald-500" : "bg-zinc-200 text-zinc-500 dark:bg-zinc-800"
                                                            )}>
                                                                {b.points}
                                                            </div>
                                                            <div className="text-[10px] text-muted-foreground font-black opacity-50">
                                                                {b.time ? `${Math.round(b.time / 60)}m` : '-'}
                                                            </div>
                                                        </div>
                                                    ) : <span className="text-muted-foreground/30 text-lg">·</span>}
                                                </td>
                                            ))}
                                            <td className="px-10 py-8 text-center font-black font-mono text-xs text-muted-foreground">
                                                {Math.floor(row.cumtime / 60)}m
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                            {isRankingFetching && (
                                <div className="p-24 space-y-4">
                                    <Skeleton className="h-16 w-full rounded-2xl" />
                                    <Skeleton className="h-16 w-full rounded-2xl opacity-60" />
                                    <Skeleton className="h-16 w-full rounded-2xl opacity-30" />
                                </div>
                            )}
                        </div>
                    </motion.div>
                )}
            </AnimatePresence>
        </div>
    );
}
