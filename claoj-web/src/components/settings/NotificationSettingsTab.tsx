'use client';

import api from '@/lib/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Loader2, Bell } from 'lucide-react';
import { cn } from '@/lib/utils';

interface NotificationPreferences {
    email_on_submission_result?: boolean;
    email_on_contest_start?: boolean;
    email_on_ticket_reply?: boolean;
    web_on_submission_result?: boolean;
    web_on_contest_start?: boolean;
    web_on_ticket_reply?: boolean;
}

export default function NotificationSettingsTab() {
    const queryClient = useQueryClient();

    const { data: preferences, isLoading } = useQuery<{
        email_on_submission_result?: boolean;
        email_on_contest_start?: boolean;
        email_on_ticket_reply?: boolean;
        web_on_submission_result?: boolean;
        web_on_contest_start?: boolean;
        web_on_ticket_reply?: boolean;
    }>({
        queryKey: ['notifications', 'preferences'],
        queryFn: async () => {
            const res = await api.get<NotificationPreferences>('/notifications/preferences');
            return res.data;
        },
    });

    const { mutate: updatePreferences, isPending } = useMutation({
        mutationFn: async (data: Record<string, boolean>) => {
            await api.patch('/notifications/preferences', data);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['notifications', 'preferences'] });
        },
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Failed to update preferences');
        }
    });

    const ToggleSwitch = ({ label, description, checked, onChange }: {
        label: string;
        description: string;
        checked: boolean;
        onChange: (checked: boolean) => void;
    }) => (
        <div className="flex items-center justify-between p-4 rounded-xl border bg-muted/30">
            <div className="space-y-1">
                <p className="text-sm font-bold">{label}</p>
                <p className="text-xs text-muted-foreground">{description}</p>
            </div>
            <button
                onClick={() => onChange(!checked)}
                className={cn(
                    "relative w-12 h-6 rounded-full transition-colors",
                    checked ? "bg-primary" : "bg-muted"
                )}
            >
                <div
                    className={cn(
                        "absolute top-1 w-4 h-4 rounded-full bg-white transition-transform",
                        checked ? "left-7" : "left-1"
                    )}
                />
            </button>
        </div>
    );

    if (isLoading) {
        return (
            <div className="flex items-center justify-center py-12">
                <Loader2 size={24} className="animate-spin text-muted-foreground" />
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <section className="space-y-4">
                <div className="flex items-center gap-2 text-primary font-bold">
                    <Bell size={18} />
                    Notification Preferences
                </div>
                <p className="text-sm text-muted-foreground">
                    Choose how you want to receive notifications.
                </p>

                <div className="space-y-4">
                    <h3 className="text-sm font-bold text-muted-foreground uppercase tracking-wider">Email Notifications</h3>
                    <ToggleSwitch
                        label="Submission Results"
                        description="Receive email when your submission is graded"
                        checked={preferences?.email_on_submission_result ?? true}
                        onChange={(val) => updatePreferences({ email_on_submission_result: val })}
                    />
                    <ToggleSwitch
                        label="Contest Start"
                        description="Receive email when a contest you joined is about to start"
                        checked={preferences?.email_on_contest_start ?? true}
                        onChange={(val) => updatePreferences({ email_on_contest_start: val })}
                    />
                    <ToggleSwitch
                        label="Ticket Replies"
                        description="Receive email when a ticket receives a reply"
                        checked={preferences?.email_on_ticket_reply ?? true}
                        onChange={(val) => updatePreferences({ email_on_ticket_reply: val })}
                    />

                    <h3 className="text-sm font-bold text-muted-foreground uppercase tracking-wider mt-6">Web Notifications</h3>
                    <ToggleSwitch
                        label="Submission Results"
                        description="Show web notification when your submission is graded"
                        checked={preferences?.web_on_submission_result ?? true}
                        onChange={(val) => updatePreferences({ web_on_submission_result: val })}
                    />
                    <ToggleSwitch
                        label="Contest Start"
                        description="Show web notification when a contest you joined is about to start"
                        checked={preferences?.web_on_contest_start ?? true}
                        onChange={(val) => updatePreferences({ web_on_contest_start: val })}
                    />
                    <ToggleSwitch
                        label="Ticket Replies"
                        description="Show web notification when a ticket receives a reply"
                        checked={preferences?.web_on_ticket_reply ?? true}
                        onChange={(val) => updatePreferences({ web_on_ticket_reply: val })}
                    />
                </div>
            </section>
        </div>
    );
}
