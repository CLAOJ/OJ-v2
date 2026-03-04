'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useState } from 'react';
import { motion } from 'framer-motion';
import { Loader2, Mail, ArrowLeft, CheckCircle } from 'lucide-react';
import { cn } from '@/lib/utils';
import api from '@/lib/api';
import Link from 'next/link';

const resendSchema = z.object({
    email: z.string().email('Please enter a valid email address'),
});

type ResendFormValues = z.infer<typeof resendSchema>;

export default function ResendVerificationPage() {
    const [isLoading, setIsLoading] = useState(false);
    const [success, setSuccess] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const {
        register,
        handleSubmit,
        formState: { errors },
    } = useForm<ResendFormValues>({
        resolver: zodResolver(resendSchema),
    });

    const onSubmit = async (data: ResendFormValues) => {
        setIsLoading(true);
        setError(null);
        try {
            await api.post('/auth/resend-verification', { email: data.email });
            setSuccess(true);
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to send verification email');
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
                    <h2 className="text-2xl font-bold">Verification Email Sent!</h2>
                    <p className="text-muted-foreground">
                        If the email address exists and is not verified, a verification link has been sent.
                        Please check your inbox and spam folder.
                    </p>
                    <Link
                        href="/login"
                        className="inline-block px-8 py-3 rounded-xl bg-primary text-primary-foreground font-bold hover:opacity-90 transition-all"
                    >
                        Go to Login
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
                    <h2 className="text-2xl font-bold">Resend Verification Email</h2>
                    <p className="text-muted-foreground">
                        Enter your email address and we&apos;ll send you a new verification link.
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
                        <label className="text-sm font-medium">Email Address</label>
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
                        Send Verification Email
                    </button>
                </form>

                <Link
                    href="/login"
                    className="flex items-center justify-center gap-2 text-sm text-muted-foreground hover:text-primary transition-colors"
                >
                    <ArrowLeft size={16} />
                    Back to Login
                </Link>
            </motion.div>
        </div>
    );
}
