package submission

import (
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupDeleteDB(t *testing.T) models.Profile {
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
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.ContestTag{},
		&models.Contest{}, &models.ContestParticipation{},
		&models.Submission{}, &models.SubmissionSource{}, &models.SubmissionTestCase{}, &models.ContestSubmission{},
	))
	db.DB = database

	u := models.AuthUser{Username: "alice", IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	prof := models.Profile{UserID: u.ID, Timezone: "UTC", Points: 150, ProblemCount: 2}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&prof).Error)
	return prof
}

// seedAC inserts a public problem worth pts and an accepted submission of it
// by prof, with a source row and two test cases.
func seedAC(t *testing.T, prof models.Profile, code string, pts float64) models.Submission {
	t.Helper()
	p := models.Problem{Code: code, Name: code, IsPublic: true, Points: pts}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	ac := "AC"
	s := models.Submission{
		UserID: prof.ID, ProblemID: p.ID, LanguageID: 1, Date: time.Now(),
		Status: "D", Result: &ac, Points: &pts, CasePoints: pts, CaseTotal: pts,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&s).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.SubmissionSource{SubmissionID: s.ID, Source: "print(1)"}).Error)
	for i := 1; i <= 2; i++ {
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.SubmissionTestCase{SubmissionID: s.ID, Case: i, Status: "AC"}).Error)
	}
	return s
}

func countWhere(t *testing.T, model any, where string, arg any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.DB.Model(model).Where(where, arg).Count(&n).Error)
	return n
}

func TestDelete_RemovesRowsAndRecalculatesPoints(t *testing.T) {
	prof := setupDeleteDB(t)
	doomed := seedAC(t, prof, "p1", 100)
	kept := seedAC(t, prof, "p2", 50)

	require.NoError(t, NewSubmissionService(nil).Delete(DeleteRequest{SubmissionID: doomed.ID}))

	require.Equal(t, int64(0), countWhere(t, &models.Submission{}, "id = ?", doomed.ID))
	require.Equal(t, int64(0), countWhere(t, &models.SubmissionSource{}, "submission_id = ?", doomed.ID))
	require.Equal(t, int64(0), countWhere(t, &models.SubmissionTestCase{}, "submission_id = ?", doomed.ID))

	// The user's other submission is untouched.
	require.Equal(t, int64(1), countWhere(t, &models.Submission{}, "id = ?", kept.ID))
	require.Equal(t, int64(1), countWhere(t, &models.SubmissionSource{}, "submission_id = ?", kept.ID))
	require.Equal(t, int64(2), countWhere(t, &models.SubmissionTestCase{}, "submission_id = ?", kept.ID))

	var got models.Profile
	require.NoError(t, db.DB.First(&got, prof.ID).Error)
	require.Equal(t, 50.0, got.Points, "points must be recalculated without the deleted submission")
	require.Equal(t, 1, got.ProblemCount)
}

func TestDelete_ContestSubmissionRecomputesParticipation(t *testing.T) {
	prof := setupDeleteDB(t)
	s := seedAC(t, prof, "p1", 100)

	contest := models.Contest{Key: "c1", Name: "C1", StartTime: time.Now().Add(-time.Hour), EndTime: time.Now().Add(time.Hour)}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&contest).Error)
	part := models.ContestParticipation{ContestID: contest.ID, UserID: prof.ID, RealStart: time.Now()}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&part).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.ContestSubmission{
		SubmissionID: s.ID, ProblemID: 1, ParticipationID: part.ID, Points: 100,
	}).Error)

	var gotID uint
	var gotKey string
	calls := 0
	old := recomputeParticipation
	t.Cleanup(func() { recomputeParticipation = old })
	recomputeParticipation = func(p *models.ContestParticipation) error {
		calls++
		gotID, gotKey = p.ID, p.Contest.Key
		return nil
	}

	require.NoError(t, NewSubmissionService(nil).Delete(DeleteRequest{SubmissionID: s.ID}))

	require.Equal(t, int64(0), countWhere(t, &models.ContestSubmission{}, "submission_id = ?", s.ID))
	require.Equal(t, int64(1), countWhere(t, &models.ContestParticipation{}, "id = ?", part.ID), "the participation itself stays")
	require.Equal(t, 1, calls)
	require.Equal(t, part.ID, gotID)
	require.Equal(t, "c1", gotKey, "the participation is handed over with its contest loaded")
}

func TestDelete_NonContestSubmissionSkipsParticipation(t *testing.T) {
	prof := setupDeleteDB(t)
	s := seedAC(t, prof, "p1", 100)

	calls := 0
	old := recomputeParticipation
	t.Cleanup(func() { recomputeParticipation = old })
	recomputeParticipation = func(*models.ContestParticipation) error { calls++; return nil }

	require.NoError(t, NewSubmissionService(nil).Delete(DeleteRequest{SubmissionID: s.ID}))
	require.Equal(t, 0, calls)
}

func TestDelete_InProgressIsRejected(t *testing.T) {
	for _, status := range []string{"QU", "P", "G"} {
		t.Run(status, func(t *testing.T) {
			prof := setupDeleteDB(t)
			s := seedAC(t, prof, "p1", 100)
			require.NoError(t, db.DB.Model(&models.Submission{}).Where("id = ?", s.ID).Update("status", status).Error)

			err := NewSubmissionService(nil).Delete(DeleteRequest{SubmissionID: s.ID})
			require.ErrorIs(t, err, ErrSubmissionInProgress)
			require.Equal(t, int64(1), countWhere(t, &models.Submission{}, "id = ?", s.ID))
			require.Equal(t, int64(1), countWhere(t, &models.SubmissionSource{}, "submission_id = ?", s.ID))
		})
	}
}

func TestDelete_NotFound(t *testing.T) {
	setupDeleteDB(t)
	require.ErrorIs(t, NewSubmissionService(nil).Delete(DeleteRequest{SubmissionID: 999}), ErrSubmissionNotFound)
}
