'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { use } from 'react';
import api from '@/lib/api';
import type { LicenseDetail } from '@/types';
import MathRenderer from '@/components/ui/MathRenderer';
import { Skeleton } from '@/components/ui/Skeleton';
import { ExternalLink, Scale } from 'lucide-react';

export default function LicensePageContent({ params }: { params: Promise<{ key: string }> }) {
    const { key } = use(params);
    const t = useTranslations('License');

    const { data: license, isLoading, error } = useQuery({
        queryKey: ['license', key],
        retry: false,
        queryFn: async () => (await api.get<LicenseDetail>(`/license/${key}`)).data,
    });

    if (isLoading) {
        return (
            <div className="max-w-3xl mx-auto mt-4 space-y-4">
                <Skeleton className="h-10 w-1/3" />
                <Skeleton className="h-64 w-full" />
            </div>
        );
    }

    if (error || !license) {
        return <div className="max-w-3xl mx-auto mt-4 p-4 text-sm">{t('notFound')}</div>;
    }

    return (
        <article className="max-w-3xl mx-auto mt-4 space-y-6">
            <header className="space-y-2">
                <h1 className="text-3xl font-semibold tracking-tight flex items-center gap-3">
                    <Scale size={28} className="text-primary" /> {license.name || license.key}
                </h1>
                <a
                    href={license.link}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-2 text-sm text-primary hover:underline"
                >
                    <ExternalLink size={14} /> {t('source')}: {license.link}
                </a>
            </header>

            <div className="bg-card border rounded-md p-6 shadow-sm">
                <MathRenderer content={license.text} />
            </div>
        </article>
    );
}