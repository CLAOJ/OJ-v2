'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/Dialog';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { Comment, CommentRevision } from '@/types';
import { Edit2, Trash2, History, User, Clock, Loader2 } from 'lucide-react';
import dayjs from 'dayjs';

interface CommentDialogsProps {
    editingComment: Comment | null;
    deletingComment: number | null;
    revisionHistory: Comment | null;
    onEditCancel: () => void;
    onEditSubmit: (id: number, body: string, reason?: string) => void;
    onDeleteCancel: () => void;
    onDeleteConfirm: (id: number) => void;
    onRevisionClose: () => void;
    isEditing: boolean;
    t: ReturnType<typeof useTranslations>;
}

export function CommentDialogs({
    editingComment,
    deletingComment,
    revisionHistory,
    onEditCancel,
    onEditSubmit,
    onDeleteCancel,
    onDeleteConfirm,
    onRevisionClose,
    isEditing,
    t
}: CommentDialogsProps) {
    return (
        <>
            <EditCommentDialog
                editingComment={editingComment}
                onEditCancel={onEditCancel}
                onEditSubmit={onEditSubmit}
                isEditing={isEditing}
                t={t}
            />
            <DeleteCommentDialog
                deletingComment={deletingComment}
                onDeleteCancel={onDeleteCancel}
                onDeleteConfirm={onDeleteConfirm}
                t={t}
            />
            <RevisionHistoryDialog
                revisionHistory={revisionHistory}
                onRevisionClose={onRevisionClose}
                t={t}
            />
        </>
    );
}

interface EditCommentDialogProps {
    editingComment: Comment | null;
    onEditCancel: () => void;
    onEditSubmit: (id: number, body: string, reason?: string) => void;
    isEditing: boolean;
    t: ReturnType<typeof useTranslations>;
}

function EditCommentDialog({ editingComment, onEditCancel, onEditSubmit, isEditing, t }: EditCommentDialogProps) {
    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>, reason: string) => {
        if (e.key === 'Enter' && editingComment) {
            onEditSubmit(editingComment.id, editingComment.body, reason);
        }
    };

    return (
        <Dialog open={!!editingComment} onOpenChange={onEditCancel}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>{t('edit')}</DialogTitle>
                    <DialogDescription>
                        Edit your comment. A revision history entry will be created.
                    </DialogDescription>
                </DialogHeader>
                <textarea
                    value={editingComment?.body || ''}
                    onChange={(e) => onEditCancel()}
                    className="w-full h-40 p-4 rounded-lg border outline-none focus:ring-2 focus:ring-primary/20 resize-none"
                />
                <input
                    type="text"
                    placeholder="Edit reason (optional)"
                    className="w-full p-3 rounded-lg border outline-none focus:ring-2 focus:ring-primary/20"
                    onKeyDown={(e) => editingComment && handleKeyDown(e, e.currentTarget.value)}
                />
                <DialogFooter>
                    <Button variant="outline" onClick={onEditCancel}>
                        {t('cancel')}
                    </Button>
                    <Button
                        onClick={() => editingComment && onEditSubmit(editingComment.id, editingComment.body)}
                        disabled={isEditing}
                    >
                        {isEditing ? <Loader2 className="animate-spin mr-2" size={16} /> : <Edit2 className="mr-2" size={16} />}
                        {t('edit')}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

interface DeleteCommentDialogProps {
    deletingComment: number | null;
    onDeleteCancel: () => void;
    onDeleteConfirm: (id: number) => void;
    t: ReturnType<typeof useTranslations>;
}

function DeleteCommentDialog({ deletingComment, onDeleteCancel, onDeleteConfirm, t }: DeleteCommentDialogProps) {
    return (
        <Dialog open={!!deletingComment} onOpenChange={onDeleteCancel}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Delete Comment</DialogTitle>
                    <DialogDescription>
                        Are you sure you want to delete this comment? This action will soft delete the comment.
                    </DialogDescription>
                </DialogHeader>
                <DialogFooter>
                    <Button variant="outline" onClick={onDeleteCancel}>
                        {t('cancel')}
                    </Button>
                    <Button
                        variant="destructive"
                        onClick={() => deletingComment && onDeleteConfirm(deletingComment)}
                    >
                        <Trash2 className="mr-2" size={16} />
                        Delete
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

interface RevisionHistoryDialogProps {
    revisionHistory: Comment | null;
    onRevisionClose: () => void;
    t: ReturnType<typeof useTranslations>;
}

function RevisionHistoryDialog({ revisionHistory, onRevisionClose, t }: RevisionHistoryDialogProps) {
    const { data: revisions } = useQuery({
        queryKey: ['comment-revisions', revisionHistory?.id],
        queryFn: async () => {
            if (!revisionHistory?.id) return [];
            const res = await import('@/lib/api');
            const apiRes = await res.default.get<{ data: CommentRevision[] }>(`/comment/${revisionHistory.id}/revisions`);
            return apiRes.data.data;
        },
        enabled: !!revisionHistory
    });

    return (
        <Dialog open={!!revisionHistory} onOpenChange={onRevisionClose}>
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
                    {revisions?.map((rev) => (
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
                        <p className="text-center text-muted-foreground py-8">{t('noRevisionHistory')}</p>
                    )}
                </div>
            </DialogContent>
        </Dialog>
    );
}
