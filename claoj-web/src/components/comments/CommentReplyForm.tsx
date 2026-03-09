'use client';

import { Send, Loader2 } from 'lucide-react';
import { Dispatch, SetStateAction } from 'react';
import { motion } from 'framer-motion';

interface CommentReplyFormProps {
    replyBody: string;
    setReplyBody: Dispatch<SetStateAction<string>>;
    authorName: string;
    isPosting: boolean;
    onCancel: () => void;
    onReply: () => void;
    t: (key: string) => string;
}

export function CommentReplyForm({
    replyBody,
    setReplyBody,
    authorName,
    isPosting,
    onCancel,
    onReply,
    t
}: CommentReplyFormProps) {
    return (
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
                placeholder={`${t('reply')} @${authorName}...`}
                className="w-full h-24 p-5 rounded-2xl bg-muted border outline-none focus:ring-4 focus:ring-primary/10 transition-all resize-none text-sm font-medium"
            />
            <div className="flex justify-end gap-3">
                <button
                    onClick={onCancel}
                    className="px-6 h-10 rounded-xl text-xs font-black uppercase tracking-widest hover:bg-muted transition-colors"
                >
                    {t('cancel')}
                </button>
                <button
                    onClick={onReply}
                    disabled={isPosting || !replyBody.trim()}
                    className="px-8 h-10 rounded-xl bg-primary text-primary-foreground text-xs font-black uppercase tracking-widest shadow-xl shadow-primary/20 hover:opacity-90 transition-all disabled:opacity-50"
                >
                    {isPosting ? <Loader2 className="animate-spin mr-2" size={16} /> : <Send className="mr-2" size={16} />}
                    {t('post')}
                </button>
            </div>
        </motion.div>
    );
}
