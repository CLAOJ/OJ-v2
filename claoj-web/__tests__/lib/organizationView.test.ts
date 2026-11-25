import { DEFAULT_MEMBER_SORT, DEFAULT_PROBLEM_SORT, formatPoints, parseOrgRequestTab, parseOrgTab } from '@/lib/organizationView';

describe('parseOrgTab', () => {
    it('accepts every organization tab', () => {
        for (const tab of ['members', 'admins', 'problems', 'contests', 'submissions'] as const) {
            expect(parseOrgTab(tab)).toBe(tab);
        }
    });

    it('falls back to the members tab for anything else', () => {
        expect(parseOrgTab(null)).toBe('members');
        expect(parseOrgTab(undefined)).toBe('members');
        expect(parseOrgTab('')).toBe('members');
        expect(parseOrgTab('nope')).toBe('members');
    });
});

describe('parseOrgRequestTab', () => {
    it('accepts the request states and the log tab', () => {
        for (const tab of ['P', 'A', 'R', 'log'] as const) {
            expect(parseOrgRequestTab(tab)).toBe(tab);
        }
    });

    it('falls back to pending (v1 OrganizationRequestView)', () => {
        expect(parseOrgRequestTab(null)).toBe('P');
        expect(parseOrgRequestTab(undefined)).toBe('P');
        expect(parseOrgRequestTab('')).toBe('P');
        expect(parseOrgRequestTab('all')).toBe('P');
    });
});

describe('DEFAULT_MEMBER_SORT', () => {
    it('matches v1 OrganizationUsers (-performance_points)', () => {
        expect(DEFAULT_MEMBER_SORT).toEqual({ sort: 'performance_points', order: 'desc' });
    });
});

describe('DEFAULT_PROBLEM_SORT', () => {
    it('matches v1 ProblemListOrganization (-date)', () => {
        expect(DEFAULT_PROBLEM_SORT).toEqual({ sort: 'date', order: 'desc' });
    });
});

describe('formatPoints', () => {
    it('always shows two decimals like v1 floatformat(2)', () => {
        expect(formatPoints(12.345)).toBe('12.35');
        expect(formatPoints(40)).toBe('40.00');
        expect(formatPoints(0)).toBe('0.00');
    });

    it('treats a missing score as zero', () => {
        expect(formatPoints(null)).toBe('0.00');
        expect(formatPoints(undefined)).toBe('0.00');
        expect(formatPoints(NaN)).toBe('0.00');
    });
});