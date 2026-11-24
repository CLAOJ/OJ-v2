export interface APIResponse<T> {
    status: string;
    data: T;
}

export interface PaginatedList<T> {
    data: T[];
    /** Total matching rows. Absent on endpoints that don't report a count. */
    total?: number;
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
    pdf_url?: string;
}

export interface Submission {
    id: number;
    user: string;
    problem: string;
    problem_name: string;
    language: string;
    // Lifecycle state from the judge: 'QU' | 'P' | 'G' | 'D' | 'CE' | 'IE' | 'AB'.
    // 'D' ("done") is not a verdict — the verdict lives in `result`.
    status: string;
    // Verdict once graded: 'AC' | 'WA' | 'TLE' | ... — null while still queued/grading.
    result: string | null;
    // Points earned (may be fractional); null while ungraded. The API returns
    // `points`, not `score`.
    points: number | null;
    time: number | null;
    memory: number | null;
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
    /** Superuser or holder of judge.organization_admin (GET /user/me). */
    can_organization_admin?: boolean;
    /** Present on GET /user/me; is_admin mirrors it. */
    is_superuser?: boolean;
    /** Effective Django permission codenames; staff only (GET /user/me). */
    permissions?: string[];
    /** True while a superuser is viewing the site as this account (GET /user/me). */
    is_impersonating?: boolean;
    /** Username of the superuser driving the impersonation, if any. */
    impersonator_username?: string;
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
    breakdown: BreakdownItem[];
    is_disqualified?: boolean;
    participation_id?: number;
}

export interface BreakdownItem {
    points: number;
    penalty?: number;
    solved?: boolean;
    label?: string;
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

export type CommentPageType = 'problem' | 'editorial' | 'blog' | 'all';

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

export interface BlogFeedItem {
    id: number;
    title: string;
    link: string;
    description: string;
    pubDate: string;
    author: string;
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

export interface AdminContestTag extends ContestTag {
    contest_count?: number;
    contest_ids?: number[];
    contests?: { id: number; key: string; name: string }[];
}

export interface AdminContestTagCreateRequest {
    name: string;
    color: string;
    description?: string;
    contest_ids?: number[];
}

export interface AdminContestTagUpdateRequest {
    name?: string;
    color?: string;
    description?: string;
    contest_ids?: number[];
}

export interface AdminUserGroupRef {
    id: number;
    name: string;
}

/** v1 django-registration-redux row shown read-only in the registrations admin. */
export interface AdminRegistrationV1 {
    /** registration_registrationprofile.id. */
    id: number;
    activated: boolean;
    has_activation_key: boolean;
}

/** Row shape of GET /admin/registrations. */
export interface AdminRegistration {
    /** auth_user.id — what the activate/resend endpoints take. */
    user_id: number;
    username: string;
    email: string;
    is_active: boolean;
    date_joined: string;
    registration: AdminRegistrationV1 | null;
    has_pending_token: boolean;
}

/** Row shape of GET /admin/users. */
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
    last_login: string | null;
    display_rank: string;
    ban_reason: string | null;
    groups: AdminUserGroupRef[];
}

export interface AdminUserPermissionRef {
    id: number;
    /** "app_label.codename", e.g. "judge.change_profile". */
    codename: string;
}

/** Detail shape of GET /admin/user/:id (no secrets). */
export interface AdminUserDetail {
    id: number;
    username: string;
    email: string;
    first_name: string;
    last_name: string;
    display_name: string;
    about: string;
    timezone: string;
    language_id: number | null;
    ace_theme: string;
    site_theme: string;
    notes: string;
    user_script: string;
    username_display_override: string;
    points: number;
    performance_points: number;
    contribution_points: number;
    rating: number | null;
    problem_count: number;
    display_rank: string;
    is_staff: boolean;
    is_super_user: boolean;
    is_superuser: boolean;
    is_active: boolean;
    is_unlisted: boolean;
    mute: boolean;
    is_muted: boolean;
    ban_reason: string | null;
    is_totp_enabled: boolean;
    is_webauthn_enabled: boolean;
    webauthn_credential_count: number;
    groups: AdminUserGroupRef[];
    group_ids: number[];
    permissions: AdminUserPermissionRef[];
    permission_ids: number[];
    organizations: { id: number; name?: string }[];
    last_access: string;
    last_login: string | null;
    date_joined: string;
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
    format_config: Record<string, unknown>;
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
}

// ============================================================
// ADMIN DETAIL CONTRACTS (flat snake_case, see api/v2/admin_problem.go,
// api/v2/admin_contest.go)
// ============================================================

export interface UserRef {
    id: number;
    username: string;
}

/** id/name pair for organization and class chips (see service/contest/types.go). */
export interface NamedRef {
    id: number;
    name: string;
}

/** v1 single-char codes, see judge/models/problem.py. */
export type SubmissionSourceVisibilityMode = 'A' | 'S' | 'O' | 'F';
export type TestcaseVisibilityMode = 'A' | 'C' | 'O';
/** v1 Contest.SCOREBOARD_VISIBILITY codes, see judge/models/contest.py. */
export type ScoreboardVisibility = 'V' | 'C' | 'P';

export interface AdminProblemDetail {
    id: number;
    code: string;
    name: string;
    description: string;
    points: number;
    partial: boolean;
    is_public: boolean;
    time_limit: number;
    memory_limit: number;
    group_id: number;
    is_manually_managed: boolean;
    pdf_url: string;
    author_ids: number[];
    curator_ids: number[];
    tester_ids: number[];
    type_ids: number[];
    allowed_lang_ids: number[];
    authors: UserRef[];
    curators: UserRef[];
    testers: UserRef[];
    source: string;
    date: string | null;
    license_id: number | null;
    og_image: string;
    summary: string;
    is_full_markup: boolean;
    submission_source_visibility_mode: SubmissionSourceVisibilityMode;
    testcase_visibility_mode: TestcaseVisibilityMode;
    is_organization_private: boolean;
    organization_ids: number[];
    banned_user_ids: number[];
    suggester_id: number | null;
    banned_users: UserRef[];
}

export interface AdminProblemUpdateRequest {
    name?: string;
    description?: string;
    points?: number;
    partial?: boolean;
    is_public?: boolean;
    time_limit?: number;
    memory_limit?: number;
    is_manually_managed?: boolean;
    pdf_url?: string;
    group_id?: number;
    author_ids?: number[];
    curator_ids?: number[];
    tester_ids?: number[];
    type_ids?: number[];
    allowed_lang_ids?: number[];
    source?: string;
    date?: string | null;
    license_id?: number | null;
    og_image?: string;
    summary?: string;
    is_full_markup?: boolean;
    submission_source_visibility_mode?: SubmissionSourceVisibilityMode;
    testcase_visibility_mode?: TestcaseVisibilityMode;
    is_organization_private?: boolean;
    organization_ids?: number[];
    banned_user_ids?: number[];
    suggester_id?: number | null;
    /** Nullable columns: a null alone means "no change" on PATCH. */
    clear_date?: boolean;
    clear_license_id?: boolean;
    clear_suggester_id?: boolean;
}

export interface ContestAnnouncement {
    id: number;
    contest_id: number;
    title: string;
    description: string;
    date: string;
}

export interface AdminContestProblemRow {
    id: number;
    problem_id: number;
    code: string;
    name: string;
    points: number;
    partial: boolean;
    is_pretested: boolean;
    order: number;
}

export interface AdminContestDetail {
    id: number;
    key: string;
    name: string;
    description: string;
    summary: string;
    start_time: string;
    end_time: string;
    /** Per-participant window in seconds; null = whole contest. */
    time_limit: number | null;
    is_visible: boolean;
    is_rated: boolean;
    format_name: string;
    format_config: Record<string, unknown> | null;
    access_code: string;
    hide_problem_tags: boolean;
    hide_problem_authors: boolean;
    run_pretests_only: boolean;
    show_short_display: boolean;
    use_clarifications: boolean;
    push_announcements: boolean;
    scoreboard_visibility: ScoreboardVisibility;
    scoreboard_cache_timeout: number;
    points_precision: number;
    rating_floor: number | null;
    rating_ceiling: number | null;
    performance_ceiling_override: number | null;
    rate_all: boolean;
    is_private: boolean;
    is_organization_private: boolean;
    limit_join_organizations: boolean;
    og_image: string;
    logo_override_image: string;
    problem_label_script: string;
    locked_after: string | null;
    author_ids: number[];
    curator_ids: number[];
    tester_ids: number[];
    tag_ids: number[];
    organization_ids: number[];
    banned_user_ids: number[];
    private_contestant_ids: number[];
    view_scoreboard_user_ids: number[];
    join_organization_ids: number[];
    class_ids: number[];
    authors: UserRef[];
    curators: UserRef[];
    testers: UserRef[];
    banned_users: UserRef[];
    private_contestants: UserRef[];
    view_scoreboard_users: UserRef[];
    join_organizations: NamedRef[];
    organizations: NamedRef[];
    classes: NamedRef[];
    problems: AdminContestProblemRow[];
}

/** One row of the ordered full problem list sent on contest update. */
export interface ContestProblemInput {
    problem_id: number;
    points?: number;
    partial?: boolean;
    is_pretested?: boolean;
}

export interface AdminContestUpdateRequest {
    name?: string;
    description?: string;
    summary?: string;
    start_time?: string;
    end_time?: string;
    /** Seconds; 0 clears. */
    time_limit?: number;
    is_visible?: boolean;
    is_rated?: boolean;
    format_name?: string;
    format_config?: string;
    access_code?: string;
    hide_problem_tags?: boolean;
    hide_problem_authors?: boolean;
    run_pretests_only?: boolean;
    show_short_display?: boolean;
    use_clarifications?: boolean;
    push_announcements?: boolean;
    scoreboard_visibility?: ScoreboardVisibility;
    scoreboard_cache_timeout?: number;
    points_precision?: number;
    rating_floor?: number;
    rating_ceiling?: number;
    performance_ceiling_override?: number;
    rate_all?: boolean;
    is_private?: boolean;
    is_organization_private?: boolean;
    limit_join_organizations?: boolean;
    og_image?: string;
    logo_override_image?: string;
    problem_label_script?: string;
    author_ids?: number[];
    curator_ids?: number[];
    tester_ids?: number[];
    tag_ids?: number[];
    organization_ids?: number[];
    banned_user_ids?: number[];
    private_contestant_ids?: number[];
    view_scoreboard_user_ids?: number[];
    join_organization_ids?: number[];
    class_ids?: number[];
    problems?: ContestProblemInput[];
    /** Nullable rating columns: an omitted value alone means "no change" on PATCH. */
    clear_rating_floor?: boolean;
    clear_rating_ceiling?: boolean;
    clear_performance_ceiling_override?: boolean;
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
    allowed_langs: number[];
    pdf_url: string;
    date: string | null;
    authors: UserRef[];
}

export interface AdminJudge {
    id: number;
    name: string;
    /** v1 tier: informational in v2, the queue has no tier routing. */
    tier: number;
    online: boolean;
    is_blocked: boolean;
    is_disabled: boolean;
    auth_key?: string;
    last_ip: string;
    ping?: number;
    load?: number;
}

export interface AdminJudgeDetail extends AdminJudge {
    start_time?: string;
    description: string;
    problems: { id: number; code: string; name: string }[];
    /** Language ids the judge is assigned (what the edit form writes). */
    problem_ids: number[];
    runtimes: { key: string; name: string; version: string }[];
    runtime_ids: number[];
    assigned_runtimes: { id: number; key: string; name: string }[];
}

export interface AdminJudgeCreateRequest {
    name: string;
    auth_key?: string;
    description?: string;
}

export interface AdminUserCreateRequest {
    username: string;
    email: string;
    password: string;
    full_name?: string;
    is_active?: boolean;
}

export interface AdminJudgeUpdateRequest {
    name?: string;
    auth_key?: string;
    tier?: number;
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
    logo_override_image?: string;
    member_count: number;
    performance_points: number;
}

/** GET /admin/organization/:id — the full editable shape. */
export interface AdminOrganizationDetail extends AdminOrganization {
    about: string;
    access_code: string | null;
    creation_date: string;
    admins: { id: number; username: string }[];
}

/** GET /admin/classes row (picker source for contest class ids). */
export interface AdminClass {
    id: number;
    name: string;
    slug: string;
    is_active: boolean;
    organization_id: number;
    organization_name: string;
}

export interface AdminSubmission {
    id: number;
    user: string;
    problem: string;
    problem_name?: string;
    language: string;
    status: string;
    result: string | null;
    score: number | null;
    time: number | null;
    memory: number | null;
    date: string;
    is_pretested: boolean;
}

/** GET /admin/submission/:id */
export interface AdminSubmissionTestCase {
    id: number;
    case: number;
    status: string;
    time: number | null;
    memory: number | null;
    points: number | null;
    total: number | null;
    feedback: string;
}

export interface AdminSubmissionDetail {
    id: number;
    user: string;
    problem: string;
    problem_name: string;
    language_id: number;
    language: string;
    language_name: string;
    date: string;
    time: number | null;
    memory: number | null;
    points: number | null;
    status: string;
    result: string | null;
    error: string | null;
    case_points: number;
    case_total: number;
    current_testcase: number;
    batch: boolean;
    is_pretested: boolean;
    judged_on_id: number | null;
    judged_on: string;
    judged_date: string | null;
    rejudged_date: string | null;
    locked_after: string | null;
    is_locked: boolean;
    is_archived: boolean;
    contest_object_id: number | null;
    contest_key: string;
    contest_name: string;
    source: string;
    test_cases: AdminSubmissionTestCase[];
}

/** PATCH /admin/submission/:id */
export interface AdminSubmissionUpdateRequest {
    language_id?: number;
    status?: string;
    result?: string | null;
    points?: number;
    case_points?: number;
    case_total?: number;
    judged_on_id?: number | null;
    judged_date?: string | null;
    error?: string;
    contest_object_id?: number | null;
    source?: string;
    locked_after?: string | null;
    is_archived?: boolean;
}

export interface AdminUserUpdateRequest {
    username?: string;
    email?: string;
    first_name?: string;
    last_name?: string;
    is_active?: boolean;
    is_staff?: boolean;
    is_superuser?: boolean;
    username_display_override?: string;
    about?: string;
    notes?: string;
    user_script?: string;
    timezone?: string;
    language_id?: number;
    ace_theme?: string;
    site_theme?: string;
    mute?: boolean;
    is_unlisted?: boolean;
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
    hide_problem_authors?: boolean;
    run_pretests_only?: boolean;
    show_short_display?: boolean;
    use_clarifications?: boolean;
    push_announcements?: boolean;
    scoreboard_visibility?: ScoreboardVisibility;
    scoreboard_cache_timeout?: number;
    points_precision?: number;
    rating_floor?: number;
    rating_ceiling?: number;
    performance_ceiling_override?: number;
    rate_all?: boolean;
    is_private?: boolean;
    is_organization_private?: boolean;
    limit_join_organizations?: boolean;
    og_image?: string;
    logo_override_image?: string;
    problem_label_script?: string;
    author_ids?: number[];
    curator_ids?: number[];
    tester_ids?: number[];
    tag_ids?: number[];
    organization_ids?: number[];
    banned_user_ids?: number[];
    private_contestant_ids?: number[];
    view_scoreboard_user_ids?: number[];
    join_organization_ids?: number[];
    class_ids?: number[];
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
    source?: string;
    date?: string | null;
    license_id?: number | null;
    og_image?: string;
    summary?: string;
    is_full_markup?: boolean;
    submission_source_visibility_mode?: SubmissionSourceVisibilityMode;
    testcase_visibility_mode?: TestcaseVisibilityMode;
    is_organization_private?: boolean;
    organization_ids?: number[];
    banned_user_ids?: number[];
    suggester_id?: number | null;
    curator_ids?: number[];
    tester_ids?: number[];
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

// Matches GET /api/tickets item shape: { id, title, is_open, is_contributive, created, assignees }
export interface Ticket {
    id: number;
    title: string;
    is_open: boolean;
    is_contributive?: boolean;
    created: string;
    assignees?: string[];
}

// Matches GET /api/ticket/:id: flat object with creator (username string) and messages
export interface TicketDetail extends Ticket {
    creator: string;
    messages: TicketMessage[];
    notes?: string;
    linked_item?: {
        type: 'problem' | 'contest';
        code?: string;
        key?: string;
        name: string;
    };
}

// Matches ticket message shape: { id, user_id, body, time, username }
export interface TicketMessage {
    id: number;
    /** Author's judge_profile id; absent from the public GET /ticket/:id payload. */
    user_id?: number;
    body: string;
    time: string;
    username: string;
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
    /** judge_profile.id of the creator, for the user picker. */
    creator_id: number;
    is_open: boolean;
    is_contributive: boolean;
    notes: string;
    created: string;
    assignees: string[];
    messages: TicketMessage[];
    linked_type: 'problem' | 'contest' | '';
    linked_id: number;
    /** "code – name" / "key – name", empty when unlinked. */
    linked_label: string;
    /** Public site path, e.g. /problem/P1, empty when unlinked. */
    linked_path: string;
}

export interface AdminTicketUpdateRequest {
    title?: string;
    user_id?: number;
    linked_type?: 'problem' | 'contest' | '';
    linked_id?: number;
}

export interface AdminTicketMessageUpdateRequest {
    body?: string;
    /** judge_profile id of the message author. */
    user_id?: number;
}

// ============================================================
// USER LIST TYPES
// ============================================================

// Matches GET /api/users item shape (no id/display_name/organizations in the payload)
export interface UserListItem {
    username: string;
    display_name?: string;
    points: number;
    performance_points: number;
    rating?: number | null;
    problem_count: number;
    contribution_points: number;
    display_rank: string;
    avatar_url?: string;
    organizations?: { id: number; name: string; slug: string }[];
}

// ============================================================
// GROUP & PERMISSION TYPES
// ============================================================

export interface PermissionInfo {
    id: number;
    codename: string;
    name: string;
    app_label: string;
    model: string;
}

export interface Group {
    id: number;
    name: string;
    user_count: number;
    permission_count: number;
}

export interface GroupDetail {
    id: number;
    name: string;
    permission_ids: number[];
    users: { id: number; username: string }[];
}

export interface GroupMember {
    id: number;
    username: string;
    email: string;
    is_active: boolean;
    points: number;
}

export interface GroupMembersResponse extends AdminListResponse<GroupMember> {
    group_id: number;
    group_name: string;
    page: number;
    page_size: number;
    has_more: boolean;
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
    clientExtensionResults: Record<string, unknown>;
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
    clientExtensionResults: Record<string, unknown>;
    response: {
        authenticatorData: number[];
        clientDataJSON: number[];
        signature: number[];
        userHandle: number[] | null;
    };
}

// WebAuthn credential summary for list display
export interface WebAuthnCredentialSummary {
    id: number;
    name: string;
    created_at: string;
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
// PROBLEM TRANSLATION TYPES
// ============================================================

export interface ProblemTranslation {
    id: number;
    problem_id: number;
    language: string;
    name: string;
    description: string;
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
    revisions: number;
}

export interface AdminCommentUpdateRequest {
    body?: string;
    hidden?: boolean;
    author_id?: number;
    page?: string;
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
    template: string;
    description: string;
}

export type AdminLanguageDetail = AdminLanguage;

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
    organization_id?: number;
    organization_name?: string;
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
    organization_id: number | null;
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
    /** null detaches the post from any organization. */
    organization_id?: number | null;
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
    /** null clears the organization; omitting the key leaves it unchanged. */
    organization_id?: number | null;
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
    problem_count?: number;
}

export interface AdminProblemType {
    id: number;
    name: string;
    full_name: string;
    problem_count?: number;
    problem_ids?: number[];
    problems?: { id: number; code: string; name: string }[];
}

export interface AdminProblemGroupCreateRequest {
    name: string;
    full_name: string;
}

export interface AdminProblemTypeCreateRequest {
    name: string;
    full_name: string;
    problem_ids?: number[];
}

export interface AdminProblemGroupUpdateRequest {
    full_name?: string;
}

export interface AdminProblemTypeUpdateRequest {
    full_name?: string;
    problem_ids?: number[];
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
export type AdminRegistrationListResponse = AdminListResponse<AdminRegistration>;
export type AdminContestListResponse = AdminListResponse<AdminContest>;
export type AdminProblemListResponse = AdminListResponse<AdminProblem>;
export type AdminJudgeListResponse = AdminListResponse<AdminJudge>;
export type AdminOrganizationListResponse = AdminListResponse<AdminOrganization>;
export type AdminClassListResponse = AdminListResponse<AdminClass>;
export type AdminSubmissionListResponse = AdminListResponse<AdminSubmission>;
export type AdminSolutionListResponse = AdminListResponse<AdminSolution>;
// GET /api/admin/tickets returns { items, total } (not { data, total })
export interface AdminTicketListResponse {
    items: AdminTicket[];
    total: number;
}

// ============================================================
// NAVIGATION BAR TYPES (Task #51)
// ============================================================

export interface AdminNavigationBar {
    id: number;
    key: string;
    label: string;
    path: string;
    regex: string;
    parent_id?: number | null;
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
    regex: string;
    parent_id?: number | null;
    parent_label?: string;
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
    regex?: string;
    parent_id?: number;
    order?: number;
}

export interface AdminNavigationBarUpdateRequest {
    label?: string;
    path?: string;
    regex?: string;
    parent_id?: number | null;
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
// ADMIN FLAT PAGE + REDIRECT TYPES (v1 contrib tables)
// ============================================================

export interface AdminFlatPage {
    id: number;
    url: string;
    title: string;
    registration_required: boolean;
}

export interface AdminFlatPageDetail extends AdminFlatPage {
    content: string;
    enable_comments: boolean;
    template_name: string;
}

export interface AdminFlatPageCreateRequest {
    url: string;
    title: string;
    content?: string;
    registration_required?: boolean;
}

export interface AdminFlatPageUpdateRequest {
    url?: string;
    title?: string;
    content?: string;
    registration_required?: boolean;
}

export type AdminFlatPageListResponse = AdminListResponse<AdminFlatPage>;

export interface AdminRedirect {
    id: number;
    old_path: string;
    new_path: string;
}

export interface AdminRedirectCreateRequest {
    old_path: string;
    new_path: string;
}

export interface AdminRedirectUpdateRequest {
    old_path?: string;
    new_path?: string;
}

export type AdminRedirectListResponse = AdminListResponse<AdminRedirect>;

// ============================================================
// ADMIN LANGUAGE LIMIT TYPES (Task #35)
// ============================================================

export interface LanguageLimit {
    id: number;
    problem_id: number;
    problem_code: string;
    problem_name: string;
    language_id: number;
    language_key: string;
    language_name: string;
    time_limit: number;
    memory_limit: number;
}

export type AdminLanguageLimit = LanguageLimit;

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

// ============================================================
// PERFORMANCE POINTS BREAKDOWN TYPES (Task #42)
// ============================================================

export interface PPBreakdownProblem {
    code: string;
    name: string;
    points: number;
    weight: number;
    contribution: number;
}

export interface PPBreakdown {
    performance_points: number;
    problems: PPBreakdownProblem[];
    bonus: {
        solved_count: number;
        bonus_points: number;
    };
    weighted_sum: number;
    total: number;
}

// ============================================================
// CONTEST STATISTICS TYPES (Task #24)
// ============================================================

export interface ContestProblemStats {
    problem_code: string;
    problem_label: string;
    points: number;
    max_score: number;
    is_solved: boolean;
    attempt_count: number;
    first_submit_time: string | null;
    solve_time: number | null;
    time_in_seconds: number;
}

export interface ContestStats {
    contest_key: string;
    contest_name: string;
    participation_id: number;
    score: number;
    cumtime: number;
    rank: number;
    total_participants: number;
    percentile: number;
    average_score: number;
    solved_count: number;
    total_problems: number;
    total_attempts: number;
    average_solve_time: number;
    problems: ContestProblemStats[];
    average_solve_times_by_problem: Record<string, number>;
}
