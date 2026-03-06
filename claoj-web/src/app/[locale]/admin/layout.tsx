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

const adminLinks = [
    {
        href: '/admin/users',
        label: 'Users',
        icon: Users
    },
    {
        href: '/admin/contests',
        label: 'Contests',
        icon: Globe
    },
    {
        href: '/admin/problems',
        label: 'Problems',
        icon: Code2
    },
    {
        href: '/admin/judges',
        label: 'Judges',
        icon: Server
    },
    {
        href: '/admin/organizations',
        label: 'Organizations',
        icon: Database
    },
    {
        href: '/admin/submissions',
        label: 'Submissions',
        icon: FileText
    },
    {
        href: '/admin/tickets',
        label: 'Tickets',
        icon: Ticket
    },
    {
        href: '/admin/comments',
        label: 'Comments',
        icon: MessageSquare
    },
    {
        href: '/admin/blog-posts',
        label: 'Blog Posts',
        icon: BookOpen
    },
    {
        href: '/admin/languages',
        label: 'Languages',
        icon: Terminal
    },
    {
        href: '/admin/language-limits',
        label: 'Language Limits',
        icon: Settings2
    },
    {
        href: '/admin/licenses',
        label: 'Licenses',
        icon: Scale
    },
    {
        href: '/admin/taxonomy',
        label: 'Taxonomy',
        icon: Folder
    },
    {
        href: '/admin/moss',
        label: 'MOSS',
        icon: BarChart3
    },
    {
        href: '/admin/roles',
        label: 'Roles',
        icon: Shield
    },
    {
        href: '/admin/navigation-bars',
        label: 'Navigation',
        icon: Menu
    },
    {
        href: '/admin/misc-configs',
        label: 'Misc Config',
        icon: Settings
    }
];

export default function AdminLayout({ children }: { children: React.ReactNode }) {
    const [isSidebarOpen, setIsSidebarOpen] = useState(false);
    const pathname = usePathname();
    const { user, logout } = useAuth();

    const isActive = (path: string) => pathname === path || pathname.startsWith(path + '/');

    return (
        <div className="min-h-screen bg-background">
            {/* Mobile Header */}
            <header className="fixed top-0 left-0 right-0 h-16 bg-card border-b z-50 flex items-center justify-between px-4 lg:hidden">
                <div className="flex items-center gap-2">
                    <LayoutDashboard className="text-primary" size={24} />
                    <span className="font-bold text-lg">Admin</span>
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
                            <span className="font-bold text-xl">Admin Panel</span>
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
                                <div className="text-xs text-muted-foreground mb-1">Logged in as</div>
                                <div className="text-sm font-medium truncate">
                                    {user?.username || 'Admin'}
                                </div>
                                {user?.is_admin && (
                                    <Badge variant="warning" className="mt-1 text-[10px]">
                                        Admin
                                    </Badge>
                                )}
                            </div>

                            <button
                                onClick={logout}
                                className="flex items-center gap-3 w-full px-3 py-2.5 rounded-xl text-sm font-medium text-destructive hover:bg-destructive/10 transition-colors"
                            >
                                <LogOut size={18} />
                                Logout
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
