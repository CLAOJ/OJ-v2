'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Organization } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Link } from '@/navigation';
import { useState } from 'react';
import {
    Users,
    Search,
    Building2,
    ChevronLeft,
    ChevronRight,
    RefreshCw,
    Shield
} from 'lucide-react';
import { cn } from '@/lib/utils';

export default function OrganizationsListPage() {
    const t = useTranslations('Organizations');
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState('');

    const { data, isLoading } = useQuery({
        queryKey: ['organizations', page, search],
        queryFn: async () => {
            const params = new URLSearchParams({
                page: page.toString(),
                page_size: '50',
                search,
            });
            const res = await api.get<{ items: Organization[]; total: number }>(`/organizations?${params.toString()}`);
            return res.data;
        }
    });

    const organizations = data?.items || [];
    const total = data?.total || 0;

    return (
        <div className="max-w-7xl mx-auto space-y-8 animate-in fade-in duration-700 mt-4 pb-20">
            <div className="flex flex-col md:flex-row justify-between items-end gap-6">
                <header className="space-y-2">
                    <h1 className="text-4xl md:text-5xl font-black tracking-tighter flex items-center gap-4">
                        <Building2 className="text-primary" size={48} />
                        {t('title') || 'Organizations'}
                    </h1>
                    <p className="text-muted-foreground font-black opacity-80">Universities, schools, and competitive programming communities.</p>
                </header>

                <div className="flex flex-wrap items-center gap-3 bg-muted/30 p-4 rounded-[2.5rem] border border-dashed">
                    <div className="flex flex-col gap-1">
                        <span className="text-[10px] font-black uppercase text-muted-foreground ml-1">Page</span>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={() => setPage(p => Math.max(1, p - 1))}
                                disabled={page === 1}
                                className="w-10 h-10 rounded-xl bg-card border flex items-center justify-center hover:bg-muted disabled:opacity-20 transition-all"
                            >
                                <ChevronLeft size={18} />
                            </button>
                            <div className="h-10 px-4 rounded-xl bg-primary text-primary-foreground font-black text-xs flex items-center shadow-lg shadow-primary/20">
                                {page}
                            </div>
                            <button
                                onClick={() => setPage(p => p + 1)}
                                disabled={organizations.length < 50}
                                className="w-10 h-10 rounded-xl bg-card border flex items-center justify-center hover:bg-muted disabled:opacity-20 transition-all"
                            >
                                <ChevronRight size={18} />
                            </button>
                        </div>
                    </div>

                    <button
                        onClick={() => {
                            setSearch('');
                            setPage(1);
                        }}
                        className="h-10 px-6 rounded-xl bg-muted/50 hover:bg-muted font-black text-[10px] uppercase tracking-widest flex items-center gap-2 mt-auto"
                    >
                        <RefreshCw size={14} /> Reset
                    </button>
                </div>
            </div>

            {/* Search Bar */}
            <div className="p-6 rounded-[2.5rem] bg-card border shadow-sm">
                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Search</label>
                    <div className="relative">
                        <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/40" size={16} />
                        <input
                            type="text"
                            placeholder="Organization name or short name..."
                            className="w-full h-12 bg-muted/30 border border-transparent rounded-2xl pl-12 pr-4 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none"
                            value={search}
                            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {isLoading ? (
                    Array.from({ length: 6 }).map((_, i) => (
                        <Skeleton key={i} className="h-48 rounded-[2.5rem]" />
                    ))
                ) : (
                    organizations.map((org) => (
                        <Link
                            key={org.id}
                            href={`/organization/${org.id}`}
                            className="group block"
                        >
                            <div className="bg-card border rounded-[2.5rem] p-8 shadow-sm hover:shadow-xl hover:border-primary/30 transition-all duration-300 h-full flex flex-col">
                                <div className="flex items-start justify-between mb-6">
                                    <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-primary/20 to-primary/5 flex items-center justify-center text-primary font-black text-2xl group-hover:scale-110 transition-transform">
                                        {org.name[0]?.toUpperCase()}
                                    </div>
                                    <div className="flex gap-2">
                                        {org.is_open && (
                                            <div className="px-3 py-1.5 rounded-xl bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                                                <span className="text-[10px] font-black uppercase tracking-widest">Open</span>
                                            </div>
                                        )}
                                        {org.is_unlisted && (
                                            <div className="px-3 py-1.5 rounded-xl bg-muted text-muted-foreground border border-transparent">
                                                <span className="text-[10px] font-black uppercase tracking-widest">Unlisted</span>
                                            </div>
                                        )}
                                    </div>
                                </div>

                                <div className="flex-1">
                                    <h3 className="text-xl font-black tracking-tight group-hover:text-primary transition-colors mb-2">
                                        {org.name}
                                    </h3>
                                    {org.short_name && (
                                        <p className="text-[10px] font-mono text-muted-foreground uppercase tracking-widest mb-4">
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
                                        <span className="text-sm font-black">{org.member_count}</span>
                                        <span className="text-[10px] font-bold uppercase tracking-widest">Members</span>
                                    </div>
                                </div>
                            </div>
                        </Link>
                    ))
                )}
            </div>

            {total > 0 && (
                <div className="text-center text-sm text-muted-foreground font-bold">
                    Showing {organizations.length} of {total} organizations
                </div>
            )}

            {!isLoading && organizations.length === 0 && (
                <div className="text-center py-20">
                    <Building2 size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                    <p className="text-muted-foreground font-bold">No organizations found</p>
                </div>
            )}
        </div>
    );
}
