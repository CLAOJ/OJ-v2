import * as React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import { Button, buttonVariants } from '@/components/ui/Button';
import { cn } from '@/lib/utils';

describe('Button Component', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    it('renders children correctly', () => {
        render(<Button>Click me</Button>);
        expect(screen.getByRole('button', { name: /click me/i })).toBeInTheDocument();
    });

    it('applies default variants', () => {
        const { container } = render(<Button>Test</Button>);
        const button = container.firstChild as HTMLElement;
        expect(button).toHaveClass('bg-primary');
        expect(button).toHaveClass('h-10');
    });

    it('applies variant variants', () => {
        const { rerender } = render(<Button variant="default">Default</Button>);
        expect(screen.getByRole('button')).toHaveClass('bg-primary');

        rerender(<Button variant="destructive">Destructive</Button>);
        expect(screen.getByRole('button')).toHaveClass('bg-destructive');

        rerender(<Button variant="outline">Outline</Button>);
        expect(screen.getByRole('button')).toHaveClass('border');

        rerender(<Button variant="secondary">Secondary</Button>);
        expect(screen.getByRole('button')).toHaveClass('bg-secondary');

        rerender(<Button variant="ghost">Ghost</Button>);
        expect(screen.getByRole('button')).toHaveClass('hover:bg-accent');

        rerender(<Button variant="link">Link</Button>);
        expect(screen.getByRole('button')).toHaveClass('underline-offset-4');

        rerender(<Button variant="success">Success</Button>);
        expect(screen.getByRole('button')).toHaveClass('bg-emerald-500');

        rerender(<Button variant="warning">Warning</Button>);
        expect(screen.getByRole('button')).toHaveClass('bg-amber-500');
    });

    it('applies size variants', () => {
        const { rerender } = render(<Button size="sm">Small</Button>);
        expect(screen.getByRole('button')).toHaveClass('h-9');

        rerender(<Button size="lg">Large</Button>);
        expect(screen.getByRole('button')).toHaveClass('h-11');

        rerender(<Button size="icon">Icon</Button>);
        expect(screen.getByRole('button')).toHaveClass('h-10 w-10');
    });

    it('applies loading state', () => {
        render(<Button loading>Loading</Button>);
        const button = screen.getByRole('button');
        expect(button).toHaveClass('pointer-events-none');
        expect(screen.getByTestId('loader')).toBeInTheDocument();
    });

    it('forwards ref correctly', () => {
        const ref = React.createRef<HTMLButtonElement>();
        render(<Button ref={ref}>Test</Button>);
        expect(ref.current).toBeInTheDocument();
    });

    it('handles click events', () => {
        const handleClick = jest.fn();
        render(<Button onClick={handleClick}>Click</Button>);
        fireEvent.click(screen.getByRole('button'));
        expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it('prevents click when loading', () => {
        const handleClick = jest.fn();
        render(<Button loading onClick={handleClick}>Loading</Button>);
        fireEvent.click(screen.getByRole('button'));
        expect(handleClick).not.toHaveBeenCalled();
    });

    it('respects disabled prop', () => {
        const handleClick = jest.fn();
        render(<Button disabled onClick={handleClick}>Disabled</Button>);
        expect(screen.getByRole('button')).toBeDisabled();
        fireEvent.click(screen.getByRole('button'));
        expect(handleClick).not.toHaveBeenCalled();
    });

    it('passes through HTML attributes', () => {
        render(
            <Button
                type="submit"
                name="test-button"
                aria-label="Test button"
                data-testid="custom-test"
            >
                Test
            </Button>
        );
        const button = screen.getByTestId('custom-test');
        expect(button).toHaveAttribute('type', 'submit');
        expect(button).toHaveAttribute('name', 'test-button');
        expect(button).toHaveAttribute('aria-label', 'Test button');
    });

    it('applies custom className', () => {
        render(<Button className="custom-class another-class">Test</Button>);
        expect(screen.getByRole('button')).toHaveClass('custom-class');
        expect(screen.getByRole('button')).toHaveClass('another-class');
    });

    it('merges className with variants', () => {
        render(<Button variant="outline" className="custom">Test</Button>);
        const button = screen.getByRole('button');
        expect(button).toHaveClass('border');
        expect(button).toHaveClass('custom');
    });
});

describe('buttonVariants function', () => {
    it('returns correct class string for default variant', () => {
        const result = buttonVariants({ variant: 'default', size: 'default' });
        expect(result).toContain('bg-primary');
        expect(result).toContain('h-10');
    });

    it('returns correct class string for destructive variant', () => {
        const result = buttonVariants({ variant: 'destructive' });
        expect(result).toContain('bg-destructive');
    });

    it('returns correct class string for loading', () => {
        const result = buttonVariants({ loading: true });
        expect(result).toContain('pointer-events-none');
    });

    it('combines multiple variants', () => {
        const result = buttonVariants({ variant: 'outline', size: 'lg', loading: true });
        expect(result).toContain('border');
        expect(result).toContain('h-11');
        expect(result).toContain('pointer-events-none');
    });
});

// Mock Loader2 for the loading test
jest.mock('lucide-react', () => ({
    Loader2: ({ className, ...props }: any) => (
        <svg data-testid="loader" className={className} {...props}>
            <circle cx="12" cy="12" r="10" />
        </svg>
    ),
}));
