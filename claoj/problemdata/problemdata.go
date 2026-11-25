// Package problemdata reads the judges' problem data (read-only mount at
// config App.ProblemDataRoot): <root>/<code>/init.yml naming a zip archive
// that holds the test files, as in v1.
package problemdata

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/CLAOJ/claoj/config"
	"gopkg.in/yaml.v3"
)

// Archive is an open test-data zip; Files is keyed by the names used in
// judge_problemtestcase.input_file/output_file.
type Archive struct {
	*zip.ReadCloser
	Files map[string]*zip.File
}

// Open opens the archive of problem code. The caller closes it.
func Open(code string) (*Archive, error) {
	root := config.C.App.ProblemDataRoot
	if root == "" || code == "" || filepath.Base(code) != code {
		return nil, os.ErrNotExist
	}
	dir := filepath.Join(root, code)
	raw, err := os.ReadFile(filepath.Join(dir, "init.yml"))
	if err != nil {
		return nil, err
	}
	var init struct {
		Archive string `yaml:"archive"`
	}
	if err := yaml.Unmarshal(raw, &init); err != nil {
		return nil, err
	}
	if init.Archive == "" {
		return nil, os.ErrNotExist
	}
	path := filepath.Clean(filepath.Join(dir, init.Archive))
	if !strings.HasPrefix(path, filepath.Clean(dir)+string(os.PathSeparator)) {
		return nil, errors.New("archive outside problem directory")
	}
	rc, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	files := make(map[string]*zip.File, len(rc.File))
	for _, f := range rc.File {
		files[f.Name] = f
	}
	return &Archive{rc, files}, nil
}

// Read returns up to limit bytes of the named file as valid UTF-8, and whether
// the file is longer than that.
func (a *Archive) Read(name string, limit int64) (string, bool, error) {
	f := a.Files[name]
	if f == nil {
		return "", false, os.ErrNotExist
	}
	rc, err := f.Open()
	if err != nil {
		return "", false, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, limit))
	if err != nil {
		return "", false, err
	}
	return strings.ToValidUTF8(string(data), ""), f.UncompressedSize64 > uint64(limit), nil
}
