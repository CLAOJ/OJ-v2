'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import { useState } from 'react';
import api from '@/lib/api';
import type { BlogPost, PaginatedList } from '@/types';
import { Link } from '@/navigation';
import { ArrowLeft, FileText } from 'lucide-react';
import { Skeleton } from '@/components/ui/Skeleton';
import { PaginationBar } from '@/components/ui/PaginationBar';
import BlogPostCard from '@/components/blog/BlogPostCard';

const PAGE_SIZE = 10;

/** v1 UserBlogPage (`/user/<username>/blog/`): one user's posts. */
export default function UserBlogPage() {
    const { username } = useParams<{ username: string }>();
    const t = useTranslations('Blog');
    const [page, setPage] = useState(1);

    const { data, isLoading } = useQuery({
        queryKey: ['user-blog', username, page],
        queryFn: async () => {
            const res = await api.get<PaginatedList<BlogPost>>('/blogs', {
                params: { author: username, page, page_size: PAGE_SIZE },
            });
            return res.data;
        },
        enabled: !!username,
    });

    const posts = data?.data ?? [];

    return (
        <div className="max-w-4xl mx-auto space-y-8 mt-4 pb-20">
            <Link
                href={`/user/${username}`}
                className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-primary transition-colors"
            >
                <ArrowLeft size={16} />
                {t('backToProfile')}
            </Link>

            <header className="space-y-2">
                <h1 className="text-4xl md:text-5xl font-semibold tracking-tighter flex items-center gap-4">
                    <FileText className="text-primary" size={40} />
                    {t('userBlogTitle')}
                </h1>
                <p className="text-muted-foreground font-medium">{t('postsFrom', { name: username })}</p>
            </header>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-40 rounded-md" />)}
                </div>
            ) : posts.length === 0 ? (
                <div className="bg-card border rounded-md p-12 text-center">
                    <FileText size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                    <h3 className="text-xl font-semibold mb-2">{t('noPostsTitle')}</h3>
                    <p className="text-muted-foreground font-medium">{t('userNoPostsDesc')}</p>
                </div>
            ) : (
                <div className="grid gap-6">
                    {posts.map(post => <BlogPostCard key={post.id} post={post} />)}
                </div>
            )}

            {typeof data?.total === 'number' && data.total > PAGE_SIZE && (
                <PaginationBar page={page} onPageChange={setPage} total={data.total} pageSize={PAGE_SIZE} />
            )}
        </div>
    );
}