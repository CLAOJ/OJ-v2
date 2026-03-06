'use client';

import React, { createContext, useContext, useEffect, useState } from 'react';
import api, { webauthnApi } from '@/lib/api';
import { User } from '@/types';

interface AuthContextType {
    user: User | null;
    loading: boolean;
    login: (username: string, password: string) => Promise<LoginResponse>;
    loginTotp: (username: string, code: string) => Promise<void>;
    loginWebAuthn: (username: string) => Promise<void>;
    logout: () => Promise<void>;
}

interface LoginResponse {
    requiresTotp?: boolean;
    username?: string;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);

    const login = async (username: string, password: string): Promise<LoginResponse> => {
        const res = await api.post('/auth/login', { username, password });
        const { requires_totp, requiresTotp } = res.data;

        // Check if TOTP is required
        if (requires_totp || requiresTotp) {
            return { requiresTotp: true, username: res.data.username };
        }

        // Tokens are set via httpOnly cookies by the server - no localStorage storage
        setUser(res.data.user);
        return {};
    };

    const loginTotp = async (username: string, code: string) => {
        const res = await api.post('/auth/totp/verify', { username, code });
        // Tokens are set via httpOnly cookies by the server - no localStorage storage
        setUser(res.data.user);
    };

    const loginWebAuthn = async (username: string) => {
        // Begin login
        const beginRes = await webauthnApi.beginLogin(username);
        const options = beginRes.data.options;

        // Convert options to use ArrayBuffer for credential IDs
        const publicKeyOptions: PublicKeyCredentialRequestOptions = {
            ...options,
            challenge: typeof options.challenge === 'string'
                ? Uint8Array.from(atob(options.challenge), (b) => b.charCodeAt(0))
                : new Uint8Array(options.challenge as ArrayBuffer),
            allowCredentials: options.allowCredentials?.map((cred: any) => ({
                ...cred,
                id: Uint8Array.from(cred.id, (b: number) => b),
            })),
        };

        // Get credential from authenticator
        const credential = await navigator.credentials.get({
            publicKey: publicKeyOptions,
        }) as PublicKeyCredential;

        // Convert response to JSON
        const response = {
            id: credential.id,
            rawId: Array.from(new Uint8Array(credential.rawId)),
            type: credential.type,
            clientExtensionResults: credential.getClientExtensionResults(),
            response: {
                authenticatorData: Array.from(
                    new Uint8Array((credential.response as AuthenticatorAssertionResponse).authenticatorData)
                ),
                clientDataJSON: Array.from(
                    new Uint8Array((credential.response as AuthenticatorAssertionResponse).clientDataJSON)
                ),
                signature: Array.from(
                    new Uint8Array((credential.response as AuthenticatorAssertionResponse).signature)
                ),
                userHandle: (credential.response as AuthenticatorAssertionResponse).userHandle
                    ? Array.from(new Uint8Array((credential.response as AuthenticatorAssertionResponse).userHandle as ArrayBuffer))
                    : null,
            },
        };

        // Finish login
        const finishRes = await webauthnApi.finishLogin(response);
        // Tokens are set via httpOnly cookies by the server
        setUser(finishRes.data.user);
    };

    const logout = async () => {
        // Call logout endpoint to revoke tokens and clear cookies
        try {
            await api.post('/auth/logout');
        } catch (err) {
            console.error('Logout error', err);
        }
        setUser(null);
    };

    useEffect(() => {
        const checkAuth = async () => {
            if (typeof window === 'undefined') {
                setLoading(false);
                return;
            }

            // Auth is handled entirely via httpOnly cookies
            // No need to check localStorage - tokens are automatically sent with requests
            try {
                const res = await api.get('/user/me', { _skipAuthRedirect: true } as any);
                if (res.status === 200) {
                    setUser(res.data.user);
                }
            } catch (err) {
                // Not logged in or token expired - silently handle
            }
            setLoading(false);
        };

        checkAuth();
    }, []);

    return (
        <AuthContext.Provider value={{ user, loading, login, loginTotp, loginWebAuthn, logout }}>
            {children}
        </AuthContext.Provider>
    );
}

export const useAuth = () => {
    const context = useContext(AuthContext);
    if (context === undefined) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
};
