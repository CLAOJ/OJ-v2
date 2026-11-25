// Client-side rules for the submit box's file picker. They mirror the API
// (claoj/api/v2/submission_file.go) so the obvious mistakes fail before the
// upload starts; the server still validates everything.

export const MAX_TEXT_SOURCE_BYTES = 65536;

export type SubmissionFileError = 'empty' | 'tooLarge' | 'wrongExtension' | 'notEnabled';

export interface LanguageLimits {
    extension?: string;
    file_only?: boolean;
    file_size_limit?: number;
}

function extensionOf(filename: string): string {
    const dot = filename.lastIndexOf('.');
    return dot > 0 ? filename.slice(dot + 1).toLowerCase() : '';
}

export function validateSubmissionFile(
    file: { name: string; size: number },
    language: LanguageLimits,
): SubmissionFileError | null {
    const expected = (language.extension ?? '').replace(/^\./, '').toLowerCase();
    if (!expected || extensionOf(file.name) !== expected) return 'wrongExtension';
    if (!language.file_size_limit || language.file_size_limit <= 0) return 'notEnabled';
    if (file.size <= 0) return 'empty';
    if (file.size > language.file_size_limit * 1024 * 1024) return 'tooLarge';
    return null;
}

// Text languages keep the editor; a picked file is read into it client-side.
export function validateTextSourceFile(file: { size: number }): 'empty' | 'tooLarge' | null {
    if (file.size <= 0) return 'empty';
    if (file.size > MAX_TEXT_SOURCE_BYTES) return 'tooLarge';
    return null;
}

// `print=1` makes the print view open the browser's print dialog on load: the
// "Download PDF" flow (there is no server-side HTML→PDF engine in v2).
export function problemPrintPath(code: string, autoPrint = true): string {
    const base = `/problem/${encodeURIComponent(code)}/print`;
    return autoPrint ? `${base}?print=1` : base;
}

export function problemReportPath(code: string): string {
    return `/ticket/create?problem=${encodeURIComponent(code)}`;
}

/** v1 /problem/<p>/resubmit/<submission>: the submit form prefilled from it. */
export function problemResubmitPath(code: string, submissionId: number | string): string {
    return `/problem/${encodeURIComponent(code)}?resubmit=${encodeURIComponent(String(submissionId))}`;
}

/** v1 /problem/<p>/tickets: the problem's tickets (gated by the API). */
export function problemTicketsPath(code: string): string {
    return `/problem/${encodeURIComponent(code)}/tickets`;
}