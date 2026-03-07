'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Link, useRouter } from '@/navigation';
import { useParams } from 'next/navigation';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import {
    FileText,
    Calendar,
    User,
    TrendingUp,
    ArrowLeft,
    Search
} from 'lucide-react';
import { useState } from 'react';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

interface BlogPost {
    id: number;
    title: string;
    slug: string;
    authors: { username: string }[];
    publish_on: string;
    summary: string;
    score: number;
    sticky: boolean;
}

interface Organization {
    id: number;
    name: string;
    slug: string;
    short_name: string;
}

export default function OrganizationBlogPage() {
    const params = useParams<{ id: string }>();
    const router = useRouter();
    const t = useTranslations('Blog');
    const [search, setSearch] = useState('');

    const { data: org, isLoading: orgLoading } = useQuery({
        queryKey: ['organization', params.id],
        queryFn: async () => {
            const res = await api.get<Organization>(`/organization/${params.id}`);
            return res.data;
        }
    });

    const { data: posts, isLoading: postsLoading } = useQuery({
        queryKey: ['organization-blog', params.id],
        queryFn: async () => {
            const res = await api.get<{
                data: BlogPost[];
                total: number;
            }>(`/blogs?organization=${params.id}`);
            return res.data;
        }
    });

    const filteredPosts = posts?.data?.filter(post =>
        post.title.toLowerCase().includes(search.toLowerCase()) ||
        post.authors.some(a => a.username.toLowerCase().includes(search.toLowerCase()))
    );

    if (orgLoading) {
        return (
            <div className="max-w-4xl mx-auto space-y-8 mt-4 pb-20">
                <Skeleton className="h-64 rounded-[3rem]" />
                <Skeleton className="h-96 rounded-[3rem]" />
            </div>
        );
    }

    if (!org) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <FileText size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                <h2 className="text-2xl font-black mb-2">Organization Not Found</h2>
                <p className="text-muted-foreground mb-6">The organization you&apos;re looking for doesn&apos;t exist or has been removed.</p>
                <Link
                    href="/organizations"
                    className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-colors"
                >
                    <ArrowLeft size={18} />
                    Back to Organizations
                </Link>
            </div>
        );
    }

    return (
        <div className="max-w-4xl mx-auto space-y-8 animate-in fade-in duration-700 mt-4 pb-20">
            {/* Header */}
            <div className="flex items-center gap-4">
                <Link
                    href={`/organization/${params.id}`}
                    className="inline-flex items-center gap-2 text-sm font-bold text-muted-foreground hover:text-primary transition-colors"
                >
                    <ArrowLeft size={16} />
                    Back to Organization
                </Link>
            </div>

            {/* Organization Info */}
            <div className="bg-card border rounded-[3rem] p-8 shadow-sm">
                <div className="flex items-center gap-4 mb-4">
                    <div className="w-16 h-16 rounded-[2rem] bg-gradient-to-br from-primary/20 to-primary/5 flex items-center justify-center text-primary font-black text-3xl">
                        {org.name[0]?.toUpperCase()}
                    </div>
                    <div>
                        <h1 className="text-3xl font-black">{org.name}</h1>
                        {org.short_name && (
                            <p className="text-[10px] font-mono text-muted-foreground uppercase tracking-widest">
                                {org.short_name}
                            </p>
                        )}
                    </div>
                </div>
                <p className="text-muted-foreground font-bold">Blog posts from {org.name}</p>
            </div>

            {/* Search */}
            <div className="relative">
                <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground" size={20} />
                <input
                    type="text"
                    placeholder="Search blog posts..."
                    className="w-full h-14 pl-14 pr-6 rounded-2xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none text-base font-medium"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                />
            </div>

            {/* Blog Posts List */}
            {postsLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map(i => <Skeleton key={i} className="h-48 rounded-[2rem]" />)}
                </div>
            ) : !filteredPosts || filteredPosts.length === 0 ? (
                <div className="bg-card border rounded-[3rem] p-12 text-center">
                    <FileText size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                    <h3 className="text-xl font-black mb-2">No Blog Posts</h3>
                    <p className="text-muted-foreground font-medium">
                        This organization hasn&apos;t published any blog posts yet.
                    </p>
                </div>
            ) : (
                <div className="grid gap-6">
                    {filteredPosts.map((post) => (
                        <Link
                            key={post.id}
                            href={`/blog/${post.id}`}
                            className="group block bg-card border rounded-[2rem] p-6 md:p-8 hover:border-primary/30 hover:shadow-xl transition-all duration-300"
                        >
                            <div className="flex items-start justify-between gap-4 mb-4">
                                <div className="flex-1">
                                    <div className="flex items-center gap-2 mb-3">
                                        {post.sticky && (
                                            <Badge className="text-[10px] font-black uppercase tracking-widest bg-amber-500/10 text-amber-500 border-amber-500/20">
                                                Sticky
                                            </Badge>
                                        )}
                                        <span className="text-xs font-bold text-muted-foreground flex items-center gap-1">
                                            <Calendar size={12} />
                                            {dayjs(post.publish_on).format('MMM D, YYYY')}
                                        </span>
                                    </div>
                                    <h2 className="text-2xl md:text-3xl font-black group-hover:text-primary transition-colors mb-2">
                                        {post.title}
                                    </h2>
                                </div>
                                <div className="flex items-center gap-1 px-4 py-2 rounded-xl bg-muted/50">
                                    <TrendingUp size={16} className={post.score > 0 ? 'text-success' : post.score < 0 ? 'text-destructive' : 'text-muted-foreground'} />
                                    <span className={`font-black ${post.score > 0 ? 'text-success' : post.score < 0 ? 'text-destructive' : 'text-muted-foreground'}`}>
                                        {post.score > 0 ? '+' : ''}{post.score}
                                    </span>
                                </div>
                            </div>

                            <p className="text-muted-foreground font-medium line-clamp-2 mb-4">
                                {post.summary}
                            </p>

                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-2">
                                    <User size={16} className="text-muted-foreground" />
                                    <span className="text-sm font-bold text-muted-foreground">
                                        {post.authors.map(a => a.username).join(', ')}
                                    </span>
                                </div>
                                <span className="text-sm font-bold text-primary opacity-0 group-hover:opacity-100 transition-opacity">
                                    Read More →
                                </span>
                            </div>
                        </Link>
                    ))}
                </div>
            )}
        </div>
    );
}
