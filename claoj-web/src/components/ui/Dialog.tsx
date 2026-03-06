'use client';

import React, { useEffect, useRef, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X } from 'lucide-react';
import { cn } from '@/lib/utils';

// Dialog Components - Compound component pattern
interface DialogContextType {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

const DialogContext = React.createContext<DialogContextType | undefined>(undefined);

interface DialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    children: React.ReactNode;
    className?: string;
}

export function Dialog({ open, onOpenChange, children, className }: DialogProps) {
    const dialogRef = useRef<HTMLDivElement>(null);
    const previousActiveElement = useRef<HTMLElement | null>(null);

    // Store previous focus and restore on close
    useEffect(() => {
        if (open) {
            previousActiveElement.current = document.activeElement as HTMLElement;
            setTimeout(() => {
                dialogRef.current?.focus();
            }, 0);
        } else {
            previousActiveElement.current?.focus();
        }
    }, [open]);

    // Handle escape key and focus trap
    useEffect(() => {
        if (!open) return;

        const handleEscape = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                onOpenChange(false);
            }
        };

        const handleTabKey = (e: KeyboardEvent) => {
            if (e.key !== 'Tab') return;

            const dialog = dialogRef.current;
            if (!dialog) return;

            const focusableElements = dialog.querySelectorAll<HTMLElement>(
                'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"]), [role="button"]'
            );

            if (focusableElements.length === 0) return;

            const firstEl = focusableElements[0];
            const lastEl = focusableElements[focusableElements.length - 1];

            if (e.shiftKey && document.activeElement === firstEl) {
                e.preventDefault();
                lastEl.focus();
            } else if (!e.shiftKey && document.activeElement === lastEl) {
                e.preventDefault();
                firstEl.focus();
            }
        };

        document.addEventListener('keydown', handleEscape);
        document.addEventListener('keydown', handleTabKey);
        document.body.style.overflow = 'hidden';

        return () => {
            document.removeEventListener('keydown', handleEscape);
            document.removeEventListener('keydown', handleTabKey);
            document.body.style.overflow = 'unset';
        };
    }, [open, onOpenChange]);

    const handleBackdropClick = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
        if (e.target === e.currentTarget) {
            onOpenChange(false);
        }
    }, [onOpenChange]);

    return (
        <DialogContext.Provider value={{ open, onOpenChange }}>
            <AnimatePresence>
                {open && (
                    <>
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            onClick={handleBackdropClick}
                            className="fixed inset-0 bg-black/50 z-50"
                            aria-hidden="true"
                        />
                        <motion.div
                            ref={dialogRef}
                            role="dialog"
                            aria-modal="true"
                            tabIndex={-1}
                            initial={{ opacity: 0, scale: 0.95, y: 20 }}
                            animate={{ opacity: 1, scale: 1, y: 0 }}
                            exit={{ opacity: 0, scale: 0.95, y: 20 }}
                            className={cn(
                                "fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[90vw] -translate-x-1/2 -translate-y-1/2 overflow-auto rounded-2xl bg-popover p-6 shadow-lg outline-none",
                                className
                            )}
                            onClick={(e) => e.stopPropagation()}
                        >
                            {children}
                        </motion.div>
                    </>
                )}
            </AnimatePresence>
        </DialogContext.Provider>
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
    id?: string;
}

export function DialogTitle({ children, className, id }: DialogTitleProps) {
    return <h2 id={id} className={cn("text-xl font-bold", className)}>{children}</h2>;
}

interface DialogDescriptionProps {
    children: React.ReactNode;
    className?: string;
    id?: string;
}

export function DialogDescription({ children, className, id }: DialogDescriptionProps) {
    return <p id={id} className={cn("text-sm text-muted-foreground mt-1", className)}>{children}</p>;
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

interface DialogCloseProps {
    className?: string;
    ariaLabel?: string;
}

export function DialogClose({ className, ariaLabel }: DialogCloseProps) {
    return (
        <button
            onClick={(e) => {
                e.stopPropagation();
                e.currentTarget.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
            }}
            className={cn("p-2 rounded-lg hover:bg-muted transition-colors", className)}
            aria-label={ariaLabel || 'Close dialog'}
        >
            <X size={20} />
        </button>
    );
}

interface DialogTriggerProps {
    children: React.ReactNode;
    className?: string;
    asChild?: boolean;
}

export function DialogTrigger({ children, className, asChild }: DialogTriggerProps) {
    const { onOpenChange } = React.useContext(DialogContext) || {};
    
    if (asChild) {
        // Return children directly to be handled by parent
        return <>{children}</>;
    }
    
    return (
        <div 
            onClick={() => onOpenChange?.(true)}
            className={cn("cursor-pointer", className)}
        >
            {children}
        </div>
    );
}
