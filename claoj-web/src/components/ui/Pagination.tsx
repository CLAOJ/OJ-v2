'use client';

import { Button } from './Button';
import { cn } from '@/lib/utils';
import { ChevronLeft, ChevronRight } from 'lucide-react';

interface PaginationProps {
    page: number;
    totalPages: number;
    total?: number;
    pageSize?: number;
    onPageChange: (page: number) => void;
    showInfo?: boolean;
    className?: string;
}

/**
 * Reusable Pagination component
 *
 * @example
 * <Pagination
 *   page={page}
 *   totalPages={Math.ceil(total / pageSize)}
 *   total={total}
 *   pageSize={pageSize}
 *   onPageChange={setPage}
 * />
 */
export function Pagination({
    page,
    totalPages,
    total,
    pageSize = 20,
    onPageChange,
    showInfo = true,
    className,
}: PaginationProps) {
    const canGoPrevious = page > 1;
    const canGoNext = page < totalPages;

    const startItem = total ? (page - 1) * pageSize + 1 : 0;
    const endItem = total ? Math.min(page * pageSize, total) : 0;

    if (totalPages <= 1 && !total) return null;

    return (
        <div className={cn(
            'flex items-center justify-between px-6 py-4 border-t bg-muted/30',
            className
        )}>
            {showInfo && total !== undefined && (
                <div className="text-sm text-muted-foreground">
                    Showing <span className="font-medium">{startItem}</span> to{' '}
                    <span className="font-medium">{endItem}</span> of{' '}
                    <span className="font-medium">{total}</span>
                </div>
            )}

            <div className={cn(
                'flex items-center gap-2',
                !showInfo && 'ml-auto'
            )}>
                <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onPageChange(page - 1)}
                    disabled={!canGoPrevious}
                    className="flex items-center gap-1"
                >
                    <ChevronLeft size={16} />
                    Previous
                </Button>

                <div className="flex items-center gap-1">
                    {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                        // Show pages around current page
                        let pageNum: number;
                        if (totalPages <= 5) {
                            pageNum = i + 1;
                        } else if (page <= 3) {
                            pageNum = i + 1;
                        } else if (page >= totalPages - 2) {
                            pageNum = totalPages - 4 + i;
                        } else {
                            pageNum = page - 2 + i;
                        }

                        return (
                            <button
                                key={pageNum}
                                onClick={() => onPageChange(pageNum)}
                                className={cn(
                                    'w-9 h-9 rounded-lg text-sm font-medium transition-colors',
                                    page === pageNum
                                        ? 'bg-primary text-primary-foreground'
                                        : 'hover:bg-muted'
                                )}
                            >
                                {pageNum}
                            </button>
                        );
                    })}
                </div>

                <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onPageChange(page + 1)}
                    disabled={!canGoNext}
                    className="flex items-center gap-1"
                >
                    Next
                    <ChevronRight size={16} />
                </Button>
            </div>
        </div>
    );
}

interface SimplePaginationProps {
    page: number;
    onPageChange: (page: number) => void;
    hasMore: boolean;
    className?: string;
}

/**
 * Simple pagination with just Previous/Next buttons
 *
 * @example
 * <SimplePagination
 *   page={page}
 *   onPageChange={setPage}
 *   hasMore={users.length === pageSize}
 * />
 */
export function SimplePagination({
    page,
    onPageChange,
    hasMore,
    className,
}: SimplePaginationProps) {
    return (
        <div className={cn(
            'flex items-center justify-between px-6 py-4 border-t bg-muted/30',
            className
        )}>
            <div className="text-sm text-muted-foreground">
                Page <span className="font-medium">{page}</span>
            </div>

            <div className="flex items-center gap-2">
                <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onPageChange(page - 1)}
                    disabled={page === 1}
                >
                    Previous
                </Button>

                <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onPageChange(page + 1)}
                    disabled={!hasMore}
                >
                    Next
                </Button>
            </div>
        </div>
    );
}
