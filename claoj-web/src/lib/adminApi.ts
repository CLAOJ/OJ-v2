import api from './api';
import type {
    AdminUser,
    AdminUserUpdateRequest,
    AdminContest,
    AdminContestCreateRequest,
    AdminProblem,
    AdminProblemCreateRequest,
    AdminJudge,
    AdminJudgeDetail,
    AdminJudgeUpdateRequest,
    AdminOrganization,
    AdminSubmission,
    Role,
    Permission,
    ProblemData,
    ProblemDataFile,
    ProblemDataUpdateRequest,
    TestCaseContent,
    TestCaseUpdateRequest,
    TestCaseReorderRequest,
    AdminSolution,
    AdminSolutionDetail,
    SolutionCreateRequest,
    SolutionUpdateRequest,
    AdminTicket,
    AdminTicketDetail,
    AdminComment,
    AdminCommentUpdateRequest,
    AdminLanguage,
    AdminLanguageCreateRequest,
    AdminLanguageUpdateRequest,
    AdminLanguageDetail,
    AdminBlogPost,
    AdminBlogPostCreateRequest,
    AdminBlogPostUpdateRequest,
    AdminBlogPostDetail,
    AdminLicense,
    AdminLicenseCreateRequest,
    AdminLicenseUpdateRequest,
    AdminLicenseDetail,
    AdminUserListResponse,
    AdminContestListResponse,
    AdminProblemListResponse,
    AdminJudgeListResponse,
    AdminOrganizationListResponse,
    AdminSubmissionListResponse,
    AdminCommentListResponse,
    AdminLanguageListResponse,
    AdminBlogPostListResponse,
    AdminLicenseListResponse,
    AdminProblemGroupListResponse,
    AdminProblemTypeListResponse,
    AdminProblemGroup,
    AdminProblemType,
    AdminProblemGroupCreateRequest,
    AdminProblemGroupUpdateRequest,
    AdminProblemTypeCreateRequest,
    AdminProblemTypeUpdateRequest,
    RoleListResponse,
    AdminSolutionListResponse,
    AdminTicketListResponse,
    AdminNavigationBar,
    AdminNavigationBarDetail,
    AdminNavigationBarCreateRequest,
    AdminNavigationBarUpdateRequest,
    AdminNavigationBarListResponse,
    AdminMiscConfig,
    AdminMiscConfigDetail,
    AdminMiscConfigCreateRequest,
    AdminMiscConfigUpdateRequest,
    AdminMiscConfigListResponse,
    LanguageLimit,
    AdminLanguageLimit,
    AdminLanguageLimitCreateRequest,
    AdminLanguageLimitUpdateRequest,
    AdminLanguageLimitListResponse,
} from '@/types';

// ============================================================
// ADMIN PROBLEM DATA API
// ============================================================

export interface ProblemTestCase {
    id: number;
    order: number;
    input_file: string;
    output_file: string;
}

export const adminProblemDataApi = {
    detail: (code: string) =>
        api.get<ProblemData>(`/admin/problem/${code}/data`),

    upload: (code: string, formData: FormData) =>
        api.post<{ success: boolean; test_cases: ProblemTestCase[] }>(`/admin/problem/${code}/data`, formData, {
            headers: { 'Content-Type': 'multipart/form-data' }
        }),

    deleteTestCase: (code: string, testCaseId: number) =>
        api.delete(`/admin/problem/${code}/data/testcase/${testCaseId}`),

    reorder: (code: string, data: TestCaseReorderRequest) =>
        api.patch<{ success: boolean; test_cases: ProblemTestCase[] }>(`/admin/problem/${code}/data/reorder`, data),

    files: (code: string) =>
        api.get<{ files: ProblemDataFile[] }>(`/admin/problem/${code}/data/files`),

    getFileContent: (code: string, path: string) =>
        api.get<{ content: string; encoding?: string }>(`/admin/problem/${code}/data/file/${encodeURIComponent(path)}`),

    deleteFile: (code: string, path: string) =>
        api.delete(`/admin/problem/${code}/data/file/${encodeURIComponent(path)}`),

    getTestCaseContent: (code: string, testCaseId: number) =>
        api.get<TestCaseContent>(`/admin/problem/${code}/data/testcase/${testCaseId}/content`),

    updateTestCase: (code: string, testCaseId: number, data: TestCaseUpdateRequest) =>
        api.patch<{ success: boolean }>(`/admin/problem/${code}/data/testcase/${testCaseId}`, data),
};

// ============================================================
// ADMIN USER API
// ============================================================


export const adminUserApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminUserListResponse>(`/admin/users?page=${page}&page_size=${pageSize}`),

    detail: (id: number) =>
        api.get<AdminUser>(`/admin/user/${id}`),

    update: (id: number, data: AdminUserUpdateRequest) =>
        api.patch(`/admin/user/${id}`, data),

    delete: (id: number) =>
        api.delete(`/admin/user/${id}`),

    ban: (id: number, reason: string, days: number) =>
        api.post(`/admin/user/${id}/ban`, { reason, day: days }),

    unban: (id: number) =>
        api.post(`/admin/user/${id}/unban`),
};

// ============================================================
// ADMIN CONTEST API
// ============================================================


export const adminContestApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminContestListResponse>(`/admin/contests?page=${page}&page_size=${pageSize}`),

    detail: (key: string) =>
        api.get<AdminContest & { problems: any[] }>(`/admin/contest/${key}`),

    create: (data: AdminContestCreateRequest) =>
        api.post<{ success: boolean; contest: any }>('/admin/contests', data),

    update: (key: string, data: Partial<AdminContestCreateRequest>) =>
        api.patch(`/admin/contest/${key}`, data),

    delete: (key: string) =>
        api.delete(`/admin/contest/${key}`),
    lock: (key: string, lockedAfter: string | null) =>
        api.post<{ success: boolean; message: string; locked: boolean }>(`/admin/contest/${key}/lock`, { locked_after: lockedAfter }),

    clone: (key: string, data: {
        new_key: string;
        new_name: string;
        copy_problems?: boolean;
        copy_settings?: boolean;
        new_start_time?: string;
        new_end_time?: string;
    }) =>
        api.post<{ success: boolean; message: string; new_contest: any }>(`/admin/contest/${key}/clone`, data),

    disqualifyParticipation: (key: string, participationId: number) =>
        api.post<{ message: string; participation: { id: number; is_disqualified: boolean } }>(
            `/admin/contest/${key}/participation/${participationId}/disqualify`
        ),

    undisqualifyParticipation: (key: string, participationId: number) =>
        api.post<{ message: string; participation: { id: number; is_disqualified: boolean } }>(
            `/admin/contest/${key}/participation/${participationId}/undisqualify`
        ),
};

// ============================================================
// ADMIN PROBLEM API
// ============================================================


export const adminProblemApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminProblemListResponse>(`/admin/problems?page=${page}&page_size=${pageSize}`),

    detail: (code: string) =>
        api.get<AdminProblem & { authors: any[]; types: any[] }>(`/admin/problem/${code}`),

    create: (data: AdminProblemCreateRequest) =>
        api.post<{ success: boolean; problem: any }>('/admin/problems', data),

    update: (code: string, data: Partial<AdminProblemCreateRequest>) =>
        api.patch(`/admin/problem/${code}`, data),

    delete: (code: string) =>
        api.delete(`/admin/problem/${code}`),

    clone: (code: string, data: {
        new_code: string;
        new_name: string;
        copy_data?: boolean;
        copy_authors?: boolean;
        copy_settings?: boolean;
        new_description?: string;
        new_summary?: string;
    }) =>
        api.post<{ success: boolean; new_problem: { id: number; code: string; name: string } }>(
            `/admin/problem/${code}/clone`,
            data
        ),
};

// ============================================================
// ADMIN JUDGE API
// ============================================================


export const adminJudgeApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminJudgeListResponse>(`/admin/judges?page=${page}&page_size=${pageSize}`),

    detail: (id: number) =>
        api.get<AdminJudgeDetail>(`/admin/judge/${id}`),

    block: (id: number) =>
        api.post(`/admin/judge/${id}/block`),

    unblock: (id: number) =>
        api.post(`/admin/judge/${id}/unblock`),

    enable: (id: number) =>
        api.post(`/admin/judge/${id}/enable`),

    disable: (id: number) =>
        api.post(`/admin/judge/${id}/disable`),

    update: (id: number, data: AdminJudgeUpdateRequest) =>
        api.patch(`/admin/judge/${id}`, data),
};

// ============================================================
// ADMIN ORGANIZATION API
// ============================================================


export interface AdminOrganizationUpdateRequest {
    name?: string;
    slug?: string;
    short_name?: string;
    about?: string;
    is_open?: boolean;
    is_unlisted?: boolean;
    slots?: number;
    access_code?: string;
}

export const adminOrganizationApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminOrganizationListResponse>(`/admin/organizations?page=${page}&page_size=${pageSize}`),

    create: (data: {
        name: string;
        slug: string;
        short_name: string;
        about?: string;
        is_open?: boolean;
        is_unlisted?: boolean;
        slots?: number;
        access_code?: string;
    }) =>
        api.post<{ success: boolean; organization: AdminOrganization }>('/admin/organizations', data),

    update: (id: number, data: AdminOrganizationUpdateRequest) =>
        api.patch(`/admin/organization/${id}`, data),

    delete: (id: number) =>
        api.delete(`/admin/organization/${id}`),
};

// ============================================================
// ADMIN SUBMISSION API
// ============================================================


export interface BatchRejudgeRequest {
    submission_ids?: number[];
    filters?: {
        user_id?: number;
        problem_id?: number;
        problem_code?: string;
        language_id?: number;
        status?: string;
        result?: string;
        from_date?: string;
        to_date?: string;
    };
    dry_run?: boolean;
}

export interface BatchRejudgeResponse {
    success?: boolean;
    count: number;
    message: string;
}

export interface MossAnalysisResult {
    id: number;
    submission_id: number;
    match_count: number;
    matches: MossMatch[];
    created_at: string;
}

export interface MossMatch {
    submission_id: number;
    username: string;
    similarity_percentage: number;
    match_lines: number;
}

export interface BatchRescoreRequest {
    submission_ids?: number[];
    problem_id?: number;
    user_id?: number;
    dry_run?: boolean;
}

export interface BatchRescoreResponse {
    rescored: number;
    total: number;
    message: string;
}

export const adminSubmissionApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminSubmissionListResponse>(`/admin/submissions?page=${page}&page_size=${pageSize}`),

    rejudge: (id: number) =>
        api.post(`/admin/submission/${id}/rejudge`),

    abort: (id: number) =>
        api.post<{ success: boolean; message: string }>(`/admin/submission/${id}/abort`),

    batchRejudge: (data: BatchRejudgeRequest) =>
        api.post<BatchRejudgeResponse>('/admin/submissions/batch-rejudge', data),

    rescore: (id: number) =>
        api.post<{ message: string; submission_id: number }>(`/admin/submission/${id}/rescore`),

    batchRescore: (data: BatchRescoreRequest) =>
        api.post<BatchRescoreResponse>('/admin/submissions/batch-rescore', data),

    rescoreAll: (problemCode: string) =>
        api.post<BatchRescoreResponse>(`/admin/problem/${problemCode}/rescore-all`),

    mossAnalyze: (id: number, language: string) =>
        api.post<{ success: boolean; message: string }>(`/admin/submission/${id}/moss`, { language }),

    mossResults: (id: number) =>
        api.get<MossAnalysisResult>(`/admin/submission/${id}/moss`),
};

// ============================================================
// ADMIN ROLES & PERMISSIONS API
// ============================================================


export interface RoleCreateRequest {
    name: string;
    display_name: string;
    description?: string;
    color?: string;
    is_default?: boolean;
    permission_ids?: number[];
}

export interface RoleUpdateRequest {
    display_name?: string;
    description?: string;
    color?: string;
    is_default?: boolean;
    permission_ids?: number[];
}

export const adminRolesApi = {
    list: () =>
        api.get<RoleListResponse>('/admin/roles'),

    detail: (id: number) =>
        api.get<Role>(`/admin/role/${id}`),

    create: (data: RoleCreateRequest) =>
        api.post<{ success: boolean; role: Role }>('/admin/roles', data),

    update: (id: number, data: RoleUpdateRequest) =>
        api.patch<{ success: boolean; role: Role }>(`/admin/role/${id}`, data),

    delete: (id: number) =>
        api.delete(`/admin/role/${id}`),

    permissions: () =>
        api.get<{ data: Permission[] }>('/admin/permissions'),

    assignRole: (profileId: number, roleId: number) =>
        api.post<{ success: boolean }>(`/admin/profile/${profileId}/roles`, { role_id: roleId }),

    removeRole: (profileId: number, roleId: number) =>
        api.delete(`/admin/profile/${profileId}/roles/${roleId}`),
};

// ============================================================
// ADMIN SOLUTION API
// ============================================================


export const adminSolutionApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminSolutionListResponse>(`/admin/solutions?page=${page}&page_size=${pageSize}`),

    detail: (id: number) =>
        api.get<AdminSolutionDetail>(`/admin/solution/${id}`),

    create: (data: SolutionCreateRequest) =>
        api.post<{ success: boolean; solution: { id: number } }>('/admin/solutions', data),

    update: (id: number, data: SolutionUpdateRequest) =>
        api.patch<{ success: boolean }>(`/admin/solution/${id}`, data),

    delete: (id: number) =>
        api.delete(`/admin/solution/${id}`),
};

// ============================================================
// ADMIN TICKET API
// ============================================================


export interface AdminTicketAssignRequest {
    profile_ids: number[];
}

export const adminTicketApi = {
    list: (page: number = 1, pageSize: number = 50, filters?: {
        search?: string;
        status?: 'open' | 'closed';
        assigned?: 'true' | 'false';
        is_contributive?: 'true' | 'false';
    }) => {
        const params = new URLSearchParams({
            page: page.toString(),
            page_size: pageSize.toString(),
            ...filters,
        });
        return api.get<AdminTicketListResponse>(`/admin/tickets?${params.toString()}`);
    },

    detail: (id: number) =>
        api.get<AdminTicketDetail>(`/admin/ticket/${id}`),

    assign: (id: number, profileIds: number[]) =>
        api.post<{ message: string }>(`/admin/ticket/${id}/assign`, { profile_ids: profileIds }),

    toggleOpen: (id: number) =>
        api.post<{ message: string; is_open: boolean }>(`/admin/ticket/${id}/toggle`),

    setContributive: (id: number, isContributive: boolean) =>
        api.post<{ message: string; is_contributive: boolean }>(`/admin/ticket/${id}/set-contributive`, { is_contributive: isContributive }),

    updateNotes: (id: number, notes: string) =>
        api.patch<{ message: string }>(`/admin/ticket/${id}/notes`, { notes }),
};

// ============================================================
// ADMIN COMMENT API
// ============================================================


export const adminCommentApi = {
    list: (page: number = 1, pageSize: number = 50, filters?: {
        search?: string;
        hidden?: 'true' | 'false';
    }) => {
        const params = new URLSearchParams({
            page: page.toString(),
            page_size: pageSize.toString(),
            ...filters,
        });
        return api.get<AdminCommentListResponse>(`/admin/comments?${params.toString()}`);
    },

    update: (id: number, data: AdminCommentUpdateRequest) =>
        api.patch<{ message: string; comment: { id: number; body: string; hidden: boolean } }>(`/admin/comment/${id}`, data),

    delete: (id: number) =>
        api.delete<{ message: string }>(`/admin/comment/${id}`),
};

// ============================================================
// ADMIN LANGUAGE API
// ============================================================


export const adminLanguageApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminLanguageListResponse>(`/admin/languages?page=${page}&page_size=${pageSize}`),

    detail: (id: number) =>
        api.get<AdminLanguageDetail>(`/admin/language/${id}`),

    create: (data: AdminLanguageCreateRequest) =>
        api.post<{ message: string; language: { id: number; key: string } }>('/admin/languages', data),

    update: (id: number, data: AdminLanguageUpdateRequest) =>
        api.patch<{ message: string; language: { id: number; key: string } }>(`/admin/language/${id}`, data),

    delete: (id: number) =>
        api.delete<{ message: string }>(`/admin/language/${id}`),
};

// ============================================================
// ADMIN BLOG POST API
// ============================================================


export const adminBlogPostApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminBlogPostListResponse>(`/admin/blog-posts?page=${page}&page_size=${pageSize}`),

    detail: (id: number) =>
        api.get<AdminBlogPostDetail>(`/admin/blog-post/${id}`),

    create: (data: AdminBlogPostCreateRequest) =>
        api.post<{ message: string; blog_post: { id: number; slug: string } }>('/admin/blog-posts', data),

    update: (id: number, data: AdminBlogPostUpdateRequest) =>
        api.patch<{ message: string; blog_post: { id: number; slug: string } }>(`/admin/blog-post/${id}`, data),

    delete: (id: number) =>
        api.delete<{ message: string }>(`/admin/blog-post/${id}`),
};

// ============================================================
// ADMIN LICENSE API
// ============================================================


export const adminLicenseApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminLicenseListResponse>(`/admin/licenses?page=${page}&page_size=${pageSize}`),

    detail: (id: number) =>
        api.get<AdminLicenseDetail>(`/admin/license/${id}`),

    create: (data: AdminLicenseCreateRequest) =>
        api.post<{ message: string; license: { id: number; key: string } }>('/admin/licenses', data),

    update: (id: number, data: AdminLicenseUpdateRequest) =>
        api.patch<{ message: string; license: { id: number; key: string } }>(`/admin/license/${id}`, data),

    delete: (id: number) =>
        api.delete<{ message: string }>(`/admin/license/${id}`),
};

// ============================================================
// ADMIN PROBLEM TAXONOMY API
// ============================================================



export const adminProblemGroupApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminProblemGroupListResponse>(`/admin/problem-groups?page=${page}&page_size=${pageSize}`),

    detail: (id: number) =>
        api.get<AdminProblemGroup>(`/admin/problem-group/${id}`),

    create: (data: AdminProblemGroupCreateRequest) =>
        api.post<{ message: string; group: { id: number; name: string } }>('/admin/problem-groups', data),

    update: (id: number, data: AdminProblemGroupUpdateRequest) =>
        api.patch<{ message: string; group: { id: number; name: string } }>(`/admin/problem-group/${id}`, data),

    delete: (id: number) =>
        api.delete<{ message: string }>(`/admin/problem-group/${id}`),
};

export const adminProblemTypeApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminProblemTypeListResponse>(`/admin/problem-types?page=${page}&page_size=${pageSize}`),

    detail: (id: number) =>
        api.get<AdminProblemType>(`/admin/problem-type/${id}`),

    create: (data: AdminProblemTypeCreateRequest) =>
        api.post<{ message: string; type: { id: number; name: string } }>('/admin/problem-types', data),

    update: (id: number, data: AdminProblemTypeUpdateRequest) =>
        api.patch<{ message: string; type: { id: number; name: string } }>(`/admin/problem-type/${id}`, data),

    delete: (id: number) =>
        api.delete<{ message: string }>(`/admin/problem-type/${id}`),
};

// ============================================================
// ADMIN NAVIGATION BAR API (Task #51)
// ============================================================

export const adminNavigationBarApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminNavigationBarListResponse>(`/admin/navigation-bars?page=${page}&page_size=${pageSize}`),

    detail: (id: number) =>
        api.get<AdminNavigationBarDetail>(`/admin/navigation-bar/${id}`),

    create: (data: AdminNavigationBarCreateRequest) =>
        api.post<{ message: string; navigation_bar: { id: number; key: string } }>('/admin/navigation-bars', data),

    update: (id: number, data: AdminNavigationBarUpdateRequest) =>
        api.patch<{ message: string }>(`/admin/navigation-bar/${id}`, data),

    delete: (id: number) =>
        api.delete<{ message: string }>(`/admin/navigation-bar/${id}`),
};

// ============================================================
// ADMIN MISC CONFIG API (Task #52)
// ============================================================

export const adminMiscConfigApi = {
    list: (page: number = 1, pageSize: number = 50, search?: string) => {
        const params = new URLSearchParams({
            page: page.toString(),
            page_size: pageSize.toString(),
            ...(search ? { search } : {}),
        });
        return api.get<AdminMiscConfigListResponse>(`/admin/misc-configs?${params.toString()}`);
    },

    detail: (id: number) =>
        api.get<AdminMiscConfigDetail>(`/admin/misc-config/${id}`),

    create: (data: AdminMiscConfigCreateRequest) =>
        api.post<AdminMiscConfigDetail>('/admin/misc-configs', data),

    update: (id: number, data: AdminMiscConfigUpdateRequest) =>
        api.patch<AdminMiscConfigDetail>(`/admin/misc-config/${id}`, data),

    delete: (id: number) =>
        api.delete<{ message: string }>(`/admin/misc-config/${id}`),
};

// ============================================================
// ADMIN LANGUAGE LIMIT API (Task #35)
// ============================================================

export const adminLanguageLimitApi = {
    list: (page: number = 1, pageSize: number = 50, problemId?: number) => {
        const params = new URLSearchParams({
            page: page.toString(),
            page_size: pageSize.toString(),
            ...(problemId ? { problem_id: problemId.toString() } : {}),
        });
        return api.get<AdminLanguageLimitListResponse>(`/admin/language-limits?${params.toString()}`);
    },

    detail: (id: number) =>
        api.get<AdminLanguageLimit>(`/admin/language-limit/${id}`),

    create: (data: AdminLanguageLimitCreateRequest) =>
        api.post<{ data: AdminLanguageLimit; success: boolean; message: string }>(`/admin/language-limits`, data),

    update: (id: number, data: AdminLanguageLimitUpdateRequest) =>
        api.patch<{ data: AdminLanguageLimit; success: boolean; message: string }>(`/admin/language-limit/${id}`, data),

    delete: (id: number) =>
        api.delete<{ success: boolean; message: string }>(`/admin/language-limit/${id}`),
};
