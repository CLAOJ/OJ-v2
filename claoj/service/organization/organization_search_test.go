package organization

import (
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestListOrganizations_Search(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := database.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.AuthUser{}, &models.Profile{}, &models.Organization{}))
	db.DB = database
	for _, o := range []models.Organization{
		{Name: "CLB ABA", Slug: "clb-aba", ShortName: "ABA"},
		{Name: "Long An HSGS", Slug: "la-hsgs", ShortName: "HSGS"},
	} {
		o := o
		require.NoError(t, database.Omit(clause.Associations).Create(&o).Error)
	}

	resp, err := NewOrganizationService().ListOrganizations(ListOrganizationsRequest{Page: 1, PageSize: 20, Search: "hsgs"})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.Total)
	require.Equal(t, "Long An HSGS", resp.Organizations[0].Name)

	resp, _ = NewOrganizationService().ListOrganizations(ListOrganizationsRequest{Page: 1, PageSize: 20})
	require.Equal(t, int64(2), resp.Total)
}
