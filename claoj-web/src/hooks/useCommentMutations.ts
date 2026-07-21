'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import api from '@/lib/api';
import { Comment } from '@/types';

export function useCommentMutations(page: string) {
    const queryClient = useQueryClient();
    const t = useTranslations('Common');

    // Every one of these mutations used to declare onSuccess only, so a
    // rejected request (a 401 for a signed-out visitor, a 403, a 500) was
    // absorbed by React Query and the click produced no visible effect at all.
    const onError = (err: unknown) => {
        const e = err as { response?: { data?: { error?: string } } };
        toast.error(e.response?.data?.error || t('error'));
    };

    const postComment = useMutation({
        mutationFn: async ({ body, parent_id }: { body: string; parent_id?: number | null }) => {
            await api.post('/comments', {
                page,
                body,
                parent_id
            });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        },
        onError
    });

    const voteComment = useMutation({
        mutationFn: async ({ id, score }: { id: number; score: number }) => {
            await api.post(`/comment/${id}/vote`, { score });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        },
        onError
    });

    const editComment = useMutation({
        mutationFn: async ({ id, body, reason }: { id: number; body: string; reason?: string }) => {
            await api.patch(`/comment/${id}`, { body, reason });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        },
        onError
    });

    const deleteComment = useMutation({
        mutationFn: async (id: number) => {
            await api.delete(`/comment/${id}`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        },
        onError
    });

    const hideComment = useMutation({
        mutationFn: async ({ id, hidden }: { id: number; hidden: boolean }) => {
            await api.post(`/admin/comment/${id}/hide`, { hidden });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        },
        onError
    });

    return {
        postComment,
        voteComment,
        editComment,
        deleteComment,
        hideComment
    };
}
