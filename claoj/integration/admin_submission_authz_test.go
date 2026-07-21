package integration_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	v2 "github.com/CLAOJ/claoj/api/v2"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/integration"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
)

func TestAdminSubmissionAuthz_DeniesPlainStaff(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	prob := models.Problem{Code: "sub_p", Name: "SubP", IsPublic: true}
	require.NoError(t, testDB.DB.Create(&prob).Error)

	staff := integration.CreateTestUser(testDB.DB, "substaff", "Password123!", true)
	makeStaff(t, testDB.DB, staff.ID, true, false)
	staffProfile := profileForUser(t, testDB.DB, staff.ID)
	staffToken := loginToken(t, "substaff", "Password123!")

	sub := models.Submission{UserID: staffProfile.ID, ProblemID: prob.ID, LanguageID: 1, Status: "QU", Date: time.Now()}
	require.NoError(t, testDB.DB.Create(&sub).Error)

	g := integration.TestRouter()
	g.Use(auth.RequiredMiddleware())
	g.Use(auth.AdminRequiredMiddleware())
	g.POST("/admin/submission/:id/rejudge", v2.AdminSubmissionRejudge)
	g.POST("/admin/submission/:id/abort", v2.AdminSubmissionAbort)
	g.POST("/admin/submissions/batch-rejudge", v2.AdminSubmissionBatchRejudge)

	hdr := map[string]string{"Authorization": "Bearer " + staffToken}

	r := integration.MakeRequest(t, g, integration.HTTPRequest{Method: "POST", Path: fmt.Sprintf("/admin/submission/%d/rejudge", sub.ID), Headers: hdr})
	require.Equal(t, http.StatusForbidden, r.Code, "rejudge: non-editor staff must be denied")

	r = integration.MakeRequest(t, g, integration.HTTPRequest{Method: "POST", Path: fmt.Sprintf("/admin/submission/%d/abort", sub.ID), Headers: hdr})
	require.Equal(t, http.StatusForbidden, r.Code, "abort: staff without abort_any_submission must be denied")

	r = integration.MakeRequest(t, g, integration.HTTPRequest{Method: "POST", Path: "/admin/submissions/batch-rejudge", Headers: hdr, Body: map[string]interface{}{"dry_run": true}})
	require.Equal(t, http.StatusForbidden, r.Code, "batch: staff without rejudge_submission_lot must be denied")
}
