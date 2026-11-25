package v2

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/CLAOJ/claoj/problemdata"
	"github.com/stretchr/testify/require"
)

// v1 get_visible_content: files up to 64 bytes verbatim, longer ones cut to
// 64 bytes plus "..."; invalid UTF-8 from the cut is dropped.
func TestVisibleContent(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	write := func(name, body string) {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(body))
		require.NoError(t, err)
	}
	write("short.in", "1 2\n")
	write("long.out", strings.Repeat("9", 100))
	write("utf8.out", strings.Repeat("a", 63)+"ế") // 3-byte rune straddles byte 64
	require.NoError(t, zw.Close())
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}

	a := &problemdata.Archive{Files: files}
	got, err := visibleContent(a, "short.in")
	require.NoError(t, err)
	require.Equal(t, "1 2\n", got)

	got, err = visibleContent(a, "long.out")
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("9", 64)+"...", got)

	got, err = visibleContent(a, "utf8.out")
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("a", 63)+"...", got)

	_, err = visibleContent(a, "missing.in")
	require.Error(t, err)

	five, zero := 5, 0
	require.Equal(t, "12345", truncatePrefix("1234567", &five))
	require.Equal(t, "1234567", truncatePrefix("1234567", &zero))
	require.Equal(t, "1234567", truncatePrefix("1234567", nil))
}
