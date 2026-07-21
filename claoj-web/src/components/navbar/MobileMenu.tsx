'use client';

import { Link, usePathname } from '@/navigation';
import { useAuth } from '@/components/providers/AuthProvider';
import { User, LogOut, Menu, X, Ticket, RefreshCw, Crown, Settings as SettingsIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
import { motion, AnimatePresence, useReducedMotion } from 'framer-motion';
import { useFocusTrap } from '@/hooks/useFocusTrap';
import { useTranslations } from 'next-intl';
import ThemeToggle from './ThemeToggle';
import LanguageSwitcher from './LanguageSwitcher';
import { rememberLoginRedirect } from '@/lib/loginRedirect';

const NAV_LINKS = [
    { href: '/problems', key: 'problems' },
    { href: '/contests', key: 'contests' },
    { href: '/submissions', key: 'submissions' },
    { href: '/users', key: 'users' },
    { href: '/ratings', key: 'ratings' },
    { href: '/organizations', key: 'organizations' },
    { href: '/stats', key: 'stats' },
];

interface MobileMenuProps {
    isOpen: boolean;
    onToggle: () => void;
}

export default function MobileMenu({ isOpen, onToggle }: MobileMenuProps) {
    const { user, logout } = useAuth();
    const pathname = usePathname();
    const t = useTranslations('Navbar');
    const reduceMotion = useReducedMotion();

    const mobileMenuContainerRef = useFocusTrap({
        isActive: isOpen,
        onEscape: () => onToggle(),
        lockBodyScroll: true,
    });

    // Drawer slides in from the right; instant when the user prefers reduced motion.
    const drawerTransition = reduceMotion
        ? { duration: 0 }
        : { type: 'spring' as const, stiffness: 320, damping: 34 };

    return (
        <>
            <button
                className="md:hidden p-2 text-muted-foreground hover:text-foreground"
                onClick={onToggle}
                aria-expanded={isOpen}
                aria-controls="mobile-menu"
                aria-label={isOpen ? t('closeMenu') : t('openMenu')}
                title={isOpen ? t('closeMenu') : t('openMenu')}
            >
                {isOpen ? <X size={24} /> : <Menu size={24} />}
            </button>

            <AnimatePresence>
                {isOpen && (
                    <>
                        {/* Backdrop */}
                        <motion.div
                            className="fixed inset-0 z-[55] bg-black/50 backdrop-blur-sm md:hidden"
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            transition={{ duration: reduceMotion ? 0 : 0.2 }}
                            onClick={onToggle}
                            aria-hidden="true"
                        />

                        {/* Drawer panel */}
                        <motion.div
                            ref={mobileMenuContainerRef}
                            id="mobile-menu"
                            role="navigation"
                            aria-label={t('mobileNavigationMenu')}
                            initial={{ x: '100%' }}
                            animate={{ x: 0 }}
                            exit={{ x: '100%' }}
                            transition={drawerTransition}
                            className="fixed right-0 top-0 z-[60] flex h-[100dvh] w-[85%] max-w-sm flex-col overflow-y-auto border-l border-border bg-card shadow-2xl md:hidden"
                        >
                            {/* Drawer header with its own close control (the navbar toggle
                                sits under the backdrop while the drawer is open). */}
                            <div className="flex h-16 shrink-0 items-center justify-between border-b border-border px-4">
                                <span className="text-lg font-black tracking-tight">{t('menu')}</span>
                                <button
                                    onClick={onToggle}
                                    aria-label={t('closeMenu')}
                                    className="-mr-2 p-2 text-muted-foreground hover:text-foreground"
                                >
                                    <X size={24} />
                                </button>
                            </div>

                            <div className="flex flex-col gap-6 px-4 py-6">
                                {/* Nav Links */}
                                {NAV_LINKS.map((link) => (
                                    <Link
                                        key={link.href}
                                        href={link.href}
                                        onClick={onToggle}
                                        className={cn(
                                            "text-xl font-bold tracking-tight",
                                            pathname.startsWith(link.href) ? "text-primary" : "text-muted-foreground"
                                        )}
                                    >
                                        {t(link.key)}
                                    </Link>
                                ))}
                                <Link
                                    href="/problems/random"
                                    onClick={onToggle}
                                    className={cn(
                                        "text-xl font-bold tracking-tight flex items-center gap-2",
                                        pathname === '/problems/random' ? "text-primary" : "text-muted-foreground"
                                    )}
                                >
                                    <RefreshCw size={20} />
                                    <span>{t('randomProblem')}</span>
                                </Link>

                                {/* Report Issue */}
                                {user && (
                                    <Link
                                        href="/ticket/create"
                                        onClick={onToggle}
                                        className="flex items-center gap-2 text-yellow-500 font-bold"
                                    >
                                        <Ticket size={18} />
                                        <span>{t('report')}</span>
                                    </Link>
                                )}

                                <div className="flex flex-col gap-4 pt-6 border-t border-border">
                                    {/* Language Switcher */}
                                    <LanguageSwitcher variant="mobile" />

                                    {/* Theme Toggle */}
                                    <ThemeToggle variant="mobile" />

                                    {!user ? (
                                        <div className="grid grid-cols-2 gap-4 mt-2">
                                            <Link
                                                href="/login"
                                                className="flex items-center justify-center h-12 rounded border border-border font-bold text-muted-foreground hover:bg-muted transition-colors"
                                                onClick={() => {
                                                    rememberLoginRedirect(window.location.pathname);
                                                    onToggle();
                                                }}
                                            >
                                                {t('login')}
                                            </Link>
                                            <Link
                                                href="/register"
                                                className="flex items-center justify-center h-12 rounded bg-primary text-white font-bold hover:bg-primary/90 transition-all"
                                                onClick={onToggle}
                                            >
                                                {t('signup')}
                                            </Link>
                                        </div>
                                    ) : (
                                        <div className="flex flex-col gap-4 mt-2">
                                            <Link
                                                href={`/user/${user.username}`}
                                                className="flex items-center gap-3 p-4 rounded bg-primary/10 text-primary font-bold"
                                                onClick={onToggle}
                                            >
                                                <User size={20} />
                                                <span>{user.username}</span>
                                            </Link>

                                            {/* Admin Section for Mobile */}
                                            {(user.is_staff || user.is_admin) && (
                                                <div className="space-y-2">
                                                    <Link
                                                        href="/admin"
                                                        className="flex items-center gap-3 p-4 rounded bg-gradient-to-r from-amber-500/20 to-orange-500/20 text-amber-400 font-bold border border-amber-500/30"
                                                        onClick={onToggle}
                                                    >
                                                        <Crown size={20} />
                                                        <span>{t('adminDashboard')}</span>
                                                    </Link>
                                                    <div className="grid grid-cols-2 gap-2 px-1">
                                                        <Link
                                                            href="/admin/problems"
                                                            onClick={onToggle}
                                                            className="flex items-center justify-center gap-2 p-3 rounded bg-muted text-xs font-bold text-muted-foreground hover:text-foreground"
                                                        >
                                                            {t('problems')}
                                                        </Link>
                                                        <Link
                                                            href="/admin/contests"
                                                            onClick={onToggle}
                                                            className="flex items-center justify-center gap-2 p-3 rounded bg-muted text-xs font-bold text-muted-foreground hover:text-foreground"
                                                        >
                                                            {t('contests')}
                                                        </Link>
                                                        <Link
                                                            href="/admin/users"
                                                            onClick={onToggle}
                                                            className="flex items-center justify-center gap-2 p-3 rounded bg-muted text-xs font-bold text-muted-foreground hover:text-foreground"
                                                        >
                                                            {t('users')}
                                                        </Link>
                                                        <Link
                                                            href="/admin/tickets"
                                                            onClick={onToggle}
                                                            className="flex items-center justify-center gap-2 p-3 rounded bg-muted text-xs font-bold text-muted-foreground hover:text-foreground"
                                                        >
                                                            {t('tickets')}
                                                        </Link>
                                                    </div>
                                                </div>
                                            )}

                                            <Link
                                                href="/settings"
                                                className="flex items-center gap-3 p-4 rounded bg-muted font-bold"
                                                onClick={onToggle}
                                            >
                                                <SettingsIcon size={20} />
                                                <span>{t('settings')}</span>
                                            </Link>
                                            <button
                                                onClick={() => { logout(); onToggle(); }}
                                                className="flex items-center gap-3 p-4 rounded bg-red-500/10 text-red-400 font-bold text-left"
                                            >
                                                <LogOut size={20} />
                                                <span>{t('logout')}</span>
                                            </button>
                                        </div>
                                    )}
                                </div>
                            </div>
                        </motion.div>
                    </>
                )}
            </AnimatePresence>
        </>
    );
}
