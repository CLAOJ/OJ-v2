package v2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// v1 /problem/<p>/resubmit/<submission> prefills the submit form from an
// earlier submission — but only its own author's (judge.resubmit_other may use
// anyone's). Another user must not be able to read the source this way.
func TestSubmissionResubmitSource_OwnershipGate(t *testing.T) {
	setupAdminSubmissionDB(t)

	owner := newSubmissionUser(t, "owner", false)
	other := newSubmissionUser(t, "other", false)
	moderator := newSubmissionUser(t, "moderator", false)
	grantSubmissionPerm(t, moderator.ID, "resubmit_other")

	var ownerProfile models.Profile
	require.NoError(t, db.DB.Where("user_id = ?", owner.ID).First(&ownerProfile).Error)

	problem := models.Problem{Code: "p1", Name: "Problem", IsPublic: true, TimeLimit: 1, MemoryLimit: 256}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&problem).Error)

	lang := models.Language{Key: "CPP17", Name: "C++17", Extension: "cpp"}
	require.NoError(t, db.DB.Create(&lang).Error)
	fileLang := models.Language{Key: "ZIP", Name: "ZIP", Extension: "zip", FileOnly: true, FileSizeLimit: 8}
	require.NoError(t, db.DB.Create(&fileLang).Error)

	sub := models.Submission{UserID: ownerProfile.ID, ProblemID: problem.ID, LanguageID: lang.ID, Date: time.Now(), Status: "D"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&sub).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.SubmissionSource{SubmissionID: sub.ID, Source: "int main(){}"}).Error)

	fileSub := models.Submission{UserID: ownerProfile.ID, ProblemID: problem.ID, LanguageID: fileLang.ID, Date: time.Now(), Status: "D"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&fileSub).Error)

	call := func(user models.AuthUser, id uint) (int, map[string]any) {
		router := authedRouter(user.ID, http.MethodGet, "/submission/:id/resubmit", SubmissionResubmitSource)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/submission/"+itoa(id)+"/resubmit", nil))
		var body map[string]any
		if w.Code == http.StatusOK {
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		}
		return w.Code, body
	}

	code, body := call(owner, sub.ID)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "int main(){}", body["source"])
	require.Equal(t, "CPP17", body["language"])

	// Someone else's submission: refused.
	code, _ = call(other, sub.ID)
	require.Equal(t, http.StatusForbidden, code)

	// judge.resubmit_other may copy it (v1's resubmit_other path).
	code, body = call(moderator, sub.ID)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "int main(){}", body["source"])

	// File-only languages have no source to prefill (v1 raises 404).
	code, _ = call(owner, fileSub.ID)
	require.Equal(t, http.StatusNotFound, code)

	// A missing submission is a 404, not a panic.
	code, _ = call(owner, 999999)
	require.Equal(t, http.StatusNotFound, code)
}
