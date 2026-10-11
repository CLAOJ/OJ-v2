package relations

import (
	"testing"

	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupDB(t *testing.T) (*gorm.DB, models.Problem, []models.ProblemType) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&models.ProblemType{}, &models.Problem{}))

	p := models.Problem{Code: "p1", Name: "P1"}
	require.NoError(t, database.Omit(clause.Associations).Create(&p).Error)
	types := []models.ProblemType{{Name: "dp"}, {Name: "graph"}, {Name: "math"}}
	for i := range types {
		require.NoError(t, database.Create(&types[i]).Error)
	}
	return database, p, types
}

func typeIDs(t *testing.T, database *gorm.DB, p models.Problem) []uint {
	t.Helper()
	var loaded models.Problem
	require.NoError(t, database.Preload("Types").First(&loaded, p.ID).Error)
	ids := []uint{}
	for _, ty := range loaded.Types {
		ids = append(ids, ty.ID)
	}
	return ids
}

func TestSync_FullListReplaces(t *testing.T) {
	database, p, types := setupDB(t)
	full := []uint{types[0].ID, types[1].ID}
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", &full, nil, nil))
	require.ElementsMatch(t, full, typeIDs(t, database, p))

	full = []uint{types[2].ID, types[2].ID} // duplicates are collapsed
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", &full, nil, nil))
	require.Equal(t, []uint{types[2].ID}, typeIDs(t, database, p))
}

func TestSync_EmptyListClears(t *testing.T) {
	database, p, types := setupDB(t)
	full := []uint{types[0].ID}
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", &full, nil, nil))
	empty := []uint{}
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", &empty, nil, nil))
	require.Empty(t, typeIDs(t, database, p))
}

func TestSync_NilFullUsesLegacyAddRemove(t *testing.T) {
	database, p, types := setupDB(t)
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", nil, []uint{types[0].ID, types[1].ID}, nil))
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", nil, nil, []uint{types[0].ID}))
	require.Equal(t, []uint{types[1].ID}, typeIDs(t, database, p))
}

func TestSync_FullListBeatsAddRemove(t *testing.T) {
	database, p, types := setupDB(t)
	full := []uint{types[2].ID}
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", &full, []uint{types[0].ID}, nil))
	require.Equal(t, []uint{types[2].ID}, typeIDs(t, database, p))
}

func TestSync_UnknownIDIsAnErrorAndChangesNothing(t *testing.T) {
	database, p, types := setupDB(t)
	full := []uint{types[0].ID}
	require.NoError(t, Sync[models.ProblemType](database, &p, "Types", &full, nil, nil))

	bad := []uint{types[1].ID, 9999}
	err := database.Transaction(func(tx *gorm.DB) error {
		return Sync[models.ProblemType](tx, &p, "Types", &bad, nil, nil)
	})
	require.ErrorIs(t, err, ErrUnknownID)
	require.Equal(t, []uint{types[0].ID}, typeIDs(t, database, p))
}

func TestUserRefs(t *testing.T) {
	refs := UserRefs([]models.Profile{
		{ID: 4, User: models.AuthUser{Username: "haruxne"}},
		{ID: 9, User: models.AuthUser{Username: "NK"}},
	})
	require.Equal(t, []UserRef{{ID: 4, Username: "haruxne"}, {ID: 9, Username: "NK"}}, refs)
	require.NotNil(t, UserRefs(nil)) // marshals as [] not null
}
