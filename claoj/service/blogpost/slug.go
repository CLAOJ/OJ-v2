package blogpost

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Slugify makes a URL slug from a title the way v1's admin pre-fills it:
// "Thông báo: CLAOJ v2!" -> "thong-bao-claoj-v2". At most 100 runes
// (judge_blogpost.slug).
func Slugify(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range norm.NFD.String(strings.ToLower(title)) {
		switch {
		case unicode.Is(unicode.Mn, r): // accent marks split off by NFD
		case r == 'đ':
			b.WriteRune('d')
			dash = false
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.TrimRight(b.String(), "-")
	if len(s) > 100 {
		s = strings.TrimRight(s[:100], "-")
	}
	return s
}
