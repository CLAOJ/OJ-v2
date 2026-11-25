'use client';

import { Link } from '@/navigation';
import { useAuth } from '@/components/providers/AuthProvider';
import NotificationBell from '@/components/notifications/NotificationBell';
import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import Logo from '@/components/navbar/Logo';
import DesktopNav from '@/components/navbar/DesktopNav';
import LanguageSwitcher from '@/components/navbar/LanguageSwitcher';
import ThemeToggle from '@/components/navbar/ThemeToggle';
import UserMenu from '@/components/navbar/UserMenu';
import MobileMenu from '@/components/navbar/MobileMenu';
import { rememberLoginRedirect } from '@/lib/loginRedirect';
import { buildNavLinks, type NavigationItem, type NavLink } from '@/lib/navbar';
import api from '@/lib/api';

// v2-only entries the DB table never held; appended after its rows. Report
// needs a session, so it is added conditionally.
const V2_ONLY_LINKS: NavLink[] = [{ key: 'ratings', href: '/ratings', tKey: 'ratings' }];
const REPORT_LINK: NavLink = { key: 'report', href: '/ticket/create', tKey: 'report' };

export default function Navbar() {
    const { user, logout } = useAuth();
    const t = useTranslations('Navbar');
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
    const [userMenuOpen, setUserMenuOpen] = useState(false);
    const [navigation, setNavigation] = useState<NavigationItem[] | null>(null);

    useEffect(() => {
        let cancelled = false;
        api.get<{ data: NavigationItem[] }>('/navigation')
            .then((res) => {
                if (!cancelled) setNavigation(res.data?.data ?? []);
            })
            // An unreachable API leaves navigation null: buildNavLinks then
            // renders the static fallback, so the header never disappears.
            .catch(() => {});
        return () => { cancelled = true; };
    }, []);

    const navLinks = buildNavLinks(
        navigation,
        user ? [...V2_ONLY_LINKS, REPORT_LINK] : V2_ONLY_LINKS,
    );

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
                            <Link href="/" className="flex shrink-0 items-center gap-2 hover:opacity-80 transition-opacity">
                                <Logo />
                            </Link>

                            {/* Desktop Nav */}
                            <DesktopNav links={navLinks} />
                        </div>

                        {/* Right Side Actions */}
                        <div className="flex items-center gap-3">
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
                            ) : (
                                <div className="hidden xl:flex items-center gap-2">
                                    <button
                                        onClick={handleLoginRedirect}
                                        className="text-sm font-medium text-muted-foreground hover:text-foreground px-3 py-1.5 rounded hover:bg-accent/10 transition-colors"
                                    >
                                        {t('login')}
                                    </button>
                                    <Link
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
