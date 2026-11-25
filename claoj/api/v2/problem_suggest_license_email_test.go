package v2

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// setupSuggestionTestDB seeds the taxonomy a suggestion needs plus two users
// (a superuser suggester and a second profile to try to smuggle in).
func setupSuggestionTestDB(t *testing.T) (*gorm.DB, models.AuthUser, models.Profile, models.Profile) {
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
		&models.AuthUserPermission{}, &models.Profile{}, &models.ProblemGroup{},
		&models.ProblemType{}, &models.Language{}, &models.License{}, &models.Problem{},
	))
	db.DB = database

	group := models.ProblemGroup{Name: "g", FullName: "Graphs"}
	require.NoError(t, db.DB.Create(&group).Error)
	ptype := models.ProblemType{Name: "dp", FullName: "DP"}
	require.NoError(t, db.DB.Create(&ptype).Error)
	require.NoError(t, db.DB.Create(&models.Language{Key: "CPP17", Name: "C++17", IncludeInProblem: true}).Error)
	require.NoError(t, db.DB.Create(&models.Language{Key: "HIDDEN", Name: "Hidden", IncludeInProblem: false}).Error)

	suggester := models.AuthUser{Username: "suggester", Email: "s@example.com", IsActive: true, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&suggester).Error)
	sp := models.Profile{UserID: suggester.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&sp).Error)

	other := models.AuthUser{Username: "other", Email: "o@example.com", IsActive: true}
	require.NoError(t, db.DB.Create(&other).Error)
	op := models.Profile{UserID: other.ID, Timezone: "UTC"}
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&op).Error)

	return database, suggester, sp, op
}

// suggestionRouter plants user_id like the auth middleware and applies the
// same permission gate the real route uses.
func suggestionRouter(userID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/problems/suggest", func(c *gin.Context) {
		c.Set("user_id", userID)
	}, auth.RequirePerm("judge.suggest_new_problem"), SuggestProblemCreate)
	return r
}

func postSuggestion(t *testing.T, r *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/problems/suggest", bytes.NewBuffer(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSuggestProblemCreate_PermissionGate(t *testing.T) {
	_, _, _, other := setupSuggestionTestDB(t)

	w := postSuggestion(t, suggestionRouter(other.UserID), map[string]any{
		"code": "sug1", "name": "Sug", "description": "d", "points": 100,
		"time_limit": 1, "memory_limit": 262144, "group_id": 1,
	})
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSuggestProblemCreate_ForcesSuggesterAndPrivacy(t *testing.T) {
	_, suggester, sp, other := setupSuggestionTestDB(t)

	w := postSuggestion(t, suggestionRouter(suggester.ID), map[string]any{
		"code": "sug1", "name": "Sug", "description": "d", "points": 100,
		"time_limit": 1, "memory_limit": 262144, "group_id": 1,
		"is_public": true, "suggester_id": other.ID, "author_ids": []uint{other.ID},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var p models.Problem
	require.NoError(t, db.DB.Preload("Authors").Preload("AllowedLangs").
		Where("code = ?", "sug1").First(&p).Error)
	assert.False(t, p.IsPublic)
	require.NotNil(t, p.SuggesterID)
	assert.Equal(t, sp.ID, *p.SuggesterID)
	assert.Empty(t, p.Authors, "suggestions carry no authors")
	require.NotNil(t, p.Date, "v1 stamps the suggestion date")
	require.Len(t, p.AllowedLangs, 1, "only include_in_problem languages are attached")
	assert.Equal(t, "CPP17", p.AllowedLangs[0].Key)
}

func TestSuggestProblemCreate_RejectsDuplicateCode(t *testing.T) {
	_, suggester, sp, _ := setupSuggestionTestDB(t)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Problem{
		Code: "taken", Name: "t", Description: "d", GroupID: 1,
		TimeLimit: 1, MemoryLimit: 256, Points: 1, SuggesterID: &sp.ID,
	}).Error)

	w := postSuggestion(t, suggestionRouter(suggester.ID), map[string]any{
		"code": "taken", "name": "Sug", "description": "d", "points": 100,
		"time_limit": 1, "memory_limit": 262144, "group_id": 1,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------
// License
// ---------------------------------------------------------------

func TestPublicLicenseAndProblemDetailLicense(t *testing.T) {
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
		&models.AuthUserPermission{}, &models.Profile{}, &models.ProblemGroup{},
		&models.ProblemType{}, &models.Language{}, &models.License{}, &models.Problem{},
	))
	db.DB = database

	group := models.ProblemGroup{Name: "g", FullName: "Graphs"}
	require.NoError(t, db.DB.Create(&group).Error)
	license := models.License{Key: "cc", Link: "https://cc.org", Name: "CC BY", Display: "CC", Text: "# Terms"}
	require.NoError(t, db.DB.Create(&license).Error)
	require.NoError(t, db.DB.Omit(clause.Associations).Create(&models.Problem{
		Code: "lic", Name: "Licensed", Description: "d", GroupID: group.ID,
		TimeLimit: 1, MemoryLimit: 256, Points: 10, IsPublic: true, LicenseID: &license.ID,
	}).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/license/:key", PublicLicenseDetail)
	r.GET("/problem/:code", ProblemDetail)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/license/cc", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "CC BY", got["name"])
	assert.Equal(t, "# Terms", got["text"])
	assert.Equal(t, "https://cc.org", got["link"])

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/license/missing", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/problem/lic", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var detail map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detail))
	lic, ok := detail["license"].(map[string]any)
	require.True(t, ok, "problem detail must carry the license")
	assert.Equal(t, "cc", lic["key"])
	assert.Equal(t, "CC BY", lic["name"])
	assert.Equal(t, "CC", lic["display"])
}

// ---------------------------------------------------------------
// Email change
// ---------------------------------------------------------------

type sentMail struct {
	to, username, newEmail, link string
}

// emailChangeRouter plants the session user and resets the per-user throttle
// so each test starts from a clean window.
func emailChangeRouter(t *testing.T, userID uint) (*gin.Engine, *[]sentMail) {
	t.Helper()
	emailChangeLimiter = ratelimit.NewInMemoryRepository(3600)
	var sent []sentMail
	oldActivate, oldNotice := sendEmailChangeActivation, sendEmailChangeNotice
	sendEmailChangeActivation = func(to, username, newEmail, link string) error {
		sent = append(sent, sentMail{to, username, newEmail, link})
		return nil
	}
	sendEmailChangeNotice = func(to, username, newEmail string) error {
		sent = append(sent, sentMail{to, username, newEmail, ""})
		return nil
	}
	t.Cleanup(func() {
		sendEmailChangeActivation, sendEmailChangeNotice = oldActivate, oldNotice
	})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/auth/email/change", func(c *gin.Context) { c.Set("user_id", userID) }, EmailChangeStart)
	r.POST("/auth/email/change/confirm", func(c *gin.Context) { c.Set("user_id", userID) }, EmailChangeConfirm)
	return r, &sent
}

func postJSON(t *testing.T, r *gin.Engine, path string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBuffer(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func setupEmailChangeTestDB(t *testing.T) (*gorm.DB, models.AuthUser) {
	t.Helper()
	database := setupLoginTestDB(t)
	db.DB = database
	user := createTestUser(t, database, "mailer", "oldpassword123", true)
	return database, user
}

// tokenFromLink returns the token query parameter of a confirmation link.
func tokenFromLink(t *testing.T, link string) string {
	t.Helper()
	u, err := url.Parse(link)
	require.NoError(t, err)
	token := u.Query().Get("token")
	require.NotEmpty(t, token)
	return token
}

func TestEmailChange_HappyPathNotifiesBothAndChangesEmail(t *testing.T) {
	database, user := setupEmailChangeTestDB(t)
	r, sent := emailChangeRouter(t, user.ID)

	w := postJSON(t, r, "/auth/email/change", map[string]any{
		"new_email": "new@example.com", "password": "oldpassword123",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, *sent, 2, "new address gets the link, old address gets a notice")
	assert.Equal(t, "new@example.com", (*sent)[0].to)
	assert.NotEmpty(t, (*sent)[0].link)
	assert.Equal(t, "mailer@example.com", (*sent)[1].to)
	assert.Empty(t, (*sent)[1].link, "v1 notice carries no activation link")

	token := tokenFromLink(t, (*sent)[0].link)
	w = postJSON(t, r, "/auth/email/change/confirm", map[string]any{"token": token})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.AuthUser
	require.NoError(t, database.First(&got, user.ID).Error)
	assert.Equal(t, "new@example.com", got.Email)

	// Single-use: the same link cannot be redeemed again.
	w = postJSON(t, r, "/auth/email/change/confirm", map[string]any{"token": token})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailChange_RejectsTakenEmailAndWrongPassword(t *testing.T) {
	_, user := setupEmailChangeTestDB(t)
	r, _ := emailChangeRouter(t, user.ID)

	w := postJSON(t, r, "/auth/email/change", map[string]any{
		"new_email": "mailer@example.com", "password": "oldpassword123",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = postJSON(t, r, "/auth/email/change", map[string]any{
		"new_email": "new@example.com", "password": "wrong",
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestEmailChange_RechecksUniquenessOnConfirm(t *testing.T) {
	database, user := setupEmailChangeTestDB(t)
	r, sent := emailChangeRouter(t, user.ID)

	w := postJSON(t, r, "/auth/email/change", map[string]any{
		"new_email": "new@example.com", "password": "oldpassword123",
	})
	require.Equal(t, http.StatusOK, w.Code)
	token := tokenFromLink(t, (*sent)[0].link)

	require.NoError(t, database.Create(&models.AuthUser{
		Username: "squatter", Email: "new@example.com", IsActive: true,
	}).Error)

	w = postJSON(t, r, "/auth/email/change/confirm", map[string]any{"token": token})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var got models.AuthUser
	require.NoError(t, database.First(&got, user.ID).Error)
	assert.Equal(t, "mailer@example.com", got.Email, "unconfirmed change must not touch the address")
}

func TestEmailChange_ConfirmRejectsExpiredAndForeignTokens(t *testing.T) {
	database, user := setupEmailChangeTestDB(t)
	other := createTestUser(t, database, "intruder", "oldpassword123", true)
	r, _ := emailChangeRouter(t, user.ID)

	expired, err := newEmailChangeToken(user.ID, "late@example.com")
	require.NoError(t, err)
	require.NoError(t, authHandlers.OneTimeTokens.Issue(tokenstore.KindEmailChange, expired, user.ID, -time.Minute))
	w := postJSON(t, r, "/auth/email/change/confirm", map[string]any{"token": expired})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	foreign, err := newEmailChangeToken(user.ID, "foreign@example.com")
	require.NoError(t, err)
	require.NoError(t, authHandlers.OneTimeTokens.Issue(tokenstore.KindEmailChange, foreign, user.ID, time.Hour))
	rOther, _ := emailChangeRouter(t, other.ID)
	w = postJSON(t, rOther, "/auth/email/change/confirm", map[string]any{"token": foreign})
	assert.Equal(t, http.StatusForbidden, w.Code)

	// The wrong-account click must not burn the token: the real owner can
	// still use the same link.
	w = postJSON(t, r, "/auth/email/change/confirm", map[string]any{"token": foreign})
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestEmailChange_RateLimited(t *testing.T) {
	_, user := setupEmailChangeTestDB(t)
	r, _ := emailChangeRouter(t, user.ID)

	for i := 0; i < 10; i++ {
		w := postJSON(t, r, "/auth/email/change", map[string]any{
			"new_email": "new@example.com", "password": "oldpassword123",
		})
		require.Equal(t, http.StatusOK, w.Code, "request %d", i+1)
	}
	w := postJSON(t, r, "/auth/email/change", map[string]any{
		"new_email": "new@example.com", "password": "oldpassword123",
	})
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}
