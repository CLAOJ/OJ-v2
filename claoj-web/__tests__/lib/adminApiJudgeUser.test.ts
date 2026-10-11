import api from '@/lib/api';
import { adminJudgeApi, adminUserApi, adminSubmissionApi, adminTicketApi } from '@/lib/adminApi';

jest.mock('@/lib/api', () => ({
    __esModule: true,
    default: { get: jest.fn(), post: jest.fn(), patch: jest.fn(), delete: jest.fn() },
}));

const mocked = api as jest.Mocked<typeof api>;

describe('admin judge/user api client', () => {
    beforeEach(() => jest.clearAllMocks());

    it('adminJudgeApi.create posts to /admin/judges', () => {
        const body = { name: 'judge1', auth_key: 'k', description: 'd' };
        adminJudgeApi.create(body);
        expect(mocked.post).toHaveBeenCalledWith('/admin/judges', body);
    });

    it('adminJudgeApi.delete deletes /admin/judge/:id', () => {
        adminJudgeApi.delete(7);
        expect(mocked.delete).toHaveBeenCalledWith('/admin/judge/7');
    });

    it('adminUserApi.create posts to /admin/users', () => {
        const body = { username: 'bob', email: 'b@x.io', password: 'Passw0rdX', is_active: true };
        adminUserApi.create(body);
        expect(mocked.post).toHaveBeenCalledWith('/admin/users', body);
    });

    it('adminUserApi.setPassword posts only the password', () => {
        adminUserApi.setPassword(3, 'Passw0rdX');
        expect(mocked.post).toHaveBeenCalledWith('/admin/user/3/password', { password: 'Passw0rdX' });
    });

    it('adminUserApi.ban sends only the reason', () => {
        adminUserApi.ban(3, 'spam');
        expect(mocked.post).toHaveBeenCalledWith('/admin/user/3/ban', { reason: 'spam' });
    });

    it('adminSubmissionApi.delete deletes /admin/submission/:id', () => {
        adminSubmissionApi.delete(5);
        expect(mocked.delete).toHaveBeenCalledWith('/admin/submission/5');
    });

    it('adminTicketApi.delete deletes /admin/ticket/:id', () => {
        adminTicketApi.delete(9);
        expect(mocked.delete).toHaveBeenCalledWith('/admin/ticket/9');
    });
});
