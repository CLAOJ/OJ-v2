'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { motion } from 'framer-motion';
import { Loader2, Mail, ArrowLeft, CheckCircle } from 'lucide-react';
import { cn } from '@/lib/utils';
import api from '@/lib/api';
import Link from 'next/link';

/**
 * The login page has always linked to /forgot-password, and the backend has
 * always exposed POST /auth/password/reset — but this page didn't exist, so the
 * link 404'd and the password-reset flow was unreachable from the UI.
 *
 * Modelled on the sibling resend-verification page, which drives the same shape
 * of request against the same kind of endpoint.
 */
export default function ForgotPasswordPage() {
    const t = useTranslations('Auth');
    const [isLoading, setIsLoading] = useState(false);
    const [success, setSuccess] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const resetSchema = z.object({
        email: z.string().email(t('invalidEmailError')),
    });

    type ResetFormValues = z.infer<typeof resetSchema>;

    const {
        register,
        handleSubmit,
        formState: { errors },
    } = useForm<ResetFormValues>({
        resolver: zodResolver(resetSchema),
    });

    const onSubmit = async (data: ResetFormValues) => {
        setIsLoading(true);
        setError(null);
        try {
            await api.post('/auth/password/reset', { email: data.email });
            // The backend answers 200 whether or not the address is registered,
            // so that this form can't be used to enumerate accounts. Show the
            // same confirmation either way.
            setSuccess(true);
        } catch (err: any) {
            setError(err.response?.data?.error || t('resetPasswordErrorDefault'));
        } finally {
            setIsLoading(false);
        }
    };

    if (success) {
        return (
            <div className="flex items-center justify-center min-h-[calc(100vh-12rem)] px-4">
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="w-full max-w-md space-y-8 p-8 rounded-3xl border bg-card shadow-2xl shadow-primary/5 text-center"
                >
                    <div className="mx-auto w-20 h-20 rounded-full bg-emerald-500/10 flex items-center justify-center">
                        <CheckCircle size={40} className="text-emerald-500" />
                    </div>
                    <h2 className="text-2xl font-bold">{t('resetEmailSentTitle')}</h2>
                    <p className="text-muted-foreground">
                        {t('resetEmailSentDesc')}
                    </p>
                    <Link
                        href="/login"
                        className="inline-block px-8 py-3 rounded-xl bg-primary text-primary-foreground font-bold hover:opacity-90 transition-all"
                    >
                        {t('goToLogin')}
                    </Link>
                </motion.div>
            </div>
        );
    }

    return (
        <div className="flex items-center justify-center min-h-[calc(100vh-12rem)] px-4">
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                className="w-full max-w-md space-y-8 p-8 rounded-3xl border bg-card shadow-2xl shadow-primary/5"
            >
                <div className="text-center space-y-2">
                    <div className="mx-auto w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mb-4">
                        <Mail size={32} className="text-primary" />
                    </div>
                    <h2 className="text-2xl font-bold">{t('resetPassword')}</h2>
                    <p className="text-muted-foreground">
                        {t('resetPasswordDesc')}
                    </p>
                </div>

                <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
                    {error && (
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            className="p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm"
                        >
                            {error}
                        </motion.div>
                    )}

                    <div className="space-y-2">
                        <label className="text-sm font-medium">{t('email')}</label>
                        <div className="relative">
                            <Mail className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                            <input
                                {...register('email')}
                                type="email"
                                placeholder="you@example.com"
                                className={cn(
                                    "flex h-12 w-full rounded-xl border border-input bg-background px-10 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all",
                                    errors.email && "border-destructive focus-visible:ring-destructive/20"
                                )}
                            />
                        </div>
                        {errors.email && <p className="text-xs text-destructive">{errors.email.message}</p>}
                    </div>

                    <button
                        type="submit"
                        disabled={isLoading}
                        className="w-full h-12 rounded-xl bg-primary text-primary-foreground font-bold hover:opacity-90 disabled:opacity-50 transition-all flex items-center justify-center gap-2"
                    >
                        {isLoading && <Loader2 size={18} className="animate-spin" />}
                        {t('sendResetLink')}
                    </button>
                </form>

                <Link
                    href="/login"
                    className="flex items-center justify-center gap-2 text-sm text-muted-foreground hover:text-primary transition-colors"
                >
                    <ArrowLeft size={16} />
                    {t('backToLogin')}
                </Link>
            </motion.div>
        </div>
    );
}
