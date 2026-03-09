'use client';

import { useState } from 'react';
import {
    User, Clock, MoreVertical, ThumbsUp, Reply,
    ChevronUp, ChevronDown, Edit2, Trash2,
    History, Shield, Eye
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { motion, AnimatePresence } from 'framer-motion';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { Comment } from '@/types';
import { CommentActions } from './CommentActions';
import { CommentReplyForm } from './CommentReplyForm';

dayjs.extend(relativeTime);

export interface CommentNode extends Comment {
    children: CommentNode[];
}

interface CommentNodeProps {
    node: CommentNode;
    level?: number;
    onReply: (id: number | null) => void;
    replyTo: number | null;
    onPost: (body: string) => void;
    isPosting: boolean;
    user: { username: string } | null;
    isAdmin: boolean;
    t: (key: string) => string;
    onEdit: (comment: Comment | null) => void;
    onDelete: (id: number | null) => void;
    onShowRevisions: (comment: Comment | null) => void;
    onHide: (params: { id: number; hidden: boolean }) => void;
    onVote: (params: { id: number; score: number }) => void;
}

export default function CommentNode({
    node,
    level = 0,
    onReply,
    replyTo,
    onPost,
    isPosting,
    user,
    isAdmin,
    t,
    onEdit,
    onDelete,
    onShowRevisions,
    onHide,
    onVote
}: CommentNodeProps) {
    const [isExpanded, setIsExpanded] = useState(true);
    const [replyBody, setReplyBody] = useState('');
    const [voteDirection, setVoteDirection] = useState<number>(0);

    const isEditable = !!user && (node.author === user.username);
    const canModerate = isAdmin;

    const handleVote = () => {
        const newDirection = voteDirection === 1 ? 0 : 1;
        setVoteDirection(newDirection);
        onVote({ id: node.id, score: newDirection === 1 ? 1 : -1 });
    };

    const handleReply = () => {
        if (replyBody.trim()) {
            onPost(replyBody);
            setReplyBody('');
        }
    };

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

                    <CommentActions
                        node={node}
                        isEditable={isEditable}
                        canModerate={canModerate}
                        t={t}
                        onEdit={onEdit}
                        onDelete={onDelete}
                        onShowRevisions={onShowRevisions}
                        onHide={onHide}
                    />
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
                            onClick={handleVote}
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
                        <CommentReplyForm
                            replyBody={replyBody}
                            setReplyBody={setReplyBody}
                            authorName={node.author}
                            isPosting={isPosting}
                            onCancel={() => onReply(null)}
                            onReply={handleReply}
                            t={t}
                        />
                    )}
                </AnimatePresence>
            </div>

            {isExpanded && node.children.length > 0 && (
                <div className="space-y-8">
                    {node.children.map((child) => (
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
