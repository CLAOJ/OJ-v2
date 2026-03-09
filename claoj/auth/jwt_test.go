package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/config"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	// Initialize test configuration
	config.C.App.JwtSecretKey = "test-secret-key-for-jwt-tokens-generation-minimum-32-characters"
}

func TestGenerateTokens(t *testing.T) {
	tests := []struct {
		name            string
		userID          uint
		username        string
		isAdmin         bool
		familyID        string
		extendedRefresh bool
		wantErr         bool
	}{
		{
			name:            "standard tokens (7-day refresh)",
			userID:          1,
			username:        "testuser",
			isAdmin:         false,
			familyID:        "",
			extendedRefresh: false,
			wantErr:         false,
		},
		{
			name:            "extended tokens (30-day refresh)",
			userID:          1,
			username:        "testuser",
			isAdmin:         true,
			familyID:        "",
			extendedRefresh: true,
			wantErr:         false,
		},
		{
			name:            "tokens with existing family ID",
			userID:          2,
			username:        "adminuser",
			isAdmin:         true,
			familyID:        "existing-family-id-123",
			extendedRefresh: false,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accessToken, refreshToken, familyID, err := GenerateTokens(tt.userID, tt.username, tt.isAdmin, tt.familyID, tt.extendedRefresh)

			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTokens() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Verify tokens are not empty
			if accessToken == "" {
				t.Error("GenerateTokens() accessToken is empty")
			}
			if refreshToken == "" {
				t.Error("GenerateTokens() refreshToken is empty")
			}
			if familyID == "" {
				t.Error("GenerateTokens() familyID is empty")
			}

			// Verify family ID is consistent
			if tt.familyID != "" && familyID != tt.familyID {
				t.Errorf("GenerateTokens() familyID = %v, want %v", familyID, tt.familyID)
			}

			// Verify token structure by parsing
			accessClaims, err := VerifyToken(accessToken, "access")
			if err != nil {
				t.Errorf("GenerateTokens() access token verification failed: %v", err)
			}

			refreshClaims, err := VerifyToken(refreshToken, "refresh")
			if err != nil {
				t.Errorf("GenerateTokens() refresh token verification failed: %v", err)
			}

			// Verify claims
			if accessClaims.UserID != tt.userID {
				t.Errorf("GenerateTokens() accessClaims.UserID = %v, want %v", accessClaims.UserID, tt.userID)
			}
			if accessClaims.Username != tt.username {
				t.Errorf("GenerateTokens() accessClaims.Username = %v, want %v", accessClaims.Username, tt.username)
			}
			if accessClaims.IsAdmin != tt.isAdmin {
				t.Errorf("GenerateTokens() accessClaims.IsAdmin = %v, want %v", accessClaims.IsAdmin, tt.isAdmin)
			}
			if accessClaims.FamilyID != familyID {
				t.Errorf("GenerateTokens() accessClaims.FamilyID = %v, want %v", accessClaims.FamilyID, familyID)
			}

			// Verify refresh token claims
			if refreshClaims.UserID != tt.userID {
				t.Errorf("GenerateTokens() refreshClaims.UserID = %v, want %v", refreshClaims.UserID, tt.userID)
			}
			if refreshClaims.FamilyID != familyID {
				t.Errorf("GenerateTokens() refreshClaims.FamilyID = %v, want %v", refreshClaims.FamilyID, familyID)
			}

			// Verify token expiration times
			accessExpiry := accessClaims.ExpiresAt.Time
			refreshExpiry := refreshClaims.ExpiresAt.Time

			// Access token should expire in ~15 minutes
			accessTTL := time.Until(accessExpiry)
			if accessTTL < 14*time.Minute || accessTTL > 16*time.Minute {
				t.Errorf("GenerateTokens() access token TTL = %v, expected ~15 minutes", accessTTL)
			}

			// Refresh token TTL depends on extendedRefresh flag
			refreshTTL := time.Until(refreshExpiry)
			if tt.extendedRefresh {
				// Should be ~30 days
				if refreshTTL < 29*24*time.Hour || refreshTTL > 31*24*time.Hour {
					t.Errorf("GenerateTokens() extended refresh token TTL = %v, expected ~30 days", refreshTTL)
				}
			} else {
				// Should be ~7 days
				if refreshTTL < 6*24*time.Hour || refreshTTL > 8*24*time.Hour {
					t.Errorf("GenerateTokens() standard refresh token TTL = %v, expected ~7 days", refreshTTL)
				}
			}
		})
	}
}

func TestVerifyToken(t *testing.T) {
	// Generate a valid token for testing
	accessToken, _, _, err := GenerateTokens(1, "testuser", false, "", false)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	tests := []struct {
		name            string
		token           string
		expectedSubject string
		wantErr         bool
		errContains     string
	}{
		{
			name:            "valid access token",
			token:           accessToken,
			expectedSubject: "access",
			wantErr:         false,
		},
		{
			name:            "invalid token format",
			token:           "invalid.token.here",
			expectedSubject: "access",
			wantErr:         true,
			errContains:     "token is malformed",
		},
		{
			name:            "empty token",
			token:           "",
			expectedSubject: "access",
			wantErr:         true,
			errContains:     "token is malformed",
		},
		{
			name:            "wrong subject type",
			token:           accessToken,
			expectedSubject: "refresh",
			wantErr:         true,
			errContains:     "invalid token type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := VerifyToken(tt.token, tt.expectedSubject)

			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && err != nil {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("VerifyToken() error = %v, should contain %v", err, tt.errContains)
					}
				}
				return
			}

			if claims == nil {
				t.Error("VerifyToken() claims is nil")
				return
			}

			if claims.Subject != tt.expectedSubject {
				t.Errorf("VerifyToken() claims.Subject = %v, want %v", claims.Subject, tt.expectedSubject)
			}
		})
	}
}

func TestVerifyToken_Expired(t *testing.T) {
	// Create an expired token manually using the same secret as config
	secret := []byte(config.C.App.JwtSecretKey)

	expiredClaims := &Claims{
		UserID:   1,
		Username: "testuser",
		IsAdmin:  false,
		FamilyID: "test-family",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Subject:   "access",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	_, err = VerifyToken(expiredTokenString, "access")
	if err == nil {
		t.Error("VerifyToken() should return error for expired token")
	}

	if !strings.Contains(err.Error(), "token has invalid claims") && !strings.Contains(err.Error(), "expired") {
		t.Errorf("VerifyToken() error = %v, expected token expired error", err)
	}
}

func TestVerifyToken_WrongSigningMethod(t *testing.T) {
	// Create a token with wrong signing method
	claims := &Claims{
		UserID:   1,
		Username: "testuser",
		IsAdmin:  false,
		FamilyID: "test-family",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			Subject:   "access",
		},
	}

	// Sign with RS256 (asymmetric) instead of HS256 (symmetric)
	// This will fail verification since we expect HS256
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	_, err = VerifyToken(tokenString, "access")
	if err == nil {
		t.Error("VerifyToken() should return error for wrong signing method")
	}
}

func TestGenerateFamilyID(t *testing.T) {
	// Generate multiple family IDs and ensure they are unique
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		_, _, familyID, err := GenerateTokens(1, "testuser", false, "", false)
		if err != nil {
			t.Fatalf("GenerateTokens() error = %v", err)
		}

		if ids[familyID] {
			t.Errorf("GenerateTokens() generated duplicate familyID: %v", familyID)
		}
		ids[familyID] = true
	}
}

func TestClaimsStructure(t *testing.T) {
	accessToken, refreshToken, familyID, err := GenerateTokens(42, "johndoe", true, "", false)
	if err != nil {
		t.Fatalf("GenerateTokens() error = %v", err)
	}

	// Parse and verify access token claims
	accessClaims, err := VerifyToken(accessToken, "access")
	if err != nil {
		t.Fatalf("VerifyToken(access) error = %v", err)
	}

	if accessClaims.UserID != 42 {
		t.Errorf("accessClaims.UserID = %v, want 42", accessClaims.UserID)
	}
	if accessClaims.Username != "johndoe" {
		t.Errorf("accessClaims.Username = %v, want johndoe", accessClaims.Username)
	}
	if !accessClaims.IsAdmin {
		t.Error("accessClaims.IsAdmin should be true")
	}
	if accessClaims.FamilyID != familyID {
		t.Errorf("accessClaims.FamilyID = %v, want %v", accessClaims.FamilyID, familyID)
	}

	// Parse and verify refresh token claims
	refreshClaims, err := VerifyToken(refreshToken, "refresh")
	if err != nil {
		t.Fatalf("VerifyToken(refresh) error = %v", err)
	}

	if refreshClaims.UserID != 42 {
		t.Errorf("refreshClaims.UserID = %v, want 42", refreshClaims.UserID)
	}
	if refreshClaims.FamilyID != familyID {
		t.Errorf("refreshClaims.FamilyID = %v, want %v", refreshClaims.FamilyID, familyID)
	}

	// IssuedAt should be set and recent
	if accessClaims.IssuedAt == nil {
		t.Error("accessClaims.IssuedAt is nil")
	} else {
		issuedAgo := time.Since(accessClaims.IssuedAt.Time)
		if issuedAgo > 5*time.Second {
			t.Errorf("accessClaims.IssuedAt is too old: %v", issuedAgo)
		}
	}
}

func TestExtendedRefreshVsStandard(t *testing.T) {
	userID := uint(1)
	username := "testuser"
	isAdmin := false

	// Generate standard tokens (7 days)
	_, standardRefresh, _, err := GenerateTokens(userID, username, isAdmin, "", false)
	if err != nil {
		t.Fatalf("GenerateTokens(standard) error = %v", err)
	}

	// Generate extended tokens (30 days)
	_, extendedRefresh, _, err := GenerateTokens(userID, username, isAdmin, "", true)
	if err != nil {
		t.Fatalf("GenerateTokens(extended) error = %v", err)
	}

	// Parse and compare expiration
	standardClaims, err := VerifyToken(standardRefresh, "refresh")
	if err != nil {
		t.Fatalf("VerifyToken(standard) error = %v", err)
	}

	extendedClaims, err := VerifyToken(extendedRefresh, "refresh")
	if err != nil {
		t.Fatalf("VerifyToken(extended) error = %v", err)
	}

	standardTTL := time.Until(standardClaims.ExpiresAt.Time)
	extendedTTL := time.Until(extendedClaims.ExpiresAt.Time)

	// Extended should be approximately 23 days longer than standard
	ttlDiff := extendedTTL - standardTTL
	expectedDiff := 23 * 24 * time.Hour // 30 - 7 = 23 days

	// Allow 1 day tolerance for the comparison
	if ttlDiff < expectedDiff-24*time.Hour || ttlDiff > expectedDiff+24*time.Hour {
		t.Errorf("TTL difference = %v, expected approximately %v", ttlDiff, expectedDiff)
	}
}
