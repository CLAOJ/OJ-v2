'use client';

import React, { useState, useEffect } from 'react';
import { motion, useReducedMotion } from 'framer-motion';
import { useTranslations } from 'next-intl';
import { useAuth } from '@/components/providers/AuthProvider';
import { Link } from '@/navigation';
import { Crown, X, Clock, ChevronRight } from 'lucide-react';
import { QUICK_ACTIONS, ADMIN_STATS } from './constants';

interface AdminWelcomeBannerProps {
    onDismiss?: () => void;
}

export function AdminWelcomeBanner({ onDismiss }: AdminWelcomeBannerProps) {
    const { user } = useAuth();
    const t = useTranslations('Admin');
    const reduceMotion = useReducedMotion();
    const [isVisible, setIsVisible] = useState(true);
    // Filled on the client only: seeding with new Date() would render a
    // different time on the server and trip a hydration mismatch.
    const [currentTime, setCurrentTime] = useState<Date | null>(null);

    useEffect(() => {
        setCurrentTime(new Date());
        const timer = setInterval(() => setCurrentTime(new Date()), 1000);
        return () => clearInterval(timer);
    }, []);

    // Superuser-inclusive: admins bypass the staff gate (see backend route gate).
    if ((!user?.is_staff && !user?.is_admin) || !isVisible) return null;

    const handleDismiss = () => {
        setIsVisible(false);
        onDismiss?.();
    };

    const formatTime = (date: Date) =>
        date.toLocaleTimeString('en-GB', {
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            hour12: false,
        });

    return (
        <motion.section
            initial={reduceMotion ? false : { opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, ease: [0.16, 1, 0.3, 1] }}
            className="mb-6 bg-card border rounded-lg overflow-hidden"
        >
            <div className="p-6">
                {/* Header */}
                <div className="flex items-start justify-between gap-4 mb-6">
                    <div className="flex items-center gap-4 min-w-0">
                        <div className="w-12 h-12 shrink-0 rounded-lg bg-primary flex items-center justify-center">
                            <Crown className="w-6 h-6 text-primary-foreground" />
                        </div>
                        <div className="min-w-0">
                            <div className="flex flex-wrap items-center gap-2 mb-1">
                                <h2 className="text-xl font-bold text-foreground">
                                    {t('shell.welcomeBack', { username: user.username })}
                                </h2>
                                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider bg-primary/15 text-primary border border-primary/30">
                                    {user.is_admin ? t('shell.superAdmin') : t('layout.adminLabel')}
                                </span>
                            </div>
                            <p className="text-sm text-muted-foreground">
                                {t('shell.elevatedPrivileges')}
                            </p>
                        </div>
                    </div>

                    <div className="flex items-center gap-2 shrink-0">
                        {/* Live clock */}
                        <div className="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg bg-muted/50 border">
                            <Clock className="w-4 h-4 text-primary" />
                            <span className="text-sm font-mono font-semibold text-foreground tabular-nums">
                                {currentTime ? formatTime(currentTime) : '--:--:--'}
                            </span>
                        </div>

                        {/* Dismiss */}
                        <button
                            onClick={handleDismiss}
                            aria-label="Close"
                            className="p-2 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
                        >
                            <X className="w-5 h-5" />
                        </button>
                    </div>
                </div>

                {/* Quick actions */}
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
                    {QUICK_ACTIONS.map((action, index) => (
                        <motion.div
                            key={action.id}
                            initial={reduceMotion ? false : { opacity: 0, y: 8 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{
                                duration: 0.3,
                                delay: reduceMotion ? 0 : 0.05 + index * 0.04,
                                ease: [0.16, 1, 0.3, 1],
                            }}
                        >
                            <Link
                                href={action.href}
                                className="group flex items-center gap-3 p-3 rounded-lg bg-muted/50 border hover:border-primary/50 hover:bg-muted transition-colors"
                            >
                                <action.icon className="w-5 h-5 shrink-0 text-primary" />
                                <span className="text-sm font-medium text-foreground truncate">
                                    {t(`shell.quickActions.${action.id}`)}
                                </span>
                                <ChevronRight className="w-4 h-4 ml-auto shrink-0 text-muted-foreground group-hover:text-primary transition-colors" />
                            </Link>
                        </motion.div>
                    ))}
                </div>

                {/* Stats */}
                <div className="mt-6 pt-5 border-t grid grid-cols-1 sm:grid-cols-3 gap-4">
                    {ADMIN_STATS.map((stat) => (
                        <div key={stat.id} className="text-center">
                            <p className="text-[11px] text-muted-foreground uppercase tracking-wider mb-1">
                                {t(`shell.stats.${stat.id}Label`)}
                            </p>
                            {stat.href ? (
                                <Link
                                    href={stat.href}
                                    className="text-sm font-semibold text-primary hover:underline"
                                >
                                    {t(`shell.stats.${stat.id}Value`)}
                                </Link>
                            ) : (
                                <p className="text-sm font-semibold text-foreground">
                                    {t(`shell.stats.${stat.id}Value`)}
                                </p>
                            )}
                        </div>
                    ))}
                </div>
            </div>
        </motion.section>
    );
}
