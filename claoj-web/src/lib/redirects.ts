/**
 * Redirect lookup for the Next proxy, fed by GET /api/redirects-map
 * (Django's django_redirect table, site-scoped).
 */

export interface RedirectPair {
    old_path: string;
    new_path: string;
}

// A chain longer than this is treated as a loop rather than followed.
const MAX_HOPS = 10;

/**
 * Django's RedirectFallbackMiddleware looks up request.path exactly, and
 * CommonMiddleware's APPEND_SLASH retries a slashed variant of the URL before
 * that. Mirror both, so "/old" finds the row stored as "/old/".
 */
export function lookupRedirect(map: Map<string, string>, pathname: string): string | null {
    const direct = map.get(pathname);
    if (direct !== undefined) return direct;
    if (!pathname.endsWith('/')) {
        const slashed = map.get(pathname + '/');
        if (slashed !== undefined) return slashed;
    }
    return null;
}

/** Follows a chain to its terminal target; null when it loops or never ends. */
function resolveTarget(map: Map<string, string>, start: string): string | null {
    let target = lookupRedirect(map, start);
    const seen = new Set<string>([start]);
    for (let hop = 0; target !== null && hop < MAX_HOPS; hop++) {
        if (/^https?:\/\//i.test(target)) return target;
        if (seen.has(target)) return null;
        seen.add(target);
        const next = lookupRedirect(map, target);
        if (next === null) return target;
        target = next;
    }
    return null;
}

/**
 * Builds the lookup map with chains already collapsed, so /a/ -> /b/ -> /c/
 * becomes /a/ -> /c/ and the proxy issues one hop. Entries whose chain loops
 * or ends on themselves are dropped: they 404 instead of redirecting forever.
 */
export function buildRedirectMap(pairs: RedirectPair[]): Map<string, string> {
    const raw = new Map<string, string>();
    for (const pair of pairs) {
        if (typeof pair?.old_path === 'string' && typeof pair?.new_path === 'string') {
            raw.set(pair.old_path, pair.new_path);
        }
    }

    const map = new Map<string, string>();
    for (const oldPath of raw.keys()) {
        const target = resolveTarget(raw, oldPath);
        if (target !== null) map.set(oldPath, target);
    }
    return map;
}

/**
 * Resolves one hop for a request path. The map's targets are terminal (see
 * buildRedirectMap), so the result is where the client ends up.
 * Absolute http(s) targets are returned untouched.
 */
export function pickRedirectTarget(map: Map<string, string>, pathname: string): string | null {
    const target = lookupRedirect(map, pathname);
    if (!target || target === pathname) return null;
    return target;
}
