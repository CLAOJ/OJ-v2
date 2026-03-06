'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api, { ppBreakdownApi } from '@/lib/api';
import { UserDetail, SolvedProblem, RatingHistoryEntry, APIResponse, PPBreakdown } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { use, useState } from 'react';
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
    BarChart3
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

export default function UserProfilePage({ params }: { params: Promise<{ username: string }> }) {
    const { username } = use(params);
    const t = useTranslations('Auth');
    const pt = useTranslations('Problems');
    const [activeTab, setActiveTab] = useState<'solved' | 'rating' | 'about' | 'organizations'>('solved');
    const [ratingView, setRatingView] = useState<'chart' | 'table'>('chart');
    const [ppBreakdownOpen, setPpBreakdownOpen] = useState(false);

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

    if (userLoading) return (
        <div className="flex flex-col lg:flex-row gap-8 max-w-7xl mx-auto p-8 animate-pulse">
            <div className="w-full lg:w-72 space-y-6">
                <Skeleton className="h-48 w-48 rounded-3xl mx-auto" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-32 w-full" />
            </div>
            <div className="flex-1 space-y-6">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-[50vh] w-full rounded-3xl" />
            </div>
        </div>
    );

    if (!user) return <div className="p-20 text-center text-muted-foreground">User not found.</div>;

    const stats = [
        { label: 'Rank', value: user.rank ? `#${user.rank}` : '-', icon: Hash, color: 'text-blue-500' },
        { label: 'Points', value: Math.round(user.points), icon: Award, color: 'text-amber-500' },
        { label: 'PP', value: Math.round(user.performance_points), icon: Zap, color: 'text-purple-500' },
        { label: 'Contr.', value: user.contribution_points, icon: Activity, color: 'text-emerald-500' },
    ];

    const RatingTooltip = ({ active, payload, label }: any) => {
        if (active && payload && payload.length) {
            const data = payload[0].payload;
            return (
                <div className="bg-card border p-4 rounded-xl shadow-xl space-y-1">
                    <p className="text-xs font-bold text-muted-foreground uppercase tracking-widest">{dayjs(data.date).format('MMM D, YYYY')}</p>
                    <p className="text-lg font-black">{data.rating}</p>
                    <p className="text-xs font-medium text-primary max-w-[200px] truncate">{data.contest}</p>
                </div>
            );
        }
        return null;
    };

    const Star = ({ size, className, fill }: any) => <Activity size={size} className={className} />;

    return (
        <div className="max-w-7xl mx-auto flex flex-col lg:flex-row gap-8 pb-20 animate-in fade-in duration-700 mt-4">
            {/* Left Sidebar */}
            <aside className="w-full lg:w-72 flex flex-col gap-6">
                {/* Avatar Card */}
                <div className="bg-card border rounded-3xl overflow-hidden shadow-sm flex flex-col items-center p-8 space-y-4">
                    <div className="relative group">
                        <img
                            src={`https://www.gravatar.com/avatar/${user.email_hash}?s=200&d=identicon`}
                            alt={user.username}
                            className="w-40 h-40 rounded-3xl border shadow-inner group-hover:scale-105 transition-transform duration-500"
                        />
                        <div className="absolute inset-0 rounded-3xl bg-gradient-to-tr from-primary/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
                    </div>

                    <div className="text-center space-y-1">
                        <h1 className="text-2xl font-black tracking-tight">{user.display_name || user.username}</h1>
                        <p className="text-sm font-mono text-muted-foreground">@{user.username}</p>
                        <Badge variant="outline" className={cn("mt-2 text-[10px] font-black uppercase tracking-widest px-3 py-1", getRankColor(user.rating))}>
                            {user.display_rank}
                        </Badge>
                    </div>
                </div>

                {/* Points Card */}
                <div className="bg-card border rounded-3xl p-6 shadow-sm space-y-6">
                    <div className="grid grid-cols-2 gap-4">
                        {stats.map((s) => (
                            <div key={s.label} className="space-y-1 group">
                                <div className="flex items-center gap-1.5 text-muted-foreground">
                                    <s.icon size={14} className={cn(s.color, "transition-transform group-hover:scale-110")} />
                                    <span className="text-[10px] font-bold uppercase tracking-wider">{s.label}</span>
                                </div>
                                <div className="text-xl font-black tracking-tighter">{s.value}</div>
                            </div>
                        ))}
                    </div>

                    <div className="pt-4 border-t space-y-3">
                        <div className="flex justify-between items-center">
                            <span className="text-xs font-medium text-muted-foreground">Solved Count</span>
                            <span className="text-sm font-black">{user.problem_count}</span>
                        </div>
                        <div className="flex justify-between items-center">
                            <span className="text-xs font-medium text-muted-foreground">Rating Rank</span>
                            <span className="text-sm font-black">{user.rating_rank ? `#${user.rating_rank}` : '-'}</span>
                        </div>
                    </div>
                </div>

                {/* Additional Info */}
                <div className="bg-muted/30 border border-dashed rounded-3xl p-6 space-y-4 text-sm">
                    <div className="flex items-center gap-3 text-muted-foreground">
                        <Calendar size={16} />
                        <span>Joined {dayjs(user.date_joined).format('MMM YYYY')}</span>
                    </div>
                    <div className="flex items-center gap-3 text-muted-foreground">
                        <Clock size={16} />
                        <span>Seen {dayjs(user.last_access).fromNow()}</span>
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
                <div className="flex bg-card border p-1 rounded-2xl shadow-sm sticky top-4 z-10 backdrop-blur-xl bg-card/80 overflow-x-auto no-scrollbar whitespace-nowrap">
                    {[
                        { id: 'solved', label: 'Solved Problems', icon: CheckCircle2 },
                        { id: 'rating', label: 'Rating History', icon: TrendingUp },
                        { id: 'organizations', label: 'Organizations', icon: Shield },
                        { id: 'about', label: 'About', icon: Info },
                    ].map((tab) => (
                        <button
                            key={tab.id}
                            //@ts-ignore
                            onClick={() => setActiveTab(tab.id)}
                            className={cn(
                                "flex items-center gap-2 px-6 py-2 rounded-xl text-sm font-bold transition-all",
                                activeTab === tab.id
                                    ? "bg-primary text-primary-foreground shadow-lg shadow-primary/20 scale-105"
                                    : "text-muted-foreground hover:bg-muted"
                            )}
                        >
                            <tab.icon size={16} />
                            {tab.label}
                        </button>
                    ))}
                </div>

                {/* PP Breakdown Section */}
                <div className="bg-card border rounded-3xl p-6 shadow-sm">
                    <button
                        onClick={() => setPpBreakdownOpen(!ppBreakdownOpen)}
                        className="w-full flex items-center justify-between"
                    >
                        <div className="flex items-center gap-3">
                            <div className="p-2 rounded-xl bg-primary/10">
                                <BarChart3 size={20} className="text-primary" />
                            </div>
                            <div className="text-left">
                                <h3 className="text-lg font-black tracking-tight">Performance Points Breakdown</h3>
                                <p className="text-xs text-muted-foreground">See how your PP is calculated</p>
                            </div>
                        </div>
                        <div className="flex items-center gap-2">
                            <span className="text-2xl font-black text-primary">{Math.round(user.performance_points)}</span>
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
                                            <Skeleton className="h-24 w-full rounded-2xl" />
                                            <Skeleton className="h-64 w-full rounded-2xl" />
                                        </div>
                                    ) : ppBreakdown ? (
                                        <>
                                            {/* Summary Cards */}
                                            <div className="grid grid-cols-3 gap-4">
                                                <div className="bg-muted/30 rounded-2xl p-4 border text-center">
                                                    <p className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Weighted Sum</p>
                                                    <p className="text-2xl font-black text-primary">{ppBreakdown.weighted_sum.toFixed(2)}</p>
                                                </div>
                                                <div className="bg-muted/30 rounded-2xl p-4 border text-center">
                                                    <p className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Bonus Points</p>
                                                    <p className="text-2xl font-black text-emerald-500">{ppBreakdown.bonus.bonus_points.toFixed(2)}</p>
                                                    <p className="text-xs text-muted-foreground mt-1">{ppBreakdown.bonus.solved_count} problems solved</p>
                                                </div>
                                                <div className="bg-primary/10 rounded-2xl p-4 border border-primary/20 text-center">
                                                    <p className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Total PP</p>
                                                    <p className="text-2xl font-black text-primary">{ppBreakdown.total.toFixed(2)}</p>
                                                </div>
                                            </div>

                                            {/* Problems Table */}
                                            <div className="overflow-x-auto custom-scrollbar">
                                                <table className="w-full text-left border-collapse">
                                                    <thead>
                                                        <tr className="bg-muted/30 border-b">
                                                            <th className="px-4 py-3 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">#</th>
                                                            <th className="px-4 py-3 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">Problem</th>
                                                            <th className="px-4 py-3 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-right">Points</th>
                                                            <th className="px-4 py-3 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-right">Weight</th>
                                                            <th className="px-4 py-3 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-right">Contribution</th>
                                                        </tr>
                                                    </thead>
                                                    <tbody className="divide-y divide-muted/50">
                                                        {ppBreakdown.problems.slice(0, 100).map((p, idx) => (
                                                            <tr key={p.code} className="hover:bg-muted/5 transition-colors">
                                                                <td className="px-4 py-3 text-sm font-bold text-muted-foreground">{idx + 1}</td>
                                                                <td className="px-4 py-3">
                                                                    <Link
                                                                        href={`/problems/${p.code}`}
                                                                        className="text-sm font-black hover:text-primary transition-colors"
                                                                    >
                                                                        {p.code}
                                                                    </Link>
                                                                    <p className="text-xs text-muted-foreground truncate max-w-[200px]">{p.name}</p>
                                                                </td>
                                                                <td className="px-4 py-3 text-right text-sm font-black">{p.points.toFixed(2)}</td>
                                                                <td className="px-4 py-3 text-right">
                                                                    <span className="text-xs font-bold text-muted-foreground">{p.weight.toFixed(4)}</span>
                                                                </td>
                                                                <td className="px-4 py-3 text-right">
                                                                    <span className="text-sm font-black text-primary">{p.contribution.toFixed(2)}</span>
                                                                </td>
                                                            </tr>
                                                        ))}
                                                    </tbody>
                                                </table>
                                            </div>

                                            {/* Formula Info */}
                                            <div className="bg-muted/20 rounded-xl p-4 border border-dashed">
                                                <p className="text-xs font-bold text-muted-foreground uppercase tracking-wider mb-2">PP Formula</p>
                                                <code className="text-xs font-mono text-muted-foreground block">
                                                    PP = sum(0.95^i * points[i]) for top 100 problems + 300 * (1 - 0.997^solved_count)
                                                </code>
                                            </div>
                                        </>
                                    ) : (
                                        <p className="text-center text-muted-foreground italic py-12">No PP breakdown available</p>
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
                                <div className="bg-card border rounded-3xl p-8 shadow-sm">
                                    <h3 className="text-xl font-black tracking-tight mb-6 flex items-center gap-2">
                                        Solved Problems
                                        <Badge variant="secondary" className="font-bold">{user.problem_count}</Badge>
                                    </h3>
                                    {solvedLoading ? (
                                        <div className="grid grid-cols-4 md:grid-cols-8 gap-3">
                                            {Array.from({ length: 40 }).map((_, i) => <Skeleton key={i} className="h-10 w-full rounded-lg" />)}
                                        </div>
                                    ) : solvedData && solvedData.length > 0 ? (
                                        <div className="flex flex-wrap gap-2">
                                            {solvedData.map((p) => (
                                                <Link
                                                    key={p.code}
                                                    href={`/problems/${p.code}`}
                                                    className="px-3 py-1.5 rounded-lg border bg-muted/30 hover:bg-primary/10 hover:border-primary/50 hover:text-primary text-sm font-bold transition-all"
                                                >
                                                    {p.code}
                                                </Link>
                                            ))}
                                        </div>
                                    ) : (
                                        <p className="text-center text-muted-foreground italic py-12 border border-dashed rounded-2xl bg-muted/20">No problems solved yet.</p>
                                    )}
                                </div>
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
                                <div className="bg-card border rounded-3xl p-8 shadow-sm">
                                    <div className="flex items-center justify-between mb-8">
                                        <h3 className="text-xl font-black tracking-tight">Performance & Rating</h3>
                                        <div className="flex bg-muted/30 p-1 rounded-xl">
                                            <button
                                                onClick={() => setRatingView('chart')}
                                                className={cn(
                                                    "px-4 py-2 rounded-lg text-xs font-black uppercase tracking-wider transition-all",
                                                    ratingView === 'chart' ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:bg-muted"
                                                )}
                                            >
                                                Chart
                                            </button>
                                            <button
                                                onClick={() => setRatingView('table')}
                                                className={cn(
                                                    "px-4 py-2 rounded-lg text-xs font-black uppercase tracking-wider transition-all",
                                                    ratingView === 'table' ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:bg-muted"
                                                )}
                                            >
                                                Table
                                            </button>
                                        </div>
                                    </div>

                                    {ratingView === 'chart' ? (
                                        <div className="h-[400px] w-full">
                                            {ratingLoading ? (
                                                <Skeleton className="h-full w-full rounded-2xl" />
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
                                                <div className="h-full border-2 border-dashed rounded-3xl flex flex-col items-center justify-center text-muted-foreground gap-4">
                                                    <TrendingUp size={48} className="opacity-10" />
                                                    <p className="font-bold">No rated contests participated yet.</p>
                                                </div>
                                            )}
                                        </div>
                                    ) : (
                                        <div className="overflow-x-auto custom-scrollbar">
                                            <table className="w-full text-left border-collapse">
                                                <thead>
                                                    <tr className="bg-muted/30 border-b">
                                                        <th className="px-6 py-4 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">Date</th>
                                                        <th className="px-6 py-4 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">Contest</th>
                                                        <th className="px-6 py-4 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center">Rank</th>
                                                        <th className="px-6 py-4 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center">Rating</th>
                                                        <th className="px-6 py-4 text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground text-center">Title</th>
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
                                                                <td className="px-6 py-4 text-sm font-bold">
                                                                    {dayjs(entry.date).format('MMM D, YYYY')}
                                                                </td>
                                                                <td className="px-6 py-4">
                                                                    <Link href={`/contests/${entry.contest_key}`} className="text-sm font-black hover:text-primary transition-colors">
                                                                        {entry.contest}
                                                                    </Link>
                                                                </td>
                                                                <td className="px-6 py-4 text-center">
                                                                    <span className="text-sm font-black">{entry.rating >= ratingData[idx]?.rating ? '#' : ''}{idx + 1}</span>
                                                                </td>
                                                                <td className="px-6 py-4 text-center">
                                                                    <span className={cn(
                                                                        "text-xs font-black px-2 py-1 rounded-md",
                                                                        getRankBadgeColor(entry.rating),
                                                                        "text-white"
                                                                    )}>
                                                                        {entry.rating}
                                                                    </span>
                                                                </td>
                                                                <td className="px-6 py-4 text-center">
                                                                    <span className="text-[10px] font-black uppercase tracking-wider text-muted-foreground">
                                                                        {getRankTitle(entry.rating)}
                                                                    </span>
                                                                </td>
                                                            </tr>
                                                        ))
                                                    ) : (
                                                        <tr>
                                                            <td colSpan={5} className="px-6 py-12 text-center text-muted-foreground">
                                                                No rating history available
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
                                    <div key={org.id} className="bg-card border rounded-3xl p-6 flex justify-between items-center group hover:border-primary/50 transition-colors shadow-sm">
                                        <div className="flex items-center gap-4">
                                            <div className="w-12 h-12 rounded-2xl bg-muted flex items-center justify-center font-black text-lg">
                                                {org.name[0]}
                                            </div>
                                            <div className="space-y-1">
                                                <h4 className="font-black tracking-tight group-hover:text-primary transition-colors">{org.name}</h4>
                                                <p className="text-[10px] uppercase font-bold text-muted-foreground tracking-widest">Active Member</p>
                                            </div>
                                        </div>
                                        <ArrowUpRight size={20} className="text-muted-foreground group-hover:text-primary transition-colors" />
                                    </div>
                                ))}
                                {user.organizations.length === 0 && (
                                    <div className="col-span-full py-20 text-center text-muted-foreground border-2 border-dashed rounded-3xl">
                                        <Shield size={48} className="mx-auto mb-4 opacity-10" />
                                        <p className="font-bold italic">No organizations joined yet.</p>
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
                                className="bg-card border rounded-3xl p-8 shadow-sm prose prose-sm dark:prose-invert max-w-none min-h-[40vh]"
                            >
                                <h3 className="text-xl font-black tracking-tight mb-6">About {user.username}</h3>
                                {user.about ? (
                                    <p className="whitespace-pre-wrap leading-relaxed text-muted-foreground">{user.about}</p>
                                ) : (
                                    <p className="italic text-muted-foreground/50">This user hasn't shared anything about themselves yet.</p>
                                )}
                            </motion.div>
                        )}
                    </AnimatePresence>
                </div>
            </main>
        </div>
    );
}
