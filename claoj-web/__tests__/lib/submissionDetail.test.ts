import { formatMemoryMb, summarizeTestScore } from '@/lib/submissionDetail';

describe('formatMemoryMb', () => {
    it('converts the API kilobyte value to megabytes', () => {
        expect(formatMemoryMb(3944)).toBe('3.9 MB');
    });

    it('keeps one decimal', () => {
        expect(formatMemoryMb(1024)).toBe('1.0 MB');
        expect(formatMemoryMb(0)).toBe('0.0 MB');
    });

    it('shows a dash when the value is missing', () => {
        expect(formatMemoryMb(null)).toBe('-');
        expect(formatMemoryMb(undefined)).toBe('-');
    });
});

describe('summarizeTestScore', () => {
    const tc = (status: string, points: number | null, total: number | null) => ({ status, points, total });

    it('counts passed cases and sums points', () => {
        expect(summarizeTestScore([
            tc('AC', 1, 1),
            tc('AC', 1, 1),
            tc('WA', 0, 1),
        ])).toEqual({ passed: 2, total: 3, points: 2, maxPoints: 3 });
    });

    it('keeps fractional partial scores', () => {
        expect(summarizeTestScore([tc('WA', 0.5, 1), tc('AC', 1, 1)]))
            .toEqual({ passed: 1, total: 2, points: 1.5, maxPoints: 2 });
    });

    it('returns null without test cases', () => {
        expect(summarizeTestScore([])).toBeNull();
    });

    it('returns null when a case has no score, so the caller keeps the old display', () => {
        expect(summarizeTestScore([tc('AC', 1, 1), tc('WA', null, 1)])).toBeNull();
        expect(summarizeTestScore([tc('AC', 1, null)])).toBeNull();
    });
});