'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { webauthnApi } from '@/lib/api';
import { Trash2, Edit2, Check, X, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import type { WebAuthnCredential } from '@/types';

interface CredentialListProps {
    credentials: WebAuthnCredential[];
}

export function CredentialList({ credentials }: CredentialListProps) {
    const queryClient = useQueryClient();
    const [editingId, setEditingId] = useState<number | null>(null);
    const [newName, setNewName] = useState('');
    const [deletingId, setDeletingId] = useState<number | null>(null);
    const [deletePassword, setDeletePassword] = useState('');

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

    const handleUpdate = (id: number) => {
        if (!newName) {
            alert('Please enter a name');
            return;
        }
        updateMutation.mutate({ id: id.toString(), name: newName });
    };

    const handleDelete = (id: number) => {
        if (!deletePassword) {
            alert('Please enter your password');
            return;
        }
        setDeletingId(id);
        deleteMutation.mutate({ id: id.toString(), password: deletePassword });
    };

    return (
        <div className="space-y-4">
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
    );
}
