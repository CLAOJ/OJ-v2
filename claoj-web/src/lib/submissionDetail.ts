// Display helpers for the submission detail page.

// The API reports submission/test-case memory in kilobytes (v1 parity); the
// page shows megabytes.
export function formatMemoryMb(kb: number | null | undefined): string {
    if (kb === null || kb === undefined) return '-';
    return `${(kb / 1024).toFixed(1)} MB`;
}

interface TestCaseScore {
    status: string;
    points: number | null;
    total: number | null;
}

export interface TestScore {
    passed: number;
    total: number;
    points: number;
    maxPoints: number;
}

// v1 header score is `20/20 (1.000/1.000 points)`: test cases passed out of
// total, plus points earned out of the maximum. Both come from the per-testcase
// rows the page already fetched; null when a row lacks a score so the caller can
// fall back to the submission-level display instead of inventing numbers.
export function summarizeTestScore(testCases: TestCaseScore[]): TestScore | null {
    if (testCases.length === 0) return null;
    if (testCases.some(tc => tc.points === null || tc.total === null)) return null;

    return {
        passed: testCases.filter(tc => tc.status === 'AC').length,
        total: testCases.length,
        points: testCases.reduce((sum, tc) => sum + (tc.points ?? 0), 0),
        maxPoints: testCases.reduce((sum, tc) => sum + (tc.total ?? 0), 0),
    };
}