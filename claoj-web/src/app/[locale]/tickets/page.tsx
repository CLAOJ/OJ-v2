'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Ticket } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link } from '@/navigation';
import { useState } from 'react';
import {
    Ticket as TicketIcon,
    Plus,
    Search,
    Clock,
    MessageSquare,
    ChevronLeft,
    ChevronRight,
    RefreshCw,
    AlertCircle,
    CheckCircle2
} from 'lucide-react';
import { cn } from '@/lib/utils';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { useAuth } from '@/components/providers/AuthProvider';
import { useRouter } from '@/navigation';

dayjs.extend(relativeTime);

export default function TicketListPage() {
    const t = useTranslations('Tickets');
    const router = useRouter();
    const { user, loading } = useAuth();
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState('');
    const [statusFilter, setStatusFilter] = useState<'all' | 'open' | 'closed'>('all');

    const isAuthenticated = !!user;

    const { data, isLoading } = useQuery({
        queryKey: ['tickets', page, search, statusFilter],
        queryFn: async () => {
            const params = new URLSearchParams({
                page: page.toString(),
                page_size: '50',
                search,
                status: statusFilter !== 'all' ? statusFilter : '',
            });
            const res = await api.get<{ items: Ticket[]; total: number }>(`/tickets?${params.toString()}`);
            return res.data;
        }
    });

    const tickets = data?.items || [];
    const total = data?.total || 0;

    if (!isAuthenticated) {
        router.push('/login');
        return null;
    }

    return (
        <div className="max-w-5xl mx-auto space-y-8 animate-in fade-in duration-700 mt-4 pb-20">
            <div className="flex flex-col md:flex-row justify-between items-end gap-6">
                <header className="space-y-2">
                    <h1 className="text-4xl md:text-5xl font-black tracking-tighter flex items-center gap-4">
                        <TicketIcon className="text-primary" size={48} />
                        {t('title') || 'Support Tickets'}
                    </h1>
                    <p className="text-muted-foreground font-black opacity-80">Get help from the CLAOJ team.</p>
                </header>

                <Link
                    href="/ticket/create"
                    className="px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-colors flex items-center gap-2 shadow-lg shadow-primary/20"
                >
                    <Plus size={18} />
                    New Ticket
                </Link>
            </div>

            {/* Filters */}
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 p-6 rounded-[2.5rem] bg-card border shadow-sm">
                <div className="space-y-2">
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
                            onClick={() => setStatusFilter('all')}
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
                            onClick={() => setStatusFilter('open')}
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
                            onClick={() => setStatusFilter('closed')}
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

                <div className="flex items-end">
                    <button
                        onClick={() => {
                            setSearch('');
                            setStatusFilter('all');
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
                            href={`/ticket/${ticket.id}`}
                            className="group block bg-card border rounded-[2.5rem] p-6 hover:border-primary/30 hover:shadow-lg transition-all"
                        >
                            <div className="flex items-start justify-between gap-4">
                                <div className="flex-1">
                                    <div className="flex items-center gap-3 mb-2">
                                        {ticket.is_closed ? (
                                            <CheckCircle2 size={20} className="text-muted-foreground flex-shrink-0" />
                                        ) : (
                                            <AlertCircle size={20} className="text-amber-500 flex-shrink-0" />
                                        )}
                                        <h3 className="text-xl font-black tracking-tight group-hover:text-primary transition-colors">
                                            {ticket.title}
                                        </h3>
                                    </div>
                                    <div className="flex flex-wrap items-center gap-4 mt-4">
                                        {ticket.problem && (
                                            <Link
                                                href={`/problems/${ticket.problem.code}`}
                                                className="inline-flex items-center gap-2 px-3 py-1.5 rounded-xl bg-muted/50 text-muted-foreground text-xs font-bold hover:bg-primary/10 hover:text-primary transition-colors"
                                            >
                                                <AlertCircle size={14} />
                                                {ticket.problem.code} - {ticket.problem.name}
                                            </Link>
                                        )}
                                        <div className="flex items-center gap-2 text-muted-foreground">
                                            <MessageSquare size={14} />
                                            <span className="text-[10px] font-black uppercase tracking-widest">{ticket.message_count} messages</span>
                                        </div>
                                        <div className="flex items-center gap-2 text-muted-foreground">
                                            <Clock size={14} />
                                            <span className="text-[10px] font-black uppercase tracking-widest">Updated {dayjs(ticket.updated_on).fromNow()}</span>
                                        </div>
                                    </div>
                                </div>
                                <div className="flex-shrink-0">
                                    {ticket.is_closed ? (
                                        <Badge variant="secondary" className="text-[10px] font-black uppercase tracking-widest">
                                            Closed
                                        </Badge>
                                    ) : (
                                        <Badge className="text-[10px] font-black uppercase tracking-widest bg-amber-500/10 text-amber-500 border-amber-500/20">
                                            Open
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
                    <p className="text-muted-foreground mb-6">Create a new ticket to get help from the team.</p>
                    <Link
                        href="/ticket/create"
                        className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-colors"
                    >
                        <Plus size={18} />
                        Create Ticket
                    </Link>
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
