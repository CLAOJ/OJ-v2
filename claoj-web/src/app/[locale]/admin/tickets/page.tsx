'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { adminTicketApi } from '@/lib/adminApi';
import { AdminTicket } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link } from '@/navigation';
import { cn } from '@/lib/utils';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import {
    Ticket as TicketIcon,
    Search,
    RefreshCw,
    CheckCircle2,
    AlertCircle,
    UserPlus,
    Star,
    FileText,
    Users,
    Clock
} from 'lucide-react';

dayjs.extend(relativeTime);

export default function AdminTicketListPage() {
    const t = useTranslations('Admin.Tickets');
    const queryClient = useQueryClient();
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState('');
    const [statusFilter, setStatusFilter] = useState<'all' | 'open' | 'closed'>('all');
    const [assignedFilter, setAssignedFilter] = useState<'all' | 'assigned' | 'unassigned'>('all');
    const [contributiveFilter, setContributiveFilter] = useState<'all' | 'contributive' | 'non-contributive'>('all');

    const { data, isLoading } = useQuery<{ data: AdminTicket[]; total: number }>({
        queryKey: ['admin-tickets', page, search, statusFilter, assignedFilter, contributiveFilter],
        queryFn: async () => {
            const filters: Record<string, string> = {};
            if (statusFilter !== 'all') filters.status = statusFilter;
            if (assignedFilter === 'assigned') filters.assigned = 'true';
            if (assignedFilter === 'unassigned') filters.assigned = 'false';
            if (contributiveFilter === 'contributive') filters.is_contributive = 'true';
            if (contributiveFilter === 'non-contributive') filters.is_contributive = 'false';

            const res = await adminTicketApi.list(page, 50, filters as any);
            return res.data;
        }
    });

    const tickets = data?.data || [];
    const total = data?.total || 0;

    return (
        <div className="max-w-7xl mx-auto space-y-8 animate-in fade-in duration-700 mt-4 pb-20">
            <div className="flex justify-between items-center">
                <header className="space-y-2">
                    <h1 className="text-4xl md:text-5xl font-black tracking-tighter flex items-center gap-4">
                        <TicketIcon className="text-primary" size={48} />
                        Admin Tickets
                    </h1>
                    <p className="text-muted-foreground font-black opacity-80">Manage all support tickets.</p>
                </header>
            </div>

            {/* Filters */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4 p-6 rounded-[2.5rem] bg-card border shadow-sm">
                <div className="space-y-2 lg:col-span-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Search</label>
                    <div className="relative">
                        <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground/40" size={16} />
                        <input
                            type="text"
                            placeholder="Search tickets..."
                            className="w-full h-12 bg-muted/30 border border-transparent rounded-2xl pl-12 pr-4 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:bg-background focus:border-muted-foreground/10 transition-all outline-none"
                            value={search}
                            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
                        />
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Status</label>
                    <div className="flex gap-2">
                        <button
                            onClick={() => { setStatusFilter('all'); setPage(1); }}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all",
                                statusFilter === 'all'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            All
                        </button>
                        <button
                            onClick={() => { setStatusFilter('open'); setPage(1); }}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all",
                                statusFilter === 'open'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            Open
                        </button>
                        <button
                            onClick={() => { setStatusFilter('closed'); setPage(1); }}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all",
                                statusFilter === 'closed'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            Closed
                        </button>
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Assignment</label>
                    <div className="flex gap-2">
                        <button
                            onClick={() => { setAssignedFilter('all'); setPage(1); }}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all",
                                assignedFilter === 'all'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            All
                        </button>
                        <button
                            onClick={() => { setAssignedFilter('assigned'); setPage(1); }}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all",
                                assignedFilter === 'assigned'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            Assigned
                        </button>
                        <button
                            onClick={() => { setAssignedFilter('unassigned'); setPage(1); }}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all",
                                assignedFilter === 'unassigned'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            Unassigned
                        </button>
                    </div>
                </div>

                <div className="space-y-2">
                    <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">Contributive</label>
                    <div className="flex gap-2">
                        <button
                            onClick={() => { setContributiveFilter('all'); setPage(1); }}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all",
                                contributiveFilter === 'all'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            All
                        </button>
                        <button
                            onClick={() => { setContributiveFilter('contributive'); setPage(1); }}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all",
                                contributiveFilter === 'contributive'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            Yes
                        </button>
                        <button
                            onClick={() => { setContributiveFilter('non-contributive'); setPage(1); }}
                            className={cn(
                                "flex-1 h-12 rounded-2xl text-[10px] font-black uppercase tracking-widest border transition-all",
                                contributiveFilter === 'non-contributive'
                                    ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/20"
                                    : "bg-muted/30 hover:bg-muted border-transparent"
                            )}
                        >
                            No
                        </button>
                    </div>
                </div>

                <div className="flex items-end">
                    <button
                        onClick={() => {
                            setSearch('');
                            setStatusFilter('all');
                            setAssignedFilter('all');
                            setContributiveFilter('all');
                            setPage(1);
                        }}
                        className="w-full h-12 rounded-2xl bg-muted/50 hover:bg-muted font-black text-[10px] uppercase tracking-widest flex items-center justify-center gap-2 transition-all"
                    >
                        <RefreshCw size={14} /> Reset
                    </button>
                </div>
            </div>

            {/* Ticket List */}
            <div className="space-y-4">
                {isLoading ? (
                    Array.from({ length: 5 }).map((_, i) => (
                        <Skeleton key={i} className="h-32 rounded-[2.5rem]" />
                    ))
                ) : (
                    tickets.map((ticket) => (
                        <Link
                            key={ticket.id}
                            href={`/admin/tickets/${ticket.id}`}
                            className="group block bg-card border rounded-[2.5rem] p-6 hover:border-primary/30 hover:shadow-lg transition-all"
                        >
                            <div className="flex items-start justify-between gap-4">
                                <div className="flex-1">
                                    <div className="flex items-center gap-3 mb-2">
                                        {ticket.is_open ? (
                                            <AlertCircle size={20} className="text-amber-500 flex-shrink-0" />
                                        ) : (
                                            <CheckCircle2 size={20} className="text-muted-foreground flex-shrink-0" />
                                        )}
                                        <h3 className="text-xl font-black tracking-tight group-hover:text-primary transition-colors">
                                            {ticket.title}
                                        </h3>
                                        {ticket.is_contributive && (
                                            <Badge className="text-[10px] font-black uppercase tracking-widest bg-primary/10 text-primary border-primary/20">
                                                <Star size={12} className="inline mr-1" /> Contributive
                                            </Badge>
                                        )}
                                    </div>
                                    <div className="flex flex-wrap items-center gap-4 mt-4">
                                        <div className="flex items-center gap-2 text-muted-foreground">
                                            <UserPlus size={14} />
                                            <span className="text-[10px] font-black uppercase tracking-widest">
                                                {ticket.user}
                                            </span>
                                        </div>
                                        {ticket.assignees && ticket.assignees.length > 0 ? (
                                            <div className="flex items-center gap-2 text-muted-foreground">
                                                <Users size={14} />
                                                <span className="text-[10px] font-black uppercase tracking-widest">
                                                    {ticket.assignees.join(', ')}
                                                </span>
                                            </div>
                                        ) : (
                                            <div className="flex items-center gap-2 text-muted-foreground/50">
                                                <Users size={14} />
                                                <span className="text-[10px] font-black uppercase tracking-widest">
                                                    Unassigned
                                                </span>
                                            </div>
                                        )}
                                        {ticket.notes && (
                                            <div className="flex items-center gap-2 text-muted-foreground">
                                                <FileText size={14} />
                                                <span className="text-[10px] font-black uppercase tracking-widest">
                                                    Has notes
                                                </span>
                                            </div>
                                        )}
                                        <div className="flex items-center gap-2 text-muted-foreground">
                                            <Clock size={14} />
                                            <span className="text-[10px] font-black uppercase tracking-widest">
                                                Created {dayjs(ticket.created).fromNow()}
                                            </span>
                                        </div>
                                    </div>
                                </div>
                                <div className="flex-shrink-0">
                                    {ticket.is_open ? (
                                        <Badge className="text-[10px] font-black uppercase tracking-widest bg-amber-500/10 text-amber-500 border-amber-500/20">
                                            Open
                                        </Badge>
                                    ) : (
                                        <Badge variant="secondary" className="text-[10px] font-black uppercase tracking-widest">
                                            Closed
                                        </Badge>
                                    )}
                                </div>
                            </div>
                        </Link>
                    ))
                )}
            </div>

            {!isLoading && tickets.length === 0 && (
                <div className="text-center py-20 border-2 border-dashed rounded-[3rem]">
                    <TicketIcon size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                    <h3 className="text-xl font-black mb-2">No tickets found</h3>
                    <p className="text-muted-foreground">Try adjusting your filters.</p>
                </div>
            )}

            {/* Pagination */}
            {total > 0 && (
                <div className="flex justify-center gap-4">
                    <button
                        onClick={() => setPage(p => Math.max(1, p - 1))}
                        disabled={page === 1}
                        className="px-6 h-12 rounded-xl bg-card border font-black text-xs uppercase tracking-widest transition-all hover:bg-muted disabled:opacity-30 disabled:pointer-events-none"
                    >
                        Previous
                    </button>
                    <div className="h-12 flex items-center px-6 rounded-xl bg-primary text-primary-foreground font-black text-xs">
                        Page {page}
                    </div>
                    <button
                        onClick={() => setPage(p => p + 1)}
                        disabled={tickets.length < 50}
                        className="px-6 h-12 rounded-xl bg-card border font-black text-xs uppercase tracking-widest transition-all hover:bg-muted disabled:opacity-30 disabled:pointer-events-none"
                    >
                        Next
                    </button>
                </div>
            )}
        </div>
    );
}
