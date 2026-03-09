'use client';

import React from 'react';
import { CheckCircle, Mail, AlertTriangle, Loader2 } from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import api from '@/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';

export function EmailVerificationSection() {
    // Email verification query
    const { data: userProfile } = useQuery({
        queryKey: ['user', 'me'],
        queryFn: async () => {
            const res = await api.get('/user/me');
            return res.data;
        },
    });

    const isEmailVerified = userProfile?.is_active ?? true;

    // Resend verification mutation
    const { mutate: resendVerification, isPending: isResendingVerification } = useMutation({
        mutationFn: async () => {
            const res = await api.post('/auth/resend-verification', {});
            return res.data;
        },
        onSuccess: (data) => {
            alert(data.message || 'Verification email sent successfully');
        },
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Failed to send verification email');
        },
    });

    return (
        <section className="space-y-6">
            <div className="flex items-center gap-2 text-primary font-bold">
                <Mail size={18} />
                Email Verification
            </div>

            {isEmailVerified ? (
                <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-between">
                    <p className="text-sm font-medium text-emerald-700 flex items-center gap-2">
                        <CheckCircle size={16} />
                        Your email is verified
                    </p>
                    <Badge variant="default" className="bg-emerald-500 text-white">
                        Verified
                    </Badge>
                </div>
            ) : (
                <div className="space-y-4">
                    <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 flex items-start gap-3">
                        <AlertTriangle size={18} className="text-amber-600 mt-0.5 flex-shrink-0" />
                        <div className="space-y-1">
                            <p className="text-sm font-medium text-amber-700">
                                Email not verified
                            </p>
                            <p className="text-sm text-amber-600">
                                Please verify your email address to access all features. Check your inbox for the verification link, or resend the email below.
                            </p>
                        </div>
                    </div>
                    <button
                        onClick={() => resendVerification()}
                        disabled={isResendingVerification}
                        className="px-6 h-12 rounded-xl bg-primary text-primary-foreground font-bold hover:bg-primary/90 transition-all disabled:opacity-50 flex items-center gap-2"
                    >
                        {isResendingVerification && <Loader2 size={18} className="animate-spin" />}
                        Resend Verification Email
                    </button>
                </div>
            )}
        </section>
    );
}
