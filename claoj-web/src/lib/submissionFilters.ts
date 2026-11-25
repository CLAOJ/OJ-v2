export interface SubmissionListQuery {
    user?: string;
    problem?: string;
    /** Verdict codes; the API takes them as one comma-separated value. */
    results?: string[];
    /** Language keys, same encoding as `results`. */
    languages?: string[];
    /** Organization id; only that organization's members' submissions. */
    organization?: string;
    /** Contest key; only that contest's submissions (v1 in-contest lists). */
    contest?: string;
    /** One best submission per user (highest points, then fastest). */
    best?: boolean;
    page?: number;
    pageSize?: number;
}

/**
 * Query string for GET /submissions. Empty filters are omitted so the API's
 * defaults apply; multi-value filters are joined with commas, which the API
 * accepts alongside repeated params.
 */
export function buildSubmissionListParams(q: SubmissionListQuery): URLSearchParams {
    const params = new URLSearchParams();
    if (q.user) params.set('user', q.user);
    if (q.problem) params.set('problem', q.problem);
    if (q.results && q.results.length > 0) params.set('result', q.results.join(','));
    if (q.languages && q.languages.length > 0) params.set('language', q.languages.join(','));
    if (q.organization) params.set('organization', q.organization);
    if (q.contest) params.set('contest', q.contest);
    if (q.best) params.set('best', '1');
    if (q.page !== undefined) params.set('page', String(q.page));
    if (q.pageSize !== undefined) params.set('page_size', String(q.pageSize));
    return params;
}