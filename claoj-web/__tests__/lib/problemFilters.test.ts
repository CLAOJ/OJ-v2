import { buildProblemListParams } from '@/lib/problemFilters';

describe('buildProblemListParams', () => {
    it('omits empty filters and keeps paging and sorting', () => {
        const params = buildProblemListParams({
            search: '',
            status: '',
            group: '',
            types: [],
            hasEditorial: false,
            sort: 'code',
            order: 'asc',
            page: 2,
            pageSize: 20,
        });
        expect(params.toString()).toBe('sort=code&order=asc&page=2&page_size=20');
    });

    it('joins type ids and sends the editorial flag as 1', () => {
        const params = buildProblemListParams({
            group: '7',
            types: [3, 5],
            hasEditorial: true,
        });
        expect(params.get('group')).toBe('7');
        expect(params.get('type')).toBe('3,5');
        expect(params.get('has_editorial')).toBe('1');
    });

    it('sends numeric point bounds only when present', () => {
        expect(buildProblemListParams({ pointsMin: 0 }).get('points_min')).toBe('0');
        expect(buildProblemListParams({ pointsMax: null }).has('points_max')).toBe(false);
        expect(buildProblemListParams({ pointsMax: 40 }).get('points_max')).toBe('40');
    });
});
