import axios, { InternalAxiosRequestConfig } from 'axios';

// Extend Axios request config to include _retry property
interface AxiosRequestConfigWithRetry extends InternalAxiosRequestConfig {
    _retry?: boolean;
}

function getApiBaseUrl(): string {
    if (typeof window === 'undefined') return 'http://localhost:8080/api/v2';
    return `${window.location.origin}/api/v2`;
}

const api = axios.create({
    baseURL: process.env.NEXT_PUBLIC_API_URL || getApiBaseUrl(),
    headers: {
        'Content-Type': 'application/json',
    },
    withCredentials: true, // Enable sending cookies with requests
});

// Get API URL - uses env var if set, otherwise derives from window.location.origin
export function getApiUrl(): string {
    return api.defaults.baseURL as string;
}

// Interceptor to add access token from cookie or localStorage
api.interceptors.request.use((config) => {
    // Try to get token from cookie first, fallback to localStorage for backwards compatibility
    const getCookie = (name: string): string | null => {
        if (typeof window === 'undefined') return null;
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) {
            return parts.pop()?.split(';').shift() || null;
        }
        return null;
    };

    const token = getCookie('access_token') || (typeof window !== 'undefined' ? localStorage.getItem('access_token') : null);
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

// Interceptor to handle periodic refresh
api.interceptors.response.use(
    (response) => response,
    async (error) => {
        const originalRequest = error.config as AxiosRequestConfigWithRetry;
        if (error.response?.status === 401 && !originalRequest._retry) {
            originalRequest._retry = true;

            // Try to get refresh token from cookie first, then localStorage
            const getCookie = (name: string): string | null => {
                if (typeof window === 'undefined') return null;
                const value = `; ${document.cookie}`;
                const parts = value.split(`; ${name}=`);
                if (parts.length === 2) {
                    return parts.pop()?.split(';').shift() || null;
                }
                return null;
            };

            const refreshToken = getCookie('refresh_token') || (typeof window !== 'undefined' ? localStorage.getItem('refresh_token') : null);

            if (refreshToken) {
                try {
                    const res = await axios.post(`${api.defaults.baseURL}/auth/refresh`, {
                        refresh_token: refreshToken,
                    }, {
                        withCredentials: true,
                    });

                    if (res.status === 200) {
                        // Tokens are now set via httpOnly cookies by the server
                        // Also update localStorage for backwards compatibility
                        if (typeof window !== 'undefined') {
                            localStorage.setItem('access_token', res.data.access_token);
                            localStorage.setItem('refresh_token', res.data.refresh_token);
                        }
                        api.defaults.headers.common['Authorization'] = `Bearer ${res.data.access_token}`;
                        return api(originalRequest);
                    }
                } catch (refreshError) {
                    // Refresh failed, logout user
                    if (typeof window !== 'undefined') {
                        localStorage.removeItem('access_token');
                        localStorage.removeItem('refresh_token');
                        window.location.href = '/login';
                    }
                }
            }
        }
        return Promise.reject(error);
    }
);

export default api;

// ============================================================
// PUBLIC SOLUTION API
// ============================================================

import type { Solution } from '@/types';

export interface SolutionExistsResponse {
    exists: boolean;
}

export const solutionApi = {
    getSolution: (problemCode: string) =>
        api.get<Solution>(`/problem/${problemCode}/solution`),

    solutionExists: (problemCode: string) =>
        api.get<SolutionExistsResponse>(`/problem/${problemCode}/solution/exists`),
};
