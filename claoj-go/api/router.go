// Package api wires the Gin HTTP engine and attaches all route groups.
package api

import (
	v2 "github.com/CLAOJ/claoj-go/api/v2"
	"github.com/CLAOJ/claoj-go/auth"
	"github.com/CLAOJ/claoj-go/cache"
	"github.com/CLAOJ/claoj-go/config"
	"github.com/CLAOJ/claoj-go/csrf"
	"github.com/CLAOJ/claoj-go/events"
	"github.com/CLAOJ/claoj-go/ratelimit"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// NewRouter creates and returns the fully configured Gin engine.
func NewRouter() *gin.Engine {
	r := gin.Default()

	// CORS middleware - only allow configured origins
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{config.C.App.SiteFullURL}
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Authorization", "Content-Type", "X-Requested-With", "X-CSRFToken"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	r.Use(cors.New(corsConfig))

	// Health-check
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	// Rate limiting middleware for API v2
	// Use Redis-backed rate limiting in production, in-memory for development
	var ratelimitRepo ratelimit.Repository
	if cache.Client != nil {
		ratelimitRepo = ratelimit.NewRedisRepository(cache.Client)
	} else {
		ratelimitRepo = ratelimit.NewInMemoryRepository(60) // 60-second window
	}
	apiv2 := r.Group("/api/v2")
	apiv2.Use(ratelimit.Middleware(ratelimitRepo))
	{
		// Public Auth
		apiv2.POST("/auth/login", v2.Login)
		apiv2.POST("/auth/logout", v2.Logout)
		apiv2.POST("/auth/refresh", v2.Refresh)
		apiv2.POST("/auth/register", v2.Register)
		apiv2.POST("/auth/password/reset", v2.PasswordResetRequest)
		apiv2.POST("/auth/password/reset/confirm", v2.PasswordResetConfirm)
		// Email verification
		apiv2.POST("/auth/verify-email", v2.VerifyEmail)
		apiv2.POST("/auth/resend-verification", v2.ResendVerification)
		// TOTP/2FA
		apiv2.GET("/auth/totp/status", v2.TotpStatus)
		apiv2.POST("/auth/totp/setup", v2.TotpSetup)
		apiv2.POST("/auth/totp/confirm", v2.TotpConfirm)
		apiv2.POST("/auth/totp/disable", v2.TotpDisable)
		apiv2.POST("/auth/totp/verify", v2.TotpVerify)
		apiv2.POST("/auth/totp/backup-codes", v2.TotpBackupCodesGenerate)
		apiv2.POST("/auth/totp/verify-backup", v2.TotpBackupVerify)
		// WebAuthn/2FA
		apiv2.GET("/auth/webauthn/status", v2.WebAuthnStatus)
		apiv2.POST("/auth/webauthn/register/begin", v2.WebAuthnBeginRegistration)
		apiv2.POST("/auth/webauthn/register/finish", v2.WebAuthnFinishRegistration)
		apiv2.POST("/auth/webauthn/login/begin", v2.WebAuthnBeginLogin)
		apiv2.POST("/auth/webauthn/login/finish", v2.WebAuthnFinishLogin)
		// WebAuthn Credentials (Protected)
		// OAuth
		apiv2.GET("/auth/oauth/:provider", v2.OAuthStart)
		apiv2.POST("/auth/oauth/:provider/callback", v2.OAuthCallback)

		// Optional Auth (for user-specific logic on public pages like problem solved status)
		// We'll apply this globally to the main apiv2 group since it doesn't block unauthenticated requests.
		apiv2.Use(auth.OptionalMiddleware())

		// =======================================================
		// ADMIN ROUTES (Admin Required)
		// =======================================================
		admin := apiv2.Group("")
		admin.Use(auth.AdminRequiredMiddleware())
		{
			// Users
			admin.GET("/admin/users", v2.AdminUserList)
			admin.GET("/admin/user/:id", v2.AdminUserDetail)
			admin.PATCH("/admin/user/:id", v2.AdminUserUpdate)
			admin.DELETE("/admin/user/:id", v2.AdminUserDelete)
			admin.POST("/admin/user/:id/ban", v2.AdminUserBan)
			admin.POST("/admin/user/:id/unban", v2.AdminUserUnban)

			// Contests
			admin.GET("/admin/contests", v2.AdminContestList)
			admin.GET("/admin/contest/:key", v2.AdminContestDetail)
			admin.POST("/admin/contests", v2.AdminContestCreate)
			admin.PATCH("/admin/contest/:key", v2.AdminContestUpdate)
			admin.DELETE("/admin/contest/:key", v2.AdminContestDelete)
			admin.POST("/admin/contest/:key/lock", v2.AdminContestLock)
			admin.POST("/admin/contest/:key/clone", v2.AdminContestClone)
			// Contest Participation Disqualify
			admin.POST("/admin/contest/:key/participation/:id/disqualify", v2.AdminContestParticipationDisqualify)
			admin.POST("/admin/contest/:key/participation/:id/undisqualify", v2.AdminContestParticipationUndisqualify)

			// Contest Tags
			admin.GET("/admin/contest-tags", v2.AdminContestTagList)
			admin.GET("/admin/contest-tag/:id", v2.AdminContestTagDetail)
			admin.POST("/admin/contest-tags", v2.AdminContestTagCreate)
			admin.PATCH("/admin/contest-tag/:id", v2.AdminContestTagUpdate)
			admin.DELETE("/admin/contest-tag/:id", v2.AdminContestTagDelete)
			admin.POST("/admin/contest/:key/tags/:tagId", v2.AdminContestAddTag)
			admin.DELETE("/admin/contest/:key/tags/:tagId", v2.AdminContestRemoveTag)

			// Problems
			admin.GET("/admin/problems", v2.AdminProblemList)
			admin.GET("/admin/problem/:code", v2.AdminProblemDetail)
			admin.POST("/admin/problems", v2.AdminProblemCreate)
			admin.PATCH("/admin/problem/:code", v2.AdminProblemUpdate)
			admin.DELETE("/admin/problem/:code", v2.AdminProblemDelete)
			// Problem Clone
			admin.POST("/admin/problem/:code/clone", v2.AdminProblemClone)
			// Problem Clarifications
			admin.POST("/admin/problem/:code/clarification", v2.ProblemClarificationCreate)
			admin.DELETE("/admin/problem/clarification/:id", v2.ProblemClarificationDelete)
			// Problem Data
			admin.GET("/admin/problem/:code/data", v2.AdminProblemData)
			admin.POST("/admin/problem/:code/data", v2.AdminProblemDataUpload)
			admin.DELETE("/admin/problem/:code/data/testcase/:id", v2.AdminProblemDataDeleteTestCase)
		// Problem PDF
		admin.POST("/admin/problem/:code/pdf", v2.AdminProblemPdfUpload)
		admin.DELETE("/admin/problem/:code/pdf", v2.AdminProblemPdfDelete)
			// Problem Data - Reorder & File Operations
			admin.PATCH("/admin/problem/:code/data/reorder", v2.AdminProblemDataReorder)
			admin.GET("/admin/problem/:code/data/files", v2.AdminProblemDataFiles)
			admin.GET("/admin/problem/:code/data/file/*path", v2.AdminProblemDataFileContent)
			admin.DELETE("/admin/problem/:code/data/file/*path", v2.AdminProblemDataFileDelete)
			admin.GET("/admin/problem/:code/data/testcase/:id/content", v2.AdminProblemDataTestCaseContent)
			admin.PATCH("/admin/problem/:code/data/testcase/:id", v2.AdminProblemDataTestCaseUpdate)

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

			// Organizations
			admin.GET("/admin/organizations", v2.AdminOrganizationList)
			admin.POST("/admin/organizations", v2.AdminOrganizationCreate)
			admin.PATCH("/admin/organization/:id", v2.AdminOrganizationUpdate)
			admin.DELETE("/admin/organization/:id", v2.AdminOrganizationDelete)

			// Submissions
			admin.GET("/admin/submissions", v2.AdminSubmissionList)
			admin.POST("/admin/submission/:id/rejudge", v2.AdminSubmissionRejudge)
			admin.POST("/admin/submission/:id/abort", v2.AdminSubmissionAbort)
			admin.POST("/admin/submissions/batch-rejudge", v2.AdminSubmissionBatchRejudge)
			admin.POST("/admin/submission/:id/moss", v2.AdminSubmissionMossAnalysis)
			admin.GET("/admin/submission/:id/moss", v2.AdminSubmissionMossResults)
			admin.POST("/admin/submission/:id/rescore", v2.AdminSubmissionRescore)
			admin.POST("/admin/submissions/batch-rescore", v2.AdminSubmissionBatchRescore)
			admin.POST("/admin/problem/:code/rescore-all", v2.AdminProblemRescoreAll)

			// Roles & Permissions
			admin.GET("/admin/roles", v2.AdminRoleList)
			admin.GET("/admin/role/:id", v2.AdminRoleDetail)
			admin.POST("/admin/roles", v2.AdminRoleCreate)
			admin.PATCH("/admin/role/:id", v2.AdminRoleUpdate)
			admin.DELETE("/admin/role/:id", v2.AdminRoleDelete)
			admin.GET("/admin/permissions", v2.AdminPermissionList)
			admin.POST("/admin/profile/:id/roles", v2.AdminProfileAssignRole)
			admin.DELETE("/admin/profile/:id/roles/:roleId", v2.AdminProfileRemoveRole)
			admin.POST("/admin/comment/:id/hide", v2.CommentHide)

			// Comments Admin
			admin.GET("/admin/comments", v2.AdminCommentList)
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

			// Problem Suggestions (Admin)
			admin.GET("/admin/problem-suggestions", v2.AdminProblemSuggestionList)
			admin.GET("/admin/problem-suggestion/:id", v2.AdminProblemSuggestionDetail)
			admin.POST("/admin/problem-suggestion/:id/approve", v2.AdminProblemSuggestionApprove)
			admin.POST("/admin/problem-suggestion/:id/reject", v2.AdminProblemSuggestionReject)
			admin.DELETE("/admin/problem-suggestion/:id", v2.AdminProblemSuggestionDelete)

		// Navigation Bars
		admin.GET("/admin/navigation-bars", v2.AdminNavigationBarList)
		admin.GET("/admin/navigation-bar/:id", v2.AdminNavigationBarDetail)
		admin.POST("/admin/navigation-bars", v2.AdminNavigationBarCreate)
		admin.PATCH("/admin/navigation-bar/:id", v2.AdminNavigationBarUpdate)
		admin.DELETE("/admin/navigation-bar/:id", v2.AdminNavigationBarDelete)

		// Misc Configs
		admin.GET("/admin/misc-configs", v2.AdminMiscConfigList)
		admin.GET("/admin/misc-config/:id", v2.AdminMiscConfigDetail)
		admin.POST("/admin/misc-configs", v2.AdminMiscConfigCreate)
		admin.PATCH("/admin/misc-config/:id", v2.AdminMiscConfigUpdate)
		admin.DELETE("/admin/misc-config/:id", v2.AdminMiscConfigDelete)
		}

		apiv2.GET("/problems", v2.ProblemList)
		apiv2.GET("/problems/random", v2.RandomProblem)
		apiv2.GET("/problem/:code", v2.ProblemDetail)
		apiv2.GET("/problem/:code/stats", v2.ProblemStats)
		apiv2.GET("/problem/:code/solution", v2.ProblemSolution)
		apiv2.GET("/problem/:code/solution/exists", v2.ProblemSolutionExists)
		// Problem PDF Statement
		apiv2.GET("/problem/:code/pdf", v2.ProblemStatementPDF)
		// Problem Language Limits
		apiv2.GET("/problem/:code/language-limits", v2.ProblemLanguageLimits)
		// Problem Clarifications (public read)
		apiv2.GET("/problem/:code/clarifications", v2.ProblemClarificationList)

		apiv2.GET("/events", events.ServeWS)

		apiv2.GET("/contests", v2.ContestList)
		apiv2.GET("/contests/calendar", v2.ContestCalendar)
		apiv2.GET("/contest/:key", v2.ContestDetail)
		apiv2.GET("/contest/:key/ranking", v2.ContestRanking)
		apiv2.GET("/contest/:key/ranking/pdf", v2.ContestRankingPDF)
		apiv2.GET("/contest/:key/stats", v2.ContestStats)
		apiv2.GET("/contest/:key/participations", v2.ParticipationList)

		apiv2.GET("/submissions", v2.SubmissionList)
		apiv2.GET("/submission/:id", v2.SubmissionDetail)

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
		apiv2.GET("/languages", v2.LanguageList)
		apiv2.GET("/judges", v2.JudgeList)

		// Stats
		apiv2.GET("/stats/languages", v2.LanguageStats)
		apiv2.GET("/stats/submissions/daily", v2.DailySubmissionStats)

		// Comments
		apiv2.GET("/comments", v2.CommentList)

		// Contest clarifications (public read)
		apiv2.GET("/contest/:key/clarifications", v2.ContestClarificationList)

		// Blogs
		apiv2.GET("/blogs", v2.BlogList)
		apiv2.GET("/blog/:id", v2.BlogDetail)
		apiv2.GET("/blogs/feed/rss", v2.BlogFeedRSS)
		apiv2.GET("/blogs/feed/atom", v2.BlogFeedAtom)

		// Protected endpoints with CSRF protection
		protected := apiv2.Group("")
		protected.Use(auth.RequiredMiddleware())
		protected.Use(csrf.Middleware(csrf.DefaultConfig()))
		{
			protected.POST("/auth/revoke-all-sessions", v2.RevokeAllSessions)
			// WebAuthn Credential Management
			protected.GET("/auth/webauthn/credentials", v2.WebAuthnCredentialsList)
			protected.PATCH("/auth/webauthn/credentials/:id", v2.WebAuthnCredentialUpdate)
			protected.DELETE("/auth/webauthn/credentials/:id", v2.WebAuthnCredentialDelete)
			protected.POST("/problem/:code/submit", v2.Submit)
			protected.GET("/user/me", v2.CurrentUser)
			protected.PATCH("/user/me", v2.UpdateProfile)

			// API Token Management
			protected.GET("/user/api-token", v2.GetAPIToken)
			protected.POST("/user/api-token", v2.GenerateAPIToken)
			protected.DELETE("/user/api-token", v2.RevokeAPIToken)

		// User Data Export
		protected.POST("/user/export/request", v2.UserExportRequest)
		protected.GET("/user/export/status", v2.UserExportStatus)
		protected.GET("/user/export/download/:export_id", v2.UserExportDownload)

			protected.POST("/contest/:key/join", v2.ContestJoin)
			protected.GET("/user/contests", v2.UserParticipationList)

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
			protected.POST("/contest/:key/clarifications", v2.ContestClarificationCreate)
			protected.POST("/contest/:key/clarification/:id/answer", v2.ContestClarificationAnswer)

			// Blogs
			protected.POST("/blog/:id/vote", v2.BlogVoteHandler)

			// Organizations
			protected.POST("/organization/:id/join", v2.JoinOrganization)
			protected.POST("/organization/:id/leave", v2.LeaveOrganization)
			protected.POST("/organization/:id/request", v2.RequestJoinOrganization)
			protected.GET("/organization/:id/requests", v2.OrganizationRequestList)
			protected.POST("/organization/request/:rid/handle", v2.HandleOrganizationRequest)
			protected.POST("/organization/:id/kick", v2.KickUser)

			// Problem Suggestions (Protected - users can submit and view their own)
			protected.POST("/problems/suggest", v2.SuggestProblem)
			protected.GET("/my-suggestions", v2.GetUserSuggestions)

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
