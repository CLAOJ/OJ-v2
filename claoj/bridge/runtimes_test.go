package bridge

import (
	"net/http"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runtimeRows loads a judge's judge_runtimeversion rows for assertions.
func runtimeRows(t *testing.T, judgeID uint) []models.RuntimeVersion {
	t.Helper()
	var rows []models.RuntimeVersion
	require.NoError(t, db.DB.Preload("Language").Where("judge_id = ?", judgeID).
		Order("language_id, priority").Find(&rows).Error)
	return rows
}

// byName indexes runtime rows by their executor name (map iteration order
// in the packet is random, so never assert on row order across languages).
func byName(rows []models.RuntimeVersion) map[string]models.RuntimeVersion {
	byName := make(map[string]models.RuntimeVersion, len(rows))
	for _, r := range rows {
		byName[r.Name] = r
	}
	return byName
}

func runtimesPacket(executors map[string]interface{}) Packet {
	return Packet{"name": "judge-runtimes", "executors": executors}
}

func TestEvent_JudgeRuntimesWritesOneRowPerExecutor(t *testing.T) {
	judge := setupEventDB(t)
	r := newEventRouter()

	pkt := runtimesPacket(map[string]interface{}{
		"C": []interface{}{
			[]interface{}{"GCC 12", []interface{}{12, 2, 0}},
			[]interface{}{"ld.so", []interface{}{2, 39}},
		},
		"PY3":  []interface{}{[]interface{}{"Python 3", []interface{}{3, 10, 12}}},
		"NOPE": []interface{}{[]interface{}{"Mystery", []interface{}{1, 0}}}, // no judge_language row
	})
	// the judge comes from Basic auth, never from the packet
	pkt["judge-id"] = 999

	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", pkt))

	rows := runtimeRows(t, judge.ID)
	require.Len(t, rows, 3, "unknown keys must be skipped, known keys one row each")
	got := byName(rows)

	c1 := got["GCC 12"]
	assert.Equal(t, judge.ID, c1.JudgeID)
	assert.Equal(t, "C", c1.Language.Key)
	assert.Equal(t, "12.2.0", c1.Version)
	assert.Equal(t, 0, c1.Priority)

	c2 := got["ld.so"]
	assert.Equal(t, "C", c2.Language.Key)
	assert.Equal(t, "2.39", c2.Version)
	assert.Equal(t, 1, c2.Priority)

	py := got["Python 3"]
	assert.Equal(t, judge.ID, py.JudgeID)
	assert.Equal(t, "PY3", py.Language.Key)
	assert.Equal(t, "3.10.12", py.Version)
	assert.Equal(t, 0, py.Priority)
}

func TestEvent_JudgeRuntimesReplayDoesNotDuplicateRows(t *testing.T) {
	judge := setupEventDB(t)
	r := newEventRouter()
	pkt := runtimesPacket(map[string]interface{}{
		"C": []interface{}{[]interface{}{"GCC 12", []interface{}{12, 2, 0}}},
	})

	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", pkt))
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", pkt)) // lost response, re-POSTed

	rows := runtimeRows(t, judge.ID)
	require.Len(t, rows, 1, "a replayed packet must not duplicate rows")
	assert.Equal(t, "12.2.0", rows[0].Version)
}

func TestEvent_JudgeRuntimesLeavesOtherJudgesRowsAlone(t *testing.T) {
	judge := setupEventDB(t)
	other := models.Judge{Name: "other judge", AuthKey: "secret", Created: time.Now()}
	require.NoError(t, db.DB.Create(&other).Error)
	var py models.Language
	require.NoError(t, db.DB.Where("key = ?", "PY3").First(&py).Error)
	seed := models.RuntimeVersion{LanguageID: py.ID, JudgeID: other.ID, Name: "Python 3", Version: "3.9.1", Priority: 0}
	require.NoError(t, db.DB.Create(&seed).Error)

	r := newEventRouter()
	pkt := runtimesPacket(map[string]interface{}{
		"C": []interface{}{[]interface{}{"GCC 12", []interface{}{12, 2, 0}}},
	})
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", pkt))

	var kept []models.RuntimeVersion
	require.NoError(t, db.DB.Where("judge_id = ?", other.ID).Find(&kept).Error)
	require.Len(t, kept, 1, "another judge's runtime versions must survive")
	assert.Equal(t, "3.9.1", kept[0].Version)

	require.Len(t, runtimeRows(t, judge.ID), 1)
}

func TestEvent_JudgeRuntimesEmptyExecutorsClearsOnlyOwnRows(t *testing.T) {
	judge := setupEventDB(t)
	other := models.Judge{Name: "other judge", AuthKey: "secret", Created: time.Now()}
	require.NoError(t, db.DB.Create(&other).Error)
	var cLang models.Language
	require.NoError(t, db.DB.Where("key = ?", "C").First(&cLang).Error)
	require.NoError(t, db.DB.Create(&models.RuntimeVersion{LanguageID: cLang.ID, JudgeID: other.ID, Name: "GCC 12", Version: "12.2.0"}).Error)

	r := newEventRouter()
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret",
		runtimesPacket(map[string]interface{}{"C": []interface{}{}})))

	assert.Empty(t, runtimeRows(t, judge.ID))

	var kept []models.RuntimeVersion
	require.NoError(t, db.DB.Where("judge_id = ?", other.ID).Find(&kept).Error)
	assert.Len(t, kept, 1)
}

func TestEvent_JudgeRuntimesWithoutExecutorsIsBadRequest(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	assert.Equal(t, http.StatusBadRequest,
		post(t, r, judgeName, "secret", Packet{"name": "judge-runtimes"}))
	assert.Equal(t, http.StatusBadRequest,
		post(t, r, judgeName, "secret", Packet{"name": "judge-runtimes", "executors": "nope"}))
}
