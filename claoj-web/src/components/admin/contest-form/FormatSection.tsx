'use client';

import { UseFormRegister } from 'react-hook-form';
import { useTranslations } from 'next-intl';
import { cn } from '@/lib/utils';

interface FormatSettings {
    format_name?: string;
    access_code?: string;
    is_visible: boolean;
    hide_problem_tags: boolean;
    run_pretests_only: boolean;
}

interface ContestFormData {
    key: string;
    name: string;
    description: string;
    summary?: string;
    start_time: string;
    end_time: string;
    time_limit?: number;
    is_visible: boolean;
    is_rated: boolean;
    format_name?: string;
    format_config?: string;
    access_code?: string;
    hide_problem_tags: boolean;
    run_pretests_only: boolean;
    is_organization_private: boolean;
    author_ids?: number[];
    curator_ids?: number[];
    tester_ids?: number[];
    problem_ids?: number[];
    tag_ids?: number[];
}

interface FormatSectionProps {
    formData: FormatSettings;
    selectedFormat: string;
    register: UseFormRegister<ContestFormData>;
    setValue: (field: keyof ContestFormData, value: any) => void;
    formatConfigError?: string;
}

const CONTEST_FORMATS = [
    { value: 'icpc', label: 'ICPC' },
    { value: 'ioi', label: 'IOI' },
    { value: 'atcoder', label: 'AtCoder' },
    { value: 'ecoo', label: 'ECOO' },
];

export function FormatSection({ formData, selectedFormat, register, setValue, formatConfigError }: FormatSectionProps) {
    const t = useTranslations('Admin');
    return (
        <div className="bg-card rounded-md border p-6 space-y-4">
            <h3 className="text-lg font-semibold">{t('contestForm.formatSettingsTitle')}</h3>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('contestForm.formatLabel')}
                    </label>
                    <select
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        value={selectedFormat || 'icpc'}
                        onChange={(e) => setValue('format_name', e.target.value)}
                    >
                        {CONTEST_FORMATS.map(f => (
                            <option key={f.value} value={f.value}>{f.label}</option>
                        ))}
                    </select>
                </div>

                <div>
                    <label className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('contestForm.accessCodeLabel')}
                    </label>
                    <input
                        type="text"
                        className="w-full px-3 py-2 rounded-xl bg-card border focus:ring-2 focus:ring-primary/20 outline-none"
                        placeholder={t('contestForm.accessCodePlaceholder')}
                        {...register('access_code')}
                    />
                </div>
            </div>

            <div>
                <label className="text-sm font-medium text-muted-foreground block mb-2">
                    {t('contestForm.formatConfigLabel')}
                </label>
                <textarea
                    rows={5}
                    className="w-full px-3 py-2 rounded-xl bg-card border font-mono text-sm focus:ring-2 focus:ring-primary/20 outline-none"
                    placeholder={t('contestForm.formatConfigPlaceholder')}
                    {...register('format_config', {
                        // Empty is fine: the format keeps its defaults.
                        validate: (value) => {
                            if (!value || !value.trim()) return true;
                            try {
                                JSON.parse(value);
                                return true;
                            } catch {
                                return t('contestForm.formatConfigInvalid');
                            }
                        },
                    })}
                />
                <p className={cn('text-xs mt-1', formatConfigError ? 'text-destructive' : 'text-muted-foreground')}>
                    {formatConfigError || t('contestForm.formatConfigHint')}
                </p>
            </div>

            <div className="flex flex-wrap items-center gap-4">
                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('is_visible')}
                    />
                    <span className="text-sm font-medium">{t('contestForm.visiblePublicLabel')}</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('hide_problem_tags')}
                    />
                    <span className="text-sm font-medium">{t('contestForm.hideProblemTagsLabel')}</span>
                </label>

                <label className="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        className="rounded w-5 h-5"
                        {...register('run_pretests_only')}
                    />
                    <span className="text-sm font-medium">{t('contestForm.pretestsOnlyLabel')}</span>
                </label>
            </div>
        </div>
    );
}
