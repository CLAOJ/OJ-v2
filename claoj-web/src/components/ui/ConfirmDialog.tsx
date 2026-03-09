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
}: ConfirmDialogProps) {
    const config = variantConfig[variant];
    const Icon = config.icon;

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
                        disabled={isLoading}
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
