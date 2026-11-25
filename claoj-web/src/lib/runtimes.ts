import type { RuntimeEntry, RuntimeMatrixCell } from '@/types';

/**
 * v1 runtime-version-fragment.html: "Name first" when there is one version,
 * "Name first - last" for a range, bare "Name" when the judge reported none.
 */
export function formatRuntimeVersions(runtimes: RuntimeEntry[]): string {
    return runtimes
        .map(({ name, versions }) => {
            if (!versions.length) return name;
            if (versions.length === 1) return `${name} ${versions[0]}`;
            return `${name} ${versions[0]} - ${versions[versions.length - 1]}`;
        })
        .join(', ');
}

/** Index a matrix payload by "judge|language_id" for table rendering. */
export function buildMatrixLookup(cells: RuntimeMatrixCell[]): Map<string, RuntimeMatrixCell> {
    const map = new Map<string, RuntimeMatrixCell>();
    for (const cell of cells) {
        map.set(`${cell.judge}|${cell.language_id}`, cell);
    }
    return map;
}

/** v1 cell styling: blank (no runtimes), latest (up to date), outdated. */
export function runtimeCellState(cell: RuntimeMatrixCell | undefined): 'blank' | 'latest' | 'outdated' {
    if (!cell) return 'blank';
    return cell.is_latest ? 'latest' : 'outdated';
}