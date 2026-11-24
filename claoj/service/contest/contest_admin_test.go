package contest

import (
	"strings"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/relations"
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
func i64p(v int64) *int64   { return &v }

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

// TestUpdateContest_BanningClearsCurrentContest mirrors v1's ContestForm.clean:
// banning kicks the user out of this contest, but never out of another one.
func TestUpdateContest_BanningClearsCurrentContest(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "banme")
	other := newContest(t, "other")
	third := newContest(t, "third")
	heres := newProfile(t, "heres")
	elsewhere := newProfile(t, "elsewhere")
	kept := newProfile(t, "kept")

	for _, p := range []struct {
		profile models.Profile
		contest uint
	}{{heres, c.ID}, {elsewhere, other.ID}, {kept, third.ID}} {
		require.NoError(t, db.DB.Model(&models.Profile{}).Where("id = ?", p.profile.ID).
			Update("current_contest_id", p.contest).Error)
	}

	// Banned: heres (in this contest) and elsewhere (in a different one).
	banned := []uint{heres.ID, elsewhere.ID}
	_, err := NewContestService().UpdateContest(UpdateContestRequest{ContestKey: c.Key, BannedUserIDs: &banned})
	require.NoError(t, err)

	current := func(p models.Profile) *uint {
		var got models.Profile
		require.NoError(t, db.DB.First(&got, p.ID).Error)
		return got.CurrentContestID
	}
	require.Nil(t, current(heres), "a user banned from this contest is kicked out")
	require.NotNil(t, current(elsewhere))
	require.Equal(t, other.ID, *current(elsewhere), "a different current contest is left alone")
	require.NotNil(t, current(kept))
	require.Equal(t, third.ID, *current(kept), "non-banned users are never touched")
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

func ip(v int) *int   { return &v }
func bp(v bool) *bool { return &v }
func up(v uint) *uint { return &v }

// newOrg/newClass seed the v1 access relations (organizations + classes).
func newOrg(t *testing.T, name string) models.Organization {
	t.Helper()
	o := models.Organization{Name: name, Slug: name, ShortName: name, CreationDate: time.Now()}
	require.NoError(t, db.DB.Omit("Admins", "Members").Create(&o).Error)
	return o
}

func newClass(t *testing.T, org models.Organization, name string) models.Class {
	t.Helper()
	c := models.Class{OrganizationID: org.ID, Name: name, Slug: name}
	require.NoError(t, db.DB.Create(&c).Error)
	return c
}

func TestCreateContest_V1FieldsRoundTrip(t *testing.T) {
	setupContestDB(t)
	viewer := newProfile(t, "viewer")
	troll := newProfile(t, "troll")
	private := newProfile(t, "private")
	org := newOrg(t, "org")
	joinOrg := newOrg(t, "joinorg")
	class := newClass(t, org, "class10a")

	noClarifications := false
	start := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	svc := NewContestService()
	_, err := svc.CreateContest(CreateContestRequest{
		Key: "full", Name: "Full", Description: "<p>d</p>",
		StartTime: start, EndTime: start.Add(2 * time.Hour),
		ScoreboardVisibility:       "C",
		ScoreboardCacheTimeout:     30,
		UseClarifications:          &noClarifications,
		PushAnnouncements:          true,
		RatingFloor:                ip(1200),
		RatingCeiling:              ip(2000),
		PerformanceCeilingOverride: ip(2100),
		RateAll:                    true,
		IsPrivate:                  true,
		HideProblemAuthors:         true,
		ShowShortDisplay:           true,
		LimitJoinOrganizations:     true,
		OgImage:                    "https://example.com/og.png",
		LogoOverrideImage:          "https://example.com/logo.png",
		ProblemLabelScript:         "return 'A'",
		PointsPrecision:            ip(5),
		OrganizationIDs:            []uint{org.ID},
		BannedUserIDs:              []uint{troll.ID},
		PrivateContestantIDs:       []uint{private.ID},
		ViewScoreboardUserIDs:      []uint{viewer.ID},
		JoinOrganizationIDs:        []uint{joinOrg.ID},
		ClassIDs:                   []uint{class.ID},
	})
	require.NoError(t, err)

	got, err := svc.GetContest(GetContestRequest{ContestKey: "full"})
	require.NoError(t, err)
	require.Equal(t, "C", got.ScoreboardVisibility)
	require.Equal(t, uint(30), got.ScoreboardCacheTimeout)
	require.False(t, got.UseClarifications)
	require.True(t, got.PushAnnouncements)
	require.Equal(t, ip(1200), got.RatingFloor)
	require.Equal(t, ip(2000), got.RatingCeiling)
	require.Equal(t, ip(2100), got.PerformanceCeilingOverride)
	require.True(t, got.RateAll)
	require.True(t, got.IsPrivate)
	require.True(t, got.HideProblemAuthors)
	require.True(t, got.ShowShortDisplay)
	require.True(t, got.LimitJoinOrganizations)
	require.Equal(t, "https://example.com/og.png", got.OgImage)
	require.Equal(t, "https://example.com/logo.png", got.LogoOverrideImage)
	require.Equal(t, "return 'A'", got.ProblemLabelScript)
	require.Equal(t, 5, got.PointsPrecision)
	require.Equal(t, []uint{org.ID}, got.OrganizationIDs)
	require.Equal(t, []uint{troll.ID}, got.BannedUserIDs)
	require.Equal(t, []uint{private.ID}, got.PrivateContestantIDs)
	require.Equal(t, []uint{viewer.ID}, got.ViewScoreboardUserIDs)
	require.Equal(t, []uint{joinOrg.ID}, got.JoinOrganizationIDs)
	require.Equal(t, []uint{class.ID}, got.ClassIDs)
	require.Equal(t, "troll", got.BannedUsers[0].Username)
	require.Equal(t, "private", got.PrivateContestants[0].Username)
	require.Equal(t, "viewer", got.ViewScoreboardUsers[0].Username)
	require.Equal(t, []NamedRef{{ID: joinOrg.ID, Name: "joinorg"}}, got.JoinOrganizations)
	require.Equal(t, []NamedRef{{ID: org.ID, Name: "org"}}, got.Organizations)
	require.Equal(t, []NamedRef{{ID: class.ID, Name: "class10a"}}, got.Classes)
}

func TestCreateContest_DefaultsMatchV1(t *testing.T) {
	setupContestDB(t)
	start := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	svc := NewContestService()
	_, err := svc.CreateContest(CreateContestRequest{
		Key: "defaults", Name: "Defaults", StartTime: start, EndTime: start.Add(time.Hour),
	})
	require.NoError(t, err)
	got, err := svc.GetContest(GetContestRequest{ContestKey: "defaults"})
	require.NoError(t, err)
	require.Equal(t, "V", got.ScoreboardVisibility)
	require.True(t, got.UseClarifications)
	require.Equal(t, 3, got.PointsPrecision)
	require.Empty(t, got.ClassIDs)
	require.Empty(t, got.Classes)
	require.Empty(t, got.Organizations)
}

func TestUpdateContest_V1FieldsReplaceAndClear(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "updv1")
	viewer := newProfile(t, "viewer")
	troll := newProfile(t, "troll")
	private := newProfile(t, "private")
	org := newOrg(t, "org")
	joinOrg := newOrg(t, "joinorg")
	class := newClass(t, org, "class10a")
	svc := NewContestService()

	users := func(p models.Profile) *[]uint { return &[]uint{p.ID} }
	_, err := svc.UpdateContest(UpdateContestRequest{
		ContestKey:                 c.Key,
		ScoreboardVisibility:       strp("P"),
		ScoreboardCacheTimeout:     up(60),
		UseClarifications:          bp(false),
		PushAnnouncements:          bp(true),
		HideProblemAuthors:         bp(true),
		ShowShortDisplay:           bp(true),
		LimitJoinOrganizations:     bp(true),
		RateAll:                    bp(true),
		IsPrivate:                  bp(true),
		RatingFloor:                ip(1200),
		RatingCeiling:              ip(2000),
		PerformanceCeilingOverride: ip(2100),
		PointsPrecision:            ip(0),
		OgImage:                    strp("https://example.com/og.png"),
		LogoOverrideImage:          strp("https://example.com/logo.png"),
		ProblemLabelScript:         strp("return 'A'"),
		OrganizationIDs:            &[]uint{org.ID},
		BannedUserIDs:              users(troll),
		PrivateContestantIDs:       users(private),
		ViewScoreboardUserIDs:      users(viewer),
		JoinOrganizationIDs:        &[]uint{joinOrg.ID},
		ClassIDs:                   &[]uint{class.ID},
	})
	require.NoError(t, err)

	got, err := svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.NoError(t, err)
	require.Equal(t, "P", got.ScoreboardVisibility)
	require.Equal(t, uint(60), got.ScoreboardCacheTimeout)
	require.False(t, got.UseClarifications)
	require.True(t, got.PushAnnouncements)
	require.True(t, got.HideProblemAuthors)
	require.True(t, got.ShowShortDisplay)
	require.True(t, got.LimitJoinOrganizations)
	require.True(t, got.RateAll)
	require.True(t, got.IsPrivate)
	require.Equal(t, ip(1200), got.RatingFloor)
	require.Equal(t, ip(2000), got.RatingCeiling)
	require.Equal(t, ip(2100), got.PerformanceCeilingOverride)
	require.Equal(t, 0, got.PointsPrecision)
	require.Equal(t, "https://example.com/og.png", got.OgImage)
	require.Equal(t, []uint{troll.ID}, got.BannedUserIDs)
	require.Equal(t, []uint{class.ID}, got.ClassIDs)
	require.Equal(t, []uint{joinOrg.ID}, got.JoinOrganizationIDs)

	// Empty full lists clear the relations; the clear flags null the ints.
	empty := []uint{}
	_, err = svc.UpdateContest(UpdateContestRequest{
		ContestKey:                      c.Key,
		OrganizationIDs:                 &empty,
		BannedUserIDs:                   &empty,
		PrivateContestantIDs:            &empty,
		ViewScoreboardUserIDs:           &empty,
		JoinOrganizationIDs:             &empty,
		ClassIDs:                        &empty,
		ClearRatingFloor:                true,
		ClearRatingCeiling:              true,
		ClearPerformanceCeilingOverride: true,
		ScoreboardCacheTimeout:          up(0),
		LimitJoinOrganizations:          bp(false),
		IsPrivate:                       bp(false),
	})
	require.NoError(t, err)

	got, err = svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.NoError(t, err)
	require.Equal(t, uint(0), got.ScoreboardCacheTimeout)
	require.False(t, got.LimitJoinOrganizations)
	require.False(t, got.IsPrivate)
	require.Empty(t, got.OrganizationIDs)
	require.Empty(t, got.BannedUserIDs)
	require.Empty(t, got.PrivateContestantIDs)
	require.Empty(t, got.ViewScoreboardUserIDs)
	require.Empty(t, got.JoinOrganizationIDs)
	require.Empty(t, got.ClassIDs)
	require.Empty(t, got.Classes)
	require.Nil(t, got.RatingFloor)
	require.Nil(t, got.RatingCeiling)
	require.Nil(t, got.PerformanceCeilingOverride)
}

func TestUpdateContest_RejectsInvalidEnumsAndUnknownRelation(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "badfields")
	svc := NewContestService()

	_, err := svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, ScoreboardVisibility: strp("X")})
	require.ErrorIs(t, err, ErrInvalidScoreboardVisibility)
	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, PointsPrecision: ip(11)})
	require.ErrorIs(t, err, ErrInvalidPointsPrecision)
	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, PointsPrecision: ip(-1)})
	require.ErrorIs(t, err, ErrInvalidPointsPrecision)
	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, ClassIDs: &[]uint{99999}})
	require.ErrorIs(t, err, relations.ErrUnknownID)

	stored, err := svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.NoError(t, err)
	require.Equal(t, "V", stored.ScoreboardVisibility)
	require.Equal(t, 3, stored.PointsPrecision)
}

func TestContest_ImageLengthCaps(t *testing.T) {
	setupContestDB(t)
	long := strings.Repeat("x", 151)
	start := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	svc := NewContestService()

	_, err := svc.CreateContest(CreateContestRequest{
		Key: "longog", Name: "Long", StartTime: start, EndTime: start.Add(time.Hour),
		OgImage: long,
	})
	require.ErrorIs(t, err, ErrFieldTooLong)
	var count int64
	require.NoError(t, db.DB.Model(&models.Contest{}).Where("`key` = ?", "longog").Count(&count).Error)
	require.Zero(t, count, "a rejected field must not leave a contest row")

	c := newContest(t, "lenlogo")
	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, LogoOverrideImage: strp(long)})
	require.ErrorIs(t, err, ErrFieldTooLong)
	ok := strings.Repeat("é", 150) // 150 runes, 300 bytes
	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, LogoOverrideImage: strp(ok)})
	require.NoError(t, err)
}

func contestRows(t *testing.T, contestID uint) []models.ContestProblem {
	t.Helper()
	var rows []models.ContestProblem
	require.NoError(t, db.DB.Where("contest_id = ?", contestID).Order("`order`").Find(&rows).Error)
	return rows
}

func TestReplaceContestProblems_UpdatesInPlaceInsertsAndOrders(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "rp")
	p1, p2, p3 := newProblem(t, "p1"), newProblem(t, "p2"), newProblem(t, "p3")
	existing := models.ContestProblem{ContestID: c.ID, ProblemID: p1.ID, Points: 100, Partial: true, Order: 1}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&existing).Error)

	items := []ContestProblemInput{
		{ProblemID: p2.ID, Points: ip(250)},
		{ProblemID: p1.ID, Partial: bp(false)},
		{ProblemID: p3.ID},
	}
	require.NoError(t, db.DB.Transaction(func(tx *gorm.DB) error {
		return NewContestProblemService().ReplaceContestProblems(tx, c.ID, items)
	}))

	rows := contestRows(t, c.ID)
	require.Len(t, rows, 3)
	require.Equal(t, []uint{p2.ID, p1.ID, p3.ID}, []uint{rows[0].ProblemID, rows[1].ProblemID, rows[2].ProblemID})
	require.Equal(t, 250, rows[0].Points)
	require.Equal(t, existing.ID, rows[1].ID, "existing row must keep its id")
	require.Equal(t, 100, rows[1].Points, "omitted points keep the existing value")
	require.False(t, rows[1].Partial)
	require.Equal(t, 100, rows[2].Points, "new rows default to 100 points")
	require.True(t, rows[2].Partial)
}

func TestReplaceContestProblems_RefusesRemovingProblemWithSubmissions(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "subs")
	p1, p2 := newProblem(t, "keep"), newProblem(t, "solved")
	cp1 := models.ContestProblem{ContestID: c.ID, ProblemID: p1.ID, Points: 100, Order: 1}
	cp2 := models.ContestProblem{ContestID: c.ID, ProblemID: p2.ID, Points: 100, Order: 2}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&cp1).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&cp2).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.ContestSubmission{SubmissionID: 1, ProblemID: cp2.ID, ParticipationID: 1}).Error)

	_, err := NewContestService().UpdateContest(UpdateContestRequest{
		ContestKey: c.Key,
		Name:       strp("changed"),
		Problems:   &[]ContestProblemInput{{ProblemID: p1.ID}},
	})
	require.ErrorIs(t, err, ErrContestProblemHasSubmissions)
	require.Contains(t, err.Error(), "solved")
	require.Len(t, contestRows(t, c.ID), 2, "nothing may be deleted")
	var stored models.Contest
	require.NoError(t, db.DB.First(&stored, c.ID).Error)
	require.Equal(t, "subs", stored.Name, "the whole update must roll back")
}

func TestReplaceContestProblems_RejectsUnknownDuplicateAndNegative(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "bad")
	p := newProblem(t, "only")
	svc := NewContestProblemService()
	run := func(items []ContestProblemInput) error {
		return db.DB.Transaction(func(tx *gorm.DB) error { return svc.ReplaceContestProblems(tx, c.ID, items) })
	}
	require.ErrorIs(t, run([]ContestProblemInput{{ProblemID: 404}}), ErrUnknownProblem)
	require.ErrorIs(t, run([]ContestProblemInput{{ProblemID: p.ID}, {ProblemID: p.ID}}), ErrUnknownProblem)
	require.ErrorIs(t, run([]ContestProblemInput{{ProblemID: p.ID, Points: ip(-1)}}), ErrInvalidPoints)
	require.Empty(t, contestRows(t, c.ID))
}

func TestUpdateContest_LegacyAddRemoveProblems(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "legacy")
	p1, p2 := newProblem(t, "l1"), newProblem(t, "l2")
	svc := NewContestService()

	_, err := svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, AddProblemIDs: []uint{p1.ID, p2.ID}})
	require.NoError(t, err)
	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, RemoveProblemIDs: []uint{p1.ID}})
	require.NoError(t, err)
	rows := contestRows(t, c.ID)
	require.Len(t, rows, 1)
	require.Equal(t, p2.ID, rows[0].ProblemID)
	require.Equal(t, uint(1), rows[0].Order)
}

func TestListContests_SearchAndStatus(t *testing.T) {
	setupContestDB(t)
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	mk := func(key, name string, start time.Time) {
		c := models.Contest{Key: key, Name: name, StartTime: start, EndTime: start.Add(2 * time.Hour)}
		require.NoError(t, db.DB.Omit(clause.Associations).Create(&c).Error)
	}
	mk("past1", "Halloween Contest", now.Add(-48*time.Hour))
	mk("live1", "Live Round", now.Add(-time.Hour))
	mk("next1", "Next Halloween", now.Add(24*time.Hour))
	svc := NewContestService()

	resp, err := svc.ListContests(ListContestsRequest{Page: 1, PageSize: 10, Search: "halloween", Now: now})
	require.NoError(t, err)
	require.Equal(t, int64(2), resp.Total)

	for status, key := range map[string]string{"past": "past1", "ongoing": "live1", "upcoming": "next1"} {
		resp, err = svc.ListContests(ListContestsRequest{Page: 1, PageSize: 10, Status: status, Now: now})
		require.NoError(t, err)
		require.Equal(t, int64(1), resp.Total, status)
		require.Equal(t, key, resp.Contests[0].Key, status)
	}

	_, err = svc.ListContests(ListContestsRequest{Page: 1, PageSize: 10, Status: "soon"})
	require.ErrorIs(t, err, ErrInvalidStatus)
}

func TestUpdateContest_ICPCFormatConfigTypes(t *testing.T) {
	setupContestDB(t)
	c := newContest(t, "icpcfmt")
	svc := NewContestService()

	_, err := svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, FormatConfig: strp(`{"penalty": "x"}`)})
	require.ErrorIs(t, err, ErrInvalidFormatConfig)

	_, err = svc.UpdateContest(UpdateContestRequest{ContestKey: c.Key, FormatConfig: strp(`{"penalty": 20}`)})
	require.NoError(t, err)
	got, err := svc.GetContest(GetContestRequest{ContestKey: c.Key})
	require.NoError(t, err)
	require.Equal(t, float64(20), got.FormatConfig["penalty"])
}
