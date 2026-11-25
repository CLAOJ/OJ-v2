import { buildMatrixLookup, formatRuntimeVersions, runtimeCellState } from '@/lib/runtimes';
import type { RuntimeMatrixCell } from '@/types';

describe('formatRuntimeVersions', () => {
    it('renders a single version, a range, and a version-less runtime', () => {
        expect(formatRuntimeVersions([{ name: 'Python', versions: ['3.10.1'] }]))
            .toBe('Python 3.10.1');
        expect(formatRuntimeVersions([{ name: 'Python', versions: ['3.9.0', '3.10.1'] }]))
            .toBe('Python 3.9.0 - 3.10.1');
        expect(formatRuntimeVersions([{ name: 'GCC', versions: [] }]))
            .toBe('GCC');
    });

    it('joins several runtimes with commas like v1', () => {
        expect(formatRuntimeVersions([
            { name: 'Python', versions: ['3.10.1'] },
            { name: 'PyPy', versions: ['7.3.9'] },
        ])).toBe('Python 3.10.1, PyPy 7.3.9');
    });

    it('returns an empty string for no runtimes', () => {
        expect(formatRuntimeVersions([])).toBe('');
    });
});

describe('matrix helpers', () => {
    const cell: RuntimeMatrixCell = {
        judge: 'w1',
        language_id: 2,
        runtimes: [{ name: 'Python', versions: ['3.10.1'] }],
        is_latest: true,
    };

    it('indexes cells by judge and language id', () => {
        const lookup = buildMatrixLookup([cell]);
        expect(lookup.get('w1|2')).toBe(cell);
        expect(lookup.get('w1|3')).toBeUndefined();
    });

    it('classifies blank, latest and outdated cells', () => {
        expect(runtimeCellState(undefined)).toBe('blank');
        expect(runtimeCellState(cell)).toBe('latest');
        expect(runtimeCellState({ ...cell, is_latest: false })).toBe('outdated');
    });
});