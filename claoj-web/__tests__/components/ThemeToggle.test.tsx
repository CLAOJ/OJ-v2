import * as React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import { useTheme } from 'next-themes';
import ThemeToggle from '@/components/navbar/ThemeToggle';

jest.mock('next-themes', () => ({
    useTheme: jest.fn(),
}));

const mockedUseTheme = useTheme as jest.MockedFunction<typeof useTheme>;

// The toggle must key off `resolvedTheme` (the concrete "dark"/"light" after
// resolving "system"), NOT the raw `theme` (which is "system" on first load).
// Reading `theme` makes the first click a no-op and shows the wrong icon.
function setTheme(resolvedTheme: 'dark' | 'light', theme = 'system') {
    const setThemeFn = jest.fn();
    mockedUseTheme.mockReturnValue({
        theme,
        resolvedTheme,
        setTheme: setThemeFn,
        themes: ['light', 'dark', 'system'],
    } as unknown as ReturnType<typeof useTheme>);
    return setThemeFn;
}

describe('ThemeToggle', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    it('offers "switch to light" when system resolves to dark on first load', () => {
        setTheme('dark');
        render(<ThemeToggle />);
        expect(screen.getByRole('button')).toHaveAttribute('aria-label', 'Switch to light mode');
    });

    it('offers "switch to dark" when system resolves to light on first load', () => {
        setTheme('light');
        render(<ThemeToggle />);
        expect(screen.getByRole('button')).toHaveAttribute('aria-label', 'Switch to dark mode');
    });

    it('switches to light on the first click when currently dark (system default)', () => {
        const setThemeFn = setTheme('dark');
        render(<ThemeToggle />);
        fireEvent.click(screen.getByRole('button'));
        expect(setThemeFn).toHaveBeenCalledWith('light');
    });

    it('switches to dark on the first click when currently light (system default)', () => {
        const setThemeFn = setTheme('light');
        render(<ThemeToggle />);
        fireEvent.click(screen.getByRole('button'));
        expect(setThemeFn).toHaveBeenCalledWith('dark');
    });

    it('shows the resolved theme name in the mobile variant', () => {
        setTheme('dark');
        const { container } = render(<ThemeToggle variant="mobile" />);
        expect(container.textContent).toContain('Theme (Dark)');
    });
});
