/**
 * Pure helpers for the public contest pages: times, countdowns, tab parsing and
 * scoreboard CSV. Kept out of the components so they can be unit-tested.
 */

export type ContestTab = 'dashboard' | 'problems' | 'scoreboard' | 'participation';

const CONTEST_TABS: ContestTab[] = ['dashboard', 'problems', 'scoreboard', 'participation'];

/** `?tab=` value → a known tab; anything else (or nothing) is the dashboard. */
export function parseContestTab(value: string | null | undefined): ContestTab {
    return CONTEST_TABS.includes(value as ContestTab) ? (value as ContestTab) : 'dashboard';
}

/**
 * Seconds → zero-padded `HH:MM:SS`, hours accumulating days like v1's
 * `nice_repr(..., 'noday')` (25h reads 25:00:00). Negative/absent → 00:00:00.
 */
export function formatClock(seconds: number | null | undefined): string {
    const total = Math.max(0, Math.floor(Number(seconds) || 0));
    const h = Math.floor(total / 3600);
    const m = Math.floor((total % 3600) / 60);
    const s = total % 60;
    return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
}

/** Remaining milliseconds → `[Nd ]Hh Mm Ss`, rounded up so it never shows 0s early. */
export function formatCountdown(ms: number): string {
    const total = Math.max(0, Math.ceil(ms / 1000));
    const d = Math.floor(total / 86400);
    const h = Math.floor((total % 86400) / 3600);
    const m = Math.floor((total % 3600) / 60);
    const s = total % 60;
    return d > 0 ? `${d}d ${h}h ${m}m ${s}s` : `${h}h ${m}m ${s}s`;
}

/**
 * `contest.time_limit` is a Django DurationField: microseconds. Whole seconds,
 * or 0 when unset/zero (no per-user window limit).
 */
export function contestTimeLimitSeconds(micros: number | null | undefined): number {
    const us = Number(micros) || 0;
    return us > 0 ? Math.round(us / 1_000_000) : 0;
}

/** Full-score cells per problem column (v1's Total AC row). */
export function countTotalAc(
    rows: { breakdown?: ({ points?: number } | null)[] }[],
    problems: { points: number }[]
): number[] {
    return problems.map((p, i) => rows.filter(r => (r.breakdown?.[i]?.points ?? 0) >= p.points && p.points > 0).length);
}

export interface ScoreboardCsvInput {
    problems: { label: string; points: number }[];
    rows: {
        rank: number;
        username: string;
        display_name?: string;
        organization?: { short_name?: string; name?: string } | null;
        rating?: number | null;
        score: number;
        cumtime: number;
        breakdown?: ({ points?: number; time?: number } | null)[];
    }[];
    includeName: boolean;
    includeOrg: boolean;
}

/**
 * Quoted RFC-4180 cell. Excel/LibreOffice still evaluate a quoted cell that
 * starts with =, +, - or @ (or a tab), so string cells with those prefixes get
 * a leading apostrophe; usernames and display names are user-controlled.
 */
function csvCell(value: string | number): string {
    let text = String(value).replace(/[\r\n]+/g, ' ');
    if (typeof value === 'string' && /^[=+\-@\t]/.test(text)) {
        text = `'${text}`;
    }
    return `"${text.replace(/"/g, '""')}"`;
}

/** The scoreboard's current view (same toggles), as RFC-4180 CSV text. */
export function buildScoreboardCsv(input: ScoreboardCsvInput): string {
    const header = ['Rank', 'Username'];
    if (input.includeName) header.push('Name');
    if (input.includeOrg) header.push('Organization');
    header.push('Rating', 'Total', ...input.problems.map(p => p.label), 'Window');

    const lines = [header.map(csvCell).join(',')];
    for (const row of input.rows) {
        const cells: (string | number)[] = [row.rank, row.username];
        if (input.includeName) cells.push(row.display_name || row.username);
        if (input.includeOrg) cells.push(row.organization?.short_name || row.organization?.name || '');
        cells.push(row.rating ?? '');
        cells.push(row.score);
        input.problems.forEach((_, i) => {
            const cell = row.breakdown?.[i];
            if (!cell) {
                cells.push('');
                return;
            }
            cells.push(cell.time != null ? `${cell.points ?? 0} ${formatClock(cell.time)}` : String(cell.points ?? 0));
        });
        cells.push(formatClock(row.cumtime));
        lines.push(cells.map(csvCell).join(','));
    }
    return lines.join('\n');
}