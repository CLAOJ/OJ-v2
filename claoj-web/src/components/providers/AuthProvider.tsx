'use client';

import React, { createContext, useContext, useEffect, useState } from 'react';
import api from '@/lib/api';
import { User } from '@/types';

interface AuthContextType {
    user: User | null;
    loading: boolean;
    login: (username: string, password: string) => Promise<void>;
    logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);

    const login = async (username: string, password: string) => {
        const res = await api.post('/auth/login', { username, password });
        const { access_token, refresh_token, user: userData } = res.data;
        // Tokens are now set via httpOnly cookies by the server
        // Also update localStorage for backwards compatibility
        if (typeof window !== 'undefined') {
            localStorage.setItem('access_token', access_token);
            localStorage.setItem('refresh_token', refresh_token);
        }
        setUser(userData);
    };

    const logout = async () => {
        // Call logout endpoint to revoke tokens and clear cookies
        try {
            await api.post('/auth/logout');
        } catch (err) {
            console.error('Logout error', err);
        }
        if (typeof window !== 'undefined') {
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
        }
        setUser(null);
    };

    useEffect(() => {
        const checkAuth = async () => {
            if (typeof window === 'undefined') {
                setLoading(false);
                return;
            }
            // Try cookie first, fallback to localStorage
            const getCookie = (name: string): string | null => {
                const value = `; ${document.cookie}`;
                const parts = value.split(`; ${name}=`);
                if (parts.length === 2) {
                    return parts.pop()?.split(';').shift() || null;
                }
                return null;
            };

            const token = getCookie('access_token') || localStorage.getItem('access_token');
            if (token) {
                try {
                    const res = await api.get('/user/me');
                    if (res.status === 200) {
                        setUser(res.data.user);
                    }
                } catch (err) {
                    console.error('Initial auth check failed', err);
                    logout();
                }
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
