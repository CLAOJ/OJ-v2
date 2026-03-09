package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

var (
	ErrInvalidHashFormat = errors.New("invalid hash format")
	ErrUnsupportedHash   = errors.New("unsupported hash algorithm")
)

// CheckPassword checks a plaintext password against a Django pbkdf2_sha256 hash.
// Django password hashes look like: pbkdf2_sha256$iterations$salt$hash_base64
func CheckPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 {
		return false, ErrInvalidHashFormat
	}

	algorithm := parts[0]
	if algorithm != "pbkdf2_sha256" {
		return false, fmt.Errorf("%w: %s", ErrUnsupportedHash, algorithm)
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, fmt.Errorf("invalid iterations format: %w", err)
	}

	salt := []byte(parts[2])

	// Django uses base64 encoding without padding sometimes, or with padding.
	// We should decode the standard base64 string.
	expectedHashBytes, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		// Try raw encoding if standard fails (missing padding)
		expectedHashBytes, err = base64.RawStdEncoding.DecodeString(parts[3])
		if err != nil {
			return false, fmt.Errorf("invalid base64 hash: %w", err)
		}
	}

	// Django's default pbkdf2_sha256 uses 32 byte keys
	keyLen := len(expectedHashBytes)
	if keyLen == 0 {
		keyLen = 32
	}

	actualHashBytes := pbkdf2.Key([]byte(password), salt, iterations, keyLen, sha256.New)

	// Constant time compare to prevent timing attacks
	return subtle.ConstantTimeCompare(actualHashBytes, expectedHashBytes) == 1, nil
}

const (
	DefaultIterations = 600000 // Django 4.x default
	DefaultSaltLength = 12
)

// HashPassword generates a Django-compatible pbkdf2_sha256 hash.
func HashPassword(password string) (string, error) {
	salt := make([]byte, DefaultSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	// Use alphanumeric salt similar to Django
	saltStr := base64.RawStdEncoding.EncodeToString(salt)[:DefaultSaltLength]

	hashBytes := pbkdf2.Key([]byte(password), []byte(saltStr), DefaultIterations, 32, sha256.New)
	hashBase64 := base64.StdEncoding.EncodeToString(hashBytes)

	return fmt.Sprintf("pbkdf2_sha256$%d$%s$%s", DefaultIterations, saltStr, hashBase64), nil
}
