'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { Link, usePathname, useRouter } from '@/navigation';
import { useAuth } from '@/components/providers/AuthProvider';
import { cn } from '@/lib/utils';
import {
    LayoutDashboard,
    Users,
    Globe,
    Server,
    Building2,
    FileText,
    Settings,
    Settings2,
    Menu,
    X,
    ScanSearch,
    Shield,
    Ticket,
    MessageSquare,
    Terminal,
    BookOpen,
    Scale,
    FolderTree,
    Code2,
    Tags,
    History,
    PanelTop,
    ArrowLeft,
    type LucideIcon,
} from 'lucide-react';

type NavItem = { href: string; key: string; icon: LucideIcon };

// Grouped the way staff actually work: what is published, who uses it,
// how it is judged, and how the site itself is configured.
const NAV_GROUPS: { key: string; items: NavItem[] }[] = [
    {
        key: 'content',
        items: [
            { href: '/admin/problems', key: 'problems', icon: Code2 },
            { href: '/admin/contests', key: 'contests', icon: Globe },
            { href: '/admin/contests/tags', key: 'contestTags', icon: Tags },
            { href: '/admin/taxonomy', key: 'taxonomy', icon: FolderTree },
            { href: '/admin/blog-posts', key: 'blogPosts', icon: BookOpen },
            { href: '/admin/licenses', key: 'licenses', icon: Scale },
        ],
    },
    {
        key: 'community',
        items: [
            { href: '/admin/users', key: 'users', icon: Users },
            { href: '/admin/groups', key: 'groups', icon: Shield },
            { href: '/admin/organizations', key: 'organizations', icon: Building2 },
            { href: '/admin/tickets', key: 'tickets', icon: Ticket },
            { href: '/admin/comments', key: 'comments', icon: MessageSquare },
        ],
    },
    {
        key: 'judging',
        items: [
            { href: '/admin/submissions', key: 'submissions', icon: FileText },
            { href: '/admin/judges', key: 'judges', icon: Server },
            { href: '/admin/languages', key: 'languages', icon: Terminal },
            { href: '/admin/language-limits', key: 'languageLimits', icon: Settings2 },
            { href: '/admin/moss', key: 'moss', icon: ScanSearch },
        ],
    },
    {
        key: 'system',
        items: [
            { href: '/admin/navigation-bars', key: 'navigationBars', icon: PanelTop },
            { href: '/admin/misc-configs', key: 'miscConfigs', icon: Settings },
            { href: '/admin/audit-logs', key: 'auditLog', icon: History },
        ],
    },
];

const ALL_ITEMS = NAV_GROUPS.flatMap(g => g.items);

export default function AdminLayout({ children }: { children: React.ReactNode }) {
    const t = useTranslations('Admin');
    const pathname = usePathname();
    const router = useRouter();
    const { user, loading } = useAuth();
    const [drawerOpen, setDrawerOpen] = useState(false);

    // Superuser-inclusive: admins (is_admin === is_superuser) pass without
    // is_staff, matching the backend admin route gate (is_staff OR is_superuser).
    const allowed = !!(user?.is_staff || user?.is_admin);

    useEffect(() => {
        if (!loading && !allowed) router.push('/');
    }, [loading, allowed, router]);

    // The most specific match wins so /admin/contests/tags lights up
    // "Contest tags", not "Contests".
    const activeHref = ALL_ITEMS
        .filter(i => pathname === i.href || pathname.startsWith(i.href + '/'))
        .sort((a, b) => b.href.length - a.href.length)[0]?.href
        ?? (pathname === '/admin' ? '/admin' : undefined);

    const activeLabel = ALL_ITEMS.find(i => i.href === activeHref)?.key;

    if (loading || !allowed) {
        return (
            <div className="flex items-center justify-center min-h-[60vh]">
                <div className="animate-spin rounded-full h-10 w-10 border-2 border-muted border-t-primary" />
            </div>
        );
    }

    const navLink = (href: string, label: string, Icon: LucideIcon) => (
        <Link
            key={href}
            href={href}
            onClick={() => setDrawerOpen(false)}
            aria-current={activeHref === href ? 'page' : undefined}
            className={cn(
                'flex items-center gap-2.5 px-2.5 py-1.5 rounded-md text-sm transition-colors',
                activeHref === href
                    ? 'bg-primary/10 text-primary font-semibold'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground'
            )}
        >
            <Icon size={16} className="shrink-0" />
            <span className="truncate">{label}</span>
        </Link>
    );

    const sidebar = (
        <nav aria-label={t('layout.panelTitle')} className="flex flex-col gap-5 p-4">
            {navLink('/admin', t('nav.dashboard'), LayoutDashboard)}
            {NAV_GROUPS.map(group => (
                <div key={group.key}>
                    <h2 className="px-2.5 mb-1 text-xs font-semibold text-muted-foreground">
                        {t(`navGroups.${group.key}`)}
                    </h2>
                    <div className="flex flex-col gap-0.5">
                        {group.items.map(item => navLink(item.href, t(`nav.${item.key}`), item.icon))}
                    </div>
                </div>
            ))}
            <div className="pt-4 border-t text-sm">
                <div className="px-2.5 text-muted-foreground text-xs">{t('layout.loggedInAs')}</div>
                <div className="px-2.5 font-medium truncate">
                    {user!.username}
                    <span className="ml-1.5 text-xs font-normal text-muted-foreground">
                        · {user!.is_admin ? t('layout.superAdmin') : t('layout.staff')}
                    </span>
                </div>
                <Link
                    href="/"
                    onClick={() => setDrawerOpen(false)}
                    className="mt-3 flex items-center gap-2 px-2.5 py-1.5 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                >
                    <ArrowLeft size={16} />
                    {t('layout.backToSite')}
                </Link>
            </div>
        </nav>
    );

    return (
        <div className="flex min-h-[calc(100vh-4rem)] bg-background">
            {/* Desktop sidebar */}
            <aside className="hidden lg:block w-60 shrink-0 border-r bg-secondary/40">
                <div className="sticky top-16 max-h-[calc(100vh-4rem)] overflow-y-auto">
                    {sidebar}
                </div>
            </aside>

            {/* Mobile drawer */}
            {drawerOpen && (
                <div className="lg:hidden fixed inset-0 z-40" role="dialog" aria-modal="true">
                    <button
                        aria-label={t('layout.closeMenu')}
                        className="absolute inset-0 bg-black/40"
                        onClick={() => setDrawerOpen(false)}
                    />
                    <aside className="absolute left-0 top-16 bottom-0 w-72 max-w-[85vw] bg-card border-r overflow-y-auto shadow-xl">
                        {sidebar}
                    </aside>
                </div>
            )}

            <div className="flex-1 min-w-0">
                {/* Mobile section bar, sits under the site navbar */}
                <div className="lg:hidden sticky top-16 z-30 flex items-center gap-2 h-12 px-4 border-b bg-card">
                    <button
                        onClick={() => setDrawerOpen(o => !o)}
                        aria-expanded={drawerOpen}
                        aria-label={drawerOpen ? t('layout.closeMenu') : t('layout.openMenu')}
                        className="p-1.5 -ml-1.5 rounded-md hover:bg-muted"
                    >
                        {drawerOpen ? <X size={20} /> : <Menu size={20} />}
                    </button>
                    <span className="text-sm font-semibold">{t('layout.panelTitle')}</span>
                    {activeLabel && (
                        <span className="text-sm text-muted-foreground truncate">/ {t(`nav.${activeLabel}`)}</span>
                    )}
                </div>

                <main className="p-4 lg:p-8 max-w-[1400px]">
                    {children}
                </main>
            </div>
        </div>
    );
}
