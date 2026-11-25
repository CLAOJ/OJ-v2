import {
    buildNavLinks,
    findActiveKey,
    isActiveLink,
    normalizeNavPath,
    translateDbLabel,
    FALLBACK_NAV_LINKS,
    type NavigationItem,
} from '@/lib/navbar';

const item = (over: Partial<NavigationItem>): NavigationItem => ({
    key: 'k',
    label: 'K',
    path: '/k/',
    regex: '',
    ...over,
});

describe('normalizeNavPath', () => {
    it.each([
        ['/problems/', '/problems'],
        ['/submissions/', '/submissions'],
        ['/users/', '/users'],
        ['/organizations/', '/organizations'],
        ['/contests/', '/contests'],
        ['/status/', '/stats'],
        ['/', '/'],
        ['/about/', '/about'],
        ['/devlog/', '/devlog'],
        ['/wiki', '/wiki'],
        ['/some/unknown/page/', '/some/unknown/page'],
        ['', '/'],
    ])('%s -> %s', (input, expected) => {
        expect(normalizeNavPath(input)).toBe(expected);
    });

    it('leaves external links untouched', () => {
        expect(normalizeNavPath('https://status.claoj.edu.vn')).toBe('https://status.claoj.edu.vn');
        expect(normalizeNavPath('http://example.org/page/')).toBe('http://example.org/page/');
    });
});

describe('buildNavLinks', () => {
    it('keeps the DB order and labels and normalizes paths', () => {
        const links = buildNavLinks([
            item({ key: 'home', label: 'Home', path: '/', regex: '^/$' }),
            item({ key: 'problem', label: 'Problems', path: '/problems/', regex: '^/problems/' }),
            item({ key: 'judge', label: 'Judges', path: '/status/' }),
        ]);
        expect(links.map((l) => l.href)).toEqual(['/', '/problems', '/stats']);
        expect(links.map((l) => l.label)).toEqual(['Home', 'Problems', 'Judges']);
    });

    it('flattens children after their parent', () => {
        const links = buildNavLinks([
            item({
                key: 'contest',
                path: '/contests/',
                children: [item({ key: 'contest_tag', path: '/contests/tag/' })],
            }),
        ]);
        expect(links.map((l) => l.key)).toEqual(['contest', 'contest_tag']);
    });

    it('appends extras and drops duplicates by normalized path', () => {
        const links = buildNavLinks(
            [item({ key: 'problem', path: '/problems/' })],
            [
                { key: 'problems', href: '/problems', tKey: 'problems' },
                { key: 'ratings', href: '/ratings', tKey: 'ratings' },
            ],
        );
        expect(links.map((l) => l.href)).toEqual(['/problems', '/ratings']);
        expect(links[1].tKey).toBe('ratings');
    });

    it('falls back to the static links when the endpoint is unavailable', () => {
        expect(buildNavLinks(null)).toEqual(FALLBACK_NAV_LINKS);
        expect(buildNavLinks(undefined)).toEqual(FALLBACK_NAV_LINKS);
        expect(buildNavLinks([]).map((l) => l.href)).toEqual([
            '/problems',
            '/contests',
            '/submissions',
            '/users',
            '/ratings',
            '/organizations',
            '/stats',
        ]);
    });

    it('still appends v2-only entries on the fallback path', () => {
        const links = buildNavLinks(null, [{ key: 'report', href: '/ticket/create', tKey: 'report' }]);
        expect(links.at(-1)?.href).toBe('/ticket/create');
        // Ratings is already in the fallback, so it is not duplicated.
        expect(links.filter((l) => l.href === '/ratings')).toHaveLength(1);
    });
});

describe('translateDbLabel', () => {
    // v1 renders the label through gettext: known labels follow the catalog,
    // arbitrary admin labels stay verbatim.
    const viLabels = { Problems: 'Bài tập', Organizations: 'Tổ chức' };

    it('translates a known DB label', () => {
        expect(translateDbLabel('Problems', viLabels)).toBe('Bài tập');
        expect(translateDbLabel('Organizations', viLabels)).toBe('Tổ chức');
    });

    it('renders an unknown DB label verbatim', () => {
        expect(translateDbLabel('Wiki', viLabels)).toBe('Wiki');
        expect(translateDbLabel('devlog', viLabels)).toBe('devlog');
        expect(translateDbLabel('Server Status', viLabels)).toBe('Server Status');
        expect(translateDbLabel('Problems', undefined)).toBe('Problems');
    });

    it('leaves English unchanged (identity catalog)', () => {
        const enLabels = { Problems: 'Problems', Organizations: 'Organizations' };
        expect(translateDbLabel('Problems', enLabels)).toBe('Problems');
        expect(translateDbLabel('Server Status', enLabels)).toBe('Server Status');
    });

    it('is applied to DB rows, never to v2-only entries', () => {
        const links = buildNavLinks(
            [item({ key: 'problem', label: 'Problems', path: '/problems/' })],
            [{ key: 'ratings', href: '/ratings', tKey: 'ratings' }],
            viLabels,
        );
        expect(links[0].label).toBe('Bài tập');
        expect(links[1].label).toBeUndefined();
        expect(links[1].tKey).toBe('ratings');
    });
});

describe('active item', () => {
    it('uses the regex, tolerating the missing v2 trailing slash', () => {
        const link = { key: 'problem', href: '/problems', regex: '^/problems/' };
        expect(isActiveLink(link, '/problems')).toBe(true);
        expect(isActiveLink(link, '/problems/123')).toBe(true);
        expect(isActiveLink(link, '/users')).toBe(false);
    });

    it('treats a compiling regex as authoritative', () => {
        expect(isActiveLink({ key: 'p', href: '/problems', regex: '^/contest/' }, '/problems')).toBe(false);
    });

    it('falls back to the path prefix when the regex cannot compile', () => {
        const link = { key: 'wiki', href: '/wiki', regex: '(?x)^/wiki/' };
        expect(isActiveLink(link, '/wiki')).toBe(true);
        expect(isActiveLink(link, '/wiki/page')).toBe(true);
    });

    it('matches the path prefix without a regex, and "/" only exactly', () => {
        expect(isActiveLink({ key: 'u', href: '/users' }, '/users/abc')).toBe(true);
        expect(isActiveLink({ key: 'home', href: '/' }, '/problems')).toBe(false);
        expect(isActiveLink({ key: 'home', href: '/' }, '/')).toBe(true);
    });

    it('never activates an external link', () => {
        expect(isActiveLink({ key: 's', href: 'https://status.claoj.edu.vn' }, '/anything')).toBe(false);
    });

    it('marks only the first matching row, like v1', () => {
        const links = [
            { key: 'all', href: '/all', regex: '^/' },
            { key: 'problem', href: '/problems', regex: '^/problems/' },
        ];
        expect(findActiveKey(links, '/problems')).toBe('all');
        expect(findActiveKey(links, '/elsewhere')).toBe('all');
        expect(findActiveKey([], '/problems')).toBeNull();
    });
});