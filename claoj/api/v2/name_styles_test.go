package v2

import "testing"

// Styles end up inside CSS on every page, so anything that could break out of
// a selector or a content string must be rejected.
func TestValidNameStyles(t *testing.T) {
	if msg := validNameStyles(defaultNameStyles); msg != "" {
		t.Fatalf("defaults rejected: %s", msg)
	}
	bad := []map[string]nameStyle{
		{"user": {}},
		{"Staff": {}},
		{"a\"]{}": {}},
		{"staff": {Color: "red;}"}},
		{"staff": {Color: "#12345"}},
		{"staff": {Prefix: "123456789"}},
	}
	for _, s := range bad {
		if validNameStyles(s) == "" {
			t.Errorf("accepted %v", s)
		}
	}
}
