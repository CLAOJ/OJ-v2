package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLAOJ/claoj-go/auth"
	"github.com/CLAOJ/claoj-go/config"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	config.C.App.JwtSecretKey = "test-secret-key-for-jwt-tokens-generation-minimum-32-characters"
	config.C.App.SecretKey = "test-secret-key-for-encryption-32-characters"
}

func setupMiddlewareTestDB(t *testing.T) *gorm.DB {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	database.AutoMigrate(&models.AuthUser{}, &models.Profile{}, &models.AuditLog{})
	return database
}

func createTestUser(t *testing.T, database *gorm.DB, username string, isAdmin bool) models.AuthUser {
	hashedPassword, _ := auth.HashPassword("password123")
	user := models.AuthUser{
		Username:    username,
		Email:       username + "@example.com",
		Password:    hashedPassword,
		IsActive:    true,
		IsStaff:     isAdmin,
		IsSuperuser: false,
		DateJoined:  time.Now(),
	}
	database.Create(&user)
	database.Create(&models.Profile{UserID: user.ID, Timezone: "UTC", LanguageID: 1})
	return user
}

func generateTestJWTToken(t *testing.T, userID uint, username string) string {
	accessToken, _, _, err := auth.GenerateTokens(userID, username, false, "", false)
	assert.NoError(t, err)
	return accessToken
}

func TestRequireUser_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupMiddlewareTestDB(t)
	db.DB = database

	user := createTestUser(t, database, "testuser", false)
	token := generateTestJWTToken(t, user.ID, user.Username)

	router := gin.New()
	router.Use(auth.RequiredMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		userID, exists := GetUser(c)
		assert.True(t, exists)
		assert.Equal(t, user.ID, userID)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireUser_Middleware_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupMiddlewareTestDB(t)
	db.DB = database

	router := gin.New()
	router.Use(auth.RequiredMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAdmin_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupMiddlewareTestDB(t)
	db.DB = database

	adminUser := createTestUser(t, database, "adminuser", true)
	adminToken := generateTestJWTToken(t, adminUser.ID, adminUser.Username)

	router := gin.New()
	router.Use(auth.RequiredMiddleware())
	router.Use(RequireAdmin())
	router.GET("/admin-only", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAdmin_Middleware_RegularUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupMiddlewareTestDB(t)
	db.DB = database

	regularUser := createTestUser(t, database, "regularuser", false)
	regularToken := generateTestJWTToken(t, regularUser.ID, regularUser.Username)

	router := gin.New()
	router.Use(auth.RequiredMiddleware())
	router.Use(RequireAdmin())
	router.GET("/admin-only", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetUser_ContextHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupMiddlewareTestDB(t)
	db.DB = database

	user := createTestUser(t, database, "testuser", false)
	token := generateTestJWTToken(t, user.ID, user.Username)

	router := gin.New()
	router.Use(auth.RequiredMiddleware())
	router.GET("/user-info", func(c *gin.Context) {
		userID, exists := GetUser(c)
		assert.True(t, exists)
		assert.Equal(t, user.ID, userID)
		username := GetUsername(c)
		assert.Equal(t, user.Username, username)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/user-info", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
