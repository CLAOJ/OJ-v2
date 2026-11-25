'use client';

import { useTranslations } from 'next-intl';

// Full-colour logo for theme-following surfaces (auth cards). The navbar uses
// navbar/Logo, which always sits on the dark bar.
export default function BrandLogo({ className = 'h-12' }: { className?: string }) {
    const t = useTranslations('Navbar');
    return (
        <>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src="/static/claoj-logo-dark.png" alt={t('logoAlt')} width={302} height={83} className={`${className} w-auto mx-auto dark:hidden`} />
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src="/static/claoj-logo-light.png" alt={t('logoAlt')} width={302} height={83} className={`${className} w-auto mx-auto hidden dark:block`} />
        </>
    );
}
