package jobs

import (
	"testing"
	"time"

	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestJudgeStatus_PerJudgeBeat(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&models.Judge{}))
	db.DB = database
	mr := miniredis.RunT(t)
	old := cache.Client
	cache.Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { cache.Client = old })

	a := models.Judge{Name: "a", AuthKey: "k", Created: time.Now()}
	b := models.Judge{Name: "b", AuthKey: "k", Created: time.Now()}
	v1 := models.Judge{Name: "v1", AuthKey: "k", Created: time.Now(), Online: true}
	require.NoError(t, db.DB.Create(&a).Error)
	require.NoError(t, db.DB.Create(&b).Error)
	require.NoError(t, db.DB.Create(&v1).Error)
	get := func(id uint) models.Judge {
		var j models.Judge
		require.NoError(t, db.DB.First(&j, id).Error)
		return j
	}

	require.NoError(t, MarkJudgeAlive(a.ID))
	require.NoError(t, MarkJudgeAlive(b.ID))
	ja := get(a.ID)
	assert.True(t, ja.Online)
	require.NotNil(t, ja.StartTime)
	started := *ja.StartTime

	// A later beat keeps start_time.
	require.NoError(t, MarkJudgeAlive(a.ID))
	assert.True(t, get(a.ID).StartTime.Equal(started))

	// b stops beating: only b goes offline; v1's row (not listed) is untouched.
	mr.Del(judgeBeatKey(b.ID))
	require.NoError(t, SyncJudgeStatus([]string{"a", "b"}))
	assert.True(t, get(a.ID).Online)
	assert.False(t, get(b.ID).Online)
	assert.True(t, get(v1.ID).Online)

	// TTL expiry does the same as a missing beat.
	mr.FastForward(judgeBeatTTL + time.Second)
	require.NoError(t, SyncJudgeStatus([]string{"a", "b"}))
	assert.False(t, get(a.ID).Online)
}
