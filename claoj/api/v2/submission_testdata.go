package v2

import (
	"strings"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/problemdata"
)

// v1's CLAOJ_TESTCASE_VISIBLE_LENGTH: bytes of each input/answer shown.
const testcaseVisibleBytes = 64

type caseData struct {
	Input  string
	Answer string
}

// problemCaseData mirrors v1 get_problem_testcases_data: the first
// testcaseVisibleBytes of each case's input and answer (with "..." when cut),
// keyed by case number, i.e. the 1-based order among cases that have an input
// file. As in v1, any problem with the data (no root configured, no init.yml,
// no archive, unreadable file) yields nil and the page simply shows no data.
func problemCaseData(problem *models.Problem) map[int]caseData {
	archive, err := problemdata.Open(problem.Code)
	if err != nil {
		return nil
	}
	defer archive.Close()

	var cases []models.ProblemTestCase
	if db.DB.Where("dataset_id = ?", problem.ID).Order("`order` ASC").Find(&cases).Error != nil {
		return nil
	}
	out := map[int]caseData{}
	order := 0
	for _, tc := range cases {
		if tc.InputFile == "" {
			continue
		}
		order++
		in, err1 := visibleContent(archive, tc.InputFile)
		ans, err2 := visibleContent(archive, tc.OutputFile)
		if err1 != nil || err2 != nil {
			return nil
		}
		out[order] = caseData{in, ans}
	}
	return out
}

// visibleContent mirrors v1 get_visible_content.
func visibleContent(a *problemdata.Archive, name string) (string, error) {
	s, cut, err := a.Read(name, testcaseVisibleBytes)
	if cut {
		s += "..."
	}
	return s, err
}

// truncatePrefix applies a contest's output_prefix_override to shown data.
func truncatePrefix(s string, prefix *int) string {
	if prefix == nil || *prefix <= 0 || len(s) <= *prefix {
		return s
	}
	return strings.ToValidUTF8(s[:*prefix], "")
}
