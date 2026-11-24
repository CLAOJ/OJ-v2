'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { adminTicketApi, adminUserApi, adminErrorMessage } from '@/lib/adminApi';
import { searchProblems, searchContests, searchUsers } from '@/lib/adminSearch';
import { EntityPicker } from '@/components/admin/EntityPicker';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { AdminTicketDetail, AdminTicketMessageUpdateRequest, AdminTicketUpdateRequest, TicketMessage } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { Badge } from '@/components/ui/Badge';
import { Link, useRouter } from '@/navigation';
import { useParams } from 'next/navigation';
import { cn } from '@/lib/utils';
import { toast } from 'sonner';
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
    Star,
    UserPlus,
    FileText,
    ToggleLeft,
    Trash2,
    Users,
    Pencil,
    Save,
    X,
    ExternalLink
} from 'lucide-react';
import { motion } from 'framer-motion';

dayjs.extend(relativeTime);

/** Title, creator and linked item are the v1-editable ticket fields; `time` stays read-only. */
function TicketDetailsForm({ ticket, onSaved }: { ticket: AdminTicketDetail; onSaved: () => void }) {
    const t = useTranslations('Admin.Tickets');
    const [title, setTitle] = useState(ticket.title);
    const [creatorIds, setCreatorIds] = useState<number[]>(ticket.creator_id ? [ticket.creator_id] : []);
    const [linkedType, setLinkedType] = useState<'problem' | 'contest' | ''>(ticket.linked_type || '');
    // Selections are kept per type so toggling problem -> contest -> problem
    // does not throw the previous pick away.
    const [linkedSelections, setLinkedSelections] = useState<Record<string, number[]>>(
        () => (ticket.linked_type && ticket.linked_id ? { [ticket.linked_type]: [ticket.linked_id] } : {})
    );
    const linkedIds = linkedSelections[linkedType] ?? [];
    const setLinkedIds = (ids: number[]) => setLinkedSelections(prev => ({ ...prev, [linkedType]: ids }));
    const linkedIncomplete = linkedType !== '' && !linkedIds[0];

    const saveMutation = useMutation({
        mutationFn: () => {
            // Only send what changed: an untouched (possibly orphaned) link must
            // survive a save, and user_id is a judge_profile id, so 0 is invalid.
            const data: AdminTicketUpdateRequest = {};
            if (title !== ticket.title) data.title = title;
            if (creatorIds[0] && creatorIds[0] !== ticket.creator_id) data.user_id = creatorIds[0];
            const linkedChanged = linkedType !== (ticket.linked_type || '')
                || (linkedType !== '' && linkedIds[0] !== ticket.linked_id);
            if (linkedChanged) {
                data.linked_type = linkedType;
                data.linked_id = linkedType === '' ? 0 : linkedIds[0];
            }
            return adminTicketApi.update(ticket.id, data);
        },
        onSuccess: () => {
            toast.success(t('detailsSaved'));
            onSaved();
        },
        onError: (error) => toast.error(adminErrorMessage(error, t('detailsSaveFailed'))),
    });

    const linkedSearch = linkedType === 'contest' ? searchContests : searchProblems;
    const linkedQueryKey = linkedType === 'contest' ? 'contests' : 'problems';
    const linkedLabel = linkedType === 'contest' ? t('linkedContestOption') : t('linkedProblemOption');
    const linkedInitial = ticket.linked_type === linkedType && ticket.linked_id
        ? [{ id: ticket.linked_id, label: ticket.linked_label }]
        : [];

    return (
        <div className="space-y-4">
            <div className="space-y-2">
                <label htmlFor="ticket-title" className="text-sm font-medium text-muted-foreground block">{t('titleLabel')}</label>
                <input
                    id="ticket-title"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    className="w-full text-3xl font-semibold tracking-tight bg-transparent border-b border-transparent hover:border-muted-foreground/20 focus:border-primary/40 outline-none pb-1 transition-colors"
                />
            </div>

            <div className="grid gap-4 md:grid-cols-2">
                <EntityPicker
                    id="ticket-creator"
                    label={t('creatorLabel')}
                    value={creatorIds}
                    onChange={setCreatorIds}
                    search={searchUsers}
                    queryKey="users"
                    initialOptions={ticket.creator_id ? [{ id: ticket.creator_id, label: ticket.creator }] : []}
                    single
                    clearable={false}
                />
                <div className="space-y-2">
                    <label htmlFor="ticket-linked-type" className="text-sm font-medium text-muted-foreground block">{t('linkedItemLabel')}</label>
                    <select
                        id="ticket-linked-type"
                        value={linkedType}
                        onChange={(e) => setLinkedType(e.target.value as 'problem' | 'contest' | '')}
                        className="w-full h-9 px-3 rounded-md border bg-card text-sm focus:outline-none focus:ring-2 focus:ring-ring/40"
                    >
                        <option value="">{t('linkedNoneOption')}</option>
                        <option value="problem">{t('linkedProblemOption')}</option>
                        <option value="contest">{t('linkedContestOption')}</option>
                    </select>
                    {linkedType !== '' && (
                        <EntityPicker
                            id="ticket-linked-item"
                            label={linkedLabel}
                            value={linkedIds}
                            onChange={setLinkedIds}
                            search={linkedSearch}
                            queryKey={linkedQueryKey}
                            initialOptions={linkedInitial}
                            single
                            clearable={false}
                        />
                    )}
                    {ticket.linked_path && (
                        <Link
                            href={ticket.linked_path}
                            className="inline-flex items-center gap-1 text-sm text-primary hover:underline"
                        >
                            <ExternalLink size={14} />
                            {ticket.linked_label}
                        </Link>
                    )}
                </div>
            </div>

            <div className="flex justify-end">
                <button
                    onClick={() => saveMutation.mutate()}
                    disabled={saveMutation.isPending || !title.trim() || linkedIncomplete}
                    className="px-6 py-2 rounded-xl bg-primary text-primary-foreground font-medium text-sm hover:bg-primary/90 transition-all flex items-center gap-2 disabled:opacity-50"
                >
                    {saveMutation.isPending ? <Loader2 size={16} className="animate-spin" /> : <Save size={16} />}
                    {t('saveDetailsButton')}
                </button>
            </div>
        </div>
    );
}

interface MessageBubbleProps {
    ticketId: number;
    message: TicketMessage;
    isOP: boolean;
    onDelete: (message: TicketMessage) => void;
    onChanged: () => void;
}

function MessageBubble({ ticketId, message, isOP, onDelete, onChanged }: MessageBubbleProps) {
    const t = useTranslations('Admin.Tickets');
    const [editing, setEditing] = useState(false);
    const [draft, setDraft] = useState(message.body);
    const [authorIds, setAuthorIds] = useState<number[]>([message.user_id]);

    const updateMutation = useMutation({
        mutationFn: (body: string) => {
            const data: AdminTicketMessageUpdateRequest = { body };
            // Only send the author when the picker changed it (profile id).
            if (authorIds[0] && authorIds[0] !== message.user_id) data.user_id = authorIds[0];
            return adminTicketApi.updateMessage(ticketId, message.id, data);
        },
        onSuccess: () => {
            toast.success(t('messageSaved'));
            setEditing(false);
            onChanged();
        },
        onError: (error) => toast.error(adminErrorMessage(error, t('messageSaveFailed'))),
    });

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className={cn(
                "flex gap-4",
                isOP ? "flex-row" : "flex-row-reverse"
            )}
        >
            <div className={cn(
                "w-12 h-12 rounded-md flex items-center justify-center flex-shrink-0 font-semibold text-lg",
                isOP ? "bg-primary/10 text-primary" : "bg-muted text-muted-foreground"
            )}>
                {message.username[0]?.toUpperCase()}
            </div>
            <div className={cn(
                "flex-1 bg-card border rounded-md p-6",
                isOP ? "" : "bg-muted/30"
            )}>
                <div className="flex items-center gap-3 mb-3">
                    <span className="font-medium">{message.username}</span>
                    {isOP && (
                        <Badge className="text-xs font-medium bg-primary/10 text-primary border-primary/20">
                            <User size={12} className="inline mr-1" /> {t('ticketOwnerBadge')}
                        </Badge>
                    )}
                    <div className="flex items-center gap-1 ml-auto">
                        {!editing && (
                            <>
                                <button
                                    onClick={() => { setDraft(message.body); setEditing(true); }}
                                    aria-label={t('editMessage')}
                                    title={t('editMessage')}
                                    className="p-1.5 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                                >
                                    <Pencil size={14} />
                                </button>
                                <button
                                    onClick={() => onDelete(message)}
                                    aria-label={t('deleteMessage')}
                                    title={t('deleteMessage')}
                                    className="p-1.5 rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors"
                                >
                                    <Trash2 size={14} />
                                </button>
                            </>
                        )}
                        <span className="text-xs text-muted-foreground font-mono ml-2">
                            {dayjs(message.time).format('MMM D, YYYY [at] HH:mm')}
                        </span>
                    </div>
                </div>
                {editing ? (
                    <div className="space-y-3">
                        <EntityPicker
                            id={`message-author-${message.id}`}
                            label={t('messageAuthorLabel')}
                            value={authorIds}
                            onChange={setAuthorIds}
                            search={searchUsers}
                            queryKey="users"
                            initialOptions={[{ id: message.user_id, label: message.username }]}
                            single
                            clearable={false}
                        />
                        <textarea
                            value={draft}
                            onChange={(e) => setDraft(e.target.value)}
                            rows={4}
                            className="w-full bg-background border border-muted-foreground/10 rounded-md px-4 py-3 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary/30 transition-all outline-none resize-none"
                        />
                        <div className="flex justify-end gap-2">
                            <button
                                onClick={() => { setDraft(message.body); setEditing(false); }}
                                disabled={updateMutation.isPending}
                                className="px-4 py-2 rounded-xl bg-muted text-muted-foreground font-medium text-sm hover:bg-muted/80 transition-all flex items-center gap-2 disabled:opacity-50"
                            >
                                <X size={14} /> {t('cancelMessageButton')}
                            </button>
                            <button
                                onClick={() => updateMutation.mutate(draft)}
                                disabled={updateMutation.isPending || !draft.trim()}
                                className="px-4 py-2 rounded-xl bg-primary text-primary-foreground font-medium text-sm hover:bg-primary/90 transition-all flex items-center gap-2 disabled:opacity-50"
                            >
                                {updateMutation.isPending ? <Loader2 size={14} className="animate-spin" /> : <Save size={14} />}
                                {t('saveMessageButton')}
                            </button>
                        </div>
                    </div>
                ) : (
                    <div className="prose prose-sm dark:prose-invert max-w-none text-muted-foreground">
                        <p className="whitespace-pre-wrap leading-relaxed">{message.body}</p>
                    </div>
                )}
            </div>
        </motion.div>
    );
}

export default function AdminTicketDetailPage() {
    const t = useTranslations('Admin.Tickets');
    const tc = useTranslations('Admin');
    const router = useRouter();
    const params = useParams();
    const id = params?.id as string;
    const queryClient = useQueryClient();
    const [replyContent, setReplyContent] = useState('');
    const [notesContent, setNotesContent] = useState('');
    const [showAssignModal, setShowAssignModal] = useState(false);
    const [selectedAssignees, setSelectedAssignees] = useState<number[]>([]);
    const [userSearch, setUserSearch] = useState('');
    const [showDelete, setShowDelete] = useState(false);
    const [messageToDelete, setMessageToDelete] = useState<TicketMessage | null>(null);

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

    const deleteMessageMutation = useMutation({
        mutationFn: (messageId: number) => adminTicketApi.deleteMessage(parseInt(id), messageId),
        onSuccess: () => {
            toast.success(t('messageDeleted'));
            queryClient.invalidateQueries({ queryKey: ['admin-ticket', id] });
            setMessageToDelete(null);
        },
        onError: (error) => toast.error(adminErrorMessage(error, t('messageDeleteFailed'))),
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

    const deleteMutation = useMutation({
        mutationFn: () => adminTicketApi.delete(parseInt(id)),
        onSuccess: () => {
            queryClient.removeQueries({ queryKey: ['admin-ticket', id] });
            queryClient.invalidateQueries({ queryKey: ['admin-tickets'] });
            router.push('/admin/tickets');
        },
    });

    if (isLoading) {
        return (
            <div className="max-w-6xl mx-auto space-y-8 mt-4 pb-20">
                <Skeleton className="h-12 w-48 rounded-xl" />
                <Skeleton className="h-64 rounded-md" />
                <Skeleton className="h-96 rounded-md" />
            </div>
        );
    }

    if (!ticket) {
        return (
            <div className="max-w-2xl mx-auto text-center py-20">
                <TicketIcon size={64} className="mx-auto text-muted-foreground opacity-20 mb-4" />
                <h2 className="text-2xl font-semibold mb-2">{t('notFoundTitle')}</h2>
                <p className="text-muted-foreground mb-6">{t('notFoundDesc')}</p>
                <Link
                    href="/admin/tickets"
                    className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-colors"
                >
                    <ArrowLeft size={18} />
                    {t('backToTickets')}
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

    return (
        <div className="max-w-6xl mx-auto space-y-8 mt-4 pb-20">
            {/* Back Button */}
            <Link
                href="/admin/tickets"
                className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-primary transition-colors"
            >
                <ArrowLeft size={16} />
                {t('backToAdminTickets')}
            </Link>

            {/* Header */}
            <div className="bg-card border rounded-md p-8 shadow-sm">
                <div className="flex items-start justify-between gap-4 mb-6">
                    <div className="flex items-center gap-4 flex-wrap">
                        {ticket.is_open ? (
                            <AlertCircle size={32} className="text-warning flex-shrink-0" />
                        ) : (
                            <CheckCircle2 size={32} className="text-muted-foreground flex-shrink-0" />
                        )}
                        <div className="flex flex-wrap items-center gap-4 text-muted-foreground">
                            <div className="flex items-center gap-2">
                                <Clock size={14} />
                                <span className="text-xs font-medium ">{t('createdLabel', { time: dayjs(ticket.created).fromNow() })}</span>
                            </div>
                            <div className="flex items-center gap-2">
                                <MessageSquare size={14} />
                                <span className="text-xs font-medium ">{t('messagesCount', { count: ticket.messages.length })}</span>
                            </div>
                        </div>
                    </div>
                    <div className="flex gap-2">
                        <button
                            onClick={() => toggleOpenMutation.mutate()}
                            className={cn(
                                "px-4 py-2 rounded-xl font-medium text-sm flex items-center gap-2 transition-all",
                                ticket.is_open
                                    ? "bg-muted text-muted-foreground hover:bg-muted/80"
                                    : "bg-primary text-primary-foreground hover:bg-primary/90"
                            )}
                        >
                            <ToggleLeft size={16} />
                            {ticket.is_open ? t('closeButton') : t('reopenButton')}
                        </button>
                        <button
                            onClick={() => setContributiveMutation.mutate(!ticket.is_contributive)}
                            className={cn(
                                "px-4 py-2 rounded-xl font-medium text-sm flex items-center gap-2 transition-all",
                                ticket.is_contributive
                                    ? "bg-primary text-primary-foreground hover:bg-primary/90"
                                    : "bg-muted text-muted-foreground hover:bg-muted/80"
                            )}
                        >
                            <Star size={16} />
                            {t('contributiveButton')}
                        </button>
                    </div>
                </div>

                <TicketDetailsForm
                    key={ticket.id}
                    ticket={ticket}
                    onSaved={() => queryClient.invalidateQueries({ queryKey: ['admin-ticket', id] })}
                />

                {/* Assignees */}
                <div className="mt-6 pt-6 border-t flex items-center gap-4">
                    <div className="flex items-center gap-2 text-muted-foreground">
                        <Users size={14} />
                        <span className="text-xs font-medium ">{t('assigneesLabel')}</span>
                    </div>
                    {ticket.assignees.length > 0 ? (
                        <div className="flex gap-2">
                            {ticket.assignees.map((assignee) => (
                                <Badge key={assignee} variant="secondary" className="text-xs font-medium ">
                                    <User size={12} className="inline mr-1" /> {assignee}
                                </Badge>
                            ))}
                        </div>
                    ) : (
                        <span className="text-xs font-medium text-muted-foreground/50">{t('unassigned')}</span>
                    )}
                    <button
                        onClick={() => {
                            setSelectedAssignees([]);
                            setShowAssignModal(true);
                        }}
                        className="ml-auto px-4 py-2 rounded-xl bg-primary/10 text-primary font-medium text-sm hover:bg-primary/20 transition-all flex items-center gap-2"
                    >
                        <UserPlus size={14} />
                        {t('manageAssigneesButton')}
                    </button>
                </div>
            </div>

            {/* Danger zone */}
            <div className="flex items-center justify-between gap-4 border border-destructive/30 rounded-md p-4">
                <p className="text-sm text-muted-foreground">{t('deleteTicketHint')}</p>
                <button
                    onClick={() => { deleteMutation.reset(); setShowDelete(true); }}
                    className="px-4 py-2 rounded-xl bg-destructive/10 text-destructive hover:bg-destructive/20 font-medium text-sm flex items-center gap-2 transition-all"
                >
                    <Trash2 size={16} />
                    {t('deleteTicketButton')}
                </button>
            </div>

            {/* Internal Notes */}
            <div className="bg-card border rounded-md p-8 shadow-sm">
                <h3 className="text-xl font-semibold mb-4 flex items-center gap-2">
                    <FileText size={20} className="text-primary" />
                    {t('internalNotesTitle')}
                </h3>
                <textarea
                    value={notesContent}
                    onChange={(e) => setNotesContent(e.target.value)}
                    onBlur={handleNotesBlur}
                    placeholder={t('notesPlaceholder')}
                    rows={4}
                    className="w-full bg-muted/30 border border-muted-foreground/10 rounded-md px-4 py-3 text-sm font-medium focus:ring-2 focus:ring-primary/20 focus:border-primary/30 transition-all outline-none resize-none"
                />
                {updateNotesMutation.isPending && (
                    <p className="text-xs text-muted-foreground mt-2">{t('savingIndicator')}</p>
                )}
            </div>

            {/* Messages */}
            <div className="space-y-6">
                {ticket.messages.length > 0 ? (
                    ticket.messages.map((message) => (
                        <MessageBubble
                            key={message.id}
                            ticketId={ticket.id}
                            message={message}
                            isOP={message.username === ticket.creator}
                            onDelete={setMessageToDelete}
                            onChanged={() => queryClient.invalidateQueries({ queryKey: ['admin-ticket', id] })}
                        />
                    ))
                ) : (
                    <div className="text-center py-12 text-muted-foreground border-2 border-dashed rounded-md">
                        <MessageSquare size={48} className="mx-auto mb-4 opacity-10" />
                        <p className="font-medium">{t('noMessagesYet')}</p>
                    </div>
                )}
            </div>

            {/* Reply Form */}
            {ticket.is_open && (
                <div className="bg-card border rounded-md p-8 shadow-sm">
                    <h3 className="text-xl font-semibold mb-4 flex items-center gap-2">
                        <MessageSquare size={20} className="text-primary" />
                        {t('replyTitle')}
                    </h3>
                    <div className="space-y-4">
                        <textarea
                            value={replyContent}
                            onChange={(e) => setReplyContent(e.target.value)}
                            placeholder={t('replyPlaceholder')}
                            rows={6}
                            className="w-full bg-muted/30 border border-muted-foreground/10 rounded-md px-4 py-3 text-sm font-medium focus:ring-2 focus:ring-primary/20 focus:border-primary/30 transition-all outline-none resize-none"
                        />
                        <div className="flex justify-end">
                            <button
                                onClick={handleReply}
                                disabled={replyMutation.isPending || !replyContent.trim()}
                                className="px-6 py-3 rounded-xl bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-all flex items-center gap-2 disabled:opacity-50 "
                            >
                                {replyMutation.isPending ? (
                                    <Loader2 size={18} className="animate-spin" />
                                ) : (
                                    <Send size={18} />
                                )}
                                {t('sendReplyButton')}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            <ConfirmDialog
                isOpen={showDelete}
                onClose={() => setShowDelete(false)}
                onConfirm={() => deleteMutation.mutate()}
                title={t('deleteTicketTitle')}
                description={t('deleteTicketConfirm')}
                confirmText={tc('common.delete')}
                cancelText={tc('common.cancel')}
                isLoading={deleteMutation.isPending}
            >
                {deleteMutation.isError && (
                    <p role="alert" className="text-sm text-destructive">
                        {adminErrorMessage(deleteMutation.error, t('deleteTicketError'))}
                    </p>
                )}
            </ConfirmDialog>

            <ConfirmDialog
                isOpen={messageToDelete !== null}
                onClose={() => setMessageToDelete(null)}
                onConfirm={() => messageToDelete && deleteMessageMutation.mutate(messageToDelete.id)}
                title={t('deleteMessageTitle')}
                description={t('deleteMessageConfirm')}
                confirmText={tc('common.delete')}
                cancelText={tc('common.cancel')}
                isLoading={deleteMessageMutation.isPending}
            />

            {/* Assign Modal */}
            {showAssignModal && (
                <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
                    <div className="bg-card border rounded-md p-8 max-w-lg w-full mx-4 max-h-[80vh] overflow-y-auto">
                        <h3 className="text-xl font-semibold mb-4 flex items-center gap-2">
                            <Users size={20} className="text-primary" />
                            {t('assignModalTitle')}
                        </h3>
                        <input
                            type="text"
                            placeholder={t('searchUsersPlaceholder')}
                            className="w-full h-12 bg-muted/30 border border-muted-foreground/10 rounded-md px-4 text-sm font-medium focus:ring-2 focus:ring-primary/20 focus:border-primary/30 transition-all outline-none mb-4"
                            value={userSearch}
                            onChange={(e) => setUserSearch(e.target.value)}
                        />
                        <div className="space-y-2 mb-6">
                            {users?.data?.map((user) => (
                                <label
                                    key={user.id}
                                    className={cn(
                                        "flex items-center gap-3 p-4 rounded-md border cursor-pointer transition-all",
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
                                    <span className="font-medium">{user.username}</span>
                                    {user.is_staff && (
                                        <Badge variant="secondary" className="text-xs font-medium ">
                                            {t('staffBadge')}
                                        </Badge>
                                    )}
                                </label>
                            ))}
                        </div>
                        <div className="flex justify-end gap-4">
                            <button
                                onClick={() => setShowAssignModal(false)}
                                className="px-6 py-3 rounded-xl bg-muted text-muted-foreground font-medium hover:bg-muted/80 transition-all"
                            >
                                {tc('common.cancel')}
                            </button>
                            <button
                                onClick={handleAssign}
                                disabled={assignMutation.isPending}
                                className="px-6 py-3 rounded-xl bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-all flex items-center gap-2 disabled:opacity-50"
                            >
                                {assignMutation.isPending && <Loader2 size={18} className="animate-spin" />}
                                {t('assignButton')}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}