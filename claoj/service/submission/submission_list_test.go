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

func TestListSubmissions_Filters(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := database.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.AuthUser{}, &models.Profile{}, &models.Problem{}, &models.Language{}, &models.Submission{}))
	db.DB = database

	mkUser := func(name string) models.Profile {
		u := models.AuthUser{Username: name, IsActive: true}
		require.NoError(t, database.Create(&u).Error)
		p := models.Profile{UserID: u.ID, Timezone: "UTC"}
		require.NoError(t, database.Omit(clause.Associations).Create(&p).Error)
		return p
	}
	alice, bob := mkUser("alice"), mkUser("bob")
	pa := models.Problem{Code: "aplusb", Name: "A+B"}
	pk := models.Problem{Code: "knap", Name: "Knapsack"}
	require.NoError(t, database.Omit(clause.Associations).Create(&pa).Error)
	require.NoError(t, database.Omit(clause.Associations).Create(&pk).Error)
	cpp := models.Language{Key: "CPP17", Name: "C++17", CommonName: "C++"}
	py := models.Language{Key: "PY3", Name: "Python 3", CommonName: "Python"}
	require.NoError(t, database.Create(&cpp).Error)
	require.NoError(t, database.Create(&py).Error)

	ac, wa := "AC", "WA"
	for _, s := range []models.Submission{
		{UserID: alice.ID, ProblemID: pa.ID, LanguageID: cpp.ID, Status: "D", Result: &ac, Date: time.Now()},
		{UserID: alice.ID, ProblemID: pk.ID, LanguageID: py.ID, Status: "D", Result: &wa, Date: time.Now()},
		{UserID: bob.ID, ProblemID: pa.ID, LanguageID: cpp.ID, Status: "QU", Date: time.Now()},
	} {
		s := s
		require.NoError(t, database.Omit(clause.Associations).Create(&s).Error)
	}

	svc := &SubmissionService{}
	cases := []struct {
		req  ListSubmissionsRequest
		want int64
	}{
		{ListSubmissionsRequest{}, 3},
		{ListSubmissionsRequest{Username: "alice"}, 2},
		{ListSubmissionsRequest{ProblemCode: "aplusb"}, 2},
		{ListSubmissionsRequest{Result: "AC"}, 1},
		{ListSubmissionsRequest{Status: "QU"}, 1},
		{ListSubmissionsRequest{Language: "PY3"}, 1},
		{ListSubmissionsRequest{Language: "C++17"}, 2},
		{ListSubmissionsRequest{Username: "alice", ProblemCode: "aplusb"}, 1},
	}
	for _, tc := range cases {
		tc.req.Page, tc.req.PageSize = 1, 20
		resp, err := svc.ListSubmissions(tc.req)
		require.NoError(t, err)
		require.Equal(t, tc.want, resp.Total, "%+v", tc.req)
		require.Len(t, resp.Submissions, int(tc.want), "%+v", tc.req)
	}
}
