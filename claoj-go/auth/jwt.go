package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/CLAOJ/claoj-go/config"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

// Claims extends standard JWT claims with CLAOJ user properties
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	FamilyID string `json:"family_id"` // Token family for rotation
	jwt.RegisteredClaims
}

// generateFamilyID creates a unique family ID for token rotation tracking
func generateFamilyID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(bytes)
}

// GenerateTokens creates an Access (15m) and Refresh (7d) token pair
func GenerateTokens(userID uint, username string, isAdmin bool, familyID string) (accessToken string, refreshToken string, newFamilyID string, err error) {
	secret := []byte(config.C.App.JwtSecretKey)
	if len(secret) == 0 {
		return "", "", "", errors.New("jwt secret key is not configured")
	}

	// Generate new family ID if not provided
	if familyID == "" {
		familyID = generateFamilyID()
	}
	newFamilyID = familyID

	// 1. Access Token
	accessClaims := &Claims{
		UserID:   userID,
		Username: username,
		IsAdmin:  isAdmin,
		FamilyID: familyID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "access",
		},
	}
	accToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accToken.SignedString(secret)
	if err != nil {
		return "", "", "", err
	}

	// 2. Refresh Token
	refreshClaims := &Claims{
		UserID:   userID,
		Username: username,
		IsAdmin:  isAdmin,
		FamilyID: familyID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "refresh",
		},
	}
	refToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refToken.SignedString(secret)
	if err != nil {
		return "", "", "", err
	}

	return accessToken, refreshToken, newFamilyID, nil
}

// VerifyToken parses a token string and returns the custom Claims
func VerifyToken(tokenString string, expectedSubject string) (*Claims, error) {
	secret := []byte(config.C.App.JwtSecretKey)

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the alg is exactly HS256
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid signing method, expected HS256")
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Subject != expectedSubject {
		return nil, errors.New("invalid token type")
	}

	return claims, nil
}
