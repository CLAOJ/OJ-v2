package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_JudgeName(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-only-secret") // Load() log.Fatal()s on a missing/default key
	t.Setenv("JUDGE_NAME", "Vịt Toàn Năng")
	Load()
	assert.Equal(t, "Vịt Toàn Năng", C.Judge.Name)
}
