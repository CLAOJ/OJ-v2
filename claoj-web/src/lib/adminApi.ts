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
        api.get<{ content: string; filename: string }>(`/admin/problem/${code}/data/file/${encodeURIComponent(path)}`),

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

export interface AdminUserListResponse {
    data: AdminUser[];
    total: number;
    page: number;
    page_size: number;
}

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

export interface AdminContestListResponse {
    data: AdminContest[];
    total: number;
    page: number;
    page_size: number;
}

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
};

// ============================================================
// ADMIN PROBLEM API
// ============================================================

export interface AdminProblemListResponse {
    data: AdminProblem[];
    total: number;
    page: number;
    page_size: number;
}

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
};

// ============================================================
// ADMIN JUDGE API
// ============================================================

export interface AdminJudgeListResponse {
    data: AdminJudge[];
    total: number;
    page: number;
    page_size: number;
}

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

export interface AdminOrganizationListResponse {
    data: AdminOrganization[];
    total: number;
    page: number;
    page_size: number;
}

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

export interface AdminSubmissionListResponse {
    data: AdminSubmission[];
    total: number;
    page: number;
    page_size: number;
}

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

export const adminSubmissionApi = {
    list: (page: number = 1, pageSize: number = 50) =>
        api.get<AdminSubmissionListResponse>(`/admin/submissions?page=${page}&page_size=${pageSize}`),

    rejudge: (id: number) =>
        api.post(`/admin/submission/${id}/rejudge`),

    abort: (id: number) =>
        api.post<{ success: boolean; message: string }>(`/admin/submission/${id}/abort`),

    batchRejudge: (data: BatchRejudgeRequest) =>
        api.post<BatchRejudgeResponse>('/admin/submissions/batch-rejudge', data),

    mossAnalyze: (id: number, language: string) =>
        api.post<{ success: boolean; message: string }>(`/admin/submission/${id}/moss`, { language }),

    mossResults: (id: number) =>
        api.get<MossAnalysisResult>(`/admin/submission/${id}/moss`),
};

// ============================================================
// ADMIN ROLES & PERMISSIONS API
// ============================================================

export interface RoleListResponse {
    data: Role[];
    total: number;
}

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

export interface AdminSolutionListResponse {
    data: AdminSolution[];
    total: number;
    page: number;
    page_size: number;
}

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
