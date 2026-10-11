package submission

import (
	"errors"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// fakeBridge records Abort calls instead of talking to the judge queue.
type fakeBridge struct {
	aborted []uint
	err     error
}

func (f *fakeBridge) Abort(subID uint) error {
	f.aborted = append(f.aborted, subID)
	return f.err
}

// setupAbortDB points db.DB at an in-memory sqlite holding submission 7 in the given status.
func setupAbortDB(t *testing.T, status string) {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&models.Submission{}))
	db.DB = database
	require.NoError(t, db.DB.Create(&models.Submission{ID: 7, UserID: 1, ProblemID: 1, LanguageID: 1, Date: time.Now(), Status: status}).Error)
}

func loadAbortSub(t *testing.T) models.Submission {
	t.Helper()
	var s models.Submission
	require.NoError(t, db.DB.First(&s, 7).Error)
	return s
}

func TestAbort_Queued(t *testing.T) {
	setupAbortDB(t, "QU")
	bridge := &fakeBridge{}

	require.NoError(t, NewSubmissionService(bridge).Abort(AbortRequest{SubmissionID: 7}))

	assert.Equal(t, []uint{7}, bridge.aborted)
	s := loadAbortSub(t)
	assert.Equal(t, "AB", s.Status)
	require.NotNil(t, s.Result)
	assert.Equal(t, "AB", *s.Result)
}

func TestAbort_ProcessingAndGradingStillWork(t *testing.T) {
	for _, status := range []string{"P", "G"} {
		t.Run(status, func(t *testing.T) {
			setupAbortDB(t, status)
			bridge := &fakeBridge{}
			require.NoError(t, NewSubmissionService(bridge).Abort(AbortRequest{SubmissionID: 7}))
			assert.Equal(t, []uint{7}, bridge.aborted)
			assert.Equal(t, "AB", loadAbortSub(t).Status)
		})
	}
}

func TestAbort_FinishedIsRejected(t *testing.T) {
	for _, status := range []string{"D", "CE", "IE", "AB"} {
		t.Run(status, func(t *testing.T) {
			setupAbortDB(t, status)
			bridge := &fakeBridge{}
			err := NewSubmissionService(bridge).Abort(AbortRequest{SubmissionID: 7})
			assert.ErrorIs(t, err, ErrSubmissionNotProcessing)
			assert.Empty(t, bridge.aborted)
			assert.Equal(t, status, loadAbortSub(t).Status)
		})
	}
}

func TestAbort_QueueErrorLeavesRowUntouched(t *testing.T) {
	setupAbortDB(t, "QU")
	bridge := &fakeBridge{err: errors.New("redis down")}

	assert.Error(t, NewSubmissionService(bridge).Abort(AbortRequest{SubmissionID: 7}))
	assert.Equal(t, "QU", loadAbortSub(t).Status)
}
