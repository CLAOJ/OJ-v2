'use client';

import { useState, useSyncExternalStore } from 'react';
import { useTranslations } from 'next-intl';
import { useAuth } from '@/components/providers/AuthProvider';
import { Link } from '@/navigation';
import { Crown, X } from 'lucide-react';

const DISMISS_KEY = 'claoj.adminShortcuts.hidden';

const SHORTCUTS = [
    { href: '/admin', key: 'banner.dashboard' },
    { href: '/admin/problems/create', key: 'dashboard.newProblem' },
    { href: '/admin/contests/create', key: 'dashboard.newContest' },
    { href: '/admin/tickets', key: 'nav.tickets' },
];

const readHidden = () => {
    try { return localStorage.getItem(DISMISS_KEY) === '1'; } catch { return false; }
};
const subscribe = (onChange: () => void) => {
    window.addEventListener('storage', onChange);
    return () => window.removeEventListener('storage', onChange);
};

// Staff-only shortcut strip on the home page. The server snapshot reports
// "hidden" so it never flashes in and out during hydration.
export function AdminWelcomeBanner() {
    const { user } = useAuth();
    const t = useTranslations('Admin');
    const storedHidden = useSyncExternalStore(subscribe, readHidden, () => true);
    const [dismissed, setDismissed] = useState(false);

    if ((!user?.is_staff && !user?.is_admin) || storedHidden || dismissed) return null;

    const dismiss = () => {
        setDismissed(true);
        try { localStorage.setItem(DISMISS_KEY, '1'); } catch { /* private mode */ }
    };

    return (
        <section
            aria-label={t('banner.label')}
            className="mb-6 flex flex-wrap items-center gap-x-4 gap-y-2 rounded-md border bg-card px-4 py-2.5 text-sm"
        >
            <span className="flex items-center gap-2 font-semibold">
                <Crown size={16} className="text-warning" />
                {t('banner.label')}
            </span>
            <nav className="flex flex-wrap items-center gap-x-4 gap-y-1">
                {SHORTCUTS.map(s => (
                    <Link key={s.href} href={s.href} className="text-accent hover:underline">
                        {t(s.key)}
                    </Link>
                ))}
            </nav>
            <button
                onClick={dismiss}
                aria-label={t('banner.dismiss')}
                title={t('banner.dismiss')}
                className="ml-auto p-1 rounded text-muted-foreground hover:text-foreground hover:bg-muted"
            >
                <X size={16} />
            </button>
        </section>
    );
}
