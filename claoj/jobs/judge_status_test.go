package jobs

import (
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupJudgeRow(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&models.Judge{}))
	db.DB = database
	require.NoError(t, db.DB.Create(&models.Judge{Name: "Vịt Toàn Năng", AuthKey: "k", Created: time.Now()}).Error)
}

func useServers(t *testing.T, servers []*asynq.ServerInfo) {
	t.Helper()
	old := listServers
	t.Cleanup(func() { listServers = old })
	listServers = func() ([]*asynq.ServerInfo, error) { return servers, nil }
}

func loadJudge(t *testing.T) models.Judge {
	t.Helper()
	var j models.Judge
	require.NoError(t, db.DB.Where("name = ?", "Vịt Toàn Năng").First(&j).Error)
	return j
}

func TestSyncJudgeStatus_OnlineWhenAJudgeWorkerIsUp(t *testing.T) {
	setupJudgeRow(t)
	started := time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)
	useServers(t, []*asynq.ServerInfo{
		{Queues: map[string]int{"default": 3}, Concurrency: 10, Started: started.Add(-time.Hour)}, // a backend
		{Queues: map[string]int{"judge": 1}, Concurrency: 1, Started: started, ActiveWorkers: []*asynq.WorkerInfo{{}}},
		{Queues: map[string]int{"judge": 1}, Concurrency: 1, Started: started.Add(time.Minute)},
	})

	require.NoError(t, SyncJudgeStatus("Vịt Toàn Năng"))
	j := loadJudge(t)
	assert.True(t, j.Online)
	require.NotNil(t, j.Load)
	assert.InDelta(t, 0.5, *j.Load, 1e-9)
	require.NotNil(t, j.StartTime)
	assert.True(t, j.StartTime.Equal(started))
}

func TestSyncJudgeStatus_OfflineWhenNoJudgeWorker(t *testing.T) {
	setupJudgeRow(t)
	require.NoError(t, db.DB.Model(&models.Judge{}).Where("name = ?", "Vịt Toàn Năng").Update("online", true).Error)
	useServers(t, []*asynq.ServerInfo{{Queues: map[string]int{"default": 3}, Concurrency: 10}})

	require.NoError(t, SyncJudgeStatus("Vịt Toàn Năng"))
	assert.False(t, loadJudge(t).Online)
}
