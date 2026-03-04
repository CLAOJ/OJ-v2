'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { TicketDetail, TicketMessage } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link, useRouter } from '@/navigation';
import { useParams } from 'next/navigation';
import {
    Ticket as TicketIcon,
    ArrowLeft,
    AlertCircle,
    CheckCircle2,
    MessageSquare,
    Clock,
    User,
    Send,
    Loader2,
    Shield
} from 'lucide-react';
import { cn } from '@/lib/utils';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { useAuth } from '@/components/providers/AuthProvider';
import { motion } from 'framer-motion';

dayjs.extend(relativeTime);

export default function TicketDetailPage() {
    const t = useTranslations('Tickets');
    const router = useRouter();
    const params = useParams();
    const id = params?.id as string;
    const { user, loading } = useAuth();
    const queryClient = useQueryClient();
    const [replyContent, setReplyContent] = useState('');

    const isAuthenticated = !!user;

    const { data: ticket, isLoading } = useQuery({
        queryKey: ['ticket', id],
        queryFn: async () => {
            const res = await api.get<TicketDetail>(`/ticket/${id}`);
            return res.data;
        }
    });

    const replyMutation = useMutation({
        mutationFn: async (body: string) => {
            const res = await api.post(`/ticket/${id}/message`, { body });
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['ticket', id] });
            setReplyContent('');
        }
    });

    if (!isAuthenticated) {
        router.push('/login');
        return null;
    }

    if (isLoading) {
        return (
            <div className="max-w-4xl mx-auto space-y-8 mt-4 pb-20">
                <Skeleton className="h-12 w-48 rounded-xl" />
                <Skeleton className="h-64 rounded-[2.5rem]" />
                <Skeleton className="h-96 rounded-[2.5rem]" />
            </div>
        );
    }

    if (!ticket) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <TicketIcon size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                <h2 className="text-2xl font-black mb-2">Ticket Not Found</h2>
                <p className="text-muted-foreground mb-6">The ticket you&apos;re looking for doesn&apos;t exist or has been removed.</p>
                <Link
                    href="/tickets"
                    className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-colors"
                >
                    <ArrowLeft size={18} />
                    Back to Tickets
                </Link>
            </div>
        );
    }

    const isTicketOwner = ticket.user.username === user?.username;
    const isStaff = user?.is_staff || user?.is_admin;

    const handleReply = () => {
        if (!replyContent.trim()) return;
        replyMutation.mutate(replyContent);
    };

    const MessageBubble = ({ message, isOP }: { message: TicketMessage; isOP: boolean }) => (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className={cn(
                "flex gap-4",
                isOP ? "flex-row" : "flex-row-reverse"
            )}
        >
            <div className={cn(
                "w-12 h-12 rounded-2xl flex items-center justify-center flex-shrink-0 font-black text-lg",
                isOP ? "bg-primary/10 text-primary" : "bg-muted text-muted-foreground"
            )}>
                {message.user.username[0]?.toUpperCase()}
            </div>
            <div className={cn(
                "flex-1 bg-card border rounded-[2rem] p-6",
                isOP ? "" : "bg-muted/30"
            )}>
                <div className="flex items-center gap-3 mb-3">
                    <span className="font-black">{message.user.username}</span>
                    {isOP && (
                        <Badge className="text-[10px] font-black uppercase tracking-widest bg-primary/10 text-primary border-primary/20">
                            <User size={12} className="inline mr-1" /> Ticket Owner
                        </Badge>
                    )}
                    {message.user.is_staff && (
                        <Badge className="text-[10px] font-black uppercase tracking-widest bg-amber-500/10 text-amber-500 border-amber-500/20">
                            <Shield size={12} className="inline mr-1" /> Staff
                        </Badge>
                    )}
                    <span className="text-[10px] text-muted-foreground font-mono ml-auto">
                        {dayjs(message.time).format('MMM D, YYYY [at] HH:mm')}
                    </span>
                </div>
                <div className="prose prose-sm dark:prose-invert max-w-none text-muted-foreground">
                    <p className="whitespace-pre-wrap leading-relaxed">{message.body}</p>
                </div>
            </div>
        </motion.div>
    );

    return (
        <div className="max-w-4xl mx-auto space-y-8 animate-in fade-in duration-700 mt-4 pb-20">
            {/* Back Button */}
            <Link
                href="/tickets"
                className="inline-flex items-center gap-2 text-sm font-bold text-muted-foreground hover:text-primary transition-colors"
            >
                <ArrowLeft size={16} />
                Back to Tickets
            </Link>

            {/* Header */}
            <div className="bg-card border rounded-[3rem] p-8 shadow-sm">
                <div className="flex items-start justify-between gap-4 mb-6">
                    <div className="flex items-center gap-4 flex-1">
                        {ticket.is_closed ? (
                            <CheckCircle2 size={32} className="text-muted-foreground flex-shrink-0" />
                        ) : (
                            <AlertCircle size={32} className="text-amber-500 flex-shrink-0" />
                        )}
                        <div>
                            <h1 className="text-3xl font-black tracking-tight mb-2">{ticket.title}</h1>
                            <div className="flex flex-wrap items-center gap-4">
                                <div className="flex items-center gap-2 text-muted-foreground">
                                    <User size={14} />
                                    <span className="text-[10px] font-black uppercase tracking-widest">
                                        {ticket.user.username}
                                    </span>
                                </div>
                                <div className="flex items-center gap-2 text-muted-foreground">
                                    <Clock size={14} />
                                    <span className="text-[10px] font-black uppercase tracking-widest">
                                        Created {dayjs(ticket.created_on).fromNow()}
                                    </span>
                                </div>
                                <div className="flex items-center gap-2 text-muted-foreground">
                                    <MessageSquare size={14} />
                                    <span className="text-[10px] font-black uppercase tracking-widest">
                                        {ticket.message_count} messages
                                    </span>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div className="flex-shrink-0">
                        {ticket.is_closed ? (
                            <Badge variant="secondary" className="text-[10px] font-black uppercase tracking-widest px-4 py-2">
                                Closed
                            </Badge>
                        ) : (
                            <Badge className="text-[10px] font-black uppercase tracking-widest px-4 py-2 bg-amber-500/10 text-amber-500 border-amber-500/20">
                                Open
                            </Badge>
                        )}
                    </div>
                </div>

                {ticket.problem && (
                    <Link
                        href={`/problems/${ticket.problem.code}`}
                        className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-muted/50 text-muted-foreground text-sm font-bold hover:bg-primary/10 hover:text-primary transition-colors"
                    >
                        <AlertCircle size={16} />
                        {ticket.problem.code} - {ticket.problem.name}
                    </Link>
                )}
            </div>

            {/* Messages */}
            <div className="space-y-6">
                {ticket.messages && ticket.messages.length > 0 ? (
                    ticket.messages.map((message, index) => (
                        <MessageBubble
                            key={message.id}
                            message={message}
                            isOP={message.user.username === ticket.user.username}
                        />
                    ))
                ) : (
                    <div className="text-center py-12 text-muted-foreground border-2 border-dashed rounded-[3rem]">
                        <MessageSquare size={48} className="mx-auto mb-4 opacity-10" />
                        <p className="font-bold">No messages yet</p>
                    </div>
                )}
            </div>

            {/* Reply Form */}
            {!ticket.is_closed && (
                <div className="bg-card border rounded-[3rem] p-8 shadow-sm">
                    <h3 className="text-xl font-black mb-4 flex items-center gap-2">
                        <MessageSquare size={20} className="text-primary" />
                        Reply to Ticket
                    </h3>
                    <div className="space-y-4">
                        <textarea
                            value={replyContent}
                            onChange={(e) => setReplyContent(e.target.value)}
                            placeholder="Write your reply..."
                            rows={6}
                            className="w-full bg-muted/30 border border-muted-foreground/10 rounded-2xl px-4 py-3 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:border-primary/30 transition-all outline-none resize-none"
                        />
                        <div className="flex justify-end">
                            <button
                                onClick={handleReply}
                                disabled={replyMutation.isPending || !replyContent.trim()}
                                className="px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-all flex items-center gap-2 disabled:opacity-50 shadow-lg shadow-primary/20"
                            >
                                {replyMutation.isPending ? (
                                    <Loader2 size={18} className="animate-spin" />
                                ) : (
                                    <Send size={18} />
                                )}
                                Send Reply
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {ticket.is_closed && (
                <div className="text-center py-8 text-muted-foreground border-2 border-dashed rounded-[3rem]">
                    <CheckCircle2 size={32} className="mx-auto mb-2 opacity-20" />
                    <p className="font-bold">This ticket has been closed</p>
                    <p className="text-sm">No further replies can be added</p>
                </div>
            )}
        </div>
    );
}
