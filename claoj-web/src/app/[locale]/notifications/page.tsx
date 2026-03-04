'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Bell, Check, Trash2, Loader2, Inbox } from 'lucide-react';
import { motion } from 'framer-motion';
import { cn } from '@/lib/utils';
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

const NOTIFICATION_TYPES = [
    { value: 'all', label: 'All' },
    { value: 'unread', label: 'Unread' },
    { value: 'submission', label: 'Submissions' },
    { value: 'contest', label: 'Contests' },
    { value: 'ticket', label: 'Tickets' },
];

export default function NotificationsPage() {
    const [filter, setFilter] = useState('all');
    const [page, setPage] = useState(1);
    const queryClient = useQueryClient();
    const pageSize = 20;

    // Fetch notifications
    const { data, isLoading } = useQuery({
        queryKey: ['notifications', 'list', filter, page],
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
    const totalPages = data?.total_pages || 1;

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

        if (minutes < 1) return 'Just now';
        if (minutes < 60) return `${minutes} minutes ago`;
        if (hours < 24) return `${hours} hours ago`;
        if (days < 7) return `${days} days ago`;
        return date.toLocaleDateString();
    };

    return (
        <div className="max-w-4xl mx-auto space-y-6 animate-in fade-in duration-500">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">Notifications</h1>
                    <p className="text-muted-foreground mt-1">
                        Stay updated with your submissions, contests, and more.
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
                    Mark all as read
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
                        <h3 className="text-xl font-bold mb-2">No notifications</h3>
                        <p className="text-muted-foreground max-w-md mx-auto">
                            You don&apos;t have any {filter !== 'all' ? filter : ''} notifications yet.
                            We&apos;ll notify you when something happens.
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
                                        title="Mark as read"
                                    >
                                        <Check size={18} />
                                    </button>
                                )}
                                <button
                                    onClick={() => deleteNotification.mutate(notification.id)}
                                    disabled={deleteNotification.isPending}
                                    className="p-2 rounded-xl hover:bg-destructive/10 text-destructive transition-colors"
                                    title="Delete"
                                >
                                    <Trash2 size={18} />
                                </button>
                            </div>
                        </motion.div>
                    ))
                )}
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
                <div className="flex items-center justify-center gap-2 pt-4">
                    <button
                        onClick={() => setPage(p => Math.max(1, p - 1))}
                        disabled={page === 1}
                        className="px-4 py-2 rounded-xl border font-bold hover:bg-muted transition-colors disabled:opacity-50"
                    >
                        Previous
                    </button>
                    <span className="px-4 py-2 font-medium text-muted-foreground">
                        Page {page} of {totalPages}
                    </span>
                    <button
                        onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                        disabled={page === totalPages}
                        className="px-4 py-2 rounded-xl border font-bold hover:bg-muted transition-colors disabled:opacity-50"
                    >
                        Next
                    </button>
                </div>
            )}
        </div>
    );
}
