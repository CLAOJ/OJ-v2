'use client';

import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
} from './Dialog';
import { Button } from './Button';
import { Loader2, AlertTriangle, Trash2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useEffect, useState } from 'react';

export type ConfirmVariant = 'danger' | 'warning' | 'info';

interface ConfirmDialogProps {
    isOpen: boolean;
    onClose: () => void;
    onConfirm: () => void;
    title: string;
    description: string;
    confirmText?: string;
    cancelText?: string;
    variant?: ConfirmVariant;
    isLoading?: boolean;
    children?: React.ReactNode;
    /** When set, confirm stays disabled until this exact text is typed. */
    confirmPhrase?: string;
    /** Label above the phrase input, e.g. "Type dct05 to confirm". */
    confirmPhraseLabel?: string;
}

const variantConfig = {
    danger: {
        icon: Trash2,
        iconColor: 'text-destructive',
        buttonVariant: 'destructive' as const,
    },
    warning: {
        icon: AlertTriangle,
        iconColor: 'text-amber-500',
        buttonVariant: 'default' as const,
    },
    info: {
        icon: AlertTriangle,
        iconColor: 'text-blue-500',
        buttonVariant: 'default' as const,
    },
};

/**
 * Reusable confirmation dialog component
 *
 * @example
 * // Delete confirmation
 * <ConfirmDialog
 *   isOpen={deleteConfirmId !== null}
 *   onClose={() => setDeleteConfirmId(null)}
 *   onConfirm={handleDelete}
 *   title="Delete User"
 *   description="Are you sure you want to delete this user? This action cannot be undone."
 *   variant="danger"
 *   isLoading={deleteMutation.isPending}
 * />
 */
export function ConfirmDialog({
    isOpen,
    onClose,
    onConfirm,
    title,
    description,
    confirmText = 'Confirm',
    cancelText = 'Cancel',
    variant = 'danger',
    isLoading = false,
    children,
    confirmPhrase,
    confirmPhraseLabel,
}: ConfirmDialogProps) {
    const config = variantConfig[variant];
    const Icon = config.icon;
    const [typed, setTyped] = useState('');
    // A fresh dialog (or a new target) starts empty.
    useEffect(() => { setTyped(''); }, [isOpen, confirmPhrase]);
    const phraseOk = !confirmPhrase || typed.trim() === confirmPhrase;

    return (
        <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader className="gap-4">
                    <div className="flex items-center gap-3">
                        <div className={cn(
                            'p-2 rounded-full bg-muted',
                            config.iconColor
                        )}>
                            <Icon size={20} />
                        </div>
                        <DialogTitle>{title}</DialogTitle>
                    </div>
                    <DialogDescription>{description}</DialogDescription>
                </DialogHeader>

                {children && (
                    <div className="py-4">{children}</div>
                )}

                {confirmPhrase && (
                    <label className="flex flex-col gap-2 text-sm">
                        <span className="text-muted-foreground">
                            {confirmPhraseLabel ?? confirmPhrase}
                        </span>
                        <input
                            value={typed}
                            onChange={e => setTyped(e.target.value)}
                            onKeyDown={e => { if (e.key === 'Enter' && phraseOk && !isLoading) onConfirm(); }}
                            autoFocus
                            autoComplete="off"
                            spellCheck={false}
                            className="h-10 rounded-md border bg-background px-3 font-mono text-sm focus:outline-none focus:ring-2 focus:ring-destructive/40"
                        />
                    </label>
                )}

                <DialogFooter className="gap-2 sm:gap-0">
                    <Button
                        variant="outline"
                        onClick={onClose}
                        disabled={isLoading}
                    >
                        {cancelText}
                    </Button>
                    <Button
                        variant={config.buttonVariant}
                        onClick={onConfirm}
                        disabled={isLoading || !phraseOk}
                    >
                        {isLoading && (
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        )}
                        {confirmText}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

interface DeleteConfirmDialogProps extends Omit<ConfirmDialogProps, 'variant' | 'confirmText'> {
    itemName?: string;
    itemType?: string;
}

/**
 * Specialized delete confirmation dialog
 *
 * @example
 * <DeleteConfirmDialog
 *   isOpen={deleteConfirmId !== null}
 *   onClose={() => setDeleteConfirmId(null)}
 *   onConfirm={handleDelete}
 *   itemType="user"
 *   itemName={selectedUser?.username}
 *   isLoading={deleteMutation.isPending}
 * />
 */
export function DeleteConfirmDialog({
    itemName,
    itemType = 'item',
    description,
    ...props
}: DeleteConfirmDialogProps) {
    const defaultDescription = itemName
        ? `Are you sure you want to delete "${itemName}"? This action cannot be undone.`
        : `Are you sure you want to delete this ${itemType}? This action cannot be undone.`;

    return (
        <ConfirmDialog
            {...props}
            title={`Delete ${itemType.charAt(0).toUpperCase() + itemType.slice(1)}`}
            description={description || defaultDescription}
            confirmText="Delete"
            variant="danger"
        />
    );
}
