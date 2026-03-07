import { useEffect, useRef } from 'react';

interface UseFocusTrapOptions {
    isActive: boolean;
    onEscape?: () => void;
}

export function useFocusTrap({ isActive, onEscape }: UseFocusTrapOptions) {
    const containerRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (!isActive || !containerRef.current) return;

        const container = containerRef.current;
        const focusableElements = container.querySelectorAll<HTMLElement>(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"]), [role="menuitem"]'
        );

        if (focusableElements.length === 0) return;

        const firstEl = focusableElements[0];
        const lastEl = focusableElements[focusableElements.length - 1];

        // Focus first element when activated
        firstEl.focus();

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

        const handleEscape = () => {
            if (onEscape) {
                onEscape();
            }
        };

        container.addEventListener('keydown', handleTabKey);
        container.addEventListener('keydown', handleEscape);

        return () => {
            container.removeEventListener('keydown', handleTabKey);
            container.removeEventListener('keydown', handleEscape);
        };
    }, [isActive, onEscape]);

    return containerRef;
}
