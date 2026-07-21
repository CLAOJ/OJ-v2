package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func runAdminGate(t *testing.T, userID uint, setUser bool) int {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if setUser {
		c.Set("user_id", userID)
	}
	AdminRequiredMiddleware()(c)
	return w.Code
}

func TestAdminGate_AllowsSuperuserNotStaff(t *testing.T) {
	setupPermsDB(t)
	u := models.AuthUser{Username: "superonly", IsActive: true, IsStaff: false, IsSuperuser: true}
	require.NoError(t, db.DB.Create(&u).Error)
	require.Equal(t, http.StatusOK, runAdminGate(t, u.ID, true)) // not aborted -> recorder stays 200
}

func TestAdminGate_AllowsStaff(t *testing.T) {
	setupPermsDB(t)
	u := models.AuthUser{Username: "staff", IsActive: true, IsStaff: true, IsSuperuser: false}
	require.NoError(t, db.DB.Create(&u).Error)
	require.Equal(t, http.StatusOK, runAdminGate(t, u.ID, true))
}

func TestAdminGate_DeniesPlainUser(t *testing.T) {
	setupPermsDB(t)
	u := models.AuthUser{Username: "plain", IsActive: true, IsStaff: false, IsSuperuser: false}
	require.NoError(t, db.DB.Create(&u).Error)
	require.Equal(t, http.StatusForbidden, runAdminGate(t, u.ID, true))
}

func TestAdminGate_DeniesAnonymous(t *testing.T) {
	setupPermsDB(t)
	require.Equal(t, http.StatusUnauthorized, runAdminGate(t, 0, false))
}
