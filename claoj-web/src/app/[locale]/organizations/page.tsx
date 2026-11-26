'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Organization, UserDetail } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import { useState } from 'react';
import {
    Users,
    Search,
    Building2,
    RefreshCw,
    Shield,
    TrendingUp,
    Plus
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { formatPoints } from '@/lib/organizationView';
import { PaginationBar, PAGE_SIZE_OPTIONS } from '@/components/ui/PaginationBar';
import { useAuth } from '@/components/providers/AuthProvider';

export default function OrganizationsListPage() {
    const t = useTranslations('Organizations');
    const tCommon = useTranslations('Common');
    const { user } = useAuth();
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(PAGE_SIZE_OPTIONS[1]);
    const [search, setSearch] = useState('');

    // "Your organizations" (v1 organization/list.html) reuses the public profile
    // payload instead of a dedicated endpoint.
    const { data: profile } = useQuery({
        queryKey: ['user-profile', user?.username],
        queryFn: async () => {
            const res = await api.get<UserDetail>(`/user/${user?.username}`);
            return res.data;
        },
        enabled: !!user?.username,
    });

    const { data, isLoading } = useQuery({
        queryKey: ['organizations', page, pageSize, search],
        queryFn: async () => {
            const params = new URLSearchParams({
                page: page.toString(),
                page_size: pageSize.toString(),
                search,
            });
            const res = await api.get<{ data: Organization[]; total: number }>(`/organizations?${params.toString()}`);
            return res.data;
        }
    });

    const organizations = data?.data || [];

    return (
        <div className="max-w-7xl mx-auto space-y-4 mt-4 pb-20">
            <div className="flex flex-col md:flex-row justify-between items-end gap-6">
                <header className="space-y-2">
                    <h1 className="text-4xl md:text-5xl font-semibold tracking-tighter flex items-center gap-4">
                        <Building2 className="text-primary" size={48} />
                        {t('title') || 'Organizations'}
                    </h1>
                    <p className="text-muted-foreground font-medium opacity-80">{t('subtitle')}</p>
                </header>

                <div className="flex flex-wrap items-center gap-3">
                    {user && (
                        (user.is_superuser === true || (user.permissions ?? []).includes('judge.add_organization')) && (
                            <Link
                                href="/organizations/create"
                                className="h-9 px-6 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 font-medium text-xs flex items-center gap-2"
                            >
                                <Plus size={14} /> {t('createTitle')}
                            </Link>
                        )
                    )}
                    <button
                        onClick={() => {
                            setSearch('');
                            setPage(1);
                        }}
                        className="h-9 px-6 rounded-md bg-muted/30 border hover:bg-muted font-medium text-xs flex items-center gap-2"
                    >
                        <RefreshCw size={14} /> {tCommon('reset')}
                    </button>
                </div>
            </div>

            {/* Search Bar */}
            <div className="p-6 rounded-md bg-card border shadow-sm">
                <div className="space-y-2">
                    <label className="text-xs font-medium text-muted-foreground ml-1">{tCommon('search')}</label>
                    <div className="relative">
                        <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/40" size={16} />
                        <input
                            type="text"
                            placeholder={t('searchPlaceholder')}
                            className="w-full h-9 bg-muted/30 border border-transparent rounded-md pl-12 pr-4 text-sm font-medium focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none"
                            value={search}
                            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>
            </div>

            {/* Your organizations (logged-in shortcut, v1 parity) */}
            {user && profile?.organizations && profile.organizations.length > 0 && (
                <div className="p-6 rounded-md bg-card border shadow-sm space-y-3">
                    <h2 className="text-xs font-medium text-muted-foreground ml-1">{t('yourOrganizations')}</h2>
                    <div className="flex flex-wrap gap-2">
                        {profile.organizations.map(org => (
                            <Link
                                key={org.id}
                                href={`/organization/${org.id}`}
                                className="inline-flex items-center gap-2 px-4 py-2 rounded-md bg-muted/30 border text-sm font-medium hover:border-primary/30 hover:text-primary transition-all"
                            >
                                <Building2 size={14} />
                                {org.name}
                            </Link>
                        ))}
                    </div>
                </div>
            )}

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {isLoading ? (
                    Array.from({ length: 6 }).map((_, i) => (
                        <Skeleton key={i} className="h-48 rounded-md" />
                    ))
                ) : (
                    organizations.map((org) => (
                        <Link
                            key={org.id}
                            href={`/organization/${org.id}-${org.slug}`}
                            className="group block"
                        >
                            <div className="bg-card border rounded-md p-4 shadow-sm hover:border-primary/30 transition-all h-full flex flex-col">
                                <div className="flex items-start justify-between mb-6">
                                    {org.logo_override_image ? (
                                        <img src={org.logo_override_image} alt="" className="w-16 h-16 rounded-md object-cover bg-muted" />
                                    ) : (
                                        <div className="w-16 h-16 rounded-md bg-muted flex items-center justify-center text-primary font-semibold text-2xl">
                                            {org.name[0]?.toUpperCase()}
                                        </div>
                                    )}
                                    <div className="flex gap-2">
                                        {org.is_open && (
                                            <div className="px-3 py-1.5 rounded-xl bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                                                <span className="text-xs font-medium ">{t('open')}</span>
                                            </div>
                                        )}
                                        {org.is_unlisted && (
                                            <div className="px-3 py-1.5 rounded-xl bg-muted text-muted-foreground border border-transparent">
                                                <span className="text-xs font-medium ">{t('unlisted')}</span>
                                            </div>
                                        )}
                                    </div>
                                </div>

                                <div className="flex-1">
                                    <h3 className="text-xl font-semibold tracking-tight group-hover:text-primary transition-colors mb-2">
                                        {org.name}
                                    </h3>
                                    {org.short_name && (
                                        <p className="text-xs font-mono text-muted-foreground mb-4">
                                            {org.short_name}
                                        </p>
                                    )}
                                    {org.about && (
                                        <p className="text-sm text-muted-foreground line-clamp-2">
                                            {org.about}
                                        </p>
                                    )}
                                </div>

                                <div className="flex items-center gap-4 mt-6 pt-6 border-t border-muted/50">
                                    <div className="flex items-center gap-2 text-muted-foreground">
                                        <Users size={16} />
                                        <span className="text-sm font-medium">{org.member_count}</span>
                                        <span className="text-xs font-medium">{t('members')}</span>
                                    </div>
                                    <div className="flex items-center gap-2 text-muted-foreground" title={t('performancePoints')}>
                                        <TrendingUp size={16} />
                                        <span className="text-sm font-medium">{formatPoints(org.performance_points)}</span>
                                        <span className="text-xs font-medium">{t('pp')}</span>
                                    </div>
                                </div>
                            </div>
                        </Link>
                    ))
                )}
            </div>

            {!isLoading && organizations.length === 0 && (
                <div className="text-center py-20">
                    <Building2 size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                    <p className="text-muted-foreground font-medium">{t('noneFound')}</p>
                </div>
            )}

            <PaginationBar
                page={page}
                onPageChange={setPage}
                total={data?.total}
                pageSize={pageSize}
                onPageSizeChange={size => { setPageSize(size); setPage(1); }}
            />
        </div>
    );
}
