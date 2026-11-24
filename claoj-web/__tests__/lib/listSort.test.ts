import { toggleSort, yearOptions, type ProblemSort } from '@/lib/listSort';

describe('toggleSort', () => {
    it('starts a new column at the given first order', () => {
        expect(toggleSort<ProblemSort>({ sort: 'date', order: 'desc' }, 'name', 'asc')).toEqual({
            sort: 'name',
            order: 'asc',
        });
    });

    it('flips order when the same column is clicked again', () => {
        expect(toggleSort<ProblemSort>({ sort: 'name', order: 'asc' }, 'name', 'asc')).toEqual({
            sort: 'name',
            order: 'desc',
        });
        expect(toggleSort<ProblemSort>({ sort: 'name', order: 'desc' }, 'name', 'asc')).toEqual({
            sort: 'name',
            order: 'asc',
        });
    });

    it('does not carry the old order into a different column', () => {
        expect(toggleSort<ProblemSort>({ sort: 'name', order: 'asc' }, 'points', 'desc')).toEqual({
            sort: 'points',
            order: 'desc',
        });
    });
});

describe('yearOptions', () => {
    it('lists the latest year down to 2021, newest first', () => {
        expect(yearOptions(2026)).toEqual([2026, 2025, 2024, 2023, 2022, 2021]);
    });

    it('stops at the oldest year and handles a range of one', () => {
        expect(yearOptions(2026, 2024)).toEqual([2026, 2025, 2024]);
        expect(yearOptions(2021)).toEqual([2021]);
    });
});