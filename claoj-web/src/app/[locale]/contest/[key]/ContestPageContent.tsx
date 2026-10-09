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
import { useRequireAuth } from '@/hooks/useRequireAuth';
import { toast } from 'sonner';

dayjs.extend(duration);
dayjs.extend(relativeTime);

export default function ContestPageContent({ params }: { params: Promise<{ key: string }> }) {
    const { key } = use(params);
    const t = useTranslations('Contest');
    const tAuth = useTranslations('Auth');
    const tCommon = useTranslations('Common');
    const { requireAuth } = useRequireAuth();
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
        },
        // Without this, React Query absorbed the rejection and a failed join
        // looked identical to no click at all.
        onError: (err: any) => {
            toast.error(err.response?.data?.error || t('joinFailed'));
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

    if (isFetching) return <div className="p-8 max-w-7xl mx-auto"><Skeleton className="h-[60vh] w-full rounded-md" /></div>;
    if (!contest) return <div className="p-8 text-center text-muted-foreground">{t('notFound')}</div>;

    const isRunning = dayjs().isAfter(contest.start_time) && dayjs().isBefore(contest.end_time);
    const isPast = dayjs().isAfter(contest.end_time);

    return (
        <div className="max-w-7xl mx-auto space-y-4 pb-20 mt-4">
            {/* Contest Header */}
            <div className="relative overflow-hidden rounded-md border bg-card">
                <div className="absolute top-0 right-0 p-16 opacity-5 pointer-events-none rotate-12">
                    <Trophy size={240} className="text-primary" />
                </div>

                <div className="p-4 md:p-6 md:pb-4 space-y-4 relative">
                    <div className="space-y-4">
                        <div className="flex flex-wrap items-center gap-4">
                            <Badge variant="outline" className="px-4 py-1.5 rounded border-primary/20 bg-primary/5 text-primary text-xs font-medium shadow-sm">
                                {t('formatBadge', { format: contest.format ?? '' })}
                            </Badge>
                            <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground bg-muted/30 px-4 py-1.5 rounded border border-dashed tracking-wide">
                                <Clock size={14} className="text-primary" />
                                {timeLeft}
                            </div>
                        </div>
                        <h1 className="text-4xl md:text-6xl font-semibold tracking-tighter leading-none">
                            {contest.name}
                        </h1>
                    </div>

                    <div className="flex flex-wrap items-center gap-4 pt-4">
                        {isRunning && (
                            <button
                                onClick={() => requireAuth(() => joinContest(), tAuth('loginToJoinContest'))}
                                disabled={isJoining || contest.is_joined}
                                className={cn(
                                    "px-4 h-10 rounded-md font-medium transition-all flex items-center gap-3",
                                    contest.is_joined
                                        ? "bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 cursor-default"
                                        : "bg-primary text-primary-foreground "
                                )}
                            >
                                {contest.is_joined ? <CheckCircle2 size={20} /> : <Play size={20} fill="currentColor" />}
                                {contest.is_joined ? t('participating') : t('joinContest')}
                            </button>
                        )}
                        <div className="flex items-center gap-2 px-6 h-10 rounded-md bg-card border font-medium text-sm shadow-sm group">
                            <BarChart3 size={18} className="text-muted-foreground" />
                            {contest.is_rated ? t('rated') : t('unrated')}
                        </div>
                    </div>
                </div>

                {/* Navigation Tabs */}
                <div className="px-4 md:px-6 border-t bg-muted/10">
                    <div className="flex gap-10 overflow-x-auto no-scrollbar whitespace-nowrap">
                        {[
                            { id: 'dashboard', label: t('dashboard'), icon: LayoutDashboard, href: null },
                            { id: 'problems', label: t('problems'), icon: ListOrdered, href: null },
                            { id: 'scoreboard', label: t('scoreboard'), icon: BarChart3, href: null },
                            { id: 'stats', label: t('statistics'), icon: Activity, href: `/contest/${key}/stats`, external: true }
                        ].map((tab) => (
                            <button
                                key={tab.id}
                                onClick={() => tab.href ? window.location.href = tab.href : setActiveTab(tab.id as any)}
                                className={cn(
                                    "relative py-6 text-xs font-medium flex items-center gap-3 transition-all cursor-pointer",
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
                        className="grid grid-cols-1 lg:grid-cols-3 gap-4"
                    >
                        <div className="lg:col-span-2 space-y-4">
                            <section className="p-4 rounded-md border bg-card shadow-sm space-y-4 min-h-[40vh]">
                                <div className="space-y-4">
                                    <h2 className="text-3xl font-semibold tracking-tight">{t('dashboard')}</h2>
                                    <div className="h-1.5 w-20 bg-primary rounded-full" />
                                </div>

                                {contest.summary && (
                                    <div className="bg-muted/30 border border-dashed rounded-md p-4 text-sm font-medium leading-relaxed italic text-muted-foreground">
                                        {contest.summary}
                                    </div>
                                )}

                                <div className="prose prose-zinc dark:prose-invert max-w-none text-muted-foreground leading-relaxed">
                                    {contest.description || t('descriptionPlaceholder')}
                                </div>
                            </section>

                            <section className="pt-4">
                                <Comments page={`c/${key}`} />
                            </section>
                        </div>

                        <div className="space-y-6">
                            <section className="p-4 rounded-md border bg-card shadow-sm space-y-4">
                                <h3 className="text-xs font-semibold text-primary flex items-center gap-3">
                                    <Info size={16} />
                                    {t('rulesAndFormat')}
                                </h3>
                                <div className="space-y-4">
                                    <div className="space-y-1.5">
                                        <p className="text-xs font-medium text-muted-foreground ">{t('contestSystem')}</p>
                                        <p className="text-sm font-medium">{contest.format} (DMOJ)</p>
                                    </div>
                                    <div className="space-y-1.5">
                                        <p className="text-xs font-medium text-muted-foreground ">{t('timeConstraint')}</p>
                                        <p className="text-sm font-medium">{contest.time_limit ? t('minutesCount', { count: contest.time_limit / 60 }) : t('infiniteWindow')}</p>
                                    </div>
                                    <div className="pt-4 border-t space-y-4">
                                        <div className="flex items-center justify-between text-xs font-bold">
                                            <span className="text-muted-foreground">{t('scoreboard')}</span>
                                            <span className="text-emerald-500">{tCommon('public')}</span>
                                        </div>
                                        <div className="flex items-center justify-between text-xs font-bold">
                                            <span className="text-muted-foreground">{t('clars')}</span>
                                            <span className="text-blue-500">{tCommon('enabled')}</span>
                                        </div>
                                    </div>
                                </div>
                            </section>

                            <section className="p-4 rounded-md bg-zinc-900 text-zinc-100 border border-zinc-800 space-y-4">
                                <h3 className="text-sm font-semibold text-foreground flex items-center gap-3">
                                    <Clock size={16} />
                                    {t('timeline')}
                                </h3>
                                <div className="space-y-6 relative pl-4 border-l border-zinc-700">
                                    <div className="relative">
                                        <div className="absolute -left-[21px] top-1 w-2.5 h-2.5 rounded-full bg-emerald-500 shadow-[0_0_10px_rgba(16,185,129,0.5)]" />
                                        <p className="text-xs font-medium text-zinc-500 ">{t('startTime')}</p>
                                        <p className="text-sm font-medium">{dayjs(contest.start_time).format('HH:mm, DD MMM YYYY')}</p>
                                    </div>
                                    <div className="relative">
                                        <div className="absolute -left-[21px] top-1 w-2.5 h-2.5 rounded-full bg-rose-500" />
                                        <p className="text-xs font-medium text-zinc-500 ">{t('endTime')}</p>
                                        <p className="text-sm font-medium">{dayjs(contest.end_time).format('HH:mm, DD MMM YYYY')}</p>
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
                            <div className="bg-card border rounded-md overflow-hidden shadow-sm">
                                <table className="w-full text-left border-collapse">
                                    <thead>
                                        <tr className="bg-muted/30 border-b">
                                            <th className="px-10 py-6 text-xs font-medium text-muted-foreground w-24 text-center">#</th>
                                            <th className="px-6 py-6 text-xs font-medium text-muted-foreground">{t('problem')}</th>
                                            <th className="px-6 py-6 text-xs font-medium text-muted-foreground text-center w-32">{tCommon('status')}</th>
                                            <th className="px-6 py-6 text-xs font-medium text-muted-foreground text-center w-32">{t('points')}</th>
                                            <th className="px-10 py-6 text-xs font-medium text-muted-foreground text-center w-32">{t('acRate')}</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y">
                                        {contest.problems.map((prob, idx) => (
                                            <tr key={prob.code} className="group hover:bg-muted/10 transition-colors">
                                                <td className="px-10 py-6 text-center">
                                                    <div className="w-10 h-10 rounded-xl bg-primary/5 flex items-center justify-center font-medium text-sm text-primary group-hover:bg-primary group-hover:text-primary-foreground transition-all">
                                                        {String.fromCharCode(65 + idx)}
                                                    </div>
                                                </td>
                                                <td className="px-6 py-6">
                                                    <Link
                                                        href={`/problem/${prob.code}?contest=${key}`}
                                                        className="block space-y-1 outline-none"
                                                    >
                                                        <h4 className="text-lg font-semibold group-hover:text-primary transition-colors">{prob.name}</h4>
                                                        <div className="text-xs font-mono font-bold text-muted-foreground opacity-50">{prob.code}</div>
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
                                                    <span className="text-sm font-medium">{prob.points}</span>
                                                </td>
                                                <td className="px-10 py-6 text-center">
                                                    <div className="flex flex-col items-center gap-1.5">
                                                        <span className="text-xs font-medium">{Math.round(prob.ac_rate)}%</span>
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
                            <div className="p-24 text-center rounded-md border border-dashed bg-card flex flex-col items-center gap-4">
                                <Lock size={64} className="text-primary opacity-10 animate-pulse" />
                                <div className="space-y-1">
                                    <p className="text-lg font-semibold tracking-tight">{t('accessRestricted')}</p>
                                    <p className="text-sm font-medium text-muted-foreground">{t('problemsVisibleAfterStart')}</p>
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
                        className="rounded-md border bg-card overflow-hidden"
                    >
                        {/* Scoreboard Header with Live Indicator */}
                        <div className="flex items-center justify-between px-4 py-4 border-b bg-muted/30">
                            <div className="flex items-center gap-3">
                                <BarChart3 size={20} className="text-primary" />
                                <h3 className="text-sm font-semibold ">{t('scoreboard')}</h3>
                            </div>
                            <div className="flex items-center gap-2">
                                {isRunning && status === 'connected' && (
                                    <span className="flex items-center gap-2 px-3 py-1.5 rounded bg-emerald-500/10 text-emerald-500 text-xs font-medium animate-pulse">
                                        <span className="w-2 h-2 rounded-full bg-emerald-500" />
                                        <Wifi size={14} />
                                        <span className="hidden sm:inline">{t('live')}</span>
                                    </span>
                                )}
                                {lastLiveUpdate && (
                                    <span className="text-xs text-muted-foreground font-medium">
                                        {t('updatedAgo', { time: dayjs(lastLiveUpdate).fromNow() })}
                                    </span>
                                )}
                            </div>
                        </div>
                        <div className="overflow-x-auto custom-scrollbar">
                            <table className="w-full text-left border-collapse">
                                <thead>
                                    <tr className="bg-muted/30 border-b">
                                        <th className="px-10 py-6 text-xs font-medium text-muted-foreground w-20 text-center">{t('rank')}</th>
                                        <th className="px-8 py-6 text-xs font-medium text-muted-foreground">{t('competitor')}</th>
                                        <th className="px-6 py-6 text-xs font-medium text-muted-foreground text-center w-24">{t('rating')}</th>
                                        <th className="px-8 py-6 text-xs font-medium text-muted-foreground text-center">{t('total')}</th>
                                        {ranking?.problems.map((p, i) => (
                                            <th key={i} className="px-4 py-6 text-xs font-medium text-muted-foreground text-center w-28">
                                                {p.label}
                                            </th>
                                        ))}
                                        <th className="px-10 py-6 text-xs font-medium text-muted-foreground text-center">{t('window')}</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-muted/50">
                                    {ranking?.rankings.map((row) => (
                                        <tr key={row.username} className="hover:bg-muted/5 transition-colors">
                                            <td className="px-10 py-8 text-center">
                                                <span className={cn(
                                                    "inline-flex items-center justify-center w-9 h-9 rounded-xl font-medium text-sm",
                                                    row.rank === 1 ? "bg-amber-500 text-white shadow-[0_5px_15px_rgba(245,158,11,0.4)]" :
                                                        row.rank === 2 ? "bg-zinc-400 text-white" :
                                                            row.rank === 3 ? "bg-orange-600 text-white" :
                                                                "bg-muted text-muted-foreground"
                                                )}>
                                                    {row.rank}
                                                </span>
                                            </td>
                                            <td className="px-8 py-8">
                                                <Link href={`/user/${row.username}`} className="font-medium text-base hover:text-primary transition-colors">
                                                    {row.username}
                                                </Link>
                                            </td>
                                            <td className="px-6 py-8 text-center">
                                                <div className="flex flex-col items-center gap-1">
                                                    {row.rating ? (
                                                        <>
                                                            <span className={cn(
                                                                "text-xs font-medium px-2 py-0.5 rounded-md",
                                                                getRankBadgeColor(row.rating),
                                                                "text-white"
                                                            )}>
                                                                {row.rating}
                                                            </span>
                                                            {row.rating_change !== undefined && row.rating_change !== null && (
                                                                <span className={cn(
                                                                    "text-xs font-medium",
                                                                    getRatingChangeColor(row.rating_change)
                                                                )}>
                                                                    {formatRatingChange(row.rating_change)}
                                                                </span>
                                                            )}
                                                        </>
                                                    ) : (
                                                        <span className="text-xs text-muted-foreground font-medium">{tCommon('notAvailable')}</span>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-8 py-8 text-center font-semibold text-primary text-xl tracking-tighter">
                                                {Math.round(row.score)}
                                            </td>
                                            {row.breakdown.map((b: any, i) => (
                                                <td key={i} className="px-4 py-8 text-center">
                                                    {b ? (
                                                        <div className="space-y-1.5">
                                                            <div className={cn(
                                                                "text-sm font-medium text-white px-2 py-1 rounded-lg inline-block min-w-[50px] shadow-sm",
                                                                b.points > 0 ? "bg-emerald-500" : "bg-zinc-200 text-zinc-500 dark:bg-zinc-800"
                                                            )}>
                                                                {b.points}
                                                            </div>
                                                            <div className="text-xs text-muted-foreground font-medium opacity-50">
                                                                {b.time ? `${Math.round(b.time / 60)}m` : '-'}
                                                            </div>
                                                        </div>
                                                    ) : <span className="text-muted-foreground/30 text-lg">·</span>}
                                                </td>
                                            ))}
                                            <td className="px-10 py-8 text-center font-medium font-mono text-xs text-muted-foreground">
                                                {Math.floor(row.cumtime / 60)}m
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                            {isRankingFetching && (
                                <div className="p-24 space-y-4">
                                    <Skeleton className="h-16 w-full rounded-md" />
                                    <Skeleton className="h-16 w-full rounded-md opacity-60" />
                                    <Skeleton className="h-16 w-full rounded-md opacity-30" />
                                </div>
                            )}
                        </div>
                    </motion.div>
                )}
            </AnimatePresence>
        </div>
    );
}
