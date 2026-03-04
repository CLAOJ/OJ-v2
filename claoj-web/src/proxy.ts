import createMiddleware from 'next-intl/middleware';
import type { NextRequest } from 'next/server';

// Next.js 16 proxy convention (replaces deprecated middleware)
const intlMiddleware = createMiddleware({
    // A list of all locales that are supported
    locales: ['en', 'vi'],

    // Used when no locale matches
    defaultLocale: 'en',
});

// Export as default "proxy" function for Next.js 16
export default function proxy(request: NextRequest) {
    return intlMiddleware(request);
}

// Matcher configuration for Next.js 16 proxy
export const matcher = ['/', '/(vi|en)/:path*'];
