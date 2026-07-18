package auth

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// newActiveUser creates an active, non-superuser AuthUser.
func newActiveUser(t *testing.T, username string) models.AuthUser {
	t.Helper()
	user := models.AuthUser{Username: username, IsActive: true}
	require.NoError(t, db.DB.Create(&user).Error)
	return user
}

// newProfileFor creates a judge_profile row for the given user.
func newProfileFor(t *testing.T, userID uint) models.Profile {
	t.Helper()
	profile := models.Profile{UserID: userID, Timezone: "UTC"}
	require.NoError(t, db.DB.Create(&profile).Error)
	return profile
}

// grantPermsViaGroup seeds each codename as a judge.<codename> permission,
// puts them all in one group, and puts userID in that group -- mirroring
// how Task 2's TestResolve_GroupAndDirectPerms grants permissions.
func grantPermsViaGroup(t *testing.T, userID uint, codenames []string) {
	t.Helper()
	if len(codenames) == 0 {
		return
	}
	group := models.AuthGroup{Name: "test-group"}
	require.NoError(t, db.DB.Create(&group).Error)
	for _, cn := range codenames {
		permID := seedPerm(t, cn)
		require.NoError(t, db.DB.Create(&models.AuthGroupPermission{GroupID: group.ID, PermissionID: permID}).Error)
	}
	require.NoError(t, db.DB.Create(&models.AuthUserGroup{UserID: userID, GroupID: group.ID}).Error)
}

// ginContextForUser builds a gin context authenticated as userID.
func ginContextForUser(userID uint) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	return c
}

// ginContextAnonymous builds a gin context with no user_id at all.
func ginContextAnonymous() *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c
}

func TestCanEditProblem_DjangoSemantics(t *testing.T) {
	// mirrors OJ/judge/models/problem.py:229-238
	cases := []struct {
		name     string
		perms    []string // judge.* codenames granted via a group
		isAuthor bool
		isPublic bool
		want     bool
	}{
		{"no perms at all", nil, true, false, false},                                        // author but lacks edit_own_problem
		{"edit_own only, not author", []string{"edit_own_problem"}, false, false, false},
		{"edit_own + author", []string{"edit_own_problem"}, true, false, true},
		{"edit_all, not author", []string{"edit_own_problem", "edit_all_problem"}, false, false, true},
		{"edit_public on public", []string{"edit_own_problem", "edit_public_problem"}, false, true, true},
		{"edit_public on private", []string{"edit_own_problem", "edit_public_problem"}, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupPermsDB(t)

			user := newActiveUser(t, "user")
			profile := newProfileFor(t, user.ID)

			problem := models.Problem{Code: "code", Name: "Problem", IsPublic: tc.isPublic}
			if tc.isAuthor {
				problem.Authors = []models.Profile{profile}
			}
			require.NoError(t, db.DB.Create(&problem).Error)

			grantPermsViaGroup(t, user.ID, tc.perms)
			c := ginContextForUser(user.ID)

			require.Equal(t, tc.want, CanEditProblem(c, &problem))
		})
	}
}

func TestCanViewProblem(t *testing.T) {
	// mirrors the codename-relevant parts of OJ/judge/models/problem.py:240-284
	cases := []struct {
		name      string
		isPublic  bool
		anonymous bool
		perms     []string
		isAuthor  bool
		isTester  bool
		want      bool
	}{
		{"public problem visible to anonymous", true, true, nil, false, false, true},
		{"public problem visible to plain user", true, false, nil, false, false, true},
		{"hidden problem invisible to plain user", false, false, nil, false, false, false},
		{"hidden problem visible with see_private_problem", false, false, []string{"see_private_problem"}, false, false, true},
		{"hidden problem visible to author with edit_own_problem", false, false, []string{"edit_own_problem"}, true, false, true},
		{"hidden problem visible to tester", false, false, nil, false, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupPermsDB(t)

			problem := models.Problem{Code: "code", Name: "Problem", IsPublic: tc.isPublic}

			var c *gin.Context
			if tc.anonymous {
				c = ginContextAnonymous()
			} else {
				user := newActiveUser(t, "user")
				profile := newProfileFor(t, user.ID)
				if tc.isAuthor {
					problem.Authors = []models.Profile{profile}
				}
				if tc.isTester {
					problem.Testers = []models.Profile{profile}
				}
				grantPermsViaGroup(t, user.ID, tc.perms)
				c = ginContextForUser(user.ID)
			}
			require.NoError(t, db.DB.Create(&problem).Error)

			require.Equal(t, tc.want, CanViewProblem(c, &problem))
		})
	}
}

func TestCanEditContest(t *testing.T) {
	// mirrors OJ/judge/models/contest.py:380-389
	cases := []struct {
		name     string
		perms    []string
		isAuthor bool
		want     bool
	}{
		{"edit_all_contest grants access without authorship", []string{"edit_all_contest"}, false, true},
		{"edit_own_contest plus authorship grants access", []string{"edit_own_contest"}, true, true},
		{"edit_own_contest alone (not an editor) denies access", []string{"edit_own_contest"}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupPermsDB(t)

			user := newActiveUser(t, "user")
			profile := newProfileFor(t, user.ID)

			contest := models.Contest{Key: "c1", Name: "Contest"}
			if tc.isAuthor {
				contest.Authors = []models.Profile{profile}
			}
			require.NoError(t, db.DB.Create(&contest).Error)

			grantPermsViaGroup(t, user.ID, tc.perms)
			c := ginContextForUser(user.ID)

			require.Equal(t, tc.want, CanEditContest(c, &contest))
		})
	}
}

func TestCanViewContest(t *testing.T) {
	// mirrors the codename-relevant parts of OJ/judge/models/contest.py:321-370
	cases := []struct {
		name      string
		isVisible bool
		perms     []string
		isEditor  bool // author/curator
		isTester  bool
		want      bool
	}{
		{"visible contest is visible to anyone", true, nil, false, false, true},
		{"hidden contest invisible to plain user", false, nil, false, false, false},
		{"hidden contest visible with see_private_contest", false, []string{"see_private_contest"}, false, false, true},
		{"hidden contest visible with edit_all_contest", false, []string{"edit_all_contest"}, false, false, true},
		{"hidden contest visible to editor with no perms", false, nil, true, false, true},
		{"hidden contest visible to tester with no perms", false, nil, false, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupPermsDB(t)

			user := newActiveUser(t, "user")
			profile := newProfileFor(t, user.ID)

			contest := models.Contest{Key: "c1", Name: "Contest", IsVisible: tc.isVisible}
			if tc.isEditor {
				contest.Authors = []models.Profile{profile}
			}
			if tc.isTester {
				contest.Testers = []models.Profile{profile}
			}
			require.NoError(t, db.DB.Create(&contest).Error)

			grantPermsViaGroup(t, user.ID, tc.perms)
			c := ginContextForUser(user.ID)

			require.Equal(t, tc.want, CanViewContest(c, &contest))
		})
	}
}

func TestCanViewSolution(t *testing.T) {
	// mirrors OJ/judge/models/problem.py:630-637
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	cases := []struct {
		name           string
		solution       models.Solution
		seePrivate     bool
		editableByUser bool // author + edit_own_problem
		want           bool
	}{
		{
			name:     "public and already published solution is visible",
			solution: models.Solution{IsPublic: true, PublishOn: &past},
			want:     true,
		},
		{
			name:       "not-yet-published solution needs see_private_solution",
			solution:   models.Solution{IsPublic: true, PublishOn: &future},
			seePrivate: true,
			want:       true,
		},
		{
			name:           "unpublished solution visible to a problem editor",
			solution:       models.Solution{IsPublic: false, PublishOn: nil},
			editableByUser: true,
			want:           true,
		},
		{
			name:     "unpublished solution invisible to a plain user",
			solution: models.Solution{IsPublic: false, PublishOn: nil},
			want:     false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupPermsDB(t)

			user := newActiveUser(t, "user")
			profile := newProfileFor(t, user.ID)

			problem := models.Problem{Code: "code", Name: "Problem"}
			if tc.editableByUser {
				problem.Authors = []models.Profile{profile}
			}
			require.NoError(t, db.DB.Create(&problem).Error)

			var perms []string
			if tc.seePrivate {
				perms = append(perms, "see_private_solution")
			}
			if tc.editableByUser {
				perms = append(perms, "edit_own_problem")
			}
			grantPermsViaGroup(t, user.ID, perms)
			c := ginContextForUser(user.ID)

			solution := tc.solution
			require.Equal(t, tc.want, CanViewSolution(c, &solution, &problem))
		})
	}
}

func TestCanRejudge(t *testing.T) {
	// mirrors OJ/judge/models/problem.py:286-287
	cases := []struct {
		name        string
		superuser   bool
		rejudgePerm bool
		editable    bool // author + edit_own_problem
		want        bool
	}{
		{"rejudge_submission plus an editable problem grants access", false, true, true, true},
		{"rejudge_submission alone (not editable) denies access", false, true, false, false},
		{"editable alone (no rejudge_submission) denies access", false, false, true, false},
		{"superuser can always rejudge", true, false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupPermsDB(t)

			user := models.AuthUser{Username: "user", IsActive: true, IsSuperuser: tc.superuser}
			require.NoError(t, db.DB.Create(&user).Error)
			profile := newProfileFor(t, user.ID)

			problem := models.Problem{Code: "code", Name: "Problem"}
			if tc.editable {
				problem.Authors = []models.Profile{profile}
			}
			require.NoError(t, db.DB.Create(&problem).Error)

			var perms []string
			if tc.rejudgePerm {
				perms = append(perms, "rejudge_submission")
			}
			if tc.editable {
				perms = append(perms, "edit_own_problem")
			}
			grantPermsViaGroup(t, user.ID, perms)
			c := ginContextForUser(user.ID)

			require.Equal(t, tc.want, CanRejudge(c, &problem))
		})
	}
}
