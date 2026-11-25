package jobs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeReport fakes a compare50 output dir with the given ranked links.
func writeReport(t *testing.T, dir string, links string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	html := "<html><script>\n            var GRAPH = {\"links\": " + links + ", \"nodes\": []};\n        </script></html>"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte(html), 0o644))
}

func writeSub(t *testing.T, subsDir, user, src string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(subsDir, user), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subsDir, user, "main.cpp"), []byte(src), 0o644))
}

func TestLLMReview_WritesVerdictsWithoutUsernames(t *testing.T) {
	var calls atomic.Int32
	var mu sync.Mutex
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(b))
		mu.Unlock()
		calls.Add(1)
		content := "```json\n{\"verdict\":\"likely_copied\",\"confidence\":0.9,\"reasons\":[\"same structure\",\"renamed variables\"]}\n```"
		resp, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": content}}}})
		w.Write(resp)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	subs, out := filepath.Join(tmp, "subs"), filepath.Join(tmp, "out")
	writeSub(t, subs, "alice_secret", "int main(){return 1;}")
	writeSub(t, subs, "bob_secret", "int main(){return 2;}")
	writeSub(t, subs, "carol_secret", "int main(){return 3;}")
	// Ranked links; the third is below MinScore and must not be sent.
	writeReport(t, out, `[{"index":0,"source":"alice_secret","target":"bob_secret","value":10.0},
		{"index":1,"source":"bob_secret","target":"carol_secret","value":6.0},
		{"index":2,"source":"alice_secret","target":"carol_secret","value":2.0}]`)

	r := &LLMReview{BaseURL: srv.URL + "/v1/", APIKey: "sk-test", Model: "m", MaxPairs: 5, MinScore: 5}
	require.NoError(t, r.Review(context.Background(), "knap", "C++", subs, out))

	require.EqualValues(t, 2, calls.Load())
	for _, b := range bodies {
		require.NotContains(t, b, "secret", "usernames must not reach the LLM")
		require.Contains(t, b, `"model":"m"`)
	}

	var got llmReport
	raw, err := os.ReadFile(filepath.Join(out, "llm.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, "m", got.Model)
	require.Len(t, got.Pairs, 2)
	require.Equal(t, 1, got.Pairs[0].Match)
	require.Equal(t, "alice_secret", got.Pairs[0].A)
	require.Equal(t, "likely_copied", got.Pairs[0].Verdict)
	require.Equal(t, []string{"same structure", "renamed variables"}, got.Pairs[0].Reasons)
	require.Equal(t, 2, got.Pairs[1].Match)
}

func TestLLMReview_BadAnswerIsRecordedNotFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "quota exceeded", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	subs, out := filepath.Join(tmp, "subs"), filepath.Join(tmp, "out")
	writeSub(t, subs, "a", "x")
	writeSub(t, subs, "b", "y")
	writeReport(t, out, `[{"index":0,"source":"a","target":"b","value":10.0}]`)

	r := &LLMReview{BaseURL: srv.URL, APIKey: "k", Model: "m", MaxPairs: 5, MinScore: 5}
	require.NoError(t, r.Review(context.Background(), "p", "C++", subs, out))

	raw, err := os.ReadFile(filepath.Join(out, "llm.json"))
	require.NoError(t, err)
	var got llmReport
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Len(t, got.Pairs, 1)
	require.Equal(t, "error", got.Pairs[0].Verdict)
	require.True(t, strings.Contains(got.Pairs[0].Error, "429"), got.Pairs[0].Error)
}

func TestLLMReviewFromEnv_OffUnlessConfigured(t *testing.T) {
	t.Setenv("PLAGIARISM_LLM_BASE_URL", "")
	t.Setenv("PLAGIARISM_LLM_API_KEY", "k")
	t.Setenv("PLAGIARISM_LLM_MODEL", "m")
	require.Nil(t, LLMReviewFromEnv())

	t.Setenv("PLAGIARISM_LLM_BASE_URL", "https://example.invalid/v1")
	r := LLMReviewFromEnv()
	require.NotNil(t, r)
	require.Equal(t, llmDefaultPairs, r.MaxPairs)
}

func TestRunContestMoss_CallsLLMReviewWhenConfigured(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"verdict\":\"independent\",\"confidence\":0.7,\"reasons\":[\"different approach\"]}"}}]}`))
	}))
	defer srv.Close()

	old := PlagiarismLLM
	PlagiarismLLM = &LLMReview{BaseURL: srv.URL, APIKey: "k", Model: "m", MaxPairs: 5, MinScore: 0}
	t.Cleanup(func() { PlagiarismLLM = old })

	useTempPlagiarismDir(t)
	database := setupMossDB(t)
	c := seed(t, database)
	_, err := RunContestMoss(context.Background(), database, c.ID, &graphChecker{})
	require.NoError(t, err)

	require.Positive(t, calls.Load())
	matches, _ := filepath.Glob(filepath.Join(ContestReportDir(c.ID), "*", "*", "llm.json"))
	require.NotEmpty(t, matches)
}

// graphChecker writes a compare50-like index.html ranking the first two folders.
type graphChecker struct{}

func (graphChecker) Check(_ context.Context, subsDir, outDir string) error {
	entries, err := os.ReadDir(subsDir)
	if err != nil {
		return err
	}
	links := `[{"index":0,"source":"` + entries[0].Name() + `","target":"` + entries[1].Name() + `","value":10.0}]`
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	html := "<script>var GRAPH = {\"links\": " + links + ", \"nodes\": []};</script>"
	return os.WriteFile(filepath.Join(outDir, "index.html"), []byte(html), 0o644)
}
