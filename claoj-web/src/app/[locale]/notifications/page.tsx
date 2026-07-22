'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import api from '@/lib/api';
import { Bell, Check, Trash2, Loader2, Inbox } from 'lucide-react';
import { motion } from 'framer-motion';
import { cn } from '@/lib/utils';
import { PaginationBar, PAGE_SIZE_OPTIONS } from '@/components/ui/PaginationBar';
import Link from 'next/link';
import { useState } from 'react';

interface Notification {
    id: number;
    type: string;
    title: string;
    message: string;
    link: string;
    read: boolean;
    created_at: string;
}

export default function NotificationsPage() {
    const t = useTranslations('Notification');
    const [filter, setFilter] = useState('all');
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(PAGE_SIZE_OPTIONS[1]);
    const queryClient = useQueryClient();

    const NOTIFICATION_TYPES = [
        { value: 'all', label: t('filterAll') },
        { value: 'unread', label: t('filterUnread') },
        { value: 'submission', label: t('filterSubmissions') },
        { value: 'contest', label: t('filterContests') },
        { value: 'ticket', label: t('filterTickets') },
    ];

    const typeLabels: Record<string, string> = {
        unread: t('typeUnread'),
        submission: t('typeSubmission'),
        contest: t('typeContest'),
        ticket: t('typeTicket'),
    };

    // Fetch notifications
    const { data, isLoading } = useQuery({
        queryKey: ['notifications', 'list', filter, page, pageSize],
        queryFn: async () => {
            let url = `/notifications?page=${page}&page_size=${pageSize}`;
            if (filter === 'unread') {
                url += '&read=false';
            }
            const res = await api.get(url);
            return res.data;
        },
    });

    // Mark as read mutation
    const markAsRead = useMutation({
        mutationFn: async (id: number) => {
            await api.post(`/notifications/${id}/read`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['notifications'] });
        },
    });

    // Mark all as read mutation
    const markAllAsRead = useMutation({
        mutationFn: async () => {
            await api.post('/notifications/read-all');
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['notifications'] });
        },
    });

    // Delete notification mutation
    const deleteNotification = useMutation({
        mutationFn: async (id: number) => {
            await api.delete(`/notifications/${id}`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['notifications'] });
        },
    });

    const notifications: Notification[] = data?.results || [];

    const getNotificationIcon = (type: string) => {
        switch (type) {
            case 'submission':
                return '📝';
            case 'contest':
                return '🏆';
            case 'ticket':
                return '🎫';
            default:
                return '📢';
        }
    };

    const formatTime = (dateStr: string) => {
        const date = new Date(dateStr);
        const now = new Date();
        const diff = now.getTime() - date.getTime();

        const minutes = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);

        if (minutes < 1) return t('justNow');
        if (minutes < 60) return t('minutesAgo', { minutes });
        if (hours < 24) return t('hoursAgo', { hours });
        if (days < 7) return t('daysAgo', { days });
        return date.toLocaleDateString();
    };

    return (
        <div className="max-w-4xl mx-auto space-y-6 animate-in fade-in duration-500">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">{t('title')}</h1>
                    <p className="text-muted-foreground mt-1">
                        {t('subtitle')}
                    </p>
                </div>
                <button
                    onClick={() => markAllAsRead.mutate()}
                    disabled={markAllAsRead.isPending || notifications.length === 0}
                    className="flex items-center justify-center gap-2 px-4 py-2 rounded-xl border font-bold hover:bg-muted transition-colors disabled:opacity-50"
                >
                    {markAllAsRead.isPending ? (
                        <Loader2 size={16} className="animate-spin" />
                    ) : (
                        <Check size={16} />
                    )}
                    {t('markAllRead')}
                </button>
            </div>

            {/* Filter Tabs */}
            <div className="flex flex-wrap gap-2">
                {NOTIFICATION_TYPES.map((type) => (
                    <button
                        key={type.value}
                        onClick={() => {
                            setFilter(type.value);
                            setPage(1);
                        }}
                        className={cn(
                            "px-4 py-2 rounded-xl font-bold text-sm transition-all",
                            filter === type.value
                                ? "bg-primary text-primary-foreground"
                                : "bg-muted hover:bg-muted/80 text-muted-foreground"
                        )}
                    >
                        {type.label}
                    </button>
                ))}
            </div>

            {/* Notifications List */}
            <div className="space-y-3">
                {isLoading ? (
                    <div className="flex items-center justify-center py-20">
                        <Loader2 size={32} className="animate-spin text-muted-foreground" />
                    </div>
                ) : notifications.length === 0 ? (
                    <div className="text-center py-20 px-4 border rounded-3xl bg-card">
                        <div className="w-24 h-24 mx-auto mb-6 rounded-full bg-muted flex items-center justify-center">
                            <Inbox size={40} className="text-muted-foreground" />
                        </div>
                        <h3 className="text-xl font-bold mb-2">{t('noNotifications')}</h3>
                        <p className="text-muted-foreground max-w-md mx-auto">
                            {filter !== 'all'
                                ? t('emptyStateFiltered', { filter: typeLabels[filter] ?? filter })
                                : t('emptyStateAll')}
                        </p>
                    </div>
                ) : (
                    notifications.map((notification) => (
                        <motion.div
                            key={notification.id}
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            className={cn(
                                "group flex items-start gap-4 p-5 rounded-2xl border bg-card hover:shadow-md transition-all",
                                !notification.read && "border-primary/20 bg-primary/5"
                            )}
                        >
                            <div className="text-3xl shrink-0">
                                {getNotificationIcon(notification.type)}
                            </div>
                            <div className="flex-1 min-w-0">
                                <Link
                                    href={notification.link}
                                    onClick={() => {
                                        if (!notification.read) {
                                            markAsRead.mutate(notification.id);
                                        }
                                    }}
                                    className="block"
                                >
                                    <div className="flex items-start justify-between gap-4">
                                        <div>
                                            <h3 className={cn(
                                                "font-bold text-lg",
                                                !notification.read && "text-primary"
                                            )}>
                                                {notification.title}
                                            </h3>
                                            <p className="text-muted-foreground mt-1">
                                                {notification.message}
                                            </p>
                                            <p className="text-sm text-muted-foreground/60 mt-2">
                                                {formatTime(notification.created_at)}
                                            </p>
                                        </div>
                                    </div>
                                </Link>
                            </div>
                            <div className="flex flex-col gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                                {!notification.read && (
                                    <button
                                        onClick={() => markAsRead.mutate(notification.id)}
                                        disabled={markAsRead.isPending}
                                        className="p-2 rounded-xl hover:bg-primary/10 text-primary transition-colors"
                                        title={t('markAsRead')}
                                    >
                                        <Check size={18} />
                                    </button>
                                )}
                                <button
                                    onClick={() => deleteNotification.mutate(notification.id)}
                                    disabled={deleteNotification.isPending}
                                    className="p-2 rounded-xl hover:bg-destructive/10 text-destructive transition-colors"
                                    title={t('delete')}
                                >
                                    <Trash2 size={18} />
                                </button>
                            </div>
                        </motion.div>
                    ))
                )}
            </div>

            <PaginationBar
                page={page}
                onPageChange={setPage}
                total={data?.total}
                pageSize={pageSize}
                onPageSizeChange={size => { setPageSize(size); setPage(1); }}
            />
        </div>
    );
}
