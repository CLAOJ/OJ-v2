'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Badge } from '@/components/ui/Badge';
import { MessageSquare, MessageSquarePlus, ChevronDown, ChevronUp, Send, Loader2 } from 'lucide-react';

export interface ContestClarification {
    id: number;
    question: string;
    answer?: string;
    is_answered: boolean;
    create_time: string;
    author?: {
        username: string;
    };
}

interface ContestClarificationListProps {
    contestKey: string;
    canCreate?: boolean;
    isAdmin?: boolean;
    onClarificationCreated?: () => void;
}

export function ContestClarificationList({ contestKey, canCreate, isAdmin, onClarificationCreated }: ContestClarificationListProps) {
    const [showCreateForm, setShowCreateForm] = useState(false);
    const [expandedId, setExpandedId] = useState<number | null>(null);
    const [newQuestion, setNewQuestion] = useState('');
    const [answerText, setAnswerText] = useState('');
    const [isPublic, setIsPublic] = useState(true);
    const queryClient = useQueryClient();

    const { data: clarifications, isLoading } = useQuery({
        queryKey: ['contest-clarifications', contestKey],
        queryFn: async () => {
            const res = await api.get<{ data: ContestClarification[] }>(
                `/contest/${contestKey}/clarifications`
            );
            return res.data.data || [];
        }
    });

    const createMutation = useMutation({
        mutationFn: async (question: string) => {
            await api.post(`/contest/${contestKey}/clarifications`, { question });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['contest-clarifications', contestKey] });
            setShowCreateForm(false);
            setNewQuestion('');
            onClarificationCreated?.();
        }
    });

    const answerMutation = useMutation({
        mutationFn: async ({ id, answer, isPublic }: { id: number; answer: string; isPublic: boolean }) => {
            await api.post(`/contest/${contestKey}/clarification/${id}/answer`, {
                answer,
                is_public: isPublic
            });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['contest-clarifications', contestKey] });
            setAnswerText('');
            setIsPublic(true);
        }
    });

    const handleCreate = () => {
        if (newQuestion.trim()) {
            createMutation.mutate(newQuestion.trim());
        }
    };

    const handleAnswer = (id: number) => {
        if (answerText.trim()) {
            answerMutation.mutate({ id, answer: answerText.trim(), isPublic });
        }
    };

    if (isLoading) {
        return (
            <div className="space-y-2">
                {[1, 2, 3].map(i => (
                    <div key={i} className="h-16 bg-muted/50 rounded-xl animate-pulse" />
                ))}
            </div>
        );
    }

    return (
        <div className="space-y-4">
            {canCreate && (
                <div className="flex justify-end">
                    <button
                        onClick={() => setShowCreateForm(!showCreateForm)}
                        className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2"
                    >
                        <MessageSquarePlus size={18} />
                        Ask Clarification
                    </button>
                </div>
            )}

            {showCreateForm && (
                <div className="bg-card border rounded-xl p-4 space-y-3">
                    <h3 className="font-bold">New Clarification Question</h3>
                    <textarea
                        className="w-full p-3 rounded-xl bg-muted border min-h-[100px] focus:ring-2 focus:ring-primary/20 outline-none resize-none"
                        placeholder="Enter your question about a problem..."
                        value={newQuestion}
                        onChange={(e) => setNewQuestion(e.target.value)}
                        disabled={createMutation.isPending}
                    />
                    <div className="flex justify-end gap-2">
                        <button
                            onClick={() => {
                                setShowCreateForm(false);
                                setNewQuestion('');
                            }}
                            className="px-4 py-2 rounded-xl hover:bg-muted transition-colors"
                        >
                            Cancel
                        </button>
                        <button
                            onClick={handleCreate}
                            disabled={!newQuestion.trim() || createMutation.isPending}
                            className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 disabled:opacity-50 transition-colors"
                        >
                            Submit Question
                        </button>
                    </div>
                </div>
            )}

            <div className="space-y-2">
                {!clarifications || clarifications.length === 0 ? (
                    <div className="text-center py-8 text-muted-foreground">
                        <MessageSquare className="mx-auto mb-2 opacity-50" size={48} />
                        <p>No clarifications yet</p>
                    </div>
                ) : (
                    clarifications.map((clarification) => (
                        <div
                            key={clarification.id}
                            className="bg-card border rounded-xl overflow-hidden"
                        >
                            <button
                                onClick={() => setExpandedId(expandedId === clarification.id ? null : clarification.id)}
                                className="w-full p-4 flex items-center justify-between hover:bg-muted/30 transition-colors"
                            >
                                <div className="flex items-center gap-3">
                                    <Badge variant={clarification.is_answered ? 'success' : 'warning'}>
                                        {clarification.is_answered ? 'Answered' : 'Pending'}
                                    </Badge>
                                    <span className="text-muted-foreground text-sm">
                                        {new Date(clarification.create_time).toLocaleString()}
                                    </span>
                                    {clarification.author && (
                                        <span className="text-muted-foreground text-sm">
                                            by {clarification.author.username}
                                        </span>
                                    )}
                                </div>
                                {expandedId === clarification.id ? (
                                    <ChevronUp size={20} className="text-muted-foreground" />
                                ) : (
                                    <ChevronDown size={20} className="text-muted-foreground" />
                                )}
                            </button>

                            {expandedId === clarification.id && (
                                <div className="px-4 pb-4 border-t">
                                    <div className="mt-4 space-y-4">
                                        <div>
                                            <div className="text-xs text-muted-foreground mb-1">Question:</div>
                                            <p className="text-sm">{clarification.question}</p>
                                        </div>
                                        {clarification.answer && (
                                            <div>
                                                <div className="text-xs text-muted-foreground mb-1">Answer:</div>
                                                <p className="text-sm font-medium bg-success/10 p-3 rounded-lg">
                                                    {clarification.answer}
                                                </p>
                                            </div>
                                        )}
                                        {isAdmin && !clarification.is_answered && (
                                            <div className="mt-4 p-4 bg-muted rounded-xl space-y-3">
                                                <h4 className="font-bold text-sm flex items-center gap-2">
                                                    <Send size={16} />
                                                    Answer Clarification
                                                </h4>
                                                <textarea
                                                    className="w-full p-3 rounded-xl bg-background border min-h-[80px] focus:ring-2 focus:ring-primary/20 outline-none resize-none"
                                                    placeholder="Enter your answer..."
                                                    value={answerText}
                                                    onChange={(e) => setAnswerText(e.target.value)}
                                                    disabled={answerMutation.isPending}
                                                />
                                                <div className="flex items-center gap-2">
                                                    <label className="flex items-center gap-2 text-sm">
                                                        <input
                                                            type="checkbox"
                                                            checked={isPublic}
                                                            onChange={(e) => setIsPublic(e.target.checked)}
                                                            className="rounded"
                                                        />
                                                        Make answer public (visible to all contestants)
                                                    </label>
                                                </div>
                                                <div className="flex justify-end">
                                                    <button
                                                        onClick={() => handleAnswer(clarification.id)}
                                                        disabled={!answerText.trim() || answerMutation.isPending}
                                                        className="px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 disabled:opacity-50 transition-colors flex items-center gap-2"
                                                    >
                                                        {answerMutation.isPending && <Loader2 size={16} className="animate-spin" />}
                                                        Submit Answer
                                                    </button>
                                                </div>
                                            </div>
                                        )}
                                    </div>
                                </div>
                            )}
                        </div>
                    ))
                )}
            </div>
        </div>
    );
}

export default ContestClarificationList;
