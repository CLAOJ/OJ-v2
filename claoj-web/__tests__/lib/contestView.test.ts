import {
    buildScoreboardCsv,
    contestTagPath,
    contestTimeLimitSeconds,
    countTotalAc,
    formatClock,
    formatCountdown,
    parseCalendarMonth,
    parseContestTab,
} from '@/lib/contestView';

describe('contestTagPath', () => {
    it('builds the v1 tag URL and escapes the name', () => {
        expect(contestTagPath('open-cup')).toBe('/contests/tag/open-cup');
        expect(contestTagPath('a b')).toBe('/contests/tag/a%20b');
    });
});

describe('parseCalendarMonth', () => {
    const today = { year: 2026, month: 11 };

    it('reads a valid year and month from the URL', () => {
        expect(parseCalendarMonth('2024', '2', today)).toEqual({ year: 2024, month: 2 });
    });

    it('falls back to today for missing or invalid values', () => {
        expect(parseCalendarMonth(null, null, today)).toEqual(today);
        expect(parseCalendarMonth('', '', today)).toEqual(today);
        expect(parseCalendarMonth('2024', '13', today)).toEqual({ year: 2024, month: 11 });
        expect(parseCalendarMonth('2024', '0', today)).toEqual({ year: 2024, month: 11 });
        expect(parseCalendarMonth('abc', '3', today)).toEqual({ year: 2026, month: 3 });
        expect(parseCalendarMonth('2024.5', '3', today)).toEqual({ year: 2026, month: 3 });
    });
});

describe('parseContestTab', () => {
    it('accepts known tabs', () => {
        expect(parseContestTab('problems')).toBe('problems');
        expect(parseContestTab('scoreboard')).toBe('scoreboard');
        expect(parseContestTab('participation')).toBe('participation');
        expect(parseContestTab('dashboard')).toBe('dashboard');
    });

    it('falls back to the dashboard', () => {
        expect(parseContestTab(null)).toBe('dashboard');
        expect(parseContestTab('nope')).toBe('dashboard');
        expect(parseContestTab('')).toBe('dashboard');
    });
});

describe('formatClock', () => {
    it('formats seconds as HH:MM:SS', () => {
        expect(formatClock(0)).toBe('00:00:00');
        expect(formatClock(59)).toBe('00:00:59');
        expect(formatClock(61)).toBe('00:01:01');
        expect(formatClock(3661)).toBe('01:01:01');
    });

    it('accumulates days into hours (v1 noday) and tolerates junk', () => {
        expect(formatClock(25 * 3600)).toBe('25:00:00');
        expect(formatClock(null)).toBe('00:00:00');
        expect(formatClock(undefined)).toBe('00:00:00');
        expect(formatClock(-5)).toBe('00:00:00');
    });
});

describe('formatCountdown', () => {
    it('shows hours when under a day', () => {
        expect(formatCountdown(3_661_000)).toBe('1h 1m 1s');
    });

    it('shows days for longer waits and never goes negative', () => {
        expect(formatCountdown((25 * 3600 + 61) * 1000)).toBe('1d 1h 1m 1s');
        expect(formatCountdown(-1000)).toBe('0h 0m 0s');
    });
});

describe('contestTimeLimitSeconds', () => {
    it('converts Django DurationField microseconds to seconds', () => {
        expect(contestTimeLimitSeconds(3_600_000_000)).toBe(3600);
        expect(contestTimeLimitSeconds(90_000_000)).toBe(90);
    });

    it('is 0 when unset or zero', () => {
        expect(contestTimeLimitSeconds(0)).toBe(0);
        expect(contestTimeLimitSeconds(null)).toBe(0);
        expect(contestTimeLimitSeconds(undefined)).toBe(0);
    });
});

describe('countTotalAc', () => {
    it('counts full-score cells per problem', () => {
        const problems = [{ points: 100 }, { points: 50 }];
        const rows = [
            { breakdown: [{ points: 100 }, { points: 50 }] },
            { breakdown: [{ points: 100 }, { points: 0 }] },
            { breakdown: [null, { points: 50 }] },
        ];
        expect(countTotalAc(rows, problems)).toEqual([2, 2]);
    });

    it('ignores zero-point problems', () => {
        expect(countTotalAc([{ breakdown: [{ points: 0 }] }], [{ points: 0 }])).toEqual([0]);
    });
});

describe('buildScoreboardCsv', () => {
    const problems = [{ label: 'A', points: 100 }, { label: 'B', points: 50 }];
    const rows = [
        {
            rank: 1,
            username: 'alice',
            display_name: 'Alice "A" Liddell',
            organization: { short_name: 'ALPHA', name: 'Alpha Org' },
            rating: 1500,
            score: 150,
            cumtime: 3725,
            breakdown: [{ points: 100, time: 60 }, { points: 50, time: 3600 }],
        },
        { rank: 2, username: 'bob', score: 0, cumtime: 0, breakdown: [null, { points: 0 }] },
    ];

    it('emits the displayed columns and escapes quotes', () => {
        const csv = buildScoreboardCsv({ problems, rows, includeName: true, includeOrg: true });
        const lines = csv.split('\n');
        expect(lines[0]).toBe('"Rank","Username","Name","Organization","Rating","Total","A","B","Window"');
        expect(lines[1]).toBe(
            '"1","alice","Alice ""A"" Liddell","ALPHA","1500","150","100 00:01:00","50 01:00:00","01:02:05"'
        );
        expect(lines[2]).toBe('"2","bob","bob","","","0","","0","00:00:00"');
    });

    it('drops name/org columns when toggled off', () => {
        const csv = buildScoreboardCsv({ problems, rows, includeName: false, includeOrg: false });
        expect(csv.split('\n')[0]).toBe('"Rank","Username","Rating","Total","A","B","Window"');
    });

    it('neutralises spreadsheet formula prefixes', () => {
        const hostile = ['=1+1', '+cmd', '-2+3', '@SUM(A1)', '\tlead'].map((username, i) => ({
            rank: i + 1,
            username,
            score: 0,
            cumtime: 0,
        }));
        const csv = buildScoreboardCsv({
            problems: [],
            rows: hostile,
            includeName: false,
            includeOrg: false,
        });
        const cells = csv.split('\n').slice(1).map(line => line.split(',')[1]);
        expect(cells).toEqual([
            '"\'=1+1"',
            '"\'+cmd"',
            '"\'-2+3"',
            '"\'@SUM(A1)"',
            '"\'\tlead"',
        ]);
    });
});