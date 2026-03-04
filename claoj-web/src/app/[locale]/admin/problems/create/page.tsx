'use client';

import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { adminProblemApi } from '@/lib/adminApi';
import { type ProblemFormData } from '@/components/admin/ProblemForm';
import ProblemForm from '@/components/admin/ProblemForm';
import { Code2, ArrowLeft } from 'lucide-react';
import { Link } from '@/navigation';

export default function CreateProblemPage() {
    const router = useRouter();

    const createMutation = useMutation({
        mutationFn: (data: ProblemFormData) => adminProblemApi.create(data),
        onSuccess: (response) => {
            if (response.data.problem?.code) {
                router.push(`/admin/problems`);
            }
        }
    });

    const handleSubmit = async (data: ProblemFormData) => {
        await createMutation.mutateAsync(data);
    };

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
                        Create Problem
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        Create a new problem with test cases
                    </p>
                </div>
            </div>

            {/* Form */}
            <ProblemForm
                onSubmit={handleSubmit}
                isLoading={createMutation.isPending}
            />
        </div>
    );
}
