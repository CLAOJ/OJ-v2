package navigation

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupNavServiceDB(t *testing.T) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.NavigationBar{}))
	db.DB = database
}

// sqlCapture records the SQL gorm logs, so a test can assert statement order.
type sqlCapture struct{ lines []string }

func (s *sqlCapture) Printf(format string, args ...interface{}) {
	s.lines = append(s.lines, fmt.Sprintf(format, args...))
}

var deleteIDPattern = regexp.MustCompile(`DELETE FROM .*navigationbar.*WHERE.*?= (\d+)`)

// deletedIDs returns the row ids of single-row DELETEs in issue order.
func (s *sqlCapture) deletedIDs() []uint {
	var ids []uint
	for _, line := range s.lines {
		if m := deleteIDPattern.FindStringSubmatch(line); m != nil {
			id, err := strconv.Atoi(m[1])
			if err == nil {
				ids = append(ids, uint(id))
			}
		}
	}
	return ids
}

func setupNavServiceDBWithLogger(t *testing.T, out *sqlCapture) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.New(out, logger.Config{SlowThreshold: time.Second, LogLevel: logger.Info}),
	})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.NavigationBar{}))
	db.DB = database
}

// MySQL enforces the self-FK per row (RESTRICT), so deleting a subtree
// root-first — or in one multi-row statement whose row order is arbitrary —
// fails with error 1451. SQLite only checks at statement end and cannot
// reproduce that, so assert the DELETE order instead: leaves first.
func TestDeleteNavigation_DeletesLeavesFirst(t *testing.T) {
	var out sqlCapture
	setupNavServiceDBWithLogger(t, &out)
	svc := NewNavigationService()

	root, err := svc.CreateNavigation(CreateNavigationRequest{Key: "root", Label: "Root", Path: "/"})
	require.NoError(t, err)
	child, err := svc.CreateNavigation(CreateNavigationRequest{Key: "child", Label: "Child", Path: "/c", ParentID: &root.ID})
	require.NoError(t, err)
	grand, err := svc.CreateNavigation(CreateNavigationRequest{Key: "grand", Label: "Grand", Path: "/c/g", ParentID: &child.ID})
	require.NoError(t, err)

	out.lines = nil
	require.NoError(t, svc.DeleteNavigation(DeleteNavigationRequest{NavID: root.ID}))

	require.Equal(t, []uint{grand.ID, child.ID, root.ID}, out.deletedIDs())

	var remaining int64
	require.NoError(t, db.DB.Model(&models.NavigationBar{}).Count(&remaining).Error)
	require.Zero(t, remaining)
}

// TestRebuildTree_LegacyCycleDoesNotHang covers parent_id rows that already
// form a loop: neither node is a root, so the remnant pass must still give
// both of them a valid nested-set range.
func TestRebuildTree_LegacyCycleDoesNotHang(t *testing.T) {
	setupNavServiceDB(t)
	x := models.NavigationBar{Key: "x", Label: "x", Path: "/x"}
	y := models.NavigationBar{Key: "y", Label: "y", Path: "/y"}
	require.NoError(t, db.DB.Create(&x).Error)
	require.NoError(t, db.DB.Create(&y).Error)
	require.NoError(t, db.DB.Model(&models.NavigationBar{}).Where("id = ?", x.ID).Update("parent_id", y.ID).Error)
	require.NoError(t, db.DB.Model(&models.NavigationBar{}).Where("id = ?", y.ID).Update("parent_id", x.ID).Error)

	require.NoError(t, db.DB.Transaction(rebuildTree))

	var got []models.NavigationBar
	require.NoError(t, db.DB.Find(&got).Error)
	require.Len(t, got, 2)
	for _, n := range got {
		require.Less(t, n.Lft, n.Rght)
		require.Greater(t, n.TreeID, 0)
	}
}

func TestCreateNavigation_AssignsMpttFields(t *testing.T) {
	setupNavServiceDB(t)
	svc := NewNavigationService()

	root, err := svc.CreateNavigation(CreateNavigationRequest{Key: "root", Label: "Root", Path: "/", Regex: "^/$"})
	require.NoError(t, err)
	require.Equal(t, 0, root.Level)
	require.Equal(t, 1, root.Lft)
	require.Equal(t, 2, root.Rght)

	child, err := svc.CreateNavigation(CreateNavigationRequest{Key: "child", Label: "Child", Path: "/c", ParentID: &root.ID})
	require.NoError(t, err)
	require.Equal(t, 1, child.Level)
	require.Equal(t, root.TreeID, child.TreeID)
	require.Greater(t, child.Lft, root.Lft)

	reloaded, err := svc.GetNavigation(GetNavigationRequest{NavID: root.ID})
	require.NoError(t, err)
	require.Equal(t, 4, reloaded.Rght) // root now contains the child
	require.Equal(t, "", reloaded.ParentLabel)
}
