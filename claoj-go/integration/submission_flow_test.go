// Package integration_test provides integration tests for submission flow.
package integration_test

import (
	"net/http"
	"testing"
	"time"

	v2 "github.com/CLAOJ/claoj-go/api/v2"
	authHandlers "github.com/CLAOJ/claoj-go/api/v2/auth"
	"github.com/CLAOJ/claoj-go/auth"
	"github.com/CLAOJ/claoj-go/integration"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/stretchr/testify/assert"
)

// TestSubmissionFlow_ViewProblemList tests browsing problems with pagination
func TestSubmissionFlow_ViewProblemList(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test problems
	integration.CreateTestProblem(testDB.DB, "TEST1", "Test Problem 1", true, 100)
	integration.CreateTestProblem(testDB.DB, "TEST2", "Test Problem 2", true, 200)
	integration.CreateTestProblem(testDB.DB, "TEST3", "Test Problem 3", false, 150) // Private problem

	// Setup router
	gin := integration.TestRouter()
	gin.GET("/problems", v2.ProblemList)

	// Get problem list
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "GET",
		Path:   "/problems",
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NotNil(t, resp.JSONBody["data"])

	problems := resp.JSONBody["data"].([]interface{})
	// Should only return public problems
	assert.Equal(t, 2, len(problems))
}

// TestSubmissionFlow_ViewProblemDetails tests viewing a single problem
func TestSubmissionFlow_ViewProblemDetails(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test problem
	problem := integration.CreateTestProblem(testDB.DB, "TEST1", "Test Problem 1", true, 100)

	// Setup router
	gin := integration.TestRouter()
	gin.GET("/problem/:code", v2.ProblemDetail)

	// Get problem details
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "GET",
		Path:   "/problem/TEST1",
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, float64(problem.ID), resp.JSONBody["id"].(float64))
	assert.Equal(t, "Test Problem 1", resp.JSONBody["name"])
	assert.Equal(t, "TEST1", resp.JSONBody["code"])
}

// TestSubmissionFlow_ViewProblemNotFound tests viewing a non-existent problem
func TestSubmissionFlow_ViewProblemNotFound(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Setup router
	gin := integration.TestRouter()
	gin.GET("/problem/:code", v2.ProblemDetail)

	// Get non-existent problem
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "GET",
		Path:   "/problem/NONEXISTENT",
	})

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

// TestSubmissionFlow_SubmitSolution_Success tests successful code submission
func TestSubmissionFlow_SubmitSolution_Success(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test user and problem
	_ = integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)
	integration.CreateTestProblem(testDB.DB, "TEST1", "Test Problem 1", true, 100)

	// Setup router with auth middleware
	gin := integration.TestRouter()
	gin.Use(auth.RequiredMiddleware())
	gin.POST("/problem/:code/submit", v2.Submit)

	// First login to get token
	loginGin := integration.TestRouter()
	loginGin.POST("/auth/login", authHandlers.Login)

	loginResp := integration.MakeRequest(t, loginGin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "testuser",
			"password": "Password123!",
		},
	})

	accessToken := loginResp.JSONBody["access_token"].(string)

	// Submit solution
	submitCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`

	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/problem/TEST1/submit",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{
			"language": "go",
			"source":   submitCode,
		},
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "submitted successfully", resp.JSONBody["message"])
	assert.NotNil(t, resp.JSONBody["submission_id"])
}

// TestSubmissionFlow_SubmitSolution_InvalidLanguage tests submission with invalid language
func TestSubmissionFlow_SubmitSolution_InvalidLanguage(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test user and problem
	_ = integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)
	integration.CreateTestProblem(testDB.DB, "TEST1", "Test Problem 1", true, 100)

	// Setup router with auth middleware
	gin := integration.TestRouter()
	gin.Use(auth.RequiredMiddleware())
	gin.POST("/problem/:code/submit", v2.Submit)

	// First login to get token
	loginGin := integration.TestRouter()
	loginGin.POST("/auth/login", authHandlers.Login)

	loginResp := integration.MakeRequest(t, loginGin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "testuser",
			"password": "Password123!",
		},
	})

	accessToken := loginResp.JSONBody["access_token"].(string)

	// Submit with invalid language
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/problem/TEST1/submit",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{
			"language": "invalid_language",
			"source":   "print('hello')",
		},
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "invalid language", resp.JSONBody["error"])
}

// TestSubmissionFlow_SubmitSolution_EmptySource tests submission with empty source
func TestSubmissionFlow_SubmitSolution_EmptySource(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test user and problem
	_ = integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)
	integration.CreateTestProblem(testDB.DB, "TEST1", "Test Problem 1", true, 100)

	// Setup router with auth middleware
	gin := integration.TestRouter()
	gin.Use(auth.RequiredMiddleware())
	gin.POST("/problem/:code/submit", v2.Submit)

	// First login to get token
	loginGin := integration.TestRouter()
	loginGin.POST("/auth/login", authHandlers.Login)

	loginResp := integration.MakeRequest(t, loginGin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "testuser",
			"password": "Password123!",
		},
	})

	accessToken := loginResp.JSONBody["access_token"].(string)

	// Submit with empty source
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/problem/TEST1/submit",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{
			"language": "go",
			"source":   "",
		},
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	// Error message format may vary depending on validation
	assert.NotNil(t, resp.JSONBody["error"])
}

// TestSubmissionFlow_SubmitSolution_Unauthorized tests submission without authentication
func TestSubmissionFlow_SubmitSolution_Unauthorized(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test problem
	integration.CreateTestProblem(testDB.DB, "TEST1", "Test Problem 1", true, 100)

	// Setup router with auth middleware
	gin := integration.TestRouter()
	gin.Use(auth.RequiredMiddleware())
	gin.POST("/problem/:code/submit", v2.Submit)

	// Submit without authentication
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/problem/TEST1/submit",
		Body: map[string]interface{}{
			"language": "go",
			"source":   "package main",
		},
	})

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

// TestSubmissionFlow_SubmitSolution_PrivateProblem tests submission to private problem
func TestSubmissionFlow_SubmitSolution_PrivateProblem(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test user
	_ = integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)
	// Create private problem
	integration.CreateTestProblem(testDB.DB, "PRIVATE1", "Private Problem", false, 100)

	// Setup router with auth middleware
	gin := integration.TestRouter()
	gin.Use(auth.RequiredMiddleware())
	gin.POST("/problem/:code/submit", v2.Submit)

	// First login to get token
	loginGin := integration.TestRouter()
	loginGin.POST("/auth/login", authHandlers.Login)

	loginResp := integration.MakeRequest(t, loginGin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "testuser",
			"password": "Password123!",
		},
	})

	accessToken := loginResp.JSONBody["access_token"].(string)

	// Submit to private problem
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/problem/PRIVATE1/submit",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{
			"language": "go",
			"source":   "package main",
		},
	})

	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Equal(t, "problem is not public", resp.JSONBody["error"])
}

// TestSubmissionFlow_ViewSubmissionResults tests viewing submission results
func TestSubmissionFlow_ViewSubmissionResults(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test user and problem
	user := integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)
	problem := integration.CreateTestProblem(testDB.DB, "TEST1", "Test Problem 1", true, 100)

	// Create a submission directly in DB
	status := "AC"
	zero := 0.0
	submission := models.Submission{
		UserID:     user.ID,
		ProblemID:  problem.ID,
		LanguageID: 1,
		Date:       time.Now(),
		Status:     "AC",
		Result:     &status,
		Time:       &zero,
		Memory:     &zero,
		Points:     &zero,
	}
	testDB.DB.Create(&submission)

	// Setup router with auth middleware
	gin := integration.TestRouter()
	gin.Use(auth.RequiredMiddleware())
	gin.GET("/submission/:id", v2.SubmissionDetail)

	// First login to get token
	loginGin := integration.TestRouter()
	loginGin.POST("/auth/login", authHandlers.Login)

	loginResp := integration.MakeRequest(t, loginGin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/auth/login",
		Body: map[string]interface{}{
			"username": "testuser",
			"password": "Password123!",
		},
	})

	accessToken := loginResp.JSONBody["access_token"].(string)

	// View submission
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "GET",
		Path:   "/submission/1",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, float64(submission.ID), resp.JSONBody["id"].(float64))
}
