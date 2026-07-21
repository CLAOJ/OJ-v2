'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { randomProblemApi } from '@/lib/api';
import { useRouter } from '@/navigation';
import { Skeleton } from '@/components/ui/Skeleton';
import { Shuffle } from 'lucide-react';

export default function RandomProblemPage() {
    const t = useTranslations('Problems');
    const tCommon = useTranslations('Common');
    const [error, setError] = useState<string | null>(null);
    const router = useRouter();

    useEffect(() => {
        randomProblemApi.getRandomProblem()
            .then((res) => {
                router.push(`/problem/${res.data.code}`);
            })
            .catch((err) => {
                setError(t('randomLoadFailed'));
            });
    }, []);

    if (error) {
        return (
            <div className="max-w-md mx-auto mt-20 p-6 text-center">
                <Shuffle size={48} className="mx-auto mb-4 text-destructive" />
                <h1 className="text-2xl font-bold mb-2">{tCommon('error')}</h1>
                <p className="text-muted-foreground mb-4">{error}</p>
                <a
                    href="/problems"
                    className="inline-block px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
                >
                    {t('backToProblems')}
                </a>
            </div>
        );
    }

    return (
        <div className="max-w-md mx-auto mt-20 p-6 text-center">
            <Shuffle size={48} className="mx-auto mb-4 animate-spin" />
            <h1 className="text-2xl font-bold mb-2">{t('findingRandomProblem')}</h1>
            <p className="text-muted-foreground mb-4">{t('findingRandomProblemDesc')}</p>
            <Skeleton className="h-4 w-48 mx-auto" />
        </div>
    );
}
