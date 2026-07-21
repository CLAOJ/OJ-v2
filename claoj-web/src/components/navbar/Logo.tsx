'use client';

import { useTranslations } from 'next-intl';

// Theme-aware logo.
//
// The site theme is class-based and *inverted* from the usual convention:
// `:root` holds the dark palette (the default) and a `.light` class on <html>
// overrides it for light mode (see globals.css). next-themes sets that class in
// a blocking script before first paint, so the correct logo can be chosen purely
// in CSS — no `useTheme`/mounted gate, which previously rendered the dark logo
// on a dark background for a frame before hydration (the "invisible logo" flash).
//
// Both images are always in the DOM; CSS shows exactly one. Keying off `.light`
// (not Tailwind's `dark:` variant, which is unconfigured here and would follow
// prefers-color-scheme instead of the toggle) keeps it correct when a user
// overrides their OS theme.
export default function Logo() {
    const t = useTranslations('Navbar');

    return (
        <>
            {/* Light/white logo — default (dark theme); hidden when .light is active */}
            <img
                src="/static/claoj-logo-light.png"
                alt={t('logoAlt')}
                className="h-8 w-auto object-contain block [.light_&]:hidden"
            />
            {/* Dark/black logo — shown only in light theme */}
            <img
                src="/static/claoj-logo-dark.png"
                alt={t('logoAlt')}
                className="h-8 w-auto object-contain hidden [.light_&]:block"
            />
        </>
    );
}
