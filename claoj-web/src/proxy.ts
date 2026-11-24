import createMiddleware from 'next-intl/middleware';
import { NextRequest, NextResponse } from 'next/server';
import { routing } from './navigation';
import { buildRedirectMap, pickRedirectTarget, type RedirectPair } from './lib/redirects';

const intlMiddleware = createMiddleware(routing);

// django_redirect rows are small in number and edited rarely, so one fetch per
// isolate per minute is enough for path redirects.
const REDIRECTS_TTL_MS = 60_000;
const FAILED_FETCH_TTL_MS = 15_000;

let cached: { map: Map<string, string>; expires: number } | null = null;
// One refresh in flight per isolate: concurrent requests share it instead of
// each hitting the backend. shortcut: the URL is derived from the first
// request's origin while NEXT_PUBLIC_API_URL is unset, fine for one origin.
let pending: Promise<Map<string, string> | null> | null = null;

// Same base URL rule as src/lib/api.ts: NEXT_PUBLIC_API_URL when configured,
// otherwise /api on the public origin (nginx forwards it to the backend).
function redirectsMapURL(request: NextRequest): string {
    const base = process.env.NEXT_PUBLIC_API_URL || `${request.nextUrl.origin}/api`;
    return `${base}/redirects-map`;
}

/** Fetches and builds the map; null means "no fresh map" (network or status). */
async function fetchRedirectsMap(url: string): Promise<Map<string, string> | null> {
    try {
        const res = await fetch(url, {
            headers: { Accept: 'application/json' },
            signal: AbortSignal.timeout(1500),
            cache: 'no-store',
        });
        if (!res.ok) return null;
        const body = (await res.json()) as { data?: RedirectPair[] };
        return buildRedirectMap(body.data ?? []);
    } catch {
        return null;
    }
}

async function loadRedirects(request: NextRequest): Promise<Map<string, string>> {
    if (cached && cached.expires > Date.now()) return cached.map;

    if (!pending) {
        pending = fetchRedirectsMap(redirectsMapURL(request)).finally(() => {
            pending = null;
        });
    }
    const map = await pending;

    if (map) {
        cached = { map, expires: Date.now() + REDIRECTS_TTL_MS };
        return map;
    }

    // Backend unreachable or answered non-2xx: keep serving the last good map
    // (stale beats no redirects) and retry sooner. Without one, fail open.
    const fallback = cached?.map ?? new Map<string, string>();
    cached = { map: fallback, expires: Date.now() + FAILED_FETCH_TTL_MS };
    return fallback;
}

// Path redirects come first: v1's RedirectFallbackMiddleware matched them
// before the app's own URLs, and next-intl must not rewrite "/old/" to a
// locale-prefixed internal path before the redirect is applied.
export default async function proxy(request: NextRequest) {
    if (request.method === 'GET' || request.method === 'HEAD') {
        const map = await loadRedirects(request);
        const target = pickRedirectTarget(map, request.nextUrl.pathname);
        if (target) {
            if (/^https?:\/\//i.test(target)) {
                const url = new URL(target);
                if (!url.search) url.search = request.nextUrl.search;
                return NextResponse.redirect(url, 301);
            }
            const url = request.nextUrl.clone();
            url.pathname = target;
            return NextResponse.redirect(url, 301);
        }
    }
    return intlMiddleware(request);
}

export const config = {
    matcher: ['/((?!api|static|_next|_vercel|.*\\..*).*)']
};
