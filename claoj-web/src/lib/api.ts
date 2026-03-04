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
});

// Get API URL - uses env var if set, otherwise derives from window.location.origin
export function getApiUrl(): string {
    return api.defaults.baseURL as string;
}

// Interceptor to add access token
api.interceptors.request.use((config) => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('access_token') : null;
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
            const refreshToken = typeof window !== 'undefined' ? localStorage.getItem('refresh_token') : null;

            if (refreshToken) {
                try {
                    const res = await axios.post(`${api.defaults.baseURL}/auth/refresh`, {
                        refresh_token: refreshToken,
                    });

                    if (res.status === 200) {
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
