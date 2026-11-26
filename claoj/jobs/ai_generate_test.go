package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeGen returns "code N" for the N-th call; failOn (1-based) makes that call fail.
type fakeGen struct {
	calls  atomic.Int32
	failOn int32
	mu     sync.Mutex
	inputs []GenerateInput
}

func (f *fakeGen) Generate(_ context.Context, in GenerateInput) (string, error) {
	n := f.calls.Add(1)
	f.mu.Lock()
	f.inputs = append(f.inputs, in)
	f.mu.Unlock()
	if n == f.failOn {
		return "", errors.New("upstream exploded")
	}
	return fmt.Sprintf("code %d", n), nil
}

func (f *fakeGen) Model() string { return "fake-model" }

func longStatement() string { return strings.Repeat("Solve the knapsack problem. ", 20) }

func countRefs(t *testing.T, database *gorm.DB, problemID uint, lang, origin string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, database.Model(&models.PlagiarismAIReference{}).
		Where("problem_id = ? AND language = ? AND origin = ?", problemID, lang, origin).Count(&n).Error)
	return n
}

func TestEnsureReferences_TopsUpToFive(t *testing.T) {
	database := setupMossDB(t)
	f := newContest(t, database)
	f.problem.Description = longStatement()
	now := time.Now()
	for _, r := range []models.PlagiarismAIReference{
		{ProblemID: f.problem.ID, Language: "C++", Source: "old 1", Origin: "generated", CreatedAt: now},
		{ProblemID: f.problem.ID, Language: "C++", Source: "old 2", Origin: "generated", CreatedAt: now},
		{ProblemID: f.problem.ID, Language: "C++", Source: "judge's", Origin: "admin", CreatedAt: now},
	} {
		r := r
		require.NoError(t, database.Create(&r).Error)
	}

	gen := &fakeGen{}
	note, err := EnsureReferences(context.Background(), database, gen, f.problem, "C++")
	require.NoError(t, err)
	require.Equal(t, "", note)
	require.Equal(t, int32(3), gen.calls.Load())
	require.Equal(t, int64(aiRefsPerLang), countRefs(t, database, f.problem.ID, "C++", "generated"))
	require.Equal(t, int64(1), countRefs(t, database, f.problem.ID, "C++", "admin"))

	var stored []models.PlagiarismAIReference
	require.NoError(t, database.Where("origin = ? AND source LIKE ?", "generated", "code %").Find(&stored).Error)
	require.Len(t, stored, 3)
	for _, r := range stored {
		require.NotNil(t, r.Model)
		require.Equal(t, "fake-model", *r.Model)
	}
}

func TestEnsureReferences_NoStatement(t *testing.T) {
	database := setupMossDB(t)
	f := newContest(t, database)
	f.problem.Description = strings.Repeat("x", 50)

	gen := &fakeGen{}
	note, err := EnsureReferences(context.Background(), database, gen, f.problem, "C++")
	require.NoError(t, err)
	require.Equal(t, "no_statement", note)
	require.Equal(t, int32(0), gen.calls.Load())
}

func TestEnsureReferences_NilGenerator(t *testing.T) {
	database := setupMossDB(t)
	f := newContest(t, database)
	f.problem.Description = longStatement()

	note, err := EnsureReferences(context.Background(), database, nil, f.problem, "C++")
	require.NoError(t, err)
	require.Equal(t, "no_llm", note)
	require.Equal(t, int64(0), countRefs(t, database, f.problem.ID, "C++", "generated"))
}

func TestEnsureReferences_PartialFailure(t *testing.T) {
	database := setupMossDB(t)
	f := newContest(t, database)
	f.problem.Description = longStatement()

	gen := &fakeGen{failOn: 2}
	note, err := EnsureReferences(context.Background(), database, gen, f.problem, "C++")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(note, "generation_failed: "), note)
	require.Contains(t, note, "upstream exploded")
	require.Equal(t, int64(aiRefsPerLang-1), countRefs(t, database, f.problem.ID, "C++", "generated"))
}

type emptyGen struct{ fakeGen }

func (e *emptyGen) Generate(ctx context.Context, in GenerateInput) (string, error) {
	if n := e.calls.Add(1); n <= 2 {
		return "  ", nil
	}
	return "real code", nil
}

func TestEnsureReferences_EmptyReplyIsFailure(t *testing.T) {
	database := setupMossDB(t)
	f := newContest(t, database)
	f.problem.Description = longStatement()
	note, err := EnsureReferences(context.Background(), database, &emptyGen{}, f.problem, "C++")
	require.NoError(t, err)
	require.Contains(t, note, "generation_failed: empty reply")
	require.Equal(t, int64(aiRefsPerLang-2), countRefs(t, database, f.problem.ID, "C++", "generated"))
}

func TestLLMGenerator_ExtractsFencedCode(t *testing.T) {
	var gotAuth, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
			"message": map[string]any{"content": "Here:\n```cpp\nint main(){}\n```\nDone"},
		}}})
	}))
	defer srv.Close()
	t.Setenv("COMPARE50_LLM_URL", srv.URL+"/v1/")
	t.Setenv("COMPARE50_LLM_KEY", "secret-key")
	t.Setenv("COMPARE50_LLM_MODEL", "test-model")

	gen := LLMGeneratorFromEnv()
	require.NotNil(t, gen)
	require.Equal(t, "test-model", gen.Model())

	code, err := gen.Generate(context.Background(), GenerateInput{
		Statement: longStatement(), Language: "C++", TimeLimit: 1, MemoryLimitKB: 262144,
	})
	require.NoError(t, err)
	require.Equal(t, "int main(){}", code)
	require.Equal(t, "/v1/chat/completions", gotPath)
	require.Equal(t, "Bearer secret-key", gotAuth)
	require.Equal(t, "test-model", gotBody["model"])
}

func TestLLMGenerator_RetriesOn429(t *testing.T) {
	t.Setenv("COMPARE50_LLM_RETRY_WAITS", "0.01")
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if hits.Add(1) == 1 {
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"int main(){}"}}]}`))
	}))
	defer srv.Close()
	t.Setenv("COMPARE50_LLM_URL", srv.URL)
	t.Setenv("COMPARE50_LLM_KEY", "k")
	t.Setenv("COMPARE50_LLM_MODEL", "m")

	code, err := LLMGeneratorFromEnv().Generate(context.Background(), GenerateInput{Statement: longStatement()})
	require.NoError(t, err)
	require.Equal(t, "int main(){}", code)
	require.Equal(t, int32(2), hits.Load())
}

// slowGen answers the first `fast` calls at once and blocks the rest until ctx ends.
type slowGen struct {
	calls atomic.Int32
	fast  int32
}

func (s *slowGen) Model() string { return "slow" }
func (s *slowGen) Generate(ctx context.Context, _ GenerateInput) (string, error) {
	if s.calls.Add(1) <= s.fast {
		return "quick code", nil
	}
	<-ctx.Done()
	return "", ctx.Err()
}

func TestEnsureAllReferences_BudgetAbandonsSlowCalls(t *testing.T) {
	database := setupMossDB(t)
	f := newContest(t, database)
	f.problem.Description = longStatement()
	gen := &slowGen{fast: 3}
	start := time.Now()
	notes, err := EnsureAllReferences(context.Background(), database, gen,
		[]aiGenTarget{{f.problem, "C++"}}, 200*time.Millisecond)
	require.NoError(t, err)
	require.Less(t, time.Since(start), 5*time.Second)
	require.Equal(t, "generation_failed: time budget", notes[[2]string{f.problem.Code, "C++"}])
	require.Equal(t, int64(3), countRefs(t, database, f.problem.ID, "C++", "generated"), "finished refs are kept")
}

func TestEnsureAllReferences_SharedPoolAcrossTargets(t *testing.T) {
	database := setupMossDB(t)
	f := newContest(t, database)
	f.problem.Description = longStatement()
	gen := &fakeGen{}
	notes, err := EnsureAllReferences(context.Background(), database, gen,
		[]aiGenTarget{{f.problem, "C++"}, {f.problem, "Python"}}, time.Minute)
	require.NoError(t, err)
	require.Empty(t, notes)
	require.Equal(t, int32(2*aiRefsPerLang), gen.calls.Load())
	require.Equal(t, int64(aiRefsPerLang), countRefs(t, database, f.problem.ID, "Python", "generated"))
}
