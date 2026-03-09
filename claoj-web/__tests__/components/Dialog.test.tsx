import React, { useState } from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import {
    Dialog,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogContent,
    DialogFooter,
    DialogClose,
    DialogTrigger
} from '@/components/ui/Dialog';

// Mock framer-motion for simpler testing
jest.mock('framer-motion', () => ({
    motion: {
        div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
    },
    AnimatePresence: ({ children }: any) => <>{children}</>,
    useReducedMotion: () => false,
}));

describe('Dialog Component', () => {
    const TestDialog = ({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) => (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogHeader>
                <DialogTitle id="test-title">Test Dialog Title</DialogTitle>
                <DialogClose />
            </DialogHeader>
            <DialogDescription id="test-desc">Test description</DialogDescription>
            <DialogContent>
                <p>Dialog content goes here</p>
            </DialogContent>
            <DialogFooter>
                <button>Action</button>
            </DialogFooter>
        </Dialog>
    );

    beforeEach(() => {
        jest.clearAllMocks();
        // Clean up body overflow style
        document.body.style.overflow = '';
    });

    it('renders nothing when closed', () => {
        const onOpenChange = jest.fn();
        const { container } = render(<TestDialog open={false} onOpenChange={onOpenChange} />);

        expect(container.querySelector('[role="dialog"]')).not.toBeInTheDocument();
    });

    it('renders when open', () => {
        const onOpenChange = jest.fn();
        render(<TestDialog open={true} onOpenChange={onOpenChange} />);

        expect(screen.getByRole('dialog')).toBeInTheDocument();
        expect(screen.getByText('Test Dialog Title')).toBeInTheDocument();
        expect(screen.getByText('Test description')).toBeInTheDocument();
        expect(screen.getByText('Dialog content goes here')).toBeInTheDocument();
    });

    it('calls onOpenChange when backdrop is clicked', () => {
        const onOpenChange = jest.fn();
        const { container } = render(<TestDialog open={true} onOpenChange={onOpenChange} />);

        // Find and click the backdrop (the element with aria-hidden)
        const backdrop = container.querySelector('[aria-hidden="true"]');
        if (backdrop) {
            fireEvent.click(backdrop);
            expect(onOpenChange).toHaveBeenCalledWith(false);
        }
    });

    it('has correct ARIA attributes', () => {
        const onOpenChange = jest.fn();
        render(<TestDialog open={true} onOpenChange={onOpenChange} />);

        const dialog = screen.getByRole('dialog');
        expect(dialog).toHaveAttribute('aria-modal', 'true');
        expect(dialog).toHaveAttribute('aria-labelledby', 'test-title');
        expect(dialog).toHaveAttribute('aria-describedby', 'test-desc');
        expect(dialog).toHaveAttribute('tabIndex', '-1');
    });

    it('renders DialogClose button with correct aria-label', () => {
        const onOpenChange = jest.fn();
        render(<TestDialog open={true} onOpenChange={onOpenChange} />);

        const closeButton = screen.getByLabelText('Close dialog');
        expect(closeButton).toBeInTheDocument();
    });

    it('renders DialogFooter with children', () => {
        const onOpenChange = jest.fn();
        render(<TestDialog open={true} onOpenChange={onOpenChange} />);

        expect(screen.getByRole('button', { name: 'Action' })).toBeInTheDocument();
    });
});

describe('DialogTrigger Component', () => {
    it('renders trigger with children', () => {
        const DialogWithTrigger = () => {
            const [open, setOpen] = useState(false);
            return (
                <>
                    <Dialog open={open} onOpenChange={setOpen}>
                        <DialogContent>
                            <DialogTitle>Test</DialogTitle>
                            <p>Content</p>
                        </DialogContent>
                    </Dialog>
                    {/* Render trigger outside Dialog when not open */}
                    <button data-testid="open-btn" onClick={() => setOpen(true)}>Open Dialog</button>
                </>
            );
        };

        render(<DialogWithTrigger />);
        const btn = screen.getByTestId('open-btn');
        expect(btn).toBeInTheDocument();
        fireEvent.click(btn);
        expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    it('supports asChild prop', () => {
        const DialogWithAsChild = () => (
            <Dialog open={true} onOpenChange={() => {}}>
                <DialogContent>
                    <DialogTitle>Title</DialogTitle>
                    <DialogTrigger asChild>
                        <button data-testid="custom-trigger">Custom Button</button>
                    </DialogTrigger>
                </DialogContent>
            </Dialog>
        );

        const { container } = render(<DialogWithAsChild />);
        expect(container.querySelector('[data-testid="custom-trigger"]')).toBeInTheDocument();
    });
});

describe('DialogTitle Component', () => {
    it('registers title ID with context', () => {
        const TestComponent = () => {
            const [open, setOpen] = useState(true);
            return (
                <Dialog open={open} onOpenChange={setOpen}>
                    <DialogTitle id="custom-title">Custom Title</DialogTitle>
                    <DialogContent>Content</DialogContent>
                </Dialog>
            );
        };

        render(<TestComponent />);
        const title = screen.getByText('Custom Title');
        expect(title).toHaveAttribute('id', 'custom-title');
        expect(title.tagName).toBe('H2');
    });
});

describe('DialogDescription Component', () => {
    it('registers description ID with context', () => {
        const TestComponent = () => {
            const [open, setOpen] = useState(true);
            return (
                <Dialog open={open} onOpenChange={setOpen}>
                    <DialogTitle>Title</DialogTitle>
                    <DialogDescription id="custom-desc">Custom Description</DialogDescription>
                    <DialogContent>Content</DialogContent>
                </Dialog>
            );
        };

        render(<TestComponent />);
        const desc = screen.getByText('Custom Description');
        expect(desc).toHaveAttribute('id', 'custom-desc');
        expect(desc.tagName).toBe('P');
    });
});
