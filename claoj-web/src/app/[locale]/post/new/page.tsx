'use client';

import { useTranslations } from 'next-intl';
import { useAuth } from '@/components/providers/AuthProvider';
import BlogPostForm, { BLOG_MIN_PROBLEM_COUNT } from '@/components/blog/BlogPostForm';
import { Link } from '@/navigation';
import { FilePlus2, Lock } from 'lucide-react';
import { Skeleton } from '@/components/ui/Skeleton';

/** v1 `/post/new` (BlogPostCreate). */
export default function NewBlogPostPage() {
    const t = useTranslations('Blog');
    const { user, loading } = useAuth();

    if (loading) return <div className="max-w-4xl mx-auto p-8"><Skeleton className="h-[60vh] w-full rounded-md" /></div>;

    if (!user) {
        return (
            <div className="max-w-2xl mx-auto py-20 text-center space-y-4">
                <Lock size={48} className="mx-auto text-muted-foreground opacity-30" />
                <p className="font-medium">{t('loginToWrite')}</p>
                <Link href="/login" className="text-primary font-medium hover:underline">{t('signIn')}</Link>
            </div>
        );
    }

    const problemCount = user.profile?.problem_count ?? 0;
    // v1: a personal post needs the problem-count gate; an organization's admin
    // may post as the organization instead (the API decides per post).
    const allowed = user.is_superuser === true
        || problemCount >= BLOG_MIN_PROBLEM_COUNT
        || (user.admin_of?.length ?? 0) > 0;

    return (
        <div className="max-w-4xl mx-auto space-y-8 mt-4 pb-20">
            <header className="space-y-2">
                <h1 className="text-4xl md:text-5xl font-semibold tracking-tighter flex items-center gap-4">
                    <FilePlus2 className="text-primary" size={40} />
                    {t('newPost')}
                </h1>
            </header>

            {allowed ? (
                <div className="bg-card border rounded-md p-6 shadow-sm">
                    <BlogPostForm mode="create" />
                </div>
            ) : (
                <div className="bg-card border rounded-md p-8 text-center space-y-2">
                    <Lock size={48} className="mx-auto text-muted-foreground opacity-30" />
                    <p className="font-medium">{t('gateTitle')}</p>
                    <p className="text-muted-foreground text-sm">
                        {t('gateDesc', { count: BLOG_MIN_PROBLEM_COUNT })}
                    </p>
                </div>
            )}
        </div>
    );
}