package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

const (
	aiGenSystemPrompt = "You solve competitive programming problems. Reply with one complete program that reads standard input and writes standard output. Reply with code only, no explanation."
	aiGenWorkers      = 4
	aiGenTimeout      = 90 * time.Second
	aiGenMaxBody      = 1 << 20
)

// aiGenDefaultWaits mirrors compare50's _llm.py RETRY_WAITS default.
var aiGenDefaultWaits = []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second}

type GenerateInput struct {
	Statement     string
	Language      string  // judge_language.common_name
	TimeLimit     float64 // seconds
	MemoryLimitKB int
}

type Generator interface {
	Generate(ctx context.Context, in GenerateInput) (string, error) // code only
	Model() string
}

// LLMGeneratorFromEnv returns nil unless COMPARE50_LLM_URL, _KEY, _MODEL are all set.
func LLMGeneratorFromEnv() Generator {
	url, key, model := os.Getenv("COMPARE50_LLM_URL"), os.Getenv("COMPARE50_LLM_KEY"), os.Getenv("COMPARE50_LLM_MODEL")
	if url == "" || key == "" || model == "" {
		return nil
	}
	return &llmGenerator{url: strings.TrimRight(url, "/") + "/chat/completions", key: key, model: model,
		waits: aiGenRetryWaits(), client: &http.Client{Timeout: aiGenTimeout}}
}

// aiGenRetryWaits reads COMPARE50_LLM_RETRY_WAITS (comma-separated seconds) like the fork.
func aiGenRetryWaits() []time.Duration {
	raw, ok := os.LookupEnv("COMPARE50_LLM_RETRY_WAITS")
	if !ok {
		return aiGenDefaultWaits
	}
	waits := []time.Duration{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if sec, err := strconv.ParseFloat(part, 64); err == nil {
			waits = append(waits, time.Duration(sec*float64(time.Second)))
		}
	}
	return waits
}

type llmGenerator struct {
	url    string
	key    string
	model  string
	waits  []time.Duration
	client *http.Client
}

func (g *llmGenerator) Model() string { return g.model }

func (g *llmGenerator) Generate(ctx context.Context, in GenerateInput) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model":       g.model,
		"temperature": 0.8,
		"messages": []map[string]string{
			{"role": "system", "content": aiGenSystemPrompt},
			{"role": "user", "content": fmt.Sprintf("Language: %s\nTime limit: %g seconds\nMemory limit: %d KB\n\n%s",
				in.Language, in.TimeLimit, in.MemoryLimitKB, in.Statement)},
		},
	})
	if err != nil {
		return "", err
	}
	raw, err := g.post(ctx, body)
	if err != nil {
		return "", err
	}
	var reply struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return "", fmt.Errorf("bad response: %w", err)
	}
	if len(reply.Choices) == 0 {
		return "", errors.New("response has no choices")
	}
	return aiGenExtractCode(reply.Choices[0].Message.Content), nil
}

// post retries 429 and 5xx responses after each wait in g.waits, then gives up.
func (g *llmGenerator) post(ctx context.Context, body []byte) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+g.key)
		resp, err := g.client.Do(req)
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, aiGenMaxBody))
		resp.Body.Close()
		if resp.StatusCode/100 == 2 {
			return data, readErr
		}
		if aiGenRetryable(resp.StatusCode) && attempt < len(g.waits) {
			select {
			case <-time.After(g.waits[attempt]):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, aiGenClip(string(data), 200))
	}
}

// aiGenExtractCode returns the inside of the first ``` fence (info string dropped),
// or the whole reply trimmed when there is no fence.
func aiGenExtractCode(reply string) string {
	open := strings.Index(reply, "```")
	if open < 0 {
		return strings.TrimSpace(reply)
	}
	rest := reply[open+3:]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[nl+1:]
	}
	if end := strings.Index(rest, "```"); end >= 0 {
		rest = rest[:end]
	}
	return strings.TrimSpace(rest)
}

// aiGenerateBudget is the total time all of a run's generation calls may take
// (a var so tests can shrink it). The job's asynq timeout is 30 minutes.
var aiGenerateBudget = 10 * time.Minute

var errAIBudget = errors.New("time budget")

type aiGenTarget struct {
	Problem models.Problem
	Lang    string
}

// EnsureReferences tops up "generated" references for one problem x language.
func EnsureReferences(ctx context.Context, tx *gorm.DB, gen Generator, p models.Problem, lang string) (note string, err error) {
	notes, err := EnsureAllReferences(ctx, tx, gen, []aiGenTarget{{p, lang}}, aiGenerateBudget)
	return notes[[2]string{p.Code, lang}], err
}

// EnsureAllReferences tops up "generated" references to aiRefsPerLang for every
// target through one shared pool of aiGenWorkers, within budget in total. Calls
// not finished in time are abandoned and their target gets a "time budget" note.
// Notes are keyed by {problem code, language}; "" means fine. Only DB failures
// return an error.
func EnsureAllReferences(ctx context.Context, tx *gorm.DB, gen Generator, targets []aiGenTarget, budget time.Duration) (map[[2]string]string, error) {
	notes := map[[2]string]string{}
	type job struct {
		t  int
		in GenerateInput
	}
	var jobs []job
	for i, t := range targets {
		key := [2]string{t.Problem.Code, t.Lang}
		if gen == nil {
			notes[key] = "no_llm"
			continue
		}
		p := t.Problem
		if utf8.RuneCountInString(strings.TrimSpace(p.Description)) < aiStatementMinChars {
			notes[key] = "no_statement"
			continue
		}
		var have int64
		if err := tx.Model(&models.PlagiarismAIReference{}).
			Where("problem_id = ? AND language = ? AND origin = ?", p.ID, t.Lang, "generated").
			Count(&have).Error; err != nil {
			return nil, err
		}
		in := GenerateInput{Statement: p.Description, Language: t.Lang, TimeLimit: p.TimeLimit, MemoryLimitKB: int(p.MemoryLimit)}
		for range int(aiRefsPerLang - have) {
			jobs = append(jobs, job{i, in})
		}
	}
	if len(jobs) == 0 {
		return notes, nil
	}

	bctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	sources := make([]string, len(jobs))
	errs := make([]error, len(jobs))
	next := make(chan int)
	var wg sync.WaitGroup
	for range aiGenWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				if bctx.Err() != nil {
					errs[i] = errAIBudget
					continue
				}
				type res struct {
					src string
					err error
				}
				done := make(chan res, 1)
				go func() { src, err := gen.Generate(bctx, jobs[i].in); done <- res{src, err} }()
				select {
				case r := <-done:
					sources[i], errs[i] = r.src, r.err
					if r.err != nil && bctx.Err() != nil {
						errs[i] = errAIBudget
					}
				case <-bctx.Done():
					errs[i] = errAIBudget // abandoned; the goroutine ends when Generate sees ctx
				}
			}
		}()
	}
	for i := range jobs {
		next <- i
	}
	close(next)
	wg.Wait()

	model := gen.Model()
	now := time.Now()
	firstErr := map[int]error{}
	for i, j := range jobs {
		src, err := sources[i], errs[i]
		if err == nil && strings.TrimSpace(src) == "" {
			err = errors.New("empty reply")
		}
		if err != nil {
			if firstErr[j.t] == nil {
				firstErr[j.t] = err
			}
			continue
		}
		p := targets[j.t].Problem
		ref := models.PlagiarismAIReference{ProblemID: p.ID, Language: targets[j.t].Lang, Source: src, Origin: "generated",
			Model: &model, CreatedAt: now}
		if err := tx.Create(&ref).Error; err != nil {
			return nil, err
		}
	}
	for t, err := range firstErr {
		notes[[2]string{targets[t].Problem.Code, targets[t].Lang}] = "generation_failed: " + aiGenClip(err.Error(), 200)
	}
	return notes, nil
}

// aiGenRetryable lists the statuses compare50 retries: rate limits and overload.
func aiGenRetryable(status int) bool {
	switch status {
	case 429, 500, 502, 503, 504:
		return true
	}
	return false
}

// aiGenClip truncates s to at most n runes.
func aiGenClip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
