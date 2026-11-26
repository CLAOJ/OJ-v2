package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/models"
	"gorm.io/gorm"
)

// Thresholds for AI submission detection (plagiarism check, Tasks 4-6).
const (
	aiRefsPerLang        = 5   // generated reference solutions kept per problem x language
	aiStatementMinChars  = 300 // shorter statements are too thin to generate references from
	aiMatchPct           = 50  // percent of a submission's lines matched to a reference to flag it
	aiTrivialShare       = 0.5 // share of trivial lines; above this a match does not count
	rewriteMaxSimilarity = 0.3 // max similarity to the original for a rewrite to count
	rewriteMinLines      = 5   // min lines a rewritten solution needs to be judged
	rewritesToFlag       = 2   // rewrites needed before a submission is flagged
	historyMargin        = 1.5 // margin over the contestant's own history
)

type AIReport struct {
	GeneratedAt time.Time   `json:"generated_at"`
	Rows        []AIRow     `json:"rows"`
	Problems    []AIProblem `json:"problems"`
}

type AIRow struct {
	Username        string   `json:"username"`
	ProblemCode     string   `json:"problem_code"`
	Language        string   `json:"language"`
	BestSubmission  uint     `json:"best_submission_id"`
	FirstSubmission uint     `json:"first_submission_id"`
	Match           *AIMatch `json:"ai_match"` // nil when no reference pair
	Rewrites        int      `json:"rewrites"`
	RewriteSubs     []uint   `json:"rewrite_submission_ids"` // the later attempt of each rewrite
	AboveHistory    *bool    `json:"above_history"`
	PrevMaxPoints   *float64 `json:"prev_max_points"`
	RatingBefore    *int     `json:"rating_before"`
	SpeedInversion  bool     `json:"speed_inversion"`
	Signals         int      `json:"signals"`
}

type AIMatch struct {
	Pct        int    `json:"pct"`
	ReportPath string `json:"report_path"` // "<problem>/<slug>/ai"
	MatchPage  int    `json:"match"`       // match_<n>.html
}

type AIProblem struct {
	ProblemCode string `json:"problem_code"`
	Language    string `json:"language"`
	Generated   int    `json:"generated"`
	Admin       int    `json:"admin"`
	Contestants int    `json:"contestants"`
	Matched     int    `json:"matched"` // contestants with Match.Pct >= aiMatchPct
	Trivial     bool   `json:"trivial"` // Contestants > 0 && Matched/Contestants >= aiTrivialShare
	Note        string `json:"note"`    // from EnsureReferences, or "no_references"
}

// aiAttempt is one contest submission of a participant.
type aiAttempt struct {
	ID       uint
	Date     time.Time
	Username string
	Source   string
}

// aiLang is one problem x language handed to the AI check.
type aiLang struct {
	Problem     models.Problem
	Lang        string
	Slug        string
	Best        []aiAttempt            // one per user: the best-scored attempt
	Attempts    map[string][]aiAttempt // username -> all attempts in (date, id) order
	Rewrites    map[string]int         // username -> rewrites summed over all language chains
	RewriteSubs map[string][]uint      // username -> later attempt of each rewrite, all languages
}

// aiUser holds, per participant, what S4 needs: the participation start and
// the first contest AC per problem.
type aiUser struct {
	start  time.Time
	solved map[uint]SolvedProblem
}

// aiRewrites counts rewrites in one language chain and returns the later
// attempt's id of each.
func aiRewrites(chain []aiAttempt) (n int, ids []uint) {
	for i := 1; i < len(chain); i++ {
		if CountRewrites([]string{chain[i-1].Source, chain[i].Source}) == 1 {
			n++
			ids = append(ids, chain[i].ID)
		}
	}
	return n, ids
}

// aiCheck runs the reference check for one problem x language and returns its
// summary plus one candidate row per contestant (before S4 and listing).
func aiCheck(ctx context.Context, tx *gorm.DB, chk Checker, gen Generator, tmp, reportRoot string, l aiLang) (AIProblem, []AIRow, error) {
	note, err := EnsureReferences(ctx, tx, gen, l.Problem, l.Lang)
	if err != nil {
		return AIProblem{}, nil, err
	}
	var refs []models.PlagiarismAIReference
	if err := tx.Where("problem_id = ? AND language = ?", l.Problem.ID, l.Lang).Order("id ASC").Find(&refs).Error; err != nil {
		return AIProblem{}, nil, err
	}
	prob := AIProblem{ProblemCode: l.Problem.Code, Language: l.Lang, Contestants: len(l.Best), Note: note}
	for _, r := range refs {
		if r.Origin == "admin" {
			prob.Admin++
		} else {
			prob.Generated++
		}
	}
	if len(refs) == 0 && prob.Note == "" {
		prob.Note = "no_references"
	}

	rows := make([]*AIRow, 0, len(l.Best))
	byUser := map[string]*AIRow{}
	for _, b := range l.Best {
		chain := l.Attempts[b.Username]
		row := &AIRow{Username: b.Username, ProblemCode: l.Problem.Code, Language: l.Lang, BestSubmission: b.ID, FirstSubmission: chain[0].ID}
		row.Rewrites, row.RewriteSubs = l.Rewrites[b.Username], l.RewriteSubs[b.Username]
		rows = append(rows, row)
		byUser[b.Username] = row
	}

	if len(refs) > 0 && len(l.Best) > 0 {
		base := filepath.Join(tmp, "ai", l.Problem.Code, l.Slug)
		subsDir, refsDir := filepath.Join(base, "subs"), filepath.Join(base, "refs")
		ext := LanguageExt(l.Lang)
		write := func(dir, src string) error {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(dir, "main."+ext), []byte(src), 0o644)
		}
		folderUser := map[string]string{} // folder name -> username
		for _, b := range l.Best {
			f := SubmissionFolder(b.Username)
			folderUser[f] = b.Username
			if err := write(filepath.Join(subsDir, f), b.Source); err != nil {
				return prob, nil, err
			}
			if first := l.Attempts[b.Username][0]; first.ID != b.ID {
				folderUser[f+"~first"] = b.Username
				if err := write(filepath.Join(subsDir, f+"~first"), first.Source); err != nil {
					return prob, nil, err
				}
			}
		}
		for _, r := range refs {
			if err := write(filepath.Join(refsDir, fmt.Sprintf("ref_%d", r.ID)), r.Source); err != nil {
				return prob, nil, err
			}
		}
		absRefs, err := filepath.Abs(refsDir)
		if err != nil {
			return prob, nil, err
		}
		outDir := filepath.Join(reportRoot, l.Problem.Code, l.Slug, "ai")
		if err := chk.Check(ctx, subsDir, absRefs, outDir); err != nil {
			return prob, nil, fmt.Errorf("ai check %s/%s: %w", l.Problem.Code, l.Lang, err)
		}
		// compare50 exits 0 without creating outDir when there are no pairs.
		b, err := os.ReadFile(filepath.Join(outDir, "results.json"))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return prob, nil, fmt.Errorf("ai check %s/%s: %w", l.Problem.Code, l.Lang, err)
		}
		if err == nil {
			var res struct {
				Pairs []struct {
					Match    int    `json:"match"`
					A        string `json:"a"`
					B        string `json:"b"`
					ArchiveA bool   `json:"archive_a"`
					ArchiveB bool   `json:"archive_b"`
					PctA     int    `json:"pct_a"`
					PctB     int    `json:"pct_b"`
				} `json:"pairs"`
			}
			if err := json.Unmarshal(b, &res); err != nil {
				return prob, nil, fmt.Errorf("ai check %s/%s: bad results.json: %w", l.Problem.Code, l.Lang, err)
			}
			for _, p := range res.Pairs {
				if p.ArchiveA == p.ArchiveB {
					continue
				}
				path, pct := p.A, p.PctA
				if p.ArchiveA {
					path, pct = p.B, p.PctB
				}
				path = strings.TrimRight(filepath.ToSlash(path), "/")
				user, ok := folderUser[path[strings.LastIndex(path, "/")+1:]]
				if !ok {
					continue
				}
				if row := byUser[user]; row.Match == nil || pct > row.Match.Pct {
					row.Match = &AIMatch{Pct: pct, ReportPath: l.Problem.Code + "/" + l.Slug + "/ai", MatchPage: p.Match}
				}
			}
		}
	}

	out := make([]AIRow, 0, len(rows))
	for _, r := range rows {
		if r.Match != nil && r.Match.Pct >= aiMatchPct {
			prob.Matched++
		}
		out = append(out, *r)
	}
	prob.Trivial = prob.Contestants > 0 && float64(prob.Matched)/float64(prob.Contestants) >= aiTrivialShare
	return prob, out, nil
}

// buildAIReport adds the S4 history signals to the candidate rows, applies the
// listing rule (S1 on a non-trivial problem, or S3) and sorts. users maps
// username -> profile id, acs is keyed by profile id; problems maps code ->
// problem.
func buildAIReport(tx *gorm.DB, contest models.Contest, rows []AIRow, probs []AIProblem, users map[string]uint, acs map[uint]*aiUser, problems map[string]models.Problem) (AIReport, error) {
	report := AIReport{GeneratedAt: time.Now(), Rows: []AIRow{}, Problems: probs}
	if report.Problems == nil {
		report.Problems = []AIProblem{}
	}
	ids := make([]uint, 0, len(acs))
	for id := range acs {
		ids = append(ids, id)
	}
	prevMax := map[uint]float64{}
	ratingBefore := map[uint]int{}
	if len(ids) > 0 {
		var pm []struct {
			UserID uint
			Max    float64
		}
		if err := tx.Table("judge_submission s").Select("s.user_id AS user_id, MAX(p.points) AS max").
			Joins("JOIN judge_problem p ON p.id = s.problem_id").
			Where("s.result = 'AC' AND s.date < ? AND s.user_id IN ?", contest.StartTime, ids).
			Group("s.user_id").Scan(&pm).Error; err != nil {
			return report, err
		}
		for _, r := range pm {
			prevMax[r.UserID] = r.Max
		}
		var rs []models.Rating
		if err := tx.Where("user_id IN ? AND last_rated < ?", ids, contest.StartTime).
			Order("last_rated ASC, id ASC").Find(&rs).Error; err != nil {
			return report, err
		}
		for _, r := range rs { // ascending, so the latest wins
			ratingBefore[r.UserID] = r.RatingVal
		}
	}
	inversions := map[uint]map[uint]bool{}
	for id, u := range acs {
		solved := make([]SolvedProblem, 0, len(u.solved))
		for _, s := range u.solved {
			solved = append(solved, s)
		}
		inversions[id] = SpeedInversions(u.start, solved)
	}
	trivial := map[[2]string]bool{}
	for _, p := range probs {
		trivial[[2]string{p.ProblemCode, p.Language}] = p.Trivial
	}

	for _, row := range rows {
		pid := users[row.Username]
		prob := problems[row.ProblemCode]
		if u := acs[pid]; u != nil {
			if _, solved := u.solved[prob.ID]; solved {
				if m, ok := prevMax[pid]; ok {
					row.PrevMaxPoints = &m
					row.AboveHistory = AboveHistory(prob.Points, &m)
				}
				if r, ok := ratingBefore[pid]; ok {
					row.RatingBefore = &r
				}
				row.SpeedInversion = inversions[pid][prob.ID]
			}
		}
		s1 := row.Match != nil && row.Match.Pct >= aiMatchPct
		s3 := row.Rewrites >= rewritesToFlag
		s4 := (row.AboveHistory != nil && *row.AboveHistory) || row.SpeedInversion
		for _, fired := range []bool{s1, s3, s4} {
			if fired {
				row.Signals++
			}
		}
		if (s1 && !trivial[[2]string{row.ProblemCode, row.Language}]) || s3 {
			report.Rows = append(report.Rows, row)
		}
	}
	pct := func(r AIRow) int {
		if r.Match == nil {
			return -1
		}
		return r.Match.Pct
	}
	sort.SliceStable(report.Rows, func(i, j int) bool {
		a, b := report.Rows[i], report.Rows[j]
		if a.Signals != b.Signals {
			return a.Signals > b.Signals
		}
		if pct(a) != pct(b) {
			return pct(a) > pct(b)
		}
		return a.Username < b.Username
	})
	return report, nil
}
