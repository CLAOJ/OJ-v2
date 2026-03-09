'use client';

import { Link } from '@/navigation';
import { useAuth } from '@/components/providers/AuthProvider';
import { Ticket } from 'lucide-react';
import NotificationBell from '@/components/notifications/NotificationBell';
import WebSocketStatusIndicator from '@/components/common/WebSocketStatus';
import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { AdminNavbarBadge, AdminQuickAccessButton, AdminSidebar } from '@/components/admin';
import Logo from '@/components/navbar/Logo';
import DesktopNav from '@/components/navbar/DesktopNav';
import LanguageSwitcher from '@/components/navbar/LanguageSwitcher';
import ThemeToggle from '@/components/navbar/ThemeToggle';
import UserMenu from '@/components/navbar/UserMenu';
import MobileMenu from '@/components/navbar/MobileMenu';
import { useContestTimer } from '@/hooks/useContestTimer';

export default function Navbar() {
    const { user, logout } = useAuth();
    const t = useTranslations('Navbar');
    const [mounted, setMounted] = useState(false);
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
    const [userMenuOpen, setUserMenuOpen] = useState(false);
    const [adminSidebarOpen, setAdminSidebarOpen] = useState(false);

    // Contest timer state (placeholder - in real app, this would come from context)
    const [inContest] = useState(false);
    const [contestEndTime] = useState<Date | null>(null);
    const { currentTime, formatTimeRemaining } = useContestTimer(inContest, contestEndTime);

    useEffect(() => {
        setMounted(true);
    }, []);

    const handleLoginRedirect = () => {
        const pathname = window.location.pathname;
        if (pathname && !pathname.includes('/login')) {
            sessionStorage.setItem('loginRedirectUrl', pathname);
        }
        window.location.href = '/login';
    };

    return (
        <>
            <header className="sticky top-0 z-50 w-full border-b bg-card/95 backdrop-blur-md shadow-lg">
                <div className="container mx-auto px-4">
                    <div className="flex h-16 items-center justify-between">
                        {/* Logo and Nav Links */}
                        <div className="flex items-center gap-6">
                            {/* Logo */}
                            <Link href="/" className="flex items-center gap-2 hover:opacity-80 transition-opacity">
                                <Logo />
                            </Link>

                            {/* Desktop Nav */}
                            <DesktopNav />
                        </div>

                        {/* Right Side Actions */}
                        <div className="flex items-center gap-3">
                            {/* Report Issue Button */}
                            {user && (
                                <Link
                                    href="/tickets/new"
                                    className="hidden md:flex items-center gap-1.5 px-3 py-1.5 rounded bg-yellow-500/10 text-yellow-500 hover:bg-yellow-500/20 transition-colors text-xs font-bold"
                                    title="Report issue"
                                >
                                    <Ticket size={14} />
                                    <span>Report</span>
                                </Link>
                            )}

                            {/* Language Flags */}
                            <LanguageSwitcher />

                            {/* Theme Toggle */}
                            <div className="hidden md:block">
                                <ThemeToggle />
                            </div>

                            {/* WebSocket Status */}
                            <WebSocketStatusIndicator />

                            {/* Notification Bell */}
                            {user && <NotificationBell />}

                            {/* User Menu */}
                            {user ? (
                                <>
                                    {/* Admin Badge */}
                                    {user.is_staff && (
                                        <AdminNavbarBadge onClick={() => setAdminSidebarOpen(true)} />
                                    )}

                                    <UserMenu
                                        username={user.username}
                                        isStaff={user.is_staff}
                                        isOpen={userMenuOpen}
                                        onToggle={() => setUserMenuOpen(!userMenuOpen)}
                                        onClose={() => setUserMenuOpen(false)}
                                        onLogout={logout}
                                    />
                                </>
                            ) : (
                                <div className="flex items-center gap-2">
                                    <button
                                        onClick={handleLoginRedirect}
                                        className="text-sm font-medium text-muted-foreground hover:text-foreground px-3 py-1.5 rounded hover:bg-accent/10 transition-colors"
                                    >
                                        {t('login')}
                                    </button>
                                    <Link
                                        href="/register"
                                        className="px-4 py-1.5 rounded bg-primary text-white text-sm font-bold hover:bg-primary/90 transition-all"
                                    >
                                        {t('signup')}
                                    </Link>
                                </div>
                            )}

                            {/* Mobile Menu Button */}
                            <MobileMenu isOpen={mobileMenuOpen} onToggle={() => setMobileMenuOpen(!mobileMenuOpen)} />
                        </div>
                    </div>
                </div>

                {/* Contest Timer Banner */}
                {inContest && currentTime && (
                    <div className="bg-card border-t border-b border-primary px-4 py-2 text-center">
                        <Link href="/contest/current" className="text-sm font-medium text-white">
                            <span className="text-primary">Current Contest:</span>{' '}
                            <span className="font-bold">Contest Name</span>{' '}
                            <span className="mx-2">|</span>{' '}
                            <span className="text-yellow-400">Ends in {formatTimeRemaining()}</span>
                        </Link>
                    </div>
                )}
            </header>

            {/* Admin Sidebar */}
            <AdminSidebar isOpen={adminSidebarOpen} onClose={() => setAdminSidebarOpen(false)} />

            {/* Admin Quick Access Button - Fixed position */}
            <AdminQuickAccessButton onClick={() => setAdminSidebarOpen(true)} />
        </>
    );
}
