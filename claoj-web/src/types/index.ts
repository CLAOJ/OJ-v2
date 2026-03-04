export interface APIResponse<T> {
    status: string;
    data: T;
}

export interface PaginatedList<T> {
    data: T[];
}

export interface Problem {
    code: string;
    name: string;
    points: number;
    partial: boolean;
    user_count: number;
    ac_rate: number;
    date?: string;
    group: string;
    is_solved: boolean;
}

export interface ProblemDetail extends Problem {
    id: number;
    description: string;
    is_full_markup: boolean;
    time_limit: number;
    memory_limit: number;
    group: string;
    languages: { key: string; name: string }[];
    types: { name: string }[];
    authors: { username: string }[];
    is_solved: boolean;
    is_attempted: boolean;
}

export interface Submission {
    id: number;
    user: string;
    problem: string;
    problem_name: string;
    language: string;
    status: string;
    score: number;
    time: number;
    memory: number;
    date: string;
}

export interface SubmissionDetail {
    id: number;
    problem: string;
    problem_name?: string;
    user: string;
    date: string;
    language: string;
    language_name?: string;
    status: string;
    result: string | null;
    points: number | null;
    time: number | null;
    memory: number | null;
    source: string;
    case_points: number;
    case_total: number;
    test_cases: TestCase[];
    error?: string;
}

export interface TestCase {
    case: number;
    status: string;
    time: number | null;
    memory: number | null;
    points: number | null;
    total: number | null;
    feedback: string;
}

export interface User {
    id: number;
    username: string;
    is_admin: boolean;
    is_staff: boolean;
    rating?: number | null;
    performance_points?: number;
}

export interface UserDetail {
    username: string;
    display_name: string;
    about: string | null;
    avatar_url: string;
    points: number;
    performance_points: number;
    contribution_points: number;
    rating: number | null;
    problem_count: number;
    display_rank: string;
    rank: number;
    rating_rank: number | null;
    email_hash: string;
    organizations: { id: number; name: string }[];
    last_access: string;
    date_joined: string;
    roles?: { id: number; name: string; display_name: string; color: string }[];
}

export interface Contest {
    key: string;
    name: string;
    start_time: string;
    end_time: string;
    is_rated: boolean;
    format?: string;
    time_limit?: number;
    user_count: number;
    is_joined: boolean;
}

export interface ContestDetail extends Contest {
    description: string;
    summary: string;
    is_joined: boolean;
    problems: {
        code: string;
        name: string;
        points: number;
        order: number;
        ac_rate: number;
        is_solved: boolean;
    }[];
}

export interface RankingRow {
    username: string;
    score: number;
    cumtime: number;
    rank: number;
    rating?: number | null;
    rating_change?: number | null;
    performance?: number | null;
    breakdown: any[];
}

export interface RankingResponse {
    contest: string;
    problems: { label: string; points: number }[];
    rankings: RankingRow[];
}
export interface Comment {
    id: number;
    body: string;
    score: number;
    time: string;
    parent_id?: number;
    author: string;
}

export interface CommentCreateRequest {
    page: string;
    body: string;
    parent_id?: number;
}

export interface BlogPost {
    id: number;
    title: string;
    slug: string;
    authors: { username: string }[];
    publish_on: string;
    summary: string;
    content?: string;
    score: number;
    visible: boolean;
    sticky: boolean;
    comment_count?: number;
}

export interface BlogPostDetail extends BlogPost {
    content: string;
}

export interface SolvedProblem {
    code: string;
    points: number;
}

export interface RatingHistoryEntry {
    date: string;
    rating: number;
    contest: string;
    contest_key: string;
}

export interface RatingLeaderboardEntry {
    rank: number;
    username: string;
    rating: number;
    contests_attended: number;
    highest_rating: number;
    avatar_url: string;
}

export interface RatingLeaderboardResponse {
    total: number;
    page: number;
    limit: number;
    data: RatingLeaderboardEntry[];
}

export interface RatingDetail {
    username: string;
    current_rating: number | null;
    rating_rank: number;
    contests_attended: number;
    highest_rating: number;
    lowest_rating: number;
    recent_changes: RatingChange[];
}

export interface RatingChange {
    date: string;
    contest: string;
    contest_key: string;
    rank: number;
    rating: number;
    performance: number;
}

// ============================================================
// ADMIN TYPES
// ============================================================

export interface AdminUser {
    id: number;
    username: string;
    email: string;
    points: number;
    performance_points: number;
    problem_count: number;
    rating: number | null;
    is_staff: boolean;
    is_super_user: boolean;
    is_active: boolean;
    is_unlisted: boolean;
    is_muted: boolean;
    date_joined: string;
    last_access: string;
    display_rank: string;
    ban_reason: string | null;
}

export interface AdminContest {
    id: number;
    key: string;
    name: string;
    description: string;
    summary: string;
    start_time: string;
    end_time: string;
    is_visible: boolean;
    is_rated: boolean;
    user_count: number;
    format_name: string;
    format_config: any;
    is_organization_private: boolean;
    hide_problem_tags: boolean;
    run_pretests_only: boolean;
    time_limit: number | null;
    access_code: string | null;
    author_ids: number[];
    curator_ids: number[];
    tester_ids: number[];
}

export interface AdminProblem {
    id: number;
    code: string;
    name: string;
    description: string;
    points: number;
    partial: boolean;
    is_public: boolean;
    group_id: number;
    group_name: string;
    user_count: number;
    ac_rate: number;
    is_manually_managed: boolean;
    time_limit: number;
    memory_limit: number;
    type_ids: number[];
    allowed_langs: any[];
    pdf_url: string;
}

export interface AdminJudge {
    id: number;
    name: string;
    online: boolean;
    is_blocked: boolean;
    auth_key: string;
    last_ip: string;
}

export interface AdminOrganization {
    id: number;
    name: string;
    slug: string;
    short_name: string;
    is_open: boolean;
    is_unlisted: boolean;
    member_count: number;
}

export interface AdminSubmission {
    id: number;
    user: string;
    problem: string;
    language: string;
    status: string;
    result: string | null;
    score: number | null;
    time: number | null;
    memory: number | null;
    date: string;
    is_pretested: boolean;
}

export interface AdminUserUpdateRequest {
    email?: string;
    display_name?: string;
    about?: string;
    is_active?: boolean;
    is_unlisted?: boolean;
    is_muted?: boolean;
    display_rank?: string;
    ban_reason?: string;
    remove_organization_ids?: number[];
    add_organization_ids?: number[];
}

export interface AdminContestCreateRequest {
    key: string;
    name: string;
    description: string;
    summary?: string;
    start_time: string;
    end_time: string;
    time_limit?: number;
    is_visible?: boolean;
    is_rated?: boolean;
    format_name?: string;
    format_config?: string;
    access_code?: string;
    hide_problem_tags?: boolean;
    run_pretests_only?: boolean;
    is_organization_private?: boolean;
    author_ids?: number[];
    curator_ids?: number[];
    tester_ids?: number[];
    problem_ids?: number[];
}

export interface AdminProblemCreateRequest {
    code: string;
    name: string;
    description: string;
    points: number;
    partial?: boolean;
    is_public?: boolean;
    time_limit: number;
    memory_limit: number;
    group_id?: number;
    type_ids?: number[];
    author_ids?: number[];
    allowed_lang_ids?: number[];
    is_manually_managed?: boolean;
    pdf_url?: string;
}

// ============================================================
// PUBLIC ORGANIZATION TYPES
// ============================================================

export interface Organization {
    id: number;
    name: string;
    slug: string;
    short_name: string;
    about?: string;
    is_open: boolean;
    is_unlisted: boolean;
    member_count: number;
}

export interface OrganizationDetail extends Organization {
    members: OrganizationMember[];
    admins: OrganizationMember[];
}

export interface OrganizationMember {
    id: number;
    username: string;
    display_name?: string;
    role: string;
    joined_at: string;
}

// ============================================================
// TICKET TYPES
// ============================================================

export interface Ticket {
    id: number;
    title: string;
    created_on: string;
    updated_on: string;
    is_closed: boolean;
    user: {
        id: number;
        username: string;
    };
    problem?: {
        code: string;
        name: string;
    };
    message_count: number;
}

export interface TicketDetail extends Ticket {
    messages: TicketMessage[];
}

export interface TicketMessage {
    id: number;
    body: string;
    time: string;
    user: {
        id: number;
        username: string;
        is_staff?: boolean;
    };
}

export interface TicketCreateRequest {
    title: string;
    body: string;
    problem_code?: string;
}

// ============================================================
// USER LIST TYPES
// ============================================================

export interface UserListItem {
    id: number;
    username: string;
    display_name?: string;
    points: number;
    performance_points: number;
    rating?: number | null;
    problem_count: number;
    display_rank: string;
    organizations: { id: number; name: string }[];
}

// ============================================================
// ROLE & PERMISSION TYPES
// ============================================================

export interface Permission {
    id: number;
    code: string;
    name: string;
    description?: string;
    category: string;
}

export interface Role {
    id: number;
    name: string;
    display_name: string;
    description?: string;
    color: string;
    is_default: boolean;
    permissions: Permission[];
}

export interface RoleWithUserCount extends Role {
    user_count: number;
}
