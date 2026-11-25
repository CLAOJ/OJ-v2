package jobs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/models"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// fakeChecker records, per Check call, every "folder/file" under subsDir with
// its content, then writes outDir/index.html (or returns err).
type fakeChecker struct {
	calls    []map[string]string
	err      error
	noReport bool // like compare50 with no similar pairs: exit 0, no -o dir
}

func (f *fakeChecker) Check(_ context.Context, subsDir, outDir string) error {
	got := map[string]string{}
	folders, err := os.ReadDir(subsDir)
	if err != nil {
		return err
	}
	for _, folder := range folders {
		files, err := os.ReadDir(filepath.Join(subsDir, folder.Name()))
		if err != nil {
			return err
		}
		for _, file := range files {
			b, err := os.ReadFile(filepath.Join(subsDir, folder.Name(), file.Name()))
			if err != nil {
				return err
			}
			got[folder.Name()+"/"+file.Name()] = string(b)
		}
	}
	f.calls = append(f.calls, got)
	if f.err != nil {
		return f.err
	}
	if f.noReport {
		return nil
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "index.html"), []byte("<html></html>"), 0o644)
}

func useTempPlagiarismDir(t *testing.T) {
	t.Helper()
	old := PlagiarismDir
	PlagiarismDir = t.TempDir()
	t.Cleanup(func() { PlagiarismDir = old })
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
		&models.ContestMoss{}, &models.PlagiarismResult{},
	))
	require.NoError(t, database.Migrator().DropTable("judge_contestproblem"))
	require.NoError(t, database.Migrator().CreateTable(&models.ContestProblem{}))
	return database
}

type fixture struct {
	contest models.Contest
	problem models.Problem
	cp      models.ContestProblem
}

// newContest creates contest "tik25" with problem "knap" and no submissions.
func newContest(t *testing.T, database *gorm.DB) fixture {
	t.Helper()
	c := models.Contest{Key: "tik25", Name: "TiK25", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}
	require.NoError(t, database.Omit(clause.Associations).Create(&c).Error)
	p := models.Problem{Code: "knap", Name: "Knapsack"}
	require.NoError(t, database.Omit(clause.Associations).Create(&p).Error)
	cp := models.ContestProblem{ContestID: c.ID, ProblemID: p.ID, Points: 100, Order: 1}
	require.NoError(t, database.Omit(clause.Associations).Create(&cp).Error)
	return fixture{contest: c, problem: p, cp: cp}
}

func addLanguage(t *testing.T, database *gorm.DB, key, name, commonName string) models.Language {
	t.Helper()
	l := models.Language{Key: key, Name: name, CommonName: commonName}
	require.NoError(t, database.Create(&l).Error)
	return l
}

func addSubmission(t *testing.T, database *gorm.DB, f fixture, lang models.Language, username string, virtual int, points float64, source string) {
	t.Helper()
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
	part := models.ContestParticipation{ContestID: f.contest.ID, UserID: prof.ID, RealStart: time.Now(), Virtual: virtual}
	require.NoError(t, database.Omit(clause.Associations).Create(&part).Error)
	contestID := f.contest.ID
	sub := models.Submission{UserID: prof.ID, ProblemID: f.problem.ID, LanguageID: lang.ID, Status: "D", Points: &points, Date: time.Now(), ContestObjectID: &contestID}
	require.NoError(t, database.Omit(clause.Associations).Create(&sub).Error)
	require.NoError(t, database.Omit(clause.Associations).Create(&models.SubmissionSource{SubmissionID: sub.ID, Source: source}).Error)
	require.NoError(t, database.Omit(clause.Associations).Create(&models.ContestSubmission{SubmissionID: sub.ID, ProblemID: f.cp.ID, ParticipationID: part.ID}).Error)
}

// seed builds contest "tik25" with problem "knap" and C++ submissions:
// alice 100pt + alice 40pt (same user, lower), bob 70pt (live), carol (virtual, excluded), dave 50pt.
func seed(t *testing.T, database *gorm.DB) models.Contest {
	t.Helper()
	f := newContest(t, database)
	cpp := addLanguage(t, database, "CPP17", "C++17", "C++")
	addSubmission(t, database, f, cpp, "alice", 0, 100, "best alice")
	addSubmission(t, database, f, cpp, "alice", 0, 40, "worse alice")
	addSubmission(t, database, f, cpp, "bob", -1, 70, "bob code")
	addSubmission(t, database, f, cpp, "carol", 1, 100, "virtual carol")
	addSubmission(t, database, f, cpp, "dave", 0, 50, "dave code")
	return f.contest
}

func TestRunContestMoss_WritesReportsAndRows(t *testing.T) {
	useTempPlagiarismDir(t)
	database := setupMossDB(t)
	c := seed(t, database)
	chk := &fakeChecker{}

	n, err := RunContestMoss(context.Background(), database, c.ID, chk)
	require.NoError(t, err)
	require.Equal(t, 1, n, "one row per language present")
	require.Equal(t, []map[string]string{{
		"alice/main.cpp": "best alice",
		"bob/main.cpp":   "bob code",
		"dave/main.cpp":  "dave code",
	}}, chk.calls)

	var rows []models.PlagiarismResult
	require.NoError(t, database.Where("contest_id = ?", c.ID).Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "C++", rows[0].Language)
	require.Equal(t, uint(3), rows[0].SubmissionCount)
	require.NotNil(t, rows[0].ReportPath)
	require.Equal(t, "knap/cpp", *rows[0].ReportPath)
	require.FileExists(t, filepath.Join(ContestReportDir(c.ID), "knap", "cpp", "index.html"))
}

func TestRunContestMoss_SingleSubmissionSkipsChecker(t *testing.T) {
	useTempPlagiarismDir(t)
	database := setupMossDB(t)
	f := newContest(t, database)
	py := addLanguage(t, database, "PY3", "Python 3", "Python")
	addSubmission(t, database, f, py, "alice", 0, 10, "print(1)")
	chk := &fakeChecker{}

	n, err := RunContestMoss(context.Background(), database, f.contest.ID, chk)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Empty(t, chk.calls)

	var rows []models.PlagiarismResult
	require.NoError(t, database.Where("contest_id = ?", f.contest.ID).Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "Python", rows[0].Language)
	require.Equal(t, uint(1), rows[0].SubmissionCount)
	require.Nil(t, rows[0].ReportPath)
}

func TestRunContestMoss_FailureKeepsOldRowsAndFiles(t *testing.T) {
	useTempPlagiarismDir(t)
	database := setupMossDB(t)
	c := seed(t, database)
	old := "old"
	require.NoError(t, database.Create(&models.PlagiarismResult{ContestID: c.ID, ProblemID: 1, Language: "C++", SubmissionCount: 5, ReportPath: &old, CreatedAt: time.Now()}).Error)
	require.NoError(t, os.MkdirAll(filepath.Join(ContestReportDir(c.ID), "old"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ContestReportDir(c.ID), "old", "index.html"), []byte("old"), 0o644))

	_, err := RunContestMoss(context.Background(), database, c.ID, &fakeChecker{err: errors.New("compare50 exploded")})
	require.Error(t, err)

	var rows []models.PlagiarismResult
	require.NoError(t, database.Where("contest_id = ?", c.ID).Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "C++", rows[0].Language)
	require.Equal(t, uint(5), rows[0].SubmissionCount)
	require.NotNil(t, rows[0].ReportPath)
	require.Equal(t, "old", *rows[0].ReportPath)
	require.FileExists(t, filepath.Join(ContestReportDir(c.ID), "old", "index.html"))

	tmps, err := filepath.Glob(filepath.Join(PlagiarismDir, ".tmp-*"))
	require.NoError(t, err)
	require.Empty(t, tmps)
}

func TestRunContestMoss_NoPairsLeavesReportPathNil(t *testing.T) {
	useTempPlagiarismDir(t)
	database := setupMossDB(t)
	f := newContest(t, database)
	py := addLanguage(t, database, "PY3", "Python 3", "Python")
	addSubmission(t, database, f, py, "alice", 0, 10, "print(1)")
	addSubmission(t, database, f, py, "bob", 0, 10, "print(2)")

	_, err := RunContestMoss(context.Background(), database, f.contest.ID, &fakeChecker{noReport: true})
	require.NoError(t, err)

	var nullPaths, count int64
	require.NoError(t, database.Model(&models.PlagiarismResult{}).
		Where("contest_id = ? AND report_path IS NULL AND submission_count = 2", f.contest.ID).Count(&nullPaths).Error)
	require.NoError(t, database.Model(&models.PlagiarismResult{}).Where("contest_id = ?", f.contest.ID).Count(&count).Error)
	require.Equal(t, int64(1), count)
	require.Equal(t, int64(1), nullPaths)
}

func TestRunContestMoss_TieOnPointsPicksLowestID(t *testing.T) {
	useTempPlagiarismDir(t)
	database := setupMossDB(t)
	f := newContest(t, database)
	cpp := addLanguage(t, database, "CPP17", "C++17", "C++")
	addSubmission(t, database, f, cpp, "alice", 0, 100, "first")
	addSubmission(t, database, f, cpp, "alice", 0, 100, "second")
	addSubmission(t, database, f, cpp, "bob", 0, 50, "bob")
	chk := &fakeChecker{}

	_, err := RunContestMoss(context.Background(), database, f.contest.ID, chk)
	require.NoError(t, err)
	require.Equal(t, "first", chk.calls[0]["alice/main.cpp"])
}

func TestRunContestMoss_SkipsUnsafeProblemCodes(t *testing.T) {
	useTempPlagiarismDir(t)
	database := setupMossDB(t)
	f := newContest(t, database)
	cpp := addLanguage(t, database, "CPP17", "C++17", "C++")
	addSubmission(t, database, f, cpp, "alice", 0, 100, "a")
	addSubmission(t, database, f, cpp, "bob", 0, 100, "b")
	bad := models.Problem{Code: "../evil", Name: "Evil"}
	require.NoError(t, database.Omit(clause.Associations).Create(&bad).Error)
	require.NoError(t, database.Omit(clause.Associations).Create(&models.ContestProblem{ContestID: f.contest.ID, ProblemID: bad.ID, Points: 1, Order: 2}).Error)
	badF := f
	badF.problem = bad
	badF.cp = models.ContestProblem{}
	require.NoError(t, database.Where("problem_id = ?", bad.ID).First(&badF.cp).Error)
	addSubmission(t, database, badF, cpp, "alice", 0, 100, "a")
	addSubmission(t, database, badF, cpp, "bob", 0, 100, "b")
	chk := &fakeChecker{}

	n, err := RunContestMoss(context.Background(), database, f.contest.ID, chk)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Len(t, chk.calls, 1)
	_, statErr := os.Stat(filepath.Join(PlagiarismDir, "evil"))
	require.True(t, os.IsNotExist(statErr))
}

func TestRunContestMoss_DBFailureRollsBackSwap(t *testing.T) {
	useTempPlagiarismDir(t)
	database := setupMossDB(t)
	c := seed(t, database)
	_, err := RunContestMoss(context.Background(), database, c.ID, &fakeChecker{})
	require.NoError(t, err)
	marker := filepath.Join(ContestReportDir(c.ID), "knap", "cpp", "marker.txt")
	require.NoError(t, os.WriteFile(marker, []byte("old"), 0o644))

	require.NoError(t, database.Migrator().DropTable(&models.PlagiarismResult{}))
	_, err = RunContestMoss(context.Background(), database, c.ID, &fakeChecker{})
	require.Error(t, err)

	b, readErr := os.ReadFile(marker)
	require.NoError(t, readErr, "old report must be restored")
	require.Equal(t, "old", string(b))
	for _, pat := range []string{".old-*", ".tmp-*"} {
		m, globErr := filepath.Glob(filepath.Join(PlagiarismDir, pat))
		require.NoError(t, globErr)
		require.Empty(t, m, pat)
	}
}

func TestRunContestMoss_LeavesV1TableAlone(t *testing.T) {
	useTempPlagiarismDir(t)
	database := setupMossDB(t)
	c := seed(t, database)
	url := "http://moss/report"
	require.NoError(t, database.Create(&models.ContestMoss{ContestID: c.ID, ProblemID: 1, Language: "C++", SubmissionCount: 4, URL: &url}).Error)

	_, err := RunContestMoss(context.Background(), database, c.ID, &fakeChecker{})
	require.NoError(t, err)

	var v1 []models.ContestMoss
	require.NoError(t, database.Where("contest_id = ?", c.ID).Find(&v1).Error)
	require.Len(t, v1, 1)
	require.Equal(t, uint(4), v1[0].SubmissionCount)
	require.NotNil(t, v1[0].URL)
	require.Equal(t, url, *v1[0].URL)
}

func TestRunContestMoss_SafeFolderNames(t *testing.T) {
	useTempPlagiarismDir(t)
	database := setupMossDB(t)
	f := newContest(t, database)
	cpp := addLanguage(t, database, "CPP17", "C++17", "C++")
	for i, u := range []string{".", "..", "con", "a.", "tik25_lêthanh"} {
		addSubmission(t, database, f, cpp, u, 0, float64(100-i), "src "+u)
	}
	chk := &fakeChecker{}

	_, err := RunContestMoss(context.Background(), database, f.contest.ID, chk)
	require.NoError(t, err)
	require.Len(t, chk.calls, 1)
	require.Equal(t, map[string]string{
		"~.~/main.cpp":           "src .",
		"~..~/main.cpp":          "src ..",
		"~con~/main.cpp":         "src con",
		"~a.~/main.cpp":          "src a.",
		"tik25_lêthanh/main.cpp": "src tik25_lêthanh",
	}, chk.calls[0])
}

func TestSubmissionFolder(t *testing.T) {
	cases := []struct{ in, want string }{
		{"alice", "alice"},
		{"tik25_lêthanh", "tik25_lêthanh"},
		{"a.b", "a.b"},
		{"bob@x+y-z", "bob@x+y-z"},
		{"console", "console"},
		{"com10", "com10"},
		{"aux2", "aux2"},
		{".", "~.~"},
		{"..", "~..~"},
		{".hidden", "~.hidden~"},
		{"a.", "~a.~"},
		{"con", "~con~"},
		{"CON", "~CON~"},
		{"nul.txt", "~nul.txt~"},
		{"com1", "~com1~"},
		{"lpt9.x", "~lpt9.x~"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, SubmissionFolder(c.in), c.in)
	}
}

func TestLanguageExtAndSlug(t *testing.T) {
	exts := []struct{ in, want string }{
		{"C", "c"}, {"C++", "cpp"}, {"Python", "py"}, {"Pascal", "pas"}, {"Java", "java"},
		{"Golang", "go"}, {"Rust", "rs"}, {"Perl", "pl"}, {"Ruby", "rb"}, {"Kotlin", "kt"},
		{"C#", "cs"}, {"PHP", "php"}, {"JavaScript", "js"}, {"Haskell", "hs"}, {"Scala", "scala"},
		{"Swift", "swift"}, {"Lua", "lua"}, {"D", "d"}, {"Brainfuck", "txt"},
	}
	for _, c := range exts {
		require.Equal(t, c.want, LanguageExt(c.in), c.in)
	}
	slugs := []struct{ in, want string }{
		{"C++", "cpp"}, {"C#", "csharp"}, {"Python", "python"}, {"Golang", "golang"},
		{"Visual Basic .NET", "visual_basic__net"},
	}
	for _, c := range slugs {
		require.Equal(t, c.want, LanguageSlug(c.in), c.in)
	}
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
