'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { useAuth } from '@/components/providers/AuthProvider';
import { webauthnApi } from '@/lib/api';
import { Shield, Loader2 } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Alert, AlertDescription } from '@/components/ui/Alert';
import { useQuery } from '@tanstack/react-query';
import { WebAuthnRegistrationForm } from './webauthn/RegistrationForm';
import { CredentialList } from './webauthn/CredentialList';

export default function WebAuthnSettings() {
    const t = useTranslations();
    const { user } = useAuth();

    const { data: status } = useQuery({
        queryKey: ['webauthn-status'],
        queryFn: async () => {
            const res = await webauthnApi.getStatus();
            return res.data;
        },
    });

    const { data: credentials } = useQuery({
        queryKey: ['webauthn-credentials'],
        queryFn: async () => {
            const res = await webauthnApi.getCredentials();
            return res.data.credentials;
        },
        enabled: status?.enabled,
    });

    if (!status) {
        return (
            <div className="flex items-center justify-center p-8">
                <Loader2 className="h-8 w-8 animate-spin" />
            </div>
        );
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle className="flex items-center gap-2">
                    <Shield className="h-5 w-5" />
                    WebAuthn / Passkey Authentication
                </CardTitle>
                <CardDescription>
                    Manage your WebAuthn credentials for passwordless login
                </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
                {!status.enabled ? (
                    <WebAuthnRegistrationForm />
                ) : (
                    <div className="space-y-4">
                        <Alert variant="default">
                            <AlertDescription>
                                WebAuthn is enabled on your account. You can use your registered credentials to log in without a password.
                            </AlertDescription>
                        </Alert>

                        {credentials && credentials.length > 0 && (
                            <CredentialList credentials={credentials} />
                        )}
                    </div>
                )}
            </CardContent>
        </Card>
    );
}
