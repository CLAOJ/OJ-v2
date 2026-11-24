/**
 * Redirect lookup for the Next proxy, fed by GET /api/redirects-map
 * (Django's django_redirect table, site-scoped).
 */

export interface RedirectPair {
    old_path: string;
    new_path: string;
}

/** Builds the lookup map; invalid pairs are skipped rather than trusted. */
export function buildRedirectMap(pairs: RedirectPair[]): Map<string, string> {
    const map = new Map<string, string>();
    for (const pair of pairs) {
        if (typeof pair?.old_path === 'string' && typeof pair?.new_path === 'string') {
            map.set(pair.old_path, pair.new_path);
        }
    }
    return map;
}

/**
 * Django's RedirectFallbackMiddleware looks up request.path exactly, then
 * retries with a trailing slash (APPEND_SLASH). Same here, so "/old" finds
 * the row stored as "/old/".
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

/**
 * Resolves at most ONE hop. Returns null (no redirect) when:
 * - there is no matching old_path,
 * - the target is the requested path itself (self loop),
 * - the target is itself an old_path (following it would chain, and a cycle
 *   among two entries is unbounded).
 * Absolute http(s) targets are returned untouched.
 */
export function pickRedirectTarget(map: Map<string, string>, pathname: string): string | null {
    const target = lookupRedirect(map, pathname);
    if (!target || target === pathname) return null;
    if (/^https?:\/\//i.test(target)) return target;
    if (lookupRedirect(map, target) !== null) return null;
    return target;
}