'use client';

import { useTranslations } from 'next-intl';
import { cn } from '@/lib/utils';

interface UploadModeToggleProps {
    mode: 'single' | 'batch';
    onModeChange: (mode: 'single' | 'batch') => void;
}

export function UploadModeToggle({ mode, onModeChange }: UploadModeToggleProps) {
    const t = useTranslations('Admin');
    return (
        <div className="flex gap-2">
            <button
                type="button"
                onClick={() => onModeChange('single')}
                className={cn(
                    "px-4 py-2 rounded-xl font-medium transition-colors",
                    mode === 'single'
                        ? "bg-primary text-white"
                        : "bg-card border hover:bg-muted"
                )}
            >
                {t('testcaseUpload.singleModeButton')}
            </button>
            <button
                type="button"
                onClick={() => onModeChange('batch')}
                className={cn(
                    "px-4 py-2 rounded-xl font-medium transition-colors",
                    mode === 'batch'
                        ? "bg-primary text-white"
                        : "bg-card border hover:bg-muted"
                )}
            >
                {t('testcaseUpload.batchModeButton')}
            </button>
        </div>
    );
}
