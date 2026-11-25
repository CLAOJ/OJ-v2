package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"golang.org/x/crypto/hkdf"
)

// TOTPRequired reports whether a password login must be followed by a TOTP
// code: a confirmed v2 device, or 2FA enabled in v1 (judge_profile is shared).
func TOTPRequired(authUserID uint) bool {
	var n int64
	db.DB.Model(&models.TotpDevice{}).Where("user_id = ? AND confirmed = ?", authUserID, true).Count(&n)
	if n > 0 {
		return true
	}
	db.DB.Model(&models.Profile{}).Where("user_id = ? AND is_totp_enabled = ?", authUserID, true).Count(&n)
	return n > 0
}

const totpChallengeTTL = 5 * time.Minute

// TOTPChallenge is handed out after a correct password; the TOTP / backup-code
// step must present it, so a code alone never replaces the password.
func TOTPChallenge(authUserID uint) string {
	payload := fmt.Sprintf("%d.%d", authUserID, time.Now().Add(totpChallengeTTL).Unix())
	return payload + "." + totpChallengeMAC(payload)
}

// ValidTOTPChallenge checks the token is unexpired, untampered and for this user.
func ValidTOTPChallenge(token string, authUserID uint) bool {
	i := strings.LastIndexByte(token, '.')
	if i < 0 || !hmac.Equal([]byte(token[i+1:]), []byte(totpChallengeMAC(token[:i]))) {
		return false
	}
	uid, exp, ok := strings.Cut(token[:i], ".")
	expUnix, err := strconv.ParseInt(exp, 10, 64)
	return ok && err == nil && uid == strconv.FormatUint(uint64(authUserID), 10) && time.Now().Unix() < expUnix
}

func totpChallengeMAC(payload string) string {
	m := hmac.New(sha256.New, []byte("totp-challenge:"+config.C.App.SecretKey))
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// V1TOTPSecret returns the base32 secret v1 stored for this user, or "" when
// v1 2FA is off. v1 keeps it in judge_profile.totp_key as a
// django-fernet-fields token keyed by HKDF(SHA256) of v1's SECRET_KEY, which
// v2 gets as V1_SECRET_KEY.
func V1TOTPSecret(authUserID uint) (string, error) {
	var p models.Profile
	if err := db.DB.Where("user_id = ?", authUserID).First(&p).Error; err != nil {
		return "", err
	}
	if !p.IsTotpEnabled || p.TotpKey == nil || *p.TotpKey == "" {
		return "", nil
	}
	if config.C.App.V1SecretKey == "" {
		return "", errors.New("V1_SECRET_KEY not set; cannot read v1 2FA keys")
	}
	return fernetDecrypt(*p.TotpKey, config.C.App.V1SecretKey)
}

// fernetDecrypt mirrors django-fernet-fields (FERNET_USE_HKDF=True, no
// FERNET_KEYS): key = HKDF-SHA256(secret, salt, info), then a standard Fernet
// token: 0x80 | ts(8) | iv(16) | AES-128-CBC ciphertext | HMAC-SHA256(32).
func fernetDecrypt(token, secret string) (string, error) {
	key := make([]byte, 32)
	r := hkdf.New(sha256.New, []byte(secret), []byte("django-fernet-fields-hkdf-salt"), []byte("django-fernet-fields"))
	if _, err := io.ReadFull(r, key); err != nil {
		return "", err
	}
	raw, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}
	if len(raw) < 1+8+16+16+32 || raw[0] != 0x80 {
		return "", errors.New("not a fernet token")
	}
	body, mac := raw[:len(raw)-32], raw[len(raw)-32:]
	m := hmac.New(sha256.New, key[:16])
	m.Write(body)
	if !hmac.Equal(mac, m.Sum(nil)) {
		return "", errors.New("fernet: bad signature (wrong V1_SECRET_KEY?)")
	}
	iv, ct := body[9:25], body[25:]
	if len(ct)%aes.BlockSize != 0 {
		return "", errors.New("fernet: bad ciphertext")
	}
	block, err := aes.NewCipher(key[16:])
	if err != nil {
		return "", err
	}
	out := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, ct)
	pad := int(out[len(out)-1])
	if pad < 1 || pad > aes.BlockSize || pad > len(out) {
		return "", errors.New("fernet: bad padding")
	}
	return string(out[:len(out)-pad]), nil
}
