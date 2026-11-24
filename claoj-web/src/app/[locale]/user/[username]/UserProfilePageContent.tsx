'use client';

import { useQuery } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import api, { ppBreakdownApi } from '@/lib/api';
import { UserDetail, SolvedProblem, RatingHistoryEntry, APIResponse, PPBreakdown, BlogPost, PaginatedList, UserAnalytics } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { PaginationBar } from '@/components/ui/PaginationBar';
import { buildHeatmapWeeks } from '@/lib/heatmap';
import { use, useMemo, useState } from 'react';
import {
    User as UserIcon,
    MapPin,
    Calendar,
    Trophy,
    Zap,
    Activity,
    BookOpen,
    ArrowUpRight,
    Shield,
    Clock,
    Award,
    Hash,
    TrendingUp,
    Info,
    CheckCircle2,
    ChevronDown,
    ChevronUp,
    BarChart3,
    Newspaper
} from 'lucide-react';
import { cn, getRankColor, getRankBadgeColor, getRankTitle } from '@/lib/utils';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { motion, AnimatePresence } from 'framer-motion';
import {
    LineChart,
    Line,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    ResponsiveContainer,
    Area,
    AreaChart
} from 'recharts';
import { Link } from '@/navigation';

dayjs.extend(relativeTime);

export default function UserProfilePageContent({ params }: { params: Promise<{ username: string }> }) {
    const { username } = use(params);
    const t = useTranslations('Users');
    const pt = useTranslations('Problems');
    const locale = useLocale();
    const [activeTab, setActiveTab] = useState<'solved' | 'rating' | 'about' | 'organizations' | 'blogs'>('solved');
    const [ratingView, setRatingView] = useState<'chart' | 'table'>('chart');
    const [ppBreakdownOpen, setPpBreakdownOpen] = useState(false);
    const [blogPage, setBlogPage] = useState(1);
    const BLOG_PAGE_SIZE = 5;

    const { data: user, isLoading: userLoading } = useQuery({
        queryKey: ['user', username],
        queryFn: async () => {
            const res = await api.get<UserDetail>(`/user/${username}`);
            return res.data;
        }
    });

    const { data: solvedData, isLoading: solvedLoading } = useQuery({
        queryKey: ['user', username, 'solved'],
        queryFn: async () => {
            const res = await api.get<APIResponse<SolvedProblem[]>>(`/user/${username}/solved`);
            return res.data.data;
        }
    });

    const { data: analytics, isLoading: analyticsLoading } = useQuery({
        queryKey: ['user', username, 'analytics'],
        queryFn: async () => {
            const res = await api.get<UserAnalytics>(`/user/${username}/analytics`);
            return res.data;
        },
        enabled: activeTab === 'about',
    });

    const { data: blogData, isLoading: blogsLoading } = useQuery({
        queryKey: ['user', username, 'blogs', blogPage],
        queryFn: async () => {
            const res = await api.get<PaginatedList<BlogPost>>('/blogs', {
                params: { author: username, page: blogPage, page_size: BLOG_PAGE_SIZE },
            });
            return res.data;
        },
        enabled: activeTab === 'blogs',
    });

    const { data: ratingData, isLoading: ratingLoading } = useQuery({
        queryKey: ['user', username, 'rating'],
        queryFn: async () => {
            const res = await api.get<APIResponse<RatingHistoryEntry[]>>(`/user/${username}/rating`);
            return res.data.data;
        }
    });

    const { data: ppBreakdown, isLoading: ppBreakdownLoading } = useQuery({
        queryKey: ['user', username, 'pp-breakdown'],
        queryFn: async () => {
            const res = await ppBreakdownApi.getPPBreakdown(username);
            return res.data;
        },
        enabled: ppBreakdownOpen,
    });

    const heatmapWeeks = useMemo(
        () => buildHeatmapWeeks(analytics?.activity_year ?? []),
        [analytics]
    );
    // Locale-aware weekday initials and day labels; 2023-01-01 was a Sunday.
    const weekdayLabels = useMemo(() => {
        const format = new Intl.DateTimeFormat(locale, { weekday: 'short' });
        return Array.from({ length: 7 }, (_, i) => format.format(new Date(Date.UTC(2023, 0, 1 + i))));
    }, [locale]);
    const dayLabel = useMemo(
        () => new Intl.DateTimeFormat(locale, { month: 'short', day: 'numeric', year: 'numeric' }),
        [locale]
    );

    if (userLoading) return (
        <div className="flex flex-col lg:flex-row gap-8 max-w-7xl mx-auto p-8 animate-pulse">
            <div className="w-full lg:w-72 space-y-6">
                <Skeleton className="h-48 w-48 rounded-md mx-auto" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-32 w-full" />
            </div>
            <div className="flex-1 space-y-6">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-[50vh] w-full rounded-md" />
            </div>
        </div>
    );

    if (!user) return <div className="p-20 text-center text-muted-foreground">{t('userNotFound')}</div>;

    const stats = [
        { label: t('rank'), value: user.rank ? `#${user.rank}` : '-', icon: Hash, color: 'text-muted-foreground' },
        { label: t('points'), value: Math.round(user.points), icon: Award, color: 'text-muted-foreground' },
        { label: t('pp'), value: Math.round(user.performance_points), icon: Zap, color: 'text-muted-foreground' },
        { label: t('contributionShort'), value: user.contribution_points, icon: Activity, color: 'text-muted-foreground' },
    ];

    const RatingTooltip = ({ active, payload, label }: any) => {
        if (active && payload && payload.length) {
            const data = payload[0].payload;
            return (
                <div className="bg-card border p-4 rounded-xl shadow-xl space-y-1">
                    <p className="text-xs font-medium text-muted-foreground ">{dayjs(data.date).format('MMM D, YYYY')}</p>
                    <p className="text-lg font-semibold">{data.rating}</p>
                    <p className="text-xs font-medium text-primary max-w-[200px] truncate">{data.contest}</p>
                </div>
            );
        }
        return null;
    };

    const Star = ({ size, className, fill }: any) => <Activity size={size} className={className} />;

    return (
        <div className="max-w-7xl mx-auto flex flex-col lg:flex-row gap-4 pb-20 mt-4">
            {/* Left Sidebar */}
            <aside className="w-full lg:w-72 flex flex-col gap-6">
                {/* Avatar Card */}
                <div className="bg-card border rounded-md overflow-hidden shadow-sm flex flex-col items-center p-4 space-y-4">
                    <div className="relative group">
                        <img
                            src={`https://www.gravatar.com/avatar/${user.email_hash}?s=200&d=identicon`}
                            alt={user.username}
                            className="w-40 h-40 rounded-md border"
                        />
                    </div>

                    <div className="text-center space-y-1">
                        <h1 className="text-2xl font-semibold tracking-tight">{user.display_name || user.username}</h1>
                        <p className="text-sm font-mono text-muted-foreground">@{user.username}</p>
                        <Badge variant="outline" className={cn("mt-2 text-xs font-medium px-3 py-1", getRankColor(user.rating))}>
                            {user.display_rank}
                        </Badge>
                    </div>
                </div>

                {/* Points Card */}
                <div className="bg-card border rounded-md p-6 shadow-sm space-y-6">
                    <div className="grid grid-cols-2 gap-4">
                        {stats.map((s) => (
                            <div key={s.label} className="space-y-1 group">
                                <div className="flex items-center gap-1.5 text-muted-foreground">
                                    <s.icon size={14} className={cn(s.color, "transition-transform ")} />
                                    <span className="text-xs font-medium ">{s.label}</span>
                                </div>
                                <div className="text-xl font-semibold tracking-tighter">{s.value}</div>
                            </div>
                        ))}
                    </div>

                    <div className="pt-4 border-t space-y-3">
                        <div className="flex justify-between items-center">
                            <span className="text-xs font-medium text-muted-foreground">{t('solvedCount')}</span>
                            <span className="text-sm font-medium">{user.problem_count}</span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-xs font-medium text-muted-foreground">{t('ratingRank')}</span>
                            <span className="text-sm font-medium">{user.rating_rank ? `#${user.rating_rank}` : '-'}</span>
                        </div>
                        <Link
                            href={`/submissions?user=${user.username}`}
                            className="flex items-center justify-between text-sm font-medium text-muted-foreground hover:text-primary transition-colors group/sub"
                        >
                            {t('viewSubmissions')}
                            <ArrowUpRight size={16} className="group-hover/sub:text-primary transition-colors" />
                        </Link>
                    </div>
                </div>

                {/* Additional Info */}
                <div className="bg-muted/30 border border-dashed rounded-md p-6 space-y-4 text-sm">
                    <div className="flex items-center gap-3 text-muted-foreground">
                        <Calendar size={16} />
                        <span>{t('joinedOn', { date: dayjs(user.date_joined).format('MMM YYYY') })}</span>
                    </div>
                    <div className="flex items-center gap-3 text-muted-foreground">
                        <Clock size={16} />
                        <span>{t('seenAt', { time: dayjs(user.last_access).fromNow() })}</span>
                    </div>
                    {user.organizations.length > 0 && (
                        <div className="flex items-center gap-3 text-muted-foreground">
                            <MapPin size={16} />
                            <span>{user.organizations[0].name}</span>
                        </div>
                    )}
                </div>
            </aside>

            {/* Right Content Area */}
            <main className="flex-1 space-y-6">
                {/* Tabs Navigation */}
                <div className="flex bg-card border p-1 rounded-md shadow-sm sticky top-4 z-10 overflow-x-auto no-scrollbar whitespace-nowrap">
                    {[
                        { id: 'solved', label: t('solvedProblems'), icon: CheckCircle2 },
                        { id: 'rating', label: t('ratingHistory'), icon: TrendingUp },
                        { id: 'blogs', label: t('blogs'), icon: Newspaper },
                        { id: 'organizations', label: t('organizations'), icon: Shield },
                        { id: 'about', label: t('about'), icon: Info },
                    ].map((tab) => (
                        <button
                            key={tab.id}
                            //@ts-ignore
                            onClick={() => setActiveTab(tab.id)}
                            className={cn(
                                "flex items-center gap-2 px-6 py-2 rounded text-sm font-medium transition-all",
                                activeTab === tab.id
                                    ? "bg-primary text-primary-foreground"
                                    : "text-muted-foreground hover:bg-muted"
                            )}
                        >
                            <tab.icon size={16} />
                            {tab.label}
                        </button>
                    ))}
                </div>

                {/* PP Breakdown Section */}
                <div className="bg-card border rounded-md p-6 shadow-sm">
                    <button
                        onClick={() => setPpBreakdownOpen(!ppBreakdownOpen)}
                        className="w-full flex items-center justify-between"
                    >
                        <div className="flex items-center gap-3">
                            <div className="p-2 rounded-xl bg-primary/10">
                                <BarChart3 size={20} className="text-primary" />
                            </div>
                            <div className="text-left">
                                <h3 className="text-lg font-semibold tracking-tight">{t('ppBreakdownTitle')}</h3>
                                <p className="text-xs text-muted-foreground">{t('ppBreakdownSubtitle')}</p>
                            </div>
                        </div>
                        <div className="flex items-center gap-2">
                            <span className="text-2xl font-semibold text-primary">{Math.round(user.performance_points)}</span>
                            {ppBreakdownOpen ? <ChevronUp size={20} className="text-muted-foreground" /> : <ChevronDown size={20} className="text-muted-foreground" />}
                        </div>
                    </button>

                    <AnimatePresence>
                        {ppBreakdownOpen && (
                            <motion.div
                                initial={{ height: 0, opacity: 0 }}
                                animate={{ height: 'auto', opacity: 1 }}
                                exit={{ height: 0, opacity: 0 }}
                                transition={{ duration: 0.3 }}
                                className="overflow-hidden"
                            >
                                <div className="pt-6 space-y-6">
                                    {ppBreakdownLoading ? (
                                        <div className="space-y-4">
                                            <Skeleton className="h-24 w-full rounded-md" />
                                            <Skeleton className="h-64 w-full rounded-md" />
                                        </div>
                                    ) : ppBreakdown ? (
                                        <>
                                            {/* Summary Cards */}
                                            <div className="grid grid-cols-3 gap-4">
                                                <div className="bg-muted/30 rounded-md p-4 border text-center">
                                                    <p className="text-xs font-medium text-muted-foreground">{t('weightedSum')}</p>
                                                    <p className="text-2xl font-semibold text-primary">{ppBreakdown.weighted_sum.toFixed(2)}</p>
                                                </div>
                                                <div className="bg-muted/30 rounded-md p-4 border text-center">
                                                    <p className="text-xs font-medium text-muted-foreground">{t('bonusPoints')}</p>
                                                    <p className="text-2xl font-semibold text-emerald-500">{ppBreakdown.bonus.bonus_points.toFixed(2)}</p>
                                                    <p className="text-xs text-muted-foreground mt-1">{t('problemsSolvedCount', { count: ppBreakdown.bonus.solved_count })}</p>
                                                </div>
                                                <div className="bg-primary/10 rounded-md p-4 border border-primary/20 text-center">
                                                    <p className="text-xs font-medium text-muted-foreground">{t('totalPP')}</p>
                                                    <p className="text-2xl font-semibold text-primary">{ppBreakdown.total.toFixed(2)}</p>
                                                </div>
                                            </div>

                                            {/* Problems Table */}
                                            <div className="overflow-x-auto custom-scrollbar">
                                                <table className="w-full text-left border-collapse">
                                                    <thead>
                                                        <tr className="bg-muted/30 border-b">
                                                            <th className="px-4 py-3 text-xs font-medium text-muted-foreground">#</th>
                                                            <th className="px-4 py-3 text-xs font-medium text-muted-foreground">{t('problem')}</th>
                                                            <th className="px-4 py-3 text-xs font-medium text-muted-foreground text-right">{t('points')}</th>
                                                            <th className="px-4 py-3 text-xs font-medium text-muted-foreground text-right">{t('weight')}</th>
                                                            <th className="px-4 py-3 text-xs font-medium text-muted-foreground text-right">{t('contribution')}</th>
                                                        </tr>
                                                    </thead>
                                                    <tbody className="divide-y divide-muted/50">
                                                        {ppBreakdown.problems.slice(0, 100).map((p, idx) => (
                                                            <tr key={p.code} className="hover:bg-muted/5 transition-colors">
                                                                <td className="px-4 py-3 text-sm font-medium text-muted-foreground">{idx + 1}</td>
                                                                <td className="px-4 py-3">
                                                                    <Link
                                                                        href={`/problem/${p.code}`}
                                                                        className="text-sm font-medium hover:text-primary transition-colors"
                                                                    >
                                                                        {p.code}
                                                                    </Link>
                                                                    <p className="text-xs text-muted-foreground truncate max-w-[200px]">{p.name}</p>
                                                                </td>
                                                                <td className="px-4 py-3 text-right text-sm font-medium">{p.points.toFixed(2)}</td>
                                                                <td className="px-4 py-3 text-right">
                                                                    <span className="text-xs font-medium text-muted-foreground">{p.weight.toFixed(4)}</span>
                                                                </td>
                                                                <td className="px-4 py-3 text-right">
                                                                    <span className="text-sm font-medium text-primary">{p.contribution.toFixed(2)}</span>
                                                                </td>
                                                            </tr>
                                                        ))}
                                                    </tbody>
                                                </table>
                                            </div>

                                            {/* Formula Info */}
                                            <div className="bg-muted/20 rounded-xl p-4 border border-dashed">
                                                <p className="text-xs font-medium text-muted-foreground mb-2">{t('ppFormula')}</p>
                                                <code className="text-xs font-mono text-muted-foreground block">
                                                    {t('ppFormulaExpression')}
                                                </code>
                                            </div>
                                        </>
                                    ) : (
                                        <p className="text-center text-muted-foreground italic py-12">{t('noPPBreakdown')}</p>
                                    )}
                                </div>
                            </motion.div>
                        )}
                    </AnimatePresence>
                </div>

                {/* Tab Content */}
                <div className="min-h-[60vh] relative">
                    <AnimatePresence mode="wait">
                        {activeTab === 'solved' && (
                            <motion.div
                                key="solved"
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                                exit={{ opacity: 0, y: -10 }}
                                className="space-y-6"
                            >
                                <div className="bg-card border rounded-md p-4 shadow-sm">
                                    <h3 className="text-xl font-semibold tracking-tight mb-6 flex items-center gap-2">
                                        {t('solvedProblems')}
                                        <Badge variant="secondary" className="font-medium">{user.problem_count}</Badge>
                                    </h3>
                                    {solvedLoading ? (
                                        <div className="space-y-3">
                                            {Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} className="h-12 w-full rounded-lg" />)}
                                        </div>
                                    ) : solvedData && solvedData.length > 0 ? (
                                        <div className="overflow-x-auto custom-scrollbar">
                                            <table className="w-full text-left border-collapse">
                                                <thead>
                                                    <tr className="bg-muted/30 border-b">
                                                        <th className="px-4 py-3 text-xs font-medium text-muted-foreground">{t('problem')}</th>
                                                        <th className="px-4 py-3 text-xs font-medium text-muted-foreground">{t('date')}</th>
                                                        <th className="px-4 py-3 text-xs font-medium text-muted-foreground">{t('language')}</th>
                                                        <th className="px-4 py-3 text-xs font-medium text-muted-foreground text-right">{t('tests')}</th>
                                                        <th className="px-4 py-3 text-xs font-medium text-muted-foreground text-right">{t('points')}</th>
                                                    </tr>
                                                </thead>
                                                <tbody className="divide-y divide-muted/50">
                                                    {solvedData.map((p) => (
                                                        <tr key={p.code} className="hover:bg-muted/5 transition-colors">
                                                            <td className="px-4 py-3">
                                                                <Link href={`/problem/${p.code}`} className="text-sm font-medium hover:text-primary transition-colors">
                                                                    {p.code}
                                                                </Link>
                                                                <p className="text-xs text-muted-foreground truncate max-w-[260px]">{p.name}</p>
                                                            </td>
                                                            <td className="px-4 py-3 text-sm text-muted-foreground whitespace-nowrap">
                                                                {dayjs(p.date).format('MMM D, YYYY')}
                                                            </td>
                                                            <td className="px-4 py-3 text-sm font-mono text-muted-foreground">{p.language}</td>
                                                            <td className="px-4 py-3 text-right text-sm text-muted-foreground whitespace-nowrap">
                                                                {p.case_points} / {p.case_total}
                                                            </td>
                                                            <td className="px-4 py-3 text-right text-sm font-medium">{p.points.toFixed(2)}</td>
                                                        </tr>
                                                    ))}
                                                </tbody>
                                            </table>
                                        </div>
                                    ) : (
                                        <p className="text-center text-muted-foreground italic py-12 border border-dashed rounded-md bg-muted/20">{t('noProblemsSolved')}</p>
                                    )}
                                </div>
                            </motion.div>
                        )}

                        {activeTab === 'blogs' && (
                            <motion.div
                                key="blogs"
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                                exit={{ opacity: 0, y: -10 }}
                                className="space-y-6"
                            >
                                <div className="bg-card border rounded-md p-4 shadow-sm">
                                    <h3 className="text-xl font-semibold tracking-tight mb-6 flex items-center gap-2">
                                        {t('blogs')}
                                        {typeof blogData?.total === 'number' && (
                                            <Badge variant="secondary" className="font-medium">{blogData.total}</Badge>
                                        )}
                                    </h3>
                                    {blogsLoading ? (
                                        <div className="space-y-3">
                                            {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-14 w-full rounded-lg" />)}
                                        </div>
                                    ) : blogData && blogData.data.length > 0 ? (
                                        <div className="divide-y divide-muted/50">
                                            {blogData.data.map((post) => (
                                                <Link
                                                    key={post.id}
                                                    href={`/post/${post.id}-${post.slug}`}
                                                    className="flex items-center justify-between gap-4 py-4 group/post"
                                                >
                                                    <div className="min-w-0">
                                                        <p className="text-sm font-medium truncate group-hover/post:text-primary transition-colors">{post.title}</p>
                                                        <p className="text-xs text-muted-foreground mt-1">
                                                            {dayjs(post.publish_on).format('MMM D, YYYY')}
                                                        </p>
                                                    </div>
                                                    <ArrowUpRight size={18} className="text-muted-foreground shrink-0" />
                                                </Link>
                                            ))}
                                        </div>
                                    ) : (
                                        <p className="text-center text-muted-foreground italic py-12 border border-dashed rounded-md bg-muted/20">{t('noBlogs')}</p>
                                    )}
                                </div>
                                {blogData && (blogData.total ?? 0) > BLOG_PAGE_SIZE && (
                                    <PaginationBar
                                        page={blogPage}
                                        onPageChange={setBlogPage}
                                        total={blogData.total}
                                        pageSize={BLOG_PAGE_SIZE}
                                    />
                                )}
                            </motion.div>
                        )}

                        {activeTab === 'rating' && (
                            <motion.div
                                key="rating"
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                                exit={{ opacity: 0, y: -10 }}
                                className="space-y-6"
                            >
                                <div className="bg-card border rounded-md p-4 shadow-sm">
                                    <div className="flex items-center justify-between mb-8">
                                        <h3 className="text-xl font-semibold tracking-tight">{t('performanceAndRating')}</h3>
                                        <div className="flex bg-muted/30 p-1 rounded-xl">
                                            <button
                                                onClick={() => setRatingView('chart')}
                                                className={cn(
                                                    "px-4 py-2 rounded-lg text-xs font-medium transition-all",
                                                    ratingView === 'chart' ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:bg-muted"
                                                )}
                                            >
                                                {t('chartView')}
                                            </button>
                                            <button
                                                onClick={() => setRatingView('table')}
                                                className={cn(
                                                    "px-4 py-2 rounded-lg text-xs font-medium transition-all",
                                                    ratingView === 'table' ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:bg-muted"
                                                )}
                                            >
                                                {t('tableView')}
                                            </button>
                                        </div>
                                    </div>

                                    {ratingView === 'chart' ? (
                                        <div className="h-[400px] w-full">
                                            {ratingLoading ? (
                                                <Skeleton className="h-full w-full rounded-md" />
                                            ) : ratingData && ratingData.length > 0 ? (
                                                <ResponsiveContainer width="100%" height="100%">
                                                    <AreaChart data={ratingData} margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
                                                        <defs>
                                                            <linearGradient id="colorRating" x1="0" y1="0" x2="0" y2="1">
                                                                <stop offset="5%" stopColor="hsl(var(--primary))" stopOpacity={0.3} />
                                                                <stop offset="95%" stopColor="hsl(var(--primary))" stopOpacity={0} />
                                                            </linearGradient>
                                                        </defs>
                                                        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="hsl(var(--muted-foreground))" opacity={0.1} />
                                                        <XAxis
                                                            dataKey="date"
                                                            axisLine={false}
                                                            tickLine={false}
                                                            tick={{ fontSize: 10, fontWeight: 700, fill: 'hsl(var(--muted-foreground))' }}
                                                            tickFormatter={(str) => dayjs(str).format('MMM YYYY')}
                                                            minTickGap={30}
                                                        />
                                                        <YAxis
                                                            axisLine={false}
                                                            tickLine={false}
                                                            tick={{ fontSize: 10, fontWeight: 700, fill: 'hsl(var(--muted-foreground))' }}
                                                            domain={['auto', 'auto']}
                                                        />
                                                        <Tooltip content={<RatingTooltip />} />
                                                        <Area
                                                            type="monotone"
                                                            dataKey="rating"
                                                            stroke="hsl(var(--primary))"
                                                            strokeWidth={3}
                                                            fillOpacity={1}
                                                            fill="url(#colorRating)"
                                                            animationDuration={1500}
                                                        />
                                                    </AreaChart>
                                                </ResponsiveContainer>
                                            ) : (
                                                <div className="h-full border-2 border-dashed rounded-md flex flex-col items-center justify-center text-muted-foreground gap-4">
                                                    <TrendingUp size={48} className="opacity-10" />
                                                    <p className="font-medium">{t('noRatedContests')}</p>
                                                </div>
                                            )}
                                        </div>
                                    ) : (
                                        <div className="overflow-x-auto custom-scrollbar">
                                            <table className="w-full text-left border-collapse">
                                                <thead>
                                                    <tr className="bg-muted/30 border-b">
                                                        <th className="px-6 py-4 text-xs font-medium text-muted-foreground">{t('date')}</th>
                                                        <th className="px-6 py-4 text-xs font-medium text-muted-foreground">{t('contest')}</th>
                                                        <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center">{t('rank')}</th>
                                                        <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center">{t('rating')}</th>
                                                        <th className="px-6 py-4 text-xs font-medium text-muted-foreground text-center">{t('rankTitle')}</th>
                                                    </tr>
                                                </thead>
                                                <tbody className="divide-y divide-muted/50">
                                                    {ratingLoading ? (
                                                        Array.from({ length: 5 }).map((_, i) => (
                                                            <tr key={i}>
                                                                <td className="px-6 py-4"><Skeleton className="h-4 w-24" /></td>
                                                                <td className="px-6 py-4"><Skeleton className="h-4 w-40" /></td>
                                                                <td className="px-6 py-4"><Skeleton className="h-4 w-12 mx-auto" /></td>
                                                                <td className="px-6 py-4"><Skeleton className="h-4 w-16 mx-auto" /></td>
                                                                <td className="px-6 py-4"><Skeleton className="h-4 w-20 mx-auto" /></td>
                                                            </tr>
                                                        ))
                                                    ) : ratingData && ratingData.length > 0 ? (
                                                        ratingData.slice().reverse().map((entry, idx) => (
                                                            <tr key={idx} className="hover:bg-muted/5 transition-colors">
                                                                <td className="px-6 py-4 text-sm font-medium">
                                                                    {dayjs(entry.date).format('MMM D, YYYY')}
                                                                </td>
                                                                <td className="px-6 py-4">
                                                                    <Link href={`/contest/${entry.contest_key}`} className="text-sm font-medium hover:text-primary transition-colors">
                                                                        {entry.contest}
                                                                    </Link>
                                                                </td>
                                                                <td className="px-6 py-4 text-center">
                                                                    <span className="text-sm font-medium">{entry.rating >= ratingData[idx]?.rating ? '#' : ''}{idx + 1}</span>
                                                                </td>
                                                                <td className="px-6 py-4 text-center">
                                                                    <span className={cn(
                                                                        "text-xs font-medium px-2 py-1 rounded-md",
                                                                        getRankBadgeColor(entry.rating),
                                                                        "text-white"
                                                                    )}>
                                                                        {entry.rating}
                                                                    </span>
                                                                </td>
                                                                <td className="px-6 py-4 text-center">
                                                                    <span className="text-xs font-medium text-muted-foreground">
                                                                        {getRankTitle(entry.rating)}
                                                                    </span>
                                                                </td>
                                                            </tr>
                                                        ))
                                                    ) : (
                                                        <tr>
                                                            <td colSpan={5} className="px-6 py-12 text-center text-muted-foreground">
                                                                {t('noRatingHistory')}
                                                            </td>
                                                        </tr>
                                                    )}
                                                </tbody>
                                            </table>
                                        </div>
                                    )}
                                </div>
                            </motion.div>
                        )}

                        {activeTab === 'organizations' && (
                            <motion.div
                                key="organizations"
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                                exit={{ opacity: 0, y: -10 }}
                                className="grid grid-cols-1 md:grid-cols-2 gap-4"
                            >
                                {user.organizations.map((org) => (
                                    <div key={org.id} className="bg-card border rounded-md p-6 flex justify-between items-center group hover:border-primary/50 transition-colors shadow-sm">
                                        <div className="flex items-center gap-4">
                                            <div className="w-12 h-12 rounded-md bg-muted flex items-center justify-center font-semibold text-lg">
                                                {org.name[0]}
                                            </div>
                                            <div className="space-y-1">
                                                <h4 className="font-medium tracking-tight group-hover:text-primary transition-colors">{org.name}</h4>
                                                <p className="text-xs font-medium text-muted-foreground ">{t('activeMember')}</p>
                                            </div>
                                        </div>
                                        <ArrowUpRight size={20} className="text-muted-foreground group-hover:text-primary transition-colors" />
                                    </div>
                                ))}
                                {user.organizations.length === 0 && (
                                    <div className="col-span-full py-20 text-center text-muted-foreground border-2 border-dashed rounded-md">
                                        <Shield size={48} className="mx-auto mb-4 opacity-10" />
                                        <p className="font-medium italic">{t('noOrganizations')}</p>
                                    </div>
                                )}
                            </motion.div>
                        )}

                        {activeTab === 'about' && (
                            <motion.div
                                key="about"
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                                exit={{ opacity: 0, y: -10 }}
                                className="space-y-6"
                            >
                                <div className="bg-card border rounded-md p-4 shadow-sm space-y-4">
                                    <div className="flex flex-wrap items-center justify-between gap-2">
                                        <h3 className="text-xl font-semibold tracking-tight">{t('submissionActivity')}</h3>
                                        <span className="text-xs font-medium text-muted-foreground">
                                            {t('activitySummary', { count: (analytics?.activity_year ?? []).reduce((sum, d) => sum + d.count, 0) })}
                                        </span>
                                    </div>
                                    {analyticsLoading ? (
                                        <Skeleton className="h-28 w-full rounded-md" />
                                    ) : heatmapWeeks.length === 0 ? (
                                        <p className="text-sm text-muted-foreground italic py-8 text-center border border-dashed rounded-md bg-muted/20">
                                            {t('noActivity')}
                                        </p>
                                    ) : (
                                        <div className="overflow-x-auto custom-scrollbar">
                                            <div className="inline-flex gap-2">
                                                <div className="flex flex-col gap-1 pr-1 text-[10px] font-medium text-muted-foreground">
                                                    {weekdayLabels.map((label, i) => (
                                                        <span key={i} className="h-3 leading-3">{label}</span>
                                                    ))}
                                                </div>
                                                <div className="flex gap-1">
                                                    {heatmapWeeks.map((week, wi) => (
                                                        <div key={wi} className="flex flex-col gap-1">
                                                            {week.map((cell, di) => cell === null ? (
                                                                <div key={di} className="w-3 h-3" />
                                                            ) : (
                                                                <div
                                                                    key={cell.date}
                                                                    role="img"
                                                                    aria-label={t('activityDay', {
                                                                        count: cell.count,
                                                                        date: dayLabel.format(new Date(`${cell.date}T00:00:00Z`)),
                                                                    })}
                                                                    title={dayLabel.format(new Date(`${cell.date}T00:00:00Z`))}
                                                                    className={cn(
                                                                        'w-3 h-3 rounded-sm',
                                                                        cell.level === 0 && 'bg-muted',
                                                                        cell.level === 1 && 'bg-primary/20',
                                                                        cell.level === 2 && 'bg-primary/40',
                                                                        cell.level === 3 && 'bg-primary/60',
                                                                        cell.level === 4 && 'bg-primary'
                                                                    )}
                                                                />
                                                            ))}
                                                        </div>
                                                    ))}
                                                </div>
                                            </div>
                                            <div className="flex items-center gap-2 pt-3 text-[10px] font-medium text-muted-foreground">
                                                <span>{t('less')}</span>
                                                {[0, 1, 2, 3, 4].map((level) => (
                                                    <span
                                                        key={level}
                                                        className={cn(
                                                            'w-3 h-3 rounded-sm',
                                                            level === 0 && 'bg-muted',
                                                            level === 1 && 'bg-primary/20',
                                                            level === 2 && 'bg-primary/40',
                                                            level === 3 && 'bg-primary/60',
                                                            level === 4 && 'bg-primary'
                                                        )}
                                                    />
                                                ))}
                                                <span>{t('more')}</span>
                                            </div>
                                        </div>
                                    )}
                                </div>

                                <div className="bg-card border rounded-md p-4 shadow-sm prose prose-sm dark:prose-invert max-w-none min-h-[40vh]">
                                    <h3 className="text-xl font-semibold tracking-tight mb-6">{t('aboutUser', { username: user.username })}</h3>
                                    {user.about ? (
                                        <p className="whitespace-pre-wrap leading-relaxed text-muted-foreground">{user.about}</p>
                                    ) : (
                                        <p className="italic text-muted-foreground/50">{t('noAboutInfo')}</p>
                                    )}
                                </div>
                            </motion.div>
                        )}
                    </AnimatePresence>
                </div>
            </main>
        </div>
    );
}
