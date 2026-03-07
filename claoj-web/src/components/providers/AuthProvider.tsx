'use client';

import React, { createContext, useContext, useEffect, useState } from 'react';
import api, { webauthnApi } from '@/lib/api';
import axios from 'axios';
import { User } from '@/types';

interface AuthContextType {
    user: User | null;
    loading: boolean;
    login: (username: string, password: string, rememberMe?: boolean) => Promise<LoginResponse>;
    loginTotp: (username: string, code: string) => Promise<User>;
    loginWebAuthn: (username: string) => Promise<User>;
    logout: () => Promise<void>;
}

interface LoginResponse {
    requiresTotp?: boolean;
    username?: string;
    user?: User;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);

    const login = async (username: string, password: string, rememberMe?: boolean): Promise<LoginResponse> => {
        const res = await api.post('/auth/login', { username, password, remember_me: rememberMe });
        const { requires_totp, requiresTotp } = res.data;

        // Check if TOTP is required
        if (requires_totp || requiresTotp) {
            return { requiresTotp: true, username: res.data.username };
        }

        // Tokens are set via httpOnly cookies by the server - no localStorage storage
        const userData = res.data.user as User;
        setUser(userData);
        return { user: userData };
    };

    const loginTotp = async (username: string, code: string): Promise<User> => {
        const res = await api.post('/auth/totp/verify', { username, code });
        // Tokens are set via httpOnly cookies by the server - no localStorage storage
        const userData = res.data.user as User;
        setUser(userData);
        return userData;
    };

    const loginWebAuthn = async (username: string): Promise<User> => {
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
        const userData = finishRes.data.user as User;
        setUser(userData);
        return userData;
    };

    const logout = async () => {
        // Call logout endpoint to revoke tokens and clear cookies
        try {
            await api.post('/auth/logout');
        } catch (err) {
            // Logout error - continue to clear user state anyway
        }
        setUser(null);
    };

    useEffect(() => {
        let isSubscribed = true;

        const checkAuth = async () => {
            if (typeof window === 'undefined') {
                setLoading(false);
                return;
            }

            try {
                const res = await api.get('/user/me', { _skipAuthRedirect: true } as any);
                // If we get here, the request succeeded (either directly or after refresh+retry)
                if (res.status === 200 && res.data && isSubscribed) {
                    // /user/me returns user directly in res.data, not wrapped in res.data.user
                    setUser(res.data as User);
                }
            } catch (err) {
                // Axios errors: distinguish between network errors and HTTP errors
                // HTTP 401 errors are handled by the interceptor (refresh + retry)
                // If refresh succeeds, the retried request returns user data above
                // If refresh fails with _skipAuthRedirect, we still get here
                if (axios.isAxiosError(err)) {
                    if (err.code === 'ERR_NETWORK' || !err.response) {
                        // Network error - silently fail, user might be offline
                    } else if (err.response?.status === 401) {
                        // 401 after refresh attempt - user is not authenticated
                        // Don't redirect (due to _skipAuthRedirect), just don't set user
                    }
                    // Other HTTP errors are silently handled
                }
                // Non-Axios error - silently handled
            } finally {
                if (isSubscribed) {
                    setLoading(false);
                }
            }
        };

        checkAuth();

        // Periodic refresh every 10 minutes to keep tokens alive
        const refreshInterval = setInterval(() => {
            if (user) {
                api.post('/auth/refresh', {}, { withCredentials: true })
                    .catch(() => { /* Silent fail - token will be cleared on next API call if truly invalid */ });
            }
        }, 10 * 60 * 1000);

        return () => {
            isSubscribed = false;
            clearInterval(refreshInterval);
        };
    }, [user]);

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
