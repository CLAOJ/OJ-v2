'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
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
    const t = useTranslations('Admin.problemData');
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
                <h3 className="font-semibold text-lg mb-4">{t('checkerConfigTitle')}</h3>
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-2">
                            {t('checkerTypeLabel')}
                        </label>
                        <div className="text-muted-foreground">{data?.checker || 'default'}</div>
                    </div>
                    {data?.has_custom_checker && (
                        <div className="p-3 bg-muted rounded-lg">
                            <div className="flex items-center gap-2 text-green-600">
                                <Check size={16} />
                                <span className="font-medium">{t('customCheckerConfigured')}</span>
                            </div>
                        </div>
                    )}
                </div>
            </div>

            {/* Grader Configuration */}
            <div className="p-6 border rounded-xl">
                <h3 className="font-semibold text-lg mb-4">{t('graderConfigTitle')}</h3>
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-2">
                            {t('graderTypeLabel')}
                        </label>
                        <div className="text-muted-foreground">{data?.grader || 'default'}</div>
                    </div>
                    {data?.has_custom_grader && (
                        <div className="p-3 bg-muted rounded-lg">
                            <div className="flex items-center gap-2 text-green-600">
                                <Check size={16} />
                                <span className="font-medium">{t('customGraderConfigured')}</span>
                            </div>
                        </div>
                    )}
                </div>
            </div>

            {/* Other Settings */}
            <div className="p-6 border rounded-xl">
                <h3 className="font-semibold text-lg mb-4">{t('otherSettingsTitle')}</h3>
                <div className="grid grid-cols-2 gap-4">
                    <div className="p-3 bg-muted rounded-lg">
                        <div className="text-sm text-muted-foreground">{t('feedbackLevelLabel')}</div>
                        <div className="font-medium">{data?.feedback || 'default'}</div>
                    </div>
                    <div className={cn(
                        "p-3 rounded-lg flex items-center gap-2",
                        data?.has_generator_yml ? "bg-muted" : "bg-muted/50"
                    )}>
                        {data?.has_generator_yml ? (
                            <>
                                <Check size={16} className="text-green-600" />
                                <span className="font-medium">{t('generatorConfigured')}</span>
                            </>
                        ) : (
                            <span className="text-muted-foreground">{t('noGenerator')}</span>
                        )}
                    </div>
                    <div className={cn(
                        "p-3 rounded-lg flex items-center gap-2",
                        data?.has_init_yml ? "bg-muted" : "bg-muted/50"
                    )}>
                        {data?.has_init_yml ? (
                            <>
                                <Check size={16} className="text-green-600" />
                                <span className="font-medium">{t('initYmlPresent')}</span>
                            </>
                        ) : (
                            <span className="text-muted-foreground">{t('noInitYml')}</span>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
