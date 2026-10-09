package db

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestContainsPattern_EscapesWildcards(t *testing.T) {
	require.Equal(t, "%abc%", ContainsPattern("abc"))
	require.Equal(t, "%a!_b%", ContainsPattern("a_b"))
	require.Equal(t, "%100!%%", ContainsPattern("100%"))
	require.Equal(t, "%x!!y%", ContainsPattern("x!y"))

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.Exec("CREATE TABLE t (code TEXT)").Error)
	require.NoError(t, database.Exec("INSERT INTO t (code) VALUES ('a_b'), ('axb'), ('100%'), ('1000')").Error)

	var got []string
	require.NoError(t, database.Raw("SELECT code FROM t WHERE code LIKE ? ESCAPE '!' ORDER BY code", ContainsPattern("a_b")).Scan(&got).Error)
	require.Equal(t, []string{"a_b"}, got)

	got = nil
	require.NoError(t, database.Raw("SELECT code FROM t WHERE code LIKE ? ESCAPE '!' ORDER BY code", ContainsPattern("100%")).Scan(&got).Error)
	require.Equal(t, []string{"100%"}, got)
}
