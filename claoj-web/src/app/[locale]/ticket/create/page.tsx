'use client';

import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Problem } from '@/types';
import { Link, useRouter } from '@/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import {
    Ticket as TicketIcon,
    ArrowLeft,
    AlertCircle,
    FileText,
    MessageSquare,
    Loader2
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useAuth } from '@/components/providers/AuthProvider';

const ticketSchema = z.object({
    title: z.string().min(5, 'Title must be at least 5 characters'),
    body: z.string().min(20, 'Description must be at least 20 characters'),
    problem_code: z.string().optional(),
});

type TicketFormValues = z.infer<typeof ticketSchema>;

export default function CreateTicketPage() {
    const t = useTranslations('Tickets');
    const router = useRouter();
    const { user, loading } = useAuth();
    const [problemSearch, setProblemSearch] = useState('');
    const [selectedProblem, setSelectedProblem] = useState<Problem | null>(null);

    const isAuthenticated = !!user;

    const { data: problems } = useQuery({
        queryKey: ['problems-search', problemSearch],
        queryFn: async () => {
            if (!problemSearch) return [];
            const res = await api.get<{ items: Problem[] }>(`/problems?search=${problemSearch}&page_size=5`);
            return res.data.items;
        },
        enabled: !!problemSearch
    });

    const createMutation = useMutation({
        mutationFn: async (data: TicketFormValues) => {
            const payload = {
                title: data.title,
                body: data.body,
                problem_code: data.problem_code || undefined,
            };
            const res = await api.post('/tickets', payload);
            return res.data;
        },
        onSuccess: (data) => {
            router.push(`/ticket/${data.data?.id || data.id}`);
        }
    });

    const {
        register,
        handleSubmit,
        formState: { errors },
        setValue,
    } = useForm<TicketFormValues>({
        resolver: zodResolver(ticketSchema),
    });

    if (!isAuthenticated) {
        router.push('/login');
        return null;
    }

    const onSubmit = async (data: TicketFormValues) => {
        createMutation.mutate(data);
    };

    const handleSelectProblem = (problem: Problem) => {
        setSelectedProblem(problem);
        setValue('problem_code', problem.code);
        setProblemSearch('');
    };

    return (
        <div className="max-w-3xl mx-auto space-y-8 animate-in fade-in duration-700 mt-4 pb-20">
            {/* Back Button */}
            <Link
                href="/tickets"
                className="inline-flex items-center gap-2 text-sm font-bold text-muted-foreground hover:text-primary transition-colors"
            >
                <ArrowLeft size={16} />
                Back to Tickets
            </Link>

            {/* Header */}
            <div className="text-center space-y-2">
                <TicketIcon className="mx-auto text-primary" size={48} />
                <h1 className="text-4xl font-black tracking-tighter">Create New Ticket</h1>
                <p className="text-muted-foreground font-black opacity-80">Describe your issue and we&apos;ll get back to you.</p>
            </div>

            {/* Form */}
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
                <div className="bg-card border rounded-[2.5rem] p-8 shadow-sm space-y-6">
                    {/* Title */}
                    <div className="space-y-2">
                        <label className="text-sm font-bold ml-1 flex items-center gap-2">
                            <MessageSquare size={16} className="text-primary" />
                            Title
                        </label>
                        <input
                            {...register('title')}
                            type="text"
                            placeholder="Brief summary of your issue"
                            className={cn(
                                "w-full h-14 bg-muted/30 border rounded-2xl px-4 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:border-primary/30 transition-all outline-none",
                                errors.title ? "border-destructive" : "border-muted-foreground/10"
                            )}
                        />
                        {errors.title && <p className="text-xs text-destructive ml-2">{errors.title.message}</p>}
                    </div>

                    {/* Problem Selection */}
                    <div className="space-y-2">
                        <label className="text-sm font-bold ml-1 flex items-center gap-2">
                            <AlertCircle size={16} className="text-primary" />
                            Related Problem (Optional)
                        </label>
                        <div className="relative">
                            <input
                                type="text"
                                placeholder="Search for a problem (e.g., P01, APB)"
                                className="w-full h-14 bg-muted/30 border border-muted-foreground/10 rounded-2xl px-4 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:border-primary/30 transition-all outline-none"
                                value={selectedProblem ? `${selectedProblem.code} - ${selectedProblem.name}` : problemSearch}
                                onChange={(e) => {
                                    setProblemSearch(e.target.value);
                                    setSelectedProblem(null);
                                    setValue('problem_code', '');
                                }}
                                onFocus={() => problemSearch && setSelectedProblem(null)}
                            />
                            {selectedProblem && (
                                <button
                                    type="button"
                                    onClick={() => {
                                        setSelectedProblem(null);
                                        setValue('problem_code', '');
                                    }}
                                    className="absolute right-4 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-destructive transition-colors"
                                >
                                    ×
                                </button>
                            )}
                        </div>

                        {/* Problem Search Results */}
                        {problemSearch && problems && problems.length > 0 && (
                            <div className="mt-2 bg-card border rounded-2xl overflow-hidden shadow-lg">
                                {problems.map((problem) => (
                                    <button
                                        key={problem.code}
                                        type="button"
                                        onClick={() => handleSelectProblem(problem)}
                                        className="w-full p-4 text-left hover:bg-muted/50 transition-colors flex items-center justify-between border-b last:border-b-0"
                                    >
                                        <div>
                                            <span className="font-mono font-black text-primary">{problem.code}</span>
                                            <span className="ml-2 font-bold">{problem.name}</span>
                                        </div>
                                        <span className="text-xs font-bold text-muted-foreground">{problem.points} pts</span>
                                    </button>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Description */}
                    <div className="space-y-2">
                        <label className="text-sm font-bold ml-1 flex items-center gap-2">
                            <FileText size={16} className="text-primary" />
                            Description
                        </label>
                        <textarea
                            {...register('body')}
                            placeholder="Describe your issue in detail. Include steps to reproduce, expected behavior, and any relevant information."
                            rows={8}
                            className={cn(
                                "w-full bg-muted/30 border rounded-2xl px-4 py-3 text-sm font-bold focus:ring-2 focus:ring-primary/20 focus:border-primary/30 transition-all outline-none resize-none",
                                errors.body ? "border-destructive" : "border-muted-foreground/10"
                            )}
                        />
                        {errors.body && <p className="text-xs text-destructive ml-2">{errors.body.message}</p>}
                    </div>
                </div>

                {/* Submit Button */}
                <div className="flex justify-end gap-4">
                    <Link
                        href="/tickets"
                        className="px-8 h-14 rounded-2xl bg-muted text-muted-foreground font-bold hover:bg-muted/80 transition-all"
                    >
                        Cancel
                    </Link>
                    <button
                        type="submit"
                        disabled={createMutation.isPending}
                        className="px-8 h-14 rounded-2xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-all flex items-center gap-2 disabled:opacity-50 shadow-lg shadow-primary/20"
                    >
                        {createMutation.isPending && <Loader2 size={18} className="animate-spin" />}
                        Create Ticket
                    </button>
                </div>
            </form>
        </div>
    );
}
