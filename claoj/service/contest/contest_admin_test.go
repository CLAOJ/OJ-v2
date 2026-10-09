package contest

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

func setupContestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Organization{}, &models.Language{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Problem{}, &models.ContestTag{},
		&models.Contest{}, &models.ContestParticipation{}, &models.Submission{}, &models.ContestSubmission{},
	))
	require.NoError(t, database.Migrator().DropTable("judge_contestproblem"))
	require.NoError(t, database.Migrator().CreateTable(&models.ContestProblem{}))
	db.DB = database
	return database
}

func newContest(t *testing.T, key string) models.Contest {
	t.Helper()
	start := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	c := models.Contest{Key: key, Name: key, StartTime: start, EndTime: start.Add(3 * time.Hour), FormatName: "icpc"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)
	return c
}

func newProfile(t *testing.T, username string) models.Profile {
	t.Helper()
	u := models.AuthUser{Username: username, IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	p := models.Profile{UserID: u.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	return p
}

func newProblem(t *testing.T, code string) models.Problem {
	t.Helper()
	p := models.Problem{Code: code, Name: code}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&p).Error)
	return p
}

func strp(s string) *string { return &s }
func i64p(v int64) *int64 { return &v }

func TestUpdateContest_AppliesSchedule(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "sched")
	svc := NewContestService()

	_, err := svc.UpdateContest(UpdateContestRequest{
		ContestKey: c.Key,
		StartTime:  strp("2026-11-01T09:00:00+07:00"),
		EndTime:    strp("2026-11-01T12:00:00+07:00"),
	})
	require.NoError(t, err)
	got, err := svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.NoError(t, err)
	require.True(t, got.StartTime.Equal(time.Date(2026, 11, 1, 2, 0, 0, 0, time.UTC)))
	require.True(t, got.EndTime.Equal(time.Date(2026, 11, 1, 5, 0, 0, 0, time.UTC)))

	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, EndTime: strp("2026-10-31T00:00:00Z")})
	require.ErrorIs(t, err, ErrInvalidSchedule)

	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, StartTime: strp("2026-11-01T09:00")})
	require.ErrorIs(t, err, ErrInvalidTime)
}

func TestUpdateContest_TimeLimitMicrosecondsAndClear(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "tl")
	svc := NewContestService()

	_, err := svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, TimeLimit: i64p(3 * 3600 * 1_000_000)})
	require.NoError(t, err)
	got, _ := svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.NotNil(t, got.TimeLimitSeconds)
	require.Equal(t, int64(10800), *got.TimeLimitSeconds)

	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, TimeLimit: i64p(0)})
	require.NoError(t, err)
	got, _ = svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.Nil(t, got.TimeLimitSeconds)
}

func TestUpdateContest_FormatValidation(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "fmt")
	svc := NewContestService()

	_, err := svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, FormatName: strp("ioi16"), FormatConfig: strp(`{"cumtime": true}`)})
	require.NoError(t, err)
	got, _ := svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.Equal(t, "ioi16", got.FormatName)
	require.Equal(t, true, got.FormatConfig["cumtime"])

	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, FormatName: strp("nope")})
	require.ErrorIs(t, err, ErrUnknownFormat)
	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, FormatConfig: strp(`[1,2]`)})
	require.ErrorIs(t, err, ErrInvalidFormatConfig)
}

func TestUpdateContest_StaffAndTagsFullLists(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "staff")
	a := newProfile(t, "alice")
	b := newProfile(t, "bob")
	tag := models.ContestTag{Name: "div2", Color: "#000000"}
	require.NoError(t, db.DB.Create(&tag).Error)
	svc := NewContestService()

	authors := []uint{a.ID, b.ID}
	testers := []uint{b.ID}
	tags := []uint{tag.ID}
	_, err := svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, AuthorIDs: &authors, TesterIDs: &testers, TagIDs: &tags})
	require.NoError(t, err)
	got, _ := svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.ElementsMatch(t, authors, got.AuthorIDs)
	require.Equal(t, testers, got.TesterIDs)
	require.Equal(t, tags, got.TagIDs)
	require.Equal(t, "bob", got.Testers[0].Username)

	name := "Renamed"
	bad := []uint{99999}
	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, Name: &name, CuratorIDs: &bad})
	require.Error(t, err)
	got, _ = svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.Equal(t, "staff", got.Name, "failed relation must roll back the rename")
}

func TestGetContest_ProblemsCarryCodeAndName(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "probs")
	p := newProblem(t, "aplusb")
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.ContestProblem{ContestID: c.ID, ProblemID: p.ID, Points: 100, Partial: true, Order: 1}).Error)

	got, err := NewContestService().GetContest(GetContestRequest{ContestKey: c.Key})
	require.NoError(t, err)
	require.Len(t, got.Problems, 1)
	require.Equal(t, "aplusb", got.Problems[0].Code)
}
