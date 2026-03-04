'use client';

import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { adminContestApi } from '@/lib/adminApi';
import { type ContestFormData } from '@/components/admin/ContestForm';
import ContestForm from '@/components/admin/ContestForm';
import { Trophy, ArrowLeft } from 'lucide-react';
import { Link } from '@/navigation';

export default function CreateContestPage() {
    const router = useRouter();

    const createMutation = useMutation({
        mutationFn: (data: ContestFormData) => adminContestApi.create(data),
        onSuccess: (response) => {
            if (response.data.contest?.key) {
                router.push(`/admin/contests`);
            }
        }
    });

    const handleSubmit = async (data: ContestFormData) => {
        await createMutation.mutateAsync(data);
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
                        Create Contest
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        Create a new contest with problems and settings
                    </p>
                </div>
            </div>

            {/* Form */}
            <ContestForm
                onSubmit={handleSubmit}
                isLoading={createMutation.isPending}
            />
        </div>
    );
}
