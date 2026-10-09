'use client';

import { useTranslations } from 'next-intl';

// The navbar keeps v1's blue-grey bar (#263238) in both themes (see `.site-nav`
// in globals.css), so the light/white logo is always the right one.
export default function Logo() {
    const t = useTranslations('Navbar');

    return (
        <img
            src="/static/claoj-logo-light.png"
            alt={t('logoAlt')}
            className="h-8 w-auto max-w-none shrink-0 object-contain block"
        />
    );
}
