package v2

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"gopkg.in/yaml.v3"
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
	root := config.C.App.ProblemDataRoot
	if root == "" || problem.Code == "" || filepath.Base(problem.Code) != problem.Code {
		return nil
	}
	dir := filepath.Join(root, problem.Code)
	initYML, err := os.ReadFile(filepath.Join(dir, "init.yml"))
	if err != nil {
		return nil
	}
	var init struct {
		Archive string `yaml:"archive"`
	}
	if yaml.Unmarshal(initYML, &init) != nil || init.Archive == "" {
		return nil
	}
	archivePath := filepath.Clean(filepath.Join(dir, init.Archive))
	if !strings.HasPrefix(archivePath, filepath.Clean(dir)+string(os.PathSeparator)) {
		return nil
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil
	}
	defer archive.Close()
	files := make(map[string]*zip.File, len(archive.File))
	for _, f := range archive.File {
		files[f.Name] = f
	}

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
		in, err1 := visibleContent(files[tc.InputFile])
		ans, err2 := visibleContent(files[tc.OutputFile])
		if err1 != nil || err2 != nil {
			return nil
		}
		out[order] = caseData{in, ans}
	}
	return out
}

// visibleContent mirrors v1 get_visible_content.
func visibleContent(f *zip.File) (string, error) {
	if f == nil {
		return "", os.ErrNotExist
	}
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, testcaseVisibleBytes))
	if err != nil {
		return "", err
	}
	s := strings.ToValidUTF8(string(data), "")
	if f.UncompressedSize64 > testcaseVisibleBytes {
		s += "..."
	}
	return s, nil
}

// truncatePrefix applies a contest's output_prefix_override to shown data.
func truncatePrefix(s string, prefix *int) string {
	if prefix == nil || *prefix <= 0 || len(s) <= *prefix {
		return s
	}
	return strings.ToValidUTF8(s[:*prefix], "")
}
