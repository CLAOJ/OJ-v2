'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { useRouter } from '@/navigation';
import api from '@/lib/api';
import { motion } from 'framer-motion';
import { CheckCircle, XCircle, Loader2, Mail } from 'lucide-react';
import Link from 'next/link';

export default function VerifyEmailPage() {
    const searchParams = useSearchParams();
    const router = useRouter();
    const token = searchParams.get('token');
    const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
    const [message, setMessage] = useState('');

    useEffect(() => {
        if (!token) {
            setStatus('error');
            setMessage('No verification token provided');
            return;
        }

        const verifyEmail = async () => {
            try {
                const res = await api.post('/auth/verify-email', { token });
                setStatus('success');
                setMessage(res.data.message || 'Email verified successfully!');
                // Redirect to login after 3 seconds
                setTimeout(() => {
                    router.push('/login');
                }, 3000);
            } catch (err: any) {
                setStatus('error');
                setMessage(err.response?.data?.error || 'Failed to verify email');
            }
        };

        verifyEmail();
    }, [token, router]);

    return (
        <div className="flex items-center justify-center min-h-[calc(100vh-12rem)] px-4">
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                className="w-full max-w-md space-y-8 p-8 rounded-3xl border bg-card shadow-2xl shadow-primary/5"
            >
                <div className="text-center space-y-4">
                    {status === 'loading' && (
                        <>
                            <div className="mx-auto w-20 h-20 rounded-full bg-primary/10 flex items-center justify-center">
                                <Loader2 size={40} className="text-primary animate-spin" />
                            </div>
                            <h2 className="text-2xl font-bold">Verifying your email...</h2>
                            <p className="text-muted-foreground">Please wait while we verify your email address.</p>
                        </>
                    )}

                    {status === 'success' && (
                        <>
                            <div className="mx-auto w-20 h-20 rounded-full bg-emerald-500/10 flex items-center justify-center">
                                <CheckCircle size={40} className="text-emerald-500" />
                            </div>
                            <h2 className="text-2xl font-bold text-emerald-500">Email Verified!</h2>
                            <p className="text-muted-foreground">{message}</p>
                            <p className="text-sm text-muted-foreground">Redirecting to login page...</p>
                        </>
                    )}

                    {status === 'error' && (
                        <>
                            <div className="mx-auto w-20 h-20 rounded-full bg-destructive/10 flex items-center justify-center">
                                <XCircle size={40} className="text-destructive" />
                            </div>
                            <h2 className="text-2xl font-bold text-destructive">Verification Failed</h2>
                            <p className="text-muted-foreground">{message}</p>
                            <div className="pt-4 space-y-3">
                                <Link
                                    href="/login"
                                    className="inline-block px-6 py-2 rounded-xl bg-primary text-primary-foreground font-bold hover:opacity-90 transition-all"
                                >
                                    Go to Login
                                </Link>
                            </div>
                        </>
                    )}
                </div>
            </motion.div>
        </div>
    );
}
