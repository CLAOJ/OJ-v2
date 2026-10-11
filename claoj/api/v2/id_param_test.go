package v2

import (
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupIDParamDB(t *testing.T) models.AuthUser {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.BlogPost{}, &models.Judge{}, &models.Ticket{}, &models.TicketMessage{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.ProblemClarification{},
	))
	db.DB = database

	u := models.AuthUser{Username: "alice", IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Profile{UserID: u.ID, Timezone: "UTC"}).Error)
	return u
}

// A non-numeric :id must be rejected before it can reach GORM, which would
// otherwise run it as a raw SQL condition.
func TestInlineIDLookups_RejectNonNumericID(t *testing.T) {
	u := setupIDParamDB(t)

	const evil = "1%20OR%201=1"
	cases := []struct {
		name    string
		method  string
		route   string
		target  string
		handler gin.HandlerFunc
	}{
		{"organization detail", http.MethodGet, "/organization/:id", "/organization/" + evil, OrganizationDetail},
		{"organization update", http.MethodPatch, "/organization/:id", "/organization/" + evil, UpdateOrganization},
		{"organization join", http.MethodPost, "/organization/:id/join", "/organization/" + evil + "/join", JoinOrganization},
		{"organization leave", http.MethodPost, "/organization/:id/leave", "/organization/" + evil + "/leave", LeaveOrganization},
		{"organization request", http.MethodPost, "/organization/:id/request", "/organization/" + evil + "/request", RequestJoinOrganization},
		{"organization kick", http.MethodPost, "/organization/:id/kick", "/organization/" + evil + "/kick", KickUser},
		{"blog detail", http.MethodGet, "/blog/:id", "/blog/" + evil, BlogDetail},
		{"blog vote", http.MethodPost, "/blog/:id/vote", "/blog/" + evil + "/vote", BlogVoteHandler},
		{"ticket detail", http.MethodGet, "/ticket/:id", "/ticket/" + evil, TicketDetail},
		{"ticket reply", http.MethodPost, "/ticket/:id/message", "/ticket/" + evil + "/message", TicketReply},
		{"admin ticket detail", http.MethodGet, "/admin/ticket/:id", "/admin/ticket/" + evil, AdminTicketDetail},
		{"judge block", http.MethodPost, "/admin/judge/:id/block", "/admin/judge/" + evil + "/block", AdminJudgeBlock},
		{"judge unblock", http.MethodPost, "/admin/judge/:id/unblock", "/admin/judge/" + evil + "/unblock", AdminJudgeUnblock},
		{"judge enable", http.MethodPost, "/admin/judge/:id/enable", "/admin/judge/" + evil + "/enable", AdminJudgeEnable},
		{"judge disable", http.MethodPost, "/admin/judge/:id/disable", "/admin/judge/" + evil + "/disable", AdminJudgeDisable},
		{"clarification delete", http.MethodDelete, "/admin/problem/clarification/:id", "/admin/problem/clarification/" + evil, ProblemClarificationDelete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := authedRouter(u.ID, tc.method, tc.route, tc.handler)
			body := ""
			if tc.method != http.MethodGet && tc.method != http.MethodDelete {
				body = `{}`
			}
			w := doRequest(router, tc.method, tc.target, body)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			require.Contains(t, w.Body.String(), "invalid id")
		})
	}
}

// A numeric id keeps working.
func TestAdminJudgeBlock_NumericIDStillWorks(t *testing.T) {
	u := setupIDParamDB(t)
	j := models.Judge{Name: "j1", AuthKey: "k", Created: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&j).Error)

	router := authedRouter(u.ID, http.MethodPost, "/admin/judge/:id/block", AdminJudgeBlock)
	w := doRequest(router, http.MethodPost, "/admin/judge/1/block", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.Judge
	require.NoError(t, db.DB.First(&got, j.ID).Error)
	require.True(t, got.IsBlocked)
}
