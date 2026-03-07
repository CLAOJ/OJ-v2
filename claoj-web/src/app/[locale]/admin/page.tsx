'use client';

import { useAuth } from '@/components/providers/AuthProvider';
import { useTranslations } from 'next-intl';
import { useRouter } from '@/navigation';
import { useEffect } from 'react';
import { motion } from 'framer-motion';
import {
    LayoutDashboard,
    Users,
    Code,
    Trophy,
    FileText,
    MessageSquare,
    Settings,
    Activity,
    Globe,
    Shield,
    BarChart3,
    Zap,
    TrendingUp,
    ArrowRight
} from 'lucide-react';
import { Link } from '@/navigation';
import { cn } from '@/lib/utils';

const ADMIN_CARDS = [
    {
        id: 'problems',
        title: 'Problems',
        description: 'Manage problem library, create new problems, and organize problem groups.',
        icon: Code,
        href: '/admin/problems',
        color: 'from-blue-500 to-cyan-500',
        bgColor: 'bg-blue-500/10',
        borderColor: 'border-blue-500/20',
        stats: 'View all problems',
    },
    {
        id: 'contests',
        title: 'Contests',
        description: 'Create and manage contests, set up ratings, and monitor participation.',
        icon: Trophy,
        href: '/admin/contests',
        color: 'from-amber-500 to-orange-500',
        bgColor: 'bg-amber-500/10',
        borderColor: 'border-amber-500/20',
        stats: 'Manage contests',
    },
    {
        id: 'users',
        title: 'Users',
        description: 'Manage user accounts, ban/unban users, and assign roles.',
        icon: Users,
        href: '/admin/users',
        color: 'from-violet-500 to-purple-500',
        bgColor: 'bg-violet-500/10',
        borderColor: 'border-violet-500/20',
        stats: 'View user list',
    },
    {
        id: 'submissions',
        title: 'Submissions',
        description: 'Monitor submission queue, rejudge submissions, and view statistics.',
        icon: Activity,
        href: '/admin/submissions',
        color: 'from-rose-500 to-pink-500',
        bgColor: 'bg-rose-500/10',
        borderColor: 'border-rose-500/20',
        stats: 'View submissions',
    },
    {
        id: 'blog',
        title: 'Blog Posts',
        description: 'Create, edit, and manage blog posts and announcements.',
        icon: FileText,
        href: '/admin/blog-posts',
        color: 'from-indigo-500 to-blue-500',
        bgColor: 'bg-indigo-500/10',
        borderColor: 'border-indigo-500/20',
        stats: 'Manage posts',
    },
    {
        id: 'tickets',
        title: 'Tickets',
        description: 'Handle user support tickets, bug reports, and feature requests.',
        icon: MessageSquare,
        href: '/admin/tickets',
        color: 'from-cyan-500 to-sky-500',
        bgColor: 'bg-cyan-500/10',
        borderColor: 'border-cyan-500/20',
        stats: 'View tickets',
    },
    {
        id: 'languages',
        title: 'Languages',
        description: 'Configure supported programming languages and their settings.',
        icon: Globe,
        href: '/admin/languages',
        color: 'from-fuchsia-500 to-pink-500',
        bgColor: 'bg-fuchsia-500/10',
        borderColor: 'border-fuchsia-500/20',
        stats: 'Manage languages',
    },
    {
        id: 'navigation',
        title: 'Navigation',
        description: 'Customize navigation bars and menu structure.',
        icon: LayoutDashboard,
        href: '/admin/navigation-bars',
        color: 'from-lime-500 to-green-500',
        bgColor: 'bg-lime-500/10',
        borderColor: 'border-lime-500/20',
        stats: 'Edit navigation',
    },
    {
        id: 'roles',
        title: 'Roles',
        description: 'Manage user roles, permissions, and access control.',
        icon: Shield,
        href: '/admin/roles',
        color: 'from-red-500 to-rose-500',
        bgColor: 'bg-red-500/10',
        borderColor: 'border-red-500/20',
        stats: 'Manage roles',
    },
];

const QUICK_ACTIONS = [
    { label: 'Create Problem', href: '/admin/problems/create', icon: Code, color: 'text-blue-400' },
    { label: 'Create Contest', href: '/admin/contests/create', icon: Trophy, color: 'text-amber-400' },
    { label: 'View Users', href: '/admin/users', icon: Users, color: 'text-violet-400' },
    { label: 'System Stats', href: '/stats', icon: BarChart3, color: 'text-emerald-400' },
];

export default function AdminDashboardPage() {
    const { user, loading } = useAuth();
    const router = useRouter();

    useEffect(() => {
        if (!loading && !user?.is_staff) {
            router.push('/');
        }
    }, [user, loading, router]);

    if (loading || !user?.is_staff) {
        return (
            <div className="flex items-center justify-center min-h-[60vh]"
            >
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary" />
            </div>
        );
    }

    return (
        <div className="space-y-8"
        >
            {/* Header */}
            <motion.div
                initial={{ opacity: 0, y: -20 }}
                animate={{ opacity: 1, y: 0 }}
                className="relative overflow-hidden rounded-2xl bg-gradient-to-br from-slate-900 via-slate-950 to-slate-900 border border-slate-800"
            >
                {/* Background effects */}
                <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-amber-500/10 via-transparent to-transparent" />
                <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_bottom_left,_var(--tw-gradient-stops))] from-indigo-500/10 via-transparent to-transparent" />

                <div className="relative p-8"
                >
                    <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4"
                    >
                        <div>
                            <div className="flex items-center gap-3 mb-2"
                            >
                                <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-amber-400 to-amber-600 flex items-center justify-center shadow-lg shadow-amber-500/25"
                                >
                                    <Settings className="w-6 h-6 text-slate-950" />
                                </div>
                                <div>
                                    <h1 className="text-3xl font-bold text-slate-100"
                                    >
                                        Admin Dashboard
                                    </h1>
                                    <p className="text-slate-400">
                                        Welcome back, {user.username}
                                    </p>
                                </div>
                            </div>
                        </div>
                        <div className="flex items-center gap-2"
                        >
                            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                            <span className="text-sm text-slate-400"
                            >
                                System Operational
                            </span>
                        </div>
                    </div>

                    {/* Quick Actions */}
                    <div className="mt-6 pt-6 border-t border-slate-800/50 grid grid-cols-2 md:grid-cols-4 gap-3"
                    >
                        {QUICK_ACTIONS.map((action, index) => (
                            <motion.div
                                key={action.label}
                                initial={{ opacity: 0, y: 10 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: 0.1 + index * 0.05 }}
                            >
                                <Link
                                    href={action.href}
                                    className={cn(
                                        'flex items-center gap-2 p-3 rounded-lg',
                                        'bg-slate-900/50 border border-slate-800',
                                        'hover:border-slate-700 hover:bg-slate-800/50',
                                        'transition-all duration-200 group'
                                    )}
                                >
                                    <action.icon className={cn('w-4 h-4', action.color)} />
                                    <span className="text-sm font-medium text-slate-300 group-hover:text-slate-200"
                                    >
                                        {action.label}
                                    </span>
                                    <ArrowRight className="w-3 h-3 text-slate-600 ml-auto group-hover:text-slate-400 group-hover:translate-x-0.5 transition-all" />
                                </Link>
                            </motion.div>
                        ))}
                    </div>
                </div>
            </motion.div>

            {/* Admin Cards Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
            >
                {ADMIN_CARDS.map((card, index) => (
                    <motion.div
                        key={card.id}
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: 0.2 + index * 0.05 }}
                    >
                        <Link
                            href={card.href}
                            className={cn(
                                'block h-full p-6 rounded-xl border transition-all duration-200 group',
                                'bg-card hover:shadow-lg',
                                card.borderColor,
                                'hover:border-opacity-50'
                            )}
                        >
                            <div className="flex items-start justify-between mb-4"
                            >
                                <div
                                    className={cn(
                                        'w-12 h-12 rounded-xl flex items-center justify-center',
                                        'bg-gradient-to-br text-white shadow-lg',
                                        card.color,
                                        'group-hover:scale-110 transition-transform duration-200'
                                    )}
                                >
                                    <card.icon className="w-6 h-6" />
                                </div>
                                <ArrowRight className="w-5 h-5 text-slate-600 group-hover:text-slate-400 group-hover:translate-x-1 transition-all" />
                            </div>

                            <h3 className="text-lg font-bold text-slate-100 mb-2"
                            >
                                {card.title}
                            </h3>
                            <p className="text-sm text-slate-400 mb-4"
                            >
                                {card.description}
                            </p>

                            <div className="pt-4 border-t border-slate-800/50"
                            >
                                <span className="text-xs font-medium text-slate-500 uppercase tracking-wider"
                                >
                                    {card.stats}
                                </span>
                            </div>
                        </Link>
                    </motion.div>
                ))}
            </div>

            {/* System Status */}
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.6 }}
                className="rounded-xl border bg-card p-6"
            >
                <div className="flex items-center gap-2 mb-4"
                >
                    <Zap className="w-5 h-5 text-amber-500" />
                    <h2 className="text-lg font-bold text-slate-100"
                    >
                        Platform Overview
                    </h2>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4"
                >
                    <div className="p-4 rounded-lg bg-slate-900/50 border border-slate-800"
                    >
                        <div className="flex items-center gap-2 mb-2"
                        >
                            <TrendingUp className="w-4 h-4 text-emerald-400" />
                            <span className="text-sm text-slate-400"
                            >System Status</span>
                        </div>
                        <p className="text-lg font-semibold text-emerald-400"
                        >
                            All Systems Operational
                        </p>
                    </div>

                    <div className="p-4 rounded-lg bg-slate-900/50 border border-slate-800"
                    >
                        <div className="flex items-center gap-2 mb-2"
                        >
                            <Users className="w-4 h-4 text-blue-400" />
                            <span className="text-sm text-slate-400"
                            >User Activity</span>
                        </div>
                        <p className="text-lg font-semibold text-slate-200"
                        >
                            View detailed analytics in <Link href="/stats" className="text-primary hover:underline">Statistics</Link>
                        </p>
                    </div>

                    <div className="p-4 rounded-lg bg-slate-900/50 border border-slate-800"
                    >
                        <div className="flex items-center gap-2 mb-2"
                        >
                            <Shield className="w-4 h-4 text-violet-400" />
                            <span className="text-sm text-slate-400"
                            >Admin Level</span>
                        </div>
                        <p className="text-lg font-semibold text-slate-200"
                        >
                            {user.is_admin ? 'Super Administrator' : 'Staff Member'}
                        </p>
                    </div>
                </div>
            </motion.div>
        </div>
    );
}
