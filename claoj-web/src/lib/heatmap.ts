/**
 * Pure helpers for the profile contribution heatmap (v1's user-about grid):
 * one cell per day, Sunday-first week columns, five intensity levels.
 * Kept out of the component so it can be tested.
 */

export interface HeatmapDay {
    /** `YYYY-MM-DD`, as returned by the analytics endpoint. */
    date: string;
    count: number;
}

export interface HeatmapCell extends HeatmapDay {
    /** 0 (no activity) to `levels - 1` (busiest day). */
    level: number;
}

/** Seven cells, Sunday first; null pads the first and last week. */
export type HeatmapWeek = (HeatmapCell | null)[];

/** Weekday of a `YYYY-MM-DD` date, 0 = Sunday. UTC keeps the date intact. */
export function weekdayOf(date: string): number {
    return new Date(`${date}T00:00:00Z`).getUTCDay();
}

/**
 * Group a dense, oldest-first day series into week columns. Levels follow v1:
 * `ceil(count / max * (levels - 1))`, with zero activity always level 0.
 *
 * shortcut: assumes the series is dense and contiguous (one entry per day,
 * oldest first, no gaps), which is what /analytics activity_year returns — a
 * missing day would shift every later cell. Upgrade to date-keyed padding if
 * the API ever returns sparse days.
 */
export function buildHeatmapWeeks(days: HeatmapDay[], levels = 5): HeatmapWeek[] {
    if (days.length === 0) return [];
    const max = Math.max(1, ...days.map(d => d.count));
    const weeks: HeatmapWeek[] = [];
    let week: HeatmapWeek = new Array(weekdayOf(days[0].date)).fill(null);

    for (const day of days) {
        const level = day.count === 0 ? 0 : Math.max(1, Math.ceil((day.count / max) * (levels - 1)));
        week.push({ ...day, level });
        if (week.length === 7) {
            weeks.push(week);
            week = [];
        }
    }
    if (week.length > 0) {
        while (week.length < 7) week.push(null);
        weeks.push(week);
    }
    return weeks;
}