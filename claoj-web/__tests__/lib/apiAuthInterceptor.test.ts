import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import api from '@/lib/api';

/**
 * The response interceptor in lib/api.ts refreshes the access token and
 * retries whenever a request comes back 401. These tests pin down which
 * requests it is allowed to do that for.
 */

type Handler = (config: InternalAxiosRequestConfig) => Promise<unknown>;

function reject(status: number, body: unknown, config: InternalAxiosRequestConfig) {
    return Promise.reject(
        new AxiosError('Request failed', String(status), config, {}, {
            status,
            statusText: '',
            data: body,
            headers: {},
            config,
        } as never)
    );
}

function resolve(status: number, body: unknown, config: InternalAxiosRequestConfig) {
    return Promise.resolve({ status, statusText: '', data: body, headers: {}, config });
}

describe('api 401 interceptor', () => {
    let postSpy: jest.SpyInstance;
    let handler: Handler;
    const originalAdapter = api.defaults.adapter;

    beforeEach(() => {
        // Route every request made through the `api` instance to `handler`.
        api.defaults.adapter = ((config: InternalAxiosRequestConfig) =>
            handler(config)) as never;
        // The refresh call is made with the bare axios export, not the instance.
        postSpy = jest.spyOn(axios, 'post');
    });

    afterEach(() => {
        postSpy.mockRestore();
        api.defaults.adapter = originalAdapter;
    });

    // The bug: POST /auth/login answering 401 "invalid username or password"
    // entered the refresh branch. For a logged-out visitor the refresh then
    // answered 401 "refresh token not found in cookie", and *that* message was
    // thrown in place of the real one and rendered on the login form.
    it('does not refresh when the login request itself returns 401', async () => {
        handler = (config) =>
            reject(401, { error: 'invalid username or password' }, config);

        await expect(
            api.post('/auth/login', { username: 'u', password: 'bad' })
        ).rejects.toMatchObject({
            response: { data: { error: 'invalid username or password' } },
        });

        expect(postSpy).not.toHaveBeenCalled();
    });

    it.each([
        '/auth/register',
        '/auth/refresh',
        '/auth/logout',
        '/auth/totp/verify',
        '/auth/webauthn/login',
        '/auth/password/reset',
    ])('does not refresh when %s returns 401', async (url) => {
        handler = (config) => reject(401, { error: 'nope' }, config);

        await expect(api.post(url, {})).rejects.toMatchObject({
            response: { data: { error: 'nope' } },
        });
        expect(postSpy).not.toHaveBeenCalled();
    });

    it('still refreshes and retries an ordinary request that returns 401', async () => {
        let calls = 0;
        handler = (config) => {
            calls += 1;
            return calls === 1
                ? reject(401, { error: 'access token required' }, config)
                : resolve(200, { ok: true }, config);
        };
        postSpy.mockResolvedValue({ status: 200, data: {} } as never);

        const res = await api.get('/user/me');

        expect(postSpy).toHaveBeenCalledTimes(1);
        expect(postSpy.mock.calls[0][0]).toContain('/auth/refresh');
        expect(res.data).toEqual({ ok: true });
    });

    // A 403/429/500 must surface its own error. Previously any status could be
    // parked in the retry queue during a refresh window and then rejected with
    // the refresh's error instead of its own.
    it.each([403, 429, 500])('passes a %s straight through without refreshing', async (status) => {
        handler = (config) => reject(status, { error: 'real reason' }, config);

        await expect(api.get('/problems')).rejects.toMatchObject({
            response: { data: { error: 'real reason' } },
        });
        expect(postSpy).not.toHaveBeenCalled();
    });

    // The backend answers 409 when another tab rotated the refresh token a
    // moment ago; the session is intact and the winner's cookies are already
    // in the shared jar, so the original request should simply go again.
    it('retries the original request when refresh reports a rotation collision', async () => {
        let calls = 0;
        handler = (config) => {
            calls += 1;
            return calls === 1
                ? reject(401, { error: 'access token required' }, config)
                : resolve(200, { ok: true }, config);
        };
        postSpy.mockRejectedValue(
            new AxiosError('conflict', '409', undefined, {}, {
                status: 409,
                statusText: '',
                data: { error: 'refresh already in progress, retry' },
                headers: {},
                config: {},
            } as never)
        );

        const res = await api.get('/user/me');

        expect(res.data).toEqual({ ok: true });
    });

    it('gives up when the refresh genuinely fails', async () => {
        handler = (config) => reject(401, { error: 'access token required' }, config);
        postSpy.mockRejectedValue(
            new AxiosError('unauthorized', '401', undefined, {}, {
                status: 401,
                statusText: '',
                data: { error: 'refresh token has expired' },
                headers: {},
                config: {},
            } as never)
        );

        await expect(api.get('/user/me')).rejects.toMatchObject({
            response: { data: { error: 'refresh token has expired' } },
        });
    });
});
