'use client';

import { Link, usePathname } from '@/navigation';
import { useAuth } from '@/components/providers/AuthProvider';
import { User, LogOut, Menu, X, RefreshCw, Crown, Settings as SettingsIcon, Ticket } from 'lucide-react';
import { cn } from '@/lib/utils';
import { motion, AnimatePresence, useReducedMotion } from 'framer-motion';
import { useFocusTrap } from '@/hooks/useFocusTrap';
import { useTranslations } from 'next-intl';
import ThemeToggle from './ThemeToggle';
import LanguageSwitcher from './LanguageSwitcher';
import { rememberLoginRedirect } from '@/lib/loginRedirect';
import { findActiveKeys, type NavLink } from '@/lib/navbar';

interface MobileMenuProps {
    isOpen: boolean;
    onToggle: () => void;
    links: NavLink[];
}

export default function MobileMenu({ isOpen, onToggle, links }: MobileMenuProps) {
    const { user, loading, logout } = useAuth();
    const pathname = usePathname();
    const t = useTranslations('Navbar');
    const reduceMotion = useReducedMotion();
    const activeKeys = findActiveKeys(links, pathname);

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
                className="xl:hidden p-2 text-muted-foreground hover:text-foreground"
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
                            className="fixed inset-0 z-[55] bg-black/50 xl:hidden"
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
                            className="fixed right-0 top-0 z-[60] flex h-[100dvh] w-[85%] max-w-sm flex-col overflow-y-auto border-l border-border bg-card shadow-lg xl:hidden"
                        >
                            {/* Drawer header with its own close control (the navbar toggle
                                sits under the backdrop while the drawer is open). */}
                            <div className="flex h-16 shrink-0 items-center justify-between border-b border-border px-4">
                                <span className="text-lg font-semibold tracking-tight">{t('menu')}</span>
                                <button
                                    onClick={onToggle}
                                    aria-label={t('closeMenu')}
                                    className="-mr-2 p-2 text-muted-foreground hover:text-foreground"
                                >
                                    <X size={24} />
                                </button>
                            </div>

                            <div className="flex flex-col gap-6 px-4 py-6">
                                {/* Nav Links: same hierarchy as the DB/v1 tree, children indented */}
                                {links.map((link) => (
                                    <div key={link.href} className="flex flex-col gap-3">
                                        <Link prefetch={false}
                                            href={link.href}
                                            onClick={onToggle}
                                            aria-current={activeKeys.has(link.key) ? 'page' : undefined}
                                            className={cn(
                                                "text-xl font-semibold tracking-tight",
                                                activeKeys.has(link.key) ? "text-primary" : "text-muted-foreground"
                                            )}
                                        >
                                            {link.label ?? t(link.tKey ?? link.key)}
                                        </Link>
                                        {link.children?.length ? (
                                            <div className="ml-2 flex flex-col gap-3 border-l border-border pl-4">
                                                {link.children.map((child) => (
                                                    <Link prefetch={false}
                                                        key={child.href}
                                                        href={child.href}
                                                        onClick={onToggle}
                                                        aria-current={activeKeys.has(child.key) ? 'page' : undefined}
                                                        className={cn(
                                                            "text-base font-medium tracking-tight",
                                                            activeKeys.has(child.key)
                                                                ? "text-primary"
                                                                : "text-muted-foreground hover:text-foreground"
                                                        )}
                                                    >
                                                        {child.label ?? t(child.tKey ?? child.key)}
                                                    </Link>
                                                ))}
                                            </div>
                                        ) : null}
                                    </div>
                                ))}
                                <Link prefetch={false}
                                    href="/problems/random"
                                    onClick={onToggle}
                                    className={cn(
                                        "text-xl font-semibold tracking-tight flex items-center gap-2",
                                        pathname === '/problems/random' ? "text-primary" : "text-muted-foreground"
                                    )}
                                >
                                    <RefreshCw size={20} />
                                    <span>{t('randomProblem')}</span>
                                </Link>

                                {/* Report issue: v2's ticket shortcut (v1 opened the
                                    ticket form from the user area, not the nav row). */}
                                {user && (
                                    <Link prefetch={false}
                                        href="/ticket/create"
                                        onClick={onToggle}
                                        className="text-xl font-semibold tracking-tight flex items-center gap-2 text-warning"
                                    >
                                        <Ticket size={20} />
                                        <span>{t('report')}</span>
                                    </Link>
                                )}

                                <div className="flex flex-col gap-4 pt-6 border-t border-border">
                                    {/* Language Switcher */}
                                    <LanguageSwitcher variant="mobile" />

                                    {/* Theme Toggle */}
                                    <ThemeToggle variant="mobile" />

                                    {/* Nothing until the session check settles, so a signed-in
                                        visitor never sees Login flash in the drawer. */}
                                    {loading ? null : !user ? (
                                        <div className="grid grid-cols-2 gap-4 mt-2">
                                            <Link prefetch={false}
                                                href="/login"
                                                className="flex items-center justify-center h-12 rounded border border-border font-medium text-muted-foreground hover:bg-muted transition-colors"
                                                onClick={() => {
                                                    rememberLoginRedirect(window.location.pathname);
                                                    onToggle();
                                                }}
                                            >
                                                {t('login')}
                                            </Link>
                                            <Link prefetch={false}
                                                href="/register"
                                                className="flex items-center justify-center h-12 rounded bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-all"
                                                onClick={onToggle}
                                            >
                                                {t('signup')}
                                            </Link>
                                        </div>
                                    ) : (
                                        <div className="flex flex-col gap-4 mt-2">
                                            <Link prefetch={false}
                                                href={`/user/${user.username}`}
                                                className="flex items-center gap-3 p-4 rounded bg-primary/10 text-primary font-medium"
                                                onClick={onToggle}
                                            >
                                                <User size={20} />
                                                <span data-name>{user.username}</span>
                                            </Link>

                                            {/* Admin Section for Mobile */}
                                            {(user.is_staff || user.is_admin) && (
                                                <div className="space-y-2">
                                                    <Link prefetch={false}
                                                        href="/admin"
                                                        className="flex items-center gap-3 p-4 rounded bg-warning/15 text-warning font-medium border border-warning/30"
                                                        onClick={onToggle}
                                                    >
                                                        <Crown size={20} />
                                                        <span>{t('adminDashboard')}</span>
                                                    </Link>
                                                    <div className="grid grid-cols-2 gap-2 px-1">
                                                        <Link prefetch={false}
                                                            href="/admin/problems"
                                                            onClick={onToggle}
                                                            className="flex items-center justify-center gap-2 p-3 rounded bg-muted text-xs font-medium text-muted-foreground hover:text-foreground"
                                                        >
                                                            {t('problems')}
                                                        </Link>
                                                        <Link prefetch={false}
                                                            href="/admin/contests"
                                                            onClick={onToggle}
                                                            className="flex items-center justify-center gap-2 p-3 rounded bg-muted text-xs font-medium text-muted-foreground hover:text-foreground"
                                                        >
                                                            {t('contests')}
                                                        </Link>
                                                        <Link prefetch={false}
                                                            href="/admin/users"
                                                            onClick={onToggle}
                                                            className="flex items-center justify-center gap-2 p-3 rounded bg-muted text-xs font-medium text-muted-foreground hover:text-foreground"
                                                        >
                                                            {t('users')}
                                                        </Link>
                                                        <Link prefetch={false}
                                                            href="/admin/tickets"
                                                            onClick={onToggle}
                                                            className="flex items-center justify-center gap-2 p-3 rounded bg-muted text-xs font-medium text-muted-foreground hover:text-foreground"
                                                        >
                                                            {t('tickets')}
                                                        </Link>
                                                    </div>
                                                </div>
                                            )}

                                            <Link prefetch={false}
                                                href="/settings"
                                                className="flex items-center gap-3 p-4 rounded bg-muted font-medium"
                                                onClick={onToggle}
                                            >
                                                <SettingsIcon size={20} />
                                                <span>{t('settings')}</span>
                                            </Link>
                                            <button
                                                onClick={() => { logout(); onToggle(); }}
                                                className="flex items-center gap-3 p-4 rounded bg-red-500/10 text-red-400 font-medium text-left"
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
