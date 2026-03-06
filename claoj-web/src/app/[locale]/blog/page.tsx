'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { BlogPost, PaginatedList, Contest, UserDetail } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link } from '@/navigation';
import {
    Newspaper,
    Calendar,
    User,
    ArrowRight,
    TrendingUp,
    Trophy,
    Clock,
    Flame,
    Hash,
    Rss
} from 'lucide-react';
import { cn } from '@/lib/utils';
import dayjs from 'dayjs';
import { motion } from 'framer-motion';
import { blogFeedApi } from '@/lib/api';

export default function BlogListPage() {
    const t = useTranslations('Blog');

    const { data: blogData, isLoading: isBlogsLoading } = useQuery({
        queryKey: ['blogs'],
        queryFn: async () => {
            const res = await api.get<PaginatedList<BlogPost>>('/blogs');
            return res.data.data;
        }
    });

    const { data: contests, isLoading: isContestsLoading } = useQuery({
        queryKey: ['sidebar-contests'],
        queryFn: async () => {
            const res = await api.get<PaginatedList<Contest>>('/contests?limit=5');
            return res.data.data;
        }
    });

    const { data: topUsers, isLoading: isUsersLoading } = useQuery({
        queryKey: ['sidebar-users'],
        queryFn: async () => {
            const res = await api.get<PaginatedList<UserDetail>>('/users?limit=5');
            return res.data.data;
        }
    });

    return (
        <div className="max-w-7xl mx-auto space-y-12 pb-20 animate-in fade-in duration-700 mt-4">
            <header className="space-y-4">
                <h1 className="text-5xl md:text-6xl font-black tracking-tighter flex items-center gap-6">
                    <Newspaper className="text-primary" size={60} />
                    {t('title')}
                </h1>
                <p className="text-xl text-muted-foreground font-black opacity-80 max-w-2xl">
                    Insights, updates, and community stories from the CLAOJ team.
                </p>
            </header>

            <div className="flex flex-col lg:flex-row gap-12">
                {/* Main Content: Blog Posts */}
                <div className="flex-grow space-y-10 min-w-0">
                    {isBlogsLoading ? (
                        <div className="space-y-8">
                            {[1, 2, 3].map(i => <Skeleton key={i} className="h-64 w-full rounded-[2.5rem]" />)}
                        </div>
                    ) : blogData?.length === 0 ? (
                        <div className="py-24 text-center rounded-[3rem] border border-dashed bg-muted/30 font-black text-muted-foreground flex flex-col items-center gap-4">
                            <Newspaper size={48} className="opacity-20" />
                            {t('noPosts')}
                        </div>
                    ) : (
                        blogData?.map((post, index) => (
                            <motion.article
                                key={post.id}
                                initial={{ opacity: 0, y: 20 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: index * 0.1 }}
                                className="group relative flex flex-col gap-6 p-8 rounded-[2.5rem] bg-card border hover:border-primary/40 hover:shadow-2xl hover:shadow-primary/5 transition-all duration-500"
                            >
                                {post.sticky && (
                                    <div className="absolute -top-3 -right-3 rotate-12 shadow-xl">
                                        <Badge className="bg-primary text-primary-foreground rounded-full px-5 py-2 font-black uppercase tracking-widest text-[10px] shadow-lg shadow-primary/30">
                                            Sticky
                                        </Badge>
                                    </div>
                                )}

                                <div className="space-y-4">
                                    <div className="flex items-center gap-6 text-[10px] font-black text-muted-foreground uppercase tracking-[0.2em]">
                                        <span className="flex items-center gap-2 bg-muted/50 px-3 py-1.5 rounded-full border">
                                            <Calendar size={14} className="text-primary" />
                                            {dayjs(post.publish_on).format('DD MMM YYYY')}
                                        </span>
                                        <span className="flex items-center gap-2 bg-muted/50 px-3 py-1.5 rounded-full border">
                                            <TrendingUp size={14} className="text-emerald-500" />
                                            {post.score} Score
                                        </span>
                                    </div>

                                    <h2 className="text-3xl md:text-4xl font-black tracking-tight group-hover:text-primary transition-colors leading-tight">
                                        {post.title}
                                    </h2>

                                    <p className="text-muted-foreground line-clamp-3 text-lg leading-relaxed font-bold opacity-80">
                                        {post.summary}
                                    </p>

                                    <div className="flex items-center justify-between pt-6 border-t mt-4">
                                        <div className="flex items-center gap-3">
                                            <div className="w-10 h-10 rounded-2xl bg-primary/10 border border-primary/20 flex items-center justify-center text-primary font-black shadow-sm">
                                                {post.authors[0]?.username?.[0]?.toUpperCase()}
                                            </div>
                                            <div className="flex flex-col">
                                                <span className="text-[10px] uppercase font-black text-muted-foreground tracking-widest">Authors</span>
                                                <span className="text-sm font-black">
                                                    {post.authors.map(a => `@${a.username}`).join(', ')}
                                                </span>
                                            </div>
                                        </div>

                                        <Link
                                            href={`/blog/${post.id}`}
                                            className="h-12 flex items-center gap-3 px-8 rounded-2xl bg-primary text-primary-foreground font-black group/btn hover:scale-[1.03] active:scale-95 transition-all shadow-lg shadow-primary/20"
                                        >
                                            {t('readMore')}
                                            <ArrowRight size={20} className="group-hover/btn:translate-x-1.5 transition-transform" />
                                        </Link>
                                    </div>
                                </div>
                            </motion.article>
                        ))
                    )}
                </div>

                {/* Sidebar Widgets */}
                <aside className="w-full lg:w-80 flex flex-col gap-8 shrink-0">
                    {/* RSS/Atom Feed Links */}
                    <div className="p-8 rounded-[2.5rem] border bg-card shadow-sm space-y-6">
                        <h3 className="text-xs font-black uppercase tracking-[0.2em] text-primary flex items-center gap-3">
                            <Rss size={16} />
                            {t('subscribeToFeed')}
                        </h3>
                        <div className="space-y-3">
                            <a
                                href={blogFeedApi.getRssUrl()}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="flex items-center gap-3 p-4 rounded-2xl bg-orange-50 border border-orange-200 hover:bg-orange-100 hover:border-orange-300 transition-all group"
                            >
                                <Rss size={20} className="text-orange-500 group-hover:scale-110 transition-transform" />
                                <span className="text-sm font-black text-orange-700">{t('rssFeed')}</span>
                            </a>
                            <a
                                href={blogFeedApi.getAtomUrl()}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="flex items-center gap-3 p-4 rounded-2xl bg-blue-50 border border-blue-200 hover:bg-blue-100 hover:border-blue-300 transition-all group"
                            >
                                <Rss size={20} className="text-blue-500 group-hover:scale-110 transition-transform" />
                                <span className="text-sm font-black text-blue-700">{t('atomFeed')}</span>
                            </a>
                        </div>
                    </div>

                    {/* Contests Widget */}
                    <div className="p-8 rounded-[2.5rem] border bg-card shadow-sm space-y-8">
                        <h3 className="text-xs font-black uppercase tracking-[0.2em] text-primary flex items-center gap-3">
                            <Clock size={16} />
                            Upcoming Events
                        </h3>
                        <div className="space-y-4">
                            {isContestsLoading ? (
                                [1, 2, 3].map(i => <Skeleton key={i} className="h-14 w-full rounded-2xl" />)
                            ) : (
                                contests?.map(ct => (
                                    <Link
                                        key={ct.key}
                                        href={`/contests/${ct.key}`}
                                        className="block p-4 rounded-3xl bg-muted/30 border border-dashed hover:bg-muted/50 hover:border-primary/30 transition-all group"
                                    >
                                        <div className="flex flex-col gap-1">
                                            <span className="text-[10px] font-black text-muted-foreground uppercase truncate opacity-60">
                                                {dayjs(ct.start_time).format('MMM DD, HH:mm')}
                                            </span>
                                            <span className="text-sm font-black group-hover:text-primary transition-colors truncate">
                                                {ct.name}
                                            </span>
                                        </div>
                                    </Link>
                                ))
                            )}
                        </div>
                    </div>

                    {/* Leaderboard Widget */}
                    <div className="p-8 rounded-[3rem] bg-zinc-900 border border-zinc-800 shadow-2xl space-y-8 overflow-hidden relative">
                        <div className="absolute -top-10 -right-10 opacity-10 pointer-events-none rotate-12">
                            <Trophy size={160} className="text-amber-500" />
                        </div>

                        <h3 className="text-xs font-black uppercase tracking-[0.2em] text-amber-500 flex items-center gap-3 relative z-10">
                            <Flame size={16} />
                            Hall of Fame
                        </h3>

                        <div className="space-y-3 relative z-10">
                            {isUsersLoading ? (
                                [1, 2, 3, 4, 5].map(i => <Skeleton key={i} className="h-12 w-full rounded-2xl bg-zinc-800" />)
                            ) : (
                                topUsers?.map((user, idx) => (
                                    <Link
                                        key={user.username}
                                        href={`/user/${user.username}`}
                                        className="flex items-center justify-between p-3 rounded-2xl bg-zinc-800/50 hover:bg-zinc-800 hover:scale-[1.02] transition-all group"
                                    >
                                        <div className="flex items-center gap-3 min-w-0">
                                            <div className={cn(
                                                "w-8 h-8 rounded-lg flex items-center justify-center font-black text-xs shrink-0 select-none",
                                                idx === 0 ? "bg-amber-500 text-white shadow-[0_0_15px_rgba(245,158,11,0.3)]" :
                                                    idx === 1 ? "bg-zinc-400 text-white" :
                                                        idx === 2 ? "bg-orange-600 text-white" : "bg-zinc-700 text-zinc-400"
                                            )}>
                                                {idx + 1}
                                            </div>
                                            <span className="text-xs font-black text-zinc-100 truncate group-hover:text-amber-400 transition-colors">
                                                {user.username}
                                            </span>
                                        </div>
                                        <span className="text-[10px] font-black font-mono text-zinc-500 shrink-0">
                                            {Math.round(user.performance_points)} PP
                                        </span>
                                    </Link>
                                ))
                            )}
                        </div>

                        <Link
                            href="/users"
                            className="flex items-center justify-center gap-2 pt-4 text-[10px] font-black uppercase tracking-widest text-zinc-500 hover:text-amber-500 transition-colors"
                        >
                            View Global Rankings <ArrowRight size={12} />
                        </Link>
                    </div>
                </aside>
            </div>
        </div>
    );
}
