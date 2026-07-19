'use client';

import React from 'react';
import { useTranslations } from 'next-intl';
import { Info } from 'lucide-react';

export function DangerZone() {
    const t = useTranslations('Settings');

    return (
        <section className="space-y-4">
            <div className="flex items-center gap-2 text-destructive font-bold">
                <Info size={18} />
                {t('dangerZone')}
            </div>
            <p className="text-sm text-muted-foreground">
                {t('deleteAccountWarning')}
            </p>
            <button className="px-6 h-12 rounded-xl bg-destructive text-destructive-foreground font-bold hover:bg-destructive/90 transition-all flex items-center gap-2">
                {t('deleteAccount')}
            </button>
        </section>
    );
}
