'use client';

import React, { createContext, useContext, useEffect, useState } from 'react';
import api, { webauthnApi } from '@/lib/api';
import axios, { AxiosRequestConfig } from 'axios';
import { User } from '@/types';

// Extended axios config for internal options
interface ApiRequestConfig extends AxiosRequestConfig {
    _skipAuthRedirect?: boolean;
}

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
        // Broadcast auth state change for other tabs/components
        if (typeof window !== 'undefined') {
            window.dispatchEvent(new Event('auth-state-change'));
        }
        return { user: userData };
    };

    const loginTotp = async (username: string, code: string): Promise<User> => {
        const res = await api.post('/auth/totp/verify', { username, code });
        // Tokens are set via httpOnly cookies by the server - no localStorage storage
        const userData = res.data.user as User;
        setUser(userData);
        // Broadcast auth state change for other tabs/components
        if (typeof window !== 'undefined') {
            window.dispatchEvent(new Event('auth-state-change'));
        }
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
            allowCredentials: options.allowCredentials?.map((cred: PublicKeyCredentialDescriptor) => ({
                ...cred,
                id: Uint8Array.from(cred.id as unknown as number[], (b: number) => b),
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
        // Broadcast auth state change for other tabs/components
        if (typeof window !== 'undefined') {
            window.dispatchEvent(new Event('auth-state-change'));
        }
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
        // Broadcast auth state change for other tabs/components
        if (typeof window !== 'undefined') {
            window.dispatchEvent(new Event('auth-state-change'));
        }
    };

    // Check auth status on mount and when auth-state-change event fires
    useEffect(() => {
        let isSubscribed = true;

        const checkAuth = async () => {
            if (typeof window === 'undefined') {
                setLoading(false);
                return;
            }

            try {
                // Try to get user info
                // If access token is invalid, the API interceptor will try to refresh
                // If refresh succeeds, the request will be retried automatically
                // If refresh fails, we'll get a 401 error here
                const res = await api.get('/user/me', { _skipAuthRedirect: true } as ApiRequestConfig);

                if (res.status === 200 && res.data && isSubscribed) {
                    // User is authenticated
                    setUser(res.data as User);
                }
            } catch (err) {
                // Handle different error scenarios
                if (axios.isAxiosError(err)) {
                    if (err.code === 'ERR_NETWORK' || !err.response) {
                        // Network error - keep current user state, user might be offline
                        // Don't clear user on network errors
                    } else if (err.response?.status === 401) {
                        // 401 means:
                        // 1. No access token AND no refresh token, OR
                        // 2. Access token invalid AND refresh token invalid/expired
                        //
                        // The API interceptor already tried to refresh if possible
                        // If we still get 401, clear user state - user is not authenticated
                        if (isSubscribed) {
                            setUser(null);
                        }
                    }
                    // Other HTTP errors (403, 500, etc.) - don't clear auth state
                }
                // Non-Axios errors - silently handled
            } finally {
                if (isSubscribed) {
                    setLoading(false);
                }
            }
        };

        checkAuth();

        // Listen for auth state changes from other components/tabs
        const handleAuthChange = () => {
            checkAuth();
        };
        window.addEventListener('auth-state-change', handleAuthChange);

        return () => {
            isSubscribed = false;
            window.removeEventListener('auth-state-change', handleAuthChange);
        };
    }, []);

    // Separate effect for periodic token refresh
    useEffect(() => {
        // Periodic refresh every 10 minutes to keep tokens alive
        const refreshInterval = setInterval(() => {
            if (user) {
                api.post('/auth/refresh', {}, { withCredentials: true })
                    .catch(() => { /* Silent fail - token will be cleared on next API call if truly invalid */ });
            }
        }, 10 * 60 * 1000);

        return () => clearInterval(refreshInterval);
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
