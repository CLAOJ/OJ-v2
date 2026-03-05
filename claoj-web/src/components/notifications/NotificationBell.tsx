'use client';

import { useState, useEffect, useRef } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Bell, Check, Trash2, Loader2 } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { cn } from '@/lib/utils';
import Link from 'next/link';
import { formatRelativeTime } from '@/lib/date';

interface Notification {
    id: number;
    type: string;
    title: string;
    message: string;
    link: string;
    read: boolean;
    created_at: string;
}

export default function NotificationBell() {
    const [isOpen, setIsOpen] = useState(false);
    const dropdownRef = useRef<HTMLDivElement>(null);
    const queryClient = useQueryClient();

    // Fetch unread count
    const { data: unreadData } = useQuery({
        queryKey: ['notifications', 'unread-count'],
        queryFn: async () => {
            const res = await api.get('/notifications/unread-count');
            return res.data.unread_count as number;
        },
        refetchInterval: 30000, // Refetch every 30 seconds
    });

    // Fetch notifications
    const { data: notificationsData, isLoading } = useQuery({
        queryKey: ['notifications'],
        queryFn: async () => {
            const res = await api.get('/notifications?page_size=10');
            return res.data.results as Notification[];
        },
        enabled: isOpen,
    });

    // Mark as read mutation
    const markAsRead = useMutation({
        mutationFn: async (id: number) => {
            await api.post(`/notifications/${id}/read`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['notifications'] });
            queryClient.invalidateQueries({ queryKey: ['notifications', 'unread-count'] });
        },
    });

    // Mark all as read mutation
    const markAllAsRead = useMutation({
        mutationFn: async () => {
            await api.post('/notifications/read-all');
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['notifications'] });
            queryClient.invalidateQueries({ queryKey: ['notifications', 'unread-count'] });
        },
    });

    // Delete notification mutation
    const deleteNotification = useMutation({
        mutationFn: async (id: number) => {
            await api.delete(`/notifications/${id}`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['notifications'] });
            queryClient.invalidateQueries({ queryKey: ['notifications', 'unread-count'] });
        },
    });

    // Close dropdown when clicking outside
    useEffect(() => {
        function handleClickOutside(event: MouseEvent) {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
                setIsOpen(false);
            }
        }

        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    const unreadCount = unreadData || 0;
    const notifications = notificationsData || [];

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

    return (
        <div className="relative" ref={dropdownRef}>
            <button
                onClick={() => setIsOpen(!isOpen)}
                className="relative p-2 rounded-full hover:bg-white/10 transition-colors text-gray-400 hover:text-white"
                aria-label="Notifications"
                aria-expanded={isOpen}
                aria-haspopup="true"
                title="Notifications"
            >
                <Bell size={20} />
                {unreadCount > 0 && (
                    <span className="absolute -top-1 -right-1 w-5 h-5 bg-red-500 text-white text-xs font-bold rounded-full flex items-center justify-center animate-pulse">
                        {unreadCount > 99 ? '99+' : unreadCount}
                    </span>
                )}
            </button>

            <AnimatePresence>
                {isOpen && (
                    <motion.div
                        initial={{ opacity: 0, y: 10, scale: 0.95 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: 10, scale: 0.95 }}
                        transition={{ duration: 0.15 }}
                        className="absolute right-0 mt-2 w-80 sm:w-96 bg-card border rounded-xl shadow-2xl z-50 overflow-hidden"
                    >
                        {/* Header */}
                        <div className="flex items-center justify-between p-4 border-b bg-muted/30">
                            <h3 className="font-bold text-lg">Notifications</h3>
                            {unreadCount > 0 && (
                                <button
                                    onClick={() => markAllAsRead.mutate()}
                                    disabled={markAllAsRead.isPending}
                                    className="text-xs font-bold text-primary hover:text-primary/80 transition-colors flex items-center gap-1"
                                >
                                    {markAllAsRead.isPending ? (
                                        <Loader2 size={12} className="animate-spin" />
                                    ) : (
                                        <Check size={12} />
                                    )}
                                    Mark all read
                                </button>
                            )}
                        </div>

                        {/* Notifications List */}
                        <div className="max-h-96 overflow-y-auto">
                            {isLoading ? (
                                <div className="flex items-center justify-center p-8">
                                    <Loader2 size={24} className="animate-spin text-muted-foreground" />
                                </div>
                            ) : notifications.length === 0 ? (
                                <div className="text-center py-12 px-4">
                                    <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-muted flex items-center justify-center">
                                        <Bell size={24} className="text-muted-foreground" />
                                    </div>
                                    <p className="text-muted-foreground font-medium">No notifications yet</p>
                                    <p className="text-xs text-muted-foreground/70 mt-1">
                                        We&apos;ll notify you when something happens
                                    </p>
                                </div>
                            ) : (
                                notifications.map((notification) => (
                                    <div
                                        key={notification.id}
                                        className={cn(
                                            "group flex items-start gap-3 p-4 border-b last:border-b-0 hover:bg-muted/50 transition-colors",
                                            !notification.read && "bg-primary/5"
                                        )}
                                    >
                                        <div className="text-2xl shrink-0">
                                            {getNotificationIcon(notification.type)}
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <Link
                                                href={notification.link}
                                                onClick={() => {
                                                    if (!notification.read) {
                                                        markAsRead.mutate(notification.id);
                                                    }
                                                    setIsOpen(false);
                                                }}
                                                className="block"
                                            >
                                                <p className={cn(
                                                    "text-sm font-semibold line-clamp-1",
                                                    !notification.read && "text-primary"
                                                )}>
                                                    {notification.title}
                                                </p>
                                                <p className="text-sm text-muted-foreground line-clamp-2 mt-0.5">
                                                    {notification.message}
                                                </p>
                                                <p className="text-xs text-muted-foreground/60 mt-1">
                                                    {formatRelativeTime(notification.created_at)}
                                                </p>
                                            </Link>
                                        </div>
                                        <div className="flex flex-col gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                            {!notification.read && (
                                                <button
                                                    onClick={(e) => {
                                                        e.stopPropagation();
                                                        markAsRead.mutate(notification.id);
                                                    }}
                                                    disabled={markAsRead.isPending}
                                                    className="p-1.5 rounded hover:bg-primary/10 text-primary transition-colors"
                                                    title="Mark as read"
                                                >
                                                    <Check size={14} />
                                                </button>
                                            )}
                                            <button
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    deleteNotification.mutate(notification.id);
                                                }}
                                                disabled={deleteNotification.isPending}
                                                className="p-1.5 rounded hover:bg-destructive/10 text-destructive transition-colors"
                                                title="Delete"
                                            >
                                                <Trash2 size={14} />
                                            </button>
                                        </div>
                                    </div>
                                ))
                            )}
                        </div>

                        {/* Footer */}
                        {notifications.length > 0 && (
                            <div className="p-3 border-t bg-muted/30 text-center">
                                <Link
                                    href="/notifications"
                                    onClick={() => setIsOpen(false)}
                                    className="text-sm font-bold text-primary hover:text-primary/80 transition-colors"
                                >
                                    View all notifications
                                </Link>
                            </div>
                        )}
                    </motion.div>
                )}
            </AnimatePresence>
        </div>
    );
}
