import { NextIntlClientProvider } from 'next-intl';
import { messagesFor } from '@/i18n/scope';

/**
 * Server wrapper for a route section that needs section-only translations
 * (see src/i18n/scope.ts). Replaces the root provider's messages for its
 * subtree with the shared catalog plus `include`.
 */
export default async function ScopedIntl({
    locale,
    include,
    children,
}: {
    locale: string;
    include: readonly string[];
    children: React.ReactNode;
}) {
    return (
        <NextIntlClientProvider locale={locale} messages={await messagesFor(locale, include)}>
            {children}
        </NextIntlClientProvider>
    );
}

/** A layout.tsx body for a section: `export default sectionLayout(['Settings'])`. */
export function sectionLayout(include: readonly string[]) {
    return async function SectionLayout({
        children,
        params,
    }: {
        children: React.ReactNode;
        params: Promise<{ locale: string }>;
    }) {
        const { locale } = await params;
        return <ScopedIntl locale={locale} include={include}>{children}</ScopedIntl>;
    };
}
