'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Comment } from '@/types';

export function useCommentMutations(page: string) {
    const queryClient = useQueryClient();

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
        }
    });

    const voteComment = useMutation({
        mutationFn: async ({ id, score }: { id: number; score: number }) => {
            await api.post(`/comment/${id}/vote`, { score });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        }
    });

    const editComment = useMutation({
        mutationFn: async ({ id, body, reason }: { id: number; body: string; reason?: string }) => {
            await api.patch(`/comment/${id}`, { body, reason });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        }
    });

    const deleteComment = useMutation({
        mutationFn: async (id: number) => {
            await api.delete(`/comment/${id}`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        }
    });

    const hideComment = useMutation({
        mutationFn: async ({ id, hidden }: { id: number; hidden: boolean }) => {
            await api.post(`/admin/comment/${id}/hide`, { hidden });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', page] });
        }
    });

    return {
        postComment,
        voteComment,
        editComment,
        deleteComment,
        hideComment
    };
}
