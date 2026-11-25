package jobs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Checker writes one HTML report for the submission folders under subsDir into outDir.
type Checker interface {
	Check(ctx context.Context, subsDir, outDir string) error
}

var PlagiarismDir = filepath.Join("data", "plagiarism") // swapped in tests

var Compare50Python = "/opt/compare50/bin/python" // swapped in tests

const (
	compare50DefaultPairs = 100 // one match page each; 250 gave ~1100 pages / 760 MB for tet26
	compare50StderrTail   = 2048
)

// Compare50 runs the local compare50 fork (python -m compare50).
type Compare50 struct{ MaxPairs int }

func (c Compare50) Check(ctx context.Context, subsDir, outDir string) error {
	// compare50 creates outDir itself but fails when its parent is missing.
	if err := os.MkdirAll(filepath.Dir(outDir), 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(subsDir)
	if err != nil {
		return err
	}
	maxPairs := c.MaxPairs
	if maxPairs == 0 {
		maxPairs = compare50DefaultPairs
	}
	// The cwd changes to subsDir so report labels are the bare "./<user>"; the
	// "./" keeps usernames starting with "-" from being parsed as flags.
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}
	args := []string{"-m", "compare50"}
	for _, e := range entries {
		if e.IsDir() {
			args = append(args, "./"+e.Name())
		}
	}
	// AI review settings (COMPARE50_LLM_*, incl. _LANGUAGE) reach compare50 through the inherited env.
	args = append(args, "-o", absOut, "-n", strconv.Itoa(maxPairs))

	// ponytail: buffers all of stderr; fine at compare50's output size.
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, Compare50Python, args...)
	cmd.Dir = subsDir
	cmd.WaitDelay = 10 * time.Second
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		tail := stderr.Bytes()
		if len(tail) > compare50StderrTail {
			tail = tail[len(tail)-compare50StderrTail:]
		}
		if ctx.Err() != nil {
			err = fmt.Errorf("%w: %v", ctx.Err(), err)
		}
		return fmt.Errorf("compare50: %w: %s", err, strings.TrimSpace(string(tail)))
	}
	return nil
}

var languageExts = map[string]string{
	"C": "c", "C++": "cpp", "Python": "py", "Pascal": "pas", "Java": "java",
	"Golang": "go", "Rust": "rs", "Perl": "pl", "Ruby": "rb", "Kotlin": "kt",
	"C#": "cs", "PHP": "php", "JavaScript": "js", "Haskell": "hs", "Scala": "scala",
	"Swift": "swift", "Lua": "lua", "D": "d",
}

// LanguageExt maps Language.common_name to the source file extension.
func LanguageExt(commonName string) string {
	if ext, ok := languageExts[commonName]; ok {
		return ext
	}
	return "txt"
}

// LanguageSlug turns Language.common_name into a path-safe folder name.
func LanguageSlug(commonName string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(commonName) {
		switch {
		case r == '+':
			b.WriteByte('p')
		case r == '#':
			b.WriteString("sharp")
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// ContestReportDir is where a contest's current compare50 reports live.
func ContestReportDir(contestID uint) string {
	return filepath.Join(PlagiarismDir, strconv.FormatUint(uint64(contestID), 10))
}

// SubmissionFolder returns the folder name for a username. Names that Windows
// cannot store (".", "..", leading or trailing dot, reserved device names) are
// wrapped in "~", which Django usernames never contain, so names stay distinct.
func SubmissionFolder(username string) string {
	stem := username
	if i := strings.IndexByte(username, '.'); i >= 0 {
		stem = username[:i]
	}
	if strings.HasPrefix(username, ".") || strings.HasSuffix(username, ".") || isWindowsDeviceName(stem) {
		return "~" + username + "~"
	}
	return username
}

func isWindowsDeviceName(stem string) bool {
	s := strings.ToLower(stem)
	switch s {
	case "con", "prn", "aux", "nul":
		return true
	}
	return len(s) == 4 && (strings.HasPrefix(s, "com") || strings.HasPrefix(s, "lpt")) && s[3] >= '1' && s[3] <= '9'
}
