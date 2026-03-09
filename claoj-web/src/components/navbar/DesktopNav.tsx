'use client';

import { Link, usePathname } from '@/navigation';
import { useTranslations } from 'next-intl';
import { RefreshCw } from 'lucide-react';
import { cn } from '@/lib/utils';

const NAV_LINKS = [
    { href: '/problems', key: 'problems' },
    { href: '/contests', key: 'contests' },
    { href: '/submissions', key: 'submissions' },
    { href: '/users', key: 'users' },
    { href: '/ratings', key: 'ratings' },
    { href: '/organizations', key: 'organizations' },
    { href: '/stats', key: 'stats' },
];

export default function DesktopNav() {
    const pathname = usePathname();
    const t = useTranslations('Navbar');

    return (
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
    );
}
