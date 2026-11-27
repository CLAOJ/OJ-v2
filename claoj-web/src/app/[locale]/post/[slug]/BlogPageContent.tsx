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
    ArrowBigDown,
    FileEdit
} from 'lucide-react';
import { cn } from '@/lib/utils';
import dayjs from 'dayjs';
import { motion } from 'framer-motion';
import { UserLinks } from '@/components/ui/UserLinks';
import { Link, useRouter } from '@/navigation';
import MathRenderer from '@/components/ui/MathRenderer';
import Comments from '@/components/common/Comments';
import { toast } from 'sonner';
import { blogVoteApi } from '@/lib/api';
import { parseLeadingId } from '@/utils/route';
import type { ApiError } from '@/lib/api';

export default function BlogPageContent({ params, initial }: { params: Promise<{ slug: string }>; initial?: BlogPostDetail | null }) {
    const { slug } = use(params);
    const id = parseLeadingId(slug);
    const t = useTranslations('Blog');
    const router = useRouter();
    const queryClient = useQueryClient();

    const { data: post, isLoading } = useQuery({
        queryKey: ['blog', id],
        initialData: initial ?? undefined,
        initialDataUpdatedAt: 0,
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
        onError: (error: ApiError) => {
            toast.error(error.response?.data?.error || t('voteFailed'));
        },
    });

    const handleVote = (delta: 1 | -1) => {
        voteMutation.mutate(delta);
    };

    if (isLoading) return <div className="p-8"><Skeleton className="h-[70vh] w-full rounded-md" /></div>;
    if (!post) return <div className="p-8 text-center">{t('postNotFound')}</div>;

    return (
        <div className="max-w-4xl mx-auto space-y-12 pb-20">
            <div className="flex items-center justify-between">
                <Link
                    href="/post"
                    className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-primary transition-colors group"
                >
                    <ArrowLeft size={18} className=" transition-transform" />
                    {t('backToBlog')}
                </Link>
                {post.can_edit && (
                    <Link
                        href={`/post/${post.id}-${post.slug}/edit`}
                        className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-primary transition-colors"
                    >
                        <FileEdit size={16} />
                        {t('editPost')}
                    </Link>
                )}
            </div>

            <article className="space-y-10">
                <header className="space-y-6">
                    <div className="flex flex-wrap items-center gap-4 text-xs font-medium text-muted-foreground">
                        <span className="flex items-center gap-2">
                            <Calendar size={14} className="text-primary" />
                            {dayjs(post.publish_on).format('LL')}
                        </span>
                        <span className="flex items-center gap-2">
                            <div className="flex items-center gap-1">
                                <button
                                    onClick={() => handleVote(1)}
                                    disabled={voteMutation.isPending}
                                    className="p-1.5 rounded-lg hover:bg-emerald-500/10 text-muted-foreground hover:text-emerald-500 transition-colors disabled:opacity-50"
                                    title={t('upvote')}
                                >
                                    <ArrowBigUp size={20} />
                                </button>
                                <span className={cn(
                                    "font-medium min-w-[3ch] text-center transition-colors",
                                    post.score > 0 ? "text-emerald-500" :
                                    post.score < 0 ? "text-red-500" : "text-muted-foreground"
                                )}>
                                    {post.score > 0 ? '+' : ''}{post.score}
                                </span>
                                <button
                                    onClick={() => handleVote(-1)}
                                    disabled={voteMutation.isPending}
                                    className="p-1.5 rounded-lg hover:bg-red-500/10 text-muted-foreground hover:text-red-500 transition-colors disabled:opacity-50"
                                    title={t('downvote')}
                                >
                                    <ArrowBigDown size={20} />
                                </button>
                            </div>
                            <span className="text-xs ">{t('points')}</span>
                        </span>
                    </div>

                    <h1 className="text-4xl md:text-6xl font-semibold tracking-tighter leading-none">
                        {post.title}
                    </h1>

                    <div className="flex items-center justify-between py-6 border-y">
                        <div className="flex items-center gap-3">
                            {post.authors[0]?.avatar_url ? (
                                <img
                                    src={post.authors[0].avatar_url}
                                    alt=""
                                    className="w-10 h-10 rounded-md bg-muted object-cover shrink-0"
                                />
                            ) : (
                                <div className="w-10 h-10 rounded-md bg-muted flex items-center justify-center text-muted-foreground">
                                    <User size={20} />
                                </div>
                            )}
                            <div className="flex flex-col">
                                <span className="text-xs font-medium text-muted-foreground">{t('authors')}</span>
                                <span className="text-sm font-medium">
                                    <UserLinks users={post.authors} />
                                </span>
                            </div>
                        </div>

                        <button className="p-3 rounded-md bg-muted/50 border hover:bg-muted transition-colors">
                            <Share2 size={20} />
                        </button>
                    </div>
                </header>

                <section className="prose dark:prose-invert max-w-none text-foreground">
                    <MathRenderer content={post.content} />
                </section>
            </article>

            <div className="pt-20 border-t">
                <Comments page={`b:${id}`} />
            </div>
        </div>
    );
}
