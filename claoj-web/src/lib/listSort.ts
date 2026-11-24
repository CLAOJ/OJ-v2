export type SortOrder = 'asc' | 'desc';

/** Sort keys the admin problem list accepts (see service/problem/problem_service.go). */
export type ProblemSort = 'code' | 'name' | 'points' | 'date' | 'ac_rate' | 'user_count';
/** Sort keys the admin contest list accepts (see service/contest/contest_service.go). */
export type ContestSort = 'name' | 'start_time' | 'end_time' | 'user_count';

/** Sort keys the admin user list accepts (see service/user/user_service.go). */
export type UserSort = 'username' | 'date_joined' | 'last_login' | 'points';

export interface SortState<T extends string = string> {
    sort: T;
    order: SortOrder;
}

/**
 * Pure: state after clicking a column header. Clicking the active column flips
 * the order; clicking another column starts it at `firstOrder` (text columns
 * ascending, numeric/date columns descending).
 */
export function toggleSort<T extends string>(
    current: SortState<T>,
    column: T,
    firstOrder: SortOrder
): SortState<T> {
    if (current.sort !== column) return { sort: column, order: firstOrder };
    return { sort: column, order: current.order === 'asc' ? 'desc' : 'asc' };
}

/** Years for the problem year filter: `latest` down to 2021 (v1's range). */
export function yearOptions(latest: number, oldest = 2021): number[] {
    const years: number[] = [];
    for (let year = latest; year >= oldest; year--) years.push(year);
    return years;
}