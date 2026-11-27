'use client';

import { Link } from '@/navigation';
import { useAuth } from '@/components/providers/AuthProvider';
import { Ticket } from 'lucide-react';
import NotificationBell from '@/components/notifications/NotificationBell';
import { useEffect, useState } from 'react';
import { useMessages, useTranslations } from 'next-intl';
import Logo from '@/components/navbar/Logo';
import DesktopNav from '@/components/navbar/DesktopNav';
import LanguageSwitcher from '@/components/navbar/LanguageSwitcher';
import ThemeToggle from '@/components/navbar/ThemeToggle';
import UserMenu from '@/components/navbar/UserMenu';
import MobileMenu from '@/components/navbar/MobileMenu';
import { rememberLoginRedirect } from '@/lib/loginRedirect';
import { buildNavLinks, type NavigationItem } from '@/lib/navbar';
import api from '@/lib/api';

// The nav row mirrors v1's DB rows only: v2-only pages (Ratings) and the
// ticket shortcut live outside it, as they did before the navbar port.

export default function Navbar({
    initialNavigation = null,
    hasSession = true,
}: {
    /** Server-fetched rows (layout.tsx); null = fetch them here instead. */
    initialNavigation?: NavigationItem[] | null;
    /** False when the request carried no session cookie: certainly logged out. */
    hasSession?: boolean;
}) {
    const { user, loading, logout } = useAuth();
    const t = useTranslations('Navbar');
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
    const [userMenuOpen, setUserMenuOpen] = useState(false);
    const [navigation, setNavigation] = useState<NavigationItem[] | null>(initialNavigation);

    useEffect(() => {
        if (initialNavigation) return;
        let cancelled = false;
        api.get<{ data: NavigationItem[] }>('/navigation')
            .then((res) => {
                if (!cancelled) setNavigation(res.data?.data ?? []);
            })
            // An unreachable API leaves navigation null: buildNavLinks then
            // renders the static fallback, so the header never disappears.
            .catch(() => {});
        return () => { cancelled = true; };
    }, [initialNavigation]);

    // v1 translated DB labels with gettext; the catalog map is the v2
    // equivalent, and labels it lacks render verbatim.
    const messages = useMessages();
    const dbLabels = (messages.Navbar as { dbLabels?: Record<string, string> } | undefined)?.dbLabels;
    const navLinks = buildNavLinks(navigation, [], dbLabels);

    const handleLoginRedirect = () => {
        // rememberLoginRedirect filters out the credential pages, so clicking
        // "Login" from /register no longer parks /register as the destination.
        rememberLoginRedirect(window.location.pathname);
        window.location.href = '/login';
    };

    return (
        <>
            <header className="site-nav sticky top-0 z-50 w-full border-b bg-card text-foreground shadow-md print:hidden">
                <div className="container mx-auto px-4">
                    <div className="flex h-16 items-center justify-between">
                        {/* Logo and Nav Links */}
                        <div className="flex items-center gap-6">
                            {/* Logo */}
                            <Link prefetch={false} href="/" className="flex shrink-0 items-center gap-2 hover:opacity-80 transition-opacity">
                                <Logo />
                            </Link>

                            {/* Desktop Nav */}
                            <DesktopNav links={navLinks} />
                        </div>

                        {/* Right Side Actions */}
                        <div className="flex items-center gap-3">
                            {/* Report Issue Button (ticket icon, as before the nav port) */}
                            {user && (
                                <Link prefetch={false}
                                    href="/ticket/create"
                                    className="hidden xl:flex items-center gap-1.5 px-3 py-1.5 rounded bg-warning/10 text-warning hover:bg-warning/20 transition-colors text-xs font-bold"
                                    title={t('report')}
                                >
                                    <Ticket size={14} />
                                    <span className="hidden 2xl:inline">{t('reportShort')}</span>
                                </Link>
                            )}

                            {/* Language Flags (moved into the drawer on mobile) */}
                            <div className="hidden xl:block">
                                <LanguageSwitcher />
                            </div>

                            {/* Theme Toggle */}
                            <div className="hidden xl:block">
                                <ThemeToggle />
                            </div>

                            {/* Notification Bell */}
                            {user && <NotificationBell />}

                            {/* User Menu */}
                            {user ? (
                                <>
                                    <UserMenu
                                        username={user.username}
                                        isStaff={user.is_staff || user.is_admin}
                                        isOpen={userMenuOpen}
                                        onToggle={() => setUserMenuOpen(!userMenuOpen)}
                                        onClose={() => setUserMenuOpen(false)}
                                        onLogout={logout}
                                    />
                                </>
                            ) : loading && hasSession ? null : (
                                <div className="hidden xl:flex items-center gap-2">
                                    <button
                                        onClick={handleLoginRedirect}
                                        className="text-sm font-medium text-muted-foreground hover:text-foreground px-3 py-1.5 rounded hover:bg-accent/10 transition-colors"
                                    >
                                        {t('login')}
                                    </button>
                                    <Link prefetch={false}
                                        href="/register"
                                        className="px-4 py-1.5 rounded bg-primary text-primary-foreground text-sm font-bold hover:bg-primary/90 transition-colors"
                                    >
                                        {t('signup')}
                                    </Link>
                                </div>
                            )}

                            {/* Mobile Menu Button */}
                            <MobileMenu isOpen={mobileMenuOpen} onToggle={() => setMobileMenuOpen(!mobileMenuOpen)} links={navLinks} />
                        </div>
                    </div>
                </div>

            </header>
        </>
    );
}
