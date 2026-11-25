package v2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

const testSubmissionFileSource = "/api/submission-file/0123456789abcdef.zip"

// A file-only submission's source is the uploaded file's public URL. v1
// (judge/views/submission.py) gives it only to the submitter, so a public
// submission must not advertise the URL to anonymous or unrelated viewers.
func TestSubmissionDetail_FileOnlySourceOnlyForOwnerAndSuperuser(t *testing.T) {
	setupAdminSubmissionDB(t)

	owner := newSubmissionUser(t, "owner", false)
	other := newSubmissionUser(t, "other", false)
	root := newSubmissionUser(t, "root", true)

	var ownerProfile models.Profile
	require.NoError(t, db.DB.Where("user_id = ?", owner.ID).First(&ownerProfile).Error)

	group := models.ProblemGroup{Name: "g", FullName: "Group"}
	require.NoError(t, db.DB.Create(&group).Error)
	problem := models.Problem{Code: "p1", Name: "Problem", GroupID: group.ID, IsPublic: true, TimeLimit: 1, MemoryLimit: 256}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&problem).Error)

	fileLang := models.Language{Key: "ZIP", Name: "ZIP", Extension: "zip", FileOnly: true, FileSizeLimit: 8}
	require.NoError(t, db.DB.Create(&fileLang).Error)
	textLang := models.Language{Key: "CPP17", Name: "C++17", Extension: "cpp"}
	require.NoError(t, db.DB.Create(&textLang).Error)

	fileSub := models.Submission{UserID: ownerProfile.ID, ProblemID: problem.ID, LanguageID: fileLang.ID, Date: time.Now(), Status: "D"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&fileSub).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.SubmissionSource{SubmissionID: fileSub.ID, Source: testSubmissionFileSource}).Error)

	textSub := models.Submission{UserID: ownerProfile.ID, ProblemID: problem.ID, LanguageID: textLang.ID, Date: time.Now(), Status: "D"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&textSub).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.SubmissionSource{SubmissionID: textSub.ID, Source: "int main(){}"}).Error)

	detail := func(user *models.AuthUser, id uint) map[string]any {
		var router *gin.Engine
		if user == nil {
			gin.SetMode(gin.TestMode)
			router = gin.New()
			router.GET("/submission/:id", SubmissionDetail)
		} else {
			router = authedRouter(user.ID, http.MethodGet, "/submission/:id", SubmissionDetail)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/submission/"+itoa(id), nil))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body
	}

	// Anonymous: no file URL at all.
	anon := detail(nil, fileSub.ID)
	require.Equal(t, "", anon["source"])

	// An unrelated logged-in user gets the same blank view.
	stranger := detail(&other, fileSub.ID)
	require.Equal(t, "", stranger["source"])

	// Owner and superuser get the stored file URL.
	for _, user := range []*models.AuthUser{&owner, &root} {
		body := detail(user, fileSub.ID)
		require.Equal(t, testSubmissionFileSource, body["source"])
	}

	// Text sources keep their existing behaviour: public problem, public source.
	for _, user := range []*models.AuthUser{nil, &other} {
		body := detail(user, textSub.ID)
		require.Equal(t, "int main(){}", body["source"])
	}
}
