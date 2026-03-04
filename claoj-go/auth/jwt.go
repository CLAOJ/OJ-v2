package auth

import (
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
	jwt.RegisteredClaims
}

// GenerateTokens creates an Access (15m) and Refresh (7d) token pair
func GenerateTokens(userID uint, username string, isAdmin bool) (accessToken string, refreshToken string, err error) {
	secret := []byte(config.C.App.SecretKey) // We reuse Django's SECRET_KEY for JWT signing
	if len(secret) == 0 {
		return "", "", errors.New("jwt secret key is not configured")
	}

	// 1. Access Token
	accessClaims := &Claims{
		UserID:   userID,
		Username: username,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "access",
		},
	}
	accToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accToken.SignedString(secret)
	if err != nil {
		return "", "", err
	}

	// 2. Refresh Token
	refreshClaims := &Claims{
		UserID:   userID,
		Username: username,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "refresh",
		},
	}
	refToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refToken.SignedString(secret)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// VerifyToken parses a token string and returns the custom Claims
func VerifyToken(tokenString string, expectedSubject string) (*Claims, error) {
	secret := []byte(config.C.App.SecretKey)

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the alg is what we expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
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
