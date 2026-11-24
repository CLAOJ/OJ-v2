import {
    MAX_TEXT_SOURCE_BYTES,
    problemPrintPath,
    problemReportPath,
    validateSubmissionFile,
    validateTextSourceFile,
} from '@/lib/submissionFile';

const zipLang = { extension: 'zip', file_only: true, file_size_limit: 2 };

describe('validateSubmissionFile', () => {
    it('accepts a file matching the language extension within the limit', () => {
        expect(validateSubmissionFile({ name: 'answer.zip', size: 1024 }, zipLang)).toBeNull();
        // The extension check is case-insensitive, like the server's.
        expect(validateSubmissionFile({ name: 'ANSWER.ZIP', size: 1024 }, zipLang)).toBeNull();
    });

    it('rejects a mismatched or missing extension', () => {
        expect(validateSubmissionFile({ name: 'answer.rar', size: 1024 }, zipLang)).toBe('wrongExtension');
        expect(validateSubmissionFile({ name: 'answer', size: 1024 }, zipLang)).toBe('wrongExtension');
        expect(validateSubmissionFile({ name: 'answer.zip', size: 1024 }, { extension: '', file_size_limit: 2 })).toBe('wrongExtension');
    });

    it('rejects an empty, oversize or disabled file upload', () => {
        expect(validateSubmissionFile({ name: 'a.zip', size: 0 }, zipLang)).toBe('empty');
        expect(validateSubmissionFile({ name: 'a.zip', size: 2 * 1024 * 1024 + 1 }, zipLang)).toBe('tooLarge');
        expect(validateSubmissionFile({ name: 'a.zip', size: 1 }, { ...zipLang, file_size_limit: 0 })).toBe('notEnabled');
        expect(validateSubmissionFile({ name: 'a.zip', size: 1 }, { extension: 'zip' })).toBe('notEnabled');
    });

    it('treats a dotted directory-free name only via its final extension', () => {
        expect(validateSubmissionFile({ name: 'a.b.c.zip', size: 1 }, zipLang)).toBeNull();
        expect(validateSubmissionFile({ name: '.zip', size: 1 }, zipLang)).toBe('wrongExtension');
    });
});

describe('validateTextSourceFile', () => {
    it('caps the editor load at the API source limit', () => {
        expect(validateTextSourceFile({ size: 10 })).toBeNull();
        expect(validateTextSourceFile({ size: 0 })).toBe('empty');
        expect(validateTextSourceFile({ size: MAX_TEXT_SOURCE_BYTES })).toBeNull();
        expect(validateTextSourceFile({ size: MAX_TEXT_SOURCE_BYTES + 1 })).toBe('tooLarge');
    });
});

describe('problem print and report paths', () => {
    it('builds the print view URL and asks it to open the print dialog', () => {
        expect(problemPrintPath('abc')).toBe('/problem/abc/print?print=1');
        expect(problemPrintPath('abc', false)).toBe('/problem/abc/print');
    });

    it('escapes the problem code in both paths', () => {
        expect(problemPrintPath('a/b')).toBe('/problem/a%2Fb/print?print=1');
        expect(problemReportPath('a/b')).toBe('/ticket/create?problem=a%2Fb');
        expect(problemReportPath('p100')).toBe('/ticket/create?problem=p100');
    });
});