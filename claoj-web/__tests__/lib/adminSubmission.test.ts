import { buildSubmissionUpdatePayload, toAdminDateTime, toSubmissionEditValues } from '@/lib/adminSubmission';
import type { AdminSubmissionDetail } from '@/types';

const detail: AdminSubmissionDetail = {
    id: 1,
    user: 'alice',
    problem: 'aplusb',
    problem_name: 'A+B',
    language_id: 3,
    language: 'CPP17',
    language_name: 'C++17',
    date: '2026-11-20T10:00:00Z',
    time: 0.5,
    memory: 1024,
    points: 100,
    status: 'D',
    result: 'AC',
    error: null,
    case_points: 100,
    case_total: 100,
    current_testcase: 2,
    batch: false,
    is_pretested: false,
    judged_on_id: 7,
    judged_on: 'judge-1',
    judged_date: '2026-11-20T10:01:00Z',
    rejudged_date: null,
    locked_after: '2026-11-21T10:00:00Z',
    is_locked: false,
    is_archived: false,
    contest_object_id: null,
    contest_key: '',
    contest_name: '',
    source: 'int main() {}',
    test_cases: [],
};

describe('adminSubmission helpers', () => {
    it('formats timestamps as datetime-local in the browser time zone', () => {
        expect(toAdminDateTime(new Date(2026, 10, 20, 10, 1).toISOString())).toBe('2026-11-20T10:01');
        expect(toAdminDateTime(null)).toBe('');
        expect(toAdminDateTime('nope')).toBe('');
    });

    it('sends an edited local time as an RFC 3339 instant', () => {
        const values = { ...toSubmissionEditValues(detail), judged_date: '2026-11-20T12:00' };
        expect(buildSubmissionUpdatePayload(detail, values)).toEqual({
            judged_date: new Date(2026, 10, 20, 12, 0).toISOString(),
        });
    });

    it('sends nothing when nothing changed', () => {
        expect(buildSubmissionUpdatePayload(detail, toSubmissionEditValues(detail))).toEqual({});
    });

    it('omits lock and archive fields the viewer did not touch', () => {
        const values = { ...toSubmissionEditValues(detail), points: '42' };
        expect(buildSubmissionUpdatePayload(detail, values)).toEqual({ points: 42 });
    });

    it('maps cleared nullable fields to null and edited values to primitives', () => {
        const values = {
            ...toSubmissionEditValues(detail),
            result: '',
            judged_on_id: '',
            judged_date: '',
            contest_object_id: '9',
            locked_after: '',
            is_archived: true,
            source: 'print(1)',
        };
        expect(buildSubmissionUpdatePayload(detail, values)).toEqual({
            result: null,
            judged_on_id: null,
            judged_date: null,
            contest_object_id: 9,
            locked_after: null,
            is_archived: true,
            source: 'print(1)',
        });
    });
});