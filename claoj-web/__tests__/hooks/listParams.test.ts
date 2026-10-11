import { readListParams, buildListQuery } from '@/hooks/useListParams';

describe('list params', () => {
    it('reads page and only the named filters', () => {
        const sp = new URLSearchParams('page=3&search=dp&junk=1&status=');
        expect(readListParams(sp, ['search', 'status'])).toEqual({ page: 3, filters: { search: 'dp' } });
    });

    it('clamps a bad page to 1', () => {
        expect(readListParams(new URLSearchParams('page=-2'), []).page).toBe(1);
        expect(readListParams(new URLSearchParams('page=abc'), []).page).toBe(1);
    });

    it('builds a query omitting empty filters and page 1', () => {
        expect(buildListQuery(1, { search: 'a b', status: '' })).toBe('?search=a+b');
        expect(buildListQuery(2, {})).toBe('?page=2');
        expect(buildListQuery(1, {})).toBe('');
    });
});
