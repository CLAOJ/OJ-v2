import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import React from 'react';
import { renderWithProviders } from '../test-utils';
import AccountSettingsTab from '@/components/settings/AccountSettingsTab';

// Mock the API client used by all four account settings sub-components
// (PasswordChangeForm, EmailVerificationSection, TwoFactorSection, DangerZone).
// None of the sub-components read auth context, so no AuthProvider mock is needed.
jest.mock('@/lib/api', () => ({
    __esModule: true,
    default: {
        post: jest.fn(),
        get: jest.fn(),
    },
}));

import api from '@/lib/api';

// `next-intl`'s `useTranslations` is mocked globally in __tests__/setup.ts to
// return the raw message key instead of translated copy, so assertions below
// target keys like 'changePassword' / 'emailVerificationTitle' rather than
// human-readable English strings.

type GetRoutes = Record<string, unknown>;

// EmailVerificationSection (`GET /user/me`) and TwoFactorSection
// (`GET /auth/totp/status`) both query in parallel whenever the full tab
// renders, so tests that care about one section's state stub both endpoints
// by URL. Unlisted URLs resolve to an empty object so unrelated sections
// don't crash while their state isn't under test.
function mockGetRoutes(routes: GetRoutes) {
    (api.get as jest.Mock).mockImplementation((url: string) => {
        if (url in routes) {
            return Promise.resolve({ data: routes[url] });
        }
        return Promise.resolve({ data: {} });
    });
}

describe('AccountSettingsTab', () => {
    beforeEach(() => {
        jest.clearAllMocks();
        // Safe baseline so every mounted section's GET resolves to something.
        (api.get as jest.Mock).mockResolvedValue({ data: {} });
    });

    describe('Password Change Section', () => {
        it('renders password change form fields', () => {
            renderWithProviders(<AccountSettingsTab />);

            expect(screen.getByText('oldPassword')).toBeInTheDocument();
            expect(screen.getByText('newPassword')).toBeInTheDocument();
            expect(screen.getByText('confirmNewPassword')).toBeInTheDocument();
            expect(screen.getByRole('button', { name: 'changePassword' })).toBeInTheDocument();
        });

        it('shows a validation error when the new and confirm passwords do not match', async () => {
            const { container } = renderWithProviders(<AccountSettingsTab />);

            const passwordInputs = container.querySelectorAll('input[type="password"]');
            expect(passwordInputs.length).toBeGreaterThanOrEqual(3);

            fireEvent.change(passwordInputs[0], { target: { value: 'oldpass123' } });
            fireEvent.change(passwordInputs[1], { target: { value: 'newpass123' } });
            fireEvent.change(passwordInputs[2], { target: { value: 'different123' } });

            fireEvent.click(screen.getByRole('button', { name: 'changePassword' }));

            await waitFor(() => {
                expect(screen.getByText('passwordsDontMatch')).toBeInTheDocument();
            });
        });

        it('shows a validation error when the new password is too short', async () => {
            const { container } = renderWithProviders(<AccountSettingsTab />);

            const passwordInputs = container.querySelectorAll('input[type="password"]');
            expect(passwordInputs.length).toBeGreaterThanOrEqual(3);

            fireEvent.change(passwordInputs[0], { target: { value: 'oldpass123' } });
            fireEvent.change(passwordInputs[1], { target: { value: '123' } });
            fireEvent.change(passwordInputs[2], { target: { value: '123' } });

            fireEvent.click(screen.getByRole('button', { name: 'changePassword' }));

            await waitFor(() => {
                expect(screen.getByText('newPasswordMinLength')).toBeInTheDocument();
            });
        });

        it('submits the password change to the API on a valid form', async () => {
            (api.post as jest.Mock).mockResolvedValue({ data: {} });

            const { container } = renderWithProviders(<AccountSettingsTab />);

            const passwordInputs = container.querySelectorAll('input[type="password"]');

            fireEvent.change(passwordInputs[0], { target: { value: 'oldpass123' } });
            fireEvent.change(passwordInputs[1], { target: { value: 'newpass123' } });
            fireEvent.change(passwordInputs[2], { target: { value: 'newpass123' } });

            fireEvent.click(screen.getByRole('button', { name: 'changePassword' }));

            await waitFor(() => {
                expect(api.post).toHaveBeenCalledWith('/auth/password/change', {
                    current_password: 'oldpass123',
                    new_password: 'newpass123',
                });
            });
        });

        it('shows a success message after the password change succeeds', async () => {
            (api.post as jest.Mock).mockResolvedValue({ data: {} });

            const { container } = renderWithProviders(<AccountSettingsTab />);

            const passwordInputs = container.querySelectorAll('input[type="password"]');

            fireEvent.change(passwordInputs[0], { target: { value: 'oldpass123' } });
            fireEvent.change(passwordInputs[1], { target: { value: 'newpass123' } });
            fireEvent.change(passwordInputs[2], { target: { value: 'newpass123' } });

            fireEvent.click(screen.getByRole('button', { name: 'changePassword' }));

            await waitFor(() => {
                expect(screen.getByText('passwordChangeSuccess')).toBeInTheDocument();
            });
        });
    });

    describe('Email Verification Section', () => {
        it('renders the email verification section', () => {
            mockGetRoutes({ '/user/me': { is_active: true } });

            renderWithProviders(<AccountSettingsTab />);

            expect(screen.getByText('emailVerificationTitle')).toBeInTheDocument();
        });

        it('shows the verified state when the email is verified', async () => {
            mockGetRoutes({
                '/user/me': { is_active: true },
                '/auth/totp/status': { enabled: false, backup_codes_remaining: 0 },
            });

            renderWithProviders(<AccountSettingsTab />);

            await waitFor(() => {
                expect(screen.getByText('emailVerifiedMsg')).toBeInTheDocument();
            });
            expect(screen.getByText('verifiedBadge')).toBeInTheDocument();
        });

        it('shows the unverified warning and a resend button when the email is not verified', async () => {
            mockGetRoutes({
                '/user/me': { is_active: false },
                '/auth/totp/status': { enabled: false, backup_codes_remaining: 0 },
            });

            renderWithProviders(<AccountSettingsTab />);

            await waitFor(() => {
                expect(screen.getByText('emailNotVerifiedTitle')).toBeInTheDocument();
            });
            expect(screen.getByText('emailNotVerifiedDesc')).toBeInTheDocument();
            expect(screen.getByRole('button', { name: 'resendVerificationEmail' })).toBeInTheDocument();
        });

        it('calls the resend verification API when the resend button is clicked', async () => {
            mockGetRoutes({ '/user/me': { is_active: false } });
            (api.post as jest.Mock).mockResolvedValue({
                data: { message: 'Verification email sent. Please check your inbox.' },
            });
            const alertMock = jest.spyOn(window, 'alert').mockImplementation(() => {});

            renderWithProviders(<AccountSettingsTab />);

            const resendButton = await screen.findByRole('button', { name: 'resendVerificationEmail' });
            fireEvent.click(resendButton);

            await waitFor(() => {
                expect(api.post).toHaveBeenCalledWith('/auth/resend-verification', {});
            });
            expect(alertMock).toHaveBeenCalledWith('Verification email sent. Please check your inbox.');

            alertMock.mockRestore();
        });

        it('shows the server error message when resending verification fails', async () => {
            mockGetRoutes({ '/user/me': { is_active: false } });
            (api.post as jest.Mock).mockRejectedValue({
                response: { data: { error: 'Rate limit exceeded' } },
            });
            const alertMock = jest.spyOn(window, 'alert').mockImplementation(() => {});

            renderWithProviders(<AccountSettingsTab />);

            const resendButton = await screen.findByRole('button', { name: 'resendVerificationEmail' });
            fireEvent.click(resendButton);

            await waitFor(() => {
                expect(alertMock).toHaveBeenCalledWith('Rate limit exceeded');
            });

            alertMock.mockRestore();
        });

        it('defaults to the verified state before the profile query has resolved', () => {
            // No route stub: api.get is pending (unresolved) at the moment of
            // this assertion, so userProfile is still undefined and
            // `userProfile?.is_active ?? true` falls back to verified.
            renderWithProviders(<AccountSettingsTab />);

            expect(screen.getByText('emailVerifiedMsg')).toBeInTheDocument();
        });
    });

    describe('Two-Factor Authentication Section', () => {
        it('renders the 2FA section', () => {
            mockGetRoutes({ '/auth/totp/status': { enabled: false, backup_codes_remaining: 0 } });

            renderWithProviders(<AccountSettingsTab />);

            expect(screen.getByText('twoFactor')).toBeInTheDocument();
        });

        it('shows the setup UI when 2FA is disabled', async () => {
            mockGetRoutes({ '/auth/totp/status': { enabled: false, backup_codes_remaining: 0 } });

            renderWithProviders(<AccountSettingsTab />);

            await waitFor(() => {
                expect(screen.getByText('twoFactorDesc')).toBeInTheDocument();
            });
            expect(screen.getByRole('button', { name: 'setup2FA' })).toBeInTheDocument();
        });

        it('shows the enabled status when 2FA is active', async () => {
            mockGetRoutes({ '/auth/totp/status': { enabled: true, backup_codes_remaining: 10 } });

            renderWithProviders(<AccountSettingsTab />);

            await waitFor(() => {
                expect(screen.getByText('twoFactorEnabledMsg')).toBeInTheDocument();
            });
            expect(screen.getByText('active')).toBeInTheDocument();
        });
    });

    describe('Danger Zone Section', () => {
        it('renders the danger zone section', () => {
            renderWithProviders(<AccountSettingsTab />);

            expect(screen.getByText('dangerZone')).toBeInTheDocument();
            expect(screen.getByText('deleteAccountWarning')).toBeInTheDocument();
            expect(screen.getByRole('button', { name: 'deleteAccount' })).toBeInTheDocument();
        });
    });
});
