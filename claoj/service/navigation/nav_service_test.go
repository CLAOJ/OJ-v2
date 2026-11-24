package navigation

import (
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
