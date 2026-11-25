import { buildHeatmapWeeks, weekdayOf } from '@/lib/heatmap';

function denseSeries(start: string, length: number, counts: Record<string, number> = {}) {
    const days = [];
    const date = new Date(`${start}T00:00:00Z`);
    for (let i = 0; i < length; i++) {
        const key = date.toISOString().slice(0, 10);
        days.push({ date: key, count: counts[key] ?? 0 });
        date.setUTCDate(date.getUTCDate() + 1);
    }
    return days;
}

describe('weekdayOf', () => {
    it('is Sunday-first and timezone-safe', () => {
        expect(weekdayOf('2026-11-22')).toBe(0); // Sunday
        expect(weekdayOf('2026-11-25')).toBe(3); // Wednesday
        expect(weekdayOf('2026-11-28')).toBe(6); // Saturday
    });
});

describe('buildHeatmapWeeks', () => {
    it('returns one column per week with the first week padded to Sunday', () => {
        const weeks = buildHeatmapWeeks(denseSeries('2026-11-25', 10));
        expect(weeks).toHaveLength(2);
        expect(weeks[0]).toHaveLength(7);
        // 2026-11-25 is a Wednesday: Sun/Mon/Tue are empty.
        expect(weeks[0].slice(0, 3)).toEqual([null, null, null]);
        expect(weeks[0][3]?.date).toBe('2026-11-25');
        expect(weeks[1].every(cell => cell === null || typeof cell.level === 'number')).toBe(true);
    });

    it('pads the trailing week to seven cells', () => {
        const weeks = buildHeatmapWeeks(denseSeries('2026-11-22', 7)); // Sun..Sat
        expect(weeks).toHaveLength(1);
        expect(weeks[0].every(cell => cell !== null)).toBe(true);
    });

    it('maps counts onto five levels, zero always level 0', () => {
        const days = denseSeries('2026-11-22', 3, {
            '2026-11-22': 0,
            '2026-11-23': 1,
            '2026-11-24': 10,
        });
        const cells = buildHeatmapWeeks(days)[0];
        expect(cells[0]?.level).toBe(0);
        expect(cells[1]?.level).toBe(1);
        expect(cells[2]?.level).toBe(4);
    });

    it('handles an empty series', () => {
        expect(buildHeatmapWeeks([])).toEqual([]);
    });

    it('lays out a full year like v1 (52 or 53 week columns)', () => {
        const weeks = buildHeatmapWeeks(denseSeries('2025-11-26', 365));
        expect([52, 53]).toContain(weeks.length);
        expect(weeks.flat().filter(Boolean)).toHaveLength(365);
    });
});