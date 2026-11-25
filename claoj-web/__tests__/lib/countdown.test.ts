import { formatCountdown } from '@/lib/countdown';

describe('formatCountdown', () => {
    it('shows HH:MM:SS under one day', () => {
        expect(formatCountdown(3 * 3600_000 + 4 * 60_000 + 5_000)).toBe('03:04:05');
        expect(formatCountdown(45_000)).toBe('00:00:45');
    });

    it('rolls hours into days instead of printing a huge hour count', () => {
        expect(formatCountdown(890 * 3600_000 + 49 * 60_000 + 45_000)).toBe('37d 02:49:45');
        expect(formatCountdown(86_400_000)).toBe('1d 00:00:00');
    });

    it('clamps past deadlines to zero', () => {
        expect(formatCountdown(0)).toBe('00:00:00');
        expect(formatCountdown(-5_000)).toBe('00:00:00');
    });

    it('falls back to zero when the duration is not a finite number', () => {
        expect(formatCountdown(Number.NaN)).toBe('00:00:00');
        expect(formatCountdown(Number.POSITIVE_INFINITY)).toBe('00:00:00');
    });
});