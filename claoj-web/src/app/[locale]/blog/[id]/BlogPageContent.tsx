'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { BlogPostDetail } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { use } from 'react';
import {
    Calendar,
    User,
    ArrowLeft,
    Share2,
    ArrowBigUp,
    ArrowBigDown
} from 'lucide-react';
import { cn } from '@/lib/utils';
import dayjs from 'dayjs';
import { motion } from 'framer-motion';
import { Link, useRouter } from '@/navigation';
import MathRenderer from '@/components/ui/MathRenderer';
import Comments from '@/components/common/Comments';
import { toast } from 'sonner';
import { blogVoteApi } from '@/lib/api';

export default function BlogPageContent({ params }: { params: Promise<{ id: string }> }) {
    const { id } = use(params);
    const t = useTranslations('Blog');
    const router = useRouter();
    const queryClient = useQueryClient();

    const { data: post, isLoading } = useQuery({
        queryKey: ['blog', id],
        queryFn: async () => {
            const res = await api.get<BlogPostDetail>(`/blog/${id}`);
            return res.data;
        }
    });

    const voteMutation = useMutation({
        mutationFn: (delta: 1 | -1) => blogVoteApi.vote(Number(id), delta),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['blog', id] });
        },
        onError: (error: any) => {
            toast.error(error.response?.data?.error || 'Failed to vote');
        },
    });

    const handleVote = (delta: 1 | -1) => {
        voteMutation.mutate(delta);
    };

    if (isLoading) return <div className="p-8"><Skeleton className="h-[70vh] w-full rounded-[2.5rem]" /></div>;
    if (!post) return <div className="p-8 text-center">Blog post not found.</div>;

    return (
        <div className="max-w-4xl mx-auto space-y-12 pb-20">
            <Link
                href="/blog"
                className="inline-flex items-center gap-2 text-sm font-black text-muted-foreground hover:text-primary transition-colors group"
            >
                <ArrowLeft size={18} className="group-hover:-translate-x-1 transition-transform" />
                Back to Blog
            </Link>

            <article className="space-y-10">
                <header className="space-y-6">
                    <div className="flex flex-wrap items-center gap-4 text-xs font-black uppercase tracking-widest text-muted-foreground">
                        <span className="flex items-center gap-2">
                            <Calendar size={14} className="text-primary" />
                            {dayjs(post.publish_on).format('DD MMMM YYYY')}
                        </span>
                        <span className="flex items-center gap-2">
                            <div className="flex items-center gap-1">
                                <button
                                    onClick={() => handleVote(1)}
                                    disabled={voteMutation.isPending}
                                    className="p-1.5 rounded-lg hover:bg-emerald-500/10 text-muted-foreground hover:text-emerald-500 transition-colors disabled:opacity-50"
                                    title="Upvote"
                                >
                                    <ArrowBigUp size={20} />
                                </button>
                                <span className={cn(
                                    "font-bold min-w-[3ch] text-center transition-colors",
                                    post.score > 0 ? "text-emerald-500" :
                                    post.score < 0 ? "text-red-500" : "text-muted-foreground"
                                )}>
                                    {post.score > 0 ? '+' : ''}{post.score}
                                </span>
                                <button
                                    onClick={() => handleVote(-1)}
                                    disabled={voteMutation.isPending}
                                    className="p-1.5 rounded-lg hover:bg-red-500/10 text-muted-foreground hover:text-red-500 transition-colors disabled:opacity-50"
                                    title="Downvote"
                                >
                                    <ArrowBigDown size={20} />
                                </button>
                            </div>
                            <span className="text-[10px] uppercase tracking-widest">Points</span>
                        </span>
                    </div>

                    <h1 className="text-4xl md:text-6xl font-black tracking-tighter leading-none">
                        {post.title}
                    </h1>

                    <div className="flex items-center justify-between py-6 border-y">
                        <div className="flex items-center gap-3">
                            <div className="w-10 h-10 rounded-2xl bg-muted flex items-center justify-center text-muted-foreground">
                                <User size={20} />
                            </div>
                            <div className="flex flex-col">
                                <span className="text-xs font-black uppercase text-muted-foreground tracking-tighter">Authors</span>
                                <span className="text-sm font-bold">
                                    {post.authors.map(a => `@${a.username}`).join(', ')}
                                </span>
                            </div>
                        </div>

                        <button className="p-3 rounded-2xl bg-muted/50 border hover:bg-muted transition-colors">
                            <Share2 size={20} />
                        </button>
                    </div>
                </header>

                <section className="prose prose-invert max-w-none text-foreground">
                    <MathRenderer content={post.content} fullMarkup={true} />
                </section>
            </article>

            <div className="pt-20 border-t">
                <Comments page={`blog/${id}`} />
            </div>
        </div>
    );
}
