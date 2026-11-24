import api from '@/lib/api';
import { adminAuditLogApi } from '@/lib/adminApi';

jest.mock('@/lib/api', () => ({
    __esModule: true,
    default: { get: jest.fn(), post: jest.fn(), patch: jest.fn(), delete: jest.fn() },
}));

const mocked = api as jest.Mocked<typeof api>;

describe('adminAuditLogApi.list', () => {
    beforeEach(() => jest.clearAllMocks());

    it('requests /audit-logs with paging only when there are no filters', () => {
        adminAuditLogApi.list(1, 50);
        expect(mocked.get).toHaveBeenCalledWith('/audit-logs?page=1&page_size=50');
    });

    it('passes exclude_action so read traffic cannot crowd out changes', () => {
        adminAuditLogApi.list(1, 8, { exclude_action: 'VIEW' });
        expect(mocked.get).toHaveBeenCalledWith('/audit-logs?page=1&page_size=8&exclude_action=VIEW');
    });

    it('omits empty filter values', () => {
        adminAuditLogApi.list(2, 20, { action: '', resource: 'Problem' });
        expect(mocked.get).toHaveBeenCalledWith('/audit-logs?page=2&page_size=20&resource=Problem');
    });
});