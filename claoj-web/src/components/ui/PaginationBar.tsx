'use client';

import { useTranslations } from 'next-intl';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';

export const PAGE_SIZE_OPTIONS = [10, 20, 50];

interface PaginationBarProps {
    page: number;
    onPageChange: (page: number) => void;
    /** Total matching rows. Omit only when the endpoint cannot report one. */
    total?: number;
    pageSize: number;
    /** Omit to hide the rows-per-page selector. */
    onPageSizeChange?: (size: number) => void;
    pageSizeOptions?: number[];
    /** Fallback for totalless endpoints: whether a next page likely exists. */
    hasMore?: boolean;
    className?: string;
}

/**
 * The single pagination control for public list pages: rows-per-page on the
 * left, page navigation in the middle, range summary on the right. Always sits
 * below the list it pages.
 *
 * @example
 * <PaginationBar
 *   page={page}
 *   onPageChange={setPage}
 *   total={data?.total}
 *   pageSize={pageSize}
 *   onPageSizeChange={setPageSize}
 * />
 */
export function PaginationBar({
    page,
    onPageChange,
    total,
    pageSize,
    onPageSizeChange,
    pageSizeOptions = PAGE_SIZE_OPTIONS,
    hasMore,
    className,
}: PaginationBarProps) {
    const t = useTranslations('Common');

    const knowsTotal = typeof total === 'number';
    const totalPages = knowsTotal ? Math.max(1, Math.ceil(total / pageSize)) : 0;
    const canGoPrevious = page > 1;
    const canGoNext = knowsTotal ? page < totalPages : !!hasMore;

    // Slide a five-wide window over the page range, clamped at both ends.
    const windowSize = Math.min(5, totalPages);
    const windowStart = Math.min(
        Math.max(1, page - 2),
        Math.max(1, totalPages - windowSize + 1)
    );
    const pageNumbers = Array.from({ length: windowSize }, (_, i) => windowStart + i);

    const firstItem = total === 0 ? 0 : (page - 1) * pageSize + 1;
    const lastItem = knowsTotal ? Math.min(page * pageSize, total) : page * pageSize;

    return (
        <div
            className={cn(
                'flex flex-col-reverse sm:flex-row items-center justify-between gap-4 px-6 py-5 rounded-[2rem] bg-card border',
                className
            )}
        >
            {onPageSizeChange ? (
                <div className="flex items-center gap-3">
                    <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">
                        {t('perPage')}
                    </span>
                    <div className="flex gap-1 p-1 rounded-2xl bg-muted/30 border">
                        {pageSizeOptions.map(size => (
                            <button
                                key={size}
                                onClick={() => onPageSizeChange(size)}
                                className={cn(
                                    'h-9 w-12 rounded-xl text-[11px] font-black transition-all',
                                    pageSize === size
                                        ? 'bg-primary text-primary-foreground shadow-lg shadow-primary/20'
                                        : 'text-muted-foreground hover:bg-muted'
                                )}
                            >
                                {size}
                            </button>
                        ))}
                    </div>
                </div>
            ) : (
                <div className="hidden sm:block" />
            )}

            <div className="flex items-center gap-2">
                <button
                    onClick={() => onPageChange(page - 1)}
                    disabled={!canGoPrevious}
                    aria-label={t('previous')}
                    className="w-10 h-10 rounded-xl bg-muted/30 border flex items-center justify-center transition-all hover:bg-muted disabled:opacity-20 disabled:pointer-events-none"
                >
                    <ChevronLeft size={18} />
                </button>

                {knowsTotal ? (
                    pageNumbers.map(pageNum => (
                        <button
                            key={pageNum}
                            onClick={() => onPageChange(pageNum)}
                            aria-current={page === pageNum ? 'page' : undefined}
                            className={cn(
                                'min-w-10 h-10 px-3 rounded-xl text-xs font-black transition-all',
                                page === pageNum
                                    ? 'bg-primary text-primary-foreground shadow-lg shadow-primary/20'
                                    : 'bg-muted/30 border hover:bg-muted text-muted-foreground'
                            )}
                        >
                            {pageNum}
                        </button>
                    ))
                ) : (
                    <div className="h-10 px-4 rounded-xl bg-primary text-primary-foreground font-black text-xs flex items-center shadow-lg shadow-primary/20">
                        {t('page')} {page}
                    </div>
                )}

                <button
                    onClick={() => onPageChange(page + 1)}
                    disabled={!canGoNext}
                    aria-label={t('next')}
                    className="w-10 h-10 rounded-xl bg-muted/30 border flex items-center justify-center transition-all hover:bg-muted disabled:opacity-20 disabled:pointer-events-none"
                >
                    <ChevronRight size={18} />
                </button>
            </div>

            {knowsTotal ? (
                <div className="text-[11px] font-bold text-muted-foreground">
                    {t.rich('paginationShowing', {
                        from: firstItem,
                        to: lastItem,
                        total,
                        b: chunks => <span className="font-black text-foreground">{chunks}</span>,
                    })}
                </div>
            ) : (
                <div className="hidden sm:block" />
            )}
        </div>
    );
}
