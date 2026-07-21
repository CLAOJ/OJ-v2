package v2

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveStatementPDFPath(t *testing.T) {
	t.Run("v2-native bare filename resolves under the problem data dir", func(t *testing.T) {
		got, err := resolveStatementPDFPath("statement.pdf", "abc", "/v1media")
		require.NoError(t, err)
		assert.Equal(t, filepath.Clean(filepath.Join("data", "problems", "abc", "statement.pdf")), got)
	})

	t.Run("v1-migrated media path resolves under the configured media root", func(t *testing.T) {
		got, err := resolveStatementPDFPath("/pdf/e19aa92f.pdf", "abc", "/v1media")
		require.NoError(t, err)
		assert.Equal(t, filepath.Clean(filepath.Join("/v1media", "pdf", "e19aa92f.pdf")), got)
	})

	t.Run("v1-migrated media path without a configured root is unavailable", func(t *testing.T) {
		_, err := resolveStatementPDFPath("/pdf/e19aa92f.pdf", "abc", "")
		assert.True(t, errors.Is(err, errPDFMediaUnavailable))
	})

	t.Run("path traversal in a v1 media path is rejected", func(t *testing.T) {
		_, err := resolveStatementPDFPath("/pdf/../../etc/passwd", "abc", "/v1media")
		assert.True(t, errors.Is(err, errPDFInvalidPath))
	})

	t.Run("path traversal in a v2-native filename is rejected", func(t *testing.T) {
		_, err := resolveStatementPDFPath("../../../etc/passwd", "abc", "/v1media")
		assert.True(t, errors.Is(err, errPDFInvalidPath))
	})
}
