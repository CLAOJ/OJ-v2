package v2

import "testing"

func TestCommentPageLink(t *testing.T) {
	for page, want := range map[string]string{
		"p:aplusb": "/problem/aplusb",
		"s:aplusb": "/problem/aplusb/editorial",
		"c:tet26":  "/contest/tet26",
		"b:51":     "/post/51",
		"weird":    "/",
	} {
		if got := commentLink(page); got != want {
			t.Errorf("commentLink(%q) = %q, want %q", page, got, want)
		}
	}
}
