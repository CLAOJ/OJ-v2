'use client';

import React from 'react';
import { PasswordChangeForm } from './account/PasswordChangeForm';
import { EmailVerificationSection } from './account/EmailVerificationSection';
import { TwoFactorSection } from './account/TwoFactorSection';
import { DangerZone } from './account/DangerZone';

export default function AccountSettingsTab() {
    return (
        <div className="space-y-8">
            <PasswordChangeForm />
            <hr className="border-border" />
            <EmailVerificationSection />
            <hr className="border-border" />
            <TwoFactorSection />
            <hr className="border-destructive/50" />
            <DangerZone />
        </div>
    );
}
