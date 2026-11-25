package v2

import "testing"

func TestProfilePrefErrors(t *testing.T) {
	s := func(v string) *string { return &v }
	for _, tc := range []struct {
		req  ProfileUpdateRequest
		want string
	}{
		{ProfileUpdateRequest{Timezone: s("Asia/Ho_Chi_Minh"), AceTheme: s("monokai"), SiteTheme: s("dark")}, ""},
		{ProfileUpdateRequest{Timezone: s("Mars/Olympus")}, "invalid timezone"},
		{ProfileUpdateRequest{Timezone: s("")}, "invalid timezone"},
		{ProfileUpdateRequest{AceTheme: s("hacker")}, "invalid editor theme"},
		{ProfileUpdateRequest{SiteTheme: s("sepia")}, "invalid site theme"},
	} {
		if got := profilePrefErrors(tc.req); got != tc.want {
			t.Errorf("%+v: got %q want %q", tc.req, got, tc.want)
		}
	}
}
