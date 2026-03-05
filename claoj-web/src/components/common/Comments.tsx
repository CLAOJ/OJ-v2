'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Comment, CommentRevision, CommentUpdateRequest, PaginatedList } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { useState } from 'react';
import {
    MessageSquare,
    Send,
    Reply,
    User,
    Clock,
    ChevronDown,
    ChevronUp,
    MoreVertical,
    ThumbsUp,
    Loader2,
    Edit2,
    Trash2,
    History,
    Eye,
    Shield,
    X
} from 'lucide-react';
import { cn } from '@/lib/utils';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { motion, AnimatePresence } from 'framer-motion';
import { useAuth } from '@/components/providers/AuthProvider';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter
} from '@/components/ui/Dialog';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';

dayjs.extend(relativeTime);

interface CommentsProps {
    page: string; // e.g. "p/aplusb" or "blog/1"
}

export default function Comments({ page }: CommentsProps) {
    const t = useTranslations('Comments');
    const { user, isAdmin } = useAuth();
    const queryClient = useQueryClient();
    const [commentBody, setCommentBody] = useState('');
    const [replyTo, setReplyTo] = useState<number | null>(null);
    const [editingComment, setEditingComment] = useState<Comment | null>(null);
    const [revisionHistory, setRevisionHistory] = useState<CommentRevision[] | null>(null);
    const [deletingComment, setDeletingComment] = useState<number | null>(null);

    const { data: comments, isLoading } = useQuery({
        queryKey: ['comments', page],
        queryFn: async () => {
            const res = await api.get<PaginatedList<Comment>>(`/comments?page=${page}`);
            return res.data.data;
        }
    });

    const { mutate: postComment, isPending: isPosting } = useMutation({
        mutationFn: async (body: string) => {
            await api.post('/comments', {
                page,
                body,
                parent_id: replyTo
            });
        },
        onSuccess: () => {
            setCommentBody('');
            setReplyTo(null);
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        }
    });

    // Vote mutation
    const { mutate: voteComment } = useMutation({
        mutationFn: async ({ id, score }: { id: number; score: number }) => {
            await api.post(`/comment/${id}/vote`, { score });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        }
    });

    // Edit mutation
    const { mutate: editComment, isPending: isEditing } = useMutation({
        mutationFn: async ({ id, body, reason }: { id: number; body: string; reason?: string }) => {
            await api.patch(`/comment/${id}`, { body, reason });
        },
        onSuccess: () => {
            setEditingComment(null);
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        }
    });

    // Delete mutation
    const { mutate: deleteComment } = useMutation({
        mutationFn: async (id: number) => {
            await api.delete(`/comment/${id}`);
        },
        onSuccess: () => {
            setDeletingComment(null);
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        }
    });

    // Hide mutation (admin only)
    const { mutate: hideComment } = useMutation({
        mutationFn: async ({ id, hidden }: { id: number; hidden: boolean }) => {
            await api.post(`/admin/comment/${id}/hide`, { hidden });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        }
    });

    // Revision history query
    const { data: revisions } = useQuery({
        queryKey: ['comment-revisions', revisionHistory?.[0]?.id],
        queryFn: async () => {
            if (!revisionHistory?.[0]?.id) return [];
            const res = await api.get<{ data: CommentRevision[] }>(`/comment/${revisionHistory[0].id}/revisions`);
            return res.data.data;
        },
        enabled: !!revisionHistory
    });

    const handlePost = (body: string) => {
        if (!body.trim()) return;
        postComment(body);
    };

    if (isLoading) return (
        <div className="space-y-6">
            <Skeleton className="h-10 w-48 rounded-full" />
            <Skeleton className="h-40 w-full rounded-3xl" />
            <div className="space-y-4 pt-8">
                {[1, 2].map(i => <Skeleton key={i} className="h-24 w-full rounded-3xl" />)}
            </div>
        </div>
    );

    // Build tree structure
    const buildTree = (list: Comment[]) => {
        const map: Record<number, Comment & { children: any[] }> = {};
        const roots: any[] = [];

        list.forEach(c => {
            map[c.id] = { ...c, children: [] };
        });

        list.forEach(c => {
            if (c.parent_id && map[c.parent_id]) {
                map[c.parent_id].children.push(map[c.id]);
            } else {
                roots.push(map[c.id]);
            }
        });

        return roots;
    };

    const tree = buildTree(comments || []);

    return (
        <div className="space-y-10">
            <div className="flex items-center justify-between border-b pb-6">
                <h3 className="text-2xl font-black tracking-tighter flex items-center gap-3">
                    <MessageSquare className="text-primary" size={28} />
                    {t('title')}
                </h3>
                <span className="text-xs font-black uppercase tracking-widest text-muted-foreground bg-muted/50 border px-4 py-1.5 rounded-full">
                    {t('count', { count: comments?.length || 0 })}
                </span>
            </div>

            {/* New Root Comment Input */}
            {user ? (
                <div className="p-8 rounded-[2rem] bg-card border border-dashed hover:border-primary/50 transition-colors space-y-4">
                    <textarea
                        value={commentBody}
                        onChange={(e) => setCommentBody(e.target.value)}
                        placeholder={t('placeholder')}
                        className="w-full h-32 p-6 rounded-2xl bg-muted/30 border outline-none focus:ring-4 focus:ring-primary/10 transition-all resize-none text-base font-medium"
                    />
                    <div className="flex justify-end">
                        <button
                            onClick={() => handlePost(commentBody)}
                            disabled={isPosting || !commentBody.trim()}
                            className="flex items-center gap-2 h-12 px-8 rounded-xl bg-primary text-primary-foreground text-sm font-black shadow-xl shadow-primary/20 hover:opacity-90 active:scale-95 transition-all disabled:opacity-50"
                        >
                            {isPosting ? <Loader2 className="animate-spin" size={18} /> : <Send size={18} />}
                            {t('post')}
                        </button>
                    </div>
                </div>
            ) : (
                <div className="p-10 text-center rounded-[2rem] bg-muted/30 border border-dashed">
                    <p className="text-sm font-bold text-muted-foreground">{t('signInToComment')}</p>
                </div>
            )}

            {/* Comment List */}
            <div className="space-y-10 pt-4">
                {tree.length === 0 ? (
                    <div className="py-20 text-center text-muted-foreground/50 flex flex-col items-center gap-4">
                        <MessageSquare size={48} className="opacity-10" />
                        <p className="font-bold">{t('noComments')}</p>
                    </div>
                ) : (
                    tree.map(node => (
                        <CommentNode
                            key={node.id}
                            node={node}
                            user={user}
                            isAdmin={isAdmin}
                            t={t}
                            onReply={setReplyTo}
                            replyTo={replyTo}
                            onPost={handlePost}
                            isPosting={isPosting}
                            onEdit={setEditingComment}
                            onDelete={setDeletingComment}
                            onShowRevisions={setRevisionHistory}
                            onHide={hideComment}
                            onVote={voteComment}
                        />
                    ))
                )}
            </div>

            {/* Edit Comment Dialog */}
            <Dialog open={!!editingComment} onOpenChange={() => setEditingComment(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{t('edit')}</DialogTitle>
                        <DialogDescription>
                            Edit your comment. A revision history entry will be created.
                        </DialogDescription>
                    </DialogHeader>
                    <textarea
                        value={editingComment?.body || ''}
                        onChange={(e) => setEditingComment(editingComment ? { ...editingComment, body: e.target.value } : null)}
                        className="w-full h-40 p-4 rounded-lg border outline-none focus:ring-2 focus:ring-primary/20 resize-none"
                    />
                    <input
                        type="text"
                        placeholder="Edit reason (optional)"
                        className="w-full p-3 rounded-lg border outline-none focus:ring-2 focus:ring-primary/20"
                        onKeyDown={(e) => {
                            if (e.key === 'Enter' && editingComment) {
                                editComment({ id: editingComment.id, body: editingComment.body, reason: e.currentTarget.value });
                            }
                        }}
                    />
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setEditingComment(null)}>
                            {t('cancel')}
                        </Button>
                        <Button 
                            onClick={() => editingComment && editComment({ id: editingComment.id, body: editingComment.body })}
                            disabled={isEditing}
                        >
                            {isEditing ? <Loader2 className="animate-spin mr-2" size={16} /> : <Edit2 className="mr-2" size={16} />}
                            {t('edit')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <Dialog open={!!deletingComment} onOpenChange={() => setDeletingComment(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete Comment</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete this comment? This action will soft delete the comment.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeletingComment(null)}>
                            {t('cancel')}
                        </Button>
                        <Button 
                            variant="destructive"
                            onClick={() => deletingComment && deleteComment(deletingComment)}
                        >
                            <Trash2 className="mr-2" size={16} />
                            Delete
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Revision History Dialog */}
            <Dialog open={!!revisionHistory} onOpenChange={() => setRevisionHistory(null)}>
                <DialogContent className="max-w-3xl">
                    <DialogHeader>
                        <DialogTitle className="flex items-center gap-2">
                            <History className="text-primary" size={20} />
                            Comment Revision History
                        </DialogTitle>
                        <DialogDescription>
                            View all edits made to this comment
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 max-h-96 overflow-y-auto">
                        {revisions?.map((rev, idx) => (
                            <div key={rev.id} className="border rounded-lg p-4 bg-muted/30">
                                <div className="flex items-center justify-between mb-2">
                                    <div className="flex items-center gap-2">
                                        <User size={16} className="text-muted-foreground" />
                                        <span className="font-medium text-sm">{rev.editor}</span>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <Clock size={14} className="text-muted-foreground" />
                                        <span className="text-xs text-muted-foreground">{dayjs(rev.time).fromNow()}</span>
                                    </div>
                                </div>
                                {rev.reason && (
                                    <Badge variant="outline" className="mb-2">
                                        Reason: {rev.reason}
                                    </Badge>
                                )}
                                <pre className="whitespace-pre-wrap text-sm font-medium bg-card p-3 rounded border">
                                    {rev.body}
                                </pre>
                            </div>
                        ))}
                        {!revisions?.length && (
                            <p className="text-center text-muted-foreground py-8">No revision history available</p>
                        )}
                    </div>
                </DialogContent>
            </Dialog>
        </div>
    );
}

interface CommentNodeProps {
    node: Comment & { children: Comment[] };
    level?: number;
    onReply: (id: number | null) => void;
    replyTo: number | null;
    onPost: (body: string) => void;
    isPosting: boolean;
    user: any;
    isAdmin: boolean;
    t: (key: string) => string;
    onEdit: (comment: Comment | null) => void;
    onDelete: (id: number | null) => void;
    onShowRevisions: (comment: Comment[] | null) => void;
    onHide: (params: { id: number; hidden: boolean }) => void;
    onVote: (params: { id: number; score: number }) => void;
}

function CommentNode({ node, level = 0, onReply, replyTo, onPost, isPosting, user, isAdmin, t, onEdit, onDelete, onShowRevisions, onHide, onVote }: CommentNodeProps) {
    const [isExpanded, setIsExpanded] = useState(true);
    const [replyBody, setReplyBody] = useState('');
    const [showMenu, setShowMenu] = useState(false);
    const [voteDirection, setVoteDirection] = useState<number>(0);

    const isEditable = user && (node.author === user.username);
    const canModerate = isAdmin;

    return (
        <div className={cn("space-y-6", level > 0 && "ml-4 md:ml-12 border-l-2 border-primary/5 pl-4 md:ml-12 pl-4")}>
            <div className="group space-y-4 bg-card/50 p-6 rounded-[2rem] border border-transparent hover:border-muted hover:shadow-xl hover:shadow-primary/5 transition-all">
                <div className="flex items-start justify-between">
                    <div className="flex items-center gap-4">
                        <div className="w-12 h-12 rounded-2xl bg-primary/10 flex items-center justify-center text-primary shadow-inner">
                            <User size={24} />
                        </div>
                        <div>
                            <div className="text-base font-black tracking-tight">@{node.author}</div>
                            <div className="text-[10px] font-black text-muted-foreground uppercase flex items-center gap-1.5 tracking-widest opacity-60">
                                <Clock size={12} />
                                {dayjs(node.time).fromNow()}
                            </div>
                        </div>
                    </div>
                    <div className="relative">
                        <button 
                            onClick={() => setShowMenu(!showMenu)}
                            className="p-2 opacity-0 group-hover:opacity-100 transition-opacity hover:bg-muted rounded-lg"
                        >
                            <MoreVertical size={18} className="text-muted-foreground" />
                        </button>
                        {showMenu && (
                            <div className="absolute right-0 top-full mt-1 bg-card border rounded-lg shadow-lg z-10 min-w-[160px] overflow-hidden">
                                {isEditable && (
                                    <>
                                        <button
                                            onClick={() => { onEdit(node); setShowMenu(false); }}
                                            className="w-full px-4 py-2 text-left text-sm flex items-center gap-2 hover:bg-muted transition-colors"
                                        >
                                            <Edit2 size={16} />
                                            {t('edit')}
                                        </button>
                                        <button
                                            onClick={() => { onDelete(node.id); setShowMenu(false); }}
                                            className="w-full px-4 py-2 text-left text-sm flex items-center gap-2 hover:bg-destructive/10 text-destructive transition-colors"
                                        >
                                            <Trash2 size={16} />
                                            Delete
                                        </button>
                                    </>
                                )}
                                <button
                                    onClick={() => { onShowRevisions([node]); setShowMenu(false); }}
                                    className="w-full px-4 py-2 text-left text-sm flex items-center gap-2 hover:bg-muted transition-colors"
                                >
                                    <History size={16} />
                                    View history
                                </button>
                                {canModerate && (
                                    <>
                                        <div className="border-t my-1" />
                                        <button
                                            onClick={() => { onHide({ id: node.id, hidden: !node.hidden }); setShowMenu(false); }}
                                            className="w-full px-4 py-2 text-left text-sm flex items-center gap-2 hover:bg-muted transition-colors"
                                        >
                                            <Shield size={16} />
                                            {node.hidden ? 'Unhide' : 'Hide'} comment
                                        </button>
                                    </>
                                )}
                            </div>
                        )}
                    </div>
                </div>

                {node.hidden ? (
                    <div className="pl-1 text-[15px] font-medium leading-relaxed text-muted-foreground/60 italic flex items-center gap-2">
                        <Eye size={16} />
                        This comment has been hidden
                    </div>
                ) : (
                    <div className="pl-1 text-[15px] font-medium leading-relaxed text-muted-foreground/90 whitespace-pre-wrap">
                        {node.body}
                    </div>
                )}

                <div className="flex items-center gap-8 pt-2">
                    <div className="flex items-center gap-2">
                        <button
                            onClick={() => { setVoteDirection(voteDirection === 1 ? 0 : 1); onVote({ id: node.id, score: voteDirection === 1 ? -1 : 1 }); }}
                            className={cn(
                                "flex items-center gap-1.5 text-xs font-black px-3 py-1.5 rounded-full transition-colors",
                                voteDirection === 1 
                                    ? "bg-green-500/10 text-green-500" 
                                    : "text-muted-foreground hover:text-primary hover:bg-primary/5"
                            )}
                        >
                            <ThumbsUp size={14} />
                            {node.score + voteDirection}
                        </button>
                    </div>
                    <button
                        onClick={() => onReply(replyTo === node.id ? null : node.id)}
                        className="flex items-center gap-2 text-xs font-black text-muted-foreground hover:text-primary transition-colors"
                    >
                        <Reply size={16} />
                        {t('reply')}
                    </button>
                    {node.children.length > 0 && (
                        <button
                            onClick={() => setIsExpanded(!isExpanded)}
                            className="flex items-center gap-1.5 text-xs font-black text-primary bg-primary/5 px-4 py-2 rounded-full"
                        >
                            {isExpanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
                            {node.children.length} {t('reply')}
                        </button>
                    )}
                </div>

                <AnimatePresence>
                    {replyTo === node.id && user && (
                        <motion.div
                            initial={{ opacity: 0, scale: 0.95 }}
                            animate={{ opacity: 1, scale: 1 }}
                            exit={{ opacity: 0, scale: 0.95 }}
                            className="pt-6 space-y-4"
                        >
                            <textarea
                                value={replyBody}
                                onChange={(e) => setReplyBody(e.target.value)}
                                autoFocus
                                placeholder={`${t('reply')} @${node.author}...`}
                                className="w-full h-24 p-5 rounded-2xl bg-muted border outline-none focus:ring-4 focus:ring-primary/10 transition-all resize-none text-sm font-medium"
                            />
                            <div className="flex justify-end gap-3">
                                <button
                                    onClick={() => onReply(null)}
                                    className="px-6 h-10 rounded-xl text-xs font-black uppercase tracking-widest hover:bg-muted transition-colors"
                                >
                                    {t('cancel')}
                                </button>
                                <button
                                    onClick={() => {
                                        onPost(replyBody);
                                        setReplyBody('');
                                    }}
                                    disabled={isPosting || !replyBody.trim()}
                                    className="px-8 h-10 rounded-xl bg-primary text-primary-foreground text-xs font-black uppercase tracking-widest shadow-xl shadow-primary/20 hover:opacity-90 transition-all disabled:opacity-50"
                                >
                                    {t('post')}
                                </button>
                            </div>
                        </motion.div>
                    )}
                </AnimatePresence>
            </div>

            {isExpanded && node.children.length > 0 && (
                <div className="space-y-8">
                    {node.children.map((child: any) => (
                        <CommentNode
                            key={child.id}
                            node={child}
                            level={level + 1}
                            onReply={onReply}
                            replyTo={replyTo}
                            onPost={onPost}
                            isPosting={isPosting}
                            user={user}
                            isAdmin={isAdmin}
                            t={t}
                            onEdit={onEdit}
                            onDelete={onDelete}
                            onShowRevisions={onShowRevisions}
                            onHide={onHide}
                            onVote={onVote}
                        />
                    ))}
                </div>
            )}
        </div>
    );
}
