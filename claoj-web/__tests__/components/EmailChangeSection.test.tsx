import { describe, it, expect, beforeEach, jest } from '@jest/globals';

type PostMock = jest.Mock<(url: string, body?: unknown) => Promise<{ data: unknown }>>;

import { screen, fireEvent, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../test-utils';
import { EmailChangeSection } from '@/components/settings/account/EmailChangeSection';

jest.mock('@/lib/api', () => ({
    __esModule: true,
    default: {
        post: jest.fn(),
    },
}));

import api from '@/lib/api';

const post = api.post as unknown as PostMock;

describe('EmailChangeSection', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    it('posts the new address and current password, then asks the user to check the inbox', async () => {
        post.mockResolvedValue({ data: { message: 'ok' } });

        renderWithProviders(<EmailChangeSection />);

        fireEvent.change(screen.getByLabelText('emailChangeNewLabel'), {
            target: { value: 'new@example.com' },
        });
        fireEvent.change(screen.getByLabelText('passwordLabel'), {
            target: { value: 'secret123' },
        });
        fireEvent.click(screen.getByRole('button', { name: 'emailChangeSubmit' }));

        await waitFor(() => {
            expect(api.post).toHaveBeenCalledWith('/auth/email/change', {
                new_email: 'new@example.com',
                password: 'secret123',
            });
        });
        expect(await screen.findByText('emailChangeSent')).toBeInTheDocument();
    });

    it('shows the server error when the request is rejected', async () => {
        post.mockRejectedValue({
            response: { data: { error: 'this email address is already taken' } },
        });

        renderWithProviders(<EmailChangeSection />);

        fireEvent.change(screen.getByLabelText('emailChangeNewLabel'), {
            target: { value: 'taken@example.com' },
        });
        fireEvent.change(screen.getByLabelText('passwordLabel'), {
            target: { value: 'secret123' },
        });
        fireEvent.click(screen.getByRole('button', { name: 'emailChangeSubmit' }));

        expect(await screen.findByRole('alert')).toHaveTextContent('this email address is already taken');
    });
});