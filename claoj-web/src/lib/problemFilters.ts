export interface ProblemListQuery {
    search?: string;
    /** 'solved' | 'unsolved' | '' (any). */
    status?: string;
    /** Problem group id; '' for any. */
    group?: string;
    /** Problem type ids; a problem matches any of them. */
    types?: number[];
    /** true = only problems with a public editorial, false = only without. */
    hasEditorial?: boolean;
    pointsMin?: number;
    /** null/undefined = no upper bound. */
    pointsMax?: number | null;
    sort?: string;
    order?: 'asc' | 'desc';
    page?: number;
    pageSize?: number;
}

/**
 * Builds the query string for GET /problems. Empty filters are omitted so the
 * API's own defaults apply; `has_editorial` is only sent when the box is
 * ticked (the API treats "0" as "must have none", not as "any").
 */
export function buildProblemListParams(q: ProblemListQuery): URLSearchParams {
    const params = new URLSearchParams();
    if (q.search) params.set('search', q.search);
    if (q.status) params.set('status', q.status);
    if (q.group) params.set('group', q.group);
    if (q.types && q.types.length > 0) params.set('type', q.types.join(','));
    if (q.hasEditorial) params.set('has_editorial', '1');
    if (q.pointsMin !== undefined) params.set('points_min', String(q.pointsMin));
    if (q.pointsMax !== undefined && q.pointsMax !== null) params.set('points_max', String(q.pointsMax));
    if (q.sort) params.set('sort', q.sort);
    if (q.order) params.set('order', q.order);
    if (q.page !== undefined) params.set('page', String(q.page));
    if (q.pageSize !== undefined) params.set('page_size', String(q.pageSize));
    return params;
}
