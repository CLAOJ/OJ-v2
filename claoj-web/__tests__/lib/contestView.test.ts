import {
    buildScoreboardCsv,
    contestTimeLimitSeconds,
    countTotalAc,
    formatClock,
    formatCountdown,
    parseContestTab,
} from '@/lib/contestView';

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
            score: 150,
            cumtime: 3725,
            breakdown: [{ points: 100, time: 60 }, { points: 50, time: 3600 }],
        },
        { rank: 2, username: 'bob', score: 0, cumtime: 0, breakdown: [null, { points: 0 }] },
    ];

    it('emits the displayed columns and escapes quotes', () => {
        const csv = buildScoreboardCsv({ problems, rows, includeName: true, includeOrg: true });
        const lines = csv.split('\n');
        expect(lines[0]).toBe('"Rank","Username","Name","Organization","Total","A","B","Window"');
        expect(lines[1]).toBe(
            '"1","alice","Alice ""A"" Liddell","ALPHA","150","100 00:01:00","50 01:00:00","01:02:05"'
        );
        expect(lines[2]).toBe('"2","bob","bob","","0","","0","00:00:00"');
    });

    it('drops name/org columns when toggled off', () => {
        const csv = buildScoreboardCsv({ problems, rows, includeName: false, includeOrg: false });
        expect(csv.split('\n')[0]).toBe('"Rank","Username","Total","A","B","Window"');
    });
});