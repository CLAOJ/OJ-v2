'use client';

import { Link, usePathname, useRouter, routing } from '@/navigation';
import { useTranslations } from 'next-intl';
import { useAuth } from '@/components/providers/AuthProvider';
import { useTheme } from 'next-themes';
import { Moon, Sun, User, LogOut, Menu, X, Settings as SettingsIcon, Flag, Ticket, ChevronDown } from 'lucide-react';
import NotificationBell from '@/components/notifications/NotificationBell';
import WebSocketStatusIndicator from '@/components/common/WebSocketStatus';
import { useState, useEffect } from 'react';
import { cn } from '@/lib/utils';
import { motion, AnimatePresence } from 'framer-motion';

const NAV_LINKS = [
    { href: '/problems', key: 'problems' },
    { href: '/contests', key: 'contests' },
    { href: '/submissions', key: 'submissions' },
    { href: '/users', key: 'users' },
    { href: '/ratings', key: 'ratings' },
    { href: '/organizations', key: 'organizations' },
];

export default function Navbar() {
    const { user, logout } = useAuth();
    const { theme, setTheme } = useTheme();
    const t = useTranslations('Navbar');
    const pathname = usePathname();
    const router = useRouter();
    const [mounted, setMounted] = useState(false);
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
    const [userMenuOpen, setUserMenuOpen] = useState(false);
    const [currentTime, setCurrentTime] = useState<Date | null>(null);
    const [contestEndTime, setContestEndTime] = useState<Date | null>(null);

    // Simulate contest participation (in real app, this would come from auth context)
    const [inContest, setInContest] = useState(false);

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

        const hours = Math.floor(diff / (1000 * 60 * 60));
        const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
        const seconds = Math.floor((diff % (1000 * 60)) / 1000);

        return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    };

    const handleLanguageChange = (lang: string) => {
        const currentLocale = pathname.split('/')[1];
        // With localePrefix: 'as-needed', default locale (en) has no prefix
        const effectiveCurrent = currentLocale === 'vi' ? 'vi' : 'en';

        if (lang !== effectiveCurrent) {
            // For default locale (en), just use the path without prefix
            // For other locales (vi), add the prefix
            if (lang === routing.defaultLocale) {
                // Switching to default locale - remove current locale prefix if any
                const newPath = effectiveCurrent === 'en' ? pathname : pathname.replace(/^\/vi/, '');
                router.push(newPath || '/');
            } else {
                // Switching to non-default locale - add prefix
                const newPath = effectiveCurrent === 'en' ? `/${lang}${pathname}` : pathname.replace(/^\/vi/, `/${lang}`);
                router.push(newPath);
            }
        }
    };

    return (
        <>
            <header className="sticky top-0 z-50 w-full border-b bg-[#263238]/95 backdrop-blur-md shadow-lg">
                <div className="container mx-auto px-4">
                    <div className="flex h-16 items-center justify-between">
                        {/* Logo and Nav Links */}
                        <div className="flex items-center gap-6">
                            {/* Logo */}
                            <Link href="/" className="flex items-center gap-2 hover:opacity-80 transition-opacity">
                                <span className="text-2xl font-bold tracking-tighter text-[#009688] italic">
                                    CLAOJ
                                </span>
                            </Link>

                            {/* Desktop Nav */}
                            <nav className="hidden md:flex items-center gap-1">
                                <span className="text-gray-500 mx-1">|</span>
                                {NAV_LINKS.map((link) => (
                                    <Link
                                        key={link.href}
                                        href={link.href}
                                        className={cn(
                                            "px-3 py-2 text-sm font-medium transition-colors rounded",
                                            pathname.startsWith(link.href)
                                                ? "text-[#009688] bg-white/5"
                                                : "text-gray-300 hover:text-white hover:bg-white/10"
                                        )}
                                    >
                                        {t(link.key)}
                                    </Link>
                                ))}
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
                            <div className="hidden md:flex items-center gap-1 border-l border-gray-600 pl-3">
                                <button
                                    onClick={() => handleLanguageChange('en')}
                                    className={cn(
                                        "flex items-center gap-1 px-2 py-1 rounded text-xs font-bold transition-colors",
                                        pathname.includes('/en')
                                            ? "bg-[#009688] text-white"
                                            : "text-gray-400 hover:text-white hover:bg-white/10"
                                    )}
                                >
                                    <img src="/static/icons/gb_flag.svg" alt="EN" className="w-4 h-3 object-cover" onError={(e) => (e.target as HTMLImageElement).src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 16" fill="blue"><rect width="24" height="16" fill="white"/><path d="M0 0h8v16H0z" fill="blue"/><path d="M16 0h8v16h-8z" fill="red"/></svg>'} />
                                    EN
                                </button>
                                <button
                                    onClick={() => handleLanguageChange('vi')}
                                    className={cn(
                                        "flex items-center gap-1 px-2 py-1 rounded text-xs font-bold transition-colors",
                                        pathname.includes('/vi')
                                            ? "bg-[#009688] text-white"
                                            : "text-gray-400 hover:text-white hover:bg-white/10"
                                    )}
                                >
                                    <img src="/static/icons/vi_flag.svg" alt="VI" className="w-4 h-3 object-cover" onError={(e) => (e.target as HTMLImageElement).src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 16" fill="red"><rect width="24" height="16" fill="red"/><text x="12" y="12" text-anchor="middle" font-size="10" fill="yellow">★</text></svg>'} />
                                    VI
                                </button>
                            </div>

                            {/* Theme Toggle */}
                            <button
                                onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
                                className="p-2 rounded-full hover:bg-white/10 transition-colors text-gray-400 hover:text-white hidden md:block"
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
                                <div className="relative">
                                    <button
                                        onClick={() => setUserMenuOpen(!userMenuOpen)}
                                        className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-[#009688]/10 text-[#009688] hover:bg-[#009688]/20 transition-all text-sm font-bold"
                                    >
                                        <User size={16} />
                                        <span className="hidden md:inline">{user.username}</span>
                                        <ChevronDown size={14} className={cn("transition-transform", userMenuOpen && "rotate-180")} />
                                    </button>

                                    <AnimatePresence>
                                        {userMenuOpen && (
                                            <motion.div
                                                initial={{ opacity: 0, y: 10 }}
                                                animate={{ opacity: 1, y: 0 }}
                                                exit={{ opacity: 0, y: 10 }}
                                                className="absolute right-0 mt-2 w-48 bg-card border rounded-lg shadow-xl py-1 z-50"
                                            >
                                                <Link
                                                    href={`/user/${user.username}`}
                                                    className="flex items-center gap-2 px-4 py-2 text-sm hover:bg-white/5 transition-colors"
                                                    onClick={() => setUserMenuOpen(false)}
                                                >
                                                    <User size={16} />
                                                    <span>Profile</span>
                                                </Link>
                                                {user.is_staff && (
                                                    <Link
                                                        href="/admin"
                                                        className="flex items-center gap-2 px-4 py-2 text-sm hover:bg-white/5 transition-colors"
                                                        onClick={() => setUserMenuOpen(false)}
                                                    >
                                                        <SettingsIcon size={16} />
                                                        <span>Admin</span>
                                                    </Link>
                                                )}
                                                <Link
                                                    href="/settings"
                                                    className="flex items-center gap-2 px-4 py-2 text-sm hover:bg-white/5 transition-colors"
                                                    onClick={() => setUserMenuOpen(false)}
                                                >
                                                    <SettingsIcon size={16} />
                                                    <span>Edit profile</span>
                                                </Link>
                                                <hr className="my-1 border-gray-700" />
                                                <button
                                                    onClick={() => { logout(); setUserMenuOpen(false); }}
                                                    className="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-400 hover:bg-red-500/10 transition-colors"
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
                                    <Link
                                        href="/login"
                                        className="text-sm font-medium text-gray-300 hover:text-white px-3 py-1.5 rounded hover:bg-white/10 transition-colors"
                                    >
                                        {t('login')}
                                    </Link>
                                    <Link
                                        href="/register"
                                        className="px-4 py-1.5 rounded bg-[#009688] text-white text-sm font-bold hover:bg-[#009688]/90 transition-all"
                                    >
                                        {t('signup')}
                                    </Link>
                                </div>
                            )}

                            {/* Mobile Menu Button */}
                            <button
                                className="md:hidden p-2 text-gray-400 hover:text-white"
                                onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
                            >
                                {mobileMenuOpen ? <X size={24} /> : <Menu size={24} />}
                            </button>
                        </div>
                    </div>
                </div>

                {/* Contest Timer Banner - Original CLAOJ Feature */}
                {inContest && currentTime && (
                    <div className="bg-[#263238] border-t border-b border-[#3b4d56] px-4 py-2 text-center">
                        <Link href="/contest/current" className="text-sm font-medium text-white">
                            <span className="text-[#009688]">Current Contest:</span>{' '}
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
                        initial={{ opacity: 0, height: 0 }}
                        animate={{ opacity: 1, height: 'auto' }}
                        exit={{ opacity: 0, height: 0 }}
                        className="md:hidden border-t bg-[#263238] px-4 py-6 flex flex-col gap-6"
                    >
                        {/* Nav Links */}
                        {NAV_LINKS.map((link) => (
                            <Link
                                key={link.href}
                                href={link.href}
                                onClick={() => setMobileMenuOpen(false)}
                                className={cn(
                                    "text-xl font-bold tracking-tight",
                                    pathname.startsWith(link.href) ? "text-[#009688]" : "text-gray-300"
                                )}
                            >
                                {t(link.key)}
                            </Link>
                        ))}

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

                        <div className="flex flex-col gap-4 pt-6 border-t border-gray-600">
                            {/* Language Switcher */}
                            <div className="flex items-center justify-between">
                                <span className="text-sm font-bold text-gray-400">Language</span>
                                <div className="flex items-center gap-2 p-1 rounded bg-white/10 px-3">
                                    <button
                                        onClick={() => handleLanguageChange('en')}
                                        className={cn("text-xs font-black", pathname.includes('/en') ? "text-[#009688]" : "text-gray-400")}
                                    >
                                        EN
                                    </button>
                                    <span className="text-gray-500">|</span>
                                    <button
                                        onClick={() => handleLanguageChange('vi')}
                                        className={cn("text-xs font-black", pathname.includes('/vi') ? "text-[#009688]" : "text-gray-400")}
                                    >
                                        VI
                                    </button>
                                </div>
                            </div>

                            {/* Theme Toggle */}
                            <button
                                onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
                                className="flex items-center gap-2 p-3 rounded bg-white/10 text-sm font-bold"
                            >
                                {theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />}
                                <span>Theme ({theme === 'dark' ? 'Dark' : 'Light'})</span>
                            </button>

                            {!user ? (
                                <div className="grid grid-cols-2 gap-4 mt-2">
                                    <Link
                                        href="/login"
                                        className="flex items-center justify-center h-12 rounded border font-bold text-gray-300"
                                        onClick={() => setMobileMenuOpen(false)}
                                    >
                                        {t('login')}
                                    </Link>
                                    <Link
                                        href="/register"
                                        className="flex items-center justify-center h-12 rounded bg-[#009688] text-white font-bold"
                                        onClick={() => setMobileMenuOpen(false)}
                                    >
                                        {t('signup')}
                                    </Link>
                                </div>
                            ) : (
                                <div className="flex flex-col gap-4 mt-2">
                                    <Link
                                        href={`/user/${user.username}`}
                                        className="flex items-center gap-3 p-4 rounded bg-[#009688]/10 text-[#009688] font-bold"
                                        onClick={() => setMobileMenuOpen(false)}
                                    >
                                        <User size={20} />
                                        <span>{user.username}</span>
                                    </Link>
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
        </>
    );
}
