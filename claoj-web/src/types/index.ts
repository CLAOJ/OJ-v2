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
    tags?: { id: number; name: string; color: string }[];
}

export interface ContestCalendarItem {
    key: string;
    name: string;
    start_time: string;
    end_time: string;
    is_rated: boolean;
    format: string;
    day: number;
}

export interface ContestCalendarResponse {
    year: number;
    month: number;
    month_name: string;
    days_in_month: number;
    first_day_of_week: number;
    contests: ContestCalendarItem[];
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
    is_disqualified?: boolean;
    participation_id?: number;
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
    hidden?: boolean;
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

export interface BlogVoteRequest {
    delta: 1 | -1;
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

export interface ContestTag {
    id: number;
    name: string;
    color: string;
    description: string;
}

export interface AdminContestTag extends ContestTag {}

export interface AdminContestTagCreateRequest {
    name: string;
    color: string;
    description?: string;
}

export interface AdminContestTagUpdateRequest {
    name?: string;
    color?: string;
    description?: string;
}

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
    slots?: number;
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
    tag_ids?: number[];
    locked_after?: string | null;
    max_submissions?: number | null;
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
    is_disabled: boolean;
    auth_key: string;
    last_ip: string;
    ping?: number;
    load?: number;
}

export interface AdminJudgeDetail extends AdminJudge {
    start_time?: string;
    description: string;
    problems: { code: string; name: string }[];
    runtimes: { key: string; name: string; version: string }[];
}

export interface AdminJudgeUpdateRequest {
    description?: string;
    problem_ids?: number[];
    runtime_ids?: number[];
}

export interface AdminOrganization {
    id: number;
    name: string;
    slug: string;
    short_name: string;
    is_open: boolean;
    is_unlisted: boolean;
    slots?: number;
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
    max_submissions?: number | null;
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
    slots?: number;
    member_count: number;
}

export interface OrganizationDetail extends Organization {
    user_id: number;
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
    is_contributive?: boolean;
    user: {
        id: number;
        username: string;
    };
    problem?: {
        code: string;
        name: string;
    };
    message_count: number;
    assignees?: string[];
}

export interface TicketDetail extends Ticket {
    messages: TicketMessage[];
    notes?: string;
    linked_item?: {
        type: 'problem' | 'contest';
        code?: string;
        key?: string;
        name: string;
    };
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

export interface AdminTicket {
    id: number;
    title: string;
    is_open: boolean;
    is_contributive: boolean;
    created: string;
    user: string;
    assignees: string[];
    notes: string;
}

export interface AdminTicketDetail {
    id: number;
    title: string;
    creator: string;
    is_open: boolean;
    is_contributive: boolean;
    notes: string;
    created: string;
    assignees: string[];
    messages: TicketMessage[];
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

// ============================================================
// PROBLEM DATA MANAGEMENT TYPES
// ============================================================

export interface ProblemTestCase {
    id: number;
    order: number;
    input_file: string;
    output_file: string;
    points?: number;
    is_pretest?: boolean;
}

export interface ProblemData {
    code: string;
    test_cases: ProblemTestCase[];
    checker: string;
    grader: string;
    generator: string;
    feedback: string;
    checker_args?: string;
    grader_args?: string;
    has_custom_checker: boolean;
    has_custom_grader: boolean;
    has_custom_header: boolean;
    has_generator_yml: boolean;
    has_init_yml: boolean;
}

export interface ProblemDataFile {
    name: string;
    path: string;
    type: 'file' | 'directory';
    size?: number;
    is_testcase?: boolean;
}

export interface ProblemDataUpdateRequest {
    checker_data?: string;
    grader_data?: string;
    header_data?: string;
    generator_data?: string;
    generator_yml_data?: string;
    feedback?: string;
    checker_args?: string;
    grader_args?: string;
}

export interface TestCaseContent {
    id: number;
    order: number;
    input_data: string;
    output_data: string;
    encoding: string;
}

export interface TestCaseUpdateRequest {
    input_data?: string;
    output_data?: string;
    order?: number;
    points?: number;
    is_pretest?: boolean;
    type?: string;
}

export interface TestCaseReorderRequest {
    test_cases: { id: number; order: number }[];
}

// ============================================================
// SOLUTION/EDITORIAL TYPES
// ============================================================

export interface Solution {
    id: number;
    problem_id: number;
    content: string;
    summary: string;
    authors: { id: number; username: string }[];
    is_official: boolean;
    publish_on?: string;
    language: string;
}

export interface AdminSolution {
    id: number;
    problem_id: number;
    problem_code: string;
    problem_name: string;
    summary: string;
    is_public: boolean;
    is_official: boolean;
    publish_on?: string;
    language: string;
}

export interface AdminSolutionDetail extends Solution {
    problem_code: string;
    problem_name: string;
    is_public: boolean;
    valid_until?: string;
}

export interface SolutionCreateRequest {
    problem_id: number;
    content: string;
    summary?: string;
    author_ids?: number[];
    is_public?: boolean;
    is_official?: boolean;
    publish_on?: string;
    valid_until?: string;
    language?: string;
}

export interface SolutionUpdateRequest {
    content?: string;
    summary?: string;
    author_ids?: number[];
    is_public?: boolean;
    is_official?: boolean;
    publish_on?: string;
    valid_until?: string;
    language?: string;
}

export interface CommentRevision {
    id: number;
    editor: string;
    time: string;
    body: string;
    reason?: string;
}

export interface CommentUpdateRequest {
    body: string;
    reason?: string;
}

// ============================================================
// API TOKEN TYPES
// ============================================================

export interface APITokenResponse {
    has_token: boolean;
    token: string;
    message?: string;
    created_at?: string | null;
}

export interface APITokenGenerateResponse {
    token: string;
    message: string;
    warning: string;
}

// ============================================================
// WEBAUTHN TYPES
// ============================================================

export interface WebAuthnCredential {
    id: number;
    name: string;
    cred_id: string;
    counter: number;
}

export interface WebAuthnStatus {
    enabled: boolean;
    credentials_count: number;
}

export interface WebAuthnSetupResponse {
    options: PublicKeyCredentialCreationOptions;
}

export interface WebAuthnLoginResponse {
    options: PublicKeyCredentialRequestOptions;
    username: string;
}

export interface WebAuthnCredentialCreationResponse {
    id: string;
    rawId: number[];
    type: string;
    clientExtensionResults: any;
    response: {
        attestationObject: number[];
        clientDataJSON: number[];
    };
    name: string;
}

export interface WebAuthnCredentialAssertionResponse {
    id: string;
    rawId: number[];
    type: string;
    clientExtensionResults: any;
    response: {
        authenticatorData: number[];
        clientDataJSON: number[];
        signature: number[];
        userHandle: number[] | null;
    };
}

// ============================================================
// PROBLEM SUGGESTION TYPES (Task #31)
// ============================================================

export interface ProblemSuggestion {
    id: number;
    code: string;
    name: string;
    points: number;
    time_limit: number;
    memory_limit: number;
    group: string;
    suggestion_status: 'none' | 'pending' | 'approved' | 'rejected';
    suggester_id?: number;
    date?: string;
}

export interface ProblemSuggestionAdmin {
    id: number;
    code: string;
    name: string;
    points: number;
    suggestion_status: 'none' | 'pending' | 'approved' | 'rejected';
    suggester_id?: number;
    suggester_username: string;
    suggestion_notes: string;
    suggestion_reviewed_at?: string;
    suggestion_reviewed_by_id?: number;
    is_public: boolean;
    date?: string;
}

export interface ProblemSuggestionDetail {
    id: number;
    code: string;
    name: string;
    description: string;
    points: number;
    partial: boolean;
    time_limit: number;
    memory_limit: number;
    group_id: number;
    group: string;
    types: { name: string }[];
    source: string;
    summary: string;
    pdf_url: string;
    is_full_markup: boolean;
    short_circuit: boolean;
    suggestion_status: 'none' | 'pending' | 'approved' | 'rejected';
    suggestion_notes: string;
    suggestion_reviewed_at?: string;
    suggester_id?: number;
    suggester_username: string;
    suggester_email: string;
    is_public: boolean;
    authors: { username: string }[];
}

export interface ProblemSuggestRequest {
    name: string;
    description: string;
    points: number;
    partial?: boolean;
    time_limit: number;
    memory_limit: number;
    group_id: number;
    type_ids?: number[];
    source?: string;
    summary?: string;
    pdf_url?: string;
    is_full_markup?: boolean;
    short_circuit?: boolean;
    additional_notes?: string;
}

export interface ApproveSuggestionRequest {
    code: string;
    admin_notes?: string;
    is_public?: boolean;
    make_full_markup?: boolean;
}

export interface RejectSuggestionRequest {
    admin_notes?: string;
    reason: string;
}

// ============================================================
// PROBLEM CLARIFICATION TYPES (Task #33)
// ============================================================

export interface ProblemClarification {
    id: number;
    description: string;
    date: string;
}

// ============================================================
// ADMIN COMMENT TYPES
// ============================================================

export interface AdminComment {
    id: number;
    author_id: number;
    username: string;
    page: string;
    body: string;
    score: number;
    hidden: boolean;
    time: string;
    parent_id?: number;
}

export interface AdminCommentUpdateRequest {
    body?: string;
    hidden?: boolean;
    reason?: string;
}

// ============================================================
// ADMIN LANGUAGE TYPES
// ============================================================

export interface AdminLanguage {
    id: number;
    key: string;
    name: string;
    short_name?: string;
    common_name: string;
    ace: string;
    pygments: string;
    extension: string;
    file_only: boolean;
    file_size_limit: number;
    include_in_problem: boolean;
    info: string;
}

export interface AdminLanguageDetail extends AdminLanguage {
    template: string;
    description: string;
}

export interface AdminLanguageCreateRequest {
    key: string;
    name: string;
    short_name?: string;
    common_name: string;
    ace: string;
    pygments: string;
    template?: string;
    description?: string;
    extension: string;
    file_only?: boolean;
    file_size_limit?: number;
    include_in_problem?: boolean;
    info?: string;
}

export interface AdminLanguageUpdateRequest {
    name?: string;
    short_name?: string;
    common_name?: string;
    ace?: string;
    pygments?: string;
    template?: string;
    description?: string;
    extension?: string;
    file_only?: boolean;
    file_size_limit?: number;
    include_in_problem?: boolean;
    info?: string;
}

// ============================================================
// ADMIN BLOG POST TYPES
// ============================================================

export interface AdminBlogPost {
    id: number;
    title: string;
    slug: string;
    author_names: string[];
    publish_on: string;
    visible: boolean;
    sticky: boolean;
    global_post: boolean;
    organization?: string;
    score: number;
}

export interface AdminBlogPostDetail {
    id: number;
    title: string;
    slug: string;
    content: string;
    summary: string;
    author_ids: number[];
    author_names: string[];
    publish_on: string;
    visible: boolean;
    sticky: boolean;
    global_post: boolean;
    og_image: string;
    organization_id?: number;
    organization_name?: string;
    score: number;
}

export interface AdminBlogPostCreateRequest {
    title: string;
    slug: string;
    content: string;
    summary: string;
    author_ids?: number[];
    publish_on: string;
    visible?: boolean;
    sticky?: boolean;
    global_post?: boolean;
    og_image?: string;
    organization_id?: number;
}

export interface AdminBlogPostUpdateRequest {
    title?: string;
    slug?: string;
    content?: string;
    summary?: string;
    author_ids?: number[];
    publish_on?: string;
    visible?: boolean;
    sticky?: boolean;
    global_post?: boolean;
    og_image?: string;
    organization_id?: number;
}

// ============================================================
// ADMIN LICENSE TYPES
// ============================================================

export interface AdminLicense {
    id: number;
    key: string;
    name: string;
    link: string;
    display: string;
    icon: string;
}

export interface AdminLicenseDetail {
    id: number;
    key: string;
    name: string;
    link: string;
    display: string;
    icon: string;
    text: string;
}

export interface AdminLicenseCreateRequest {
    key: string;
    link: string;
    name: string;
    display?: string;
    icon?: string;
    text?: string;
}

export interface AdminLicenseUpdateRequest {
    link?: string;
    name?: string;
    display?: string;
    icon?: string;
    text?: string;
}

// ============================================================
// ADMIN PROBLEM TAXONOMY TYPES
// ============================================================

export interface AdminProblemGroup {
    id: number;
    name: string;
    full_name: string;
}

export interface AdminProblemType {
    id: number;
    name: string;
    full_name: string;
}

export interface AdminProblemGroupCreateRequest {
    name: string;
    full_name: string;
}

export interface AdminProblemTypeCreateRequest {
    name: string;
    full_name: string;
}

export interface AdminProblemGroupUpdateRequest {
    full_name?: string;
}

export interface AdminProblemTypeUpdateRequest {
    full_name?: string;
}

// ============================================================
// ADMIN LIST RESPONSE TYPES
// ============================================================

export interface AdminListResponse<T> {
    data: T[];
    total: number;
}

export type AdminCommentListResponse = AdminListResponse<AdminComment>;
export type AdminLanguageListResponse = AdminListResponse<AdminLanguage>;
export type AdminBlogPostListResponse = AdminListResponse<AdminBlogPost>;
export type AdminLicenseListResponse = AdminListResponse<AdminLicense>;
export type AdminProblemGroupListResponse = AdminListResponse<AdminProblemGroup>;
export type AdminProblemTypeListResponse = AdminListResponse<AdminProblemType>;
export type AdminUserListResponse = AdminListResponse<AdminUser>;
export type AdminContestListResponse = AdminListResponse<AdminContest>;
export type AdminProblemListResponse = AdminListResponse<AdminProblem>;
export type AdminJudgeListResponse = AdminListResponse<AdminJudge>;
export type AdminOrganizationListResponse = AdminListResponse<AdminOrganization>;
export type AdminSubmissionListResponse = AdminListResponse<AdminSubmission>;
export type RoleListResponse = AdminListResponse<Role>;
export type AdminSolutionListResponse = AdminListResponse<AdminSolution>;
export type AdminTicketListResponse = AdminListResponse<AdminTicket>;

// ============================================================
// NAVIGATION BAR TYPES (Task #51)
// ============================================================

export interface AdminNavigationBar {
    id: number;
    key: string;
    label: string;
    path: string;
    parent_id?: number;
    order: number;
    level: number;
    parent?: {
        id: number;
        label: string;
    };
}

export interface AdminNavigationBarDetail {
    id: number;
    key: string;
    label: string;
    path: string;
    parent_id?: number;
    order: number;
    level: number;
    lft: number;
    rght: number;
    tree_id: number;
}

export interface AdminNavigationBarCreateRequest {
    key: string;
    label: string;
    path: string;
    parent_id?: number;
    order?: number;
}

export interface AdminNavigationBarUpdateRequest {
    label?: string;
    path?: string;
    order?: number;
}

export type AdminNavigationBarListResponse = AdminListResponse<AdminNavigationBar>;

// ============================================================
// ADMIN MISC CONFIG TYPES (Task #52)
// ============================================================

export interface AdminMiscConfig {
    id: number;
    key: string;
    value: string;
}

export interface AdminMiscConfigDetail {
    id: number;
    key: string;
    value: string;
}

export interface AdminMiscConfigCreateRequest {
    key: string;
    value?: string;
}


export interface AdminMiscConfigUpdateRequest {
    value: string;
}

export type AdminMiscConfigListResponse = AdminListResponse<AdminMiscConfig>;

// ============================================================
// ADMIN LANGUAGE LIMIT TYPES (Task #35)
// ============================================================

export interface LanguageLimit {
    id: number;
    problem_id: number;
    language_id: number;
    time_limit: number;
    memory_limit: number;
    problem?: { id: number; code: string; name: string };
    language?: { id: number; key: string; name: string };
}

export interface AdminLanguageLimit extends LanguageLimit {}

export interface AdminLanguageLimitCreateRequest {
    problem_id: number;
    language_id: number;
    time_limit: number;
    memory_limit: number;
}

export interface AdminLanguageLimitUpdateRequest {
    time_limit?: number;
    memory_limit?: number;
}

export type AdminLanguageLimitListResponse = AdminListResponse<AdminLanguageLimit>;
