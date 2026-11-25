'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Link } from '@/navigation';
import { CheckCircle, Loader2, XCircle } from 'lucide-react';

/** v1 EmailChangeActivateView: redeems the one-time link for the logged-in user. */
export default function EmailChangeConfirmContent({ token }: { token: string }) {
    const t = useTranslations('EmailChange');
    const [state, setState] = useState<'loading' | 'done' | 'failed'>(token ? 'loading' : 'failed');
    const [message, setMessage] = useState<string>(token ? '' : t('invalid'));
    // React 19 StrictMode mounts effects twice in dev; redeeming twice would
    // turn a valid single-use token into "expired".
    const started = useRef(false);

    useEffect(() => {
        if (!token || started.current) return;
        started.current = true;
        api.post('/auth/email/change/confirm', { token })
            .then((res) => {
                setMessage((res.data as { message?: string }).message || t('success'));
                setState('done');
            })
            .catch((err: unknown) => {
                const error = (err as { response?: { data?: { error?: string } } }).response?.data?.error;
                setMessage(error || t('failed'));
                setState('failed');
            });
    }, [token, t]);

    return (
        <div className="max-w-lg mx-auto mt-16 p-8 rounded-md border bg-card shadow-sm text-center space-y-4">
            {state === 'loading' && <Loader2 size={32} className="mx-auto animate-spin text-primary" />}
            {state === 'done' && <CheckCircle size={32} className="mx-auto text-emerald-500" />}
            {state === 'failed' && <XCircle size={32} className="mx-auto text-destructive" />}

            <h1 className="text-xl font-semibold">
                {state === 'loading' ? t('working') : state === 'done' ? t('successTitle') : t('failedTitle')}
            </h1>
            {message && <p className="text-sm text-muted-foreground">{message}</p>}

            <Link href="/settings" className="inline-block text-sm font-medium text-primary hover:underline">
                {t('backToSettings')}
            </Link>
        </div>
    );
}