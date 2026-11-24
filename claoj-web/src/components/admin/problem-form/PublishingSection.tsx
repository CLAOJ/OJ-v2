'use client';

import { UseFormRegister } from 'react-hook-form';
import { useTranslations } from 'next-intl';
import type { AdminLicense } from '@/types';
import type { ProblemFormData } from '../ProblemForm';

const SOURCE_MODES = ['A', 'S', 'O', 'F'] as const;
const TESTCASE_MODES = ['A', 'C', 'O'] as const;

interface PublishingValues {
    source: string;
    date: string;
    license_id?: number;
    submissionSourceMode: string;
    testcaseMode: string;
    is_full_markup: boolean;
    og_image: string;
    summary: string;
}

interface PublishingSectionProps {
    licenses?: AdminLicense[];
    values: PublishingValues;
    register: UseFormRegister<ProblemFormData>;
    onDateChange: (value: string) => void;
    onLicenseChange: (id?: number) => void;
    onSubmissionSourceModeChange: (mode: string) => void;
    onTestcaseModeChange: (mode: string) => void;
}

export function PublishingSection({
    licenses,
    values,
    register,
    onDateChange,
    onLicenseChange,
    onSubmissionSourceModeChange,
    onTestcaseModeChange
}: PublishingSectionProps) {
    const t = useTranslations('Admin.problemForm');
    const fieldClass = 'w-full px-3 py-2 rounded-lg bg-card border focus:ring-2 focus:ring-primary/20 outline-none';

    return (
        <div className="bg-card rounded-md border p-6 space-y-4">
            <h3 className="text-lg font-semibold">{t('publishingTitle')}</h3>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <label htmlFor="publish-date" className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('dateLabel')}
                    </label>
                    <input
                        id="publish-date"
                        type="datetime-local"
                        className={fieldClass}
                        value={values.date}
                        onChange={(e) => onDateChange(e.target.value)}
                    />
                </div>

                <div>
                    <label htmlFor="problem-license" className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('licenseLabel')}
                    </label>
                    <select
                        id="problem-license"
                        className={fieldClass}
                        value={values.license_id ?? ''}
                        onChange={(e) => onLicenseChange(e.target.value ? Number(e.target.value) : undefined)}
                    >
                        <option value="">{t('licenseNoneOption')}</option>
                        {licenses?.map(license => (
                            <option key={license.id} value={license.id}>{license.name || license.key}</option>
                        ))}
                    </select>
                </div>
            </div>

            <div>
                <label htmlFor="problem-source" className="text-sm font-medium text-muted-foreground block mb-2">
                    {t('sourceLabel')}
                </label>
                <input
                    id="problem-source"
                    type="text"
                    className={fieldClass}
                    placeholder={t('sourcePlaceholder')}
                    maxLength={200}
                    {...register('source')}
                    value={values.source}
                />
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <label htmlFor="submission-visibility" className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('submissionVisibilityLabel')}
                    </label>
                    <select
                        id="submission-visibility"
                        className={fieldClass}
                        value={values.submissionSourceMode}
                        onChange={(e) => onSubmissionSourceModeChange(e.target.value)}
                    >
                        {SOURCE_MODES.map(mode => (
                            <option key={mode} value={mode}>{t(`submissionVisibility.${mode}`)}</option>
                        ))}
                    </select>
                </div>

                <div>
                    <label htmlFor="testcase-visibility" className="text-sm font-medium text-muted-foreground block mb-2">
                        {t('testcaseVisibilityLabel')}
                    </label>
                    <select
                        id="testcase-visibility"
                        className={fieldClass}
                        value={values.testcaseMode}
                        onChange={(e) => onTestcaseModeChange(e.target.value)}
                    >
                        {TESTCASE_MODES.map(mode => (
                            <option key={mode} value={mode}>{t(`testcaseVisibility.${mode}`)}</option>
                        ))}
                    </select>
                </div>
            </div>

            <label className="flex items-center gap-2 cursor-pointer">
                <input
                    type="checkbox"
                    className="rounded w-5 h-5"
                    {...register('is_full_markup')}
                    checked={values.is_full_markup}
                />
                <span className="text-sm font-medium">{t('fullMarkupLabel')}</span>
            </label>

            <div>
                <label htmlFor="problem-og-image" className="text-sm font-medium text-muted-foreground block mb-2">
                    {t('ogImageLabel')}
                </label>
                <input
                    id="problem-og-image"
                    type="text"
                    className={fieldClass}
                    placeholder="https://example.com/og-image.png"
                    maxLength={150}
                    {...register('og_image')}
                    value={values.og_image}
                />
            </div>

            <div>
                <label htmlFor="problem-summary" className="text-sm font-medium text-muted-foreground block mb-2">
                    {t('summaryLabel')}
                </label>
                <textarea
                    id="problem-summary"
                    className={`${fieldClass} min-h-[80px]`}
                    placeholder={t('summaryPlaceholder')}
                    {...register('summary')}
                    value={values.summary}
                />
            </div>
        </div>
    );
}