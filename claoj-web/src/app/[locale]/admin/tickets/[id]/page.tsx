'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { adminTicketApi, adminUserApi } from '@/lib/adminApi';
import { AdminTicketDetail, AdminUser } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link, useRouter, usePathname } from '@/navigation';
import { useParams } from 'next/navigation';
import { cn } from '@/lib/utils';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
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
    Shield,
    Star,
    UserPlus,
    FileText,
    ToggleLeft,
    Users
} from 'lucide-react';
import { motion } from 'framer-motion';

dayjs.extend(relativeTime);

export default function AdminTicketDetailPage() {
    const t = useTranslations('Admin.Tickets');
    const router = useRouter();
    const params = useParams();
    const id = params?.id as string;
    const queryClient = useQueryClient();
    const [replyContent, setReplyContent] = useState('');
    const [notesContent, setNotesContent] = useState('');
    const [showAssignModal, setShowAssignModal] = useState(false);
    const [selectedAssignees, setSelectedAssignees] = useState<number[]>([]);
    const [userSearch, setUserSearch] = useState('');

    const { data: ticket, isLoading } = useQuery({
        queryKey: ['admin-ticket', id],
        queryFn: async () => {
            const res = await adminTicketApi.detail(parseInt(id));
            setNotesContent(res.data.notes || '');
            return res.data;
        }
    });

    const { data: users } = useQuery({
        queryKey: ['admin-users-search', userSearch],
        queryFn: async () => {
            if (!userSearch) return { data: [], total: 0 };
            const res = await adminUserApi.list(1, 20);
            return res.data;
        },
        enabled: showAssignModal
    });

    const replyMutation = useMutation({
        mutationFn: async (body: string) => {
            const res = await fetch(`/api/ticket/${id}/message`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ body })
            });
            return res.json();
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-ticket', id] });
            setReplyContent('');
        }
    });

    const toggleOpenMutation = useMutation({
        mutationFn: async () => {
            const res = await adminTicketApi.toggleOpen(parseInt(id));
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-ticket', id] });
        }
    });

    const setContributiveMutation = useMutation({
        mutationFn: async (isContributive: boolean) => {
            const res = await adminTicketApi.setContributive(parseInt(id), isContributive);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-ticket', id] });
        }
    });

    const updateNotesMutation = useMutation({
        mutationFn: async (notes: string) => {
            const res = await adminTicketApi.updateNotes(parseInt(id), notes);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-ticket', id] });
        }
    });

    const assignMutation = useMutation({
        mutationFn: async (profileIds: number[]) => {
            const res = await adminTicketApi.assign(parseInt(id), profileIds);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-ticket', id] });
            setShowAssignModal(false);
            setSelectedAssignees([]);
        }
    });

    if (isLoading) {
        return (
            <div className="max-w-6xl mx-auto space-y-8 mt-4 pb-20">
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
                    href="/admin/tickets"
                    className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-colors"
                >
                    <ArrowLeft size={18} />
                    Back to Tickets
                </Link>
            </div>
        );
    }

    const handleReply = () => {
        if (!replyContent.trim()) return;
        replyMutation.mutate(replyContent);
    };

    const handleNotesBlur = () => {
        if (notesContent !== (ticket.notes || '')) {
            updateNotesMutation.mutate(notesContent);
        }
    };

    const handleAssign = () => {
        assignMutation.mutate(selectedAssignees);
    };

    const MessageBubble = ({ message, isOP }: { message: any; isOP: boolean }) => (
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
        <div className="max-w-6xl mx-auto space-y-8 animate-in fade-in duration-700 mt-4 pb-20">
            {/* Back Button */}
            <Link
                href="/admin/tickets"
                className="inline-flex items-center gap-2 text-sm font-bold text-muted-foreground hover:text-primary transition-colors"
            >
                <ArrowLeft size={16} />
                Back to Admin Tickets
            </Link>

            {/* Header */}
            <div className="bg-card border rounded-[3rem] p-8 shadow-sm">
                <div className="flex items-start justify-between gap-4 mb-6">
                    <div className="flex items-center gap-4 flex-1">
                        {ticket.is_open ? (
                            <AlertCircle size={32} className="text-amber-500 flex-shrink-0" />
                        ) : (
                            <CheckCircle2 size={32} className="text-muted-foreground flex-shrink-0" />
                        )}
                        <div>
                            <h1 className="text-3xl font-black tracking-tight mb-2">{ticket.title}</h1>
                            <div className="flex flex-wrap items-center gap-4">
                                <div className="flex items-center gap-2 text-muted-foreground">
                                    <User size={14} />
                                    <span className="text-[10px] font-black uppercase tracking-widest">
                                        {ticket.creator}
                                    </span>
                                </div>
                                <div className="flex items-center gap-2 text-muted-foreground">
                                    <Clock size={14} />
                                    <span className="text-[10px] font-black uppercase tracking-widest">
                                        Created {dayjs(ticket.created).fromNow()}
                                    </span>
                                </div>
                                <div className="flex items-center gap-2 text-muted-foreground">
                                    <MessageSquare size={14} />
                                    <span className="text-[10px] font-black uppercase tracking-widest">
                                        {ticket.messages.length} messages
                                    </span>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div className="flex gap-2">
                        <button
                            onClick={() => toggleOpenMutation.mutate()}
                            className={cn(
                                "px-4 py-2 rounded-xl font-bold text-sm flex items-center gap-2 transition-all",
                                ticket.is_open
                                    ? "bg-muted text-muted-foreground hover:bg-muted/80"
                                    : "bg-primary text-primary-foreground hover:bg-primary/90"
                            )}
                        >
                            <ToggleLeft size={16} />
                            {ticket.is_open ? 'Close' : 'Reopen'}
                        </button>
                        <button
                            onClick={() => setContributiveMutation.mutate(!ticket.is_contributive)}
                            className={cn(
                                "px-4 py-2 rounded-xl font-bold text-sm flex items-center gap-2 transition-all",
                                ticket.is_contributive
                                    ? "bg-primary text-primary-foreground hover:bg-primary/90"
                                    : "bg-muted text-muted-foreground hover:bg-muted/80"
                            )}
                        >
                            <Star size={16} />
                            Contributive
                        </button>
                    </div>
                </div>

                {/* Assignees */}
                <div className="flex items-center gap-4 mb-4">
                    <div className="flex items-center gap-2 text-muted-foreground">
                        <Users size={14} />
                        <span className="text-[10px] font-black uppercase tracking-widest">Assignees:</span>
                    </div>
                    {ticket.assignees.length > 0 ? (
                        <div className="flex gap-2">
                            {ticket.assignees.map((assignee) => (
                                <Badge key={assignee} variant="secondary" className="text-[10px] font-black uppercase tracking-widest">
                                    <User size={12} className="inline mr-1" /> {assignee}
                                </Badge>
                            ))}
                        </div>
                    ) : (
                        <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground/50">Unassigned</span>
                    )}
                    <button
                        onClick={() => {
                            setSelectedAssignees([]);
                            setShowAssignModal(true);
                        }}
                        className="ml-auto px-4 py-2 rounded-xl bg-primary/10 text-primary font-bold text-sm hover:bg-primary/20 transition-all flex items-center gap-2"
                    >
                        <UserPlus size={14} />
                        Manage Assignees
                    </button>
                </div>
            </div>

            {/* Internal Notes */}
            <div className="bg-card border rounded-[3rem] p-8 shadow-sm">
                <h3 className="text-xl font-black mb-4 flex items-center gap-2">
                    <FileText size={20} className="text-primary" />
                    Internal Notes
                </h3>
                <textarea
                    value={notesContent}
                    onChange={(e) => setNotesContent(e.target.value)}
                    onBlur={handleNotesBlur}
                    placeholder="Add internal notes (only visible to staff)..."
                    rows={4}
                    className="w-full bg-muted/30 border border-muted-foreground/10 rounded-2xl px-4 py-3 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:border-primary/30 transition-all outline-none resize-none"
                />
                {updateNotesMutation.isPending && (
                    <p className="text-xs text-muted-foreground mt-2">Saving...</p>
                )}
            </div>

            {/* Messages */}
            <div className="space-y-6">
                {ticket.messages.length > 0 ? (
                    ticket.messages.map((message, index) => (
                        <MessageBubble
                            key={message.id}
                            message={message}
                            isOP={message.user.username === ticket.creator}
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
            {ticket.is_open && (
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

            {/* Assign Modal */}
            {showAssignModal && (
                <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
                    <div className="bg-card border rounded-[3rem] p-8 max-w-lg w-full mx-4 max-h-[80vh] overflow-y-auto">
                        <h3 className="text-xl font-black mb-4 flex items-center gap-2">
                            <Users size={20} className="text-primary" />
                            Assign Ticket
                        </h3>
                        <input
                            type="text"
                            placeholder="Search users..."
                            className="w-full h-12 bg-muted/30 border border-muted-foreground/10 rounded-2xl px-4 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:border-primary/30 transition-all outline-none mb-4"
                            value={userSearch}
                            onChange={(e) => setUserSearch(e.target.value)}
                        />
                        <div className="space-y-2 mb-6">
                            {users?.data?.map((user) => (
                                <label
                                    key={user.id}
                                    className={cn(
                                        "flex items-center gap-3 p-4 rounded-2xl border cursor-pointer transition-all",
                                        selectedAssignees.includes(user.id)
                                            ? "bg-primary/10 border-primary"
                                            : "bg-muted/30 border-transparent hover:bg-muted"
                                    )}
                                >
                                    <input
                                        type="checkbox"
                                        checked={selectedAssignees.includes(user.id)}
                                        onChange={(e) => {
                                            if (e.target.checked) {
                                                setSelectedAssignees([...selectedAssignees, user.id]);
                                            } else {
                                                setSelectedAssignees(selectedAssignees.filter(id => id !== user.id));
                                            }
                                        }}
                                        className="w-4 h-4 rounded"
                                    />
                                    <span className="font-black">{user.username}</span>
                                    {user.is_staff && (
                                        <Badge variant="secondary" className="text-[10px] font-black uppercase tracking-widest">
                                            Staff
                                        </Badge>
                                    )}
                                </label>
                            ))}
                        </div>
                        <div className="flex justify-end gap-4">
                            <button
                                onClick={() => setShowAssignModal(false)}
                                className="px-6 py-3 rounded-xl bg-muted text-muted-foreground font-bold hover:bg-muted/80 transition-all"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleAssign}
                                disabled={assignMutation.isPending}
                                className="px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-all flex items-center gap-2 disabled:opacity-50"
                            >
                                {assignMutation.isPending && <Loader2 size={18} className="animate-spin" />}
                                Assign
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
