'use client';

import React, { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { motion } from 'framer-motion';
import { Loader2, CheckCircle, Lock, Key } from 'lucide-react';
import api from '@/lib/api';
import { useMutation } from '@tanstack/react-query';

const passwordSchema = z.object({
    current_password: z.string().min(1, 'Current password is required'),
    new_password: z.string().min(6, 'New password must be at least 6 characters'),
    confirm_password: z.string().min(6, 'Please confirm your new password'),
}).refine((data) => data.new_password === data.confirm_password, {
    message: "Passwords don't match",
    path: ['confirm_password'],
});

type PasswordFormValues = z.infer<typeof passwordSchema>;

export function PasswordChangeForm() {
    const [passwordSuccess, setPasswordSuccess] = useState(false);

    const {
        register,
        handleSubmit,
        reset,
        formState: { errors },
    } = useForm<PasswordFormValues>({
        resolver: zodResolver(passwordSchema),
    });

    const { mutate: changePassword, isPending: isChangingPassword } = useMutation({
        mutationFn: async (data: PasswordFormValues) => {
            await api.post('/auth/password/change', {
                current_password: data.current_password,
                new_password: data.new_password,
            });
        },
        onSuccess: () => {
            setPasswordSuccess(true);
            reset();
            setTimeout(() => setPasswordSuccess(false), 3000);
        },
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Failed to change password');
        },
    });

    return (
        <form onSubmit={handleSubmit((data) => changePassword(data))} className="space-y-8">
            <section className="space-y-6">
                <div className="flex items-center gap-2 text-primary font-bold">
                    <Lock size={18} />
                    Change Password
                </div>

                <div className="grid grid-cols-1 gap-6">
                    <div className="space-y-2">
                        <label className="text-sm font-medium">Current Password</label>
                        <div className="relative">
                            <Lock className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                            <input
                                {...register('current_password')}
                                type="password"
                                className="flex h-12 w-full rounded-xl border border-input bg-background px-10 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all font-medium"
                            />
                        </div>
                        {errors.current_password && (
                            <p className="text-xs text-destructive">{errors.current_password.message}</p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <label className="text-sm font-medium">New Password</label>
                        <div className="relative">
                            <Key className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                            <input
                                {...register('new_password')}
                                type="password"
                                className="flex h-12 w-full rounded-xl border border-input bg-background px-10 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all font-medium"
                            />
                        </div>
                        {errors.new_password && (
                            <p className="text-xs text-destructive">{errors.new_password.message}</p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <label className="text-sm font-medium">Confirm New Password</label>
                        <div className="relative">
                            <Key className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                            <input
                                {...register('confirm_password')}
                                type="password"
                                className="flex h-12 w-full rounded-xl border border-input bg-background px-10 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 transition-all font-medium"
                            />
                        </div>
                        {errors.confirm_password && (
                            <p className="text-xs text-destructive">{errors.confirm_password.message}</p>
                        )}
                    </div>
                </div>
            </section>

            {passwordSuccess && (
                <motion.div
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: 'auto' }}
                    className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-500 text-sm flex items-center gap-3 font-medium"
                >
                    <CheckCircle size={18} />
                    Password changed successfully
                </motion.div>
            )}

            <button
                type="submit"
                disabled={isChangingPassword}
                className="px-8 h-12 rounded-xl bg-primary text-primary-foreground font-bold hover:opacity-90 active:scale-95 disabled:opacity-50 transition-all flex items-center justify-center gap-2"
            >
                {isChangingPassword && <Loader2 size={18} className="animate-spin" />}
                Change Password
            </button>
        </form>
    );
}
