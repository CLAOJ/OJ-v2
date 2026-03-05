'use client';

import React, { createContext, useContext, useEffect, useState } from 'react';
import api from '@/lib/api';
import { User } from '@/types';

interface AuthContextType {
    user: User | null;
    loading: boolean;
    login: (username: string, password: string) => Promise<LoginResponse>;
    loginTotp: (username: string, code: string) => Promise<void>;
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
                const res = await api.get('/user/me');
                if (res.status === 200) {
                    setUser(res.data.user);
                }
            } catch (err) {
                // Not logged in or token expired
                console.log('Not authenticated');
            }
            setLoading(false);
        };

        checkAuth();
    }, []);

    return (
        <AuthContext.Provider value={{ user, loading, login, logout }}>
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
