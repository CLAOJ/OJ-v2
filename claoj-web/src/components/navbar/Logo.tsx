'use client';

import { useTheme } from 'next-themes';
import { useState, useEffect } from 'react';

export default function Logo() {
    const { theme, resolvedTheme } = useTheme();
    const [mounted, setMounted] = useState(false);

    useEffect(() => {
        setMounted(true);
    }, []);

    // Use resolvedTheme to handle 'system' theme correctly
    const currentTheme = mounted ? (resolvedTheme || theme) : 'light';

    // Use dark logo for light theme, light logo for dark theme
    const logoSrc = currentTheme === 'dark'
        ? '/static/claoj-logo-light.png'
        : '/static/claoj-logo-dark.png';

    return (
        <img
            src={logoSrc}
            alt="CLAOJ"
            className="h-8 w-auto object-contain"
        />
    );
}
