'use client';

import React, { useState } from 'react';
import { useTranslations } from 'next-intl';
import { useMutation } from '@tanstack/react-query';
import { CheckCircle, Loader2, Mail } from 'lucide-react';
import api from '@/lib/api';

/** v1 EmailChangeRequestView: new address + current password, link by email. */
export function EmailChangeSection() {
    const t = useTranslations('Settings');
    const [newEmail, setNewEmail] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState<string | null>(null);
    const [sentTo, setSentTo] = useState<string | null>(null);

    const { mutate: requestChange, isPending } = useMutation({
        mutationFn: async () => {
            const res = await api.post('/auth/email/change', { new_email: newEmail, password });
            return res.data as { message?: string };
        },
        onSuccess: () => {
            setSentTo(newEmail);
            setNewEmail('');
            setPassword('');
        },
        onError: (err: unknown) => {
            const message = (err as { response?: { data?: { error?: string } } }).response?.data?.error;
            setError(message || t('emailChangeError'));
        },
    });

    const submit = (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);
        setSentTo(null);
        requestChange();
    };

    return (
        <section className="space-y-6">
            <div className="flex items-center gap-2 text-primary font-medium">
                <Mail size={18} />
                {t('emailChangeTitle')}
            </div>
            <p className="text-sm text-muted-foreground">{t('emailChangeDesc')}</p>

            <form onSubmit={submit} className="grid grid-cols-1 gap-6">
                <div className="space-y-2">
                    <label htmlFor="email-change-new" className="text-sm font-medium">{t('emailChangeNewLabel')}</label>
                    <input
                        id="email-change-new"
                        type="email"
                        required
                        value={newEmail}
                        onChange={(e) => setNewEmail(e.target.value)}
                        placeholder="you@example.com"
                        className="flex h-12 w-full rounded-xl border border-input bg-background px-4 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20"
                    />
                </div>
                <div className="space-y-2">
                    <label htmlFor="email-change-password" className="text-sm font-medium">{t('passwordLabel')}</label>
                    <input
                        id="email-change-password"
                        type="password"
                        required
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        placeholder={t('enterYourPasswordPlaceholder')}
                        className="flex h-12 w-full rounded-xl border border-input bg-background px-4 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20"
                    />
                </div>

                {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
                {sentTo && (
                    <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 text-sm flex items-center gap-3 font-medium">
                        <CheckCircle size={18} />
                        {t('emailChangeSent', { email: sentTo })}
                    </div>
                )}

                <button
                    type="submit"
                    disabled={isPending}
                    className="px-8 h-12 w-fit rounded-xl bg-primary text-primary-foreground font-medium hover:opacity-90 disabled:opacity-50 transition-all flex items-center justify-center gap-2"
                >
                    {isPending && <Loader2 size={18} className="animate-spin" />}
                    {t('emailChangeSubmit')}
                </button>
            </form>
        </section>
    );
}