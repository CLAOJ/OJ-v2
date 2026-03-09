'use client';

import { Search, X } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useState, useCallback } from 'react';
import { useDebounce } from '@/hooks/useDebounce';

interface SearchInputProps {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    className?: string;
    debounceMs?: number;
    onDebouncedChange?: (value: string) => void;
    showClearButton?: boolean;
    autoFocus?: boolean;
}

/**
 * Search input with icon and optional debouncing
 *
 * @example
 * // Basic usage
 * <SearchInput
 *   value={search}
 *   onChange={setSearch}
 *   placeholder="Search users..."
 * />
 *
 * @example
 * // With debouncing for API calls
 * <SearchInput
 *   value={search}
 *   onChange={setSearch}
 *   debounceMs={300}
 *   onDebouncedChange={(value) => fetchResults(value)}
 *   placeholder="Search..."
 * />
 */
export function SearchInput({
    value,
    onChange,
    placeholder = 'Search...',
    className,
    debounceMs,
    onDebouncedChange,
    showClearButton = true,
    autoFocus = false,
}: SearchInputProps) {
    const debouncedValue = useDebounce(value, debounceMs || 0);

    // Track if we've fired the initial debounced change
    const [hasFiredInitial, setHasFiredInitial] = useState(false);

    // Fire debounced change when value changes
    useState(() => {
        if (onDebouncedChange && debounceMs && (hasFiredInitial || debouncedValue === value)) {
            onDebouncedChange(debouncedValue);
        }
        setHasFiredInitial(true);
    });

    const handleClear = useCallback(() => {
        onChange('');
        if (onDebouncedChange) {
            onDebouncedChange('');
        }
    }, [onChange, onDebouncedChange]);

    return (
        <div className={cn('relative', className)}>
            <Search
                className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground pointer-events-none"
                size={18}
            />
            <input
                type="text"
                value={value}
                onChange={(e) => onChange(e.target.value)}
                placeholder={placeholder}
                autoFocus={autoFocus}
                className={cn(
                    'w-full h-10 pl-10 pr-10 rounded-xl bg-card border',
                    'focus:ring-2 focus:ring-primary/20 focus:outline-none',
                    'placeholder:text-muted-foreground',
                    'transition-all'
                )}
            />
            {showClearButton && value && (
                <button
                    type="button"
                    onClick={handleClear}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                    aria-label="Clear search"
                >
                    <X size={16} />
                </button>
            )}
        </div>
    );
}

interface AdminSearchInputProps extends Omit<SearchInputProps, 'className'> {
    wrapperClassName?: string;
}

/**
 * Search input styled for admin pages with full width on mobile
 */
export function AdminSearchInput({
    wrapperClassName,
    ...props
}: AdminSearchInputProps) {
    return (
        <div className={cn(
            'w-full md:w-80',
            wrapperClassName
        )}>
            <SearchInput {...props} className="w-full" />
        </div>
    );
}
