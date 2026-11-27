'use client';

import { Link, usePathname } from '@/navigation';
import { useTranslations } from 'next-intl';
import { ChevronDown, RefreshCw } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { cn } from '@/lib/utils';
import { findActiveKeys, type NavLink } from '@/lib/navbar';

interface DesktopNavProps {
    links: NavLink[];
}

export default function DesktopNav({ links }: DesktopNavProps) {
    const pathname = usePathname();
    const t = useTranslations('Navbar');
    // v1 marks the first matching row plus its ancestors, so a parent reads as
    // active when the current page is one of its children.
    const activeKeys = findActiveKeys(links, pathname);
    const [openKey, setOpenKey] = useState<string | null>(null);
    const navRef = useRef<HTMLElement>(null);

    // Outside click closes the dropdown; Escape closes and returns focus to
    // the toggle that opened it.
    useEffect(() => {
        if (openKey === null) return;
        const handlePointerDown = (event: MouseEvent) => {
            if (navRef.current && !navRef.current.contains(event.target as Node)) {
                setOpenKey(null);
            }
        };
        document.addEventListener('mousedown', handlePointerDown);
        return () => document.removeEventListener('mousedown', handlePointerDown);
    }, [openKey]);

    const handleKeyDown = (event: React.KeyboardEvent) => {
        if (event.key !== 'Escape' || openKey === null) return;
        const toggle = navRef.current?.querySelector<HTMLElement>('[aria-expanded="true"]');
        setOpenKey(null);
        toggle?.focus();
    };

    return (
        <nav
            ref={navRef}
            className="hidden xl:flex items-center gap-1"
            aria-label={t('mainNavigation')}
            onKeyDown={handleKeyDown}
        >
            <span className="text-muted-foreground/50 mx-1">|</span>
            {links.map((link) => {
                const active = activeKeys.has(link.key);
                const hasChildren = !!link.children?.length;
                const open = openKey === link.key;
                const label = link.label ?? t(link.tKey ?? link.key);

                return (
                    <div
                        key={link.href}
                        className="relative"
                        // v1 opens the submenu on hover; the toggle below covers
                        // click and keyboard.
                        onMouseEnter={() => hasChildren && setOpenKey(link.key)}
                        onMouseLeave={() => setOpenKey((current) => (current === link.key ? null : current))}
                    >
                        <div className="flex items-center">
                            <Link prefetch={false}
                                href={link.href}
                                aria-current={active ? 'page' : undefined}
                                className={cn(
                                    "px-3 py-2 text-sm font-medium transition-colors rounded",
                                    active
                                        ? "text-primary bg-muted/50"
                                        : "text-muted-foreground hover:text-foreground hover:bg-accent/10"
                                )}
                            >
                                {label}
                            </Link>
                            {hasChildren && (
                                <button
                                    type="button"
                                    onClick={() => setOpenKey(open ? null : link.key)}
                                    aria-expanded={open}
                                    aria-haspopup="true"
                                    aria-controls={`nav-menu-${link.key}`}
                                    aria-label={label}
                                    className="p-1 text-muted-foreground hover:text-foreground rounded transition-colors"
                                >
                                    <ChevronDown size={12} className={cn("transition-transform", open && "rotate-180")} />
                                </button>
                            )}
                        </div>

                        {hasChildren && open && (
                            <div
                                id={`nav-menu-${link.key}`}
                                className="absolute left-0 top-full min-w-48 rounded border bg-card shadow-xl py-1 z-50"
                            >
                                {link.children!.map((child) => {
                                    const childActive = activeKeys.has(child.key);
                                    return (
                                        <Link prefetch={false}
                                            key={child.href}
                                            href={child.href}
                                            onClick={() => setOpenKey(null)}
                                            aria-current={childActive ? 'page' : undefined}
                                            className={cn(
                                                "block whitespace-nowrap px-4 py-2 text-sm transition-colors",
                                                childActive
                                                    ? "text-primary bg-muted/50"
                                                    : "text-muted-foreground hover:text-foreground hover:bg-muted"
                                            )}
                                        >
                                            {child.label ?? t(child.tKey ?? child.key)}
                                        </Link>
                                    );
                                })}
                            </div>
                        )}
                    </div>
                );
            })}
            <Link prefetch={false}
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