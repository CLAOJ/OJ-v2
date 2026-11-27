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

/** The problems page's first view; the server renders this list into the HTML. */
export const DEFAULT_PROBLEM_PAGE_SIZE = 50;
export const TRENDING_LIMIT = 5;

export function defaultProblemListParams(): URLSearchParams {
    return buildProblemListParams({ pointsMin: 0, sort: 'date', order: 'desc', page: 1, pageSize: DEFAULT_PROBLEM_PAGE_SIZE });
}

/** Adds a problem type to the filter list, ignoring duplicates. */
export function addProblemType(ids: number[], id: number): number[] {
    return ids.includes(id) ? ids : [...ids, id];
}

/** Removes a problem type from the filter list. */
export function removeProblemType(ids: number[], id: number): number[] {
    return ids.filter(x => x !== id);
}
