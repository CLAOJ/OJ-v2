package auth

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// runGuard invokes a guard on a context authenticated as userID with the given
// path params, returning the recorder status and whether the chain was aborted.
func runGuard(userID uint, params gin.Params, guard gin.HandlerFunc) (int, bool) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Params = params
	guard(c)
	return w.Code, c.IsAborted()
}

func TestRequirePerm(t *testing.T) {
	setupPermsDB(t)
	granted := newActiveUser(t, "granted")
	newProfileFor(t, granted.ID)
	grantPermsViaGroup(t, granted.ID, []string{"add_problem"})
	denied := newActiveUser(t, "denied")
	newProfileFor(t, denied.ID)

	code, aborted := runGuard(granted.ID, nil, RequirePerm("judge.add_problem"))
	require.False(t, aborted)
	require.Equal(t, http.StatusOK, code)

	code, aborted = runGuard(denied.ID, nil, RequirePerm("judge.add_problem"))
	require.True(t, aborted)
	require.Equal(t, http.StatusForbidden, code)
}

func TestRequireProblemEdit(t *testing.T) {
	setupPermsDB(t)
	prob := models.Problem{Code: "p1", Name: "P1", IsPublic: false}
	require.NoError(t, db.DB.Create(&prob).Error)

	// Author with edit_own_problem may edit.
	author := newActiveUser(t, "author")
	authorProfile := newProfileFor(t, author.ID)
	require.NoError(t, db.DB.Model(&prob).Association("Authors").Append(&authorProfile))
	grantPermsViaGroup(t, author.ID, []string{"edit_own_problem"})

	// Plain staff (no perms, not author) may not.
	outsider := newActiveUser(t, "outsider")
	newProfileFor(t, outsider.ID)

	p := gin.Params{{Key: "code", Value: "p1"}}
	code, aborted := runGuard(author.ID, p, RequireProblemEdit())
	require.False(t, aborted)
	require.Equal(t, http.StatusOK, code)

	code, aborted = runGuard(outsider.ID, p, RequireProblemEdit())
	require.True(t, aborted)
	require.Equal(t, http.StatusForbidden, code)

	// Missing problem -> 404.
	code, aborted = runGuard(outsider.ID, gin.Params{{Key: "code", Value: "nope"}}, RequireProblemEdit())
	require.True(t, aborted)
	require.Equal(t, http.StatusNotFound, code)
}

func TestRequireContestEdit(t *testing.T) {
	setupPermsDB(t)
	ct := models.Contest{Key: "c1", Name: "C1"}
	require.NoError(t, db.DB.Create(&ct).Error)

	author := newActiveUser(t, "cauthor")
	authorProfile := newProfileFor(t, author.ID)
	require.NoError(t, db.DB.Model(&ct).Association("Authors").Append(&authorProfile))
	grantPermsViaGroup(t, author.ID, []string{"edit_own_contest"})
	outsider := newActiveUser(t, "coutsider")
	newProfileFor(t, outsider.ID)

	p := gin.Params{{Key: "key", Value: "c1"}}
	code, aborted := runGuard(author.ID, p, RequireContestEdit())
	require.False(t, aborted)
	require.Equal(t, http.StatusOK, code)

	code, aborted = runGuard(outsider.ID, p, RequireContestEdit())
	require.True(t, aborted)
	require.Equal(t, http.StatusForbidden, code)
}

func TestRequireOrgEdit(t *testing.T) {
	setupPermsDB(t)
	org := models.Organization{Name: "O1", Slug: "o1", ShortName: "O1"}
	require.NoError(t, db.DB.Create(&org).Error)

	admin := newActiveUser(t, "oadmin")
	adminProfile := newProfileFor(t, admin.ID)
	require.NoError(t, db.DB.Model(&org).Association("Admins").Append(&adminProfile))
	outsider := newActiveUser(t, "ooutsider")
	newProfileFor(t, outsider.ID)

	p := gin.Params{{Key: "id", Value: itoa(org.ID)}}
	code, aborted := runGuard(admin.ID, p, RequireOrgEdit())
	require.False(t, aborted)
	require.Equal(t, http.StatusOK, code)

	code, aborted = runGuard(outsider.ID, p, RequireOrgEdit())
	require.True(t, aborted)
	require.Equal(t, http.StatusForbidden, code)
}

func itoa(u uint) string { return strconv.FormatUint(uint64(u), 10) }
