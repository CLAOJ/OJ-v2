import {
    Shield,
    Users,
    FileText,
    Trophy,
    Code,
    MessageSquare,
    BarChart3,
    Layout,
    Crown,
    Activity,
    Globe,
    Terminal,
} from 'lucide-react';

export interface AdminSection {
    id: string;
    label: string;
    icon: React.ComponentType<{ className?: string }>;
    href: string;
    color: string;
    badge?: string;
}

export const ADMIN_SECTIONS: AdminSection[] = [
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
        label: 'Groups',
        icon: Shield,
        href: '/admin/groups',
        color: 'from-red-500 to-rose-500',
    },
];

export const QUICK_ACTIONS = [
    { id: 'newProblem', icon: Code, label: 'New Problem', href: '/admin/problems/create', color: 'text-blue-400' },
    { id: 'newContest', icon: Trophy, label: 'New Contest', href: '/admin/contests/create', color: 'text-amber-400' },
    { id: 'manageUsers', icon: Users, label: 'Manage Users', href: '/admin/users', color: 'text-violet-400' },
    { id: 'analytics', icon: BarChart3, label: 'Analytics', href: '/admin', color: 'text-emerald-400' },
];

export const ADMIN_STATS = [
    { id: 'systemStatus', label: 'System Status', value: 'Operational', color: 'text-emerald-400' },
    { id: 'activeUsers', label: 'Active Users', value: 'View Analytics', color: 'text-blue-400', href: '/admin/users' },
    { id: 'pendingTickets', label: 'Pending Tickets', value: 'Check Now', color: 'text-amber-400', href: '/admin/tickets' },
];
