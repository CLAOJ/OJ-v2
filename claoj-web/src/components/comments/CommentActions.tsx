'use client';

import { MoreVertical, Edit2, Trash2, History, Shield } from 'lucide-react';
import { useState } from 'react';
import { CommentNode } from './CommentNode';

interface CommentActionsProps {
    node: CommentNode;
    isEditable: boolean;
    canModerate: boolean;
    t: (key: string) => string;
    onEdit: (comment: CommentNode | null) => void;
    onDelete: (id: number | null) => void;
    onShowRevisions: (comment: CommentNode | null) => void;
    onHide: (params: { id: number; hidden: boolean }) => void;
}

export function CommentActions({
    node,
    isEditable,
    canModerate,
    t,
    onEdit,
    onDelete,
    onShowRevisions,
    onHide
}: CommentActionsProps) {
    const [showMenu, setShowMenu] = useState(false);

    return (
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
                        onClick={() => { onShowRevisions(node); setShowMenu(false); }}
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
    );
}
