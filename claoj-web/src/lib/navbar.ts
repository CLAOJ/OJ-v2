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
    /** DB children, rendered as a dropdown (v1's nested mptt_tree). */
    children?: NavLink[];
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

// v1 passes every DB label through gettext (`{{ _(node.label) }}`). Labels the
// catalogs do not know stay verbatim: admins can add arbitrary rows.
export function translateDbLabel(
    label: string,
    labels: Record<string, string> | undefined,
): string {
    return labels?.[label] ?? label;
}

export function buildNavLinks(
    items: NavigationItem[] | null | undefined,
    extras: NavLink[] = [],
    dbLabels?: Record<string, string>,
): NavLink[] {
    const links: NavLink[] = [];
    const seen = new Set<string>();
    const add = (link: NavLink): boolean => {
        if (seen.has(link.href)) return false;
        seen.add(link.href);
        return true;
    };

    // Children stay nested: v1 renders them as a dropdown under the parent.
    const toLink = (item: NavigationItem): NavLink | null => {
        const link: NavLink = {
            key: item.key,
            label: translateDbLabel(item.label, dbLabels),
            href: normalizeNavPath(item.path),
            regex: item.regex,
        };
        if (!add(link)) return null;
        const children = (item.children ?? [])
            .map(toLink)
            .filter((child): child is NavLink => child !== null);
        if (children.length) link.children = children;
        return link;
    };

    if (items?.length) {
        for (const item of items) {
            const link = toLink(item);
            if (link) links.push(link);
        }
    } else {
        for (const link of FALLBACK_NAV_LINKS) {
            if (add(link)) links.push(link);
        }
    }
    // v2-only entries are always top level and only appear once.
    for (const extra of extras) {
        if (add(extra)) links.push(extra);
    }
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

/**
 * v1 marks the first row whose regex matches the request path, plus its
 * ancestors (`nav_tab`), so a dropdown parent lights up for its children too.
 */
export function findActiveKeys(links: NavLink[], pathname: string): Set<string> {
    const active = new Set<string>();

    const walk = (nodes: NavLink[], ancestors: string[]): boolean => {
        for (const node of nodes) {
            const trail = [...ancestors, node.key];
            if (isActiveLink(node, pathname)) {
                trail.forEach((key) => active.add(key));
                return true;
            }
            if (node.children?.length && walk(node.children, trail)) return true;
        }
        return false;
    };

    walk(links, []);
    return active;
}