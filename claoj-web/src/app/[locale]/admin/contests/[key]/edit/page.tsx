'use client';

import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation } from '@tanstack/react-query';
import { adminContestApi } from '@/lib/adminApi';
import { type ContestFormData } from '@/components/admin/ContestForm';
import ContestForm from '@/components/admin/ContestForm';
import { Trophy, ArrowLeft } from 'lucide-react';
import { Link } from '@/navigation';

export default function EditContestPage() {
    const params = useParams();
    const router = useRouter();
    const key = params.key as string;

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

    const handleSubmit = async (data: ContestFormData) => {
        await updateMutation.mutateAsync({ key, data });
    };

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
        author_ids: contest.author_ids || [],
        curator_ids: contest.curator_ids || [],
        tester_ids: contest.tester_ids || [],
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
