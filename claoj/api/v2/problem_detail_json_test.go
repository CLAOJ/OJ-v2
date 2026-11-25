package v2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The problem detail payload is consumed by claoj-web, whose ProblemDetail type
// (claoj-web/src/types/index.ts) declares these nested objects with lowercase
// keys:
//
//	languages: { key: string; name: string }[]
//	types:     { name: string }[]
//	authors:   { username: string }[]
//
// These DTOs previously had no json tags, so encoding/json emitted the Go field
// names verbatim ("Key", "Name", "Username"). The frontend read l.key/l.name and
// got undefined for every entry, which is why the submit-language <select>
// rendered a list of blank options.
func TestProblemDetailDTOsMarshalWithLowercaseKeys(t *testing.T) {
	t.Run("language items expose key, name and submission mode", func(t *testing.T) {
		b, err := json.Marshal(problemLangItem{Key: "CPP17", Name: "C++17", Extension: "cpp"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"key":"CPP17","name":"C++17","extension":"cpp","file_only":false,"file_size_limit":0}`, string(b))
	})

	t.Run("file-only language items carry the limits the submit UI needs", func(t *testing.T) {
		b, err := json.Marshal(problemLangItem{Key: "ZIP", Name: "ZIP", Extension: "zip", FileOnly: true, FileSizeLimit: 20})
		require.NoError(t, err)
		assert.JSONEq(t, `{"key":"ZIP","name":"ZIP","extension":"zip","file_only":true,"file_size_limit":20}`, string(b))
	})

	t.Run("type items expose name", func(t *testing.T) {
		b, err := json.Marshal(problemTypeItem{Name: "Graph Theory"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"name":"Graph Theory"}`, string(b))
	})

	t.Run("author items expose username", func(t *testing.T) {
		b, err := json.Marshal(problemAuthorItem{Username: "alice"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"username":"alice"}`, string(b))
	})

	t.Run("a language slice marshals as an array of lowercase-keyed objects", func(t *testing.T) {
		b, err := json.Marshal([]problemLangItem{
			{Key: "PY3", Name: "Python 3", Extension: "py"},
			{Key: "C", Name: "C", Extension: "c"},
		})
		require.NoError(t, err)
		assert.JSONEq(t, `[
			{"key":"PY3","name":"Python 3","extension":"py","file_only":false,"file_size_limit":0},
			{"key":"C","name":"C","extension":"c","file_only":false,"file_size_limit":0}
		]`, string(b))
	})
}
