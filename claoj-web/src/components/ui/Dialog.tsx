'use client';

import { useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X } from 'lucide-react';
import { cn } from '@/lib/utils';

interface DialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    children: React.ReactNode;
    className?: string;
}

export function Dialog({ open, onOpenChange, children, className }: DialogProps) {
    useEffect(() => {
        const handleEscape = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                onOpenChange(false);
            }
        };

        if (open) {
            document.addEventListener('keydown', handleEscape);
            document.body.style.overflow = 'hidden';
        }

        return () => {
            document.removeEventListener('keydown', handleEscape);
            document.body.style.overflow = 'unset';
        };
    }, [open, onOpenChange]);

    return (
        <AnimatePresence>
            {open && (
                <>
                    {/* Backdrop */}
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        onClick={() => onOpenChange(false)}
                        className="fixed inset-0 bg-black/50 z-50"
                    />
                    {/* Dialog */}
                    <motion.div
                        initial={{ opacity: 0, scale: 0.95, y: 20 }}
                        animate={{ opacity: 1, scale: 1, y: 0 }}
                        exit={{ opacity: 0, scale: 0.95, y: 20 }}
                        className={cn(
                            "fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[90vw] -translate-x-1/2 -translate-y-1/2 overflow-auto rounded-2xl bg-popover p-6 shadow-lg",
                            className
                        )}
                    >
                        {children}
                    </motion.div>
                </>
            )}
        </AnimatePresence>
    );
}

interface DialogHeaderProps {
    children: React.ReactNode;
    className?: string;
}

export function DialogHeader({ children, className }: DialogHeaderProps) {
    return <div className={cn("flex items-center justify-between mb-4", className)}>{children}</div>;
}

interface DialogTitleProps {
    children: React.ReactNode;
    className?: string;
}

export function DialogTitle({ children, className }: DialogTitleProps) {
    return <h2 className={cn("text-xl font-bold", className)}>{children}</h2>;
}

interface DialogCloseProps {
    className?: string;
}

export function DialogClose({ className }: DialogCloseProps) {
    return (
        <button
            onClick={(e) => {
                e.stopPropagation();
                const dialog = e.currentTarget.closest('[role="dialog"]');
                if (dialog) {
                    const event = new KeyboardEvent('keydown', { key: 'Escape' });
                    document.dispatchEvent(event);
                }
            }}
            className={cn("p-2 rounded-lg hover:bg-muted transition-colors", className)}
        >
            <X size={20} />
        </button>
    );
}

interface DialogContentProps {
    children: React.ReactNode;
    className?: string;
}

export function DialogContent({ children, className }: DialogContentProps) {
    return <div className={cn("mt-4", className)}>{children}</div>;
}

interface DialogFooterProps {
    children: React.ReactNode;
    className?: string;
}

export function DialogFooter({ children, className }: DialogFooterProps) {
    return <div className={cn("flex justify-end gap-2 mt-6 pt-4 border-t", className)}>{children}</div>;
}
