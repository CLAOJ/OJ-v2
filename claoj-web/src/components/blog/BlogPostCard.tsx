'use client';

import dayjs from 'dayjs';
import { Calendar, TrendingUp, User } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { Link } from '@/navigation';
import { Badge } from '@/components/ui/Badge';
import type { BlogPost } from '@/types';

/** One blog post row, shared by the organization blog and /user/<u>/blog. */
export default function BlogPostCard({ post }: { post: BlogPost }) {
    const t = useTranslations('Blog');

    return (
        <Link
            href={`/post/${post.id}-${post.slug}`}
            className="group block bg-card border rounded-md p-4 hover:border-primary/30 transition-all"
        >
            <div className="flex items-start justify-between gap-4 mb-4">
                <div className="flex-1">
                    <div className="flex items-center gap-2 mb-3">
                        {post.sticky && (
                            <Badge className="text-xs font-medium bg-amber-500/10 text-amber-500 border-amber-500/20">
                                {t('sticky')}
                            </Badge>
                        )}
                        <span className="text-xs font-medium text-muted-foreground flex items-center gap-1">
                            <Calendar size={12} />
                            {dayjs(post.publish_on).format('ll')}
                        </span>
                    </div>
                    <h2 className="text-2xl md:text-3xl font-semibold group-hover:text-primary transition-colors mb-2">
                        {post.title}
                    </h2>
                </div>
                <div className="flex items-center gap-1 px-4 py-2 rounded-xl bg-muted/50">
                    <TrendingUp size={16} className={post.score > 0 ? 'text-success' : post.score < 0 ? 'text-destructive' : 'text-muted-foreground'} />
                    <span className={`font-medium ${post.score > 0 ? 'text-success' : post.score < 0 ? 'text-destructive' : 'text-muted-foreground'}`}>
                        {post.score > 0 ? '+' : ''}{post.score}
                    </span>
                </div>
            </div>

            {post.summary && (
                <p className="text-muted-foreground font-medium line-clamp-2 mb-4">
                    {post.summary}
                </p>
            )}

            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <User size={16} className="text-muted-foreground" />
                    <span className="text-sm font-medium text-muted-foreground">
                        {post.authors.map(a => a.username).join(', ')}
                    </span>
                </div>
                <span className="text-sm font-medium text-primary opacity-0 group-hover:opacity-100 transition-opacity">
                    {t('readMore')} →
                </span>
            </div>
        </Link>
    );
}