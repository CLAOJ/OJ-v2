import { buildSubmissionListParams, addFilterValue, removeFilterValue } from '@/lib/submissionFilters';

describe('buildSubmissionListParams', () => {
    it('omits empty filters and keeps paging', () => {
        const params = buildSubmissionListParams({
            user: '',
            problem: '',
            results: [],
            languages: [],
            organization: '',
            page: 2,
            pageSize: 50,
        });
        expect(params.toString()).toBe('page=2&page_size=50');
    });

    it('joins multi-value verdict and language filters with commas', () => {
        const params = buildSubmissionListParams({
            results: ['AC', 'WA'],
            languages: ['CPP17', 'PY3'],
        });
        expect(params.get('result')).toBe('AC,WA');
        expect(params.get('language')).toBe('CPP17,PY3');
    });

    it('sends best only when the mode is on', () => {
        expect(buildSubmissionListParams({ problem: 'P01', best: true }).get('best')).toBe('1');
        expect(buildSubmissionListParams({ problem: 'P01', best: false }).has('best')).toBe(false);
        expect(buildSubmissionListParams({ problem: 'P01' }).has('best')).toBe(false);
    });

    it('sends the contest key only when given', () => {
        expect(buildSubmissionListParams({ contest: 'cplusplusnc' }).get('contest')).toBe('cplusplusnc');
        expect(buildSubmissionListParams({}).has('contest')).toBe(false);
    });

    it('passes the user, problem and organization filters through', () => {
        const params = buildSubmissionListParams({
            user: 'alice',
            problem: 'P01',
            organization: '7',
            results: ['AC'],
        });
        expect(params.get('user')).toBe('alice');
        expect(params.get('problem')).toBe('P01');
        expect(params.get('organization')).toBe('7');
        expect(params.get('result')).toBe('AC');
    });
});

describe('multi-value filter tags', () => {
    it('adds without duplicating and removes by value', () => {
        expect(addFilterValue([], 'WA')).toEqual(['WA']);
        expect(addFilterValue(['WA'], 'WA')).toEqual(['WA']);
        expect(addFilterValue(['WA'], 'TLE')).toEqual(['WA', 'TLE']);
        expect(removeFilterValue(['WA', 'TLE'], 'WA')).toEqual(['TLE']);
        expect(removeFilterValue(['WA'], 'AC')).toEqual(['WA']);
    });
});