'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { useRouter } from '@/navigation';
import { useAuth } from '@/components/providers/AuthProvider';
import { adminUserApi, adminErrorMessage } from '@/lib/adminApi';
import { Eye, LogOut } from 'lucide-react';

// Site-wide strip shown whenever the session is an impersonation. It lives
// outside the admin layout on purpose: most impersonated accounts are not
// staff, so the admin layout would bounce them to "/" with no way back.
export function ImpersonationBanner() {
    const { user } = useAuth();
    const t = useTranslations('Admin.impersonation');
    const router = useRouter();
    const [leaving, setLeaving] = useState(false);
    const [error, setError] = useState('');

    if (!user?.is_impersonating) return null;

    const exit = async () => {
        setLeaving(true);
        setError('');
        try {
            await adminUserApi.exitImpersonation();
            // AuthProvider refetches /user/me on this event, so the whole UI
            // (navbar, permissions, this banner) switches back to the actor.
            window.dispatchEvent(new Event('auth-state-change'));
            router.push('/admin/users');
            router.refresh();
        } catch (err) {
            setError(adminErrorMessage(err, t('exitError')));
            setLeaving(false);
        }
    };

    return (
        <div role="status" className="bg-warning/15 border-b border-warning/40 text-sm">
            <div className="container mx-auto px-4 py-2 flex flex-wrap items-center gap-x-3 gap-y-2">
                <Eye size={16} className="text-warning shrink-0" aria-hidden="true" />
                <span>
                    {t('viewingAs', { username: user.username })}
                    {user.impersonator_username && (
                        <span className="text-muted-foreground"> · {t('startedBy', { username: user.impersonator_username })}</span>
                    )}
                </span>
                {error && <span role="alert" className="text-destructive">{error}</span>}
                <button
                    type="button"
                    onClick={exit}
                    disabled={leaving}
                    className="ml-auto flex items-center gap-1.5 px-3 py-1 rounded-md bg-warning/20 hover:bg-warning/30 font-medium transition-colors disabled:opacity-60"
                >
                    <LogOut size={14} aria-hidden="true" />
                    {t('exit')}
                </button>
            </div>
        </div>
    );
}