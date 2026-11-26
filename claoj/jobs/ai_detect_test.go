package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func pairJSON(sub string, ref uint, pct int) string {
	return fmt.Sprintf(`{"pairs":[{"match":1,"a":"./%s","b":"/x/refs/ref_%d","archive_a":false,"archive_b":true,"score":1,"pct_a":%d,"pct_b":10}]}`, sub, ref, pct)
}

// pairsJSON merges the pairs of several single-pair documents.
func pairsJSON(items ...string) string {
	var all []json.RawMessage
	for _, it := range items {
		var d struct{ Pairs []json.RawMessage }
		_ = json.Unmarshal([]byte(it), &d)
		all = append(all, d.Pairs...)
	}
	b, _ := json.Marshal(map[string]any{"pairs": all})
	return string(b)
}

func readAI(t *testing.T, contestID uint) AIReport {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(ContestReportDir(contestID), "ai.json"))
	require.NoError(t, err)
	var r AIReport
	require.NoError(t, json.Unmarshal(b, &r))
	return r
}

// aiFixture: contest with problem knap, C++, one admin reference.
func aiFixture(t *testing.T) (*gorm.DB, fixture, models.Language, models.PlagiarismAIReference) {
	t.Helper()
	useTempPlagiarismDir(t)
	database := setupMossDB(t)
	f := newContest(t, database)
	cpp := addLanguage(t, database, "CPP17", "C++17", "C++")
	ref := addRef(t, database, f.problem.ID, "C++", "int main(){}", "admin")
	return database, f, cpp, ref
}

func lines(tag string) string {
	s := ""
	for i := range 8 {
		s += fmt.Sprintf("%s_line_%d unique statement;\n", tag, i)
	}
	return s
}

func aiCall(chk *fakeChecker) (map[string]string, string) {
	for i, a := range chk.archives {
		if a != "" {
			return chk.calls[i], a
		}
	}
	return nil, ""
}

func TestAIDetect_MatchFlagsRow(t *testing.T) {
	database, f, cpp, ref := aiFixture(t)
	at := f.contest.StartTime.Add(time.Minute)
	for _, u := range []string{"alice", "bob", "carol", "dave"} {
		addAttempt(t, database, f, cpp, u, at, "AC", 100, "code "+u)
	}
	chk := &fakeChecker{results: map[string]string{"knap/cpp/ai": pairsJSON(
		pairJSON("alice", ref.ID, 82), pairJSON("bob", ref.ID, 20), pairJSON("carol", ref.ID, 10), pairJSON("dave", ref.ID, 30))}}
	_, err := RunContestMoss(context.Background(), database, f.contest.ID, chk, nil)
	require.NoError(t, err)

	r := readAI(t, f.contest.ID)
	require.Len(t, r.Rows, 1)
	require.Equal(t, "alice", r.Rows[0].Username)
	require.NotNil(t, r.Rows[0].Match)
	require.Equal(t, 82, r.Rows[0].Match.Pct)
	require.Equal(t, "knap/cpp/ai", r.Rows[0].Match.ReportPath)
	require.Len(t, r.Problems, 1)
	require.Equal(t, 4, r.Problems[0].Contestants)
	require.Equal(t, 1, r.Problems[0].Matched)
	require.False(t, r.Problems[0].Trivial)
	require.Equal(t, 1, r.Problems[0].Admin)
	_, archive := aiCall(chk)
	require.NotEmpty(t, archive)
	require.True(t, filepath.IsAbs(archive))
	require.Equal(t, "refs", filepath.Base(archive))
	require.Contains(t, chk.archiveFolders[len(chk.archiveFolders)-1], fmt.Sprintf("ref_%d", ref.ID))
}

func TestAIDetect_TrivialProblemNeedsRewrites(t *testing.T) {
	database, f, cpp, ref := aiFixture(t)
	at := f.contest.StartTime.Add(time.Minute)
	for _, u := range []string{"alice", "bob", "carol", "dave"} {
		addAttempt(t, database, f, cpp, u, at, "AC", 100, "code "+u)
	}
	chk := &fakeChecker{results: map[string]string{"knap/cpp/ai": pairsJSON(
		pairJSON("alice", ref.ID, 82), pairJSON("bob", ref.ID, 70), pairJSON("carol", ref.ID, 60), pairJSON("dave", ref.ID, 30))}}
	_, err := RunContestMoss(context.Background(), database, f.contest.ID, chk, nil)
	require.NoError(t, err)
	r := readAI(t, f.contest.ID)
	require.True(t, r.Problems[0].Trivial)
	require.Empty(t, r.Rows)

	// alice gets two full rewrites
	addAttempt(t, database, f, cpp, "alice", at.Add(time.Minute), "WA", 0, lines("a"))
	addAttempt(t, database, f, cpp, "alice", at.Add(2*time.Minute), "WA", 0, lines("b"))
	addAttempt(t, database, f, cpp, "alice", at.Add(3*time.Minute), "WA", 0, lines("c"))
	_, err = RunContestMoss(context.Background(), database, f.contest.ID, chk, nil)
	require.NoError(t, err)
	r = readAI(t, f.contest.ID)
	require.Len(t, r.Rows, 1)
	require.Equal(t, "alice", r.Rows[0].Username)
	require.Equal(t, 2, r.Rows[0].Signals)
	require.GreaterOrEqual(t, r.Rows[0].Rewrites, 2)
	require.Len(t, r.Rows[0].RewriteSubs, r.Rows[0].Rewrites)
}

func TestAIDetect_FirstAttemptFolder(t *testing.T) {
	database, f, cpp, ref := aiFixture(t)
	at := f.contest.StartTime.Add(time.Minute)
	addAttempt(t, database, f, cpp, "alice", at, "WA", 0, "alice first")
	addAttempt(t, database, f, cpp, "alice", at.Add(time.Minute), "AC", 100, "alice best")
	addAttempt(t, database, f, cpp, "bob", at, "AC", 100, "bob only")
	addAttempt(t, database, f, cpp, "carol", at, "AC", 100, "carol only")
	addAttempt(t, database, f, cpp, "dave", at, "AC", 100, "dave only")
	chk := &fakeChecker{results: map[string]string{"knap/cpp/ai": pairJSON("alice~first", ref.ID, 77)}}
	_, err := RunContestMoss(context.Background(), database, f.contest.ID, chk, nil)
	require.NoError(t, err)

	call, _ := aiCall(chk)
	require.Equal(t, map[string]string{
		"alice/main.cpp": "alice best", "alice~first/main.cpp": "alice first", "bob/main.cpp": "bob only",
		"carol/main.cpp": "carol only", "dave/main.cpp": "dave only",
	}, call)
	r := readAI(t, f.contest.ID)
	require.Len(t, r.Rows, 1)
	require.Equal(t, "alice", r.Rows[0].Username)
	require.Equal(t, 77, r.Rows[0].Match.Pct)
}

func TestAIDetect_WrappedUsername(t *testing.T) {
	database, f, cpp, ref := aiFixture(t)
	at := f.contest.StartTime.Add(time.Minute)
	addAttempt(t, database, f, cpp, "con", at, "WA", 0, "first")
	addAttempt(t, database, f, cpp, "con", at.Add(time.Minute), "AC", 100, "best")
	for _, u := range []string{"bob", "carol", "dave"} {
		addAttempt(t, database, f, cpp, u, at, "AC", 100, u)
	}
	chk := &fakeChecker{results: map[string]string{"knap/cpp/ai": pairJSON("~con~~first", ref.ID, 90)}}
	_, err := RunContestMoss(context.Background(), database, f.contest.ID, chk, nil)
	require.NoError(t, err)
	call, _ := aiCall(chk)
	require.Contains(t, call, "~con~/main.cpp")
	require.Contains(t, call, "~con~~first/main.cpp")
	r := readAI(t, f.contest.ID)
	require.Len(t, r.Rows, 1)
	require.Equal(t, "con", r.Rows[0].Username)
}

func TestAIDetect_NoReferencesNote(t *testing.T) {
	useTempPlagiarismDir(t)
	database := setupMossDB(t)
	f := newContest(t, database)
	cpp := addLanguage(t, database, "CPP17", "C++17", "C++")
	addAttempt(t, database, f, cpp, "alice", f.contest.StartTime.Add(time.Minute), "AC", 100, "x")
	chk := &fakeChecker{}
	_, err := RunContestMoss(context.Background(), database, f.contest.ID, chk, nil)
	require.NoError(t, err)
	require.Empty(t, chk.calls)
	require.Equal(t, "no_llm", readAI(t, f.contest.ID).Problems[0].Note)

	_, err = RunContestMoss(context.Background(), database, f.contest.ID, chk, &fakeGen{})
	require.NoError(t, err)
	require.Equal(t, "no_statement", readAI(t, f.contest.ID).Problems[0].Note)
	require.Empty(t, chk.calls)
}

func TestAIDetect_OutlierAddsSignalOnly(t *testing.T) {
	database, f, cpp, _ := aiFixture(t)
	require.NoError(t, database.Model(&models.Problem{}).Where("id = ?", f.problem.ID).Update("points", 50).Error)
	old := models.Problem{Code: "old", Name: "Old", Points: 10}
	require.NoError(t, database.Omit("Types", "Groups").Create(&old).Error)
	start := f.contest.StartTime
	a1 := addAttempt(t, database, f, cpp, "alice", start.Add(time.Minute), "AC", 100, "alice code")
	res := "AC"
	pre := models.Submission{UserID: a1.UserID, ProblemID: old.ID, LanguageID: cpp.ID, Status: "D", Result: &res, Date: start.Add(-time.Hour)}
	require.NoError(t, database.Omit("User", "Problem", "Language").Create(&pre).Error)

	_, err := RunContestMoss(context.Background(), database, f.contest.ID, &fakeChecker{}, nil)
	require.NoError(t, err)
	require.Empty(t, readAI(t, f.contest.ID).Rows, "S4 alone is not listed")

	addAttempt(t, database, f, cpp, "alice", start.Add(2*time.Minute), "WA", 0, lines("a"))
	addAttempt(t, database, f, cpp, "alice", start.Add(3*time.Minute), "WA", 0, lines("b"))
	addAttempt(t, database, f, cpp, "alice", start.Add(4*time.Minute), "WA", 0, lines("c"))
	_, err = RunContestMoss(context.Background(), database, f.contest.ID, &fakeChecker{}, nil)
	require.NoError(t, err)
	r := readAI(t, f.contest.ID)
	require.Len(t, r.Rows, 1)
	row := r.Rows[0]
	require.NotNil(t, row.AboveHistory)
	require.True(t, *row.AboveHistory)
	require.NotNil(t, row.PrevMaxPoints)
	require.Equal(t, 10.0, *row.PrevMaxPoints)
	require.Equal(t, 2, row.Signals)
}

func TestAIDetect_FailedRunKeepsOldAIJSON(t *testing.T) {
	database, f, cpp, _ := aiFixture(t)
	at := f.contest.StartTime.Add(time.Minute)
	addAttempt(t, database, f, cpp, "alice", at, "AC", 100, "a")
	addAttempt(t, database, f, cpp, "bob", at, "AC", 100, "b")
	require.NoError(t, os.MkdirAll(ContestReportDir(f.contest.ID), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ContestReportDir(f.contest.ID), "ai.json"), []byte(`{"old":true}`), 0o644))
	_, err := RunContestMoss(context.Background(), database, f.contest.ID, &fakeChecker{err: errors.New("boom")}, nil)
	require.Error(t, err)
	b, err := os.ReadFile(filepath.Join(ContestReportDir(f.contest.ID), "ai.json"))
	require.NoError(t, err)
	require.Equal(t, `{"old":true}`, string(b))
}

func TestAIDetect_RewritesSumAcrossLanguages(t *testing.T) {
	database, f, cpp, _ := aiFixture(t)
	py := addLanguage(t, database, "PY3", "Python 3", "Python")
	at := f.contest.StartTime.Add(time.Minute)
	addAttempt(t, database, f, cpp, "alice", at, "WA", 0, lines("a"))
	addAttempt(t, database, f, cpp, "alice", at.Add(time.Minute), "WA", 0, lines("b"))
	addAttempt(t, database, f, py, "alice", at.Add(2*time.Minute), "WA", 0, lines("c"))
	addAttempt(t, database, f, py, "alice", at.Add(3*time.Minute), "WA", 0, lines("d"))
	_, err := RunContestMoss(context.Background(), database, f.contest.ID, &fakeChecker{}, nil)
	require.NoError(t, err)
	r := readAI(t, f.contest.ID)
	require.NotEmpty(t, r.Rows)
	carriers := 0
	for _, row := range r.Rows {
		require.Equal(t, "alice", row.Username)
		if row.Rewrites > 0 {
			carriers++
			require.Equal(t, 2, row.Rewrites)
		}
	}
	require.Equal(t, 1, carriers, "the rewrite total is carried by exactly one language row")
}
