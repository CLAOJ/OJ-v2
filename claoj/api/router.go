// Package api wires the Gin HTTP engine and attaches all route groups.
package api

import (
	"github.com/CLAOJ/claoj/api/v2"
	adminHandlers "github.com/CLAOJ/claoj/api/v2/admin"
	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	contestHandlers "github.com/CLAOJ/claoj/api/v2/contest"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/bridge"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/csrf"
	"github.com/CLAOJ/claoj/events"
	"github.com/CLAOJ/claoj/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter creates and returns the fully configured Gin engine.
func NewRouter() *gin.Engine {
	r := gin.New()

	// Recovery middleware - recovers from panics
	r.Use(middleware.RecoveryMiddleware())

	// Logger middleware
	r.Use(gin.Logger())

	// Error handler middleware
	r.Use(middleware.ErrorHandlerMiddleware())

	// CORS middleware - only allow configured origins
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{config.C.App.SiteFullURL}
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Authorization", "Content-Type", "X-Requested-With", "X-CSRFToken", "X-CSRF-Token"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	r.Use(cors.New(corsConfig))

	// Prometheus metrics middleware
	r.Use(middleware.Metrics())

	// Health-check endpoint
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	// Prometheus metrics endpoint (public - can be restricted by network config)
	r.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})

	// Judge agents POST grading packets here. Deliberately outside /api/:
	// the public nginx/ingress only forwards /api/ to the backend.
	r.POST("/internal/judge/event", bridge.EventHandler)

	apiv2 := r.Group("/api")
	{
		// Public Auth
		apiv2.POST("/auth/login", authHandlers.Login)
		apiv2.POST("/auth/logout", authHandlers.Logout)
		apiv2.POST("/auth/refresh", authHandlers.Refresh)
		apiv2.POST("/auth/register", v2.Register)
		apiv2.POST("/auth/password/reset", authHandlers.PasswordResetRequest)
		apiv2.POST("/auth/password/reset/confirm", authHandlers.PasswordResetConfirm)
		// Email verification
		apiv2.POST("/auth/verify-email", v2.VerifyEmail)
		apiv2.POST("/auth/resend-verification", v2.ResendVerification)
		// TOTP/2FA — login-completion steps only. These run pre-session (the
		// user has passed the password step but has no access token yet), so
		// they must stay public. The management routes (status/setup/confirm/
		// disable/backup-codes) require an authenticated session and live in
		// the protected group below.
		apiv2.POST("/auth/totp/verify", v2.TotpVerify)
		apiv2.POST("/auth/totp/verify-backup", v2.TotpBackupVerify)
		// WebAuthn/2FA
		apiv2.GET("/auth/webauthn/status", v2.WebAuthnStatus)
		apiv2.POST("/auth/webauthn/register/begin", v2.WebAuthnBeginRegistration)
		apiv2.POST("/auth/webauthn/register/finish", v2.WebAuthnFinishRegistration)
		apiv2.POST("/auth/webauthn/login/begin", v2.WebAuthnBeginLogin)
		apiv2.POST("/auth/webauthn/login/finish", v2.WebAuthnFinishLogin)
		// WebAuthn Credentials (Protected)
		// OAuth
		apiv2.GET("/auth/oauth/:provider", authHandlers.OAuthStart)
		apiv2.POST("/auth/oauth/:provider/callback", authHandlers.OAuthCallback)

		// Optional Auth (for user-specific logic on public pages like problem solved status)
		// We'll apply this globally to the main apiv2 group since it doesn't block unauthenticated requests.
		apiv2.Use(auth.OptionalMiddleware())

		// =======================================================
		// ADMIN ROUTES (Admin Required)
		// =======================================================
		admin := apiv2.Group("")
		admin.Use(auth.AdminRequiredMiddleware())
		// Session cookies are SameSite=None, so without this an admin write
		// is forgeable cross-site. The web client echoes the csrf_token
		// cookie in X-CSRF-Token on every non-GET request.
		admin.Use(csrf.Middleware(csrf.DefaultConfig()))
		admin.Use(middleware.Audit()) // Audit logging for all admin actions
		{
			// Users
			admin.GET("/admin/users", v2.AdminUserList)
			admin.GET("/admin/user/:id", v2.AdminUserDetail)
			admin.PATCH("/admin/user/:id", v2.AdminUserUpdate)
			admin.DELETE("/admin/user/:id", v2.AdminUserDelete)
			admin.POST("/admin/user/:id/ban", v2.AdminUserBan)
			admin.POST("/admin/user/:id/unban", v2.AdminUserUnban)
			// Impersonation start lives in the CSRF-protected group below:
			// it mutates the session, so it must carry the double-submit
			// token. The admin gate + audit are re-applied per route there.
			admin.POST("/admin/users", auth.RequirePerm("auth.add_user"), v2.AdminUserCreate)
			admin.POST("/admin/user/:id/password", auth.RequirePerm("auth.change_user"), v2.AdminUserSetPassword)

			// Registrations / account activation (v1 registration profiles +
			// v2 email-verification tokens). Mutations are superuser-only
			// inside the handlers.
			admin.GET("/admin/registrations", v2.AdminRegistrationList)
			admin.POST("/admin/registration/:userId/activate", v2.AdminRegistrationActivate)
			admin.POST("/admin/registration/:userId/resend", v2.AdminRegistrationResend)

			// Django group (role) management — writes rows in Django's own auth tables
			admin.GET("/admin/groups", v2.AdminGroupList)
			admin.GET("/admin/group/:id", v2.AdminGroupDetail)
			admin.POST("/admin/groups", v2.AdminGroupCreate)
			admin.PATCH("/admin/group/:id", v2.AdminGroupUpdate)
			admin.DELETE("/admin/group/:id", v2.AdminGroupDelete)
			admin.GET("/admin/group/:id/members", v2.AdminGroupMembers)
			admin.POST("/admin/group/:id/members", v2.AdminGroupAddMember)
			admin.DELETE("/admin/group/:id/members/:userId", v2.AdminGroupRemoveMember)
			admin.GET("/admin/permissions", v2.AdminPermissionList)
			admin.POST("/admin/user/:id/groups", v2.AdminUserAddGroup)
			admin.DELETE("/admin/user/:id/groups/:groupId", v2.AdminUserRemoveGroup)
			admin.PUT("/admin/user/:id/groups", v2.AdminUserReplaceGroups)
			admin.PUT("/admin/user/:id/permissions", v2.AdminUserReplacePermissions)
			admin.POST("/admin/user/:id/security", v2.AdminUserSecurity)

			// Contests
			admin.GET("/admin/contests", v2.AdminContestList)
			admin.GET("/admin/contest/:key", v2.AdminContestDetail)
			admin.POST("/admin/contests", auth.RequirePerm("judge.add_contest"), v2.AdminContestCreate)
			admin.PATCH("/admin/contest/:key", auth.RequireContestEdit(), v2.AdminContestUpdate)
			admin.DELETE("/admin/contest/:key", auth.RequireContestEdit(), v2.AdminContestDelete)
			admin.POST("/admin/contest/:key/lock", auth.RequireContestEdit(), v2.AdminContestLock)
			admin.POST("/admin/contest/:key/clone", auth.RequirePerm("judge.clone_contest"), v2.AdminContestClone)
			// Contest Participation Disqualify
			admin.POST("/admin/contest/:key/participation/:id/disqualify", auth.RequireContestEdit(), v2.AdminContestParticipationDisqualify)
			admin.POST("/admin/contest/:key/participation/:id/undisqualify", auth.RequireContestEdit(), v2.AdminContestParticipationUndisqualify)

			// Contest Announcements
			admin.GET("/admin/contest/:key/announcements", auth.RequireContestEdit(), v2.AdminContestAnnouncementList)
			admin.POST("/admin/contest/:key/announcements", auth.RequireContestEdit(), v2.AdminContestAnnouncementCreate)
			admin.PATCH("/admin/contest/:key/announcements/:id", auth.RequireContestEdit(), v2.AdminContestAnnouncementUpdate)
			admin.DELETE("/admin/contest/:key/announcements/:id", auth.RequireContestEdit(), v2.AdminContestAnnouncementDelete)

			// Contest plagiarism check (compare50; was /moss)
			admin.GET("/admin/contest/:key/plagiarism", auth.RequireContestEdit(), v2.AdminContestMossList)
			admin.POST("/admin/contest/:key/plagiarism", auth.RequireContestEdit(), v2.AdminContestMossRun)
			admin.DELETE("/admin/contest/:key/plagiarism", auth.RequireContestEdit(), v2.AdminContestMossDelete)
			admin.GET("/admin/contest/:key/plagiarism/report/*path", auth.RequireContestEdit(), v2.AdminContestMossReport)

			// Contest Tags
			admin.GET("/admin/contest-tags", v2.AdminContestTagList)
			admin.GET("/admin/contest-tag/:id", v2.AdminContestTagDetail)
			admin.POST("/admin/contest-tags", v2.AdminContestTagCreate)
			admin.PATCH("/admin/contest-tag/:id", v2.AdminContestTagUpdate)
			admin.DELETE("/admin/contest-tag/:id", v2.AdminContestTagDelete)
			admin.POST("/admin/contest/:key/tags/:tagId", auth.RequireContestEdit(), v2.AdminContestAddTag)
			admin.DELETE("/admin/contest/:key/tags/:tagId", auth.RequireContestEdit(), v2.AdminContestRemoveTag)

			// Problems
			admin.GET("/admin/problems", v2.AdminProblemList)
			admin.GET("/admin/problem/:code", v2.AdminProblemDetail)
			admin.POST("/admin/problems", auth.RequirePerm("judge.add_problem"), v2.AdminProblemCreate)
			admin.PATCH("/admin/problem/:code", auth.RequireProblemEdit(), v2.AdminProblemUpdate)
			admin.DELETE("/admin/problem/:code", auth.RequireProblemEdit(), v2.AdminProblemDelete)
			// Problem Clone
			admin.POST("/admin/problem/:code/clone", auth.RequirePerm("judge.clone_problem"), v2.AdminProblemClone)
			// Problem Clarifications
			admin.POST("/admin/problem/:code/clarification", auth.RequireProblemEdit(), v2.ProblemClarificationCreate)
			admin.DELETE("/admin/problem/clarification/:id", v2.ProblemClarificationDelete)
			admin.PATCH("/admin/problem/clarification/:id", v2.ProblemClarificationUpdate)
			// Problem Translations
			admin.GET("/admin/problem/:code/translations", auth.RequireProblemEdit(), v2.ProblemTranslationList)
			admin.POST("/admin/problem/:code/translations", auth.RequireProblemEdit(), v2.ProblemTranslationCreate)
			admin.PATCH("/admin/problem/:code/translations/:id", auth.RequireProblemEdit(), v2.ProblemTranslationUpdate)
			admin.DELETE("/admin/problem/:code/translations/:id", auth.RequireProblemEdit(), v2.ProblemTranslationDelete)
			// Problem Data
			admin.GET("/admin/problem/:code/data", auth.RequireProblemEdit(), adminHandlers.AdminProblemData)
			admin.POST("/admin/problem/:code/data", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataUpload)
			admin.DELETE("/admin/problem/:code/data/testcase/:id", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataDeleteTestCase)
		// Problem PDF
		admin.POST("/admin/problem/:code/pdf", auth.RequireProblemEdit(), adminHandlers.AdminProblemPdfUpload)
		admin.POST("/admin/problem/:code/pdf/commit", auth.RequireProblemEdit(), v2.AdminProblemPdfCommit)
			admin.DELETE("/admin/problem/:code/pdf", auth.RequireProblemEdit(), adminHandlers.AdminProblemPdfDelete)
			// Problem Data - Reorder & File Operations
			admin.PATCH("/admin/problem/:code/data/reorder", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataReorder)
			admin.GET("/admin/problem/:code/data/files", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataFiles)
			admin.GET("/admin/problem/:code/data/file/*path", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataFileContent)
			admin.DELETE("/admin/problem/:code/data/file/*path", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataFileDelete)
			admin.GET("/admin/problem/:code/data/testcase/:id/content", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataTestCaseContent)
			admin.PATCH("/admin/problem/:code/data/testcase/:id", auth.RequireProblemEdit(), adminHandlers.AdminProblemDataTestCaseUpdate)
			// Problem Submission Diff Matrix
			admin.GET("/admin/problem/:code/diff", auth.RequireProblemEdit(), v2.AdminProblemSubmissionDiff)

			// Solutions
			admin.GET("/admin/solutions", v2.AdminSolutionList)
			admin.GET("/admin/solution/:id", v2.AdminSolutionDetail)
			admin.POST("/admin/solutions", v2.AdminSolutionCreate)
			admin.PATCH("/admin/solution/:id", v2.AdminSolutionUpdate)
			admin.DELETE("/admin/solution/:id", v2.AdminSolutionDelete)

			// Judges
			admin.GET("/admin/judges", v2.AdminJudgeList)
			admin.GET("/admin/judge/:id", v2.AdminJudgeDetail)
			admin.POST("/admin/judge/:id/block", v2.AdminJudgeBlock)
			admin.POST("/admin/judge/:id/unblock", v2.AdminJudgeUnblock)
			admin.POST("/admin/judge/:id/enable", v2.AdminJudgeEnable)
			admin.POST("/admin/judge/:id/disable", v2.AdminJudgeDisable)
			admin.PATCH("/admin/judge/:id", v2.AdminJudgeUpdate)
			admin.POST("/admin/judges", auth.RequirePerm("judge.add_judge"), v2.AdminJudgeCreate)
			admin.DELETE("/admin/judge/:id", auth.RequirePerm("judge.delete_judge"), v2.AdminJudgeDelete)

			// Organizations
			admin.GET("/admin/organizations", v2.AdminOrganizationList)
			admin.GET("/admin/classes", v2.AdminClassList)
			admin.GET("/admin/class/:id", v2.AdminClassDetail)
			admin.POST("/admin/classes", auth.RequirePerm("judge.add_class"), v2.AdminClassCreate)
			admin.PATCH("/admin/class/:id", auth.RequirePerm("judge.change_class"), v2.AdminClassUpdate)
			admin.DELETE("/admin/class/:id", auth.RequirePerm("judge.delete_class"), v2.AdminClassDelete)
			// Read-only v1 screens (ProblemPointsVoteAdmin / OrganizationRequestAdmin)
			admin.GET("/admin/points-votes", v2.AdminPointsVoteList)
			admin.GET("/admin/organization-requests", v2.AdminOrganizationRequestList)
			// Comment locks (v1 judge_commentlock)
			admin.GET("/admin/comment-locks", v2.AdminCommentLockList)
			admin.POST("/admin/comment-locks", v2.AdminCommentLockCreate)
			admin.PATCH("/admin/comment-lock/:id", v2.AdminCommentLockUpdate)
			admin.DELETE("/admin/comment-lock/:id", v2.AdminCommentLockDelete)
			admin.GET("/admin/organization/:id", v2.AdminOrganizationDetail)
			admin.POST("/admin/organizations", auth.RequirePerm("judge.add_organization"), v2.AdminOrganizationCreate)
			admin.PATCH("/admin/organization/:id", auth.RequireOrgEdit(), v2.AdminOrganizationUpdate)
			admin.POST("/admin/organization/:id/recalculate", auth.RequireOrgEdit(), v2.AdminOrganizationRecalculate)
			admin.DELETE("/admin/organization/:id", auth.RequirePerm("judge.edit_all_organization"), v2.AdminOrganizationDelete)

			// Submissions
			admin.GET("/admin/submissions", v2.AdminSubmissionList)
			admin.GET("/admin/submission/:id", v2.AdminSubmissionDetail)
			admin.PATCH("/admin/submission/:id", auth.RequirePerm("judge.change_submission"), v2.AdminSubmissionUpdate)
			admin.POST("/admin/submission/:id/rejudge", v2.AdminSubmissionRejudge)
			admin.POST("/admin/submission/:id/abort", v2.AdminSubmissionAbort)
			admin.DELETE("/admin/submission/:id", auth.RequirePerm("judge.delete_submission"), v2.AdminSubmissionDelete)
			admin.POST("/admin/submissions/bulk-delete", auth.RequirePerm("judge.delete_submission"), v2.AdminSubmissionBulkDelete)
			admin.POST("/admin/submissions/batch-rejudge", v2.AdminSubmissionBatchRejudge)
			admin.POST("/admin/submission/:id/rescore", v2.AdminSubmissionRescore)
			admin.POST("/admin/submissions/batch-rescore", v2.AdminSubmissionBatchRescore)
			admin.POST("/admin/problem/:code/rescore-all", v2.AdminProblemRescoreAll)
			admin.POST("/admin/recalculate-points", v2.AdminRecalculatePoints)

			admin.POST("/admin/comment/:id/hide", v2.CommentHide)

			// Comments Admin
			admin.GET("/admin/comments", v2.AdminCommentList)
			admin.POST("/admin/comments/bulk-hide", v2.AdminCommentBulkHide)
			admin.PATCH("/admin/comment/:id", v2.AdminCommentUpdate)
			admin.DELETE("/admin/comment/:id", v2.AdminCommentDelete)

			// Languages Admin
			admin.GET("/admin/languages", v2.AdminLanguageList)
			admin.GET("/admin/language/:id", v2.AdminLanguageDetail)
			admin.POST("/admin/languages", v2.AdminLanguageCreate)
			admin.PATCH("/admin/language/:id", v2.AdminLanguageUpdate)
			admin.DELETE("/admin/language/:id", v2.AdminLanguageDelete)

			// Language Limits Admin
			admin.GET("/admin/language-limits", v2.AdminLanguageLimitList)
			admin.GET("/admin/language-limit/:id", v2.AdminLanguageLimitDetail)
			admin.POST("/admin/language-limits", v2.AdminLanguageLimitCreate)
			admin.PATCH("/admin/language-limit/:id", v2.AdminLanguageLimitUpdate)
			admin.DELETE("/admin/language-limit/:id", v2.AdminLanguageLimitDelete)

			// Blog Posts Admin
			admin.GET("/admin/blog-posts", v2.AdminBlogPostList)
			admin.GET("/admin/blog-post/:id", v2.AdminBlogPostDetail)
			admin.POST("/admin/blog-posts", v2.AdminBlogPostCreate)
			admin.PATCH("/admin/blog-post/:id", v2.AdminBlogPostUpdate)
			admin.DELETE("/admin/blog-post/:id", v2.AdminBlogPostDelete)

			// Shared markdown image upload (blog posts, license text)
			admin.POST("/admin/markdown/image", v2.AdminMarkdownImageUpload)
			// Direct-to-R2 uploads: sign, browser PUTs to R2, then commit.
			admin.POST("/admin/uploads/sign", v2.AdminUploadSign)
			admin.POST("/admin/markdown/image/commit", v2.AdminMarkdownImageCommit)

			// Licenses Admin
			admin.GET("/admin/licenses", v2.AdminLicenseList)
			admin.GET("/admin/license/:id", v2.AdminLicenseDetail)
			admin.POST("/admin/licenses", v2.AdminLicenseCreate)
			admin.PATCH("/admin/license/:id", v2.AdminLicenseUpdate)
			admin.DELETE("/admin/license/:id", v2.AdminLicenseDelete)

			// Problem Taxonomy Admin
			admin.GET("/admin/problem-groups", v2.AdminProblemGroupList)
			admin.GET("/admin/problem-group/:id", v2.AdminProblemGroupDetail)
			admin.POST("/admin/problem-groups", v2.AdminProblemGroupCreate)
			admin.PATCH("/admin/problem-group/:id", v2.AdminProblemGroupUpdate)
			admin.DELETE("/admin/problem-group/:id", v2.AdminProblemGroupDelete)
			admin.GET("/admin/problem-types", v2.AdminProblemTypeList)
			admin.GET("/admin/problem-type/:id", v2.AdminProblemTypeDetail)
			admin.POST("/admin/problem-types", v2.AdminProblemTypeCreate)
			admin.PATCH("/admin/problem-type/:id", v2.AdminProblemTypeUpdate)
			admin.DELETE("/admin/problem-type/:id", v2.AdminProblemTypeDelete)

			// Tickets (Admin)
			admin.GET("/admin/tickets", v2.AdminTicketList)
			admin.GET("/admin/ticket/:id", v2.AdminTicketDetail)
			admin.POST("/admin/ticket/:id/assign", v2.AdminTicketAssign)
			admin.POST("/admin/ticket/:id/toggle", v2.AdminTicketToggleOpen)
			admin.POST("/admin/ticket/:id/set-contributive", v2.AdminTicketSetContributive)
			admin.PATCH("/admin/ticket/:id/notes", v2.AdminTicketUpdateNotes)
			admin.PATCH("/admin/ticket/:id", auth.RequirePerm("judge.change_ticket"), v2.AdminTicketUpdate)
			admin.PATCH("/admin/ticket/:id/message/:messageId", auth.RequirePerm("judge.change_ticket"), v2.AdminTicketMessageUpdate)
			admin.DELETE("/admin/ticket/:id/message/:messageId", auth.RequirePerm("judge.change_ticket"), v2.AdminTicketMessageDelete)
			admin.DELETE("/admin/ticket/:id", auth.RequirePerm("judge.delete_ticket"), v2.AdminTicketDelete)

		// Navigation Bars (v1 admin gates the whole model with
		// judge.change_navigationbar; HasPerm lets superusers pass)
		admin.GET("/admin/navigation-bars", v2.AdminNavigationBarList)
		admin.GET("/admin/navigation-bar/:id", v2.AdminNavigationBarDetail)
		admin.POST("/admin/navigation-bars", auth.RequirePerm("judge.change_navigationbar"), v2.AdminNavigationBarCreate)
		admin.PATCH("/admin/navigation-bar/:id", auth.RequirePerm("judge.change_navigationbar"), v2.AdminNavigationBarUpdate)
		admin.DELETE("/admin/navigation-bar/:id", auth.RequirePerm("judge.change_navigationbar"), v2.AdminNavigationBarDelete)

		// Flat pages + redirects (Django contrib tables, single site). Reads are
		// staff-level; writes use the same codenames v1's Django admin checks.
		admin.GET("/admin/flat-pages", v2.AdminFlatPageList)
		admin.GET("/admin/flat-page/:id", v2.AdminFlatPageDetail)
		admin.POST("/admin/flat-pages", auth.RequirePerm("flatpages.change_flatpage"), v2.AdminFlatPageCreate)
		admin.PATCH("/admin/flat-page/:id", auth.RequirePerm("flatpages.change_flatpage"), v2.AdminFlatPageUpdate)
		admin.DELETE("/admin/flat-page/:id", auth.RequirePerm("flatpages.change_flatpage"), v2.AdminFlatPageDelete)
		admin.GET("/admin/redirects", v2.AdminRedirectList)
		admin.GET("/admin/redirect/:id", v2.AdminRedirectDetail)
		admin.POST("/admin/redirects", auth.RequirePerm("redirects.change_redirect"), v2.AdminRedirectCreate)
		admin.PATCH("/admin/redirect/:id", auth.RequirePerm("redirects.change_redirect"), v2.AdminRedirectUpdate)
		admin.DELETE("/admin/redirect/:id", auth.RequirePerm("redirects.change_redirect"), v2.AdminRedirectDelete)

		// Misc Configs
		admin.GET("/admin/misc-configs", v2.AdminMiscConfigList)
		admin.GET("/admin/misc-config/:id", v2.AdminMiscConfigDetail)
		admin.POST("/admin/misc-configs", v2.AdminMiscConfigCreate)
		admin.PATCH("/admin/misc-config/:id", v2.AdminMiscConfigUpdate)
		admin.DELETE("/admin/misc-config/:id", v2.AdminMiscConfigDelete)
		admin.PUT("/admin/name-styles", auth.RequirePerm("judge.change_miscconfig"), v2.AdminNameStylesUpdate)

		// Audit Logs
		admin.GET("/audit-logs", v2.AdminAuditLogList)
		admin.GET("/audit-log/:id", v2.AdminAuditLogDetail)
		}

		apiv2.GET("/problems", v2.ProblemList)
		apiv2.GET("/problems/random", v2.RandomProblem)
		// Suggestions awaiting review (judge.suggest_new_problem), and the
		// public taxonomy the problem-list filters are built from.
		apiv2.GET("/problems/suggest", v2.SuggestProblemList)
		apiv2.GET("/license/:key", v2.PublicLicenseDetail)
		// Public navbar (v1 judge_navigationbar) that drives the site header.
		apiv2.GET("/navigation", v2.PublicNavigation)
		apiv2.GET("/problem-groups", v2.ProblemGroupList)
		apiv2.GET("/problem-types", v2.ProblemTypeList)
		apiv2.GET("/licenses", v2.LicenseList)
		apiv2.GET("/problem/:code", v2.ProblemDetail)
		apiv2.GET("/problem/:code/stats", v2.ProblemStats)
		apiv2.GET("/problem/:code/solution", v2.ProblemSolution)
		apiv2.GET("/problem/:code/solution/exists", v2.ProblemSolutionExists)
		// Problem PDF Statement
		apiv2.GET("/problem/:code/pdf", v2.ProblemStatementPDF)
		// Problem Feeds
		apiv2.GET("/problems/feed/rss", v2.ProblemFeedRSS)
		apiv2.GET("/problems/feed/atom", v2.ProblemFeedAtom)
		// Problem Language Limits
		apiv2.GET("/problem/:code/language-limits", v2.ProblemLanguageLimits)
		// Problem Clarifications (public read)
		apiv2.GET("/problem/:code/clarifications", v2.ProblemClarificationList)

		apiv2.GET("/events", events.ServeWS)

		apiv2.GET("/contests", contestHandlers.ContestList)
		apiv2.GET("/contest-tags", v2.ContestTagList)
		apiv2.GET("/contests/calendar", v2.ContestCalendar)
		apiv2.GET("/contest/:key", contestHandlers.ContestDetail)
		apiv2.GET("/contest/:key/ranking", v2.ContestRanking)
		apiv2.GET("/contest/:key/ranking/pdf", v2.ContestRankingPDF)
		apiv2.GET("/contest/:key/rank/:problem", v2.ContestProblemRank)
		apiv2.GET("/contest/:key/stats", contestHandlers.ContestStats)
		apiv2.GET("/contest/:key/stats/public", contestHandlers.ContestPublicStats)
		apiv2.GET("/contest/:key/participations", contestHandlers.ParticipationList)

		apiv2.GET("/submissions", v2.SubmissionList)
		apiv2.GET("/submission/:id", v2.SubmissionDetail)
		apiv2.GET("/submissions/:id1/diff/:id2", v2.SubmissionDiff)

		apiv2.GET("/users", v2.UserList)
		apiv2.GET("/user/:user", v2.UserDetail)
		apiv2.GET("/user/:user/solved", v2.UserSolvedProblems)
		apiv2.GET("/user/:user/rating", v2.UserRatingHistory)
		apiv2.GET("/user/:user/rating-detail", v2.UserRatingDetail)
		apiv2.GET("/user/:user/analytics", v2.UserAnalytics)
		apiv2.GET("/user/:user/pp-breakdown", v2.UserPPBreakdown)

		// Ratings leaderboard
		apiv2.GET("/ratings/leaderboard", v2.RatingLeaderboard)

		apiv2.GET("/organizations", v2.OrganizationList)
		apiv2.GET("/organization/:id", v2.OrganizationDetail)
		apiv2.GET("/organization/:id/problems", v2.OrganizationProblemList)
		apiv2.GET("/organization/:id/contests", v2.OrganizationContestList)
		apiv2.GET("/organization/:id/submissions", v2.OrganizationSubmissionList)
		apiv2.GET("/languages", v2.LanguageList)
		apiv2.GET("/judges", v2.JudgeList)
		// v1 /runtimes + /runtimes/matrix (public; non-staff see only runtimes
		// reported by online judges)
		apiv2.GET("/runtimes", v2.RuntimeList)
		apiv2.GET("/runtimes/matrix", v2.RuntimeMatrix)

		// Stats
		apiv2.GET("/stats/languages", v2.LanguageStats)
		apiv2.GET("/stats/submissions/daily", v2.DailySubmissionStats)
		apiv2.GET("/stats/overall", v2.OverallStats)
		apiv2.GET("/stats/submissions", v2.SubmissionStats)
		apiv2.GET("/stats/problems", v2.ProblemStatsList)
		apiv2.GET("/stats/judges", v2.JudgeStats)
		apiv2.GET("/stats/users", v2.UserStats)
		apiv2.GET("/stats/contests", v2.ContestStatsList)

		// Comments
		apiv2.GET("/comments", v2.CommentList)
		apiv2.GET("/comments/feed/rss", v2.CommentFeedRSS)
		apiv2.GET("/comments/feed/atom", v2.CommentFeedAtom)

		// Contest clarifications (public read)
		apiv2.GET("/contest/:key/clarifications", contestHandlers.ContestClarificationList)

		// Blogs
		apiv2.GET("/blogs", v2.BlogList)
		apiv2.GET("/name-styles", v2.NameStyles)
		apiv2.GET("/blog/:id", v2.BlogDetail)
		apiv2.GET("/blogs/feed/rss", v2.BlogFeedRSS)
		apiv2.GET("/blogs/feed/atom", v2.BlogFeedAtom)

		// Uploaded markdown images (blog posts, license text)
		apiv2.GET("/media/:name", v2.MarkdownMediaServe)

		// Uploaded submission files for file-only languages. The judge agent
		// fetches SiteFullURL + SubmissionSource.source, i.e. this route.
		apiv2.GET("/submission-file/:name", v2.SubmissionFileServe)

		// Flat pages + redirects (v1 contrib tables). /page feeds the web
		// catch-all route, /redirects-map feeds the web proxy's path redirects.
		apiv2.GET("/page", v2.FlatPageLookup)
		apiv2.GET("/redirects-map", v2.RedirectMap)

		// Sitemap (public)
		apiv2.GET("/sitemap.xml", v2.SitemapIndex)
		apiv2.GET("/sitemap-static.xml", v2.SitemapStatic)
		apiv2.GET("/sitemap-problems.xml", v2.SitemapProblems)
		apiv2.GET("/sitemap-problem-groups.xml", v2.SitemapProblemGroups)
		apiv2.GET("/sitemap-contests.xml", v2.SitemapContests)
		apiv2.GET("/sitemap-users.xml", v2.SitemapUsers)
		apiv2.GET("/sitemap-blog-posts.xml", v2.SitemapBlogPosts)
		apiv2.GET("/sitemap-organizations.xml", v2.SitemapOrganizations)

		// Protected endpoints with CSRF protection
		protected := apiv2.Group("")
		protected.Use(auth.RequiredMiddleware())
		protected.Use(csrf.Middleware(csrf.DefaultConfig()))
		{
			protected.POST("/auth/revoke-all-sessions", authHandlers.RevokeAllSessions)
			// Logged-in password change (the settings page). Distinct from the
			// public /auth/password/reset* pair, which is the forgot-password
			// flow for users who can't sign in.
			protected.POST("/auth/password/change", authHandlers.PasswordChange)
			// Self-service email change (settings Account tab): start mails a
			// one-time link, confirm redeems it for the logged-in account.
			protected.POST("/auth/email/change", v2.EmailChangeStart)
			protected.POST("/auth/email/change/confirm", v2.EmailChangeConfirm)
			// Hidden problem suggestions (v1 ProblemSuggest form).
			protected.POST("/problems/suggest", auth.RequirePerm("judge.suggest_new_problem"), v2.SuggestProblemCreate)

			// TOTP/2FA management (requires an authenticated session).
			// Previously mis-registered in the public group before the auth
			// middleware, so these ran with user_id=0 — setup returned "user
			// not found" and 2FA could never be enabled.
			protected.GET("/auth/totp/status", v2.TotpStatus)
			protected.POST("/auth/totp/setup", v2.TotpSetup)
			protected.POST("/auth/totp/confirm", v2.TotpConfirm)
			protected.POST("/auth/totp/disable", v2.TotpDisable)
			protected.POST("/auth/totp/backup-codes", v2.TotpBackupCodesGenerate)
			// WebAuthn Credential Management
			protected.GET("/auth/webauthn/credentials", v2.WebAuthnCredentialsList)
			protected.PATCH("/auth/webauthn/credentials/:id", v2.WebAuthnCredentialUpdate)
			protected.DELETE("/auth/webauthn/credentials/:id", v2.WebAuthnCredentialDelete)
			protected.POST("/problem/:code/submit", v2.Submit)
			protected.GET("/user/me", v2.CurrentUser)
			protected.PATCH("/user/me", v2.UpdateProfile)
			protected.POST("/user/avatar/sign", v2.AvatarUploadSign)
			protected.PUT("/user/avatar", v2.AvatarSet)
			protected.DELETE("/user/avatar", v2.AvatarDelete)
			protected.POST("/admin/impersonate/:userId", auth.AdminRequiredMiddleware(), middleware.Audit(), v2.AdminImpersonateStart)
			protected.POST("/admin/impersonate/exit", v2.AdminImpersonateExit)

			// API Token Management
			protected.GET("/user/api-token", v2.GetAPIToken)
			protected.POST("/user/api-token", v2.GenerateAPIToken)
			protected.DELETE("/user/api-token", v2.RevokeAPIToken)

		// User Data Export
		protected.POST("/user/export/request", v2.UserExportRequest)
		protected.GET("/user/export/status", v2.UserExportStatus)
		protected.GET("/user/export/download/:export_id", v2.UserExportDownload)

			protected.POST("/contest/:key/join", v2.ContestJoin)
			protected.POST("/contest/:key/leave", v2.ContestLeave)
			protected.POST("/submission/:id/abort", v2.SubmissionAbort)
			// v1 /problem/<p>/resubmit/<submission>: prefill from an earlier
			// submission (owner, or judge.resubmit_other).
			protected.GET("/submission/:id/resubmit", v2.SubmissionResubmitSource)
			protected.GET("/user/contests", contestHandlers.UserParticipationList)

			// Tickets
			protected.GET("/tickets", v2.TicketList)
			protected.POST("/tickets", v2.TicketCreate)
			protected.GET("/ticket/:id", v2.TicketDetail)
			protected.POST("/ticket/:id/message", v2.TicketReply)

			// Comments
			protected.POST("/comments", v2.CommentCreate)
			protected.POST("/comment/:id/vote", v2.CommentVote)
			protected.PATCH("/comment/:id", v2.CommentUpdate)
			protected.DELETE("/comment/:id", v2.CommentDelete)
			protected.GET("/comment/:id/revisions", v2.CommentRevisionList)

			// Contest clarifications (protected write)
			protected.POST("/contest/:key/clarifications", contestHandlers.ContestClarificationCreate)
			protected.POST("/contest/:key/clarification/:id/answer", contestHandlers.ContestClarificationAnswer)
			protected.DELETE("/contest/:key/clarification/:id", contestHandlers.ContestClarificationDelete)

			// Blogs
			protected.POST("/blog/:id/vote", v2.BlogVoteHandler)
			// Public blog authoring (v1 /post/new, /post/<id>/edit).
			protected.POST("/blogs", v2.PublicBlogPostCreate)
			protected.PATCH("/blog/:id", v2.PublicBlogPostUpdate)

			// Organizations
			protected.POST("/organizations", v2.PublicOrganizationCreate)
			protected.PATCH("/organization/:id", v2.UpdateOrganization)
			protected.POST("/organization/:id/join", v2.JoinOrganization)
			protected.POST("/organization/:id/leave", v2.LeaveOrganization)
			protected.POST("/organization/:id/request", v2.RequestJoinOrganization)
			protected.GET("/organization/:id/requests", v2.OrganizationRequestList)
			protected.POST("/organization/request/:rid/handle", v2.HandleOrganizationRequest)
			protected.POST("/organization/:id/kick", v2.KickUser)
			// Org-admin scoped content creation (v1 *CreateOrganization views):
			// create permission + org-admin membership, organization forced.
			protected.POST("/organization/:id/problems", v2.OrganizationProblemCreate)
			protected.POST("/organization/:id/contests", v2.OrganizationContestCreate)
			protected.POST("/organization/:id/blog-posts", v2.OrganizationBlogPostCreate)

			// Notifications
			protected.GET("/notifications", v2.NotificationList)
			protected.GET("/notifications/unread-count", v2.NotificationUnreadCount)
			protected.POST("/notifications/:id/read", v2.NotificationMarkRead)
			protected.POST("/notifications/read-all", v2.NotificationMarkAllRead)
			protected.DELETE("/notifications/:id", v2.NotificationDelete)
			protected.GET("/notifications/preferences", v2.NotificationPreferencesGet)
			protected.PATCH("/notifications/preferences", v2.NotificationPreferencesUpdate)
		}
	}

	return r
}
