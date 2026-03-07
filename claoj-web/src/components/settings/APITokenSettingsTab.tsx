'use client';

import { useState } from 'react';
import api from '@/lib/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Loader2, CheckCircle, Copy, Shield, Info } from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { cn } from '@/lib/utils';

export default function APITokenSettingsTab() {
    const queryClient = useQueryClient();
    const [showToken, setShowToken] = useState(false);
    const [generatedToken, setGeneratedToken] = useState<string | null>(null);

    const { data: tokenInfo, isLoading } = useQuery({
        queryKey: ['api-token'],
        queryFn: async () => {
            const res = await api.get('/user/api-token');
            return res.data;
        },
    });

    const { mutate: generateToken, isPending: isGenerating } = useMutation({
        mutationFn: async () => {
            const res = await api.post('/user/api-token');
            return res.data;
        },
        onSuccess: (data: unknown) => {
            queryClient.invalidateQueries({ queryKey: ['api-token'] });
            const typedData = data as { token?: string };
            if (typedData.token) {
                setGeneratedToken(typedData.token);
                navigator.clipboard.writeText(typedData.token);
            }
        },
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Failed to generate API token');
        },
    });

    const { mutate: revokeToken, isPending: isRevoking } = useMutation({
        mutationFn: async () => {
            const res = await api.delete('/user/api-token');
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['api-token'] });
            setGeneratedToken(null);
            setShowToken(false);
        },
        onError: (err: unknown) => {
            const error = err as { response?: { data?: { error?: string } } };
            alert(error.response?.data?.error || 'Failed to revoke API token');
        },
    });

    const handleGenerate = () => {
        generateToken(undefined, {
            onSuccess: (data: unknown) => {
                const typedData = data as { token?: string };
                if (typedData.token) {
                    setGeneratedToken(typedData.token);
                    setShowToken(true);
                }
            },
        });
    };

    if (isLoading) {
        return <div className="flex items-center justify-center py-12"><Loader2 size={24} className="animate-spin text-muted-foreground" /></div>;
    }

    if (tokenInfo?.has_token && !generatedToken) {
        return (
            <div className="p-6 rounded-2xl border bg-card space-y-4">
                <div className="flex items-center gap-2 text-amber-600">
                    <Info size={20} />
                    <span className="font-bold text-sm">Existing Token</span>
                </div>
                <p className="text-sm text-muted-foreground">
                    You already have an API token. For security reasons, you cannot view it again.
                    Generate a new token if you&apos;ve lost yours (this will invalidate the old token).
                </p>
                <button
                    onClick={() => revokeToken()}
                    disabled={isRevoking}
                    className="px-6 py-3 rounded-xl bg-destructive text-destructive-foreground font-bold text-sm hover:bg-destructive/90 transition-colors disabled:opacity-50"
                >
                    {isRevoking ? 'Revoking...' : 'Revoke Token'}
                </button>
            </div>
        );
    }

    if (generatedToken) {
        return (
            <div className="p-6 rounded-2xl border bg-card space-y-4">
                <div className="flex items-center gap-2 text-green-600">
                    <CheckCircle size={20} />
                    <span className="font-bold text-sm">Token Generated</span>
                </div>
                <div className="p-4 bg-muted rounded-xl">
                    <code className="text-xs break-all">{generatedToken}</code>
                </div>
                <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-700 text-sm">
                    <p className="font-medium flex items-center gap-2">
                        <Info size={16} />
                        Important: Save this token now
                    </p>
                    <p className="mt-1">
                        For security reasons, the token will not be shown again. Store it in a secure location.
                    </p>
                </div>
                <button
                    onClick={() => {
                        setShowToken(false);
                        setGeneratedToken(null);
                    }}
                    className="px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold text-sm hover:bg-primary/90 transition-colors"
                >
                    Done
                </button>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="space-y-2">
                <h2 className="text-2xl font-bold">API Token</h2>
                <p className="text-muted-foreground text-sm">
                    Generate a token to access your account programmatically.
                </p>
            </div>

            <div className="p-6 rounded-2xl border bg-card">
                <button
                    onClick={handleGenerate}
                    disabled={isGenerating}
                    className="w-full px-6 py-3 rounded-xl bg-primary text-primary-foreground font-bold text-sm hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                >
                    {isGenerating && <Loader2 size={16} className="animate-spin" />}
                    Generate API Token
                </button>
            </div>

            <div className="p-4 rounded-xl bg-blue-500/10 border border-blue-500/20 text-blue-600 text-sm space-y-2">
                <p className="font-medium flex items-center gap-2">
                    <Shield size={16} />
                    How to use your API token
                </p>
                <p>Include the token in the Authorization header of your API requests:</p>
                <pre className="bg-muted p-3 rounded-lg text-xs overflow-x-auto">
                    <code>Authorization: Bearer YOUR_API_TOKEN</code>
                </pre>
            </div>
        </div>
    );
}
