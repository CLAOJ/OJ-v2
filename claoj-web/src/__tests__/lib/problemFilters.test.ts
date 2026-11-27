import { buildProblemListParams, defaultProblemListParams, DEFAULT_PROBLEM_PAGE_SIZE } from '@/lib/problemFilters';

// ProblemsPageContent only uses the server-rendered list when its initial
// filter state serialises to exactly defaultProblemListParams(); if they drift
// the SSR data is silently ignored and the page is slow again.
test('client initial filter state matches the server-rendered default view', () => {
    const clientInitial = buildProblemListParams({
        search: '', status: '', group: '', types: [], hasEditorial: false,
        pointsMin: 0, pointsMax: null, sort: 'date', order: 'desc',
        page: 1, pageSize: DEFAULT_PROBLEM_PAGE_SIZE,
    });
    expect(clientInitial.toString()).toBe(defaultProblemListParams().toString());
    expect(defaultProblemListParams().toString()).toBe('points_min=0&sort=date&order=desc&page=1&page_size=50');
});
