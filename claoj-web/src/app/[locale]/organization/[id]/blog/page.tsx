'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Link } from '@/navigation';
import { useParams } from 'next/navigation';
import { parseLeadingId } from '@/utils/route';
import { Skeleton } from '@/components/ui/Skeleton';
import BlogPostCard from '@/components/blog/BlogPostCard';
import {
    FileText,
    ArrowLeft,
    Search
} from 'lucide-react';
import { useState } from 'react';
import type { BlogPost } from '@/types';

interface Organization {
    id: number;
    name: string;
    slug: string;
    short_name: string;
}

export default function OrganizationBlogPage() {
    const params = useParams<{ id: string }>();
    const id = parseLeadingId(params.id);
    const t = useTranslations('Blog');
    const tOrg = useTranslations('Organization');
    const [search, setSearch] = useState('');

    const { data: org, isLoading: orgLoading } = useQuery({
        queryKey: ['organization', id],
        queryFn: async () => {
            const res = await api.get<Organization>(`/organization/${id}`);
            return res.data;
        }
    });

    const { data: posts, isLoading: postsLoading } = useQuery({
        queryKey: ['organization-blog', id],
        queryFn: async () => {
            const res = await api.get<{
                data: BlogPost[];
                total: number;
            }>(`/blogs?organization=${id}`);
            return res.data;
        }
    });

    const filteredPosts = posts?.data?.filter(post =>
        post.title.toLowerCase().includes(search.toLowerCase()) ||
        post.authors.some(a => a.username.toLowerCase().includes(search.toLowerCase()))
    );

    if (orgLoading) {
        return (
            <div className="max-w-4xl mx-auto space-y-4 mt-4 pb-20">
                <Skeleton className="h-64 rounded-md" />
                <Skeleton className="h-96 rounded-md" />
            </div>
        );
    }

    if (!org) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <FileText size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                <h2 className="text-2xl font-semibold mb-2">{tOrg('notFound')}</h2>
                <p className="text-muted-foreground mb-6">{tOrg('notFoundDesc')}</p>
                <Link
                    href="/organizations"
                    className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-colors"
                >
                    <ArrowLeft size={18} />
                    {tOrg('backToOrganizations')}
                </Link>
            </div>
        );
    }

    return (
        <div className="max-w-4xl mx-auto space-y-8 mt-4 pb-20">
            {/* Header */}
            <div className="flex items-center gap-4">
                <Link
                    href={`/organization/${org.id}-${org.slug}`}
                    className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-primary transition-colors"
                >
                    <ArrowLeft size={16} />
                    {tOrg('backToOrganization')}
                </Link>
            </div>

            {/* Organization Info */}
            <div className="bg-card border rounded-md p-4 shadow-sm">
                <div className="flex items-center gap-4 mb-4">
                    <div className="w-16 h-16 rounded-md bg-muted flex items-center justify-center text-primary font-semibold text-3xl">
                        {org.name[0]?.toUpperCase()}
                    </div>
                    <div>
                        <h1 className="text-3xl font-semibold">{org.name}</h1>
                        {org.short_name && (
                            <p className="text-xs font-mono text-muted-foreground ">
                                {org.short_name}
                            </p>
                        )}
                    </div>
                </div>
                <p className="text-muted-foreground font-medium">{t('postsFrom', { name: org.name })}</p>
            </div>

            {/* Search */}
            <div className="relative">
                <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground" size={20} />
                <input
                    type="text"
                    placeholder={t('searchPlaceholder')}
                    className="w-full h-10 pl-14 pr-6 rounded-md bg-card border focus:ring-2 focus:ring-primary/20 outline-none text-base font-medium"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                />
            </div>

            {/* Blog Posts List */}
            {postsLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-48 rounded-md" />)}
                </div>
            ) : !filteredPosts || filteredPosts.length === 0 ? (
                <div className="bg-card border rounded-md p-12 text-center">
                    <FileText size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                    <h3 className="text-xl font-semibold mb-2">{t('noPostsTitle')}</h3>
                    <p className="text-muted-foreground font-medium">
                        {t('orgNoPostsDesc')}
                    </p>
                </div>
            ) : (
                <div className="grid gap-6">
                    {filteredPosts.map((post) => (
                        <BlogPostCard key={post.id} post={post} />
                    ))}
                </div>
            )}
        </div>
    );
}
