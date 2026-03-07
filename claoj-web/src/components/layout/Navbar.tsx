'use client';

import { Link, usePathname, useRouter } from '@/navigation';
import { useAuth } from '@/components/providers/AuthProvider';
import { useTheme } from 'next-themes';
import { useLocale } from 'next-intl';
import { Moon, Sun, User, LogOut, Menu, X, Settings as SettingsIcon, Ticket, ChevronDown, RefreshCw, Crown, Shield } from 'lucide-react';
import NotificationBell from '@/components/notifications/NotificationBell';
import WebSocketStatusIndicator from '@/components/common/WebSocketStatus';
import { useFocusTrap } from '@/hooks/useFocusTrap';
import { GB_FLAG_SVG, VI_FLAG_SVG } from '@/lib/flag-icons';
import { useState, useEffect, useRef } from 'react';
import { cn } from '@/lib/utils';
import { motion, AnimatePresence, useReducedMotion } from 'framer-motion';
import { useTranslations } from 'next-intl';
import { AdminNavbarBadge, AdminQuickAccessButton, AdminSidebar } from '@/components/admin';

// Time conversion constants
const MS_PER_SECOND = 1000;
const MS_PER_MINUTE = MS_PER_SECOND * 60;
const MS_PER_HOUR = MS_PER_MINUTE * 60;

function Logo() {
    const { theme } = useTheme();
    const [mounted, setMounted] = useState(false);

    useEffect(() => {
        setMounted(true);
    }, []);

    // Use dark logo for light theme, light logo for dark theme
    const logoSrc = mounted && theme === 'dark'
        ? '/static/claoj-logo-light.png'
        : '/static/claoj-logo-dark.png';

    return (
        <img
            src={logoSrc}
            alt="CLAOJ"
            className="h-8 w-auto object-contain"
        />
    );
}

const NAV_LINKS = [
    { href: '/problems', key: 'problems' },
    { href: '/contests', key: 'contests' },
    { href: '/submissions', key: 'submissions' },
    { href: '/users', key: 'users' },
    { href: '/ratings', key: 'ratings' },
    { href: '/organizations', key: 'organizations' },
    { href: '/stats', key: 'stats' },
];

export default function Navbar() {
    const { user, logout } = useAuth();
    const { theme, setTheme } = useTheme();
    const t = useTranslations('Navbar');
    const locale = useLocale();
    const pathname = usePathname();
    const router = useRouter();
    const [mounted, setMounted] = useState(false);
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
    const [userMenuOpen, setUserMenuOpen] = useState(false);
    const [currentTime, setCurrentTime] = useState<Date | null>(null);
    const [contestEndTime, setContestEndTime] = useState<Date | null>(null);
    const [adminSidebarOpen, setAdminSidebarOpen] = useState(false);
    const mobileMenuRef = useRef<HTMLDivElement>(null);
    const userMenuRef = useRef<HTMLDivElement>(null);
    const reduceMotion = useReducedMotion();

    // Focus trap for user dropdown menu
    const userMenuContainerRef = useFocusTrap({
        isActive: userMenuOpen,
        onEscape: () => setUserMenuOpen(false),
    });

    // Combine refs for user menu container
    const setUserMenuContainerRef = (node: HTMLDivElement) => {
        userMenuRef.current = node;
        if (userMenuContainerRef.current) {
            userMenuContainerRef.current = node;
        }
    };

    // Click outside to close user menu
    useEffect(() => {
        if (!userMenuOpen) return;

        const handleClickOutside = (e: MouseEvent) => {
            if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
                setUserMenuOpen(false);
            }
        };

        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, [userMenuOpen]);

    // Simulate contest participation (in real app, this would come from auth context)
    const [inContest, setInContest] = useState(false);

    // Focus trap for mobile menu
    useEffect(() => {
        if (!mobileMenuOpen) return;

        const menu = mobileMenuRef.current;
        if (!menu) return;

        const focusableElements = menu.querySelectorAll<HTMLElement>(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"]), [role="button"]'
        );

        if (focusableElements.length === 0) return;

        const firstEl = focusableElements[0];
        const lastEl = focusableElements[focusableElements.length - 1];

        // Focus first element when menu opens
        firstEl.focus();

        const handleTabKey = (e: KeyboardEvent) => {
            if (e.key !== 'Tab') return;

            if (e.shiftKey && document.activeElement === firstEl) {
                e.preventDefault();
                lastEl.focus();
            } else if (!e.shiftKey && document.activeElement === lastEl) {
                e.preventDefault();
                firstEl.focus();
            }
        };

        const handleEscape = () => setMobileMenuOpen(false);

        document.addEventListener('keydown', handleTabKey);
        document.addEventListener('keydown', handleEscape);
        document.body.style.overflow = 'hidden';

        return () => {
            document.removeEventListener('keydown', handleTabKey);
            document.removeEventListener('keydown', handleEscape);
            document.body.style.overflow = 'unset';
        };
    }, [mobileMenuOpen]);

    useEffect(() => {
        setMounted(true);
    }, []);

    // Timer countdown for contest
    useEffect(() => {
        if (!inContest || !contestEndTime) return;

        const timer = setInterval(() => {
            const now = new Date();
            setCurrentTime(now);

            if (now >= contestEndTime) {
                // Contest ended
                setInContest(false);
            }
        }, 1000);

        return () => clearInterval(timer);
    }, [inContest, contestEndTime]);

    const formatTimeRemaining = () => {
        if (!currentTime || !contestEndTime) return '';

        const diff = contestEndTime.getTime() - currentTime.getTime();
        if (diff <= 0) return 'Ended';

        const hours = Math.floor(diff / MS_PER_HOUR);
        const minutes = Math.floor((diff % MS_PER_HOUR) / MS_PER_MINUTE);
        const seconds = Math.floor((diff % MS_PER_MINUTE) / MS_PER_SECOND);

        return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    };

    const handleLanguageChange = (locale: string) => {
        // Use next-intl's built-in locale switching which handles all edge cases
        router.push(pathname, { locale });
    };

    const handleLoginRedirect = () => {
        if (pathname && !pathname.includes('/login')) {
            sessionStorage.setItem('loginRedirectUrl', pathname);
        }
        router.push('/login');
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
                            <nav className="hidden md:flex items-center gap-1" aria-label="Main navigation">
                                <span className="text-muted-foreground/50 mx-1">|</span>
                                {NAV_LINKS.map((link) => (
                                    <Link
                                        key={link.href}
                                        href={link.href}
                                        className={cn(
                                            "px-3 py-2 text-sm font-medium transition-colors rounded",
                                            pathname.startsWith(link.href)
                                                ? "text-primary bg-muted/50"
                                                : "text-muted-foreground hover:text-foreground hover:bg-accent/10"
                                        )}
                                    >
                                        {t(link.key)}
                                    </Link>
                                ))}
                                <Link
                                    href="/problems/random"
                                    className={cn(
                                        "px-3 py-2 text-sm font-medium transition-colors rounded flex items-center gap-1",
                                        pathname === '/problems/random'
                                            ? "text-primary bg-muted/50"
                                            : "text-muted-foreground hover:text-foreground hover:bg-accent/10"
                                    )}
                                    title="Random Problem"
                                >
                                    <RefreshCw size={14} />
                                </Link>
                            </nav>
                        </div>

                        {/* Right Side Actions */}
                        <div className="flex items-center gap-3">
                            {/* Report Issue Button - Original CLAOJ Feature */}
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

                            {/* Language Flags - Original CLAOJ Style */}
                            <div className="hidden md:flex items-center gap-1 border-l border-border pl-3">
                                <button
                                    onClick={() => handleLanguageChange('en')}
                                    className={cn(
                                        "flex items-center gap-1 px-2 py-1 rounded text-xs font-bold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
                                        locale === 'en'
                                            ? "bg-[#009688] text-white"
                                            : "text-muted-foreground hover:text-foreground hover:bg-accent/10"
                                    )}
                                    aria-label="Switch to English"
                                    aria-pressed={locale === 'en'}
                                >
                                    <img src="/static/icons/gb_flag.svg" alt="" className="w-4 h-3 object-cover" onError={(e) => (e.target as HTMLImageElement).src = GB_FLAG_SVG} />
                                    EN
                                </button>
                                <button
                                    onClick={() => handleLanguageChange('vi')}
                                    className={cn(
                                        "flex items-center gap-1 px-2 py-1 rounded text-xs font-bold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
                                        locale === 'vi'
                                            ? "bg-[#009688] text-white"
                                            : "text-muted-foreground hover:text-foreground hover:bg-accent/10"
                                    )}
                                    aria-label="Switch to Vietnamese"
                                    aria-pressed={locale === 'vi'}
                                >
                                    <img src="/static/icons/vi_flag.svg" alt="" className="w-4 h-3 object-cover" onError={(e) => (e.target as HTMLImageElement).src = VI_FLAG_SVG} />
                                    VI
                                </button>
                            </div>

                            {/* Theme Toggle */}
                            <button
                                onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
                                className="p-2 rounded-full hover:bg-accent/10 transition-colors text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary hidden md:block"
                                aria-label={mounted && theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
                                title="Toggle Theme"
                            >
                                {mounted && (theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />)}
                            </button>

                            {/* WebSocket Status */}
                            <WebSocketStatusIndicator />

                            {/* Notification Bell */}
                            {user && <NotificationBell />}

                            {/* User Menu */}
                            {user ? (
                                <div className="relative flex items-center gap-2">
                                    {/* Admin Badge */}
                                    {user.is_staff && (
                                        <AdminNavbarBadge onClick={() => setAdminSidebarOpen(true)} />
                                    )}

                                    <button
                                        onClick={() => setUserMenuOpen(!userMenuOpen)}
                                        className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-primary/10 text-primary hover:bg-primary/20 transition-all text-sm font-bold"
                                        aria-expanded={userMenuOpen}
                                        aria-haspopup="true"
                                    >
                                        <User size={16} />
                                        <span className="hidden md:inline">{user.username}</span>
                                        <ChevronDown size={14} className={cn("transition-transform", userMenuOpen && "rotate-180")} />
                                    </button>

                                    <AnimatePresence>
                                        {userMenuOpen && (
                                            <motion.div
                                                ref={setUserMenuContainerRef}
                                                initial={{ opacity: 0, y: 10 }}
                                                animate={{ opacity: 1, y: 0 }}
                                                exit={{ opacity: 0, y: 10 }}
                                                transition={{ duration: reduceMotion ? 0 : 0.15 }}
                                                className="absolute right-0 mt-2 w-48 bg-card border rounded-lg shadow-xl py-1 z-50"
                                                role="menu"
                                                aria-orientation="vertical"
                                            >
                                                <Link
                                                    href={`/user/${user.username}`}
                                                    className="flex items-center gap-2 px-4 py-2 text-sm hover:bg-muted transition-colors"
                                                    onClick={() => setUserMenuOpen(false)}
                                                    role="menuitem"
                                                >
                                                    <User size={16} />
                                                    <span>Profile</span>
                                                </Link>
                                                {user.is_staff && (
                                                    <>
                                                        <Link
                                                            href="/admin"
                                                            className="flex items-center gap-2 px-4 py-2 text-sm bg-gradient-to-r from-amber-500/10 to-orange-500/10 hover:from-amber-500/20 hover:to-orange-500/20 border-l-2 border-amber-500 transition-colors"
                                                            onClick={() => setUserMenuOpen(false)}
                                                            role="menuitem"
                                                        >
                                                            <Crown size={16} className="text-amber-500" />
                                                            <span className="font-semibold text-amber-500">Admin Dashboard</span>
                                                        </Link>
                                                        <div className="px-4 py-1.5 border-t border-border/50">
                                                            <p className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold mb-1.5">Quick Access</p>
                                                            <div className="flex gap-1.5">
                                                                <Link
                                                                    href="/admin/problems/create"
                                                                    onClick={() => setUserMenuOpen(false)}
                                                                    className="flex-1 px-2 py-1 text-[10px] bg-muted hover:bg-muted/80 rounded text-center transition-colors"
                                                                >
                                                                    + Problem
                                                                </Link>
                                                                <Link
                                                                    href="/admin/contests/create"
                                                                    onClick={() => setUserMenuOpen(false)}
                                                                    className="flex-1 px-2 py-1 text-[10px] bg-muted hover:bg-muted/80 rounded text-center transition-colors"
                                                                >
                                                                    + Contest
                                                                </Link>
                                                            </div>
                                                        </div>
                                                    </>
                                                )}
                                                <Link
                                                    href="/settings"
                                                    className="flex items-center gap-2 px-4 py-2 text-sm hover:bg-muted transition-colors"
                                                    onClick={() => setUserMenuOpen(false)}
                                                    role="menuitem"
                                                >
                                                    <SettingsIcon size={16} />
                                                    <span>Edit profile</span>
                                                </Link>
                                                <hr className="my-1 border-border" />
                                                <button
                                                    onClick={() => { logout(); setUserMenuOpen(false); }}
                                                    className="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-400 hover:bg-red-500/10 transition-colors"
                                                    role="menuitem"
                                                >
                                                    <LogOut size={16} />
                                                    <span>Log out</span>
                                                </button>
                                            </motion.div>
                                        )}
                                    </AnimatePresence>
                                </div>
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
                            <button
                                className="md:hidden p-2 text-muted-foreground hover:text-foreground"
                                onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
                                aria-expanded={mobileMenuOpen}
                                aria-controls="mobile-menu"
                                aria-label={mobileMenuOpen ? 'Close menu' : 'Open menu'}
                                title={mobileMenuOpen ? 'Close menu' : 'Open menu'}
                            >
                                {mobileMenuOpen ? <X size={24} /> : <Menu size={24} />}
                            </button>
                        </div>
                    </div>
                </div>

                {/* Contest Timer Banner - Original CLAOJ Feature */}
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

            {/* Mobile Menu */}
            <AnimatePresence>
                {mobileMenuOpen && (
                    <motion.div
                        ref={mobileMenuRef}
                        id="mobile-menu"
                        role="navigation"
                        aria-label="Mobile navigation menu"
                        initial={{ opacity: 0, height: 0 }}
                        animate={{ opacity: 1, height: 'auto' }}
                        exit={{ opacity: 0, height: 0 }}
                        transition={{ duration: reduceMotion ? 0 : 0.2 }}
                        className="md:hidden border-t bg-card px-4 py-6 flex flex-col gap-6"
                    >
                        {/* Nav Links */}
                        {NAV_LINKS.map((link) => (
                            <Link
                                key={link.href}
                                href={link.href}
                                onClick={() => setMobileMenuOpen(false)}
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
                            onClick={() => setMobileMenuOpen(false)}
                            className={cn(
                                "text-xl font-bold tracking-tight flex items-center gap-2",
                                pathname === '/problems/random' ? "text-primary" : "text-muted-foreground"
                            )}
                        >
                            <RefreshCw size={20} />
                            <span>Random Problem</span>
                        </Link>

                        {/* Report Issue */}
                        {user && (
                            <Link
                                href="/tickets/new"
                                onClick={() => setMobileMenuOpen(false)}
                                className="flex items-center gap-2 text-yellow-500 font-bold"
                            >
                                <Ticket size={18} />
                                <span>Report issue</span>
                            </Link>
                        )}

                        <div className="flex flex-col gap-4 pt-6 border-t border-border">
                            {/* Language Switcher */}
                            <div className="flex items-center justify-between">
                                <span className="text-sm font-bold text-foreground">Language</span>
                                <div className="flex items-center gap-2 p-1 rounded bg-muted">
                                    <button
                                        onClick={() => handleLanguageChange('en')}
                                        className={cn("px-3 py-2 rounded text-sm font-bold", locale === 'en' ? "text-[#009688] bg-white/20" : "text-muted-foreground hover:text-foreground hover:bg-accent/10")}
                                        aria-label="Switch to English"
                                        aria-pressed={locale === 'en'}
                                    >
                                        EN
                                    </button>
                                    <span className="text-muted-foreground/50" aria-hidden="true">|</span>
                                    <button
                                        onClick={() => handleLanguageChange('vi')}
                                        className={cn("px-3 py-2 rounded text-sm font-bold", locale === 'vi' ? "text-[#009688] bg-white/20" : "text-muted-foreground hover:text-foreground hover:bg-accent/10")}
                                        aria-label="Switch to Vietnamese"
                                        aria-pressed={locale === 'vi'}
                                    >
                                        VI
                                    </button>
                                </div>
                            </div>

                            {/* Theme Toggle */}
                            <button
                                onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
                                className="flex items-center gap-2 p-3 rounded bg-muted text-sm font-bold"
                                aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
                            >
                                {theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />}
                                <span>Theme ({theme === 'dark' ? 'Dark' : 'Light'})</span>
                            </button>

                            {!user ? (
                                <div className="grid grid-cols-2 gap-4 mt-2">
                                    <button
                                        className="flex items-center justify-center h-12 rounded border border-border font-bold text-muted-foreground hover:bg-muted transition-colors"
                                        onClick={() => {
                                            setMobileMenuOpen(false);
                                            handleLoginRedirect();
                                        }}
                                    >
                                        {t('login')}
                                    </button>
                                    <Link
                                        href="/register"
                                        className="flex items-center justify-center h-12 rounded bg-primary text-white font-bold hover:bg-primary/90 transition-all"
                                        onClick={() => setMobileMenuOpen(false)}
                                    >
                                        {t('signup')}
                                    </Link>
                                </div>
                            ) : (
                                <div className="flex flex-col gap-4 mt-2">
                                    <Link
                                        href={`/user/${user.username}`}
                                        className="flex items-center gap-3 p-4 rounded bg-primary/10 text-primary font-bold"
                                        onClick={() => setMobileMenuOpen(false)}
                                    >
                                        <User size={20} />
                                        <span>{user.username}</span>
                                    </Link>

                                    {/* Admin Section for Mobile */}
                                    {user.is_staff && (
                                        <div className="space-y-2">
                                            <Link
                                                href="/admin"
                                                className="flex items-center gap-3 p-4 rounded bg-gradient-to-r from-amber-500/20 to-orange-500/20 text-amber-400 font-bold border border-amber-500/30"
                                                onClick={() => setMobileMenuOpen(false)}
                                            >
                                                <Crown size={20} />
                                                <span>Admin Dashboard</span>
                                            </Link>
                                            <div className="grid grid-cols-2 gap-2 px-1">
                                                <Link
                                                    href="/admin/problems"
                                                    onClick={() => setMobileMenuOpen(false)}
                                                    className="flex items-center justify-center gap-2 p-3 rounded bg-muted text-xs font-bold text-muted-foreground hover:text-foreground"
                                                >
                                                    Problems
                                                </Link>
                                                <Link
                                                    href="/admin/contests"
                                                    onClick={() => setMobileMenuOpen(false)}
                                                    className="flex items-center justify-center gap-2 p-3 rounded bg-muted text-xs font-bold text-muted-foreground hover:text-foreground"
                                                >
                                                    Contests
                                                </Link>
                                                <Link
                                                    href="/admin/users"
                                                    onClick={() => setMobileMenuOpen(false)}
                                                    className="flex items-center justify-center gap-2 p-3 rounded bg-muted text-xs font-bold text-muted-foreground hover:text-foreground"
                                                >
                                                    Users
                                                </Link>
                                                <Link
                                                    href="/admin/tickets"
                                                    onClick={() => setMobileMenuOpen(false)}
                                                    className="flex items-center justify-center gap-2 p-3 rounded bg-muted text-xs font-bold text-muted-foreground hover:text-foreground"
                                                >
                                                    Tickets
                                                </Link>
                                            </div>
                                        </div>
                                    )}

                                    <Link
                                        href="/settings"
                                        className="flex items-center gap-3 p-4 rounded bg-white/10 font-bold"
                                        onClick={() => setMobileMenuOpen(false)}
                                    >
                                        <SettingsIcon size={20} />
                                        <span>Settings</span>
                                    </Link>
                                    <button
                                        onClick={() => { logout(); setMobileMenuOpen(false); }}
                                        className="flex items-center gap-3 p-4 rounded bg-red-500/10 text-red-400 font-bold text-left"
                                    >
                                        <LogOut size={20} />
                                        <span>Logout</span>
                                    </button>
                                </div>
                            )}
                        </div>
                    </motion.div>
                )}
            </AnimatePresence>

            {/* Admin Sidebar */}
            <AdminSidebar isOpen={adminSidebarOpen} onClose={() => setAdminSidebarOpen(false)} />

            {/* Admin Quick Access Button - Fixed position */}
            <AdminQuickAccessButton onClick={() => setAdminSidebarOpen(true)} />
        </>
    );
}
