'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { BlogPostDetail } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { use } from 'react';
import {
    Calendar,
    User,
    TrendingUp,
    ArrowLeft,
    Share2
} from 'lucide-react';
import { cn } from '@/lib/utils';
import dayjs from 'dayjs';
import { motion } from 'framer-motion';
import { Link, useRouter } from '@/navigation';
import MathRenderer from '@/components/ui/MathRenderer';
import Comments from '@/components/common/Comments';

export default function BlogDetailPage({ params }: { params: Promise<{ id: string }> }) {
    const { id } = use(params);
    const t = useTranslations('Blog');
    const router = useRouter();

    const { data: post, isLoading } = useQuery({
        queryKey: ['blog', id],
        queryFn: async () => {
            const res = await api.get<BlogPostDetail>(`/blog/${id}`);
            return res.data;
        }
    });

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
                            <TrendingUp size={14} className="text-primary" />
                            {post.score} Points
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

                <section className="prose prose-zinc dark:prose-invert max-w-none">
                    <MathRenderer content={post.content} fullMarkup={true} />
                </section>
            </article>

            <div className="pt-20 border-t">
                <Comments page={`blog/${id}`} />
            </div>
        </div>
    );
}
