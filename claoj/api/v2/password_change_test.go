package v2

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authHandlers "github.com/CLAOJ/claoj/api/v2/auth"
	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/auth/tokenstore"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// passwordChangeRouter wires PasswordChange behind a stub that plants the
// user_id the real auth middleware would set. Passing userID 0 simulates an
// unauthenticated request.
func passwordChangeRouter(userID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/auth/password/change", func(c *gin.Context) {
		if userID != 0 {
			c.Set("user_id", userID)
		}
		authHandlers.PasswordChange(c)
	})
	return r
}

func postPasswordChange(r *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/password/change", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func currentPasswordHash(t *testing.T, userID uint) string {
	t.Helper()
	var u models.AuthUser
	require.NoError(t, db.DB.Where("id = ?", userID).First(&u).Error)
	return u.Password
}

func TestPasswordChange_Success(t *testing.T) {
	database := setupLoginTestDB(t)
	db.DB = database
	user := createTestUser(t, database, "changer", "oldpassword123", true)

	w := postPasswordChange(passwordChangeRouter(user.ID), map[string]any{
		"current_password": "oldpassword123",
		"new_password":     "brandnewpassword456",
	})

	assert.Equal(t, http.StatusOK, w.Code)

	ok, err := auth.CheckPassword("brandnewpassword456", currentPasswordHash(t, user.ID))
	require.NoError(t, err)
	assert.True(t, ok, "the stored hash should verify against the new password")

	ok, err = auth.CheckPassword("oldpassword123", currentPasswordHash(t, user.ID))
	require.NoError(t, err)
	assert.False(t, ok, "the old password must no longer work")
}

// The tab that changed the password must stay signed in, so the handler mints
// a fresh token pair after revoking the old ones.
func TestPasswordChange_ReissuesTheCallersSession(t *testing.T) {
	database := setupLoginTestDB(t)
	db.DB = database
	user := createTestUser(t, database, "changer", "oldpassword123", true)

	w := postPasswordChange(passwordChangeRouter(user.ID), map[string]any{
		"current_password": "oldpassword123",
		"new_password":     "brandnewpassword456",
	})
	require.Equal(t, http.StatusOK, w.Code)

	var access, refresh *http.Cookie
	for _, ck := range w.Result().Cookies() {
		switch ck.Name {
		case "access_token":
			access = ck
		case "refresh_token":
			refresh = ck
		}
	}
	require.NotNil(t, access, "a fresh access_token cookie must be set")
	require.NotNil(t, refresh, "a fresh refresh_token cookie must be set")
	assert.NotEmpty(t, access.Value)
	assert.NotEmpty(t, refresh.Value)
	assert.True(t, access.HttpOnly)
	assert.True(t, refresh.HttpOnly)
}

// Changing a password exists to lock out whoever might know the old one, so
// every other outstanding session must die.
func TestPasswordChange_RevokesOtherSessions(t *testing.T) {
	database := setupLoginTestDB(t)
	db.DB = database
	user := createTestUser(t, database, "changer", "oldpassword123", true)

	otherSession := "some-other-tabs-refresh-token"
	require.NoError(t, authHandlers.RefreshStore.Save(otherSession, tokenstore.Entry{
		UserID:    user.ID,
		FamilyID:  "other-family",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}))

	w := postPasswordChange(passwordChangeRouter(user.ID), map[string]any{
		"current_password": "oldpassword123",
		"new_password":     "brandnewpassword456",
	})
	require.Equal(t, http.StatusOK, w.Code)

	entry, found, err := authHandlers.RefreshStore.Get(otherSession)
	require.NoError(t, err)
	require.True(t, found)
	assert.True(t, entry.Revoked, "sessions opened with the old password must be revoked")
}

func TestPasswordChange_RejectsWrongCurrentPassword(t *testing.T) {
	database := setupLoginTestDB(t)
	db.DB = database
	user := createTestUser(t, database, "changer", "oldpassword123", true)
	before := currentPasswordHash(t, user.ID)

	w := postPasswordChange(passwordChangeRouter(user.ID), map[string]any{
		"current_password": "not-the-right-one",
		"new_password":     "brandnewpassword456",
	})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, before, currentPasswordHash(t, user.ID), "password must be untouched")
}

func TestPasswordChange_RejectsUnauthenticated(t *testing.T) {
	database := setupLoginTestDB(t)
	db.DB = database
	user := createTestUser(t, database, "changer", "oldpassword123", true)
	before := currentPasswordHash(t, user.ID)

	w := postPasswordChange(passwordChangeRouter(0), map[string]any{
		"current_password": "oldpassword123",
		"new_password":     "brandnewpassword456",
	})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, before, currentPasswordHash(t, user.ID))
}

func TestPasswordChange_RejectsShortAndUnchangedPasswords(t *testing.T) {
	database := setupLoginTestDB(t)
	db.DB = database
	user := createTestUser(t, database, "changer", "oldpassword123", true)
	before := currentPasswordHash(t, user.ID)

	t.Run("too short", func(t *testing.T) {
		w := postPasswordChange(passwordChangeRouter(user.ID), map[string]any{
			"current_password": "oldpassword123",
			"new_password":     "abc",
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, before, currentPasswordHash(t, user.ID))
	})

	t.Run("same as current", func(t *testing.T) {
		w := postPasswordChange(passwordChangeRouter(user.ID), map[string]any{
			"current_password": "oldpassword123",
			"new_password":     "oldpassword123",
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, before, currentPasswordHash(t, user.ID))
	})
}
