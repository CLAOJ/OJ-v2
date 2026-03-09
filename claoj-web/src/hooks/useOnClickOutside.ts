'use client';

import { useEffect, RefObject, useCallback } from 'react';

type EventType = 'mousedown' | 'mouseup' | 'touchstart' | 'touchend';

interface UseOnClickOutsideOptions {
    enabled?: boolean;
    eventType?: EventType;
}

/**
 * Custom hook for handling clicks outside a referenced element
 *
 * @example
 * const ref = useRef<HTMLDivElement>(null);
 * useOnClickOutside(ref, () => {
 *   setIsOpen(false);
 * });
 *
 * return (
 *   <div ref={ref}>
 *     {isOpen && <Dropdown />}
 *   </div>
 * );
 *
 * @example
 * // With multiple refs (e.g., for a dropdown with a separate trigger)
 * const dropdownRef = useRef<HTMLDivElement>(null);
 * const triggerRef = useRef<HTMLButtonElement>(null);
 * useOnClickOutside([dropdownRef, triggerRef], () => setIsOpen(false));
 */
export function useOnClickOutside<T extends HTMLElement = HTMLElement>(
    ref: RefObject<T | null> | RefObject<T | null>[],
    handler: (event: MouseEvent | TouchEvent) => void,
    options: UseOnClickOutsideOptions = {}
): void {
    const { enabled = true, eventType = 'mousedown' } = options;

    const listener = useCallback(
        (event: MouseEvent | TouchEvent) => {
            const target = event.target as Node;

            // Handle single ref or array of refs
            const refs = Array.isArray(ref) ? ref : [ref];

            // Check if click was inside any of the refs
            const isInside = refs.some(
                r => r.current && r.current.contains(target)
            );

            // Do nothing if clicking ref's element or descendent elements
            if (isInside) {
                return;
            }

            handler(event);
        },
        [ref, handler]
    );

    useEffect(() => {
        if (!enabled) return;

        document.addEventListener(eventType, listener);

        return () => {
            document.removeEventListener(eventType, listener);
        };
    }, [enabled, eventType, listener]);
}

/**
 * Hook that combines onClickOutside with Escape key handling
 *
 * @example
 * const { ref, isOpen, setIsOpen } = useDismissible({
 *   onDismiss: () => console.log('Dismissed')
 * });
 */
export function useDismissible<T extends HTMLElement = HTMLElement>(options: {
    onDismiss?: () => void;
    enabled?: boolean;
}): {
    ref: React.RefObject<T | null>;
    dismiss: () => void;
} {
    const { onDismiss, enabled = true } = options;
    const ref = React.useRef<T>(null);

    useOnClickOutside(ref, () => {
        onDismiss?.();
    }, { enabled });

    useEffect(() => {
        if (!enabled) return;

        const handleEscape = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                onDismiss?.();
            }
        };

        document.addEventListener('keydown', handleEscape);
        return () => document.removeEventListener('keydown', handleEscape);
    }, [enabled, onDismiss]);

    const dismiss = useCallback(() => {
        onDismiss?.();
    }, [onDismiss]);

    return { ref, dismiss };
}

// Import React for useDismissible
import React from 'react';
