package jobs

import (
	"errors"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/moss"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type call struct {
	lang, comment string
	files         []moss.File
}

type fakeSubmitter struct {
	calls []call
	err   error
}

func (f *fakeSubmitter) Submit(lang, comment string, files []moss.File) (string, error) {
	f.calls = append(f.calls, call{lang, comment, files})
	if f.err != nil {
		return "", f.err
	}
	return "http://moss.stanford.edu/results/1/" + lang, nil
}

func setupMossDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := database.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.Profile{}, &models.Language{}, &models.Problem{}, &models.Contest{},
		&models.ContestParticipation{}, &models.Submission{}, &models.SubmissionSource{}, &models.ContestSubmission{},
		&models.ContestMoss{}, &models.MiscConfig{},
	))
	require.NoError(t, database.Migrator().DropTable("judge_contestproblem"))
	require.NoError(t, database.Migrator().CreateTable(&models.ContestProblem{}))
	return database
}

// seed builds contest "tik25" with problem "knap" and C++ submissions:
// alice 100pt + alice 40pt (same user, lower), bob 70pt (live), carol (virtual, excluded).
func seed(t *testing.T, database *gorm.DB) models.Contest {
	t.Helper()
	c := models.Contest{Key: "tik25", Name: "TiK25", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}
	require.NoError(t, database.Omit(clause.Associations).Create(&c).Error)
	p := models.Problem{Code: "knap", Name: "Knapsack"}
	require.NoError(t, database.Omit(clause.Associations).Create(&p).Error)
	cp := models.ContestProblem{ContestID: c.ID, ProblemID: p.ID, Points: 100, Order: 1}
	require.NoError(t, database.Omit(clause.Associations).Create(&cp).Error)
	cpp := models.Language{Key: "CPP17", Name: "C++17", CommonName: "C++"}
	require.NoError(t, database.Create(&cpp).Error)

	add := func(username string, virtual int, points float64, source string) {
		var u models.AuthUser
		database.Where("username = ?", username).First(&u)
		if u.ID == 0 {
			u = models.AuthUser{Username: username, IsActive: true}
			require.NoError(t, database.Create(&u).Error)
		}
		var prof models.Profile
		database.Where("user_id = ?", u.ID).First(&prof)
		if prof.ID == 0 {
			prof = models.Profile{UserID: u.ID, Timezone: "UTC"}
			require.NoError(t, database.Omit(clause.Associations).Create(&prof).Error)
		}
		part := models.ContestParticipation{ContestID: c.ID, UserID: prof.ID, RealStart: time.Now(), Virtual: virtual}
		require.NoError(t, database.Omit(clause.Associations).Create(&part).Error)
		sub := models.Submission{UserID: prof.ID, ProblemID: p.ID, LanguageID: cpp.ID, Status: "D", Points: &points, Date: time.Now(), ContestObjectID: &c.ID}
		require.NoError(t, database.Omit(clause.Associations).Create(&sub).Error)
		require.NoError(t, database.Omit(clause.Associations).Create(&models.SubmissionSource{SubmissionID: sub.ID, Source: source}).Error)
		require.NoError(t, database.Omit(clause.Associations).Create(&models.ContestSubmission{SubmissionID: sub.ID, ProblemID: cp.ID, ParticipationID: part.ID}).Error)
	}
	add("alice", 0, 100, "best alice")
	add("alice", 0, 40, "worse alice")
	add("bob", -1, 70, "bob code")
	add("carol", 1, 100, "virtual carol")
	return c
}

func TestRunContestMoss_MirrorsV1(t *testing.T) {
	database := setupMossDB(t)
	c := seed(t, database)
	sub := &fakeSubmitter{}

	n, err := RunContestMoss(database, c.ID, sub)
	require.NoError(t, err)
	require.Equal(t, len(MossLanguages), n, "one row per problem x language, even when empty")

	require.Len(t, sub.calls, 1, "only C++ has submissions")
	require.Equal(t, "cc", sub.calls[0].lang)
	require.Equal(t, "tik25 - knap", sub.calls[0].comment)
	require.Equal(t, []moss.File{{Name: "alice", Content: []byte("best alice")}, {Name: "bob", Content: []byte("bob code")}}, sub.calls[0].files)

	var rows []models.ContestMoss
	require.NoError(t, database.Where("contest_id = ?", c.ID).Find(&rows).Error)
	require.Len(t, rows, len(MossLanguages))
	for _, r := range rows {
		if r.Language == "C++" {
			require.Equal(t, uint(2), r.SubmissionCount)
			require.Equal(t, "http://moss.stanford.edu/results/1/cc", *r.URL)
		} else {
			require.Zero(t, r.SubmissionCount)
			require.Nil(t, r.URL)
		}
	}
}

func TestRunContestMoss_FailureKeepsOldRows(t *testing.T) {
	database := setupMossDB(t)
	c := seed(t, database)
	old := "http://old/report"
	require.NoError(t, database.Create(&models.ContestMoss{ContestID: c.ID, ProblemID: 1, Language: "C++", SubmissionCount: 5, URL: &old}).Error)

	_, err := RunContestMoss(database, c.ID, &fakeSubmitter{err: errors.New("connection reset")})
	require.Error(t, err)

	var rows []models.ContestMoss
	require.NoError(t, database.Where("contest_id = ?", c.ID).Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, old, *rows[0].URL)
}

func TestMossUserID(t *testing.T) {
	database := setupMossDB(t)
	_, err := MossUserID(database)
	require.ErrorIs(t, err, ErrMossNotConfigured)
	require.NoError(t, database.Create(&models.MiscConfig{Key: "moss_user_id", Value: " 12345 "}).Error)
	id, err := MossUserID(database)
	require.NoError(t, err)
	require.Equal(t, "12345", id)
}

func setupMossCache(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	cache.Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { cache.Client = nil })
}

func TestMossRunError_StoresAndClears(t *testing.T) {
	setupMossCache(t)
	require.Equal(t, "", MossRunError(7))
	setMossRunError(7, "connection reset")
	require.Equal(t, "connection reset", MossRunError(7))
	clearMossRunError(7)
	require.Equal(t, "", MossRunError(7))
}

func TestMossRunError_NilCacheIsSafe(t *testing.T) {
	cache.Client = nil
	t.Cleanup(func() { cache.Client = nil })
	require.Equal(t, "", MossRunError(1))
	require.NotPanics(t, func() {
		setMossRunError(1, "boom")
		clearMossRunError(1)
	})
}
