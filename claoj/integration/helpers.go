// Package integration provides end-to-end integration tests for critical user journeys.
// These tests verify full workflows across multiple API endpoints.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLAOJ/claoj/auth"
	"github.com/CLAOJ/claoj/config"
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/jobs"
	"github.com/CLAOJ/claoj/models"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// init() sets up test configuration
func init() {
	config.C.App.JwtSecretKey = "test-secret-key-for-jwt-tokens-generation-minimum-32-characters"
	config.C.App.SecretKey = "test-secret-key-for-encryption-32-characters"
	config.C.App.SiteFullURL = "http://localhost:3000"
	config.C.Email.FromName = "CLAOJ Test"
	config.C.App.DefaultLanguage = "go"
}

// MockAsynqClient is a mock implementation for tests
type MockAsynqClient struct{}

// Enqueue mocks the asynq client Enqueue method
func (m *MockAsynqClient) Enqueue(task interface{}, opts ...interface{}) (interface{}, error) {
	// Return a mock info object, don't actually queue anything
	return struct{ ID string }{ID: "mock-task-id"}, nil
}

// TestDB holds the test database instance
type TestDB struct {
	DB *gorm.DB
}

// SetupIntegrationDB creates an in-memory SQLite database for integration testing.
// It migrates all tables needed for integration tests and seeds default data.
func SetupIntegrationDB(t *testing.T) *TestDB {
	t.Helper()

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate all tables needed for integration tests
	err = database.AutoMigrate(
		&models.AuthUser{},
		&models.Profile{},
		&models.Language{},
		&models.Judge{},
		&models.ProblemGroup{},
		&models.ProblemType{},
		&models.License{},
		&models.Problem{},
		&models.Submission{},
		&models.SubmissionSource{},
		&models.SubmissionTestCase{},
		&models.Contest{},
		&models.ContestProblem{},
		&models.ContestParticipation{},
		&models.ContestSubmission{},
		&models.RefreshToken{},
		&models.EmailVerificationToken{},
		&models.TotpDevice{},
		&models.Organization{},
		&models.Role{},
	)
	require.NoError(t, err)

	// Seed default data
	seedDefaultData(t, database)

	// Set global DB for handlers
	db.DB = database

	// Set mock asynq client for jobs package
	setupMockAsynq()

	return &TestDB{DB: database}
}

// setupMockAsynq sets up a mock enqueue function for the jobs package
func setupMockAsynq() {
	// Import jobs package and set mock enqueue
	jobs.SetMockEnqueue(func(task *asynq.Task, queueOpt asynq.Option) (*asynq.TaskInfo, error) {
		// Mock successful enqueue
		return &asynq.TaskInfo{ID: "mock-task-id"}, nil
	})
}

// seedDefaultData inserts essential test data
func seedDefaultData(t *testing.T, database *gorm.DB) {
	t.Helper()

	// Create default language
	defaultLang := models.Language{
		ID:            1,
		Key:           "go",
		Name:          "Go 1.x",
		CommonName:    "Go",
		Ace:           "golang",
		Pygments:      "go",
		Extension:     "go",
		Template:      "package main\n\nfunc main() {\n}",
		Description:   "Go programming language",
		FileSizeLimit: 65536,
	}
	database.Create(&defaultLang)

	// Create problem group
	defaultGroup := models.ProblemGroup{
		ID:       1,
		Name:     "default",
		FullName: "Default Group",
	}
	database.Create(&defaultGroup)

	// Create a default license
	defaultLicense := models.License{
		ID:      1,
		Key:     "mit",
		Link:    "https://opensource.org/licenses/MIT",
		Name:    "MIT License",
		Display: "MIT",
		Text:    "MIT License Text",
	}
	database.Create(&defaultLicense)
}

// CreateTestUser creates a test user with the given options.
// Returns the created user and password hash.
func CreateTestUser(db *gorm.DB, username string, password string, isActive bool) models.AuthUser {
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		panic(fmt.Sprintf("Failed to hash password: %v", err))
	}

	user := models.AuthUser{
		Username:    username,
		Email:       fmt.Sprintf("%s@example.com", username),
		Password:    hashedPassword,
		IsActive:    isActive,
		IsStaff:     false,
		IsSuperuser: false,
		DateJoined:  time.Now(),
	}
	db.Create(&user)

	// Update IsActive if it should be false (GORM ignores bool zero values with defaults)
	if !isActive {
		db.Model(&user).Update("is_active", false)
		// Refresh the user object
		db.First(&user, user.ID)
	}

	// Create profile
	profile := models.Profile{
		UserID:      user.ID,
		Timezone:    "UTC",
		LanguageID:  1,
		LastAccess:  time.Now(),
		DisplayRank: "user",
		AceTheme:    "auto",
		SiteTheme:   "auto",
		MathEngine:  "TeX",
	}
	db.Create(&profile)

	return user
}

// CreateTestProblem creates a test problem with the given options.
func CreateTestProblem(db *gorm.DB, code, name string, isPublic bool, points float64) models.Problem {
	problem := models.Problem{
		Code:         code,
		Name:         name,
		Description:  "This is a test problem description.",
		Summary:      "Test problem summary",
		GroupID:      1,
		TimeLimit:    1.0,
		MemoryLimit:  256,
		Points:       points,
		Partial:      true,
		IsPublic:     isPublic,
		ShortCircuit: false,
		LicenseID:    func() *uint { i := uint(1); return &i }(),
	}
	db.Create(&problem)
	return problem
}

// CreateTestContest creates a test contest with the given time range.
func CreateTestContest(db *gorm.DB, key, name, description string, startTime, endTime time.Time) models.Contest {
	contest := models.Contest{
		Key:                  key,
		Name:                 name,
		Description:          description,
		StartTime:            startTime,
		EndTime:              endTime,
		IsVisible:            true,
		IsRated:              false,
		ScoreboardVisibility: "V",
		UseClarifications:    true,
		PushAnnouncements:    false,
		RateAll:              false,
		IsPrivate:            false,
		HideProblemTags:      false,
		HideProblemAuthors:   false,
		RunPretestsOnly:      false,
		ShowShortDisplay:     false,
		IsOrganizationPrivate: false,
		FormatName:           "default",
		FormatConfig:         nil,
		ProblemLabelScript:   "",
		PointsPrecision:      3,
	}
	db.Create(&contest)
	return contest
}

// TestRouter creates a new Gin router configured for testing
func TestRouter(middleware ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	for _, m := range middleware {
		router.Use(m)
	}
	return router
}

// HTTPRequest is a helper for making HTTP requests in tests
type HTTPRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
}

// HTTPResponse holds the response from an HTTP request
type HTTPResponse struct {
	Code       int
	Headers    http.Header
	Body       []byte
	Cookies    []*http.Cookie
	JSONBody   map[string]interface{}
	StringBody string
}

// MakeRequest makes an HTTP request to the router and returns the response
func MakeRequest(t *testing.T, router *gin.Engine, req HTTPRequest) *HTTPResponse {
	t.Helper()

	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = json.Marshal(req.Body)
		require.NoError(t, err)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bytes.NewReader(bodyBytes))
	httpReq.Header.Set("Content-Type", "application/json")

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	resp := &HTTPResponse{
		Code:    w.Code,
		Headers: w.Header(),
		Body:    w.Body.Bytes(),
		Cookies: w.Result().Cookies(),
	}

	// Try to parse JSON
	if len(w.Body.Bytes()) > 0 {
		err := json.Unmarshal(w.Body.Bytes(), &resp.JSONBody)
		if err != nil {
			resp.StringBody = string(w.Body.Bytes())
		}
	}

	return resp
}

// GetCookie extracts a cookie by name from response cookies
func GetCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// AuthenticatedRequest creates an HTTP request with authentication headers
func AuthenticatedRequest(method, path string, accessToken string, body interface{}) HTTPRequest {
	req := HTTPRequest{
		Method: method,
		Path:   path,
		Body:   body,
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
	}
	return req
}

// CleanupDB clears all test data from the database
func CleanupDB(t *testing.T, testDB *TestDB) {
	t.Helper()

	// Delete in reverse order of dependencies
	testDB.DB.Exec("DELETE FROM judge_contestsubmission")
	testDB.DB.Exec("DELETE FROM judge_contestparticipation")
	testDB.DB.Exec("DELETE FROM judge_contestproblem")
	testDB.DB.Exec("DELETE FROM judge_contest")
	testDB.DB.Exec("DELETE FROM judge_submissiontestcase")
	testDB.DB.Exec("DELETE FROM judge_submissionsource")
	testDB.DB.Exec("DELETE FROM judge_submission")
	testDB.DB.Exec("DELETE FROM judge_problem")
	testDB.DB.Exec("DELETE FROM refresh_token")
	testDB.DB.Exec("DELETE FROM email_verification_token")
	testDB.DB.Exec("DELETE FROM judge_profile")
	testDB.DB.Exec("DELETE FROM auth_user")
}
