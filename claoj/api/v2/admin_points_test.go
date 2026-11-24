package v2

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/scoring"
	"github.com/stretchr/testify/require"
)

// The one-off backfill corrects stored profile and organization points that
// predate grading end recalculating them.
func TestAdminRecalculatePoints_BackfillsStaleRows(t *testing.T) {
	fx := setupProfileDB(t)
	addSubmission(t, fx.visibleA, fx.profile, 100, "AC", false, time.Now().Add(-time.Hour))

	org := models.Organization{Name: "Org", Slug: "org", ShortName: "org", CreationDate: time.Now()}
	require.NoError(t, db.DB.Create(&org).Error)
	require.NoError(t, db.DB.Exec(
		"INSERT INTO judge_profile_organizations (profile_id, organization_id) VALUES (?, ?)",
		fx.profile.ID, org.ID,
	).Error)

	root, _ := seedAccount(t, "root", true, true)
	router := authedRouter(root.ID, http.MethodPost, "/admin/recalculate-points", AdminRecalculatePoints)
	w := doRequest(router, http.MethodPost, "/admin/recalculate-points", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct{ Profiles, Organizations, Failed int }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 2, body.Profiles, "solver + root")
	require.Equal(t, 1, body.Organizations)
	require.Zero(t, body.Failed)

	var p models.Profile
	require.NoError(t, db.DB.First(&p, fx.profile.ID).Error)
	require.InDelta(t, 100.0, p.Points, 1e-9)
	require.Equal(t, 1, p.ProblemCount)
	require.InDelta(t, 100+scoring.PpBonusFunction(1), p.PerformancePoints, 1e-9)

	var o models.Organization
	require.NoError(t, db.DB.First(&o, org.ID).Error)
	require.InDelta(t, p.PerformancePoints, o.PerformancePoints, 1e-9)
}

func TestAdminRecalculatePoints_StaffIsForbidden(t *testing.T) {
	fx := setupProfileDB(t)
	addSubmission(t, fx.visibleA, fx.profile, 100, "AC", false, time.Now())

	staff, _ := seedAccount(t, "staff", true, false)
	router := authedRouter(staff.ID, http.MethodPost, "/admin/recalculate-points", AdminRecalculatePoints)
	w := doRequest(router, http.MethodPost, "/admin/recalculate-points", "")
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	var p models.Profile
	require.NoError(t, db.DB.First(&p, fx.profile.ID).Error)
	require.Zero(t, p.PerformancePoints, "a forbidden call must not touch stored points")
}