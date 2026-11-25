'use client';

import { useParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import type { BlogPostDetail } from '@/types';
import BlogPostForm from '@/components/blog/BlogPostForm';
import { Link } from '@/navigation';
import { ArrowLeft, FileEdit, Lock } from 'lucide-react';
import { Skeleton } from '@/components/ui/Skeleton';
import { parseLeadingId } from '@/utils/route';

/** v1 `/post/<id>/edit` (BlogPostEdit). The segment is the same `<id>-<slug>`
 *  as the post page: Next.js forbids two different dynamic names under /post. */
export default function EditBlogPostPage() {
    const { slug } = useParams<{ slug: string }>();
    const id = parseLeadingId(slug);
    const t = useTranslations('Blog');

    const { data: post, isLoading } = useQuery({
        queryKey: ['blog', Number(id)],
        queryFn: async () => (await api.get<BlogPostDetail>(`/blog/${id}`)).data,
    });

    if (isLoading) return <div className="max-w-4xl mx-auto p-8"><Skeleton className="h-[60vh] w-full rounded-md" /></div>;

    if (!post) {
        return <div className="max-w-2xl mx-auto py-20 text-center text-muted-foreground">{t('postNotFound')}</div>;
    }

    return (
        <div className="max-w-4xl mx-auto space-y-8 mt-4 pb-20">
            <Link
                href={`/post/${post.id}-${post.slug}`}
                className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-primary transition-colors"
            >
                <ArrowLeft size={16} />
                {t('backToPost')}
            </Link>

            <header className="space-y-2">
                <h1 className="text-4xl md:text-5xl font-semibold tracking-tighter flex items-center gap-4">
                    <FileEdit className="text-primary" size={40} />
                    {t('editPost')}
                </h1>
            </header>

            {post.can_edit ? (
                <div className="bg-card border rounded-md p-6 shadow-sm">
                    <BlogPostForm
                        mode="edit"
                        postId={post.id}
                        initial={{
                            title: post.title,
                            content: post.content,
                            publish_on: post.publish_on,
                            visible: post.visible,
                            organization_id: post.organization_id ?? null,
                            organization_name: post.organization_name,
                        }}
                    />
                </div>
            ) : (
                <div className="bg-card border rounded-md p-8 text-center space-y-2">
                    <Lock size={48} className="mx-auto text-muted-foreground opacity-30" />
                    <p className="font-medium">{t('editNotAllowed')}</p>
                </div>
            )}
        </div>
    );
}