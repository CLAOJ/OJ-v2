package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LLMReview asks an OpenAI-compatible chat model for a second opinion on the
// top compare50 pairs of one report and writes the answers to <report>/llm.json.
// Only the two sources are sent, labelled A and B; usernames never leave the server.
type LLMReview struct {
	BaseURL  string // e.g. https://dashscope-intl.aliyuncs.com/compatible-mode/v1
	APIKey   string
	Model    string
	MaxPairs int     // per report
	MinScore float64 // compare50 relative score (0-10) a pair needs to be reviewed
}

const (
	llmDefaultPairs    = 5
	llmDefaultMinScore = 5
	llmSourceLimit     = 12000 // chars per source sent to the model
	llmConcurrency     = 4
	llmCallTimeout     = 90 * time.Second
)

// PlagiarismLLM is nil (off) unless PLAGIARISM_LLM_BASE_URL, _API_KEY and _MODEL are set.
var PlagiarismLLM = LLMReviewFromEnv()

func LLMReviewFromEnv() *LLMReview {
	r := &LLMReview{
		BaseURL:  os.Getenv("PLAGIARISM_LLM_BASE_URL"),
		APIKey:   os.Getenv("PLAGIARISM_LLM_API_KEY"),
		Model:    os.Getenv("PLAGIARISM_LLM_MODEL"),
		MaxPairs: llmDefaultPairs,
		MinScore: llmDefaultMinScore,
	}
	if r.BaseURL == "" || r.APIKey == "" || r.Model == "" {
		return nil
	}
	if n, err := strconv.Atoi(os.Getenv("PLAGIARISM_LLM_MAX_PAIRS")); err == nil && n > 0 {
		r.MaxPairs = n
	}
	return r
}

type llmPair struct {
	Match      int      `json:"match"` // match_<n>.html in the report
	A          string   `json:"a"`
	B          string   `json:"b"`
	Score      float64  `json:"score"`
	Verdict    string   `json:"verdict"` // likely_copied | shared_template | independent | unclear | error
	Confidence float64  `json:"confidence"`
	Reasons    []string `json:"reasons"`
	Error      string   `json:"error,omitempty"`
}

type llmReport struct {
	Model       string    `json:"model"`
	GeneratedAt time.Time `json:"generated_at"`
	Pairs       []llmPair `json:"pairs"`
}

const graphPrefix = "var GRAPH = "

// Review reviews the top pairs in reportDir. It only fails when it cannot read
// the report itself; a failed model call is recorded per pair as "error".
func (r *LLMReview) Review(ctx context.Context, problemCode, language, subsDir, reportDir string) error {
	html, err := os.ReadFile(filepath.Join(reportDir, "index.html"))
	if err != nil {
		return err
	}
	i := bytes.Index(html, []byte(graphPrefix))
	if i < 0 {
		return fmt.Errorf("llm review: no GRAPH in %s", reportDir)
	}
	var graph struct {
		Links []struct {
			Index  int     `json:"index"`
			Source string  `json:"source"`
			Target string  `json:"target"`
			Value  float64 `json:"value"`
		} `json:"links"`
	}
	if err := json.NewDecoder(bytes.NewReader(html[i+len(graphPrefix):])).Decode(&graph); err != nil {
		return fmt.Errorf("llm review: %w", err)
	}
	sort.SliceStable(graph.Links, func(i, j int) bool { return graph.Links[i].Value > graph.Links[j].Value })

	var pairs []llmPair
	for _, l := range graph.Links {
		if len(pairs) == r.MaxPairs || l.Value < r.MinScore {
			break
		}
		pairs = append(pairs, llmPair{Match: l.Index + 1, A: l.Source, B: l.Target, Score: l.Value})
	}

	sem := make(chan struct{}, llmConcurrency)
	var wg sync.WaitGroup
	for i := range pairs {
		wg.Add(1)
		go func(p *llmPair) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := r.reviewPair(ctx, problemCode, language, subsDir, p); err != nil {
				p.Verdict, p.Error = "error", err.Error()
				log.Printf("jobs: llm review %s/%s match %d: %v", problemCode, language, p.Match, err)
			}
		}(&pairs[i])
	}
	wg.Wait()

	out, err := json.MarshalIndent(llmReport{Model: r.Model, GeneratedAt: time.Now(), Pairs: pairs}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(reportDir, "llm.json"), out, 0o644)
}

const llmSystemPrompt = `You help programming-contest judges review plagiarism reports.
You get two solutions (A and B) to the same problem, flagged as similar by an automatic checker.
Decide whether one was likely copied from the other (or both from a shared source).
Standard algorithms, common templates (fast I/O, #define shortcuts, typical DP loops) and
short solutions are often similar by nature: they are not evidence on their own.
Evidence of copying: same unusual structure or order, identical quirks, dead code or comments,
consistent identifier renaming, same bugs.
Answer with JSON only: {"verdict": "likely_copied" | "shared_template" | "independent" | "unclear",
"confidence": 0.0-1.0, "reasons": [at most 3 short sentences citing concrete code]}`

func (r *LLMReview) reviewPair(ctx context.Context, problemCode, language, subsDir string, p *llmPair) error {
	a, err := readSubmission(subsDir, p.A)
	if err != nil {
		return err
	}
	b, err := readSubmission(subsDir, p.B)
	if err != nil {
		return err
	}
	user := fmt.Sprintf("Problem: %s\nLanguage: %s\nChecker similarity: %.1f/10 (relative to the most similar pair in this problem)\n\n--- Solution A ---\n%s\n\n--- Solution B ---\n%s",
		problemCode, language, p.Score, a, b)

	body, _ := json.Marshal(map[string]any{
		"model":       r.Model,
		"temperature": 0,
		"messages": []map[string]string{
			{"role": "system", "content": llmSystemPrompt},
			{"role": "user", "content": user},
		},
	})
	ctx, cancel := context.WithTimeout(ctx, llmCallTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(r.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %.200s", resp.StatusCode, raw)
	}
	var chat struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &chat); err != nil || len(chat.Choices) == 0 {
		return fmt.Errorf("unexpected response: %.200s", raw)
	}
	content := chat.Choices[0].Message.Content
	// Models often wrap JSON in a ```json fence; take the outermost object.
	if i, j := strings.IndexByte(content, '{'), strings.LastIndexByte(content, '}'); i >= 0 && j > i {
		content = content[i : j+1]
	}
	var ans struct {
		Verdict    string   `json:"verdict"`
		Confidence float64  `json:"confidence"`
		Reasons    []string `json:"reasons"`
	}
	if err := json.Unmarshal([]byte(content), &ans); err != nil {
		return fmt.Errorf("model did not answer JSON: %.200s", content)
	}
	switch ans.Verdict {
	case "likely_copied", "shared_template", "independent", "unclear":
	default:
		ans.Verdict = "unclear"
	}
	if len(ans.Reasons) > 3 {
		ans.Reasons = ans.Reasons[:3]
	}
	p.Verdict, p.Confidence, p.Reasons = ans.Verdict, ans.Confidence, ans.Reasons
	return nil
}

// readSubmission returns the (truncated) source in subsDir/<folder>/main.*.
func readSubmission(subsDir, folder string) (string, error) {
	if folder == "" || strings.ContainsAny(folder, `/\`) || folder == "." || folder == ".." {
		return "", fmt.Errorf("bad submission folder %q", folder)
	}
	files, _ := filepath.Glob(filepath.Join(subsDir, folder, "main.*"))
	if len(files) == 0 {
		return "", fmt.Errorf("no source for %q", folder)
	}
	src, err := os.ReadFile(files[0])
	if err != nil {
		return "", err
	}
	if len(src) > llmSourceLimit {
		src = append(src[:llmSourceLimit], "\n... (truncated)"...)
	}
	return string(src), nil
}
