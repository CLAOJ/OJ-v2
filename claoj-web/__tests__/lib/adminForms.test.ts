import {
    toLocalInput,
    fromLocalInput,
    problemToFormData,
    problemFormToCreate,
    problemFormToUpdate,
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
    source: 'VOI 2026', date: '2026-11-01T02:00:00Z', license_id: 4, og_image: '/og.png', summary: 'short',
    is_full_markup: true, submission_source_visibility_mode: 'S', testcase_visibility_mode: 'O',
    is_organization_private: true, organization_ids: [5], banned_user_ids: [6], suggester_id: 8,
    banned_users: [{ id: 6, username: 'd' }],
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

    it('problemToFormData maps every v1 publishing/access field', () => {
        const f = problemToFormData(problem);
        expect(f.source).toBe('VOI 2026');
        expect(f.date).toBe(toLocalInput('2026-11-01T02:00:00Z'));
        expect(f.license_id).toBe(4);
        expect(f.og_image).toBe('/og.png');
        expect(f.summary).toBe('short');
        expect(f.is_full_markup).toBe(true);
        expect(f.submission_source_visibility_mode).toBe('S');
        expect(f.testcase_visibility_mode).toBe('O');
        expect(f.is_organization_private).toBe(true);
        expect(f.organization_ids).toEqual([5]);
        expect(f.banned_user_ids).toEqual([6]);
        expect(f.suggester_id).toBe(8);
        expect(f.curator_ids).toEqual([2]);
        expect(f.tester_ids).toEqual([]);
    });

    it('problemToFormData renders an empty date as ""', () => {
        expect(problemToFormData({ ...problem, date: null }).date).toBe('');
    });

    it('problemFormToUpdate maps the whole form to the PATCH payload', () => {
        const f = problemToFormData(problem);
        const req = problemFormToUpdate(f, problem);
        expect(req.source).toBe('VOI 2026');
        expect(req.date).toBe(fromLocalInput(f.date!));
        expect(req.license_id).toBe(4);
        expect(req.og_image).toBe('/og.png');
        expect(req.summary).toBe('short');
        expect(req.is_full_markup).toBe(true);
        expect(req.submission_source_visibility_mode).toBe('S');
        expect(req.testcase_visibility_mode).toBe('O');
        expect(req.is_organization_private).toBe(true);
        expect(req.organization_ids).toEqual([5]);
        expect(req.banned_user_ids).toEqual([6]);
        expect(req.suggester_id).toBe(8);
        expect(req.curator_ids).toEqual([2]);
        expect(req.tester_ids).toEqual([]);
        expect(req.clear_date).toBeUndefined();
        expect(req.clear_license_id).toBeUndefined();
        expect(req.clear_suggester_id).toBeUndefined();
        expect('code' in req).toBe(false);
    });

    it('an emptied nullable field is cleared with its flag, never sent as null', () => {
        const f = {
            ...problemToFormData(problem),
            date: '',
            license_id: undefined,
            suggester_id: undefined,
        };
        const req = problemFormToUpdate(f, problem);
        expect(req.clear_date).toBe(true);
        expect(req.clear_license_id).toBe(true);
        expect(req.clear_suggester_id).toBe(true);
        expect(req.date).toBeUndefined();
        expect(req.license_id).toBeUndefined();
        expect(req.suggester_id).toBeUndefined();
    });

    it('an already empty nullable field sets no clear flag', () => {
        const empty = problemToFormData({ ...problem, date: null, license_id: null, suggester_id: null });
        const req = problemFormToUpdate(empty, { ...problem, date: null, license_id: null, suggester_id: null });
        expect(req.clear_date).toBeUndefined();
        expect(req.clear_license_id).toBeUndefined();
        expect(req.clear_suggester_id).toBeUndefined();
        expect(req.date).toBeNull();
        expect(req.license_id).toBeNull();
        expect(req.suggester_id).toBeNull();
    });

    it('problemFormToCreate posts the extended payload and the code', () => {
        const f = problemToFormData(problem);
        const req = problemFormToCreate(f);
        expect(req.code).toBe('aplusb');
        expect(req.source).toBe('VOI 2026');
        expect(req.date).toBe(fromLocalInput(f.date!));
        expect(req.license_id).toBe(4);
        expect(req.submission_source_visibility_mode).toBe('S');
        expect(req.testcase_visibility_mode).toBe('O');
        expect(req.suggester_id).toBe(8);
        expect(req.organization_ids).toEqual([5]);
        expect(req.banned_user_ids).toEqual([6]);
        expect(req.curator_ids).toEqual([2]);
        expect(req.tester_ids).toEqual([]);
        expect('clear_date' in req).toBe(false);
    });

    it('problemFormToCreate sends null for empty nullable fields', () => {
        const f = { ...problemToFormData(problem), date: '', license_id: undefined, suggester_id: undefined };
        const req = problemFormToCreate(f);
        expect(req.date).toBeNull();
        expect(req.license_id).toBeNull();
        expect(req.suggester_id).toBeNull();
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
