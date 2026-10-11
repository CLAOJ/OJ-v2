import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { EntityPicker, EntityOption } from '@/components/admin/EntityPicker';

function renderPicker(props: Partial<React.ComponentProps<typeof EntityPicker>> = {}) {
    const onChange = jest.fn();
    const search = jest.fn(async (term: string): Promise<EntityOption[]> =>
        [{ id: 7, label: `${term}-match`, detail: 'Seven' }]);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
        <QueryClientProvider client={client}>
            <EntityPicker
                id="authors"
                label="Authors"
                value={[4]}
                onChange={onChange}
                search={search}
                queryKey="users"
                initialOptions={[{ id: 4, label: 'alice' }]}
                {...props}
            />
        </QueryClientProvider>
    );
    return { onChange, search };
}

describe('EntityPicker', () => {
    it('shows labels for preselected ids', () => {
        renderPicker();
        expect(screen.getByText('alice')).toBeInTheDocument();
    });

    it('searches after typing and adds the clicked option', async () => {
        const { onChange, search } = renderPicker();
        fireEvent.change(screen.getByRole('combobox'), { target: { value: 'bo' } });
        const option = await screen.findByRole('option', { name: /bo-match/ });
        expect(search).toHaveBeenCalledWith('bo');
        fireEvent.click(option);
        expect(onChange).toHaveBeenCalledWith([4, 7]);
    });

    it('removes a chip', () => {
        const { onChange } = renderPicker();
        fireEvent.click(screen.getByRole('button', { name: 'picker.remove' }));
        expect(onChange).toHaveBeenCalledWith([]);
    });

    it('does not search for an empty term', async () => {
        const { search } = renderPicker();
        fireEvent.change(screen.getByRole('combobox'), { target: { value: '   ' } });
        await waitFor(() => expect(search).not.toHaveBeenCalled());
    });
});
