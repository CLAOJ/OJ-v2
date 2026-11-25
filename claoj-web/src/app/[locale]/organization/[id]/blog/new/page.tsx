'use client';

import { useParams, useRouter } from 'next/navigation';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { isAxiosError } from 'axios';
import { ArrowLeft, Loader2, PenLine, Shield } from 'lucide-react';
import { toast } from 'sonner';
import api from '@/lib/api';
import { Link } from '@/navigation';
import BlogPostForm, { type BlogPostFormData } from '@/components/admin/BlogPostForm';
import { fromLocalInput, toLocalInput } from '@/lib/adminForms';
import { parseLeadingId } from '@/utils/route';
import { useAuth } from '@/components/providers/AuthProvider';
import { hasPerm } from '@/lib/permissions';
import type { OrganizationDetail } from '@/types';

/** v1 BlogPostCreateOrganization: org admins post on the organization blog. */
export default function OrganizationBlogCreatePage() {
    const { id: idParam } = useParams<{ id: string }>();
    const id = parseLeadingId(idParam);
    const t = useTranslations('Organization');
    const tCommon = useTranslations('Common');
    const router = useRouter();
    const { user } = useAuth();

    const { data: org, isLoading } = useQuery({
        queryKey: ['organization', id],
        queryFn: async () => {
            const res = await api.get<OrganizationDetail>(`/organization/${id}`);
            return res.data;
        }
    });

    const createMutation = useMutation({
        mutationFn: async (data: BlogPostFormData) => {
            const res = await api.post(`/organization/${id}/blog-posts`, {
                title: data.title,
                slug: data.slug,
                content: data.content,
                summary: data.summary,
                author_ids: org ? [org.user_id] : [],
                publish_on: data.publish_on ? fromLocalInput(data.publish_on) : new Date().toISOString(),
                visible: data.visible,
            });
            return res.data as { blog_post: { id: number; slug: string } };
        },
        onSuccess: (data) => {
            toast.success(t('created'));
            router.push(`/post/${data.blog_post.slug}`);
        },
        onError: (error) => {
            toast.error(isAxiosError(error) ? error.response?.data?.error : tCommon('error'));
        }
    });

    const allowed = hasPerm(user, 'judge.edit_organization_post')
        && (!!org?.viewer_is_admin || hasPerm(user, 'judge.edit_all_post'));

    if (isLoading) {
        return (
            <div className="flex items-center justify-center py-20">
                <Loader2 size={32} className="animate-spin text-muted-foreground" />
            </div>
        );
    }

    if (!org) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <h2 className="text-2xl font-semibold mb-2">{t('notFound')}</h2>
                <Link href="/organizations" className="text-primary hover:underline">{t('backToOrganizations')}</Link>
            </div>
        );
    }

    if (!allowed) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <Shield size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                <h2 className="text-2xl font-semibold mb-2">{t('accessDenied')}</h2>
                <p className="text-muted-foreground">{t('createDenied')}</p>
            </div>
        );
    }

    const slugPrefix = org.slug.toLowerCase().replace(/[^a-z]/g, '') || 'post';
    const now = toLocalInput(new Date().toISOString());

    return (
        <div className="max-w-5xl mx-auto space-y-6 py-8">
            <div className="flex items-center gap-4">
                <Link
                    href={`/organization/${org.id}-${org.slug}`}
                    className="p-2 hover:bg-muted rounded-xl transition-colors"
                >
                    <ArrowLeft size={20} />
                </Link>
                <div>
                    <h1 className="text-3xl font-semibold flex items-center gap-3">
                        <PenLine className="text-primary" size={32} />
                        {t('newBlogPost')}
                    </h1>
                    <p className="text-muted-foreground mt-1">{org.name}</p>
                </div>
            </div>

            <BlogPostForm
                initialData={{ slug: slugPrefix, publish_on: now, visible: true }}
                onSubmit={async data => { await createMutation.mutateAsync(data); }}
                isLoading={createMutation.isPending}
            />
        </div>
    );
}