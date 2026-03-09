import { useEffect, useRef, useCallback } from 'react';

interface UseFocusTrapOptions {
    isActive: boolean;
    onEscape?: () => void;
    lockBodyScroll?: boolean;
    restoreFocus?: boolean;
}

export function useFocusTrap({
    isActive,
    onEscape,
    lockBodyScroll = false,
    restoreFocus = false
}: UseFocusTrapOptions) {
    const containerRef = useRef<HTMLDivElement>(null);
    const previousActiveElement = useRef<HTMLElement | null>(null);

    // Store the previously focused element when trap activates
    useEffect(() => {
        if (isActive && restoreFocus) {
            previousActiveElement.current = document.activeElement as HTMLElement;
        }
    }, [isActive, restoreFocus]);

    // Restore focus when trap deactivates
    useEffect(() => {
        if (!isActive && restoreFocus && previousActiveElement.current) {
            previousActiveElement.current.focus();
        }
    }, [isActive, restoreFocus]);

    // Handle body scroll lock
    useEffect(() => {
        if (!isActive || !lockBodyScroll) return;

        const originalOverflow = document.body.style.overflow;
        document.body.style.overflow = 'hidden';

        return () => {
            document.body.style.overflow = originalOverflow;
        };
    }, [isActive, lockBodyScroll]);

    const handleEscape = useCallback((e: KeyboardEvent) => {
        if (e.key === 'Escape' && onEscape) {
            onEscape();
        }
    }, [onEscape]);

    useEffect(() => {
        if (!isActive) return;

        const container = containerRef.current;
        if (!container) return;

        const focusableElements = container.querySelectorAll<HTMLElement>(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"]), [role="button"], [role="menuitem"]'
        );

        if (focusableElements.length === 0) {
            // If no focusable elements, focus the container itself
            container.focus();
            return;
        }

        const firstEl = focusableElements[0];
        const lastEl = focusableElements[focusableElements.length - 1];

        // Focus first element when activated (with small delay for DOM to settle)
        const focusTimeout = setTimeout(() => {
            firstEl.focus();
        }, 0);

        const handleTabKey = (e: KeyboardEvent) => {
            if (e.key !== 'Tab') return;

            if (e.shiftKey && document.activeElement === firstEl) {
                e.preventDefault();
                lastEl.focus();
            } else if (!e.shiftKey && document.activeElement === lastEl) {
                e.preventDefault();
                firstEl.focus();
            }
        };

        // Use document for escape to catch it anywhere
        document.addEventListener('keydown', handleEscape);
        // Use container for tab to trap focus within
        container.addEventListener('keydown', handleTabKey);

        return () => {
            clearTimeout(focusTimeout);
            document.removeEventListener('keydown', handleEscape);
            container.removeEventListener('keydown', handleTabKey);
        };
    }, [isActive, handleEscape]);

    return containerRef;
}
