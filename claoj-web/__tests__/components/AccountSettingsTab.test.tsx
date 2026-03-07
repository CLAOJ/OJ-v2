import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import AccountSettingsTab from '@/components/settings/AccountSettingsTab';

// Mock the AuthProvider
jest.mock('@/components/providers/AuthProvider', () => ({
    useAuth: () => ({
        user: {
            id: 1,
            username: 'testuser',
            is_admin: false,
            is_staff: false,
        },
    }),
}));

// Mock the API
jest.mock('@/lib/api', () => ({
    __esModule: true,
    default: {
        post: jest.fn(),
        get: jest.fn(),
    },
}));

import api from '@/lib/api';

const createTestQueryClient = () =>
    new QueryClient({
        defaultOptions: {
            queries: {
                retry: false,
            },
        },
    });

const wrapper = ({ children }: { children: React.ReactNode }) => {
    const testQueryClient = createTestQueryClient();
    return (
        <QueryClientProvider client={testQueryClient}>
            {children}
        </QueryClientProvider>
    );
};

describe('AccountSettingsTab', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    describe('Password Change Section', () => {
        it('renders password change form', () => {
            render(<AccountSettingsTab />, { wrapper });

            expect(screen.getAllByText('Change Password')[0]).toBeInTheDocument();
            // Check for password inputs
            expect(screen.getByRole('button', { name: /change password/i })).toBeInTheDocument();
        });

        it('shows validation error when passwords do not match', async () => {
            const { container } = render(<AccountSettingsTab />, { wrapper });

            // Find inputs by placeholder text
            const passwordInputs = container.querySelectorAll('input[type="password"]');

            expect(passwordInputs.length).toBeGreaterThanOrEqual(3);

            // Fill in the password fields
            fireEvent.change(passwordInputs[0], { target: { value: 'oldpass123' } });
            fireEvent.change(passwordInputs[1], { target: { value: 'newpass123' } });
            fireEvent.change(passwordInputs[2], { target: { value: 'different123' } });

            const submitButton = screen.getByRole('button', { name: /change password/i });
            fireEvent.click(submitButton);

            await waitFor(() => {
                expect(screen.getByText(/passwords don't match/i)).toBeInTheDocument();
            });
        });

        it('shows validation error for short passwords', async () => {
            const { container } = render(<AccountSettingsTab />, { wrapper });

            const passwordInputs = container.querySelectorAll('input[type="password"]');
            const submitButton = screen.getByRole('button', { name: /change password/i });

            fireEvent.change(passwordInputs[1], { target: { value: '123' } });
            fireEvent.click(submitButton);

            await waitFor(() => {
                expect(screen.getByText(/at least 6 characters/i)).toBeInTheDocument();
            });
        });

        it('submits password change on valid form', async () => {
            (api.post as jest.Mock).mockResolvedValue({ data: {} });

            const { container } = render(<AccountSettingsTab />, { wrapper });

            const passwordInputs = container.querySelectorAll('input[type="password"]');
            const submitButton = screen.getByRole('button', { name: /change password/i });

            fireEvent.change(passwordInputs[0], { target: { value: 'oldpass123' } });
            fireEvent.change(passwordInputs[1], { target: { value: 'newpass123' } });
            fireEvent.change(passwordInputs[2], { target: { value: 'newpass123' } });
            fireEvent.click(submitButton);

            await waitFor(() => {
                expect(api.post).toHaveBeenCalledWith('/auth/password/change', {
                    current_password: 'oldpass123',
                    new_password: 'newpass123',
                });
            });
        });

        it('shows success message after password change', async () => {
            (api.post as jest.Mock).mockResolvedValue({ data: {} });

            const { container } = render(<AccountSettingsTab />, { wrapper });

            const passwordInputs = container.querySelectorAll('input[type="password"]');
            const submitButton = screen.getByRole('button', { name: /change password/i });

            fireEvent.change(passwordInputs[0], { target: { value: 'oldpass123' } });
            fireEvent.change(passwordInputs[1], { target: { value: 'newpass123' } });
            fireEvent.change(passwordInputs[2], { target: { value: 'newpass123' } });
            fireEvent.click(submitButton);

            await waitFor(() => {
                expect(screen.getByText(/password changed successfully/i)).toBeInTheDocument();
            });
        });
    });

    describe('Email Verification Section', () => {
        it('renders email verification section', () => {
            (api.get as jest.Mock).mockResolvedValue({
                data: { is_active: true },
            });

            render(<AccountSettingsTab />, { wrapper });

            expect(screen.getByText('Email Verification')).toBeInTheDocument();
        });

        it('shows verified status when email is verified', async () => {
            (api.get as jest.Mock).mockImplementation((url: string) => {
                if (url === '/user/me') {
                    return Promise.resolve({ data: { is_active: true } });
                }
                if (url === '/auth/totp/status') {
                    return Promise.resolve({ data: { enabled: false, backup_codes_remaining: 0 } });
                }
                return Promise.resolve({ data: {} });
            });

            render(<AccountSettingsTab />, { wrapper });

            await waitFor(() => {
                expect(screen.getByText(/your email is verified/i)).toBeInTheDocument();
            });

            expect(screen.getByText('Verified')).toBeInTheDocument();
        });

        it('shows unverified warning and resend button when email is not verified', async () => {
            (api.get as jest.Mock).mockImplementation((url: string) => {
                if (url === '/user/me') {
                    return Promise.resolve({ data: { is_active: false } });
                }
                if (url === '/auth/totp/status') {
                    return Promise.resolve({ data: { enabled: false, backup_codes_remaining: 0 } });
                }
                return Promise.resolve({ data: {} });
            });

            render(<AccountSettingsTab />, { wrapper });

            await waitFor(() => {
                expect(screen.getByText(/email not verified/i)).toBeInTheDocument();
            });

            expect(screen.getByText(/check your inbox for the verification link/i)).toBeInTheDocument();
            expect(screen.getByRole('button', { name: /resend verification email/i })).toBeInTheDocument();
        });

        it('calls resend verification API when button is clicked', async () => {
            (api.get as jest.Mock).mockImplementation((url: string) => {
                if (url === '/user/me') {
                    return Promise.resolve({ data: { is_active: false } });
                }
                if (url === '/auth/totp/status') {
                    return Promise.resolve({ data: { enabled: false, backup_codes_remaining: 0 } });
                }
                return Promise.resolve({ data: {} });
            });

            (api.post as jest.Mock).mockResolvedValue({
                data: { message: 'Verification email sent. Please check your inbox.' },
            });

            // Mock alert
            const alertMock = jest.spyOn(window, 'alert').mockImplementation(() => {});

            render(<AccountSettingsTab />, { wrapper });

            await waitFor(() => {
                expect(screen.getByRole('button', { name: /resend verification email/i })).toBeInTheDocument();
            });

            const resendButton = screen.getByRole('button', { name: /resend verification email/i });
            fireEvent.click(resendButton);

            await waitFor(() => {
                expect(api.post).toHaveBeenCalledWith('/auth/resend-verification', {});
            });

            expect(alertMock).toHaveBeenCalledWith('Verification email sent. Please check your inbox.');

            alertMock.mockRestore();
        });

        it('shows error when resend verification fails', async () => {
            (api.get as jest.Mock).mockImplementation((url: string) => {
                if (url === '/user/me') {
                    return Promise.resolve({ data: { is_active: false } });
                }
                if (url === '/auth/totp/status') {
                    return Promise.resolve({ data: { enabled: false, backup_codes_remaining: 0 } });
                }
                return Promise.resolve({ data: {} });
            });

            (api.post as jest.Mock).mockRejectedValue({
                response: { data: { error: 'Rate limit exceeded' } },
            });

            const alertMock = jest.spyOn(window, 'alert').mockImplementation(() => {});

            render(<AccountSettingsTab />, { wrapper });

            await waitFor(() => {
                expect(screen.getByRole('button', { name: /resend verification email/i })).toBeInTheDocument();
            });

            const resendButton = screen.getByRole('button', { name: /resend verification email/i });
            fireEvent.click(resendButton);

            await waitFor(() => {
                expect(alertMock).toHaveBeenCalledWith('Rate limit exceeded');
            });

            alertMock.mockRestore();
        });

        it('defaults to verified state when profile data is not loaded', () => {
            (api.get as jest.Mock).mockResolvedValue({
                data: { enabled: false, backup_codes_remaining: 0 },
            });

            render(<AccountSettingsTab />, { wrapper });

            // When userProfile is undefined, isEmailVerified should default to true
            expect(screen.getByText(/your email is verified/i)).toBeInTheDocument();
        });
    });

    describe('Two-Factor Authentication Section', () => {
        it('renders 2FA section', () => {
            (api.get as jest.Mock).mockResolvedValue({
                data: { enabled: false, backup_codes_remaining: 0 },
            });

            render(<AccountSettingsTab />, { wrapper });

            expect(screen.getByText('Two-Factor Authentication')).toBeInTheDocument();
        });

        it('shows setup UI when 2FA is disabled', async () => {
            (api.get as jest.Mock).mockImplementation((url: string) => {
                if (url === '/auth/totp/status') {
                    return Promise.resolve({ data: { enabled: false, backup_codes_remaining: 0 } });
                }
                return Promise.resolve({ data: {} });
            });

            render(<AccountSettingsTab />, { wrapper });

            await waitFor(() => {
                expect(screen.getByText(/add an extra layer of security/i)).toBeInTheDocument();
            });

            expect(screen.getByRole('button', { name: /setup 2fa/i })).toBeInTheDocument();
        });

        it('shows enabled status when 2FA is active', async () => {
            (api.get as jest.Mock).mockImplementation((url: string) => {
                if (url === '/auth/totp/status') {
                    return Promise.resolve({ data: { enabled: true, backup_codes_remaining: 10 } });
                }
                return Promise.resolve({ data: {} });
            });

            render(<AccountSettingsTab />, { wrapper });

            await waitFor(() => {
                expect(screen.getByText(/two-factor authentication is enabled/i)).toBeInTheDocument();
            });

            expect(screen.getByText('Active')).toBeInTheDocument();
        });
    });

    describe('Danger Zone Section', () => {
        it('renders danger zone section', () => {
            render(<AccountSettingsTab />, { wrapper });

            expect(screen.getByText('Danger Zone')).toBeInTheDocument();
            expect(screen.getByText(/once you delete your account/i)).toBeInTheDocument();
            expect(screen.getByRole('button', { name: /delete account/i })).toBeInTheDocument();
        });
    });
});
