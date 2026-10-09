package jobs

import (
	"encoding/json"
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

// setupJudgeDB points db.DB at an in-memory sqlite with one C++17 submission (id 7) on problem "aplusb".
func setupJudgeDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&models.Language{}, &models.Problem{}, &models.Submission{},
		&models.SubmissionSource{}, &models.LanguageLimit{},
	))
	db.DB = database
	require.NoError(t, db.DB.Create(&models.Language{ID: 1, Key: "CPP17", Name: "C++17", CommonName: "C++", Ace: "c_cpp", Pygments: "cpp", Extension: "cpp"}).Error)
	require.NoError(t, db.DB.Create(&models.Problem{ID: 1, Code: "aplusb", Name: "A+B", GroupID: 1, TimeLimit: 1, MemoryLimit: 65536, Points: 100}).Error)
	require.NoError(t, db.DB.Create(&models.Submission{ID: 7, UserID: 1, ProblemID: 1, LanguageID: 1, Date: time.Now(), Status: "QU"}).Error)
	require.NoError(t, db.DB.Create(&models.SubmissionSource{SubmissionID: 7, Source: "int main(){}"}).Error)
}

// stubQueue replaces the asynq seams for one test and records calls.
type stubQueue struct {
	enqueues  [][]asynq.Option
	payloads  [][]byte
	enqErrs   []error // returned by successive enqueue calls; nil when exhausted
	state     asynq.TaskState
	infoErr   error
	deleted   []string
	cancelled []string
}

func useStubQueue(t *testing.T) *stubQueue {
	t.Helper()
	s := &stubQueue{}
	oldEnq, oldInfo, oldDel, oldCancel := enqueueFunc, getTaskInfo, deleteTask, cancelProcessing
	t.Cleanup(func() { enqueueFunc, getTaskInfo, deleteTask, cancelProcessing = oldEnq, oldInfo, oldDel, oldCancel })
	enqueueFunc = func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
		s.enqueues = append(s.enqueues, opts)
		s.payloads = append(s.payloads, task.Payload())
		var err error
		if len(s.enqErrs) > 0 {
			err, s.enqErrs = s.enqErrs[0], s.enqErrs[1:]
		}
		return &asynq.TaskInfo{ID: "sub-7"}, err
	}
	getTaskInfo = func(queue, id string) (*asynq.TaskInfo, error) {
		if s.infoErr != nil {
			return nil, s.infoErr
		}
		return &asynq.TaskInfo{ID: id, Queue: queue, State: s.state}, nil
	}
	deleteTask = func(queue, id string) error { s.deleted = append(s.deleted, queue+"/"+id); return nil }
	cancelProcessing = func(id string) error { s.cancelled = append(s.cancelled, id); return nil }
	return s
}

func optValue(opts []asynq.Option, typ asynq.OptionType) interface{} {
	for _, o := range opts {
		if o.Type() == typ {
			return o.Value()
		}
	}
	return nil
}

func TestBuildSubmissionRequest_UsesLanguageLimitOverride(t *testing.T) {
	setupJudgeDB(t)
	require.NoError(t, db.DB.Create(&models.LanguageLimit{ProblemID: 1, LanguageID: 1, TimeLimit: 2.5, MemoryLimit: 131072}).Error)

	pkt, err := BuildSubmissionRequest(7)
	require.NoError(t, err)
	assert.Equal(t, "submission-request", pkt["name"])
	assert.Equal(t, uint(7), pkt["submission-id"])
	assert.Equal(t, "aplusb", pkt["problem-id"])
	assert.Equal(t, "CPP17", pkt["language"])
	assert.Equal(t, "int main(){}", pkt["source"])
	assert.Equal(t, 2.5, pkt["time-limit"])
	assert.Equal(t, uint(131072), pkt["memory-limit"])
}

func TestBuildSubmissionRequest_MissingSubmission(t *testing.T) {
	setupJudgeDB(t)
	_, err := BuildSubmissionRequest(999)
	assert.Error(t, err)
}

func TestEnqueueJudgeSubmission_QueueTaskIDAndPayload(t *testing.T) {
	setupJudgeDB(t)
	s := useStubQueue(t)

	require.NoError(t, EnqueueJudgeSubmission(7))
	require.Len(t, s.enqueues, 1)
	assert.Equal(t, "judge", optValue(s.enqueues[0], asynq.QueueOpt))
	assert.Equal(t, "sub-7", optValue(s.enqueues[0], asynq.TaskIDOpt))
	assert.Equal(t, 2, optValue(s.enqueues[0], asynq.MaxRetryOpt))

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(s.payloads[0], &payload))
	assert.Equal(t, "aplusb", payload["problem-id"])
}

func TestEnqueueJudgeSubmission_ConflictWhilePendingIsNoop(t *testing.T) {
	setupJudgeDB(t)
	s := useStubQueue(t)
	s.enqErrs = []error{asynq.ErrTaskIDConflict}
	s.state = asynq.TaskStatePending

	require.NoError(t, EnqueueJudgeSubmission(7))
	assert.Len(t, s.enqueues, 1)
	assert.Empty(t, s.deleted)
}

func TestEnqueueJudgeSubmission_ConflictWhileActiveIsNoop(t *testing.T) {
	setupJudgeDB(t)
	s := useStubQueue(t)
	s.enqErrs = []error{asynq.ErrTaskIDConflict}
	s.state = asynq.TaskStateActive

	require.NoError(t, EnqueueJudgeSubmission(7))
	assert.Len(t, s.enqueues, 1)
	assert.Empty(t, s.deleted)
}

func TestEnqueueJudgeSubmission_ConflictWithArchivedRequeues(t *testing.T) {
	setupJudgeDB(t)
	s := useStubQueue(t)
	s.enqErrs = []error{asynq.ErrTaskIDConflict}
	s.state = asynq.TaskStateArchived

	require.NoError(t, EnqueueJudgeSubmission(7))
	assert.Equal(t, []string{"judge/sub-7"}, s.deleted)
	assert.Len(t, s.enqueues, 2)
}

func TestEnqueueJudgeSubmission_ConflictButTaskGoneRequeues(t *testing.T) {
	setupJudgeDB(t)
	s := useStubQueue(t)
	s.enqErrs = []error{asynq.ErrTaskIDConflict}
	s.infoErr = asynq.ErrTaskNotFound

	require.NoError(t, EnqueueJudgeSubmission(7))
	assert.Empty(t, s.deleted)
	assert.Len(t, s.enqueues, 2)
}

func TestAbortJudgeSubmission(t *testing.T) {
	t.Run("pending is deleted", func(t *testing.T) {
		s := useStubQueue(t)
		s.state = asynq.TaskStatePending
		require.NoError(t, AbortJudgeSubmission(7))
		assert.Equal(t, []string{"judge/sub-7"}, s.deleted)
		assert.Empty(t, s.cancelled)
	})
	t.Run("active is cancelled", func(t *testing.T) {
		s := useStubQueue(t)
		s.state = asynq.TaskStateActive
		require.NoError(t, AbortJudgeSubmission(7))
		assert.Equal(t, []string{"sub-7"}, s.cancelled)
		assert.Empty(t, s.deleted)
	})
	t.Run("unknown task is a no-op", func(t *testing.T) {
		s := useStubQueue(t)
		s.infoErr = asynq.ErrTaskNotFound
		require.NoError(t, AbortJudgeSubmission(7))
		assert.Empty(t, s.deleted)
		assert.Empty(t, s.cancelled)
	})
}
