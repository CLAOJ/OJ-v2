'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from '@/navigation';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { problemClarificationApi } from '@/lib/api';
import { ProblemClarification } from '@/types';
import { useState } from 'react';
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle
} from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Textarea } from '@/components/ui/Textarea';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger, DialogFooter, DialogDescription } from '@/components/ui/Dialog';
import { Badge } from '@/components/ui/Badge';
import {
    MessageSquare,
    Plus,
    Trash2,
    Clock,
    AlertCircle,
    CheckCircle
} from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';

export default function ProblemClarificationsPage() {
    const params = useParams();
    const router = useRouter();
    const t = useTranslations('admin');
    const tCommon = useTranslations('Common');
    const code = params.code as string;
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const [description, setDescription] = useState('');
    const queryClient = useQueryClient();

    const { data: clarifications, isLoading } = useQuery({
        queryKey: ['problem-clarifications', code],
        queryFn: async () => {
            const res = await problemClarificationApi.getClarifications(code);
            return res.data.data;
        }
    });

    const { mutate: createClarification, isPending } = useMutation({
        mutationFn: async () => {
            return await problemClarificationApi.createClarification(code, description);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-clarifications', code] });
            setIsCreateOpen(false);
            setDescription('');
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to create clarification');
        }
    });

    const { mutate: deleteClarification } = useMutation({
        mutationFn: async (id: number) => {
            return await problemClarificationApi.deleteClarification(id);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-clarifications', code] });
        },
        onError: (err: any) => {
            alert(err.response?.data?.error || 'Failed to delete clarification');
        }
    });

    const handleSubmit = () => {
        if (!description.trim()) {
            alert('Please enter a description');
            return;
        }
        createClarification();
    };

    return (
        <div className="container mx-auto py-8">
            <div className="flex items-center justify-between mb-8">
                <div>
                    <h1 className="text-3xl font-bold mb-2">{t('problem_clarifications')}</h1>
                    <p className="text-muted-foreground">{t('manage_problem_clarifications', { code })}</p>
                </div>
                <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
                    <DialogTrigger asChild>
                        <Button>
                            <Plus className="w-4 h-4 mr-2" />
                            {t('add_clarification')}
                        </Button>
                    </DialogTrigger>
                    <DialogContent>
                        <DialogHeader>
                            <DialogTitle>{t('add_problem_clarification')}</DialogTitle>
                            <DialogDescription>
                                {t('add_problem_clarification_desc', { code })}
                            </DialogDescription>
                        </DialogHeader>
                        <div className="py-4">
                            <Textarea
                                placeholder={t('clarification_description_placeholder')}
                                value={description}
                                onChange={(e) => setDescription(e.target.value)}
                                className="min-h-[150px]"
                            />
                        </div>
                        <DialogFooter>
                            <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                                {tCommon('cancel')}
                            </Button>
                            <Button onClick={handleSubmit} disabled={isPending}>
                                {isPending ? t('creating') : t('create_clarification')}
                            </Button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
            </div>

            {isLoading ? (
                <div className="space-y-4">
                    {[1, 2, 3].map((i) => (
                        <Card key={i} className="animate-pulse">
                            <CardHeader className="pb-3">
                                <div className="h-4 bg-muted rounded w-3/4" />
                            </CardHeader>
                            <CardContent>
                                <div className="h-3 bg-muted rounded w-1/4" />
                            </CardContent>
                        </Card>
                    ))}
                </div>
            ) : !clarifications || clarifications.length === 0 ? (
                <Card>
                    <CardContent className="flex items-center gap-4 py-8">
                        <AlertCircle className="w-12 h-12 text-muted-foreground" />
                        <div>
                            <p className="font-medium">{t('no_clarifications')}</p>
                            <p className="text-sm text-muted-foreground">
                                {t('be_first_clarification')}
                            </p>
                        </div>
                    </CardContent>
                </Card>
            ) : (
                <div className="space-y-4">
                    {clarifications.map((clar) => (
                        <Card key={clar.id}>
                            <CardHeader className="pb-3">
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-2">
                                        <MessageSquare className="w-5 h-5 text-primary" />
                                        <CardTitle className="text-lg">{t('clarification_number', { id: clar.id })}</CardTitle>
                                    </div>
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => deleteClarification(clar.id)}
                                    >
                                        <Trash2 className="w-4 h-4" />
                                    </Button>
                                </div>
                            </CardHeader>
                            <CardContent>
                                <p className="text-muted-foreground whitespace-pre-wrap mb-4">
                                    {clar.description}
                                </p>
                                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                                    <Clock className="w-4 h-4" />
                                    {formatDistanceToNow(new Date(clar.date), { addSuffix: true })}
                                </div>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            )}
        </div>
    );
}
