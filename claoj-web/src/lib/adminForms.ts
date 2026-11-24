// Pure mappers between the admin API detail shapes and the edit forms.
// Kept free of React so the round-trips are unit-tested.
import type {
    AdminProblemDetail,
    AdminProblemCreateRequest,
    AdminProblemUpdateRequest,
    AdminContestDetail,
    AdminContestProblemRow,
    AdminContestUpdateRequest,
    AdminContestCreateRequest,
    ContestProblemInput,
} from '@/types';
import type { ProblemFormData } from '@/components/admin/ProblemForm';
import type { ContestFormData } from '@/components/admin/ContestForm';

const pad = (n: number) => String(n).padStart(2, '0');

/** ISO timestamp → `datetime-local` value in the browser's time zone. */
export function toLocalInput(iso: string): string {
    if (!iso) return '';
    const d = new Date(iso);
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** `datetime-local` value (browser time zone) → RFC 3339 UTC. */
export function fromLocalInput(local: string): string {
    return new Date(local).toISOString();
}

export function problemToFormData(p: AdminProblemDetail): ProblemFormData {
    return {
        code: p.code,
        name: p.name,
        description: p.description,
        points: p.points,
        partial: p.partial,
        is_public: p.is_public,
        time_limit: p.time_limit,
        memory_limit: p.memory_limit,
        group_id: p.group_id,
        type_ids: [...p.type_ids],
        author_ids: [...p.author_ids],
        allowed_lang_ids: [...p.allowed_lang_ids],
        is_manually_managed: p.is_manually_managed,
        pdf_url: p.pdf_url,
        source: p.source,
        date: p.date ? toLocalInput(p.date) : '',
        license_id: p.license_id ?? undefined,
        og_image: p.og_image,
        summary: p.summary,
        is_full_markup: p.is_full_markup,
        submission_source_visibility_mode: p.submission_source_visibility_mode,
        testcase_visibility_mode: p.testcase_visibility_mode,
        is_organization_private: p.is_organization_private,
        organization_ids: [...p.organization_ids],
        banned_user_ids: [...p.banned_user_ids],
        suggester_id: p.suggester_id ?? undefined,
        curator_ids: [...p.curator_ids],
        tester_ids: [...p.tester_ids],
    };
}

function sharedProblemFields(data: ProblemFormData) {
    return {
        name: data.name,
        description: data.description,
        points: data.points,
        partial: data.partial,
        is_public: data.is_public,
        time_limit: data.time_limit,
        memory_limit: data.memory_limit,
        is_manually_managed: data.is_manually_managed ?? false,
        pdf_url: data.pdf_url ?? '',
        group_id: data.group_id,
        type_ids: data.type_ids ?? [],
        author_ids: data.author_ids ?? [],
        allowed_lang_ids: data.allowed_lang_ids ?? [],
        source: data.source ?? '',
        date: data.date ? fromLocalInput(data.date) : null,
        license_id: data.license_id ?? null,
        og_image: data.og_image ?? '',
        summary: data.summary ?? '',
        is_full_markup: data.is_full_markup ?? false,
        submission_source_visibility_mode: data.submission_source_visibility_mode,
        testcase_visibility_mode: data.testcase_visibility_mode,
        is_organization_private: data.is_organization_private ?? false,
        organization_ids: data.organization_ids ?? [],
        banned_user_ids: data.banned_user_ids ?? [],
        suggester_id: data.suggester_id ?? null,
        curator_ids: data.curator_ids ?? [],
        tester_ids: data.tester_ids ?? [],
    };
}

export function problemFormToCreate(data: ProblemFormData): AdminProblemCreateRequest {
    return { ...sharedProblemFields(data), code: data.code };
}

/**
 * PATCH ignores a null on `date`/`license_id`/`suggester_id` (nil = no change),
 * so an emptied field has to say so with its clear flag instead.
 */
export function problemFormToUpdate(data: ProblemFormData, existing?: AdminProblemDetail): AdminProblemUpdateRequest {
    const fields = sharedProblemFields(data);
    const clearDate = !data.date && !!existing?.date;
    const clearLicenseID = data.license_id == null && existing?.license_id != null;
    const clearSuggesterID = data.suggester_id == null && existing?.suggester_id != null;
    return {
        ...fields,
        date: clearDate ? undefined : fields.date,
        license_id: clearLicenseID ? undefined : fields.license_id,
        suggester_id: clearSuggesterID ? undefined : fields.suggester_id,
        ...(clearDate ? { clear_date: true } : {}),
        ...(clearLicenseID ? { clear_license_id: true } : {}),
        ...(clearSuggesterID ? { clear_suggester_id: true } : {}),
    };
}

export function contestToFormData(c: AdminContestDetail): ContestFormData {
    return {
        key: c.key,
        name: c.name,
        description: c.description,
        summary: c.summary,
        start_time: toLocalInput(c.start_time),
        end_time: toLocalInput(c.end_time),
        time_limit: c.time_limit ? Math.round(c.time_limit / 60) : undefined,
        is_visible: c.is_visible,
        is_rated: c.is_rated,
        format_name: c.format_name || 'default',
        format_config: c.format_config ? JSON.stringify(c.format_config) : '',
        access_code: c.access_code || '',
        hide_problem_tags: c.hide_problem_tags,
        run_pretests_only: c.run_pretests_only,
        is_organization_private: c.is_organization_private,
        author_ids: [...c.author_ids],
        curator_ids: [...c.curator_ids],
        tester_ids: [...c.tester_ids],
        tag_ids: [...c.tag_ids],
        problem_ids: c.problems.map(p => p.problem_id),
    };
}

/** Minutes from the form → seconds for the API (undefined when empty). */
const minutesToSeconds = (m: number | undefined) =>
    m && Number.isFinite(m) && m > 0 ? Math.round(m * 60) : undefined;

function sharedContestFields(data: ContestFormData) {
    return {
        name: data.name,
        description: data.description,
        summary: data.summary,
        start_time: fromLocalInput(data.start_time),
        end_time: fromLocalInput(data.end_time),
        is_visible: data.is_visible,
        is_rated: data.is_rated,
        format_name: data.format_name,
        format_config: data.format_config ?? '',
        access_code: data.access_code ?? '',
        hide_problem_tags: data.hide_problem_tags,
        run_pretests_only: data.run_pretests_only,
        is_organization_private: data.is_organization_private,
        author_ids: data.author_ids ?? [],
        curator_ids: data.curator_ids ?? [],
        tester_ids: data.tester_ids ?? [],
        tag_ids: data.tag_ids ?? [],
    };
}

export function contestFormToUpdate(data: ContestFormData, existing: AdminContestProblemRow[]): AdminContestUpdateRequest {
    const byProblem = new Map(existing.map(row => [row.problem_id, row]));
    const problems: ContestProblemInput[] = (data.problem_ids ?? []).map(id => {
        const row = byProblem.get(id);
        return row
            ? { problem_id: id, points: row.points, partial: row.partial, is_pretested: row.is_pretested }
            : { problem_id: id };
    });
    return {
        ...sharedContestFields(data),
        time_limit: minutesToSeconds(data.time_limit) ?? 0,
        problems,
    };
}

export function contestFormToCreate(data: ContestFormData): AdminContestCreateRequest {
    return {
        ...sharedContestFields(data),
        key: data.key,
        time_limit: minutesToSeconds(data.time_limit),
        problem_ids: data.problem_ids ?? [],
    } as AdminContestCreateRequest;
}
