'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Comment, PaginatedList } from '@/types';
import { Skeleton } from '@/components/ui/Skeleton';
import { useState } from 'react';
import { MessageSquare, BookOpen, FileText, Send, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useAuth } from '@/components/providers/AuthProvider';
import { useCommentMutations } from '@/hooks/useCommentMutations';
import { buildCommentTree, CommentNode } from '@/utils/commentTree';
import CommentNodeComponent from '@/components/comments/CommentNode';
import { CommentDialogs } from '@/components/comments/CommentDialogs';

interface CommentsProps {
    page: string;
    problemName?: string;
    contextType?: 'problem' | 'editorial';
}

export default function Comments({ page, problemName, contextType }: CommentsProps) {
    const t = useTranslations('Comments');
    const { user } = useAuth();
    const isAdmin = user?.is_admin || false;
    const [commentBody, setCommentBody] = useState('');
    const [replyTo, setReplyTo] = useState<number | null>(null);
    const [editingComment, setEditingComment] = useState<Comment | null>(null);
    const [revisionHistory, setRevisionHistory] = useState<Comment | null>(null);
    const [deletingComment, setDeletingComment] = useState<number | null>(null);

    const getTypeFromPage = (): 'problem' | 'editorial' | 'blog' => {
        if (contextType) return contextType;
        if (page.startsWith('e/')) return 'editorial';
        if (page.startsWith('p/')) return 'problem';
        if (page.startsWith('blog/')) return 'blog';
        return 'problem';
    };

    const commentType = getTypeFromPage();
    const contextTranslations = useTranslations(
        commentType === 'editorial' ? 'editorialComments' : commentType === 'problem' ? 'problemComments' : 'Comments'
    );

    const { data: comments, isLoading } = useQuery({
        queryKey: ['comments', page],
        queryFn: async () => {
            const res = await api.get<PaginatedList<Comment>>(`/comments?page=${page}`);
            return res.data.data;
        }
    });

    const {
        postComment,
        voteComment,
        editComment,
        deleteComment,
        hideComment
    } = useCommentMutations(page);

    const handlePost = (body: string) => {
        if (!body.trim()) return;
        postComment.mutate({ body, parent_id: replyTo });
    };

    const ContextHeader = () => {
        if (commentType !== 'problem' && commentType !== 'editorial') return null;

        return (
            <div className={cn(
                "p-4 rounded-2xl border flex items-start gap-3",
                commentType === 'editorial'
                    ? "bg-emerald-50/50 border-emerald-200 dark:bg-emerald-900/20 dark:border-emerald-800"
                    : "bg-blue-50/50 border-blue-200 dark:bg-blue-900/20 dark:border-blue-800"
            )}>
                <div className={cn(
                    "p-2 rounded-lg",
                    commentType === 'editorial'
                        ? "bg-emerald-100 dark:bg-emerald-900/40 text-emerald-600 dark:text-emerald-400"
                        : "bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400"
                )}>
                    {commentType === 'editorial' ? <BookOpen size={18} /> : <FileText size={18} />}
                </div>
                <div className="flex-1">
                    <div className="text-sm font-bold text-foreground">
                        {contextTranslations('commentingOn')}
                    </div>
                    {problemName && (
                        <div className="text-xs text-muted-foreground mt-0.5">
                            {contextTranslations('problemName', { problemName })}
                        </div>
                    )}
                </div>
            </div>
        );
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

    const tree = buildCommentTree(comments || []);

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

            {user ? (
                <div className="space-y-4">
                    <ContextHeader />
                    <div className="p-8 rounded-[2rem] bg-card border border-dashed hover:border-primary/50 transition-colors space-y-4">
                        <textarea
                            value={commentBody}
                            onChange={(e) => setCommentBody(e.target.value)}
                            placeholder={contextTranslations('placeholder')}
                            className="w-full h-32 p-6 rounded-2xl bg-muted/30 border outline-none focus:ring-4 focus:ring-primary/10 transition-all resize-none text-base font-medium"
                        />
                        <div className="flex justify-end">
                            <button
                                onClick={() => handlePost(commentBody)}
                                disabled={postComment.isPending || !commentBody.trim()}
                                className="flex items-center gap-2 h-12 px-8 rounded-xl bg-primary text-primary-foreground text-sm font-black shadow-xl shadow-primary/20 hover:opacity-90 active:scale-95 transition-all disabled:opacity-50"
                            >
                                {postComment.isPending ? <Loader2 className="animate-spin" size={18} /> : <Send size={18} />}
                                {t('post')}
                            </button>
                        </div>
                    </div>
                </div>
            ) : (
                <div className="p-10 text-center rounded-[2rem] bg-muted/30 border border-dashed">
                    <p className="text-sm font-bold text-muted-foreground">{t('signInToComment')}</p>
                </div>
            )}

            <div className="space-y-10 pt-4">
                {tree.length === 0 ? (
                    <div className="py-20 text-center text-muted-foreground/50 flex flex-col items-center gap-4">
                        <MessageSquare size={48} className="opacity-10" />
                        <p className="font-bold">{contextTranslations('noComments')}</p>
                    </div>
                ) : (
                    tree.map(node => (
                        <CommentNodeComponent
                            key={node.id}
                            node={node}
                            user={user}
                            isAdmin={isAdmin}
                            t={t}
                            onReply={setReplyTo}
                            replyTo={replyTo}
                            onPost={handlePost}
                            isPosting={postComment.isPending}
                            onEdit={setEditingComment}
                            onDelete={setDeletingComment}
                            onShowRevisions={setRevisionHistory}
                            onHide={(params) => hideComment.mutate(params)}
                            onVote={(params) => voteComment.mutate(params)}
                        />
                    ))
                )}
            </div>

            <CommentDialogs
                editingComment={editingComment}
                deletingComment={deletingComment}
                revisionHistory={revisionHistory}
                onEditCancel={() => setEditingComment(null)}
                onEditSubmit={(id, body, reason) => editComment.mutate({ id, body, reason })}
                onDeleteCancel={() => setDeletingComment(null)}
                onDeleteConfirm={(id) => deleteComment.mutate(id)}
                onRevisionClose={() => setRevisionHistory(null)}
                isEditing={editComment.isPending}
                t={t}
            />
        </div>
    );
}
