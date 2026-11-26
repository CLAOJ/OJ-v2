package blogpost

import "testing"

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"Thông báo: CLAOJ chuyển sang nền tảng mới": "thong-bao-claoj-chuyen-sang-nen-tang-moi",
		"  Đề thi -- HSG 2026!  ":                   "de-thi-hsg-2026",
		"???":                                       "",
	} {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
