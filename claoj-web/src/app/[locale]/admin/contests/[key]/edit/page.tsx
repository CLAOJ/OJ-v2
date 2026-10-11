'use client';

import { useParams } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { adminContestApi } from '@/lib/adminApi';
import { contestToFormData, contestFormToUpdate } from '@/lib/adminForms';
import { type ContestFormData } from '@/components/admin/ContestForm';
import ContestForm from '@/components/admin/ContestForm';
import { Trophy, ArrowLeft, Lock, Unlock, Users } from 'lucide-react';
import { Link } from '@/navigation';
import { Button } from '@/components/ui/Button';
import { useState } from 'react';
import { useTranslations } from 'next-intl';

export default function EditContestPage() {
    const t = useTranslations('Admin');
    const params = useParams();
    const key = params.key as string;
    const queryClient = useQueryClient();
    const [lockTime, setLockTime] = useState('');

    const { data: contestData, isLoading: isLoadingContest } = useQuery({
        queryKey: ['admin-contest-detail', key],
        queryFn: async () => {
            const res = await adminContestApi.detail(key);
            return res.data;
        }
    });

    const updateMutation = useMutation({
        mutationFn: (data: ContestFormData) =>
            adminContestApi.update(key, contestFormToUpdate(data, contestData?.problems ?? [])),
        onSuccess: () => {
            toast.success(t('contests.saved'));
            queryClient.invalidateQueries({ queryKey: ['admin-contest-detail', key] });
        },
        onError: (err: unknown) => {
            const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? String(err);
            toast.error(t('contests.saveFailed', { error: msg }));
        },
    });

    const lockMutation = useMutation({
        mutationFn: async (lockedAfter: string | null) => {
            const res = await adminContestApi.lock(key, lockedAfter);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['admin-contest-detail', key] });
        }
    });

    const handleSubmit = async (data: ContestFormData) => {
        await updateMutation.mutateAsync(data);
    };

    const handleLock = async () => {
        if (!lockTime) return;
        const isoTime = new Date(lockTime).toISOString();
        await lockMutation.mutateAsync(isoTime);
        setLockTime('');
    };

    const handleUnlock = async () => {
        await lockMutation.mutateAsync(null);
    };

    const isLocked = contestData?.locked_after != null;
    const lockedAfter = contestData?.locked_after ? new Date(contestData.locked_after) : null;

    if (isLoadingContest || !contestData) {
        return (
            <div className="flex items-center justify-center py-12">
                <div className="text-muted-foreground">{t('contests.loadingContest')}</div>
            </div>
        );
    }

    const initialData = contestToFormData(contestData);
    const initialOptions = {
        problems: contestData.problems.map(p => ({ id: p.problem_id, label: p.code, detail: p.name })),
        author_ids: contestData.authors.map(u => ({ id: u.id, label: u.username })),
        curator_ids: contestData.curators.map(u => ({ id: u.id, label: u.username })),
        tester_ids: contestData.testers.map(u => ({ id: u.id, label: u.username })),
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center gap-4">
                <Link
                    href="/admin/contests"
                    className="p-2 hover:bg-muted rounded-xl transition-colors"
                >
                    <ArrowLeft size={20} />
                </Link>
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Trophy className="text-primary" size={32} />
                        {t('contests.editContestTitle')}
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        {contestData.name} ({key})
                    </p>
                </div>
                <Link
                    href={`/admin/contests/${key}/participations`}
                    className="ml-auto px-4 py-2 rounded-xl bg-muted hover:bg-muted/80 transition-colors flex items-center gap-2 font-medium"
                >
                    <Users size={18} />
                    {t('contests.participationsLink')}
                </Link>
            </div>

            {/* Submission Lock Section */}
            <div className="p-6 rounded-2xl border bg-card">
                <div className="flex items-center gap-3 mb-4">
                    {isLocked ? (
                        <Lock className="text-warning" size={24} />
                    ) : (
                        <Unlock className="text-success" size={24} />
                    )}
                    <h2 className="text-xl font-bold">{t('contests.submissionLockTitle')}</h2>
                </div>
                <p className="text-sm text-muted-foreground mb-4">
                    {t('contests.submissionLockDesc')}
                </p>
                {isLocked && lockedAfter && (
                    <div className="p-3 rounded-xl bg-warning/10 border border-warning/20 text-warning text-sm mb-4">
                        <p className="font-medium">{t('contests.currentlyLockedMsg')}</p>
                        <p>{t('contests.lockedAfterMsg', { time: lockedAfter.toLocaleString() })}</p>
                    </div>
                )}
                <div className="flex gap-3 items-end">
                    <div className="flex-1 space-y-2">
                        <label className="text-sm font-medium">{t('contests.lockAfterLabel')}</label>
                        <input
                            type="datetime-local"
                            value={lockTime}
                            onChange={(e) => setLockTime(e.target.value)}
                            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                        />
                    </div>
                    <Button
                        onClick={handleLock}
                        disabled={!lockTime || lockMutation.isPending}
                        loading={lockMutation.isPending}
                        variant="warning"
                    >
                        <Lock size={16} />
                        {t('contests.lockSubmissionsButton')}
                    </Button>
                    {isLocked && (
                        <Button
                            onClick={handleUnlock}
                            disabled={lockMutation.isPending}
                            loading={lockMutation.isPending}
                            variant="outline"
                        >
                            <Unlock size={16} />
                            {t('contests.unlockButton')}
                        </Button>
                    )}
                </div>
            </div>

            {/* Form */}
            <ContestForm
                initialData={initialData}
                initialOptions={initialOptions}
                onSubmit={handleSubmit}
                isLoading={updateMutation.isPending}
            />
        </div>
    );
}
