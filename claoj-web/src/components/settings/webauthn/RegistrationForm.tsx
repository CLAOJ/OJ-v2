'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { webauthnApi } from '@/lib/api';
import { creationOptions, credentialJSON } from '@/lib/webauthn';
import { Loader2, Key } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Alert, AlertDescription } from '@/components/ui/Alert';
import type { ApiError } from '@/lib/api';

interface WebAuthnRegistrationFormProps {
    onSuccess?: () => void;
}

export function WebAuthnRegistrationForm({ onSuccess }: WebAuthnRegistrationFormProps) {
    const t = useTranslations('Settings');
    const tc = useTranslations('Common');
    const [credentialName, setCredentialName] = useState('');
    const [password, setPassword] = useState('');
    const [isRegistering, setIsRegistering] = useState(false);
    const queryClient = useQueryClient();

    const registerMutation = useMutation({
        mutationFn: async () => {
            const beginRes = await webauthnApi.beginRegistration(password);
            const credential = await navigator.credentials.create({
                publicKey: creationOptions(beginRes.data.options),
            }) as PublicKeyCredential;
            return await webauthnApi.finishRegistration(credentialJSON(credential), credentialName);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['webauthn-status'] });
            queryClient.invalidateQueries({ queryKey: ['webauthn-credentials'] });
            setIsRegistering(false);
            setCredentialName('');
            setPassword('');
            onSuccess?.();
        },
        onError: (error: ApiError) => {
            alert(error.response?.data?.error || t('registrationFailedError'));
        },
    });

    const handleRegister = () => {
        if (!credentialName || !password) {
            alert(t('enterCredentialNameAndPassword'));
            return;
        }
        setIsRegistering(true);
        registerMutation.mutate();
    };

    if (!isRegistering) {
        return (
            <div className="space-y-4">
                <Alert>
                    <AlertDescription>
                        {t('webauthnNotEnabled')}
                    </AlertDescription>
                </Alert>
                <Button onClick={() => setIsRegistering(true)}>
                    <Key className="h-4 w-4 mr-2" />
                    {t('registerWebauthnCredential')}
                </Button>
            </div>
        );
    }

    return (
        <div className="space-y-4 p-4 border rounded-lg bg-muted/50">
            <h4 className="font-medium">{t('registerNewCredential')}</h4>
            <div className="space-y-3">
                <Input
                    placeholder={t('credentialNamePlaceholder')}
                    value={credentialName}
                    onChange={(e) => setCredentialName(e.target.value)}
                />
                <Input
                    type="password"
                    placeholder={t('confirmPasswordPlaceholder')}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                />
                <div className="flex gap-2">
                    <Button
                        onClick={handleRegister}
                        disabled={registerMutation.isPending || !credentialName || !password}
                    >
                        {registerMutation.isPending && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                        {t('register')}
                    </Button>
                    <Button
                        variant="outline"
                        onClick={() => {
                            setIsRegistering(false);
                            setCredentialName('');
                            setPassword('');
                        }}
                    >
                        {tc('cancel')}
                    </Button>
                </div>
            </div>
        </div>
    );
}
