// Package api wires the Gin HTTP engine and attaches all route groups.
package api

import (
	v2 "github.com/CLAOJ/claoj-go/api/v2"
	"github.com/CLAOJ/claoj-go/auth"
	"github.com/CLAOJ/claoj-go/cache"
	"github.com/CLAOJ/claoj-go/config"
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
	corsConfig.AllowHeaders = []string{"Authorization", "Content-Type", "X-Requested-With"}
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

			// Problems
			admin.GET("/admin/problems", v2.AdminProblemList)
			admin.GET("/admin/problem/:code", v2.AdminProblemDetail)
			admin.POST("/admin/problems", v2.AdminProblemCreate)
			admin.PATCH("/admin/problem/:code", v2.AdminProblemUpdate)
			admin.DELETE("/admin/problem/:code", v2.AdminProblemDelete)
			// Problem Data
			admin.GET("/admin/problem/:code/data", v2.AdminProblemData)
			admin.POST("/admin/problem/:code/data", v2.AdminProblemDataUpload)
			admin.DELETE("/admin/problem/:code/data/testcase/:id", v2.AdminProblemDataDeleteTestCase)

			// Judges
			admin.GET("/admin/judges", v2.AdminJudgeList)
			admin.POST("/admin/judge/:id/block", v2.AdminJudgeBlock)
			admin.POST("/admin/judge/:id/unblock", v2.AdminJudgeUnblock)

			// Organizations
			admin.GET("/admin/organizations", v2.AdminOrganizationList)
			admin.POST("/admin/organizations", v2.AdminOrganizationCreate)
			admin.PATCH("/admin/organization/:id", v2.AdminOrganizationUpdate)
			admin.DELETE("/admin/organization/:id", v2.AdminOrganizationDelete)

			// Submissions
			admin.GET("/admin/submissions", v2.AdminSubmissionList)
			admin.POST("/admin/submission/:id/rejudge", v2.AdminSubmissionRejudge)
			admin.POST("/admin/submission/:id/moss", v2.AdminSubmissionMossAnalysis)
			admin.GET("/admin/submission/:id/moss", v2.AdminSubmissionMossResults)

			// Roles & Permissions
			admin.GET("/admin/roles", v2.AdminRoleList)
			admin.GET("/admin/role/:id", v2.AdminRoleDetail)
			admin.POST("/admin/roles", v2.AdminRoleCreate)
			admin.PATCH("/admin/role/:id", v2.AdminRoleUpdate)
			admin.DELETE("/admin/role/:id", v2.AdminRoleDelete)
			admin.GET("/admin/permissions", v2.AdminPermissionList)
			admin.POST("/admin/profile/:id/roles", v2.AdminProfileAssignRole)
			admin.DELETE("/admin/profile/:id/roles/:roleId", v2.AdminProfileRemoveRole)
		}

		apiv2.GET("/problems", v2.ProblemList)
		apiv2.GET("/problems/random", v2.RandomProblem)
		apiv2.GET("/problem/:code", v2.ProblemDetail)
		apiv2.GET("/problem/:code/stats", v2.ProblemStats)

		apiv2.GET("/events", events.ServeWS)

		apiv2.GET("/contests", v2.ContestList)
		apiv2.GET("/contest/:key", v2.ContestDetail)
		apiv2.GET("/contest/:key/ranking", v2.ContestRanking)
		apiv2.GET("/contest/:key/ranking/pdf", v2.ContestRankingPDF)
		apiv2.GET("/contest/:key/participations", v2.ParticipationList)

		apiv2.GET("/submissions", v2.SubmissionList)
		apiv2.GET("/submission/:id", v2.SubmissionDetail)

		apiv2.GET("/users", v2.UserList)
		apiv2.GET("/user/:user", v2.UserDetail)
		apiv2.GET("/user/:user/solved", v2.UserSolvedProblems)
		apiv2.GET("/user/:user/rating", v2.UserRatingHistory)
		apiv2.GET("/user/:user/rating-detail", v2.UserRatingDetail)
		apiv2.GET("/user/:user/analytics", v2.UserAnalytics)

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

		// Protected endpoints
		protected := apiv2.Group("")
		protected.Use(auth.RequiredMiddleware())
		{
			protected.POST("/problem/:code/submit", v2.Submit)
			protected.GET("/user/me", v2.CurrentUser)
			protected.PATCH("/user/me", v2.UpdateProfile)
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
