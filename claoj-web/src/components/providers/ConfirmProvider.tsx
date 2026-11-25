'use client';

import { createContext, useCallback, useContext, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { ConfirmDialog, type ConfirmVariant } from '@/components/ui/ConfirmDialog';

type ConfirmOptions = { variant?: ConfirmVariant; title?: string; confirmText?: string };
type ConfirmFn = (message: string, options?: ConfirmOptions) => Promise<boolean>;

const ConfirmContext = createContext<ConfirmFn | null>(null);

/**
 * App-styled replacement for window.confirm(): `if (await confirm(msg)) ...`.
 * One dialog for the whole app, so every confirmation looks and behaves the same.
 */
export function ConfirmProvider({ children }: { children: React.ReactNode }) {
    const t = useTranslations('Common');
    const [state, setState] = useState<{ message: string; options: ConfirmOptions } | null>(null);
    const resolver = useRef<((ok: boolean) => void) | null>(null);

    const confirm = useCallback<ConfirmFn>((message, options = {}) => {
        resolver.current?.(false);
        setState({ message, options });
        return new Promise<boolean>(resolve => { resolver.current = resolve; });
    }, []);

    const settle = (ok: boolean) => {
        resolver.current?.(ok);
        resolver.current = null;
        setState(null);
    };

    return (
        <ConfirmContext.Provider value={confirm}>
            {children}
            <ConfirmDialog
                isOpen={state !== null}
                onClose={() => settle(false)}
                onConfirm={() => settle(true)}
                title={state?.options.title ?? t('confirm')}
                description={state?.message ?? ''}
                confirmText={state?.options.confirmText ?? t('confirm')}
                cancelText={t('cancel')}
                variant={state?.options.variant ?? 'warning'}
            />
        </ConfirmContext.Provider>
    );
}

export function useConfirm(): ConfirmFn {
    const ctx = useContext(ConfirmContext);
    if (!ctx) throw new Error('useConfirm must be used inside ConfirmProvider');
    return ctx;
}
