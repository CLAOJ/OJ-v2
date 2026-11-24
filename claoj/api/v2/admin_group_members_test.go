package v2

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
)

// newMemberUser creates an auth_user plus its judge_profile with the given
// point total. Member lists carry judge_profile.id and join judge_profile for
// points.
func newMemberUser(t *testing.T, username string, points float64) (models.AuthUser, models.Profile) {
	t.Helper()
	u := models.AuthUser{Username: username, Email: username + "@example.com", IsActive: true}
	require.NoError(t, db.DB.Create(&u).Error)
	p := models.Profile{UserID: u.ID, Timezone: "UTC", Points: points}
	require.NoError(t, db.DB.Create(&p).Error)
	return u, p
}

type groupMembersBody struct {
	Data []struct {
		ID       uint    `json:"id"`
		Username string  `json:"username"`
		Email    string  `json:"email"`
		IsActive bool    `json:"is_active"`
		Points   float64 `json:"points"`
	} `json:"data"`
	Total     int64  `json:"total"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	GroupID   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
}

func decodeMembers(t *testing.T, body []byte) groupMembersBody {
	t.Helper()
	var out groupMembersBody
	require.NoError(t, json.Unmarshal(body, &out))
	return out
}

func membersPath(groupID uint) string {
	return "/admin/group/" + strconv.Itoa(int(groupID)) + "/members"
}

func TestAdminGroupMembers_ListSearchPagination(t *testing.T) {
	setupAdminGroupsTestDB(t)
	root := newSuperuserUser(t)
	group := models.AuthGroup{Name: "Judges"}
	require.NoError(t, db.DB.Create(&group).Error)

	alice, aliceProfile := newMemberUser(t, "alice", 10)
	bob, _ := newMemberUser(t, "bob", 20)
	carol, _ := newMemberUser(t, "carol", 30)
	newMemberUser(t, "dave", 40) // not a member

	for _, u := range []models.AuthUser{alice, bob, carol} {
		require.NoError(t, db.DB.Create(&models.AuthUserGroup{UserID: u.ID, GroupID: group.ID}).Error)
	}

	router := authedRouter(root.ID, http.MethodGet, "/admin/group/:id/members", AdminGroupMembers)
	target := membersPath(group.ID)

	w := doRequest(router, http.MethodGet, target, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := decodeMembers(t, w.Body.Bytes())
	require.Equal(t, int64(3), body.Total)
	require.Equal(t, group.ID, body.GroupID)
	require.Equal(t, "Judges", body.GroupName)
	require.Len(t, body.Data, 3)
	require.Equal(t, "alice", body.Data[0].Username)
	require.Equal(t, "alice@example.com", body.Data[0].Email)
	require.True(t, body.Data[0].IsActive)
	require.Equal(t, 10.0, body.Data[0].Points)
	require.Equal(t, aliceProfile.ID, body.Data[0].ID, "member id must be judge_profile.id")

	// Pagination: 3 members, 2 per page.
	w = doRequest(router, http.MethodGet, target+"?page=2&page_size=2", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body = decodeMembers(t, w.Body.Bytes())
	require.Equal(t, int64(3), body.Total)
	require.Equal(t, 2, body.Page)
	require.Equal(t, 2, body.PageSize)
	require.Len(t, body.Data, 1)
	require.Equal(t, "carol", body.Data[0].Username)

	// Search narrows to matching members and keeps the total in sync.
	w = doRequest(router, http.MethodGet, target+"?search=bob", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body = decodeMembers(t, w.Body.Bytes())
	require.Equal(t, int64(1), body.Total)
	require.Len(t, body.Data, 1)
	require.Equal(t, "bob", body.Data[0].Username)

	// Search also matches the email column.
	w = doRequest(router, http.MethodGet, target+"?search="+url.QueryEscape("bob@example.com"), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body = decodeMembers(t, w.Body.Bytes())
	require.Equal(t, int64(1), body.Total)
	require.Len(t, body.Data, 1)
	require.Equal(t, "bob", body.Data[0].Username)

	// A non-member matching the search still returns nothing.
	w = doRequest(router, http.MethodGet, target+"?search=dave", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body = decodeMembers(t, w.Body.Bytes())
	require.Equal(t, int64(0), body.Total)
	require.Empty(t, body.Data)
}

func TestAdminGroupMembers_StaffCanRead(t *testing.T) {
	setupAdminGroupsTestDB(t)
	staff := newStaffUser(t)
	group := models.AuthGroup{Name: "Judges"}
	require.NoError(t, db.DB.Create(&group).Error)

	router := authedRouter(staff.ID, http.MethodGet, "/admin/group/:id/members", AdminGroupMembers)
	w := doRequest(router, http.MethodGet, membersPath(group.ID), "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestAdminGroupMembers_GroupNotFound(t *testing.T) {
	setupAdminGroupsTestDB(t)
	root := newSuperuserUser(t)

	router := authedRouter(root.ID, http.MethodGet, "/admin/group/:id/members", AdminGroupMembers)
	w := doRequest(router, http.MethodGet, membersPath(999), "")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdminGroupAddMember_IdempotentAndValidated(t *testing.T) {
	setupAdminGroupsTestDB(t)
	root := newSuperuserUser(t)
	group := models.AuthGroup{Name: "Judges"}
	require.NoError(t, db.DB.Create(&group).Error)
	_, profile := newMemberUser(t, "alice", 0)

	router := authedRouter(root.ID, http.MethodPost, "/admin/group/:id/members", AdminGroupAddMember)
	target := membersPath(group.ID)
	body := `{"user_id":` + strconv.Itoa(int(profile.ID)) + `}`

	for i := 0; i < 2; i++ {
		w := doRequest(router, http.MethodPost, target, body)
		require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())
	}
	var count int64
	require.NoError(t, db.DB.Model(&models.AuthUserGroup{}).
		Where("group_id = ? AND user_id = ?", group.ID, profile.UserID).Count(&count).Error)
	require.Equal(t, int64(1), count, "duplicate add must stay idempotent")

	w := doRequest(router, http.MethodPost, target, `{"user_id":9999}`)
	require.Equal(t, http.StatusNotFound, w.Code)

	w = doRequest(router, http.MethodPost, target, `{}`)
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = doRequest(router, http.MethodPost, membersPath(999), body)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdminGroupRemoveMember_RemovesAndValidates(t *testing.T) {
	setupAdminGroupsTestDB(t)
	root := newSuperuserUser(t)
	group := models.AuthGroup{Name: "Judges"}
	require.NoError(t, db.DB.Create(&group).Error)
	member, profile := newMemberUser(t, "alice", 0)
	_, otherProfile := newMemberUser(t, "bob", 0) // exists, never a member
	require.NoError(t, db.DB.Create(&models.AuthUserGroup{UserID: member.ID, GroupID: group.ID}).Error)

	router := authedRouter(root.ID, http.MethodDelete, "/admin/group/:id/members/:userId", AdminGroupRemoveMember)
	target := membersPath(group.ID) + "/" + strconv.Itoa(int(profile.ID))

	w := doRequest(router, http.MethodDelete, target, "")
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())

	var count int64
	require.NoError(t, db.DB.Model(&models.AuthUserGroup{}).
		Where("group_id = ? AND user_id = ?", group.ID, member.ID).Count(&count).Error)
	require.Equal(t, int64(0), count)

	// Removing again is a no-op success.
	w = doRequest(router, http.MethodDelete, target, "")
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())

	// Unknown user.
	w = doRequest(router, http.MethodDelete, membersPath(group.ID)+"/9999", "")
	require.Equal(t, http.StatusNotFound, w.Code)

	// Unknown group.
	w = doRequest(router, http.MethodDelete, membersPath(999)+"/"+strconv.Itoa(int(otherProfile.ID)), "")
	require.Equal(t, http.StatusNotFound, w.Code)
}