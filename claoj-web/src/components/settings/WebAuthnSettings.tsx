'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { useAuth } from '@/components/providers/AuthProvider';
import { webauthnApi } from '@/lib/api';
import { Loader2, Key, Shield, Trash2, Edit2, Check, X } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Alert, AlertDescription } from '@/components/ui/Alert';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { WebAuthnCredential } from '@/types';

export default function WebAuthnSettings() {
    const t = useTranslations();
    const { user } = useAuth();
    const queryClient = useQueryClient();
    const [isRegistering, setIsRegistering] = useState(false);
    const [credentialName, setCredentialName] = useState('');
    const [password, setPassword] = useState('');
    const [editingId, setEditingId] = useState<number | null>(null);
    const [newName, setNewName] = useState('');
    const [deletePassword, setDeletePassword] = useState('');
    const [deletingId, setDeletingId] = useState<number | null>(null);

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

    const registerMutation = useMutation({
        mutationFn: async () => {
            // Begin registration
            const beginRes = await webauthnApi.beginRegistration(password);
            const options = beginRes.data.options;

            // Convert options to use ArrayBuffer for credential IDs
            const publicKeyOptions: PublicKeyCredentialCreationOptions = {
                ...options,
                challenge: typeof options.challenge === 'string'
                    ? Uint8Array.from(atob(options.challenge), (b) => b.charCodeAt(0))
                    : new Uint8Array(options.challenge as ArrayBuffer),
                user: {
                    ...options.user,
                    id: typeof options.user.id === 'string'
                        ? Uint8Array.from(atob(options.user.id), (b) => b.charCodeAt(0))
                        : new Uint8Array(options.user.id as ArrayBuffer),
                },
            };

            // Create credential
            const credential = await navigator.credentials.create({
                publicKey: publicKeyOptions,
            }) as PublicKeyCredential;

            // Convert response to JSON
            const response = {
                id: credential.id,
                rawId: Array.from(new Uint8Array(credential.rawId)),
                type: credential.type,
                clientExtensionResults: credential.getClientExtensionResults(),
                response: {
                    attestationObject: Array.from(
                        new Uint8Array((credential.response as AuthenticatorAttestationResponse).attestationObject)
                    ),
                    clientDataJSON: Array.from(
                        new Uint8Array((credential.response as AuthenticatorAttestationResponse).clientDataJSON)
                    ),
                },
            };

            // Finish registration
            return await webauthnApi.finishRegistration(response, credentialName);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['webauthn-status'] });
            queryClient.invalidateQueries({ queryKey: ['webauthn-credentials'] });
            setIsRegistering(false);
            setCredentialName('');
            setPassword('');
        },
        onError: (error: any) => {
            alert(error.response?.data?.error || 'Registration failed');
        },
    });

    const deleteMutation = useMutation({
        mutationFn: async ({ id, password }: { id: string; password: string }) => {
            return await webauthnApi.deleteCredential(id, password);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['webauthn-status'] });
            queryClient.invalidateQueries({ queryKey: ['webauthn-credentials'] });
            setDeletingId(null);
            setDeletePassword('');
        },
        onError: (error: any) => {
            alert(error.response?.data?.error || 'Delete failed');
        },
    });

    const updateMutation = useMutation({
        mutationFn: async ({ id, name }: { id: string; name: string }) => {
            return await webauthnApi.updateCredential(id, name);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['webauthn-credentials'] });
            setEditingId(null);
            setNewName('');
        },
    });

    const handleRegister = () => {
        if (!credentialName || !password) {
            alert('Please enter a credential name and password');
            return;
        }
        setIsRegistering(true);
        registerMutation.mutate();
    };

    const handleDelete = (id: number) => {
        if (!deletePassword) {
            alert('Please enter your password');
            return;
        }
        setDeletingId(id);
        deleteMutation.mutate({ id: id.toString(), password: deletePassword });
    };

    const handleUpdate = (id: number) => {
        if (!newName) {
            alert('Please enter a name');
            return;
        }
        updateMutation.mutate({ id: id.toString(), name: newName });
    };

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
                    <div className="space-y-4">
                        <Alert>
                            <AlertDescription>
                                WebAuthn is not enabled on your account. Register a credential to enable passwordless login.
                            </AlertDescription>
                        </Alert>

                        {!isRegistering ? (
                            <div className="space-y-4">
                                <Button onClick={() => setIsRegistering(true)}>
                                    <Key className="h-4 w-4 mr-2" />
                                    Register WebAuthn Credential
                                </Button>
                            </div>
                        ) : (
                            <div className="space-y-4 p-4 border rounded-lg bg-muted/50">
                                <h4 className="font-medium">Register New Credential</h4>
                                <div className="space-y-3">
                                    <Input
                                        placeholder="Credential name (e.g., 'My YubiKey')"
                                        value={credentialName}
                                        onChange={(e) => setCredentialName(e.target.value)}
                                    />
                                    <Input
                                        type="password"
                                        placeholder="Enter your password to confirm"
                                        value={password}
                                        onChange={(e) => setPassword(e.target.value)}
                                    />
                                    <div className="flex gap-2">
                                        <Button
                                            onClick={handleRegister}
                                            disabled={registerMutation.isPending || !credentialName || !password}
                                        >
                                            {registerMutation.isPending && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                                            Register
                                        </Button>
                                        <Button
                                            variant="outline"
                                            onClick={() => {
                                                setIsRegistering(false);
                                                setCredentialName('');
                                                setPassword('');
                                            }}
                                        >
                                            Cancel
                                        </Button>
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>
                ) : (
                    <div className="space-y-4">
                        <Alert variant="default">
                            <AlertDescription>
                                WebAuthn is enabled on your account. You can use your registered credentials to log in without a password.
                            </AlertDescription>
                        </Alert>

                        {credentials && credentials.length > 0 && (
                            <div className="space-y-2">
                                <h4 className="font-medium">Registered Credentials</h4>
                                {credentials.map((cred) => (
                                    <div
                                        key={cred.id}
                                        className="flex items-center justify-between p-3 border rounded-lg"
                                    >
                                        {editingId === cred.id ? (
                                            <div className="flex items-center gap-2 flex-1">
                                                <Input
                                                    value={newName}
                                                    onChange={(e) => setNewName(e.target.value)}
                                                    className="flex-1"
                                                />
                                                <Button
                                                    size="sm"
                                                    onClick={() => handleUpdate(cred.id)}
                                                    disabled={updateMutation.isPending}
                                                >
                                                    <Check className="h-4 w-4" />
                                                </Button>
                                                <Button
                                                    size="sm"
                                                    variant="outline"
                                                    onClick={() => {
                                                        setEditingId(null);
                                                        setNewName('');
                                                    }}
                                                >
                                                    <X className="h-4 w-4" />
                                                </Button>
                                            </div>
                                        ) : (
                                            <>
                                                <div>
                                                    <p className="font-medium">{cred.name}</p>
                                                    <p className="text-sm text-muted-foreground">
                                                        ID: {cred.cred_id.slice(0, 20)}...
                                                    </p>
                                                </div>
                                                <div className="flex gap-2">
                                                    <Button
                                                        size="sm"
                                                        variant="outline"
                                                        onClick={() => {
                                                            setEditingId(cred.id);
                                                            setNewName(cred.name);
                                                        }}
                                                    >
                                                        <Edit2 className="h-4 w-4" />
                                                    </Button>
                                                </div>
                                            </>
                                        )}
                                    </div>
                                ))}
                            </div>
                        )}

                        {deletingId !== null ? (
                            <div className="space-y-3 p-4 border rounded-lg bg-destructive/10">
                                <p className="text-sm font-medium">Enter your password to confirm deletion:</p>
                                <Input
                                    type="password"
                                    value={deletePassword}
                                    onChange={(e) => setDeletePassword(e.target.value)}
                                />
                                <div className="flex gap-2">
                                    <Button
                                        variant="destructive"
                                        onClick={() => handleDelete(deletingId)}
                                        disabled={deleteMutation.isPending}
                                    >
                                        {deleteMutation.isPending && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                                        Confirm Delete
                                    </Button>
                                    <Button
                                        variant="outline"
                                        onClick={() => {
                                            setDeletingId(null);
                                            setDeletePassword('');
                                        }}
                                    >
                                        Cancel
                                    </Button>
                                </div>
                            </div>
                        ) : (
                            <Button
                                variant="destructive"
                                onClick={() => setDeletingId(credentials?.[0]?.id ?? null)}
                            >
                                <Trash2 className="h-4 w-4 mr-2" />
                                Delete Credential
                            </Button>
                        )}
                    </div>
                )}
            </CardContent>
        </Card>
    );
}
