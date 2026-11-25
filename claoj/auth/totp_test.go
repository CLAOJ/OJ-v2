package auth

import (
	"testing"

	"github.com/CLAOJ/claoj/config"
)

// Vector made by django-fernet-fields' scheme (HKDF + cryptography.Fernet)
// with SECRET_KEY "test-v1-secret".
func TestFernetDecryptV1(t *testing.T) {
	tok := "gAAAAABrB0tp9oHfDL5XQGFb0aNw2pPTDX4dSmvbfWs43PSB-a-v3JBjVVKnhynDX9nyhTpB3j-bEVetuvy3TXGCdv-MkdOBqZ0g0oom-FWCoFyPG54xmxk="
	got, err := fernetDecrypt(tok, "test-v1-secret")
	if err != nil || got != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := fernetDecrypt(tok, "wrong"); err == nil {
		t.Fatal("wrong key must fail")
	}
}

func TestTOTPChallenge(t *testing.T) {
	config.C.App.SecretKey = "k"
	tok := TOTPChallenge(7)
	if !ValidTOTPChallenge(tok, 7) {
		t.Fatal("own token rejected")
	}
	if ValidTOTPChallenge(tok, 8) || ValidTOTPChallenge(tok+"x", 7) || ValidTOTPChallenge("7.99999999999.abc", 7) {
		t.Fatal("bad token accepted")
	}
}
