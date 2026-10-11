import {
    toLocalInput,
    fromLocalInput,
    problemToFormData,
    contestToFormData,
    contestFormToUpdate,
    contestFormToCreate,
} from '@/lib/adminForms';
import type { AdminProblemDetail, AdminContestDetail } from '@/types';

const problem: AdminProblemDetail = {
    id: 1, code: 'aplusb', name: 'A+B', description: 'd', points: 5, partial: false, is_public: true,
    time_limit: 1.5, memory_limit: 262144, group_id: 3, is_manually_managed: false, pdf_url: '/x.pdf',
    author_ids: [4, 9], curator_ids: [2], tester_ids: [], type_ids: [7], allowed_lang_ids: [1, 2],
    authors: [{ id: 4, username: 'a' }, { id: 9, username: 'b' }], curators: [{ id: 2, username: 'c' }], testers: [],
};

const contest: AdminContestDetail = {
    id: 1, key: 'tik25', name: 'TiK25', description: 'd', summary: 's',
    start_time: '2026-11-01T02:00:00Z', end_time: '2026-11-01T05:00:00Z', time_limit: 5400,
    is_visible: true, is_rated: false, format_name: 'ioi16', format_config: { cumtime: true },
    access_code: '', hide_problem_tags: false, run_pretests_only: false, is_organization_private: false,
    locked_after: null, author_ids: [4], curator_ids: [], tester_ids: [9], tag_ids: [3],
    authors: [{ id: 4, username: 'a' }], curators: [], testers: [{ id: 9, username: 'b' }],
    problems: [
        { id: 11, problem_id: 20, code: 'p20', name: 'P20', points: 250, partial: false, is_pretested: true, order: 1 },
        { id: 12, problem_id: 21, code: 'p21', name: 'P21', points: 100, partial: true, is_pretested: false, order: 2 },
    ],
};

describe('adminForms', () => {
    it('local datetime round-trips through ISO', () => {
        const local = '2026-11-01T09:30';
        const iso = fromLocalInput(local);
        expect(iso).toMatch(/Z$/);
        expect(toLocalInput(iso)).toBe(local);
        expect(new Date(iso).getTime()).toBe(new Date(2026, 10, 1, 9, 30).getTime());
    });

    it('problemToFormData keeps every relation id', () => {
        const f = problemToFormData(problem);
        expect(f.author_ids).toEqual([4, 9]);
        expect(f.type_ids).toEqual([7]);
        expect(f.allowed_lang_ids).toEqual([1, 2]);
        expect(f.group_id).toBe(3);
        expect(f.pdf_url).toBe('/x.pdf');
    });

    it('contestToFormData keeps every relation id and shows minutes', () => {
        const f = contestToFormData(contest);
        expect(f.author_ids).toEqual([4]);
        expect(f.tester_ids).toEqual([9]);
        expect(f.curator_ids).toEqual([]);
        expect(f.tag_ids).toEqual([3]);
        expect(f.problem_ids).toEqual([20, 21]);
        expect(f.time_limit).toBe(90);
        expect(f.format_config).toBe('{"cumtime":true}');
        expect(f.start_time).toBe(toLocalInput(contest.start_time));
    });

    it('contestFormToUpdate preserves per-problem settings and appends new problems', () => {
        const f = { ...contestToFormData(contest), problem_ids: [21, 20, 30] };
        const req = contestFormToUpdate(f, contest.problems);
        expect(req.problems).toEqual([
            { problem_id: 21, points: 100, partial: true, is_pretested: false },
            { problem_id: 20, points: 250, partial: false, is_pretested: true },
            { problem_id: 30 },
        ]);
        expect(req.time_limit).toBe(5400);
        expect(req.start_time).toBe(fromLocalInput(f.start_time));
        expect('key' in req).toBe(false);
        expect('problem_ids' in req).toBe(false);
    });

    it('an emptied time limit clears it on update and is omitted on create', () => {
        const f = { ...contestToFormData(contest), time_limit: undefined };
        expect(contestFormToUpdate(f, contest.problems).time_limit).toBe(0);
        expect(contestFormToCreate(f).time_limit).toBeUndefined();
    });
});
