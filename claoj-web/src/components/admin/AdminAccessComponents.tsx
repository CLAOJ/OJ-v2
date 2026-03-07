'use client';

import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence, useReducedMotion } from 'framer-motion';
import { useAuth } from '@/components/providers/AuthProvider';
import { Link } from '@/navigation';
import {
    Shield,
    Settings,
    Users,
    FileText,
    Trophy,
    Code,
    MessageSquare,
    BarChart3,
    Layout,
    X,
    ChevronRight,
    Crown,
    Activity,
    LogOut,
    Zap,
    Terminal,
    Globe,
    Bell
} from 'lucide-react';
import { cn } from '@/lib/utils';

// =============================================================================
// ADMIN DASHBOARD SIDEBAR - Persistent admin navigation
// =============================================================================

interface AdminSidebarProps {
    isOpen: boolean;
    onClose: () => void;
}

const ADMIN_SECTIONS = [
    {
        id: 'overview',
        label: 'Overview',
        icon: Activity,
        href: '/admin',
        color: 'from-emerald-500 to-teal-500',
    },
    {
        id: 'problems',
        label: 'Problems',
        icon: Code,
        href: '/admin/problems',
        color: 'from-blue-500 to-cyan-500',
        badge: 'Manage',
    },
    {
        id: 'contests',
        label: 'Contests',
        icon: Trophy,
        href: '/admin/contests',
        color: 'from-amber-500 to-orange-500',
    },
    {
        id: 'users',
        label: 'Users',
        icon: Users,
        href: '/admin/users',
        color: 'from-violet-500 to-purple-500',
    },
    {
        id: 'submissions',
        label: 'Submissions',
        icon: Terminal,
        href: '/admin/submissions',
        color: 'from-rose-500 to-pink-500',
    },
    {
        id: 'blog',
        label: 'Blog Posts',
        icon: FileText,
        href: '/admin/blog-posts',
        color: 'from-indigo-500 to-blue-500',
    },
    {
        id: 'tickets',
        label: 'Tickets',
        icon: MessageSquare,
        href: '/admin/tickets',
        color: 'from-cyan-500 to-sky-500',
    },
    {
        id: 'languages',
        label: 'Languages',
        icon: Globe,
        href: '/admin/languages',
        color: 'from-fuchsia-500 to-pink-500',
    },
    {
        id: 'navigation',
        label: 'Navigation',
        icon: Layout,
        href: '/admin/navigation-bars',
        color: 'from-lime-500 to-green-500',
    },
    {
        id: 'roles',
        label: 'Roles',
        icon: Shield,
        href: '/admin/roles',
        color: 'from-red-500 to-rose-500',
    },
];

export function AdminSidebar({ isOpen, onClose }: AdminSidebarProps) {
    const { user } = useAuth();
    const reduceMotion = useReducedMotion();
    const [hoveredItem, setHoveredItem] = useState<string | null>(null);

    if (!user?.is_staff) return null;

    return (
        <AnimatePresence>
            {isOpen && (
                <>
                    {/* Backdrop */}
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: reduceMotion ? 0 : 0.2 }}
                        className="fixed inset-0 bg-black/60 backdrop-blur-sm z-40"
                        onClick={onClose}
                    />

                    {/* Sidebar */}
                    <motion.aside
                        initial={{ x: '-100%', opacity: 0.8 }}
                        animate={{ x: 0, opacity: 1 }}
                        exit={{ x: '-100%', opacity: 0.8 }}
                        transition={{
                            type: 'spring',
                            damping: 30,
                            stiffness: 300,
                            mass: 0.8,
                        }}
                        className="fixed left-0 top-0 h-full w-80 z-50"
                    >
                        <div className="h-full bg-slate-950 border-r border-slate-800/50 flex flex-col overflow-hidden">
                            {/* Header */}
                            <div className="relative overflow-hidden">
                                <div className="absolute inset-0 bg-gradient-to-br from-amber-500/10 via-transparent to-transparent" />
                                <div className="relative p-6 border-b border-slate-800/50">
                                    <div className="flex items-center justify-between">
                                        <div className="flex items-center gap-3">
                                            <div className="relative">
                                                <div className="absolute inset-0 bg-amber-500/30 blur-xl rounded-full" />
                                                <div className="relative w-12 h-12 rounded-xl bg-gradient-to-br from-amber-400 to-amber-600 flex items-center justify-center shadow-lg shadow-amber-500/25">
                                                    <Crown className="w-6 h-6 text-slate-950" />
                                                </div>
                                            </div>
                                            <div>
                                                <h2 className="text-lg font-bold text-slate-100">
                                                    Admin Panel
                                                </h2>
                                                <p className="text-xs text-slate-400 font-medium">
                                                    Command Center
                                                </p>
                                            </div>
                                        </div>
                                        <button
                                            onClick={onClose}
                                            className="p-2 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition-colors"
                                        >
                                            <X className="w-5 h-5" />
                                        </button>
                                    </div>
                                </div>
                            </div>

                            {/* Admin Info */}
                            <div className="px-4 py-3 border-b border-slate-800/50">
                                <div className="flex items-center gap-3 p-3 rounded-lg bg-slate-900/50 border border-slate-800">
                                    <div className="w-10 h-10 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-bold">
                                        {user.username[0].toUpperCase()}
                                    </div>
                                    <div className="flex-1 min-w-0">
                                        <p className="text-sm font-semibold text-slate-200 truncate">
                                            {user.username}
                                        </p>
                                        <div className="flex items-center gap-1.5">
                                            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                                            <span className="text-xs text-slate-400">
                                                {user.is_admin ? 'Super Admin' : 'Staff'}
                                            </span>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            {/* Navigation */}
                            <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-1">
                                {ADMIN_SECTIONS.map((section, index) => (
                                    <motion.div
                                        key={section.id}
                                        initial={{ opacity: 0, x: -20 }}
                                        animate={{ opacity: 1, x: 0 }}
                                        transition={{ delay: index * 0.03 }}
                                    >
                                        <Link
                                            href={section.href}
                                            onClick={onClose}
                                            onMouseEnter={() => setHoveredItem(section.id)}
                                            onMouseLeave={() => setHoveredItem(null)}
                                            className={cn(
                                                'group flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all duration-200',
                                                'hover:bg-slate-800/50 relative overflow-hidden'
                                            )}
                                        >
                                            {/* Hover gradient */}
                                            <AnimatePresence>
                                                {hoveredItem === section.id && (
                                                    <motion.div
                                                        layoutId="hoverGradient"
                                                        className={cn(
                                                            'absolute inset-0 bg-gradient-to-r opacity-10',
                                                            section.color
                                                        )}
                                                        initial={{ opacity: 0 }}
                                                        animate={{ opacity: 0.1 }}
                                                        exit={{ opacity: 0 }}
                                                    />
                                                )}
                                            </AnimatePresence>

                                            {/* Icon */}
                                            <div
                                                className={cn(
                                                    'relative w-9 h-9 rounded-lg flex items-center justify-center',
                                                    'bg-gradient-to-br text-white shadow-lg',
                                                    section.color,
                                                    'group-hover:scale-110 transition-transform duration-200'
                                                )}
                                            >
                                                <section.icon className="w-4.5 h-4.5" />
                                            </div>

                                            {/* Label */}
                                            <span className="relative flex-1 text-sm font-medium text-slate-300 group-hover:text-slate-100">
                                                {section.label}
                                            </span>

                                            {/* Badge */}
                                            {section.badge && (
                                                <span className="relative text-[10px] font-bold px-2 py-0.5 rounded-full bg-slate-800 text-slate-400">
                                                    {section.badge}
                                                </span>
                                            )}

                                            {/* Arrow */}
                                            <ChevronRight className="relative w-4 h-4 text-slate-600 group-hover:text-slate-400 group-hover:translate-x-0.5 transition-all" />
                                        </Link>
                                    </motion.div>
                                ))}
                            </nav>

                            {/* Footer */}
                            <div className="p-4 border-t border-slate-800/50">
                                <div className="flex items-center gap-2 text-xs text-slate-500">
                                    <Shield className="w-3.5 h-3.5" />
                                    <span>Secure Admin Session</span>
                                </div>
                            </div>
                        </div>
                    </motion.aside>
                </>
            )}
        </AnimatePresence>
    );
}

// =============================================================================
// ADMIN QUICK ACCESS BUTTON - Floating trigger for sidebar
// =============================================================================

interface AdminQuickAccessButtonProps {
    onClick: () => void;
}

export function AdminQuickAccessButton({ onClick }: AdminQuickAccessButtonProps) {
    const { user } = useAuth();
    const [isHovered, setIsHovered] = useState(false);
    const [showPulse, setShowPulse] = useState(true);

    if (!user?.is_staff) return null;

    return (
        <motion.button
            initial={{ scale: 0, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ type: 'spring', damping: 20, stiffness: 300, delay: 0.5 }}
            onClick={onClick}
            onMouseEnter={() => {
                setIsHovered(true);
                setShowPulse(false);
            }}
            onMouseLeave={() => setIsHovered(false)}
            className={cn(
                'fixed left-4 top-24 z-30 group',
                'flex items-center gap-2 pl-3 pr-4 py-2.5 rounded-full',
                'bg-slate-950/90 backdrop-blur-md border border-amber-500/30',
                'shadow-lg shadow-amber-500/10 hover:shadow-amber-500/20',
                'hover:border-amber-500/50 transition-all duration-300'
            )}
        >
            {/* Pulse effect */}
            {showPulse && (
                <span className="absolute inset-0 rounded-full bg-amber-500/20 animate-ping" />
            )}

            {/* Icon container */}
            <div className="relative">
                <motion.div
                    animate={{ rotate: isHovered ? 180 : 0 }}
                    transition={{ duration: 0.3 }}
                    className="w-8 h-8 rounded-full bg-gradient-to-br from-amber-400 to-amber-600 flex items-center justify-center"
                >
                    <Settings className="w-4 h-4 text-slate-950" />
                </motion.div>
                <span className="absolute -top-0.5 -right-0.5 w-2.5 h-2.5 rounded-full bg-emerald-500 border-2 border-slate-950" />
            </div>

            {/* Label */}
            <motion.span
                animate={{ x: isHovered ? 2 : 0 }}
                className="text-sm font-semibold text-amber-400"
            >
                Admin
            </motion.span>
        </motion.button>
    );
}

// =============================================================================
// ADMIN WELCOME BANNER - Shown on homepage after admin login
// =============================================================================

interface AdminWelcomeBannerProps {
    onDismiss?: () => void;
}

export function AdminWelcomeBanner({ onDismiss }: AdminWelcomeBannerProps) {
    const { user } = useAuth();
    const [isVisible, setIsVisible] = useState(true);
    const [currentTime, setCurrentTime] = useState(new Date());

    useEffect(() => {
        const timer = setInterval(() => setCurrentTime(new Date()), 1000);
        return () => clearInterval(timer);
    }, []);

    if (!user?.is_staff || !isVisible) return null;

    const handleDismiss = () => {
        setIsVisible(false);
        onDismiss?.();
    };

    const formatTime = (date: Date) => {
        return date.toLocaleTimeString('en-US', {
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            hour12: false,
        });
    };

    const quickActions = [
        { icon: Code, label: 'New Problem', href: '/admin/problems/create', color: 'text-blue-400' },
        { icon: Trophy, label: 'New Contest', href: '/admin/contests/create', color: 'text-amber-400' },
        { icon: Users, label: 'Manage Users', href: '/admin/users', color: 'text-violet-400' },
        { icon: BarChart3, label: 'Analytics', href: '/admin', color: 'text-emerald-400' },
    ];

    return (
        <motion.div
            initial={{ opacity: 0, y: -20, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -20, scale: 0.95 }}
            transition={{ type: 'spring', damping: 25, stiffness: 300 }}
            className="relative mb-6 overflow-hidden rounded-2xl"
        >
            {/* Background effects */}
            <div className="absolute inset-0 bg-gradient-to-br from-slate-900 via-slate-950 to-slate-900" />
            <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-amber-500/10 via-transparent to-transparent" />
            <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_bottom_left,_var(--tw-gradient-stops))] from-indigo-500/10 via-transparent to-transparent" />

            {/* Grid pattern */}
            <div
                className="absolute inset-0 opacity-[0.03]"
                style={{
                    backgroundImage: `linear-gradient(rgba(255,255,255,0.1) 1px, transparent 1px),
                                      linear-gradient(90deg, rgba(255,255,255,0.1) 1px, transparent 1px)`,
                    backgroundSize: '40px 40px',
                }}
            />

            {/* Animated border */}
            <div className="absolute inset-0 rounded-2xl border border-amber-500/20" />
            <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-amber-500/50 to-transparent" />

            <div className="relative p-6">
                {/* Header */}
                <div className="flex items-start justify-between mb-6">
                    <div className="flex items-center gap-4">
                        <div className="relative">
                            <div className="absolute inset-0 bg-amber-500/30 blur-2xl rounded-full animate-pulse" />
                            <div className="relative w-16 h-16 rounded-2xl bg-gradient-to-br from-amber-400 via-amber-500 to-amber-600 flex items-center justify-center shadow-xl shadow-amber-500/25">
                                <Crown className="w-8 h-8 text-slate-950" />
                            </div>
                        </div>
                        <div>
                            <div className="flex items-center gap-2 mb-1">
                                <h2 className="text-2xl font-bold text-slate-100">
                                    Welcome back, {user.username}
                                </h2>
                                <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-amber-500/20 text-amber-400 border border-amber-500/30 uppercase tracking-wider">
                                    {user.is_admin ? 'Super Admin' : 'Admin'}
                                </span>
                            </div>
                            <p className="text-slate-400">
                                You have elevated privileges. Manage the platform wisely.
                            </p>
                        </div>
                    </div>

                    <div className="flex items-center gap-4">
                        {/* Live clock */}
                        <div className="hidden sm:flex items-center gap-2 px-4 py-2 rounded-lg bg-slate-900/50 border border-slate-800">
                            <Zap className="w-4 h-4 text-amber-400" />
                            <span className="text-lg font-mono font-bold text-slate-300 tracking-wider">
                                {formatTime(currentTime)}
                            </span>
                        </div>

                        {/* Dismiss button */}
                        <button
                            onClick={handleDismiss}
                            className="p-2 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition-colors"
                        >
                            <X className="w-5 h-5" />
                        </button>
                    </div>
                </div>

                {/* Quick Actions */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                    {quickActions.map((action, index) => (
                        <motion.div
                            key={action.label}
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.1 + index * 0.05 }}
                        >
                            <Link
                                href={action.href}
                                className={cn(
                                    'group flex items-center gap-3 p-4 rounded-xl',
                                    'bg-slate-900/50 border border-slate-800',
                                    'hover:border-slate-700 hover:bg-slate-800/50',
                                    'transition-all duration-200'
                                )}
                            >
                                <action.icon className={cn('w-5 h-5', action.color)} />
                                <span className="text-sm font-medium text-slate-300 group-hover:text-slate-200">
                                    {action.label}
                                </span>
                                <ChevronRight className="w-4 h-4 text-slate-600 ml-auto group-hover:text-slate-400 group-hover:translate-x-0.5 transition-all" />
                            </Link>
                        </motion.div>
                    ))}
                </div>

                {/* Stats row */}
                <div className="mt-6 pt-6 border-t border-slate-800/50 grid grid-cols-3 gap-4">
                    {[
                        { label: 'System Status', value: 'Operational', color: 'text-emerald-400' },
                        { label: 'Active Users', value: 'View Analytics', color: 'text-blue-400', href: '/admin/users' },
                        { label: 'Pending Tickets', value: 'Check Now', color: 'text-amber-400', href: '/admin/tickets' },
                    ].map((stat) => (
                        <div key={stat.label} className="text-center">
                            <p className="text-xs text-slate-500 uppercase tracking-wider mb-1">
                                {stat.label}
                            </p>
                            {stat.href ? (
                                <Link href={stat.href} className={cn('text-sm font-semibold hover:underline', stat.color)}>
                                    {stat.value}
                                </Link>
                            ) : (
                                <p className={cn('text-sm font-semibold', stat.color)}>
                                    {stat.value}
                                </p>
                            )}
                        </div>
                    ))}
                </div>
            </div>
        </motion.div>
    );
}

// =============================================================================
// ADMIN NAVBAR BADGE - Enhanced admin indicator in navbar
// =============================================================================

interface AdminNavbarBadgeProps {
    onClick?: () => void;
}

export function AdminNavbarBadge({ onClick }: AdminNavbarBadgeProps) {
    const { user } = useAuth();

    if (!user?.is_staff) return null;

    return (
        <motion.button
            onClick={onClick}
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            className={cn(
                'flex items-center gap-2 px-3 py-1.5 rounded-full',
                'bg-gradient-to-r from-amber-500/20 to-orange-500/20',
                'border border-amber-500/30',
                'text-amber-400 text-xs font-bold uppercase tracking-wider',
                'hover:from-amber-500/30 hover:to-orange-500/30 hover:border-amber-500/50',
                'transition-all duration-200'
            )}
        >
            <Shield className="w-3.5 h-3.5" />
            <span>Admin</span>
        </motion.button>
    );
}

// =============================================================================
// ADMIN ACCESS WRAPPER - Combines all components
// =============================================================================

interface AdminAccessWrapperProps {
    children: React.ReactNode;
    showWelcomeBanner?: boolean;
}

export function AdminAccessWrapper({ children, showWelcomeBanner = false }: AdminAccessWrapperProps) {
    const { user } = useAuth();
    const [sidebarOpen, setSidebarOpen] = useState(false);
    const [bannerDismissed, setBannerDismissed] = useState(false);

    const isAdmin = user?.is_staff;

    return (
        <>
            {isAdmin && (
                <>
                    <AdminSidebar isOpen={sidebarOpen} onClose={() => setSidebarOpen(false)} />
                    <AdminQuickAccessButton onClick={() => setSidebarOpen(true)} />
                </>
            )}

            <div className="relative">
                {isAdmin && showWelcomeBanner && !bannerDismissed && (
                    <AdminWelcomeBanner onDismiss={() => setBannerDismissed(true)} />
                )}
                {children}
            </div>
        </>
    );
}

export default AdminAccessWrapper;
