'use client';

import { Link, usePathname } from '@/navigation';
import { useTranslations } from 'next-intl';
import { RefreshCw } from 'lucide-react';
import { cn } from '@/lib/utils';
import { findActiveKey, type NavLink } from '@/lib/navbar';

interface DesktopNavProps {
    links: NavLink[];
}

export default function DesktopNav({ links }: DesktopNavProps) {
    const pathname = usePathname();
    const t = useTranslations('Navbar');
    // v1 marks the first row whose highlight regex matches the current path.
    const activeKey = findActiveKey(links, pathname);

    return (
        <nav className="hidden xl:flex items-center gap-1" aria-label={t('mainNavigation')}>
            <span className="text-muted-foreground/50 mx-1">|</span>
            {links.map((link) => (
                <Link
                    key={link.href}
                    href={link.href}
                    aria-current={link.key === activeKey ? 'page' : undefined}
                    className={cn(
                        "px-3 py-2 text-sm font-medium transition-colors rounded",
                        link.key === activeKey
                            ? "text-primary bg-muted/50"
                            : "text-muted-foreground hover:text-foreground hover:bg-accent/10"
                    )}
                >
                    {link.label ?? t(link.tKey ?? link.key)}
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
                title={t('randomProblem')}
            >
                <RefreshCw size={14} />
            </Link>
        </nav>
    );
}
