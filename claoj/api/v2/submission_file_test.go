package v2

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/CLAOJ/claoj/models"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// submitFixture is one public problem with a text language and a file-only
// language, plus the submitting user.
type submitFixture struct {
	userID   uint
	textLang models.Language
	fileLang models.Language
	problem  models.Problem
}

func useTempSubmissionFileDir(t *testing.T) string {
	t.Helper()
	old := submissionFileDir
	submissionFileDir = t.TempDir()
	t.Cleanup(func() { submissionFileDir = old })
	return submissionFileDir
}

func setupSubmitDB(t *testing.T) submitFixture {
	t.Helper()
	old := db.DB
	t.Cleanup(func() { db.DB = old })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&models.AuthUser{}, &models.AuthGroup{}, &models.DjangoContentType{},
		&models.AuthPermission{}, &models.AuthUserGroup{}, &models.AuthGroupPermission{},
		&models.AuthUserPermission{}, &models.Profile{}, &models.Organization{},
		&models.ProblemGroup{}, &models.ProblemType{}, &models.Language{},
		&models.LanguageLimit{}, &models.Problem{}, &models.Submission{},
		&models.SubmissionSource{}, &models.Contest{}, &models.ContestProblem{},
		&models.ContestParticipation{}, &models.ContestSubmission{},
	))
	db.DB = database

	group := models.ProblemGroup{Name: "g", FullName: "Group"}
	require.NoError(t, db.DB.Create(&group).Error)

	user := models.AuthUser{Username: "submitter", IsActive: true}
	require.NoError(t, db.DB.Create(&user).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Profile{UserID: user.ID, Timezone: "UTC"}).Error)

	textLang := models.Language{Key: "C", Name: "C", Extension: "c"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&textLang).Error)
	fileLang := models.Language{Key: "ZIP", Name: "ZIP", Extension: "zip", FileOnly: true, FileSizeLimit: 1}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&fileLang).Error)
	noLimitLang := models.Language{Key: "BIN", Name: "BIN", Extension: "zip", FileOnly: true, FileSizeLimit: 0}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&noLimitLang).Error)
	// Languages whose configured extension is outside the old serving
	// whitelist: a Scratch project (.sb3) and an HTML-only language. The
	// upload must still be accepted — the language's extension is the gate.
	sb3Lang := models.Language{Key: "SB3", Name: "Scratch", Extension: "sb3", FileOnly: true, FileSizeLimit: 1}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&sb3Lang).Error)
	htmlLang := models.Language{Key: "HTML", Name: "HTML", Extension: "html", FileOnly: true, FileSizeLimit: 1}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&htmlLang).Error)
	// An extension that cannot be part of a stored name: rejected even though
	// it matches the language (no punctuation in generated file names).
	weirdLang := models.Language{Key: "WEIRD", Name: "Weird", Extension: "tar-gz", FileOnly: true, FileSizeLimit: 1}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&weirdLang).Error)

	problem := models.Problem{
		Code: "FILE1", Name: "File Problem", Description: "d", GroupID: group.ID,
		TimeLimit: 1, MemoryLimit: 256, Points: 100, IsPublic: true,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&problem).Error)
	require.NoError(t, db.DB.Model(&problem).Association("AllowedLangs").Append(
		[]models.Language{textLang, fileLang, noLimitLang, sb3Lang, htmlLang, weirdLang}))

	// Enqueueing is mocked: these tests assert the stored row and the packet,
	// not Redis.
	jobs.SetMockEnqueue(func(*asynq.Task, ...asynq.Option) (*asynq.TaskInfo, error) {
		return &asynq.TaskInfo{ID: "mock"}, nil
	})

	return submitFixture{userID: user.ID, textLang: textLang, fileLang: fileLang, problem: problem}
}

func submitFileUpload(t *testing.T, userID uint, code, language, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	require.NoError(t, writer.WriteField("language", language))
	part, err := writer.CreateFormFile("submission_file", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/problem/"+code+"/submit", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	authedRouter(userID, http.MethodPost, "/problem/:code/submit", Submit).ServeHTTP(w, req)
	return w
}

func submitJSON(t *testing.T, userID uint, code, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/problem/"+code+"/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	authedRouter(userID, http.MethodPost, "/problem/:code/submit", Submit).ServeHTTP(w, req)
	return w
}

func submissionID(t *testing.T, w *httptest.ResponseRecorder) uint {
	t.Helper()
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		SubmissionID uint `json:"submission_id"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.SubmissionID
}

func TestSubmit_TextLanguageStillAcceptsJSON(t *testing.T) {
	fx := setupSubmitDB(t)

	w := submitJSON(t, fx.userID, "FILE1", `{"language":"C","source":"int main(){}"}`)
	id := submissionID(t, w)

	var source models.SubmissionSource
	require.NoError(t, db.DB.Where("submission_id = ?", id).First(&source).Error)
	require.Equal(t, "int main(){}", source.Source, "the JSON path stores the code verbatim")
}

func TestSubmit_FileOnlyLanguageStoresFileAndBuildsJudgePacket(t *testing.T) {
	dir := useTempSubmissionFileDir(t)
	fx := setupSubmitDB(t)

	content := []byte("PK\x03\x04fake zip body")
	w := submitFileUpload(t, fx.userID, "FILE1", "ZIP", "answer.zip", content)
	id := submissionID(t, w)

	var source models.SubmissionSource
	require.NoError(t, db.DB.Where("submission_id = ?", id).First(&source).Error)
	require.Regexp(t, `^/api/submission-file/[0-9a-f]{16}\.zip$`, source.Source)

	name := strings.TrimPrefix(source.Source, "/api/submission-file/")
	onDisk, err := os.ReadFile(filepath.Join(dir, name))
	require.NoError(t, err)
	require.Equal(t, content, onDisk)

	// The judge packet resolves the stored path against the public site URL and
	// flags the submission as file-only, so the agent downloads it instead of
	// compiling source text.
	pkt, err := jobs.BuildSubmissionRequest(id)
	require.NoError(t, err)
	require.Equal(t, "ZIP", pkt["language"])
	require.True(t, strings.HasSuffix(pkt["source"].(string), source.Source))
	meta := pkt["meta"].(map[string]interface{})
	require.Equal(t, true, meta["file-only"])
	require.Equal(t, 1, meta["file-size-limit"])

	var count int64
	db.DB.Model(&models.Submission{}).Where("id = ?", id).Count(&count)
	require.EqualValues(t, 1, count)
}

func TestSubmit_FileOnlyLanguageRejectsBadUploads(t *testing.T) {
	cases := []struct {
		name     string
		language string
		filename string
		content  []byte
		message  string
	}{
		{"wrong extension", "ZIP", "answer.rar", []byte("data"), "wrong file type"},
		{"extension unfit for a stored name", "WEIRD", "answer.tar-gz", []byte("data"), "cannot be submitted"},
		{"oversize", "ZIP", "answer.zip", make([]byte, 1<<20+1), "too large"},
		{"empty", "ZIP", "answer.zip", nil, "empty"},
		{"limit not configured", "BIN", "answer.zip", []byte("data"), "not enabled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := useTempSubmissionFileDir(t)
			fx := setupSubmitDB(t)

			w := submitFileUpload(t, fx.userID, "FILE1", tc.language, tc.filename, tc.content)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			require.Contains(t, w.Body.String(), tc.message)
			entries, err := os.ReadDir(dir)
			require.NoError(t, err)
			require.Empty(t, entries, "a rejected upload must not leave a file behind")
		})
	}
}

func TestSubmit_LanguageSubmissionModeMismatch(t *testing.T) {
	useTempSubmissionFileDir(t)
	fx := setupSubmitDB(t)

	// A file uploaded for a text language: v1 rejects either-or.
	w := submitFileUpload(t, fx.userID, "FILE1", "C", "main.c", []byte("int main(){}"))
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "does not accept file submissions")

	// JSON source for a file-only language.
	w = submitJSON(t, fx.userID, "FILE1", `{"language":"ZIP","source":"irrelevant"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "requires a file submission")
}

func TestSubmit_ConfiguredExtensionOutsideContentTypeMapStillUploads(t *testing.T) {
	dir := useTempSubmissionFileDir(t)
	fx := setupSubmitDB(t)

	// The language's own extension is the gate. These are real file-only
	// languages (Scratch .sb3, an HTML language) whose extensions were not in
	// the old closed serving map; rejecting them made the language unusable.
	for _, tc := range []struct {
		language string
		filename string
		content  []byte
	}{
		{"SB3", "game.sb3", []byte("PK\x03\x04scratch")},
		{"HTML", "answer.html", []byte("<script>alert(1)</script>")},
	} {
		t.Run(tc.language, func(t *testing.T) {
			w := submitFileUpload(t, fx.userID, "FILE1", tc.language, tc.filename, tc.content)
			id := submissionID(t, w)

			var source models.SubmissionSource
			require.NoError(t, db.DB.Where("submission_id = ?", id).First(&source).Error)
			name := strings.TrimPrefix(source.Source, "/api/submission-file/")
			onDisk, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			require.Equal(t, tc.content, onDisk)
		})
	}
}

func TestSubmit_OversizeBodyIs413(t *testing.T) {
	dir := useTempSubmissionFileDir(t)
	fx := setupSubmitDB(t)

	// One byte over the whole-body cap: rejected before the language (and its
	// own smaller limit) is even resolved.
	w := submitFileUpload(t, fx.userID, "FILE1", "ZIP", "answer.zip", make([]byte, maxSubmissionUploadBytes+1))
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "too large")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestSubmit_RemovesFileWhenTheSubmissionTransactionFails(t *testing.T) {
	dir := useTempSubmissionFileDir(t)
	setupSubmitDB(t)

	// No judge_profile for this auth user id: the file is stored first, then
	// the transaction fails, and the deferred cleanup must delete the orphan.
	w := submitFileUpload(t, 9999, "FILE1", "ZIP", "answer.zip", []byte("PK\x03\x04"))
	require.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "user profile not found")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries, "a failed submission must not leave its upload behind")
}

// The web client submits from the contest problem page with ?contest=<key>;
// without reading that query param the submission was never linked to the
// contest (no judge_contestsubmission row, so no scoreboard/participation).
func TestSubmit_LinksContestFromQueryParam(t *testing.T) {
	useTempSubmissionFileDir(t)
	fx := setupSubmitDB(t)

	now := time.Now()
	contest := models.Contest{
		Key: "c1", Name: "C1",
		StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour),
		IsVisible: true,
	}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&contest).Error)
	var profile models.Profile
	require.NoError(t, db.DB.Where("user_id = ?", fx.userID).First(&profile).Error)
	participation := models.ContestParticipation{ContestID: contest.ID, UserID: profile.ID, RealStart: now}
	require.NoError(t, db.DB.Omit("Contest", "User").Create(&participation).Error)
	contestProblem := models.ContestProblem{ContestID: contest.ID, ProblemID: fx.problem.ID, Points: 100, Order: 1}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&contestProblem).Error)

	// JSON path.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/problem/FILE1/submit?contest=c1",
		strings.NewReader(`{"language":"C","source":"int main(){}"}`))
	req.Header.Set("Content-Type", "application/json")
	authedRouter(fx.userID, http.MethodPost, "/problem/:code/submit", Submit).ServeHTTP(w, req)
	id := submissionID(t, w)

	var linked models.ContestSubmission
	require.NoError(t, db.DB.Where("submission_id = ?", id).First(&linked).Error)
	require.Equal(t, participation.ID, linked.ParticipationID)
	require.Equal(t, contestProblem.ID, linked.ProblemID, "the link points at the contest problem row")
	var sub models.Submission
	require.NoError(t, db.DB.First(&sub, id).Error)
	require.NotNil(t, sub.ContestObjectID)
	require.Equal(t, contest.ID, *sub.ContestObjectID)

	// Multipart path, same query param.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	require.NoError(t, writer.WriteField("language", "ZIP"))
	part, err := writer.CreateFormFile("submission_file", "answer.zip")
	require.NoError(t, err)
	_, err = part.Write([]byte("PK\x03\x04"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/problem/FILE1/submit?contest=c1", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	authedRouter(fx.userID, http.MethodPost, "/problem/:code/submit", Submit).ServeHTTP(w, req)
	uploadID := submissionID(t, w)

	var uploadLink models.ContestSubmission
	require.NoError(t, db.DB.Where("submission_id = ?", uploadID).First(&uploadLink).Error)
	require.Equal(t, participation.ID, uploadLink.ParticipationID)
}

func TestSubmissionFileServe(t *testing.T) {
	dir := useTempSubmissionFileDir(t)
	content := []byte("PK\x03\x04")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0123456789abcdef.zip"), content, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0123456789abcdef.html"), []byte("<script>alert(1)</script>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0123456789abcdef.sb3"), content, 0o644))

	router := authedRouter(0, http.MethodGet, "/submission-file/:name", SubmissionFileServe)
	get := func(name string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/submission-file/"+name, nil))
		return w
	}

	w := get("0123456789abcdef.zip")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/zip", w.Header().Get("Content-Type"))
	require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	require.Equal(t, content, w.Body.Bytes())

	// A text-ish stored extension is served as text/plain, never as HTML: the
	// attachment + nosniff pair is what makes that safe.
	w = get("0123456789abcdef.html")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, textPlain, w.Header().Get("Content-Type"))
	require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))

	// Anything else is still an attachment, forced to octet-stream.
	w = get("0123456789abcdef.sb3")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
	require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))

	// Name shape: anything else is a 404, never a filesystem peek.
	for _, name := range []string{
		"0123456789abcdef.exe", "abc.zip", "0123456789ABCDEF.zip",
		"..%2Fsecret.zip", "0123456789abcdef.zip%00", "0123456789abcdef",
		"0123456789abcdef..zip",
	} {
		require.Equal(t, http.StatusNotFound, get(name).Code, name)
	}
	// Valid shape but missing.
	require.Equal(t, http.StatusNotFound, get("fedcba9876543210.zip").Code)
}

func TestSubmissionFileNameShape(t *testing.T) {
	for _, name := range []string{
		"0123456789abcdef.zip", "0123456789abcdef.py", "0123456789abcdef.in",
		"0123456789abcdef.html", "0123456789abcdef.sb3",
	} {
		require.True(t, submissionFileName.MatchString(name), name)
	}
	for _, name := range []string{
		"", ".", "..", "../etc/passwd", "/etc/passwd", `..\windows.zip`,
		"0123456789abcdef.zip/../x", "0123456789abcdef.", "0123456789abcdef.toolongextensions",
		"0123456789abcdef.png\x00", "0123456789abcde.zip",
	} {
		require.False(t, submissionFileName.MatchString(name), name)
	}
}
