'use client';

import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { useAuth } from '@/components/providers/AuthProvider';
import { Link } from '@/navigation';
import {
    Crown,
    X,
    Zap,
    ChevronRight,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { QUICK_ACTIONS, ADMIN_STATS } from './constants';

interface AdminWelcomeBannerProps {
    onDismiss?: () => void;
}

export function AdminWelcomeBanner({ onDismiss }: AdminWelcomeBannerProps) {
    const { user } = useAuth();
    const [isVisible, setIsVisible] = useState(true);
    const [currentTime, setCurrentTime] = useState(new Date());

    useEffect(() => {
        const timer = setInterval(() => setCurrentTime(new Date()), 1000);
        return () => clearInterval(timer);
    }, []);

    if (!user?.is_staff || !isVisible) return null;

    const handleDismiss = () => {
        setIsVisible(false);
        onDismiss?.();
    };

    const formatTime = (date: Date) => {
        return date.toLocaleTimeString('en-US', {
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            hour12: false,
        });
    };

    return (
        <motion.div
            initial={{ opacity: 0, y: -20, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -20, scale: 0.95 }}
            transition={{ type: 'spring', damping: 25, stiffness: 300 }}
            className="relative mb-6 overflow-hidden rounded-2xl"
        >
            {/* Background effects */}
            <div className="absolute inset-0 bg-gradient-to-br from-slate-900 via-slate-950 to-slate-900" />
            <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-amber-500/10 via-transparent to-transparent" />
            <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_bottom_left,_var(--tw-gradient-stops))] from-indigo-500/10 via-transparent to-transparent" />

            {/* Grid pattern */}
            <div
                className="absolute inset-0 opacity-[0.03]"
                style={{
                    backgroundImage: `linear-gradient(rgba(255,255,255,0.1) 1px, transparent 1px),
                                      linear-gradient(90deg, rgba(255,255,255,0.1) 1px, transparent 1px)`,
                    backgroundSize: '40px 40px',
                }}
            />

            {/* Animated border */}
            <div className="absolute inset-0 rounded-2xl border border-amber-500/20" />
            <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-amber-500/50 to-transparent" />

            <div className="relative p-6">
                {/* Header */}
                <div className="flex items-start justify-between mb-6">
                    <div className="flex items-center gap-4">
                        <div className="relative">
                            <div className="absolute inset-0 bg-amber-500/30 blur-2xl rounded-full animate-pulse" />
                            <div className="relative w-16 h-16 rounded-2xl bg-gradient-to-br from-amber-400 via-amber-500 to-amber-600 flex items-center justify-center shadow-xl shadow-amber-500/25">
                                <Crown className="w-8 h-8 text-slate-950" />
                            </div>
                        </div>
                        <div>
                            <div className="flex items-center gap-2 mb-1">
                                <h2 className="text-2xl font-bold text-slate-100">
                                    Welcome back, {user.username}
                                </h2>
                                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-amber-500/20 text-amber-400 border border-amber-500/30 uppercase tracking-wider">
                                    {user.is_admin ? 'Super Admin' : 'Admin'}
                                </span>
                            </div>
                            <p className="text-slate-400">
                                You have elevated privileges. Manage the platform wisely.
                            </p>
                        </div>
                    </div>

                    <div className="flex items-center gap-4">
                        {/* Live clock */}
                        <div className="hidden sm:flex items-center gap-2 px-4 py-2 rounded-lg bg-slate-900/50 border border-slate-800">
                            <Zap className="w-4 h-4 text-amber-400" />
                            <span className="text-lg font-mono font-bold text-slate-300 tracking-wider">
                                {formatTime(currentTime)}
                            </span>
                        </div>

                        {/* Dismiss button */}
                        <button
                            onClick={handleDismiss}
                            className="p-2 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition-colors"
                        >
                            <X className="w-5 h-5" />
                        </button>
                    </div>
                </div>

                {/* Quick Actions */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                    {QUICK_ACTIONS.map((action, index) => (
                        <motion.div
                            key={action.label}
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.1 + index * 0.05 }}
                        >
                            <Link
                                href={action.href}
                                className={cn(
                                    'group flex items-center gap-3 p-4 rounded-xl',
                                    'bg-slate-900/50 border border-slate-800',
                                    'hover:border-slate-700 hover:bg-slate-800/50',
                                    'transition-all duration-200'
                                )}
                            >
                                <action.icon className={cn('w-5 h-5', action.color)} />
                                <span className="text-sm font-medium text-slate-300 group-hover:text-slate-200">
                                    {action.label}
                                </span>
                                <ChevronRight className="w-4 h-4 text-slate-600 ml-auto group-hover:text-slate-400 group-hover:translate-x-0.5 transition-all" />
                            </Link>
                        </motion.div>
                    ))}
                </div>

                {/* Stats row */}
                <div className="mt-6 pt-6 border-t border-slate-800/50 grid grid-cols-3 gap-4">
                    {ADMIN_STATS.map((stat) => (
                        <div key={stat.label} className="text-center">
                            <p className="text-xs text-slate-500 uppercase tracking-wider mb-1">
                                {stat.label}
                            </p>
                            {stat.href ? (
                                <Link href={stat.href} className={cn('text-sm font-semibold hover:underline', stat.color)}>
                                    {stat.value}
                                </Link>
                            ) : (
                                <p className={cn('text-sm font-semibold', stat.color)}>
                                    {stat.value}
                                </p>
                            )}
                        </div>
                    ))}
                </div>
            </div>
        </motion.div>
    );
}
