import { buildRedirectMap, lookupRedirect, pickRedirectTarget } from '@/lib/redirects';
import { normalizeFlatPagePath } from '@/lib/flatpage';

describe('buildRedirectMap', () => {
    it('keeps valid pairs and skips malformed ones', () => {
        const map = buildRedirectMap([
            { old_path: '/old/', new_path: '/new/' },
            { old_path: '/broken/' } as never,
            null as never,
        ]);
        expect(map.size).toBe(1);
        expect(map.get('/old/')).toBe('/new/');
    });

    it('collapses a chain to its final target', () => {
        const map = buildRedirectMap([
            { old_path: '/a/', new_path: '/b/' },
            { old_path: '/b/', new_path: '/c/' },
        ]);
        expect(map.get('/a/')).toBe('/c/');
        expect(map.get('/b/')).toBe('/c/');
    });

    it('drops entries whose chain loops', () => {
        const map = buildRedirectMap([
            { old_path: '/a/', new_path: '/b/' },
            { old_path: '/b/', new_path: '/a/' },
        ]);
        expect(map.size).toBe(0);
    });
});

describe('lookupRedirect', () => {
    const map = buildRedirectMap([{ old_path: '/old/', new_path: '/new/' }]);

    it('matches the exact stored path', () => {
        expect(lookupRedirect(map, '/old/')).toBe('/new/');
    });

    it('appends a missing trailing slash like Django APPEND_SLASH', () => {
        expect(lookupRedirect(map, '/old')).toBe('/new/');
    });

    it('returns null when nothing matches', () => {
        expect(lookupRedirect(map, '/old/page/')).toBeNull();
    });
});

describe('pickRedirectTarget', () => {
    it('returns the target for a normal redirect', () => {
        const map = buildRedirectMap([{ old_path: '/a/', new_path: '/b/' }]);
        expect(pickRedirectTarget(map, '/a/')).toBe('/b/');
    });

    it('refuses a self redirect', () => {
        const map = buildRedirectMap([{ old_path: '/a/', new_path: '/a/' }]);
        expect(pickRedirectTarget(map, '/a/')).toBeNull();
    });

    it('resolves a chain to its final target in one hop', () => {
        const map = buildRedirectMap([
            { old_path: '/a/', new_path: '/b/' },
            { old_path: '/b/', new_path: '/c/' },
        ]);
        expect(pickRedirectTarget(map, '/a/')).toBe('/c/');
        expect(pickRedirectTarget(map, '/b/')).toBe('/c/');
    });

    it('refuses a two-entry cycle', () => {
        const map = buildRedirectMap([
            { old_path: '/a/', new_path: '/b' },
            { old_path: '/b/', new_path: '/a/' },
        ]);
        // "/b" is not a key, but its append-slash form "/b/" is -> cycle.
        expect(pickRedirectTarget(map, '/a/')).toBeNull();
        expect(pickRedirectTarget(map, '/b/')).toBeNull();
    });

    it('allows an absolute http(s) target untouched', () => {
        const map = buildRedirectMap([{ old_path: '/ext/', new_path: 'https://example.com/x' }]);
        expect(pickRedirectTarget(map, '/ext/')).toBe('https://example.com/x');
    });
});

describe('normalizeFlatPagePath', () => {
    it('joins catch-all segments with leading and trailing slashes', () => {
        expect(normalizeFlatPagePath(['about'])).toBe('/about/');
    });

    it('adds a trailing slash and collapses duplicate slashes', () => {
        expect(normalizeFlatPagePath(['docs', 'intro'])).toBe('/docs/intro/');
        expect(normalizeFlatPagePath('/a//b')).toBe('/a/b/');
    });

    it('leaves an already-normalized path alone', () => {
        expect(normalizeFlatPagePath('/about/')).toBe('/about/');
    });
});
