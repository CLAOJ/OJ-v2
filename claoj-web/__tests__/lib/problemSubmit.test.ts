import { InternalAxiosRequestConfig } from 'axios';
import api, { problemSubmitApi } from '@/lib/api';

/**
 * File-only submissions must reach the API as multipart/form-data. The shared
 * axios instance defaults to application/json, and axios 1.x JSON-stringifies a
 * FormData body when the request looks like JSON — the server then sees no
 * file. These tests run the real axios transform pipeline (only the adapter is
 * replaced), so a regression to the old call shows up here.
 */

type Sent = { url?: string; data: unknown; contentType: string | undefined };

describe('problemSubmitApi', () => {
    const originalAdapter = api.defaults.adapter;
    let sent: Sent;

    beforeEach(() => {
        sent = { data: undefined, contentType: undefined };
        api.defaults.adapter = ((config: InternalAxiosRequestConfig) => {
            const headers = config.headers;
            sent = {
                url: config.url,
                data: config.data,
                contentType: headers?.get?.('Content-Type') as string | undefined,
            };
            return Promise.resolve({
                status: 200,
                statusText: '',
                data: { submission_id: 7 },
                headers: {},
                config,
            });
        }) as never;
    });

    afterEach(() => {
        api.defaults.adapter = originalAdapter;
    });

    it('keeps FormData intact for file submissions', async () => {
        const file = new File(['PK\x03\x04'], 'answer.zip', { type: 'application/zip' });

        const res = await problemSubmitApi.submit('aplusb', { language: 'ZIP', file });

        expect(res.data.submission_id).toBe(7);
        expect(sent.data).toBeInstanceOf(FormData);
        expect((sent.data as FormData).get('submission_file')).toBe(file);
        expect((sent.data as FormData).get('language')).toBe('ZIP');
        // Not application/json: axios would have stringified the body.
        expect(sent.contentType ?? '').not.toContain('application/json');
    });

    it('still posts text sources as JSON, with the contest in the URL', async () => {
        await problemSubmitApi.submit('aplusb', { language: 'CPP17', source: 'int main(){}' }, 'tik25');

        expect(sent.url).toBe('/problem/aplusb/submit?contest=tik25');
        expect(sent.data).toBe(JSON.stringify({ language: 'CPP17', source: 'int main(){}' }));
        expect(sent.contentType).toContain('application/json');
    });
});