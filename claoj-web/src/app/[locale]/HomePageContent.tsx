'use client';

import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api, { blogVoteApi } from '@/lib/api';
import { useAuth } from '@/components/providers/AuthProvider';
import { toast } from 'sonner';
import { BlogPost, Contest, UserListItem, Problem } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link, useRouter } from '@/navigation';
import { useState, useEffect, useRef, useCallback } from 'react';
import { AdminWelcomeBanner } from '@/components/admin/AdminWelcomeBanner';
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
import { cn, getRankColor } from '@/lib/utils';
import { RatingCircle } from '@/components/ui/RatingCircle';
import { formatCountdown } from '@/lib/countdown';
import { formatDuration } from '@/lib/date';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';

// GET /comments list item (claoj/api/v2/comment.go).
type RecentComment = { id: number; author: string; author_name: string; link: string; page_title: string };

dayjs.extend(relativeTime);

const BLOG_PAGE_SIZE = 5;

export default function HomePageContent() {
    const t = useTranslations('Home');
    const tUsers = useTranslations('Users');
    const tBlog = useTranslations('Blog');
    const tContest = useTranslations('Contest');
    const { user: viewer } = useAuth();
    const queryClient = useQueryClient();
    const voteMutation = useMutation({
        mutationFn: ({ id, delta }: { id: number; delta: 1 | -1 }) => blogVoteApi.vote(id, delta),
        onSuccess: () => queryClient.invalidateQueries({ queryKey: ['blog-posts'] }),
        onError: (error: { response?: { data?: { error?: string } } }) => {
            toast.error(error.response?.data?.error || tBlog('voteFailed'));
        },
    });
    const router = useRouter();
    const [activeTab, setActiveTab] = useState<'blog' | 'events'>('blog');
    const loaderRef = useRef<HTMLDivElement>(null);

    // Fetch blog posts with infinite scroll
    const {
        data: postsData,
        fetchNextPage,
        hasNextPage,
        isFetchingNextPage,
        isLoading: postsLoading
    } = useInfiniteQuery({
        queryKey: ['blog-posts'],
        queryFn: async ({ pageParam = 1 }) => {
            const res = await api.get<{ data: BlogPost[] }>('/blogs', {
                params: { page_size: BLOG_PAGE_SIZE, page: pageParam }
            });
            return { data: res.data.data, page: pageParam as number };
        },
        getNextPageParam: (lastPage, allPages) => {
            if (lastPage.data.length < BLOG_PAGE_SIZE) return undefined;
            return lastPage.page + 1;
        },
        initialPageParam: 1
    });

    // Flatten infinite query data
    const posts = postsData?.pages.flatMap(page => page.data) || [];

    // Fetch ongoing contests (the API has no status filter; it returns
    // { results } sorted by recency — filter by time window client-side)
    const { data: ongoingContests } = useQuery({
        queryKey: ['ongoing-contests'],
        queryFn: async () => {
            const res = await api.get<{ results: Contest[] }>('/contests?page_size=100');
            const now = Date.now();
            return (res.data.results || []).filter(ct =>
                new Date(ct.start_time).getTime() <= now && now <= new Date(ct.end_time).getTime()
            ).slice(0, 5);
        }
    });

    // Fetch upcoming contests
    const { data: upcomingContests } = useQuery({
        queryKey: ['upcoming-contests'],
        queryFn: async () => {
            const res = await api.get<{ results: Contest[] }>('/contests?page_size=100');
            const now = Date.now();
            return (res.data.results || []).filter(ct =>
                new Date(ct.start_time).getTime() > now
            ).slice(0, 5);
        }
    });

    // Fetch top users by rating (limit 10). The API sorts server-side
    // (sort=rating is rating DESC with unrated users last), so no client sort.
    // Unrated users are dropped rather than printed as "N/A": v1's widget only
    // lists users that have a rating.
    const { data: topRatingUsers } = useQuery({
        queryKey: ['top-rating-users'],
        queryFn: async () => {
            const res = await api.get<{ data: UserListItem[] }>('/users?sort=rating&page_size=10');
            return (res.data.data || []).filter(u => u.rating != null);
        }
    });

    // Fetch top scorers (limit 10). The API's default order is performance points DESC.
    const { data: topScorers } = useQuery({
        queryKey: ['top-scorers'],
        queryFn: async () => {
            const res = await api.get<{ data: UserListItem[] }>('/users?page_size=10');
            return res.data.data;
        }
    });

    // Fetch top contributors (limit 10). v1 shows this list under the rating
    // and points widgets; contribution_points DESC is the API's own ordering.
    const { data: topContributors } = useQuery({
        queryKey: ['top-contributors'],
        queryFn: async () => {
            const res = await api.get<{ data: UserListItem[] }>('/users?sort=contribution&page_size=10');
            return (res.data.data || []).filter(u => u.contribution_points > 0);
        }
    });

    // Fetch new problems (limit 10). The API has no date sort; code DESC is the
    // closest available ordering (backend work needed for a true "newest" sort).
    const { data: newProblems } = useQuery({
        queryKey: ['new-problems'],
        queryFn: async () => {
            const res = await api.get<{ data: Problem[] }>('/problems?sort=code&order=desc&page_size=10');
            return res.data.data;
        }
    });

    // Fetch recent comments
    const { data: recentComments } = useQuery({
        queryKey: ['recent-comments'],
        queryFn: async () => {
            const res = await api.get<{ data: RecentComment[] }>('/comments?page_type=blog&page_size=10');
            return res.data.data;
        }
    });

    // Infinite scroll handler
    const handleObserver = useCallback((entries: IntersectionObserverEntry[]) => {
        const target = entries[0];
        if (target.isIntersecting && hasNextPage && !isFetchingNextPage) {
            fetchNextPage();
        }
    }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

    useEffect(() => {
        const observer = new IntersectionObserver(handleObserver, {
            root: null,
            rootMargin: '200px',
            threshold: 0
        });

        if (loaderRef.current) {
            observer.observe(loaderRef.current);
        }

        return () => {
            if (loaderRef.current) {
                observer.unobserve(loaderRef.current);
            }
        };
    }, [handleObserver]);

    const [now, setNow] = useState(dayjs());
    useEffect(() => {
        const timer = setInterval(() => setNow(dayjs()), 1000);
        return () => clearInterval(timer);
    }, []);

    const formatContestTime = (startTime: string, endTime: string) =>
        formatDuration(dayjs(endTime).diff(dayjs(startTime)), 3);

    const getTimeRemaining = (endTime: string) => {
        const diff = dayjs(endTime).diff(now);
        if (diff <= 0) return tContest('ended');

        return formatCountdown(diff);
    };


    return (
        <div className="flex flex-col gap-6">
            {/* The home page opens on the feed; the page still needs a top-level
                heading for screen-reader navigation. */}
            <h1 className="sr-only">CLAOJ: {t('subtitle')}</h1>
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
                                "flex-1 py-3 text-sm font-medium transition-colors flex items-center justify-center gap-2",
                                activeTab === 'blog'
                                    ? "bg-primary text-white"
                                    : "text-muted-foreground hover:bg-white/5"
                            )}
                        >
                            <BookOpen size={16} />
                            {t('blogTab')}
                        </button>
                        <button
                            onClick={() => setActiveTab('events')}
                            className={cn(
                                "flex-1 py-3 text-sm font-medium transition-colors flex items-center justify-center gap-2",
                                activeTab === 'events'
                                    ? "bg-primary text-white"
                                    : "text-muted-foreground hover:bg-white/5"
                            )}
                        >
                            <Flame size={16} />
                            {t('eventsTab')}
                        </button>
                    </div>
                </div>

                {/* Blog Posts */}
                <div className={cn("space-y-4", activeTab !== 'blog' && "hidden md:block")}>
                    <div className="flex items-center justify-between mb-4">
                        <h2 className="text-xl font-semibold tracking-tight flex items-center gap-2">
                            <BookOpen className="text-muted-foreground" size={24} />
                            {t('latestPosts')}
                        </h2>
                        <Link href="/post" className="text-sm font-medium text-primary hover:underline flex items-center gap-1">
                            {t('viewAll')} <ChevronRight size={16} />
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
                        <>
                            {posts.map((post) => (
                                <article
                                    key={post.id}
                                    className={cn(
                                        "bg-card border rounded-lg overflow-hidden transition-all",
                                        post.sticky && "border-primary border-2"
                                    )}
                                >
                                    <div className="p-4 sm:p-5">
                                        <div className="flex gap-3">
                                            {/* Vote Section */}
                                            <div className="flex flex-col items-center gap-0.5 pt-0.5">
                                                <button
                                                    type="button"
                                                    onClick={() => viewer ? voteMutation.mutate({ id: post.id, delta: 1 }) : router.push('/login')}
                                                    disabled={voteMutation.isPending}
                                                    aria-label={tBlog('upvote')}
                                                    title={tBlog('upvote')}
                                                    className="p-1 rounded text-muted-foreground hover:text-primary hover:bg-muted transition-colors disabled:opacity-50"
                                                >
                                                    <ThumbsUp size={18} />
                                                </button>
                                                <span className="text-sm font-medium text-muted-foreground">{post.score || 0}</span>
                                                <button
                                                    type="button"
                                                    onClick={() => viewer ? voteMutation.mutate({ id: post.id, delta: -1 }) : router.push('/login')}
                                                    disabled={voteMutation.isPending}
                                                    aria-label={tBlog('downvote')}
                                                    title={tBlog('downvote')}
                                                    className="p-1 rounded text-muted-foreground hover:text-destructive hover:bg-muted transition-colors disabled:opacity-50"
                                                >
                                                    <ThumbsDown size={18} />
                                                </button>
                                            </div>

                                            {/* Content Section */}
                                            <div className="flex-grow min-w-0">
                                                <div className="flex items-center gap-2 mb-2">
                                                    <h3 className="text-xl font-semibold hover:text-primary transition-colors">
                                                        <Link href={`/post/${post.id}-${post.slug}`} className="hover:underline">
                                                            {post.visible === false && <Lock size={14} className="inline text-red-500 mr-1" />}
                                                            {post.title}
                                                        </Link>
                                                    </h3>
                                                    {post.sticky && <Star size={16} className="text-yellow-500 fill-yellow-500" />}
                                                </div>

                                                <div className="flex items-center gap-4 text-xs text-muted-foreground mb-3 flex-wrap">
                                                    <span className="flex items-center gap-1">
                                                        <Users size={12} />
                                                        {post.authors?.map(a => a.username).join(', ')}
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

                                                {(post.summary || post.content) && (
                                                    <div className="prose dark:prose-invert max-w-none text-sm text-muted-foreground">
                                                        <ReactMarkdown
                                                            remarkPlugins={[remarkGfm]}
                                                            rehypePlugins={[rehypeRaw]}
                                                        >
                                                            {post.summary || post.content}
                                                        </ReactMarkdown>
                                                    </div>
                                                )}

                                                {post.summary && (
                                                    <div className="mt-3">
                                                        <Link
                                                            href={`/post/${post.id}-${post.slug}`}
                                                            className="text-sm font-medium text-primary hover:underline"
                                                        >
                                                            {t('continueReading')}
                                                        </Link>
                                                    </div>
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                </article>
                            ))}

                            {/* Infinite Scroll Loader */}
                            <div ref={loaderRef} className="py-8 text-center">
                                {isFetchingNextPage && (
                                    <div className="flex items-center justify-center gap-2 text-muted-foreground">
                                        <div className="w-2 h-2 rounded-full bg-primary motion-safe:animate-pulse" style={{ animationDelay: '0ms' }} />
                                        <div className="w-2 h-2 rounded-full bg-primary motion-safe:animate-pulse" style={{ animationDelay: '150ms' }} />
                                        <div className="w-2 h-2 rounded-full bg-primary motion-safe:animate-pulse" style={{ animationDelay: '300ms' }} />
                                        <span className="text-sm font-medium">{t('loadingMorePosts')}</span>
                                    </div>
                                )}
                                {!hasNextPage && posts.length > 0 && (
                                    <div className="text-sm text-gray-500">
                                        {t('noMorePosts')}
                                    </div>
                                )}
                            </div>
                        </>
                    ) : (
                        <div className="bg-card border border-dashed rounded-lg p-4 text-center text-muted-foreground">
                            {t('noPosts')}
                        </div>
                    )}
                </div>

                {/* Events Tab (for mobile) */}
                <div className={cn("space-y-6", activeTab !== 'events' && "hidden md:block")}>
                    {/* Upcoming Contests - Only show, no ongoing in main content */}
                    {upcomingContests && upcomingContests.length > 0 && (
                        <div className="bg-card border rounded-xl overflow-hidden">
                            <div className="bg-muted px-4 py-3 border-b border-primary-50">
                                <h3 className="text-sm font-semibold text-foreground flex items-center gap-2">
                                    <Calendar className="text-muted-foreground" size={18} />
                                    {t('upcomingContests')}
                                </h3>
                            </div>
                            <div className="p-4 space-y-3">
                                {upcomingContests.map(contest => (
                                    <div
                                        key={contest.key}
                                        className="bg-primary-10 rounded-lg p-4 border border-muted hover:border-primary/50 transition-colors"
                                    >
                                        <Link
                                            href={`/contest/${contest.key}`}
                                            className="text-lg font-semibold hover:text-primary transition-colors block mb-2"
                                        >
                                            {contest.name}
                                        </Link>
                                        <div className="flex items-center justify-between text-xs text-muted-foreground">
                                            <span className="flex items-center gap-2">
                                                <Clock size={14} />
                                                {t('startsRelative', { time: dayjs(contest.start_time).fromNow() })}
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
                {/* Ongoing Contests (Desktop - Right Sidebar Only) */}
                {ongoingContests && ongoingContests.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden">
                        <h3 className="bg-muted px-4 py-3 text-sm font-semibold text-foreground flex items-center gap-2">
                            <Trophy className="text-muted-foreground" size={16} />
                            {t('ongoingContests')}
                        </h3>
                        <div className="p-4 space-y-3">
                            {ongoingContests.map(contest => (
                                <div key={contest.key}>
                                    <Link
                                        href={`/contest/${contest.key}`}
                                        className="text-sm font-medium hover:text-primary transition-colors block"
                                    >
                                        {contest.name}
                                    </Link>
                                    <div className="text-xs text-muted-foreground">
                                        {t.rich('endsInCountdown', {
                                            countdown: getTimeRemaining(contest.end_time),
                                            highlight: (chunks) => <span className="text-yellow-400 font-mono">{chunks}</span>
                                        })}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {/* Upcoming Contests */}
                {upcomingContests && upcomingContests.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden">
                        <h3 className="bg-muted px-4 py-3 text-sm font-semibold text-foreground flex items-center gap-2">
                            <Calendar className="text-muted-foreground" size={16} />
                            {t('upcomingContests')}
                        </h3>
                        <div className="p-4 space-y-3">
                            {upcomingContests.map(contest => (
                                <div key={contest.key}>
                                    <Link
                                        href={`/contest/${contest.key}`}
                                        className="text-sm font-medium hover:text-primary transition-colors block"
                                    >
                                        {contest.name}
                                    </Link>
                                    <div className="text-xs text-muted-foreground">
                                        {t('startingRelative', { time: dayjs(contest.start_time).fromNow() })}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {/* Top Rating Users (limit 10) */}
                {topRatingUsers && topRatingUsers.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden">
                        <h3 className="bg-muted px-4 py-3 text-sm font-semibold text-foreground flex items-center gap-2">
                            <Trophy className="text-muted-foreground" size={16} />
                            {t('topRatingUsers')}
                        </h3>
                        <div className="p-2">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="text-xs text-muted-foreground border-b border-muted">
                                        <th className="px-3 py-2 text-left">#</th>
                                        <th className="px-3 py-2 text-left">{tUsers('username')}</th>
                                        <th className="px-3 py-2 text-right">{tUsers('rating')}</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {topRatingUsers.map((user, i) => (
                                        <tr key={user.username} className="border-b border-muted/50 last:border-0 hover:bg-muted/50">
                                            <td className="px-3 py-2 text-muted-foreground">{i + 1}</td>
                                            <td className="px-3 py-2">
                                                <Link href={`/user/${user.username}`} className={cn("font-medium hover:text-primary", getRankColor(user.rating))}>
                                                    {user.username}
                                                </Link>
                                            </td>
                                            <td className={cn("px-3 py-2 text-right font-semibold tabular-nums whitespace-nowrap", getRankColor(user.rating))}>
                                                <RatingCircle rating={user.rating} className="mr-1" />{user.rating}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                            <div className="px-3 py-2 text-xs flex justify-between">
                                <Link href="/organizations" className="text-muted-foreground hover:text-primary">{t('organizations')}</Link>
                                <Link href="/users?sort=rating" className="text-muted-foreground hover:text-primary">{t('viewAllArrow')}</Link>
                            </div>
                        </div>
                    </div>
                )}

                {/* Top Scorers (limit 10) */}
                {topScorers && topScorers.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden">
                        <h3 className="bg-muted px-4 py-3 text-sm font-semibold text-foreground flex items-center gap-2">
                            <Star className="text-muted-foreground" size={16} />
                            {t('topScorers')}
                        </h3>
                        <div className="p-2">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="text-xs text-muted-foreground border-b border-muted">
                                        <th className="px-3 py-2 text-left">#</th>
                                        <th className="px-3 py-2 text-left">{tUsers('username')}</th>
                                        <th className="px-3 py-2 text-right">{tUsers('points')}</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {topScorers.map((user, i) => (
                                        <tr key={user.username} className="border-b border-muted/50 last:border-0 hover:bg-muted/50">
                                            <td className="px-3 py-2 text-muted-foreground">{i + 1}</td>
                                            <td className="px-3 py-2">
                                                <Link href={`/user/${user.username}`} className={cn("font-medium hover:text-primary", getRankColor(user.rating))}>
                                                    {user.username}
                                                </Link>
                                            </td>
                                            <td className="px-3 py-2 text-right text-muted-foreground font-mono">
                                                {user.performance_points?.toFixed(2) || '0.00'}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                            <div className="px-3 py-2 text-xs flex justify-between">
                                <Link href="/organizations" className="text-muted-foreground hover:text-primary">{t('organizations')}</Link>
                                <Link href="/users?sort=points" className="text-muted-foreground hover:text-primary">{t('viewAllArrow')}</Link>
                            </div>
                        </div>
                    </div>
                )}

                {/* Top Contributors (limit 10) */}
                {topContributors && topContributors.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden">
                        <h3 className="bg-muted px-4 py-3 text-sm font-semibold text-foreground flex items-center gap-2">
                            <ThumbsUp className="text-muted-foreground" size={16} />
                            {t('topContributors')}
                        </h3>
                        <div className="p-2">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="text-xs text-muted-foreground border-b border-muted">
                                        <th className="px-3 py-2 text-left">#</th>
                                        <th className="px-3 py-2 text-left">{tUsers('username')}</th>
                                        <th className="px-3 py-2 text-right">{tUsers('contribution')}</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {topContributors.map((user, i) => (
                                        <tr key={user.username} className="border-b border-muted/50 last:border-0 hover:bg-muted/50">
                                            <td className="px-3 py-2 text-muted-foreground">{i + 1}</td>
                                            <td className="px-3 py-2">
                                                <Link href={`/user/${user.username}`} className="font-medium hover:text-primary">
                                                    {user.display_name || user.username}
                                                </Link>
                                            </td>
                                            <td className="px-3 py-2 text-right text-muted-foreground font-mono">
                                                {user.contribution_points}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                            <div className="px-3 py-2 text-xs flex justify-between">
                                <Link href="/organizations" className="text-muted-foreground hover:text-primary">{t('organizations')}</Link>
                                <Link href="/users?sort=contribution" className="text-muted-foreground hover:text-primary">{t('viewAllArrow')}</Link>
                            </div>
                        </div>
                    </div>
                )}

                {/* New Problems (limit 10) */}
                {newProblems && newProblems.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden">
                        <h3 className="bg-muted px-4 py-3 text-sm font-semibold text-foreground flex items-center gap-2">
                            <TrendingUp className="text-muted-foreground" size={16} />
                            {t('newProblems')}
                        </h3>
                        <div className="p-4 space-y-2">
                            {newProblems.map(problem => (
                                <div key={problem.code} className="flex items-center justify-between">
                                    <Link
                                        href={`/problem/${problem.code}`}
                                        className="text-sm font-medium hover:text-primary transition-colors truncate flex-grow"
                                    >
                                        {problem.name}
                                    </Link>
                                    <span className="px-2 py-0.5 bg-muted rounded text-xs font-medium text-muted-foreground ml-2">
                                        {problem.points}
                                    </span>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {/* Comment Stream */}
                {recentComments && recentComments.length > 0 && (
                    <div className="bg-primary-10 rounded-lg overflow-hidden">
                        <h3 className="bg-muted px-4 py-3 text-sm font-semibold text-foreground flex items-center gap-2">
                            <MessageSquare className="text-muted-foreground" size={16} />
                            <span className="mr-2">{t('commentStream')}</span>
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
                                {recentComments.map(comment => (
                                    <li key={comment.id} className="text-xs">
                                        <Link href={`/user/${comment.author}`} className="text-primary hover:underline font-medium">
                                            {comment.author_name}
                                        </Link>
                                        {' → '}
                                        <Link href={comment.link} className="text-foreground hover:text-accent truncate block">
                                            {comment.page_title}
                                        </Link>
                                    </li>
                                ))}
                            </ul>
                            <div className="mt-3 pt-3 border-t border-muted text-xs flex gap-2">
                                {/* Feeds are API responses, not pages: a full navigation is intended. */}
                                {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
                                <a href="/api/comments/feed/rss" className="text-muted-foreground hover:text-primary flex items-center gap-1">
                                    <span className="text-muted-foreground"><svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><path d="M6.18 15.64a2.18 2.18 0 0 1 2.18 2.18C8.36 19 7.38 20 6.18 20S4 19 4 17.82a2.18 2.18 0 0 1 2.18-2.18zM4 4.44A15.56 15.56 0 0 1 19.56 20h-2.83A12.73 12.73 0 0 0 4 7.27V4.44zm0 5.66a9.9 9.9 0 0 1 9.9 9.9h-2.83A7.07 7.07 0 0 0 4 12.93V10.1z"/></svg></span>
                                    RSS
                                </a>
                                <span className="text-muted-foreground" aria-hidden="true">/</span>
                                {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
                                <a href="/api/comments/feed/atom" className="text-muted-foreground hover:text-primary">Atom</a>
                            </div>
                        </div>
                    </div>
                )}
            </aside>
        </div>
        </div>
    );
}
