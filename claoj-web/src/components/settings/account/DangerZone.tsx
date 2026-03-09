'use client';

import React from 'react';
import { Info } from 'lucide-react';

export function DangerZone() {
    return (
        <section className="space-y-4">
            <div className="flex items-center gap-2 text-destructive font-bold">
                <Info size={18} />
                Danger Zone
            </div>
            <p className="text-sm text-muted-foreground">
                Once you delete your account, there is no going back. This action is permanent and irreversible.
            </p>
            <button className="px-6 h-12 rounded-xl bg-destructive text-destructive-foreground font-bold hover:bg-destructive/90 transition-all flex items-center gap-2">
                Delete Account
            </button>
        </section>
    );
}
