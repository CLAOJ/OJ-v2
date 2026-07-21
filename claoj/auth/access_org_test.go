package auth

import (
	"testing"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/stretchr/testify/require"
)

func TestCanEditOrganization(t *testing.T) {
	cases := []struct {
		name       string
		superuser  bool
		perms      []string
		isOrgAdmin bool
		want       bool
	}{
		{"outsider denied", false, nil, false, false},
		{"org admin allowed", false, nil, true, true},
		{"edit_all_organization allowed", false, []string{"edit_all_organization"}, false, true},
		{"organization_admin perm allowed", false, []string{"organization_admin"}, false, true},
		{"superuser allowed", true, nil, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupPermsDB(t)

			org := models.Organization{Name: "Org", Slug: "org", ShortName: "ORG"}
			require.NoError(t, db.DB.Create(&org).Error)

			user := models.AuthUser{Username: "u", IsActive: true, IsSuperuser: tc.superuser}
			require.NoError(t, db.DB.Create(&user).Error)
			profile := newProfileFor(t, user.ID)
			if tc.isOrgAdmin {
				require.NoError(t, db.DB.Model(&org).Association("Admins").Append(&profile))
			}
			grantPermsViaGroup(t, user.ID, tc.perms)

			c := ginContextForUser(user.ID)
			require.Equal(t, tc.want, CanEditOrganization(c, org.ID))
		})
	}
}
