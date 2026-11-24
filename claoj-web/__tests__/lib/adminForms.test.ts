import {
    toLocalInput,
    fromLocalInput,
    problemToFormData,
    problemFormToCreate,
    problemFormToUpdate,
    contestToFormData,
    contestFormToUpdate,
    contestFormToCreate,
    commentUpdatePayload,
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
    is_visible: true, is_rated: true, format_name: 'ioi16', format_config: { cumtime: true },
    access_code: '', hide_problem_tags: false, hide_problem_authors: true, run_pretests_only: false,
    show_short_display: true, use_clarifications: false, push_announcements: true,
    scoreboard_visibility: 'C', scoreboard_cache_timeout: 600, points_precision: 5,
    rating_floor: 1200, rating_ceiling: 2400, performance_ceiling_override: 3000, rate_all: true,
    is_private: true, is_organization_private: true, limit_join_organizations: true,
    og_image: '/og.png', logo_override_image: '/logo.png', problem_label_script: 'print(1)',
    locked_after: null, author_ids: [4], curator_ids: [], tester_ids: [9], tag_ids: [3],
    organization_ids: [5], banned_user_ids: [6], private_contestant_ids: [7],
    view_scoreboard_user_ids: [8], join_organization_ids: [12], class_ids: [13],
    authors: [{ id: 4, username: 'a' }], curators: [], testers: [{ id: 9, username: 'b' }],
    banned_users: [{ id: 6, username: 'd' }], private_contestants: [{ id: 7, username: 'e' }],
    view_scoreboard_users: [{ id: 8, username: 'f' }],
    join_organizations: [{ id: 12, name: 'Org' }], organizations: [{ id: 5, name: 'Org 5' }],
    classes: [{ id: 13, name: 'Class 10A' }],
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
        const req = contestFormToUpdate(f, contest);
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
        expect(contestFormToUpdate(f, contest).time_limit).toBe(0);
        expect(contestFormToCreate(f).time_limit).toBeUndefined();
    });

    it('contestToFormData maps every v1 rating/scoreboard/access field', () => {
        const f = contestToFormData(contest);
        expect(f.rate_all).toBe(true);
        expect(f.rating_floor).toBe(1200);
        expect(f.rating_ceiling).toBe(2400);
        expect(f.performance_ceiling_override).toBe(3000);
        expect(f.scoreboard_visibility).toBe('C');
        expect(f.scoreboard_cache_timeout).toBe(600);
        expect(f.points_precision).toBe(5);
        expect(f.is_private).toBe(true);
        expect(f.limit_join_organizations).toBe(true);
        expect(f.og_image).toBe('/og.png');
        expect(f.logo_override_image).toBe('/logo.png');
        expect(f.problem_label_script).toBe('print(1)');
        expect(f.push_announcements).toBe(true);
        expect(f.hide_problem_authors).toBe(true);
        expect(f.show_short_display).toBe(true);
        expect(f.use_clarifications).toBe(false);
        expect(f.organization_ids).toEqual([5]);
        expect(f.banned_user_ids).toEqual([6]);
        expect(f.private_contestant_ids).toEqual([7]);
        expect(f.view_scoreboard_user_ids).toEqual([8]);
        expect(f.join_organization_ids).toEqual([12]);
        expect(f.class_ids).toEqual([13]);
    });

    it('contestToFormData renders a null rating column as an empty field', () => {
        const f = contestToFormData({ ...contest, rating_floor: null, rating_ceiling: null, performance_ceiling_override: null });
        expect(f.rating_floor).toBeUndefined();
        expect(f.rating_ceiling).toBeUndefined();
        expect(f.performance_ceiling_override).toBeUndefined();
    });

    it('contestFormToUpdate round-trips every extended field', () => {
        const req = contestFormToUpdate(contestToFormData(contest), contest);
        expect(req.rate_all).toBe(true);
        expect(req.rating_floor).toBe(1200);
        expect(req.rating_ceiling).toBe(2400);
        expect(req.performance_ceiling_override).toBe(3000);
        expect(req.scoreboard_visibility).toBe('C');
        expect(req.scoreboard_cache_timeout).toBe(600);
        expect(req.points_precision).toBe(5);
        expect(req.is_private).toBe(true);
        expect(req.limit_join_organizations).toBe(true);
        expect(req.og_image).toBe('/og.png');
        expect(req.logo_override_image).toBe('/logo.png');
        expect(req.problem_label_script).toBe('print(1)');
        expect(req.push_announcements).toBe(true);
        expect(req.hide_problem_authors).toBe(true);
        expect(req.show_short_display).toBe(true);
        expect(req.use_clarifications).toBe(false);
        expect(req.organization_ids).toEqual([5]);
        expect(req.banned_user_ids).toEqual([6]);
        expect(req.private_contestant_ids).toEqual([7]);
        expect(req.view_scoreboard_user_ids).toEqual([8]);
        expect(req.join_organization_ids).toEqual([12]);
        expect(req.class_ids).toEqual([13]);
        expect(req.clear_rating_floor).toBeUndefined();
        expect(req.clear_rating_ceiling).toBeUndefined();
        expect(req.clear_performance_ceiling_override).toBeUndefined();
    });

    it('an emptied rating column is cleared with its flag, never sent as null', () => {
        const f = { ...contestToFormData(contest), rating_floor: undefined, rating_ceiling: undefined, performance_ceiling_override: undefined };
        const req = contestFormToUpdate(f, contest);
        expect(req.clear_rating_floor).toBe(true);
        expect(req.clear_rating_ceiling).toBe(true);
        expect(req.clear_performance_ceiling_override).toBe(true);
        expect(req.rating_floor).toBeUndefined();
        expect(req.rating_ceiling).toBeUndefined();
        expect(req.performance_ceiling_override).toBeUndefined();
    });

    it('an already empty rating column sets no clear flag', () => {
        const empty = { ...contest, rating_floor: null, rating_ceiling: null, performance_ceiling_override: null };
        const req = contestFormToUpdate(contestToFormData(empty), empty);
        expect(req.clear_rating_floor).toBeUndefined();
        expect(req.clear_rating_ceiling).toBeUndefined();
        expect(req.clear_performance_ceiling_override).toBeUndefined();
    });

    it('contestFormToCreate posts the extended payload', () => {
        const req = contestFormToCreate(contestToFormData(contest));
        expect(req.key).toBe('tik25');
        expect(req.rate_all).toBe(true);
        expect(req.rating_floor).toBe(1200);
        expect(req.scoreboard_visibility).toBe('C');
        expect(req.points_precision).toBe(5);
        expect(req.use_clarifications).toBe(false);
        expect(req.problem_label_script).toBe('print(1)');
        expect(req.private_contestant_ids).toEqual([7]);
        expect(req.view_scoreboard_user_ids).toEqual([8]);
        expect(req.join_organization_ids).toEqual([12]);
        expect(req.class_ids).toEqual([13]);
        expect('clear_rating_floor' in req).toBe(false);
    });

    it('contestFormToCreate keeps a zero points precision', () => {
        const req = contestFormToCreate({ ...contestToFormData(contest), points_precision: 0 });
        expect(req.points_precision).toBe(0);
    });

    it('contestFormToUpdate keeps a zero points precision and zero cache timeout', () => {
        const f = { ...contestToFormData(contest), points_precision: 0, scoreboard_cache_timeout: 0 };
        const req = contestFormToUpdate(f, contest);
        expect(req.points_precision).toBe(0);
        expect(req.scoreboard_cache_timeout).toBe(0);
    });

    it('contestFormToUpdate works without the existing detail', () => {
        const f = contestToFormData(contest);
        const req = contestFormToUpdate(f);
        expect(req.problems).toEqual([{ problem_id: 20 }, { problem_id: 21 }]);
        expect(req.rating_floor).toBe(1200);
        expect(req.clear_rating_floor).toBeUndefined();
        expect(req.clear_rating_ceiling).toBeUndefined();
        expect(req.clear_performance_ceiling_override).toBeUndefined();
        expect(req.scoreboard_visibility).toBe('C');
    });

    it('an empty scoreboard visibility falls back to the visible code', () => {
        const req = contestFormToCreate({ ...contestToFormData(contest), scoreboard_visibility: undefined });
        expect(req.scoreboard_visibility).toBe('V');
    });

    it('commentUpdatePayload omits an unchanged page instead of echoing it', () => {
        const current = {
            id: 7, author_id: 4, username: 'ada', page: 'p/round#1', body: 'old',
            score: 0, hidden: false, time: '2026-11-01T00:00:00Z', revisions: 1,
        };
        const unchanged = commentUpdatePayload(current, {
            body: 'new', hidden: false, authorId: 4, page: 'p/round#1', reason: '',
        });
        expect(unchanged).toEqual({ body: 'new' });

        const changed = commentUpdatePayload(current, {
            body: 'new', hidden: true, authorId: 9, page: 'p/round-1.2', reason: 'fix',
        });
        expect(changed).toEqual({ body: 'new', hidden: true, author_id: 9, page: 'p/round-1.2', reason: 'fix' });
    });
});
