import type { AdminProblemDiffCase, AdminProblemDiffSubmission } from '@/types';

export interface DiffMatrixRow extends AdminProblemDiffSubmission {
    /** Case number -> result, for O(1) column lookup while rendering. */
    cells: Map<number, AdminProblemDiffCase>;
}

export interface DiffMatrix {
    caseNumbers: number[];
    rows: DiffMatrixRow[];
}

/**
 * Turns the diff endpoint's rows into a matrix: the columns are every case
 * number any submission ran (v1 counted the problem's cases; the union is the
 * same when all submissions were judged against the current data).
 */
export function buildDiffMatrix(submissions: AdminProblemDiffSubmission[]): DiffMatrix {
    const caseNumbers = Array.from(
        new Set(submissions.flatMap(s => s.cases.map(c => c.case))),
    ).sort((a, b) => a - b);
    return {
        caseNumbers,
        rows: submissions.map(s => ({
            ...s,
            cells: new Map(s.cases.map(c => [c.case, c] as const)),
        })),
    };
}

/**
 * Checkbox behaviour: toggling off removes, and a third pick replaces the
 * oldest choice so the pair is always the last two clicked.
 */
export function toggleDiffSelection(selected: number[], id: number): number[] {
    if (selected.includes(id)) return selected.filter(x => x !== id);
    if (selected.length < 2) return [...selected, id];
    return [selected[1], id];
}

/** Viewer URL for the selected pair, or null until two are chosen. */
export function diffSelectionUrl(selected: number[]): string | null {
    if (selected.length !== 2) return null;
    return `/submissions/diff/${selected[0]}?compare=${selected[1]}`;
}