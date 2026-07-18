package db

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDDLGuardBlocksDDLAllowsDML(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// table created BEFORE the guard is installed
	require.NoError(t, database.Exec("CREATE TABLE t (id INTEGER)").Error)
	RegisterDDLGuard(database)
	require.NoError(t, database.Exec("INSERT INTO t (id) VALUES (1)").Error)
	require.NoError(t, database.Exec("SELECT * FROM t").Error)
	require.Panics(t, func() { database.Exec("ALTER TABLE t ADD COLUMN x INTEGER") })
	require.Panics(t, func() { database.Exec("CREATE TABLE u (id INTEGER)") })
	require.Panics(t, func() { database.Exec("DROP TABLE t") })
}
