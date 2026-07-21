'use client';

import { useEffect, useState } from 'react';
import { useTheme } from 'next-themes';
import { useTranslations } from 'next-intl';
import { Moon, Sun } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ThemeToggleProps {
    variant?: 'default' | 'mobile';
}

export default function ThemeToggle({ variant = 'default' }: ThemeToggleProps) {
    // `resolvedTheme` is the concrete "dark"/"light" after resolving "system";
    // `theme` is "system" on first load, which would show the wrong icon and
    // make the first click a no-op. Always key off `resolvedTheme`.
    const t = useTranslations('Navbar');
    const { resolvedTheme, setTheme } = useTheme();
    const [mounted, setMounted] = useState(false);

    useEffect(() => {
        setMounted(true);
    }, []);

    const isDark = resolvedTheme === 'dark';

    const handleClick = () => {
        setTheme(isDark ? 'light' : 'dark');
    };

    // Gate the label on `mounted` like the icon: before mount `resolvedTheme`
    // is undefined (SSR/first paint), so a neutral label keeps the server and
    // client markup in sync. Deriving it straight from `isDark` leaves the
    // aria-label/title stuck at the server value until the first interaction.
    const label = !mounted
        ? t('toggleTheme')
        : isDark ? t('switchToLightMode') : t('switchToDarkMode');

    if (variant === 'mobile') {
        return (
            <button
                onClick={handleClick}
                className="flex items-center gap-2 p-3 rounded bg-muted text-sm font-bold"
                aria-label={label}
            >
                {mounted ? (
                    isDark ? <Sun size={18} /> : <Moon size={18} />
                ) : (
                    <Moon size={18} className="opacity-50" />
                )}
                <span>{t('themeWithMode', { mode: mounted ? (isDark ? t('themeDark') : t('themeLight')) : '...' })}</span>
            </button>
        );
    }

    return (
        <button
            onClick={handleClick}
            className="p-2 rounded-full hover:bg-accent/10 transition-colors text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            aria-label={label}
            title={label}
        >
            {mounted ? (
                isDark ? <Sun size={18} /> : <Moon size={18} />
            ) : (
                <Moon size={18} className="opacity-50" />
            )}
        </button>
    );
}
