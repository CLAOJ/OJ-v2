package bridge

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/CLAOJ/claoj/cache"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const judgeName = "Vịt Toàn Năng" // prod name: non-ASCII on purpose

// setupEventDB: sqlite with judge "Vịt Toàn Năng"/"secret", a blocked judge,
// languages C and PY3, problem aplusb (100 pts, partial) and submission 7;
// Redis via miniredis.
func setupEventDB(t *testing.T) *models.Judge {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&models.Judge{}, &models.Problem{}, &models.Profile{},
		&models.Submission{}, &models.SubmissionTestCase{},
		&models.Language{}, &models.RuntimeVersion{},
	))
	db.DB = database

	mr := miniredis.RunT(t)
	cache.Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})

	judge := models.Judge{Name: judgeName, AuthKey: "secret", Created: time.Now()}
	require.NoError(t, db.DB.Create(&judge).Error)
	require.NoError(t, db.DB.Create(&models.Judge{Name: "blocked", AuthKey: "secret", Created: time.Now(), IsBlocked: true}).Error)
	require.NoError(t, db.DB.Create(&models.Problem{ID: 1, Code: "aplusb", Name: "A+B", GroupID: 1, TimeLimit: 1, MemoryLimit: 65536, Points: 100, Partial: true}).Error)
	require.NoError(t, db.DB.Create(&models.Profile{ID: 1, UserID: 1, Timezone: "UTC", LanguageID: 1}).Error)
	require.NoError(t, db.DB.Create(&models.Submission{ID: 7, UserID: 1, ProblemID: 1, LanguageID: 1, Date: time.Now(), Status: "QU"}).Error)
	require.NoError(t, db.DB.Create(&models.Language{Key: "C", Name: "C", CommonName: "C", Ace: "c", Pygments: "c", Extension: "c"}).Error)
	require.NoError(t, db.DB.Create(&models.Language{Key: "PY3", Name: "Python 3", CommonName: "Python", Ace: "python", Pygments: "python", Extension: "py"}).Error)
	return &judge
}

func newEventRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/internal/judge/event", EventHandler)
	return r
}

func post(t *testing.T, r *gin.Engine, user, pass string, pkt Packet) int {
	t.Helper()
	body, err := json.Marshal(pkt)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/internal/judge/event", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func loadSub(t *testing.T) models.Submission {
	t.Helper()
	var s models.Submission
	require.NoError(t, db.DB.First(&s, 7).Error)
	return s
}

func TestEvent_Auth(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	ack := Packet{"name": "submission-acknowledged", "submission-id": 7}

	assert.Equal(t, http.StatusUnauthorized, post(t, r, "", "", ack))
	assert.Equal(t, http.StatusUnauthorized, post(t, r, judgeName, "wrong", ack))
	assert.Equal(t, http.StatusUnauthorized, post(t, r, "blocked", "secret", ack))
	assert.Equal(t, http.StatusUnauthorized, post(t, r, "nobody", "secret", ack))
	assert.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", ack))
	assert.Equal(t, "P", loadSub(t).Status)
}

func TestEvent_MissingSubmissionIDIsBadRequest(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	assert.Equal(t, http.StatusBadRequest, post(t, r, judgeName, "secret", Packet{"name": "grading-begin"}))
}

func TestEvent_FullGradingFlowIsReplaySafe(t *testing.T) {
	judge := setupEventDB(t)
	r := newEventRouter()
	send := func(p Packet) { require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", p)) }

	send(Packet{"name": "grading-begin", "submission-id": 7, "pretested": false})
	s := loadSub(t)
	assert.Equal(t, "G", s.Status)
	require.NotNil(t, s.JudgedOnID)
	assert.Equal(t, judge.ID, *s.JudgedOnID)

	cases := Packet{"name": "test-case-status", "submission-id": 7, "cases": []interface{}{
		map[string]interface{}{"position": 1, "status": 0, "time": 0.1, "memory": 1024, "points": 10, "total-points": 10, "feedback": "", "output": ""},
		map[string]interface{}{"position": 2, "status": 1, "time": 0.2, "memory": 2048, "points": 0, "total-points": 10, "feedback": "wrong", "output": ""},
	}}
	send(cases)
	send(cases) // replayed after a lost response: must not duplicate rows

	var n int64
	require.NoError(t, db.DB.Model(&models.SubmissionTestCase{}).Where("submission_id = ?", 7).Count(&n).Error)
	assert.Equal(t, int64(2), n)

	send(Packet{"name": "grading-end", "submission-id": 7})
	s = loadSub(t)
	assert.Equal(t, "D", s.Status)
	require.NotNil(t, s.Result)
	assert.Equal(t, "WA", *s.Result)
	require.NotNil(t, s.Points)
	assert.InDelta(t, 50.0, *s.Points, 1e-9) // 10/20 of 100, partial
	assert.InDelta(t, 0.3, *s.Time, 1e-9)
}

func TestEvent_CompileMessageKeepsGrading(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "grading-begin", "submission-id": 7}))
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "compile-message", "submission-id": 7, "log": "warning: unused"}))
	s := loadSub(t)
	assert.Equal(t, "G", s.Status)
	require.NotNil(t, s.Error)
	assert.Equal(t, "warning: unused", *s.Error)
}

func TestEvent_CompileErrorAndInternalError(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "compile-error", "submission-id": 7, "log": "boom"}))
	s := loadSub(t)
	assert.Equal(t, "CE", s.Status)
	assert.Equal(t, "CE", *s.Result)

	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "internal-error", "submission-id": 7, "message": "judge crashed"}))
	s = loadSub(t)
	assert.Equal(t, "IE", s.Status)
	assert.Equal(t, "IE", *s.Result)
}

func TestEvent_SubmissionTerminatedIsAborted(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "submission-terminated", "submission-id": 7}))
	s := loadSub(t)
	assert.Equal(t, "AB", s.Status)
	assert.Equal(t, "AB", *s.Result)
}

func TestEvent_LongFeedbackIsTruncated(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	long := strings.Repeat("ố", 80) // multibyte: truncate by runes, not bytes
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "test-case-status", "submission-id": 7, "cases": []interface{}{
		map[string]interface{}{"position": 1, "status": 1, "feedback": long},
	}}))
	var tc models.SubmissionTestCase
	require.NoError(t, db.DB.Where("submission_id = ?", 7).First(&tc).Error)
	assert.Equal(t, 50, len([]rune(tc.Feedback)))
}

func TestEvent_UnknownPacketIsIgnored(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	assert.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "batch-begin", "submission-id": 7}))
	assert.Equal(t, "QU", loadSub(t).Status)
}

func TestEvent_JudgeStatusRecordsPing(t *testing.T) {
	judge := setupEventDB(t)
	r := newEventRouter()
	require.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "judge-status", "ping": 42.5}))

	var j models.Judge
	require.NoError(t, db.DB.First(&j, judge.ID).Error)
	require.NotNil(t, j.Ping)
	assert.InDelta(t, 42.5, *j.Ping, 1e-9)
}

func TestEvent_JudgeStatusWithoutPingIsIgnored(t *testing.T) {
	setupEventDB(t)
	r := newEventRouter()
	assert.Equal(t, http.StatusNoContent, post(t, r, judgeName, "secret", Packet{"name": "judge-status"}))
}
