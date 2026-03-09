'use client';

import { useState, useCallback, useMemo } from 'react';

interface UsePaginationOptions {
    initialPage?: number;
    initialPageSize?: number;
}

interface UsePaginationReturn {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
    setPage: (page: number) => void;
    setPageSize: (pageSize: number) => void;
    setTotal: (total: number) => void;
    nextPage: () => void;
    previousPage: () => void;
    canGoNext: boolean;
    canGoPrevious: boolean;
    startIndex: number;
    endIndex: number;
    reset: () => void;
}

/**
 * Custom hook for managing pagination state
 *
 * @example
 * const { page, pageSize, total, setTotal, nextPage, previousPage, canGoNext, canGoPrevious } = usePagination({ initialPage: 1, initialPageSize: 20 });
 *
 * // In your component:
 * const { data } = useQuery({
 *   queryKey: ['items', page],
 *   queryFn: () => fetchItems({ page, pageSize })
 * });
 *
 * useEffect(() => {
 *   if (data) setTotal(data.total);
 * }, [data, setTotal]);
 */
export function usePagination(options: UsePaginationOptions = {}): UsePaginationReturn {
    const { initialPage = 1, initialPageSize = 20 } = options;

    const [page, setPage] = useState(initialPage);
    const [pageSize, setPageSize] = useState(initialPageSize);
    const [total, setTotal] = useState(0);

    const totalPages = useMemo(() => Math.ceil(total / pageSize), [total, pageSize]);

    const canGoNext = useMemo(() => page < totalPages, [page, totalPages]);
    const canGoPrevious = useMemo(() => page > 1, [page]);

    const startIndex = useMemo(() => (page - 1) * pageSize + 1, [page, pageSize]);
    const endIndex = useMemo(() => Math.min(page * pageSize, total), [page, pageSize, total]);

    const nextPage = useCallback(() => {
        if (canGoNext) {
            setPage(p => p + 1);
        }
    }, [canGoNext]);

    const previousPage = useCallback(() => {
        if (canGoPrevious) {
            setPage(p => p - 1);
        }
    }, [canGoPrevious]);

    const reset = useCallback(() => {
        setPage(initialPage);
        setTotal(0);
    }, [initialPage]);

    return {
        page,
        pageSize,
        total,
        totalPages,
        setPage,
        setPageSize,
        setTotal,
        nextPage,
        previousPage,
        canGoNext,
        canGoPrevious,
        startIndex,
        endIndex,
        reset,
    };
}
