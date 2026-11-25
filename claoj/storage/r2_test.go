package storage

import "testing"

func TestPDFKey(t *testing.T) {
	cases := map[[2]string]string{
		{"/pdf/0a1b.pdf", "aplusb"}: "pdf/0a1b.pdf",
		{"statement.pdf", "aplusb"}: "problems/aplusb/statement.pdf",
	}
	for in, want := range cases {
		if got := PDFKey(in[0], in[1]); got != want {
			t.Errorf("PDFKey(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}

func TestDisabledWithoutConfig(t *testing.T) {
	if PublicEnabled() || PDFEnabled() {
		t.Fatal("R2 must be disabled when R2_* is unset")
	}
	if _, ok, err := PDFLink(t.Context(), "pdf/x.pdf", "x.pdf"); ok || err != nil {
		t.Fatalf("PDFLink without config = ok %v err %v, want fallback", ok, err)
	}
}
