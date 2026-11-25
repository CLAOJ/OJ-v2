import { buildDiffMatrix, diffSelectionUrl, toggleDiffSelection } from '@/lib/diffMatrix';
import type { AdminProblemDiffSubmission } from '@/types';

const rows: AdminProblemDiffSubmission[] = [
    {
        id: 9,
        user: 'alice',
        date: '2026-01-02T03:04:05Z',
        language: 'CPP17',
        status: 'D',
        result: 'AC',
        points: 100,
        time: 0.5,
        cases: [
            { case: 2, status: 'AC', points: 50, total: 50 },
            { case: 1, status: 'AC', points: 50, total: 50 },
        ],
    },
    {
        id: 4,
        user: 'bob',
        date: '2026-01-01T03:04:05Z',
        language: 'PY3',
        status: 'D',
        result: 'WA',
        points: 40,
        time: 0.9,
        cases: [
            { case: 1, status: 'AC', points: 50, total: 50 },
            { case: 3, status: 'WA', points: 0, total: 50 },
        ],
    },
];

describe('buildDiffMatrix', () => {
    it('takes the sorted union of case numbers as columns', () => {
        const matrix = buildDiffMatrix(rows);
        expect(matrix.caseNumbers).toEqual([1, 2, 3]);
        expect(matrix.rows).toHaveLength(2);
    });

    it('indexes each row by case and leaves gaps empty', () => {
        const matrix = buildDiffMatrix(rows);
        expect(matrix.rows[0].cells.get(2)?.status).toBe('AC');
        expect(matrix.rows[0].cells.get(3)).toBeUndefined();
        expect(matrix.rows[1].cells.get(3)?.points).toBe(0);
    });

    it('handles an empty list', () => {
        expect(buildDiffMatrix([])).toEqual({ caseNumbers: [], rows: [] });
    });
});

describe('toggleDiffSelection', () => {
    it('adds and removes a row', () => {
        expect(toggleDiffSelection([], 9)).toEqual([9]);
        expect(toggleDiffSelection([9], 9)).toEqual([]);
    });

    it('keeps at most two and drops the oldest on a third pick', () => {
        expect(toggleDiffSelection([9], 4)).toEqual([9, 4]);
        expect(toggleDiffSelection([9, 4], 7)).toEqual([4, 7]);
    });
});

describe('diffSelectionUrl', () => {
    it('links only a complete pair to the existing diff viewer', () => {
        expect(diffSelectionUrl([9])).toBeNull();
        expect(diffSelectionUrl([])).toBeNull();
        expect(diffSelectionUrl([9, 4])).toBe('/submissions/diff/9?compare=4');
    });
});