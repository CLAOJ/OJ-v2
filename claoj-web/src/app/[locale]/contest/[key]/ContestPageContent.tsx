'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useConfirm } from '@/components/providers/ConfirmProvider';
import api from '@/lib/api';
import { ContestDetail, RankingResponse, ContestParticipation } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { PaginationBar } from '@/components/ui/PaginationBar';
import {
    buildScoreboardCsv,
    contestTimeLimitSeconds,
    countTotalAc,
    formatClock,
    formatCountdown,
    parseContestTab,
    type ContestTab,
} from '@/lib/contestView';
import { useState, useEffect, use } from 'react';
import { useSearchParams } from 'next/navigation';
import { useRouter, usePathname } from '@/navigation';
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
    Wifi,
    Megaphone,
    Users,
    Download,
    FileText,
    LogOut
} from 'lucide-react';
import { cn, getRankColor, getRankBadgeColor, getRatingChangeColor, formatRatingChange, contestFormatLabel } from '@/lib/utils';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { motion, AnimatePresence } from 'framer-motion';
import { Link } from '@/navigation';
import Comments from '@/components/common/Comments';
import MathRenderer from '@/components/ui/MathRenderer';
import { useWebSocketContext } from '@/contexts/WebSocketContext';
import { useRequireAuth } from '@/hooks/useRequireAuth';
import { toast } from 'sonner';

dayjs.extend(relativeTime);

const PARTICIPATION_PAGE_SIZE = 50;

export default function ContestPageContent({ params }: { params: Promise<{ key: string }> }) {
    const { key } = use(params);
    const t = useTranslations('Contest');
    const confirm = useConfirm();
    const tAuth = useTranslations('Auth');
    const tCommon = useTranslations('Common');
    const { requireAuth } = useRequireAuth();
    const searchParams = useSearchParams();
    const router = useRouter();
    const pathname = usePathname();
    // Deep-linkable tabs: ?tab=participation etc. Read once on mount, then the
    // URL follows every tab change.
    const [activeTab, setActiveTab] = useState<ContestTab>(() => parseContestTab(searchParams.get('tab')));
    const [timeLeft, setTimeLeft] = useState<string>('');
    const [includeVirtual, setIncludeVirtual] = useState(false);
    const [showNames, setShowNames] = useState(true);
    const [showOrgs, setShowOrgs] = useState(true);
    const [participationPage, setParticipationPage] = useState(1);
    // v1 /contest/<c>/participations/<user>: ?user= narrows the participation
    // tab to one competitor. Follow the URL so the view is linkable.
    const urlUser = searchParams.get('user') || '';
    const [participationUser, setParticipationUser] = useState(urlUser);
    const [seenUrlUser, setSeenUrlUser] = useState(urlUser);
    if (urlUser !== seenUrlUser) {
        setSeenUrlUser(urlUser);
        setParticipationUser(urlUser);
        setParticipationPage(1);
    }
    const queryClient = useQueryClient();
    const { subscribe, unsubscribe, status } = useWebSocketContext();
    const [lastLiveUpdate, setLastLiveUpdate] = useState<Date | null>(null);

    const selectTab = (tab: ContestTab) => {
        setActiveTab(tab);
        // Keep any other query params (e.g. a future ?from= link) intact.
        const next = new URLSearchParams(searchParams.toString());
        next.set('tab', tab);
        router.replace(`${pathname}?${next.toString()}`, { scroll: false });
    };

    const { data: contest, isLoading: isFetching } = useQuery({
        queryKey: ['contest', key],
        queryFn: async () => {
            const res = await api.get<ContestDetail>(`/contest/${key}`);
            return res.data;
        }
    });

    const { data: ranking, isLoading: isRankingFetching } = useQuery({
        queryKey: ['contest-ranking', key, includeVirtual],
        queryFn: async () => {
            const res = await api.get<RankingResponse>(`/contest/${key}/ranking`, {
                params: includeVirtual ? { include_virtual: 'true' } : {}
            });
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

    const { data: participations, isLoading: isParticipationsFetching } = useQuery({
        queryKey: ['contest-participations', key, includeVirtual, participationUser, participationPage],
        queryFn: async () => {
            const res = await api.get<{ results: ContestParticipation[]; total: number; scoreboard_hidden?: boolean }>(
                `/contest/${key}/participations`,
                {
                    params: {
                        page: participationPage,
                        page_size: PARTICIPATION_PAGE_SIZE,
                        ...(includeVirtual ? { include_virtual: 'true' } : {}),
                        ...(participationUser ? { user: participationUser } : {})
                    }
                }
            );
            return res.data;
        },
        enabled: activeTab === 'participation'
    });

    const clearParticipationUser = () => {
        setParticipationUser('');
        const next = new URLSearchParams(searchParams.toString());
        next.delete('user');
        const query = next.toString();
        router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false });
    };

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

    // v1 ContestLeave only detaches the current-contest pointer; the scoreboard
    // row stays, so a later join resumes the same participation.
    const { mutate: leaveContest, isPending: isLeaving } = useMutation({
        mutationFn: async () => {
            await api.post(`/contest/${key}/leave`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['contest', key] });
        },
        onError: (err: unknown) => {
            const message = (err as { response?: { data?: { error?: string } } }).response?.data?.error;
            toast.error(message || t('leaveFailed'));
        }
    });

    useEffect(() => {
        if (!contest) return;

        let interval: ReturnType<typeof setInterval> | undefined;
        const tick = () => {
            const now = dayjs();
            const start = dayjs(contest.start_time);
            const end = dayjs(contest.end_time);

            if (now.isBefore(start)) {
                setTimeLeft(`${t('startIn')} ${formatCountdown(start.diff(now))}`);
            } else if (now.isBefore(end)) {
                setTimeLeft(`${t('endsIn')} ${formatCountdown(end.diff(now))}`);
            } else {
                setTimeLeft(t('ended'));
                clearInterval(interval);
            }
        };
        // Fill the badge right away; it used to stay empty for the first second.
        tick();
        if (dayjs().isBefore(dayjs(contest.end_time))) interval = setInterval(tick, 1000);

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
    const limitSeconds = contestTimeLimitSeconds(contest.time_limit);
    const totalAc = ranking ? countTotalAc(ranking.rankings, ranking.problems) : [];

    const downloadScoreboardCsv = () => {
        if (!ranking) return;
        const csv = buildScoreboardCsv({
            problems: ranking.problems,
            rows: ranking.rankings,
            includeName: showNames,
            includeOrg: showOrgs,
        });
        // BOM so Excel opens the UTF-8 usernames correctly.
        const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8' });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = `ranking_${key}.csv`;
        link.click();
        // Revoking synchronously can cancel the download in some browsers.
        setTimeout(() => URL.revokeObjectURL(url), 0);
    };

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
                                {t('formatBadge', { format: contestFormatLabel(contest.format) })}
                            </Badge>
                            <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground bg-muted/30 px-4 py-1.5 rounded border border-dashed">
                                <Clock size={14} className="text-primary" />
                                {timeLeft}
                            </div>
                        </div>
                        <h1 className="text-4xl md:text-6xl font-semibold tracking-tighter leading-none">
                            {contest.name}
                        </h1>
                    </div>

                    <div className="flex flex-wrap items-center gap-4 pt-4">
                        {contest.is_in_contest ? (
                            <button
                                onClick={() => requireAuth(async () => {
                                    if (await confirm(t('leaveConfirm'))) leaveContest();
                                }, tAuth('loginToJoinContest'))}
                                disabled={isLeaving}
                                className="px-4 h-10 rounded-md font-medium transition-all flex items-center gap-3 bg-muted/40 border hover:bg-muted disabled:opacity-50"
                            >
                                <LogOut size={18} />
                                {t('leaveContest')}
                            </button>
                        ) : isRunning && (
                            <button
                                onClick={() => requireAuth(() => joinContest(), tAuth('loginToJoinContest'))}
                                disabled={isJoining}
                                className="px-4 h-10 rounded-md font-medium transition-all flex items-center gap-3 bg-primary text-primary-foreground"
                            >
                                <Play size={20} fill="currentColor" />
                                {contest.is_joined ? t('rejoinContest') : t('joinContest')}
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
                            { id: 'participation', label: t('participation'), icon: Users, href: null },
                            { id: 'submissions', label: t('submissions'), icon: FileText, href: `/submissions?contest=${key}`, external: true },
                            { id: 'stats', label: t('statistics'), icon: Activity, href: `/contest/${key}/stats`, external: true }
                        ].map((tab) => (
                            <button
                                key={tab.id}
                                onClick={() => tab.href ? window.location.href = tab.href : selectTab(tab.id as ContestTab)}
                                className={cn(
                                    "relative py-6 text-xs font-medium flex items-center gap-3 transition-all cursor-pointer",
                                    activeTab === tab.id ? "text-primary" : "text-muted-foreground hover:text-foreground"
                                )}
                            >
                                <tab.icon size={16} />
                                {tab.label}
                                {activeTab === tab.id && !tab.href && (
                                    <motion.div layoutId="tab-underline-contest" className="absolute bottom-0 left-0 right-0 h-1.5 bg-primary rounded-t-full" />
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
                            {contest.announcements && contest.announcements.length > 0 && (
                                <section className="p-4 rounded-md border bg-card shadow-sm space-y-4">
                                    <div className="flex items-center gap-3">
                                        <Megaphone size={18} className="text-primary" />
                                        <h2 className="text-xl font-semibold tracking-tight">{t('announcements')}</h2>
                                    </div>
                                    <div className="space-y-4">
                                        {contest.announcements.map((announcement) => (
                                            <article key={announcement.id} className="border-l-2 border-primary/40 pl-4 space-y-1">
                                                <div className="flex flex-wrap items-baseline justify-between gap-2">
                                                    <h3 className="text-sm font-semibold">{announcement.title}</h3>
                                                    <time className="text-xs text-muted-foreground">
                                                        {dayjs(announcement.date).format('MMM D, YYYY HH:mm')}
                                                    </time>
                                                </div>
                                                <div className="prose prose-zinc dark:prose-invert max-w-none text-sm leading-relaxed">
                                                    <MathRenderer content={announcement.description} />
                                                </div>
                                            </article>
                                        ))}
                                    </div>
                                </section>
                            )}

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

                                {contest.description?.trim() && (
                                    <div className="prose prose-zinc dark:prose-invert max-w-none leading-relaxed">
                                        <MathRenderer content={contest.description} />
                                    </div>
                                )}

                                {/* v1 shows the problem list on the contest page itself;
                                    a placeholder sentence used to stand in for it. */}
                                {contest.problems.length > 0 && (
                                    <ol className="divide-y border rounded-md">
                                        {contest.problems.map((prob, idx) => (
                                            <li key={prob.code}>
                                                <Link
                                                    href={`/problem/${prob.code}?contest=${key}`}
                                                    className="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/40 transition-colors"
                                                >
                                                    <span className="w-6 font-mono text-sm text-muted-foreground">{String.fromCharCode(65 + idx)}</span>
                                                    <span className="flex-1 font-medium">{prob.name}</span>
                                                    {prob.is_solved && <CheckCircle2 size={16} className="text-emerald-600" aria-label={t('solved')} />}
                                                    <span className="text-sm tabular-nums text-muted-foreground">{prob.points}</span>
                                                </Link>
                                            </li>
                                        ))}
                                    </ol>
                                )}
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
                                        <p className="text-sm font-medium">{contestFormatLabel(contest.format)}</p>
                                    </div>
                                    <div className="space-y-1.5">
                                        <p className="text-xs font-medium text-muted-foreground ">{t('timeConstraint')}</p>
                                        <p className="text-sm font-medium">
                                            {limitSeconds > 0
                                                ? t('fixedWindow', { duration: formatClock(limitSeconds) })
                                                : t('infiniteWindow')}
                                        </p>
                                    </div>
                                    <div className="pt-4 border-t space-y-4">
                                        <div className="flex items-center justify-between text-xs font-medium">
                                            <span className="text-muted-foreground">{t('scoreboard')}</span>
                                            <span className="text-emerald-500">{tCommon('public')}</span>
                                        </div>
                                        <div className="flex items-center justify-between text-xs font-medium">
                                            <span className="text-muted-foreground">{t('clars')}</span>
                                            <span className="text-blue-500">{tCommon('enabled')}</span>
                                        </div>
                                    </div>
                                </div>
                            </section>

                            <section className="p-4 rounded-md bg-zinc-900 text-zinc-100 border border-zinc-800 space-y-4">
                                <h3 className="text-sm font-semibold text-zinc-100 flex items-center gap-3">
                                    <Clock size={16} />
                                    {t('timeline')}
                                </h3>
                                <div className="space-y-6 relative pl-4 border-l border-zinc-700">
                                    <div className="relative">
                                        <div className="absolute -left-[21px] top-1 w-2.5 h-2.5 rounded-full bg-emerald-500" />
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
                                                        <div className="text-xs font-mono font-medium text-muted-foreground opacity-50">{prob.code}</div>
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
                        <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-4 border-b bg-muted/30">
                            <div className="flex items-center gap-3">
                                <BarChart3 size={20} className="text-primary" />
                                <h3 className="text-sm font-semibold ">{t('scoreboard')}</h3>
                            </div>
                            <div className="flex flex-wrap items-center gap-3">
                                <label className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={showNames}
                                        onChange={(e) => setShowNames(e.target.checked)}
                                        className="accent-primary"
                                    />
                                    {t('showNames')}
                                </label>
                                <label className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={showOrgs}
                                        onChange={(e) => setShowOrgs(e.target.checked)}
                                        className="accent-primary"
                                    />
                                    {t('showOrganizations')}
                                </label>
                                <label className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={includeVirtual}
                                        onChange={(e) => setIncludeVirtual(e.target.checked)}
                                        className="accent-primary"
                                    />
                                    {t('showVirtual')}
                                </label>
                                <button
                                    onClick={downloadScoreboardCsv}
                                    disabled={!ranking}
                                    className="flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs font-medium transition-colors hover:bg-muted disabled:opacity-40"
                                >
                                    <Download size={14} />
                                    {t('exportCsv')}
                                </button>
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
                                                {p.code ? (
                                                    <Link
                                                        href={`/contest/${key}/rank/${p.code}`}
                                                        title={t('bestSolutionsFor', { label: p.label })}
                                                        className="hover:text-primary transition-colors"
                                                    >
                                                        {p.label}
                                                    </Link>
                                                ) : p.label}
                                            </th>
                                        ))}
                                        <th className="px-10 py-6 text-xs font-medium text-muted-foreground text-center">{t('window')}</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-muted/50">
                                    {ranking?.rankings.map((row, rowIndex) => (
                                        <tr key={row.participation_id ?? `${row.username}-${rowIndex}`} className="hover:bg-muted/5 transition-colors">
                                            <td className="px-10 py-8 text-center">
                                                <span className={cn(
                                                    "inline-flex items-center justify-center w-9 h-9 rounded-xl font-medium text-sm",
                                                    row.rank === 1 ? "bg-amber-500 text-white" :
                                                        row.rank === 2 ? "bg-zinc-400 text-white" :
                                                            row.rank === 3 ? "bg-orange-600 text-white" :
                                                                "bg-muted text-muted-foreground"
                                                )}>
                                                    {row.rank}
                                                </span>
                                            </td>
                                            <td className="px-8 py-8">
                                                <div className="flex flex-col gap-0.5">
                                                    <div className="flex items-center gap-2">
                                                        <Link href={`/user/${row.username}`} className="font-medium text-base hover:text-primary transition-colors">
                                                            {row.username}
                                                        </Link>
                                                        {row.is_virtual && (
                                                            <Badge variant="outline" className="text-xs h-4">{t('virtual')}</Badge>
                                                        )}
                                                    </div>
                                                    {showNames && row.display_name && row.display_name !== row.username && (
                                                        <span className="text-xs text-muted-foreground">{row.display_name}</span>
                                                    )}
                                                    {showOrgs && row.organization && (
                                                        <span className="text-xs text-muted-foreground">{row.organization.short_name}</span>
                                                    )}
                                                </div>
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
                                            {row.breakdown.map((b, i) => {
                                            const problemCode = ranking?.problems[i]?.code;
                                            const scoreClass = cn(
                                                "text-sm font-medium text-white px-2 py-1 rounded-lg inline-block min-w-[50px] shadow-sm",
                                                b && b.points > 0 ? "bg-emerald-500" : "bg-zinc-200 text-zinc-500 dark:bg-zinc-800"
                                            );
                                            return (
                                                <td key={i} className="px-4 py-8 text-center">
                                                    {b ? (
                                                        <div className="space-y-1.5">
                                                            {problemCode ? (
                                                                <Link
                                                                    href={`/contest/${key}/rank/${problemCode}`}
                                                                    className={cn(scoreClass, b.points > 0 && "hover:bg-emerald-600")}
                                                                >
                                                                    {b.points}
                                                                </Link>
                                                            ) : (
                                                                <div className={scoreClass}>{b.points}</div>
                                                            )}
                                                            <div className="text-xs text-muted-foreground font-medium opacity-50">
                                                                {b.time != null ? formatClock(b.time) : '-'}
                                                            </div>
                                                        </div>
                                                    ) : <span className="text-muted-foreground/30 text-lg">·</span>}
                                                </td>
                                            );
                                        })}
                                            <td className="px-10 py-8 text-center font-medium font-mono text-xs text-muted-foreground">
                                                {formatClock(row.cumtime)}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                                {ranking && ranking.rankings.length > 0 && (
                                    <tfoot>
                                        <tr className="border-t bg-muted/20">
                                            <td colSpan={2} className="px-8 py-3 text-xs font-semibold text-muted-foreground">
                                                {t('totalAc')}
                                            </td>
                                            <td colSpan={2} />
                                            {totalAc.map((count, i) => (
                                                <td key={i} className="px-4 py-3 text-center text-xs font-semibold">
                                                    {count}
                                                </td>
                                            ))}
                                            <td />
                                        </tr>
                                    </tfoot>
                                )}
                            </table>
                            {ranking?.scoreboard_hidden && (
                                <p className="p-12 text-center text-sm text-muted-foreground">{t('scoreboardHidden')}</p>
                            )}
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
            {activeTab === 'participation' && (
                    <motion.div
                        key="participation"
                        initial={{ opacity: 0, scale: 0.95 }}
                        animate={{ opacity: 1, scale: 1 }}
                        exit={{ opacity: 0, scale: 0.95 }}
                        className="rounded-md border bg-card overflow-hidden"
                    >
                        <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-4 border-b bg-muted/30">
                            <div className="flex items-center gap-3">
                                <Users size={20} className="text-primary" />
                                <h3 className="text-sm font-semibold ">{t('participation')}</h3>
                                {participationUser && (
                                    <>
                                        <Badge variant="secondary" className="text-xs">
                                            {t('participationUserFilter', { user: participationUser })}
                                        </Badge>
                                        <button
                                            onClick={clearParticipationUser}
                                            className="text-xs font-medium text-muted-foreground hover:text-primary transition-colors"
                                        >
                                            {t('clearUserFilter')}
                                        </button>
                                    </>
                                )}
                            </div>
                            <label className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground cursor-pointer">
                                <input
                                    type="checkbox"
                                    checked={includeVirtual}
                                    onChange={(e) => { setIncludeVirtual(e.target.checked); setParticipationPage(1); }}
                                    className="accent-primary"
                                />
                                {t('showVirtual')}
                            </label>
                        </div>
                        <div className="overflow-x-auto custom-scrollbar">
                            <table className="w-full text-left border-collapse">
                                <thead>
                                    <tr className="bg-muted/30 border-b">
                                        <th className="px-8 py-4 text-xs font-medium text-muted-foreground w-16 text-center">#</th>
                                        <th className="px-8 py-4 text-xs font-medium text-muted-foreground">{t('competitor')}</th>
                                        <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-28">{t('score')}</th>
                                        <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center w-32">{t('time')}</th>
                                        <th className="px-8 py-4 text-xs font-medium text-muted-foreground text-center w-32">{tCommon('status')}</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-muted/50">
                                    {participations?.results.map((row, index) => (
                                        <tr key={`${row.user}-${index}`} className="hover:bg-muted/5 transition-colors">
                                            <td className="px-8 py-4 text-center text-xs font-medium text-muted-foreground">
                                                {(participationPage - 1) * PARTICIPATION_PAGE_SIZE + index + 1}
                                            </td>
                                            <td className="px-8 py-4">
                                                <div className="flex flex-col gap-0.5">
                                                    <div className="flex items-center gap-2">
                                                        <Link href={`/user/${row.user}`} className="font-medium hover:text-primary transition-colors">
                                                            {row.user}
                                                        </Link>
                                                        {row.is_virtual && (
                                                            <Badge variant="outline" className="text-xs h-4">{t('virtual')}</Badge>
                                                        )}
                                                    </div>
                                                    {row.display_name && row.display_name !== row.user && (
                                                        <span className="text-xs text-muted-foreground">{row.display_name}</span>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-center font-semibold text-primary">
                                                {Math.round(row.score)}
                                            </td>
                                            <td className="px-6 py-4 text-center font-mono text-xs text-muted-foreground">
                                                {formatClock(row.cumtime)}
                                            </td>
                                            <td className="px-8 py-4 text-center">
                                                {row.is_disqualified ? (
                                                    <Badge variant="destructive" className="text-xs">{t('disqualified')}</Badge>
                                                ) : (
                                                    <span className="text-xs text-muted-foreground">-</span>
                                                )}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                            {isParticipationsFetching && (
                                <div className="p-12 space-y-4">
                                    <Skeleton className="h-12 w-full rounded-md" />
                                    <Skeleton className="h-12 w-full rounded-md opacity-60" />
                                </div>
                            )}
                            {participations && participations.results.length === 0 && !isParticipationsFetching && (
                                <p className="p-12 text-center text-sm text-muted-foreground">
                                    {participations.scoreboard_hidden ? t('scoreboardHidden') : t('noParticipants')}
                                </p>
                            )}
                        </div>
                        {participations && !participations.scoreboard_hidden && (
                            <PaginationBar
                                page={participationPage}
                                onPageChange={setParticipationPage}
                                total={participations.total}
                                pageSize={PARTICIPATION_PAGE_SIZE}
                                className="rounded-none border-0 border-t"
                            />
                        )}
                    </motion.div>
                )}
            </AnimatePresence>
        </div>
    );
}
