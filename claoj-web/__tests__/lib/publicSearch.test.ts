import api from '@/lib/api';
import {
    fetchPublicLicenses,
    searchOrganizationProblems,
    searchPublicUsers,
} from '@/lib/publicSearch';

jest.mock('@/lib/api', () => ({
    __esModule: true,
    default: { get: jest.fn(), post: jest.fn(), patch: jest.fn(), delete: jest.fn() },
}));

const mockedGet = api.get as jest.MockedFunction<typeof api.get>;

describe('organization picker sources', () => {
    beforeEach(() => jest.clearAllMocks());

    it('searches only the organization problem list and maps ids for the contest form', async () => {
        mockedGet.mockResolvedValue({
            data: { data: [{ id: 11, code: 'orgp', name: 'Org Problem' }], total: 1 },
        } as never);

        const rows = await searchOrganizationProblems(7)('orgp');

        expect(mockedGet).toHaveBeenCalledWith('/organization/7/problems', {
            params: { search: 'orgp', page_size: 20 },
        });
        expect(rows).toEqual([{ id: 11, label: 'orgp', detail: 'Org Problem' }]);
    });

    it('searches the public user list (profile ids) instead of /admin/users', async () => {
        mockedGet.mockResolvedValue({
            data: { data: [{ id: 42, username: 'alice' }], total: 1 },
        } as never);

        const rows = await searchPublicUsers('ali');

        expect(mockedGet).toHaveBeenCalledWith('/users', {
            params: { search: 'ali', page_size: 20 },
        });
        expect(rows).toEqual([{ id: 42, label: 'alice' }]);
    });

    it('reads licenses from the public list', async () => {
        const body = { data: [{ id: 3, key: 'cc-by', name: 'CC BY', link: '', display: '', icon: '' }] };
        mockedGet.mockResolvedValue({ data: body } as never);

        await expect(fetchPublicLicenses()).resolves.toEqual(body);
        expect(mockedGet).toHaveBeenCalledWith('/licenses');
    });
});