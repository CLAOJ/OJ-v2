import { useEffect, useRef } from 'react';

interface UseFocusTrapOptions {
    isActive: boolean;
    onEscape?: () => void;
    lockBodyScroll?: boolean;
    restoreFocus?: boolean;
}

const FOCUSABLE =
    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"]), [role="button"], [role="menuitem"]';

export function useFocusTrap({
    isActive,
    onEscape,
    lockBodyScroll = false,
    restoreFocus = false
}: UseFocusTrapOptions) {
    const containerRef = useRef<HTMLDivElement>(null);
    const previousActiveElement = useRef<HTMLElement | null>(null);
    // Callers pass a fresh onEscape every render; reading it through a ref
    // keeps the trap effect below from re-running (and re-focusing the first
    // field) on every keystroke in the dialog.
    const onEscapeRef = useRef(onEscape);
    useEffect(() => { onEscapeRef.current = onEscape; });

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

    // Runs once per activation: focus the first field, trap Tab, handle Escape.
    useEffect(() => {
        if (!isActive) return;

        const container = containerRef.current;
        if (!container) return;

        const focusables = () => container.querySelectorAll<HTMLElement>(FOCUSABLE);

        // Focus the first element once the DOM has settled
        const focusTimeout = setTimeout(() => {
            const first = focusables()[0];
            (first ?? container).focus();
        }, 0);

        const handleEscape = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onEscapeRef.current?.();
        };

        // Content can change while open, so look the ends up on each Tab.
        const handleTabKey = (e: KeyboardEvent) => {
            if (e.key !== 'Tab') return;
            const els = focusables();
            if (els.length === 0) return;
            const firstEl = els[0];
            const lastEl = els[els.length - 1];

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
    }, [isActive]);

    return containerRef;
}
