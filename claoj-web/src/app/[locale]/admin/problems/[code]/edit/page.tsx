'use client';

import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation } from '@tanstack/react-query';
import { adminProblemApi } from '@/lib/adminApi';
import { type ProblemFormData } from '@/components/admin/ProblemForm';
import ProblemForm from '@/components/admin/ProblemForm';
import { Code2, ArrowLeft, Database } from 'lucide-react';
import { Link } from '@/navigation';

export default function EditProblemPage() {
    const params = useParams();
    const router = useRouter();
    const code = params.code as string;

    const { data: problemData, isLoading: isLoadingProblem } = useQuery({
        queryKey: ['admin-problem-detail', code],
        queryFn: async () => {
            const res = await adminProblemApi.detail(code);
            return res.data;
        }
    });

    const updateMutation = useMutation({
        mutationFn: ({ code, data }: { code: string; data: ProblemFormData }) =>
            adminProblemApi.update(code, data),
        onSuccess: () => {
            router.push(`/admin/problems`);
        }
    });

    const handleSubmit = async (data: ProblemFormData) => {
        await updateMutation.mutateAsync({ code, data });
    };

    if (isLoadingProblem) {
        return (
            <div className="flex items-center justify-center py-12">
                <div className="text-muted-foreground">Loading problem...</div>
            </div>
        );
    }

    const initialData = problemData ? {
        code: problemData.code,
        name: problemData.name,
        description: problemData.description,
        points: problemData.points,
        partial: problemData.partial,
        is_public: problemData.is_public,
        time_limit: problemData.time_limit,
        memory_limit: problemData.memory_limit,
        group_id: problemData.group_id,
        type_ids: problemData.types?.map((t: any) => t.id),
        author_ids: problemData.authors?.map((a: any) => a.id),
        allowed_lang_ids: problemData.allowed_langs?.map((l: any) => l.id),
        is_manually_managed: problemData.is_manually_managed,
        pdf_url: problemData.pdf_url,
    } : undefined;

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center gap-4">
                <Link
                    href="/admin/problems"
                    className="p-2 hover:bg-muted rounded-xl transition-colors"
                >
                    <ArrowLeft size={20} />
                </Link>
                <div>
                    <h1 className="text-3xl font-bold flex items-center gap-3">
                        <Code2 className="text-primary" size={32} />
                        Edit Problem
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        {problemData?.name} ({code})
                    </p>
                </div>
                <Link
                    href={`/admin/problems/${code}/data`}
                    className="ml-auto px-4 py-2 rounded-xl bg-primary text-white hover:bg-primary/90 transition-colors flex items-center gap-2 font-medium"
                >
                    <Database size={18} />
                    Manage Data
                </Link>
            </div>

            {/* Form */}
            <ProblemForm
                initialData={initialData}
                onSubmit={handleSubmit}
                isLoading={updateMutation.isPending}
            />
        </div>
    );
}
