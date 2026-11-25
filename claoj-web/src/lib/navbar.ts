// Public header items: v1's judge_navigationbar rows (GET /navigation) plus the
// v2-only entries that table never held.

export interface NavigationItem {
    key: string;
    label: string;
    path: string;
    regex: string;
    children?: NavigationItem[];
}

export interface NavLink {
    key: string;
    href: string;
    /** DB label: site content, rendered as-is. */
    label?: string;
    /** Translation key for entries that are not DB content. */
    tKey?: string;
    /** v1 highlight regex (Python); a path prefix is used when it cannot run. */
    regex?: string;
}

// v1's routes that v2 serves elsewhere. Everything else keeps its path minus
// the trailing slash, because the flat-page catch-all serves those.
const PATH_MAP: Record<string, string> = {
    '/problems/': '/problems',
    '/submissions/': '/submissions',
    '/users/': '/users',
    '/organizations/': '/organizations',
    '/contests/': '/contests',
    '/status/': '/stats',
};

export function normalizeNavPath(raw: string): string {
    const path = raw.trim();
    if (path === '') return '/';
    if (/^https?:\/\//i.test(path)) return path;
    if (PATH_MAP[path]) return PATH_MAP[path];
    const collapsed = ('/' + path).replace(/\/{2,}/g, '/').replace(/\/+$/, '');
    return collapsed === '' ? '/' : collapsed;
}

// Today's static links, used until the endpoint answers (and if it never does).
export const FALLBACK_NAV_LINKS: NavLink[] = [
    { key: 'problems', href: '/problems', tKey: 'problems' },
    { key: 'contests', href: '/contests', tKey: 'contests' },
    { key: 'submissions', href: '/submissions', tKey: 'submissions' },
    { key: 'users', href: '/users', tKey: 'users' },
    { key: 'ratings', href: '/ratings', tKey: 'ratings' },
    { key: 'organizations', href: '/organizations', tKey: 'organizations' },
    { key: 'stats', href: '/stats', tKey: 'stats' },
];

function flatten(items: NavigationItem[]): NavigationItem[] {
    // v1 hangs children under their parent as a dropdown; the v2 header is a
    // single row, so children follow their parent instead of disappearing.
    return items.flatMap((item) => [item, ...flatten(item.children ?? [])]);
}

export function buildNavLinks(
    items: NavigationItem[] | null | undefined,
    extras: NavLink[] = [],
): NavLink[] {
    const links: NavLink[] = [];
    const seen = new Set<string>();
    const add = (link: NavLink) => {
        if (seen.has(link.href)) return;
        seen.add(link.href);
        links.push(link);
    };

    if (items?.length) {
        for (const item of flatten(items)) {
            add({
                key: item.key,
                label: item.label,
                href: normalizeNavPath(item.path),
                regex: item.regex,
            });
        }
    } else {
        FALLBACK_NAV_LINKS.forEach(add);
    }
    extras.forEach(add);
    return links;
}

function isExternal(href: string): boolean {
    return /^https?:\/\//i.test(href);
}

function prefixMatches(href: string, pathname: string): boolean {
    if (href === '/') return pathname === '/';
    return pathname === href || pathname.startsWith(href + '/');
}

/** Returns null when the pattern does not compile (e.g. Python's (?x) flag). */
function regexMatches(pattern: string, pathname: string): boolean | null {
    try {
        const re = new RegExp(pattern);
        // v1 URLs end in a slash, v2 routes do not: try both so patterns like
        // ^/problems/ still light up /problems and /problems/123.
        return re.test(pathname) || (!pathname.endsWith('/') && re.test(pathname + '/'));
    } catch {
        return null;
    }
}

export function isActiveLink(link: NavLink, pathname: string): boolean {
    if (isExternal(link.href)) return false;
    if (link.regex) {
        const matched = regexMatches(link.regex, pathname);
        // A pattern that compiles is authoritative, exactly like v1; only an
        // uncompilable one degrades to the path prefix.
        if (matched !== null) return matched;
    }
    return prefixMatches(link.href, pathname);
}

/** v1 marks the first row whose regex matches the request path. */
export function findActiveKey(links: NavLink[], pathname: string): string | null {
    return links.find((link) => isActiveLink(link, pathname))?.key ?? null;
}