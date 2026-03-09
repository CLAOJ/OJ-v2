'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { adminProblemDataApi } from '@/lib/adminApi';
import { Check } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ConfigTabProps {
    code: string;
    data?: {
        checker?: string;
        has_custom_checker?: boolean;
        grader?: string;
        has_custom_grader?: boolean;
        feedback?: string;
        has_generator_yml?: boolean;
        has_init_yml?: boolean;
    } | null;
}

export function ConfigTab({ code, data }: ConfigTabProps) {
    const queryClient = useQueryClient();

    const updateMutation = useMutation({
        mutationFn: async (formData: FormData) => {
            return adminProblemDataApi.upload(code, formData);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['problem-data', code] });
            alert('Configuration updated successfully');
        }
    });

    return (
        <div className="space-y-6">
            {/* Checker Configuration */}
            <div className="p-6 border rounded-xl">
                <h3 className="font-semibold text-lg mb-4">Checker Configuration</h3>
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-2">
                            Checker Type
                        </label>
                        <div className="text-muted-foreground">{data?.checker || 'default'}</div>
                    </div>
                    {data?.has_custom_checker && (
                        <div className="p-3 bg-muted rounded-lg">
                            <div className="flex items-center gap-2 text-green-600">
                                <Check size={16} />
                                <span className="font-medium">Custom checker configured</span>
                            </div>
                        </div>
                    )}
                </div>
            </div>

            {/* Grader Configuration */}
            <div className="p-6 border rounded-xl">
                <h3 className="font-semibold text-lg mb-4">Grader Configuration</h3>
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-2">
                            Grader Type
                        </label>
                        <div className="text-muted-foreground">{data?.grader || 'default'}</div>
                    </div>
                    {data?.has_custom_grader && (
                        <div className="p-3 bg-muted rounded-lg">
                            <div className="flex items-center gap-2 text-green-600">
                                <Check size={16} />
                                <span className="font-medium">Custom grader configured</span>
                            </div>
                        </div>
                    )}
                </div>
            </div>

            {/* Other Settings */}
            <div className="p-6 border rounded-xl">
                <h3 className="font-semibold text-lg mb-4">Other Settings</h3>
                <div className="grid grid-cols-2 gap-4">
                    <div className="p-3 bg-muted rounded-lg">
                        <div className="text-sm text-muted-foreground">Feedback Level</div>
                        <div className="font-medium">{data?.feedback || 'default'}</div>
                    </div>
                    <div className={cn(
                        "p-3 rounded-lg flex items-center gap-2",
                        data?.has_generator_yml ? "bg-muted" : "bg-muted/50"
                    )}>
                        {data?.has_generator_yml ? (
                            <>
                                <Check size={16} className="text-green-600" />
                                <span className="font-medium">Generator configured</span>
                            </>
                        ) : (
                            <span className="text-muted-foreground">No generator</span>
                        )}
                    </div>
                    <div className={cn(
                        "p-3 rounded-lg flex items-center gap-2",
                        data?.has_init_yml ? "bg-muted" : "bg-muted/50"
                    )}>
                        {data?.has_init_yml ? (
                            <>
                                <Check size={16} className="text-green-600" />
                                <span className="font-medium">init.yml present</span>
                            </>
                        ) : (
                            <span className="text-muted-foreground">No init.yml</span>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
