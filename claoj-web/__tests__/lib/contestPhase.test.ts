import { contestPhase } from '@/lib/contestPhase';

const now = Date.parse('2026-11-24T12:00:00Z');

describe('contestPhase', () => {
    it('is upcoming before the start', () => {
        expect(contestPhase('2026-11-25T00:00:00Z', '2026-11-26T00:00:00Z', now)).toBe('upcoming');
    });

    it('is ongoing between start and end', () => {
        expect(contestPhase('2026-11-23T00:00:00Z', '2026-11-25T00:00:00Z', now)).toBe('ongoing');
    });

    it('is past after the end', () => {
        expect(contestPhase('2026-11-01T00:00:00Z', '2026-11-02T00:00:00Z', now)).toBe('past');
    });

    it('falls back to ongoing when a time is missing or invalid', () => {
        expect(contestPhase(null, null, now)).toBe('ongoing');
        expect(contestPhase('not-a-date', 'not-a-date', now)).toBe('ongoing');
        expect(contestPhase('2026-11-01T00:00:00Z', null, now)).toBe('ongoing');
    });
});