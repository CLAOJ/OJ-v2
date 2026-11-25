import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CloneProblemButton } from '@/components/admin/CloneProblemModal';
import { CloneContestButton } from '@/components/admin/CloneContestModal';

jest.mock('@/lib/adminApi', () => ({
    adminProblemApi: {
        clone: jest.fn(async () => ({
            data: { success: true, new_problem: { id: 5, code: 'newcode', name: 'New Name' } },
        })),
    },
    adminContestApi: {
        clone: jest.fn(async () => ({
            data: { success: true, message: 'ok', new_contest: { key: 'newkey', name: 'New Contest' } },
        })),
    },
}));

import { adminProblemApi, adminContestApi } from '@/lib/adminApi';

const cloneProblem = adminProblemApi.clone as jest.Mock;
const cloneContest = adminContestApi.clone as jest.Mock;

function renderWithQuery(ui: React.ReactElement) {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

/** Open the dialog from its title-labelled toolbar button, fill the fields. */
function openAndFill(title: string, keyPlaceholder: string, namePlaceholder: string, key: string, name: string) {
    fireEvent.click(screen.getByRole('button', { name: title }));
    fireEvent.change(screen.getByPlaceholderText(keyPlaceholder), { target: { value: key } });
    fireEvent.change(screen.getByPlaceholderText(namePlaceholder), { target: { value: name } });
    const submit = screen.getAllByRole('button', { name: title });
    fireEvent.click(submit[submit.length - 1]);
}

describe('clone actions', () => {
    beforeEach(() => {
        cloneProblem.mockClear();
        cloneContest.mockClear();
        window.alert = jest.fn();
    });

    it('clones a problem from the toolbar button with the chosen flags', async () => {
        renderWithQuery(<CloneProblemButton code="old" name="Old Problem" />);

        openAndFill(
            'problems.cloneModalTitle',
            'problems.newCodePlaceholder',
            'problems.newNamePlaceholder',
            'newcode',
            'New Name'
        );

        await waitFor(() => expect(cloneProblem).toHaveBeenCalledWith('old', {
            new_code: 'newcode',
            new_name: 'New Name',
            copy_data: false,
            copy_authors: true,
            copy_settings: true,
        }));
    });

    it('clones a contest with problems and settings copied by default', async () => {
        renderWithQuery(<CloneContestButton contestKey="old" contestName="Old Contest" />);

        openAndFill(
            'contests.cloneModalTitle',
            'contests.newKeyPlaceholder',
            'contests.newNamePlaceholder',
            'newkey',
            'New Contest'
        );

        await waitFor(() => expect(cloneContest).toHaveBeenCalledWith('old', {
            new_key: 'newkey',
            new_name: 'New Contest',
            copy_problems: true,
            copy_settings: true,
        }));
    });
});