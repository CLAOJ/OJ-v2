package jobs

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The fake interpreter is a .bat on Windows and a sh script elsewhere; either
// records its cwd on line 1 and its argv on line 2 into $COMPARE50_REC.
func TestCompare50Check_RelativeLabelsAndAbsoluteOut(t *testing.T) {
	dir := t.TempDir()
	var script, body string
	if runtime.GOOS == "windows" {
		script = filepath.Join(dir, "fakepy.bat")
		body = "@echo off\r\ncd > \"%COMPARE50_REC%\"\r\necho %* >> \"%COMPARE50_REC%\"\r\n"
	} else {
		script = filepath.Join(dir, "fakepy.sh")
		body = "#!/bin/sh\npwd > \"$COMPARE50_REC\"\necho \"$@\" >> \"$COMPARE50_REC\"\n"
	}
	require.NoError(t, os.WriteFile(script, []byte(body), 0o755))
	rec := filepath.Join(dir, "rec.txt")
	t.Setenv("COMPARE50_REC", rec)
	old := Compare50Python
	Compare50Python = script
	t.Cleanup(func() { Compare50Python = old })

	subsDir := filepath.Join(dir, "subs")
	for _, n := range []string{"a", "-b"} {
		require.NoError(t, os.MkdirAll(filepath.Join(subsDir, n), 0o755))
	}
	t.Chdir(dir)

	require.NoError(t, Compare50{MaxPairs: 7}.Check(context.Background(), subsDir, filepath.Join("rel", "out")))

	b, err := os.ReadFile(rec)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(b), "\r\n", "\n")), "\n")
	require.Len(t, lines, 2)
	gotCwd, err := filepath.EvalSymlinks(lines[0])
	require.NoError(t, err)
	wantCwd, err := filepath.EvalSymlinks(subsDir)
	require.NoError(t, err)
	require.Equal(t, wantCwd, gotCwd)

	wantOut, err := filepath.Abs(filepath.Join("rel", "out"))
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(wantOut))
	// os.ReadDir sorts: "-b" before "a".
	require.Equal(t, "-m compare50 ./-b ./a -o "+wantOut+" -n 7", strings.TrimSpace(lines[1]))
}
