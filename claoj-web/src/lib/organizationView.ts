/**
 * Pure helpers for the public organization pages: tab parsing and the member
 * table's stats formatting. Kept out of the components so they can be tested.
 */

export type OrgTab = 'members' | 'admins' | 'problems' | 'contests' | 'submissions';

const ORG_TABS: OrgTab[] = ['members', 'admins', 'problems', 'contests', 'submissions'];

/** `?tab=` value → a known tab; anything else (or nothing) is the members tab. */
export function parseOrgTab(value: string | null | undefined): OrgTab {
    return ORG_TABS.includes(value as OrgTab) ? (value as OrgTab) : 'members';
}

/** Sort keys the organization member table accepts (v1 OrganizationUsers). */
export type OrgMemberSort = 'performance_points' | 'points' | 'problem_count' | 'rating';

/** Sort keys the organization problems tab accepts (v1 ProblemListOrganization). */
export type OrgProblemSort = 'code' | 'name' | 'points' | 'date' | 'user_count';

/** v1 sorts the member table by -performance_points and every column desc-first. */
export const DEFAULT_MEMBER_SORT: { sort: OrgMemberSort; order: 'desc' } = {
    sort: 'performance_points',
    order: 'desc',
};

/** v1's default problem order is -date (newest first). */
export const DEFAULT_PROBLEM_SORT: { sort: OrgProblemSort; order: 'desc' } = {
    sort: 'date',
    order: 'desc',
};

/** Points as v1's `floatformat(2)` renders them: always two decimals. */
export function formatPoints(points: number | null | undefined): string {
    const value = Number(points);
    return Number.isFinite(value) ? value.toFixed(2) : '0.00';
}