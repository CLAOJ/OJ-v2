'use client';

import { usePathname } from 'next/navigation';
import { Badge } from '@/components/ui/Badge';
import { Link } from '@/navigation';
import {
    LayoutDashboard,
    Users,
    UsersRound,
    Globe,
    Server,
    Database,
    Code2,
    FileText,
    Settings,
    Settings2,
    LogOut,
    Menu,
    X,
    BarChart3,
    Shield,
    Ticket,
    MessageSquare,
    Terminal,
    BookOpen,
    Scale,
    Folder
} from 'lucide-react';
import { useState } from 'react';
import { useAuth } from '@/components/providers/AuthProvider';
import { useTranslations } from 'next-intl';

export default function AdminLayout({ children }: { children: React.ReactNode }) {
    const t = useTranslations('Admin');
    const [isSidebarOpen, setIsSidebarOpen] = useState(false);
    const pathname = usePathname();
    const { user, logout } = useAuth();

    const adminLinks = [
        {
            href: '/admin/users',
            label: t('nav.users'),
            icon: Users
        },
        {
            href: '/admin/contests',
            label: t('nav.contests'),
            icon: Globe
        },
        {
            href: '/admin/problems',
            label: t('nav.problems'),
            icon: Code2
        },
        {
            href: '/admin/judges',
            label: t('nav.judges'),
            icon: Server
        },
        {
            href: '/admin/organizations',
            label: t('nav.organizations'),
            icon: Database
        },
        {
            href: '/admin/submissions',
            label: t('nav.submissions'),
            icon: FileText
        },
        {
            href: '/admin/tickets',
            label: t('nav.tickets'),
            icon: Ticket
        },
        {
            href: '/admin/comments',
            label: t('nav.comments'),
            icon: MessageSquare
        },
        {
            href: '/admin/blog-posts',
            label: t('nav.blogPosts'),
            icon: BookOpen
        },
        {
            href: '/admin/languages',
            label: t('nav.languages'),
            icon: Terminal
        },
        {
            href: '/admin/language-limits',
            label: t('nav.languageLimits'),
            icon: Settings2
        },
        {
            href: '/admin/licenses',
            label: t('nav.licenses'),
            icon: Scale
        },
        {
            href: '/admin/taxonomy',
            label: t('nav.taxonomy'),
            icon: Folder
        },
        {
            href: '/admin/moss',
            label: t('nav.moss'),
            icon: BarChart3
        },
        {
            href: '/admin/groups',
            label: t('nav.groups'),
            icon: Shield
        },
        {
            href: '/admin/navigation-bars',
            label: t('nav.navigationBars'),
            icon: Menu
        },
        {
            href: '/admin/misc-configs',
            label: t('nav.miscConfigs'),
            icon: Settings
        }
    ];

    const isActive = (path: string) => pathname === path || pathname.startsWith(path + '/');

    return (
        <div className="min-h-screen bg-background">
            {/* Mobile Header */}
            <header className="fixed top-0 left-0 right-0 h-16 bg-card border-b z-50 flex items-center justify-between px-4 lg:hidden">
                <div className="flex items-center gap-2">
                    <LayoutDashboard className="text-primary" size={24} />
                    <span className="font-bold text-lg">{t('layout.adminLabel')}</span>
                </div>
                <button
                    onClick={() => setIsSidebarOpen(!isSidebarOpen)}
                    className="p-2 hover:bg-muted rounded-lg"
                >
                    {isSidebarOpen ? <X size={24} /> : <Menu size={24} />}
                </button>
            </header>

            <div className="flex pt-16 lg:pt-0">
                {/* Sidebar */}
                <aside
                    className={`
                        fixed lg:sticky top-16 lg:top-0 left-0 h-full w-64 bg-card border-r z-40
                        transition-transform duration-300 ease-in-out
                        ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
                    `}
                >
                    <div className="flex flex-col h-full p-4">
                        <div className="hidden lg:flex items-center gap-2 mb-8 px-2">
                            <LayoutDashboard className="text-primary" size={28} />
                            <span className="font-bold text-xl">{t('layout.panelTitle')}</span>
                        </div>

                        <nav className="space-y-1 flex-1">
                            {adminLinks.map((link) => (
                                <Link
                                    key={link.href}
                                    href={link.href}
                                    onClick={() => setIsSidebarOpen(false)}
                                    className={`
                                        flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all
                                        ${isActive(link.href)
                                            ? 'bg-primary text-primary-foreground shadow-lg shadow-primary/20'
                                            : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                                        }
                                    `}
                                >
                                    <link.icon size={18} />
                                    {link.label}
                                </Link>
                            ))}
                        </nav>

                        <div className="mt-auto pt-4 border-t space-y-3">
                            <div className="px-3 py-2 bg-muted/30 rounded-xl">
                                <div className="text-xs text-muted-foreground mb-1">{t('layout.loggedInAs')}</div>
                                <div className="text-sm font-medium truncate">
                                    {user?.username || t('layout.adminLabel')}
                                </div>
                                {user?.is_admin && (
                                    <Badge variant="warning" className="mt-1 text-[10px]">
                                        {t('layout.adminLabel')}
                                    </Badge>
                                )}
                            </div>

                            <button
                                onClick={logout}
                                className="flex items-center gap-3 w-full px-3 py-2.5 rounded-xl text-sm font-medium text-destructive hover:bg-destructive/10 transition-colors"
                            >
                                <LogOut size={18} />
                                {t('layout.logout')}
                            </button>
                        </div>
                    </div>
                </aside>

                {/* Overlay for mobile */}
                {isSidebarOpen && (
                    <div
                        className="fixed inset-0 bg-black/20 z-30 lg:hidden"
                        onClick={() => setIsSidebarOpen(false)}
                    />
                )}

                {/* Main Content */}
                <main className="flex-1 p-4 lg:p-8 w-full">
                    {children}
                </main>
            </div>
        </div>
    );
}
