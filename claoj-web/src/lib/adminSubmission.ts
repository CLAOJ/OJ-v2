import type { AdminSubmissionDetail } from '@/types';

/** Editable form state for the admin submission detail page. */
export interface SubmissionEditValues {
    language_id: number;
    status: string;
    result: string;
    points: string;
    case_points: string;
    case_total: string;
    judged_on_id: string;
    judged_date: string;
    error: string;
    contest_object_id: string;
    locked_after: string;
    is_archived: boolean;
    source: string;
}

/** ISO timestamp -> value for <input type="datetime-local">, in the browser's time zone (sent back as RFC 3339). */
export function toAdminDateTime(iso: string | null | undefined): string {
    if (!iso) return '';
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return '';
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/**
 * Builds the PATCH body from the form: only fields that actually changed are
 * sent. In particular locked_after/is_archived stay out unless the viewer
 * edited them, so a staff editor without the lock/archive permission can still
 * save ordinary fields.
 */
export function buildSubmissionUpdatePayload(
    original: AdminSubmissionDetail,
    values: SubmissionEditValues,
): Record<string, unknown> {
    const payload: Record<string, unknown> = {};

    if (values.language_id !== original.language_id) payload.language_id = values.language_id;
    if (values.status !== original.status) payload.status = values.status;
    if (values.result !== (original.result ?? '')) payload.result = values.result || null;

    if (values.points !== '' && Number(values.points) !== original.points) payload.points = Number(values.points);
    if (values.case_points !== '' && Number(values.case_points) !== original.case_points) payload.case_points = Number(values.case_points);
    if (values.case_total !== '' && Number(values.case_total) !== original.case_total) payload.case_total = Number(values.case_total);

    const originalJudge = original.judged_on_id ? String(original.judged_on_id) : '';
    if (values.judged_on_id !== originalJudge) {
        payload.judged_on_id = values.judged_on_id ? Number(values.judged_on_id) : null;
    }
    if (values.judged_date !== toAdminDateTime(original.judged_date)) {
        payload.judged_date = values.judged_date ? new Date(values.judged_date).toISOString() : null;
    }

    if (values.error !== (original.error ?? '')) payload.error = values.error;

    const originalContest = original.contest_object_id ? String(original.contest_object_id) : '';
    if (values.contest_object_id !== originalContest) {
        payload.contest_object_id = values.contest_object_id ? Number(values.contest_object_id) : null;
    }

    if (values.locked_after !== toAdminDateTime(original.locked_after)) {
        payload.locked_after = values.locked_after ? new Date(values.locked_after).toISOString() : null;
    }
    if (values.is_archived !== original.is_archived) payload.is_archived = values.is_archived;
    if (values.source !== original.source) payload.source = values.source;

    return payload;
}

/** Form state seeded from the detail response. */
export function toSubmissionEditValues(sub: AdminSubmissionDetail): SubmissionEditValues {
    return {
        language_id: sub.language_id,
        status: sub.status,
        result: sub.result ?? '',
        points: sub.points === null ? '' : String(sub.points),
        case_points: String(sub.case_points),
        case_total: String(sub.case_total),
        judged_on_id: sub.judged_on_id ? String(sub.judged_on_id) : '',
        judged_date: toAdminDateTime(sub.judged_date),
        error: sub.error ?? '',
        contest_object_id: sub.contest_object_id ? String(sub.contest_object_id) : '',
        locked_after: toAdminDateTime(sub.locked_after),
        is_archived: sub.is_archived,
        source: sub.source,
    };
}