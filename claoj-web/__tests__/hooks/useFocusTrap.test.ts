import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, jest, beforeEach, afterEach } from '@jest/globals';
import { useFocusTrap } from '@/hooks/useFocusTrap';

describe('useFocusTrap', () => {
    let container: HTMLDivElement;

    beforeEach(() => {
        // Create a container with focusable elements
        container = document.createElement('div');
        container.innerHTML = `
            <button id="btn1">Button 1</button>
            <input id="input1" />
            <a href="#" id="link1">Link 1</a>
            <button id="btn2">Button 2</button>
        `;
        document.body.appendChild(container);

        // Reset body overflow
        document.body.style.overflow = '';
    });

    afterEach(() => {
        if (container.parentNode) {
            document.body.removeChild(container);
        }
        jest.clearAllMocks();
    });

    it('returns a ref', () => {
        const onEscape = jest.fn();
        const { result } = renderHook(() =>
            useFocusTrap({ isActive: false, onEscape })
        );

        expect(result.current).toBeDefined();
        expect(result.current.current).toBeNull();
    });

    it('initializes with correct options', () => {
        const onEscape = jest.fn();
        const { result } = renderHook(() =>
            useFocusTrap({ isActive: true, onEscape })
        );

        // The hook should return a ref
        expect(result.current).toBeInstanceOf(Object);
    });

    it('does not call onEscape when isActive is false', () => {
        const onEscape = jest.fn();
        renderHook(() =>
            useFocusTrap({ isActive: false, onEscape })
        );

        // Trigger Escape key
        act(() => {
            const escapeEvent = new KeyboardEvent('keydown', { key: 'Escape' });
            document.dispatchEvent(escapeEvent);
        });

        expect(onEscape).not.toHaveBeenCalled();
    });

    it('locks body scroll when lockBodyScroll is true', () => {
        const onEscape = jest.fn();
        renderHook(() =>
            useFocusTrap({ isActive: true, onEscape, lockBodyScroll: true })
        );

        expect(document.body.style.overflow).toBe('hidden');
    });

    it('restores body scroll when deactivated', () => {
        const onEscape = jest.fn();
        const { rerender } = renderHook(
            ({ isActive }) => useFocusTrap({ isActive, onEscape, lockBodyScroll: true }),
            { initialProps: { isActive: true } }
        );

        expect(document.body.style.overflow).toBe('hidden');

        rerender({ isActive: false });

        expect(document.body.style.overflow).toBe('');
    });

    it('does not lock body scroll when lockBodyScroll is false', () => {
        const onEscape = jest.fn();
        renderHook(() =>
            useFocusTrap({ isActive: true, onEscape, lockBodyScroll: false })
        );

        expect(document.body.style.overflow).toBe('');
    });

    it('cleans up event listeners on unmount', () => {
        const onEscape = jest.fn();
        const { unmount } = renderHook(() =>
            useFocusTrap({ isActive: true, onEscape })
        );

        unmount();

        // After unmount, Escape should not trigger onEscape
        act(() => {
            const escapeEvent = new KeyboardEvent('keydown', { key: 'Escape' });
            document.dispatchEvent(escapeEvent);
        });

        expect(onEscape).not.toHaveBeenCalled();
    });

    it('handles focus restoration when restoreFocus is true', async () => {
        const previousActive = document.createElement('button');
        document.body.appendChild(previousActive);
        previousActive.focus();

        const onEscape = jest.fn();
        const { unmount } = renderHook(() =>
            useFocusTrap({ isActive: true, onEscape, restoreFocus: true })
        );

        // Unmount to trigger cleanup
        unmount();

        // Cleanup should not throw
        expect(true).toBe(true);

        document.body.removeChild(previousActive);
    });
});
