'use client';

import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adminContestApi } from '@/lib/adminApi';
import { type ContestFormData } from '@/components/admin/ContestForm';
import ContestForm from '@/components/admin/ContestForm';
import { Trophy, ArrowLeft, Lock, Unlock, Users } from 'lucide-react';
import { Link } from '@/navigation';
import { Button } from '@/components/ui/Button';
import { useState } from 'react';

export default function EditContestPage() {
    const params = useParams();
    const router = useRouter();
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
        mutationFn: ({ key, data }: { key: string; data: ContestFormData }) =>
            adminContestApi.update(key, data),
        onSuccess: () => {
            router.push(`/admin/contests`);
        }
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
        await updateMutation.mutateAsync({ key, data });
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

    if (isLoadingContest) {
        return (
            <div className="flex items-center justify-center py-12">
                <div className="text-muted-foreground">Loading contest...</div>
            </div>
        );
    }

    // contestData is already the contest object with problems array
    const contest = contestData;
    const problems = contestData?.problems || [];

    const initialData = contest ? {
        key: contest.key,
        name: contest.name,
        description: contest.description,
        summary: contest.summary,
        start_time: formatDateTimeLocal(contest.start_time),
        end_time: formatDateTimeLocal(contest.end_time),
        time_limit: contest.time_limit || undefined,
        is_visible: contest.is_visible,
        is_rated: contest.is_rated,
        format_name: contest.format_name || 'icpc',
        format_config: typeof contest.format_config === 'object'
            ? JSON.stringify(contest.format_config)
            : contest.format_config,
        access_code: contest.access_code || '',
        hide_problem_tags: contest.hide_problem_tags,
        run_pretests_only: contest.run_pretests_only,
        is_organization_private: contest.is_organization_private,
        max_submissions: contest.max_submissions || undefined,
        author_ids: contest.author_ids || [],
        curator_ids: contest.curator_ids || [],
        tester_ids: contest.tester_ids || [],
        tag_ids: contest.tag_ids || [],
        problem_ids: problems.map((p: any) => p.id),
    } : undefined;

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
                        Edit Contest
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        {contest?.name} ({key})
                    </p>
                </div>
                <Link
                    href={`/admin/contests/${key}/participations`}
                    className="ml-auto px-4 py-2 rounded-xl bg-muted hover:bg-muted/80 transition-colors flex items-center gap-2 font-medium"
                >
                    <Users size={18} />
                    Participations
                </Link>
            </div>

            {/* Submission Lock Section */}
            <div className="p-6 rounded-2xl border bg-card">
                <div className="flex items-center gap-3 mb-4">
                    {isLocked ? (
                        <Lock className="text-amber-500" size={24} />
                    ) : (
                        <Unlock className="text-emerald-500" size={24} />
                    )}
                    <h2 className="text-xl font-bold">Submission Lock</h2>
                </div>
                <p className="text-sm text-muted-foreground mb-4">
                    Lock submissions after a specific time. Once locked, users cannot submit to this contest.
                </p>
                {isLocked && lockedAfter && (
                    <div className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-600 text-sm mb-4">
                        <p className="font-medium">Contest is currently locked</p>
                        <p>Submissions locked after: {lockedAfter.toLocaleString()}</p>
                    </div>
                )}
                <div className="flex gap-3 items-end">
                    <div className="flex-1 space-y-2">
                        <label className="text-sm font-medium">Lock After</label>
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
                        Lock Submissions
                    </Button>
                    {isLocked && (
                        <Button
                            onClick={handleUnlock}
                            disabled={lockMutation.isPending}
                            loading={lockMutation.isPending}
                            variant="outline"
                        >
                            <Unlock size={16} />
                            Unlock
                        </Button>
                    )}
                </div>
            </div>

            {/* Form */}
            <ContestForm
                initialData={initialData}
                onSubmit={handleSubmit}
                isLoading={updateMutation.isPending}
            />
        </div>
    );
}

function formatDateTimeLocal(dateString: string): string {
    if (!dateString) return '';
    const date = new Date(dateString);
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${year}-${month}-${day}T${hours}:${minutes}`;
}
