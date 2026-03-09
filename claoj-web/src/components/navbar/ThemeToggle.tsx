'use client';

import { useEffect, useState } from 'react';
import { useTheme } from 'next-themes';
import { Moon, Sun } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ThemeToggleProps {
    variant?: 'default' | 'mobile';
}

export default function ThemeToggle({ variant = 'default' }: ThemeToggleProps) {
    const { theme, setTheme } = useTheme();
    const [mounted, setMounted] = useState(false);

    useEffect(() => {
        setMounted(true);
    }, []);

    const handleClick = () => {
        setTheme(theme === 'dark' ? 'light' : 'dark');
    };

    if (variant === 'mobile') {
        return (
            <button
                onClick={handleClick}
                className="flex items-center gap-2 p-3 rounded bg-muted text-sm font-bold"
                aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
            >
                {mounted ? (
                    theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />
                ) : (
                    <Moon size={18} className="opacity-50" />
                )}
                <span>Theme ({mounted ? (theme === 'dark' ? 'Dark' : 'Light') : '...'})</span>
            </button>
        );
    }

    return (
        <button
            onClick={handleClick}
            className="p-2 rounded-full hover:bg-accent/10 transition-colors text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
            title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
        >
            {mounted ? (
                theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />
            ) : (
                <Moon size={18} className="opacity-50" />
            )}
        </button>
    );
}
