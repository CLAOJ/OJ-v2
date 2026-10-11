'use client';

import { useCallback, useMemo } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';

export type ListFilters = Record<string, string>;

/** Pure: page (>= 1) and the named, non-empty filters from a query string. */
export function readListParams(sp: URLSearchParams, keys: readonly string[]): { page: number; filters: ListFilters } {
    const page = Math.max(1, parseInt(sp.get('page') ?? '1', 10) || 1);
    const filters: ListFilters = {};
    for (const k of keys) {
        const v = sp.get(k);
        if (v) filters[k] = v;
    }
    return { page, filters };
}

/** Pure: query string for page + non-empty filters; page 1 is implied. */
export function buildListQuery(page: number, filters: ListFilters): string {
    const sp = new URLSearchParams();
    for (const [k, v] of Object.entries(filters)) if (v) sp.set(k, v);
    if (page > 1) sp.set('page', String(page));
    const s = sp.toString();
    return s ? `?${s}` : '';
}

/**
 * List page + filters kept in the URL so a filtered view is linkable and
 * survives reload. Changing a filter resets to page 1. Pass a module-level
 * constant for `keys`.
 */
export function useListParams(keys: readonly string[]) {
    const router = useRouter();
    const pathname = usePathname();
    const searchParams = useSearchParams();
    const { page, filters } = useMemo(
        () => readListParams(new URLSearchParams(searchParams.toString()), keys),
        [searchParams, keys]
    );

    const go = useCallback(
        (p: number, f: ListFilters) => router.replace(pathname + buildListQuery(p, f), { scroll: false }),
        [router, pathname]
    );
    const setFilters = useCallback((patch: ListFilters) => go(1, { ...filters, ...patch }), [go, filters]);
    const setPage = useCallback((p: number) => go(p, filters), [go, filters]);
    const reset = useCallback(() => go(1, {}), [go]);

    return { page, filters, setFilters, setPage, reset };
}
