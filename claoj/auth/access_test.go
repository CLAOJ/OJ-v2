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

func TestCanViewProblem_OrganizationPrivate(t *testing.T) {
	// mirrors the organization-private branch of Problem.is_accessible_by
	// (OJ/judge/models/problem.py:250-263): a public problem private to
	// organizations is only visible to members, holders of
	// see_organization_problem/see_private_problem, or editors/testers.
	cases := []struct {
		name      string
		userInOrg bool
		anonymous bool
		perms     []string
		isEditor  bool
		isTester  bool
		want      bool
	}{
		{"hidden from non-member", false, false, nil, false, false, false},
		{"hidden from anonymous", false, true, nil, false, false, false},
		{"shown to member", true, false, nil, false, false, true},
		{"shown with see_organization_problem", false, false, []string{"see_organization_problem"}, false, false, true},
		{"shown with see_private_problem", false, false, []string{"see_private_problem"}, false, false, true},
		{"shown to author", false, false, []string{"edit_own_problem"}, true, false, true},
		{"shown to tester", false, false, nil, false, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupPermsDB(t)

			org := models.Organization{Name: "Org", Slug: "org", ShortName: "ORG"}
			require.NoError(t, db.DB.Create(&org).Error)

			problem := models.Problem{
				Code: "code", Name: "Problem", IsPublic: true, IsOrganizationPrivate: true,
				Organizations: []models.Organization{org},
			}

			var c *gin.Context
			if tc.anonymous {
				require.NoError(t, db.DB.Create(&problem).Error)
				c = ginContextAnonymous()
			} else {
				user := newActiveUser(t, "user")
				profile := newProfileFor(t, user.ID)
				if tc.isEditor {
					problem.Authors = []models.Profile{profile}
				}
				if tc.isTester {
					problem.Testers = []models.Profile{profile}
				}
				require.NoError(t, db.DB.Create(&problem).Error)
				if tc.userInOrg {
					require.NoError(t, db.DB.Model(&profile).Association("Organizations").Append(&org))
				}
				grantPermsViaGroup(t, user.ID, tc.perms)
				c = ginContextForUser(user.ID)
			}

			require.Equal(t, tc.want, CanViewProblem(c, &problem))
		})
	}
}

func TestVisibleProblemFilter(t *testing.T) {
	// mirrors Problem.get_visible_problems (OJ/judge/models/problem.py:309-348).
	type fixture struct {
		orgPriv, hidden models.Problem
		org             models.Organization
	}
	seed := func(t *testing.T) fixture {
		t.Helper()
		f := fixture{}
		f.org = models.Organization{Name: "Org", Slug: "org", ShortName: "ORG"}
		require.NoError(t, db.DB.Create(&f.org).Error)
		pub := models.Problem{Code: "pub", Name: "Public", IsPublic: true}
		f.hidden = models.Problem{Code: "hidden", Name: "Hidden", IsPublic: false}
		f.orgPriv = models.Problem{Code: "org", Name: "Org", IsPublic: true, IsOrganizationPrivate: true,
			Organizations: []models.Organization{f.org}}
		require.NoError(t, db.DB.Create(&pub).Error)
		require.NoError(t, db.DB.Create(&f.hidden).Error)
		require.NoError(t, db.DB.Create(&f.orgPriv).Error)
		return f
	}
	visibleCodes := func(t *testing.T, c *gin.Context) map[string]bool {
		t.Helper()
		expr, args := VisibleProblemFilter(c)
		var got []models.Problem
		require.NoError(t, db.DB.Model(&models.Problem{}).Where(expr, args...).Find(&got).Error)
		codes := map[string]bool{}
		for _, p := range got {
			codes[p.Code] = true
		}
		return codes
	}

	t.Run("anonymous sees only fully public", func(t *testing.T) {
		setupPermsDB(t)
		seed(t)
		require.Equal(t, map[string]bool{"pub": true}, visibleCodes(t, ginContextAnonymous()))
	})

	t.Run("plain user sees only fully public", func(t *testing.T) {
		setupPermsDB(t)
		seed(t)
		user := newActiveUser(t, "plain")
		newProfileFor(t, user.ID)
		require.Equal(t, map[string]bool{"pub": true}, visibleCodes(t, ginContextForUser(user.ID)))
	})

	t.Run("org member also sees org-private problem", func(t *testing.T) {
		setupPermsDB(t)
		f := seed(t)
		user := newActiveUser(t, "member")
		profile := newProfileFor(t, user.ID)
		require.NoError(t, db.DB.Model(&profile).Association("Organizations").Append(&f.org))
		require.Equal(t, map[string]bool{"pub": true, "org": true}, visibleCodes(t, ginContextForUser(user.ID)))
	})

	t.Run("author also sees their hidden problem", func(t *testing.T) {
		setupPermsDB(t)
		f := seed(t)
		user := newActiveUser(t, "author")
		profile := newProfileFor(t, user.ID)
		require.NoError(t, db.DB.Model(&f.hidden).Association("Authors").Append(&profile))
		require.Equal(t, map[string]bool{"pub": true, "hidden": true}, visibleCodes(t, ginContextForUser(user.ID)))
	})

	t.Run("superuser sees everything", func(t *testing.T) {
		setupPermsDB(t)
		seed(t)
		user := models.AuthUser{Username: "root", IsActive: true, IsSuperuser: true}
		require.NoError(t, db.DB.Create(&user).Error)
		newProfileFor(t, user.ID)
		require.Equal(t, map[string]bool{"pub": true, "hidden": true, "org": true},
			visibleCodes(t, ginContextForUser(user.ID)))
	})
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

func TestCanViewContest_OrgAndPrivateMembership(t *testing.T) {
	// mirrors the organization-private / private-contestant branches of
	// Contest.access_check (OJ/judge/models/contest.py:347-370). A merely
	// visible contest must NOT be viewable when it is restricted to an
	// organization or a private-contestant list the user is not part of.
	cases := []struct {
		name        string
		orgPrivate  bool
		private     bool
		userInOrg   bool
		userPrivate bool // listed in private_contestants
		anonymous   bool
		perms       []string
		isEditor    bool // author
		want        bool
	}{
		{"org-private hides from non-member", true, false, false, false, false, nil, false, false},
		{"org-private shows to member", true, false, true, false, false, nil, false, true},
		{"org-private hides from anonymous", true, false, false, false, true, nil, false, false},
		{"org-private shown with see_private_contest", true, false, false, false, false, []string{"see_private_contest"}, false, true},
		{"org-private shown to editor despite non-membership", true, false, false, false, false, nil, true, true},
		{"private hides from non-contestant", false, true, false, false, false, nil, false, false},
		{"private shows to contestant", false, true, false, true, false, nil, false, true},
		{"org+private hides when only in org", true, true, true, false, false, nil, false, false},
		{"org+private hides when only a contestant", true, true, false, true, false, nil, false, false},
		{"org+private shows when in org and a contestant", true, true, true, true, false, nil, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupPermsDB(t)

			contest := models.Contest{
				Key: "c1", Name: "Contest", IsVisible: true,
				IsOrganizationPrivate: tc.orgPrivate, IsPrivate: tc.private,
			}
			require.NoError(t, db.DB.Create(&contest).Error)

			org := models.Organization{Name: "Org", Slug: "org", ShortName: "ORG"}
			require.NoError(t, db.DB.Create(&org).Error)
			if tc.orgPrivate {
				require.NoError(t, db.DB.Model(&contest).Association("Organizations").Append(&org))
			}

			var c *gin.Context
			if tc.anonymous {
				c = ginContextAnonymous()
			} else {
				user := newActiveUser(t, "user")
				profile := newProfileFor(t, user.ID)
				if tc.userInOrg {
					require.NoError(t, db.DB.Model(&profile).Association("Organizations").Append(&org))
				}
				if tc.userPrivate {
					require.NoError(t, db.DB.Model(&contest).Association("PrivateContestants").Append(&profile))
				}
				if tc.isEditor {
					require.NoError(t, db.DB.Model(&contest).Association("Authors").Append(&profile))
				}
				grantPermsViaGroup(t, user.ID, tc.perms)
				c = ginContextForUser(user.ID)
			}

			require.Equal(t, tc.want, CanViewContest(c, &contest))
		})
	}
}

func TestVisibleContestFilter(t *testing.T) {
	// mirrors Contest.get_visible_contests (OJ/judge/models/contest.py:391-417).
	// Seeds a fixed set of contests, then asserts which keys each persona's
	// filter expression selects.
	type fixture struct {
		orgPriv, priv, hidden models.Contest
		org                   models.Organization
	}
	seed := func(t *testing.T) fixture {
		t.Helper()
		f := fixture{}
		public := models.Contest{Key: "public", Name: "Public", IsVisible: true}
		f.hidden = models.Contest{Key: "hidden", Name: "Hidden", IsVisible: false}
		f.orgPriv = models.Contest{Key: "org", Name: "Org", IsVisible: true, IsOrganizationPrivate: true}
		f.priv = models.Contest{Key: "priv", Name: "Priv", IsVisible: true, IsPrivate: true}
		require.NoError(t, db.DB.Create(&public).Error)
		require.NoError(t, db.DB.Create(&f.hidden).Error)
		require.NoError(t, db.DB.Create(&f.orgPriv).Error)
		require.NoError(t, db.DB.Create(&f.priv).Error)
		f.org = models.Organization{Name: "Org", Slug: "org", ShortName: "ORG"}
		require.NoError(t, db.DB.Create(&f.org).Error)
		require.NoError(t, db.DB.Model(&f.orgPriv).Association("Organizations").Append(&f.org))
		return f
	}
	visibleKeys := func(t *testing.T, c *gin.Context) map[string]bool {
		t.Helper()
		expr, args := VisibleContestFilter(c)
		var got []models.Contest
		require.NoError(t, db.DB.Model(&models.Contest{}).Where(expr, args...).Find(&got).Error)
		keys := map[string]bool{}
		for _, ct := range got {
			keys[ct.Key] = true
		}
		return keys
	}

	t.Run("anonymous sees only public", func(t *testing.T) {
		setupPermsDB(t)
		seed(t)
		require.Equal(t, map[string]bool{"public": true}, visibleKeys(t, ginContextAnonymous()))
	})

	t.Run("plain authenticated user sees only public", func(t *testing.T) {
		setupPermsDB(t)
		seed(t)
		user := newActiveUser(t, "plain")
		newProfileFor(t, user.ID)
		require.Equal(t, map[string]bool{"public": true}, visibleKeys(t, ginContextForUser(user.ID)))
	})

	t.Run("org member also sees their org contest", func(t *testing.T) {
		setupPermsDB(t)
		f := seed(t)
		user := newActiveUser(t, "member")
		profile := newProfileFor(t, user.ID)
		require.NoError(t, db.DB.Model(&profile).Association("Organizations").Append(&f.org))
		require.Equal(t, map[string]bool{"public": true, "org": true}, visibleKeys(t, ginContextForUser(user.ID)))
	})

	t.Run("private contestant also sees their private contest", func(t *testing.T) {
		setupPermsDB(t)
		f := seed(t)
		user := newActiveUser(t, "contestant")
		profile := newProfileFor(t, user.ID)
		require.NoError(t, db.DB.Model(&f.priv).Association("PrivateContestants").Append(&profile))
		require.Equal(t, map[string]bool{"public": true, "priv": true}, visibleKeys(t, ginContextForUser(user.ID)))
	})

	t.Run("author also sees their hidden contest", func(t *testing.T) {
		setupPermsDB(t)
		f := seed(t)
		user := newActiveUser(t, "author")
		profile := newProfileFor(t, user.ID)
		require.NoError(t, db.DB.Model(&f.hidden).Association("Authors").Append(&profile))
		require.Equal(t, map[string]bool{"public": true, "hidden": true}, visibleKeys(t, ginContextForUser(user.ID)))
	})

	t.Run("superuser sees everything", func(t *testing.T) {
		setupPermsDB(t)
		seed(t)
		user := models.AuthUser{Username: "root", IsActive: true, IsSuperuser: true}
		require.NoError(t, db.DB.Create(&user).Error)
		newProfileFor(t, user.ID)
		require.Equal(t, map[string]bool{"public": true, "hidden": true, "org": true, "priv": true},
			visibleKeys(t, ginContextForUser(user.ID)))
	})

	t.Run("aliased contest table correlates correctly", func(t *testing.T) {
		// Exercises VisibleContestFilterFor with a non-default table name, as
		// used by the profile rating/contest-history queries (JOIN ... jc).
		setupPermsDB(t)
		f := seed(t)
		user := newActiveUser(t, "member")
		profile := newProfileFor(t, user.ID)
		require.NoError(t, db.DB.Model(&profile).Association("Organizations").Append(&f.org))

		expr, args := VisibleContestFilterFor(ginContextForUser(user.ID), "jc")
		var got []models.Contest
		require.NoError(t, db.DB.Table("judge_contest jc").Where(expr, args...).Find(&got).Error)
		keys := map[string]bool{}
		for _, ct := range got {
			keys[ct.Key] = true
		}
		require.Equal(t, map[string]bool{"public": true, "org": true}, keys)
	})
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
