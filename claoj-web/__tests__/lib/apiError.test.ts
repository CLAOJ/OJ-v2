import { apiErrorMessage } from '@/lib/apiError';

describe('apiErrorMessage', () => {
    it('returns the API message when present', () => {
        expect(apiErrorMessage({ response: { data: { error: 'class name already exists' } } }, 'fallback'))
            .toBe('class name already exists');
    });

    it('falls back for network errors and non-string payloads', () => {
        expect(apiErrorMessage(new Error('offline'), 'fallback')).toBe('fallback');
        expect(apiErrorMessage({ response: { data: { error: 42 } } }, 'fallback')).toBe('fallback');
        expect(apiErrorMessage(undefined, 'fallback')).toBe('fallback');
    });
});
