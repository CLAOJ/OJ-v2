import { formatLimitSeconds, formatLimitMemory } from '@/lib/utils';

// The admin Language Limits table crashed on a row whose numeric fields were
// missing (the API used to send Go field names, so time_limit was undefined and
// `.toFixed` threw). These helpers must return null rather than throw.
describe('language limit formatting', () => {
    it('formats seconds with two decimals', () => {
        expect(formatLimitSeconds(1.5)).toBe('1.50s');
        expect(formatLimitSeconds(0.001)).toBe('0.00s');
    });

    it('formats memory from KB to whole MB', () => {
        expect(formatLimitMemory(262144)).toBe('256 MB');
        expect(formatLimitMemory(1536)).toBe('2 MB');
    });

    it('returns null for missing or non-numeric values', () => {
        expect(formatLimitSeconds(undefined)).toBeNull();
        expect(formatLimitSeconds(null)).toBeNull();
        expect(formatLimitSeconds(Number.NaN)).toBeNull();
        expect(formatLimitMemory(undefined)).toBeNull();
        expect(formatLimitMemory(Number.POSITIVE_INFINITY)).toBeNull();
    });
});