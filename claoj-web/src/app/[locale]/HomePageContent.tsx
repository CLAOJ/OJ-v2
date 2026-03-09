'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { BlogPost, Contest, User, Problem } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link, useRouter } from '@/navigation';
import { useState, useEffect } from 'react';
import { AdminWelcomeBanner } from '@/components/admin';
import {
    Flame,
    Trophy,
    Users,
    MessageSquare,
    Star,
    TrendingUp,
    BookOpen,
    Clock,
    Calendar,
    ChevronRight,
    ThumbsUp,
    ThumbsDown,
    Lock,
    MessageCircle
} from 'lucide-react';
import { cn } from '@/lib/utils';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';

dayjs.extend(relativeTime);

export default function HomePageContent() {
    const t = useTranslations('Home');
    const router = useRouter();
    const [activeTab, setActiveTab] = useState<'blog' | 'events'>('blog');

    // Fetch blog posts
    const { data: posts, isLoading: postsLoading } = useQuery({
        queryKey: ['blog-posts'],
        queryFn: async () => {
            const res = await api.get<{ data: BlogPost[] }>('/blogs?limit=10');
            return res.data.data;
        }
    });

    // Fetch ongoing contests
    const { data: ongoingContests } = useQuery({
        queryKey: ['ongoing-contests'],
        queryFn: async () => {
            const res = await api.get<{ data: Contest[] }>('/contests?status=ongoing&limit=5');
            return res.data.data;
        }
    });

    // Fetch upcoming contests
    const { data: upcomingContests } = useQuery({
        queryKey: ['upcoming-contests'],
        queryFn: async () => {
            const res = await api.get<{ data: Contest[] }>('/contests?status=upcoming&limit=5');
            return res.data.data;
        }
    });

    // Fetch top users by rating
    const { data: topRatingUsers } = useQuery({
        queryKey: ['top-rating-users'],
        queryFn: async () => {
            const res = await api.get<{ data: User[] }>('/users?order=-rating&limit=5');
            return res.data.data;
        }
    });

    // Fetch top scorers
    const { data: topScorers } = useQuery({
        queryKey: ['top-scorers'],
        queryFn: async () => {
            const res = await api.get<{ data: User[] }>('/users?order=-performance_points&limit=5');
            return res.data.data;
        }
    });

    // Fetch new problems
    const { data: newProblems } = useQuery({
        queryKey: ['new-problems'],
        queryFn: async () => {
            const res = await api.get<{ data: Problem[] }>('/problems?sort=date&order=desc&limit=5');
            return res.data.data;
        }
    });

    // Fetch recent comments
    const { data: recentComments } = useQuery({
        queryKey: ['recent-comments'],
        queryFn: async () => {
            const res = await api.get<{ data: any[] }>('/comments?page_type=blog&limit=10');
            return res.data.data;
        }
    });

    const [now, setNow] = useState(dayjs());
    useEffect(() => {
        const timer = setInterval(() => setNow(dayjs()), 1000);
        return () => clearInterval(timer);
    }, []);

    const formatContestTime = (startTime: string, endTime: string) => {
        const start = dayjs(startTime);
        const end = dayjs(endTime);
        const duration = end.diff(start);
        const days = Math.floor(duration / (1000 * 60 * 60 * 24));
        const hours = Math.floor((duration % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
        const minutes = Math.floor((duration % (1000 * 60 * 60)) / (1000 * 60));

        if (days > 0) return `${days}d ${hours}h ${minutes}m`;
        if (hours > 0) return `${hours}h ${minutes}m`;
        return `${minutes}m`;
    };

    const getTimeRemaining = (endTime: string) => {
        const end = dayjs(endTime);
        const diff = end.diff(now);
        if (diff <= 0) return 'Ended';

        const hours = Math.floor(diff / (1000 * 60 * 60));
        const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
        const seconds = Math.floor((diff % (1000 * 60)) / 1000);

        return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    };

    const getRatingClass = (rating?: number | null) => {
        if (!rating) return '';
        if (rating < 1000) return 'text-[#988f81]';
        if (rating < 1200) return 'text-[#72ff72]';
        if (rating < 1400) return 'text-[#57fcf2]';
        if (rating < 1600) return 'text-[#337dff]';
        if (rating < 1800) return 'text-[#ff55ff]';
        if (rating < 2000) return 'text-[#ff981a]';
        if (rating < 2200) return 'text-[#ff1a1a]';
        return 'text-[#ff1a1a] font-bold';
    };

    return (
        <div className="flex flex-col gap-6">
            {/* Admin Welcome Banner - Shown for admin users */}
            <AdminWelcomeBanner />

            <div className="flex flex-col lg:flex-row gap-6">
                {/* Main Content - Blog Posts */}
            <div className="flex-grow min-w-0">
                {/* Mobile Tabs */}
                <div className="md:hidden mb-4">
                    <div className="flex border rounded-lg overflow-hidden bg-card">
                        <button
                            onClick={() => setActiveTab('blog')}
                            className={cn(
                                "flex-1 py-3 text-sm font-bold transition-colors flex items-center justify-center gap-2",
                                activeTab === 'blog'
                                    ? "bg-primary text-white"
                                    : "text-gray-400 hover:bg-white/5"
                            )}
                        >
                            <BookOpen size={16} />
                            Blog
                        </button>
                        <button
                            onClick={() => setActiveTab('events')}
                            className={cn(
                                "flex-1 py-3 text-sm font-bold transition-colors flex items-center justify-center gap-2",
                                activeTab === 'events'
                                    ? "bg-primary text-white"
                                    : "text-gray-400 hover:bg-white/5"
                            )}
                        >
                            <Flame size={16} />
                            Events
                        </button>
                    </div>
                </div>

                {/* Blog Posts */}
                <div className={cn("space-y-4", activeTab !== 'blog' && "hidden md:block")}>
                    <div className="flex items-center justify-between mb-4">
                        <h2 className="text-xl font-bold tracking-tight flex items-center gap-2">
                            <BookOpen className="text-primary" size={24} />
                            Latest Blog Posts
                        </h2>
                        <Link href="/blog" className="text-sm font-bold text-primary hover:underline flex items-center gap-1">
                            View all <ChevronRight size={16} />
                        </Link>
                    </div>

                    {postsLoading ? (
                        [1, 2, 3].map(i => (
                            <div key={i} className="bg-card border rounded-lg p-6">
                                <Skeleton className="h-6 w-3/4 mb-2" />
                                <Skeleton className="h-4 w-1/2 mb-4" />
                                <Skeleton className="h-24 w-full" />
                            </div>
                        ))
                    ) : posts && posts.length > 0 ? (
                        posts.map(post => (
                            <article
                                key={post.id}
                                className={cn(
                                    "bg-card border rounded-lg overflow-hidden transition-all hover:shadow-lg hover:shadow-primary/5",
                                    post.sticky && "border-primary border-2"
                                )}
                            >
                                <div className="p-6">
                                    <div className="flex gap-4">
                                        {/* Vote Section */}
                                        <div className="flex flex-col items-center gap-1">
                                            <button className="text-gray-400 hover:text-primary transition-colors">
                                                <ThumbsUp size={20} />
                                            </button>
                                            <span className="text-sm font-bold text-gray-400">{post.score || 0}</span>
                                            <button className="text-gray-400 hover:text-red-400 transition-colors">
                                                <ThumbsDown size={20} />
                                            </button>
                                        </div>

                                        {/* Content Section */}
                                        <div className="flex-grow min-w-0">
                                            <div className="flex items-center gap-2 mb-2">
                                                <h3 className="text-xl font-bold hover:text-primary transition-colors">
                                                    <Link href={`/blog/${post.id}`} className="hover:underline">
                                                        {!post.visible && <Lock size={14} className="inline text-red-500 mr-1" />}
                                                        {post.title}
                                                    </Link>
                                                </h3>
                                                {post.sticky && <Star size={16} className="text-yellow-500 fill-yellow-500" />}
                                            </div>

                                            <div className="flex items-center gap-4 text-xs text-gray-400 mb-3 flex-wrap">
                                                <span className="flex items-center gap-1">
                                                    <Users size={12} />
                                                    {post.authors?.map((a: any) => a.username).join(', ')}
                                                </span>
                                                <span className="flex items-center gap-1">
                                                    <Clock size={12} />
                                                    {dayjs(post.publish_on).fromNow()}
                                                </span>
                                                <span className="flex items-center gap-1">
                                                    <MessageCircle size={12} />
                                                    {post.comment_count || 0}
                                                </span>
                                            </div>

                                            <div className="prose dark:prose-invert max-w-none text-sm text-muted-foreground line-clamp-3">
                                                <ReactMarkdown
                                                    remarkPlugins={[remarkGfm]}
                                                    rehypePlugins={[rehypeRaw]}
                                                >
                                                    {post.summary || post.content}
                                                </ReactMarkdown>
                                            </div>

                                            {post.summary && (
                                                <div className="mt-3">
                                                    <Link
                                                        href={`/blog/${post.id}`}
                                                        className="text-sm font-bold text-primary hover:underline"
                                                    >
                                                        Continue reading...
                                                    </Link>
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                </div>
                            </article>
                        ))
                    ) : (
                        <div className="bg-card border border-dashed rounded-lg p-8 text-center text-gray-400">
                            No blog posts available
                        </div>
                    )}
                </div>

                {/* Events Tab (for mobile) */}
                <div className={cn("space-y-6", activeTab !== 'events' && "hidden md:block")}>
                    {/* Ongoing Contests */}
                    {ongoingContests && ongoingContests.length > 0 && (
                        <div className="bg-card border rounded-xl overflow-hidden">
                            <div className="bg-muted px-4 py-3 border-b border-primary-50">
                                <h3 className="text-sm font-bold uppercase tracking-wider text-white flex items-center gap-2">
                                    <Flame className="text-yellow-500" size={18} />
                                    Ongoing Contests
                                </h3>
                            </div>
                            <div className="p-4 space-y-3">
                                {ongoingContests.map(contest => (
                                    <div
                                        key={contest.key}
                                        className="bg-primary-10 rounded-lg p-4 border border-muted hover:border-primary/50 transition-colors"
                                    >
                                        <Link
                                            href={`/contests/${contest.key}`}
                                            className="text-lg font-bold hover:text-primary transition-colors block mb-2"
                                        >
                                            {contest.name}
                                        </Link>
                                        <div className="flex items-center justify-between text-xs text-gray-400">
                                            <span className="flex items-center gap-2">
                                                <Clock size={14} />
                                                Ends in <span className="text-yellow-400 font-bold">{getTimeRemaining(contest.end_time)}</span>
                                            </span>
                                            <span className="flex items-center gap-1">
                                                <Users size={14} />
                                                {contest.user_count} users
                                            </span>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    {/* Upcoming Contests */}
                    {upcomingContests && upcomingContests.length > 0 && (
                        <div className="bg-card border rounded-xl overflow-hidden">
                            <div className="bg-muted px-4 py-3 border-b border-primary-50">
                                <h3 className="text-sm font-bold uppercase tracking-wider text-white flex items-center gap-2">
                                    <Calendar className="text-primary" size={18} />
                                    Upcoming Contests
                                </h3>
                            </div>
                            <div className="p-4 space-y-3">
                                {upcomingContests.map(contest => (
                                    <div
                                        key={contest.key}
                                        className="bg-primary-10 rounded-lg p-4 border border-muted hover:border-primary/50 transition-colors"
                                    >
                                        <Link
                                            href={`/contests/${contest.key}`}
                                            className="text-lg font-bold hover:text-primary transition-colors block mb-2"
                                        >
                                            {contest.name}
                                        </Link>
                                        <div className="flex items-center justify-between text-xs text-gray-400">
                                            <span className="flex items-center gap-2">
                                                <Clock size={14} />
                                                Starts {dayjs(contest.start_time).fromNow()}
                                            </span>
                                            <span className="text-gray-500">
                                                {formatContestTime(contest.start_time, contest.end_time)}
                                            </span>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            </div>

            {/* Sidebar - Original CLAOJ Style */}
            <aside className="w-full lg:w-80 space-y-4 shrink-0">
                {/* Ongoing Contests (Desktop) */}
                {ongoingContests && ongoingContests.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden shadow-lg hidden lg:block">
                        <h3 className="bg-muted px-4 py-3 text-sm font-bold uppercase tracking-wider text-white flex items-center gap-2">
                            <Trophy className="text-yellow-500" size={16} />
                            Ongoing contests
                        </h3>
                        <div className="p-4 space-y-3">
                            {ongoingContests.map(contest => (
                                <div key={contest.key}>
                                    <Link
                                        href={`/contests/${contest.key}`}
                                        className="text-sm font-bold hover:text-primary transition-colors block"
                                    >
                                        {contest.name}
                                    </Link>
                                    <div className="text-xs text-gray-400">
                                        Ends in <span className="text-yellow-400 font-mono">{getTimeRemaining(contest.end_time)}</span>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {/* Upcoming Contests */}
                {upcomingContests && upcomingContests.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden shadow-lg hidden lg:block">
                        <h3 className="bg-muted px-4 py-3 text-sm font-bold uppercase tracking-wider text-white flex items-center gap-2">
                            <Calendar className="text-primary" size={16} />
                            Upcoming contests
                        </h3>
                        <div className="p-4 space-y-3">
                            {upcomingContests.map(contest => (
                                <div key={contest.key}>
                                    <Link
                                        href={`/contests/${contest.key}`}
                                        className="text-sm font-bold hover:text-primary transition-colors block"
                                    >
                                        {contest.name}
                                    </Link>
                                    <div className="text-xs text-gray-400">
                                        Starting {dayjs(contest.start_time).fromNow()}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {/* Top Rating Users */}
                {topRatingUsers && topRatingUsers.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden shadow-lg">
                        <h3 className="bg-muted px-4 py-3 text-sm font-bold uppercase tracking-wider text-white flex items-center gap-2">
                            <Trophy className="text-primary" size={16} />
                            Top rating users
                        </h3>
                        <div className="p-2">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="text-xs text-gray-400 border-b border-muted">
                                        <th className="px-3 py-2 text-left">#</th>
                                        <th className="px-3 py-2 text-left">Username</th>
                                        <th className="px-3 py-2 text-right">Rating</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {topRatingUsers.map((user, i) => (
                                        <tr key={user.id} className="border-b border-muted/50 last:border-0 hover:bg-muted/50">
                                            <td className="px-3 py-2 text-gray-400">{i + 1}</td>
                                            <td className="px-3 py-2">
                                                <Link href={`/user/${user.username}`} className={cn("font-bold hover:text-primary", getRatingClass(user.rating))}>
                                                    {user.username}
                                                </Link>
                                            </td>
                                            <td className="px-3 py-2 text-right text-gray-400 font-mono">{user.rating || 'N/A'}</td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                            <div className="px-3 py-2 text-xs flex justify-between">
                                <Link href="/organizations" className="text-gray-400 hover:text-primary">Organizations</Link>
                                <Link href="/users?order=-rating" className="text-gray-400 hover:text-primary">View all &gt;&gt;&gt;</Link>
                            </div>
                        </div>
                    </div>
                )}

                {/* Top Scorers */}
                {topScorers && topScorers.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden shadow-lg">
                        <h3 className="bg-muted px-4 py-3 text-sm font-bold uppercase tracking-wider text-white flex items-center gap-2">
                            <Star className="text-yellow-500" size={16} />
                            Top scorers
                        </h3>
                        <div className="p-2">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="text-xs text-gray-400 border-b border-muted">
                                        <th className="px-3 py-2 text-left">#</th>
                                        <th className="px-3 py-2 text-left">Username</th>
                                        <th className="px-3 py-2 text-right">Points</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {topScorers.map((user, i) => (
                                        <tr key={user.id} className="border-b border-muted/50 last:border-0 hover:bg-muted/50">
                                            <td className="px-3 py-2 text-gray-400">{i + 1}</td>
                                            <td className="px-3 py-2">
                                                <Link href={`/user/${user.username}`} className="font-bold text-[#72ff72] hover:text-primary">
                                                    {user.username}
                                                </Link>
                                            </td>
                                            <td className="px-3 py-2 text-right text-gray-400 font-mono">
                                                {user.performance_points?.toFixed(2) || '0.00'}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                            <div className="px-3 py-2 text-xs flex justify-between">
                                <Link href="/organizations" className="text-gray-400 hover:text-primary">Organizations</Link>
                                <Link href="/users?order=-performance_points" className="text-gray-400 hover:text-primary">View all &gt;&gt;&gt;</Link>
                            </div>
                        </div>
                    </div>
                )}

                {/* New Problems */}
                {newProblems && newProblems.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden shadow-lg">
                        <h3 className="bg-muted px-4 py-3 text-sm font-bold uppercase tracking-wider text-white flex items-center gap-2">
                            <TrendingUp className="text-primary" size={16} />
                            New problems
                        </h3>
                        <div className="p-4 space-y-2">
                            {newProblems.map(problem => (
                                <div key={problem.code} className="flex items-center justify-between">
                                    <Link
                                        href={`/problems/${problem.code}`}
                                        className="text-sm font-bold hover:text-primary transition-colors truncate flex-grow"
                                    >
                                        {problem.name}
                                    </Link>
                                    <span className="px-2 py-0.5 bg-muted rounded text-xs font-bold text-gray-300 ml-2">
                                        {problem.points}
                                    </span>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {/* Comment Stream */}
                {recentComments && recentComments.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden shadow-lg">
                        <h3 className="bg-muted px-4 py-3 text-sm font-bold uppercase tracking-wider text-white flex items-center gap-2">
                            <MessageSquare className="text-primary" size={16} />
                            <span className="mr-2">Comment stream</span>
                            <a href="https://discord.gg/xdMrcJHxZv" target="_blank" rel="noreferrer">
                                <img
                                    src="https://img.shields.io/discord/1123916507861237881?color=009688&label=Discord&logo=Discord&logoColor=white"
                                    alt="Discord"
                                    className="h-5"
                                />
                            </a>
                        </h3>
                        <div className="p-4">
                            <ul className="space-y-2 text-sm">
                                {recentComments.map((comment: any) => (
                                    <li key={comment.id} className="text-xs">
                                        <Link href={`/user/${comment.author}`} className="text-primary hover:underline font-bold">
                                            {comment.author_name}
                                        </Link>
                                        {' → '}
                                        <Link href={comment.link} className="text-gray-300 hover:text-white truncate block">
                                            {comment.page_title}
                                        </Link>
                                    </li>
                                ))}
                            </ul>
                            <div className="mt-3 pt-3 border-t border-muted text-xs flex gap-2">
                                <a href="/comments/rss" className="text-gray-400 hover:text-primary flex items-center gap-1">
                                    <span className="text-[#ff8f00]"><svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><path d="M6.18 15.64a2.18 2.18 0 0 1 2.18 2.18C8.36 19 7.38 20 6.18 20S4 19 4 17.82a2.18 2.18 0 0 1 2.18-2.18zM4 4.44A15.56 15.56 0 0 1 19.56 20h-2.83A12.73 12.73 0 0 0 4 7.27V4.44zm0 5.66a9.9 9.9 0 0 1 9.9 9.9h-2.83A7.07 7.07 0 0 0 4 12.93V10.1z"/></svg></span>
                                    RSS
                                </a>
                                <span className="text-gray-600">/</span>
                                <a href="/comments/atom" className="text-gray-400 hover:text-[#009688]">Atom</a>
                            </div>
                        </div>
                    </div>
                )}
            </aside>
        </div>
        </div>
    );
}
