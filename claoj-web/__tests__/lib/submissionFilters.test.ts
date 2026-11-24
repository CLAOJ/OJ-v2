import { buildSubmissionListParams } from '@/lib/submissionFilters';

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