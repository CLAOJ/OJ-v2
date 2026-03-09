// Package integration_test provides integration tests for contest flow.
package integration_test

import (
	"net/http"
	"testing"
	"time"

	contestHandlers "github.com/CLAOJ/claoj-go/api/v2/contest"
	v2 "github.com/CLAOJ/claoj-go/api/v2"
	authHandlers "github.com/CLAOJ/claoj-go/api/v2/auth"
	"github.com/CLAOJ/claoj-go/auth"
	"github.com/CLAOJ/claoj-go/integration"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/stretchr/testify/assert"
)

// TestContestFlow_ViewContestList tests browsing contests
func TestContestFlow_ViewContestList(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test contests
	now := time.Now()
	integration.CreateTestContest(testDB.DB, "contest1", "Contest 1", "Test contest 1", now.Add(-1*time.Hour), now.Add(2*time.Hour))
	integration.CreateTestContest(testDB.DB, "contest2", "Contest 2", "Test contest 2", now.Add(-1*time.Hour), now.Add(2*time.Hour))
	integration.CreateTestContest(testDB.DB, "contest3", "Private Contest", "Private contest", now.Add(-1*time.Hour), now.Add(2*time.Hour))
	// Mark contest3 as not visible
	testDB.DB.Model(&models.Contest{}).Where("key = ?", "contest3").Update("is_visible", false)

	// Setup router
	gin := integration.TestRouter()
	gin.GET("/contests", contestHandlers.ContestList)

	// Get contest list
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "GET",
		Path:   "/contests",
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NotNil(t, resp.JSONBody["results"])

	contests := resp.JSONBody["results"].([]interface{})
	// Should only return visible contests
	assert.Equal(t, 2, len(contests))
}

// TestContestFlow_JoinContest_Success tests joining a contest
func TestContestFlow_JoinContest_Success(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test user and contest
	_ = integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)
	now := time.Now()
	integration.CreateTestContest(testDB.DB, "contest1", "Test Contest", "A test contest", now.Add(-1*time.Hour), now.Add(2*time.Hour))

	// Setup router with auth middleware
	gin := integration.TestRouter()
	gin.Use(auth.RequiredMiddleware())
	gin.POST("/contest/:key/join", v2.ContestJoin)

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

	// Join contest
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/contest/contest1/join",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{},
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "successfully joined contest", resp.JSONBody["message"])
}

// TestContestFlow_JoinContest_AlreadyJoined tests joining a contest twice
func TestContestFlow_JoinContest_AlreadyJoined(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test user and contest
	_ = integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)
	now := time.Now()
	_ = integration.CreateTestContest(testDB.DB, "contest1", "Test Contest", "A test contest", now.Add(-1*time.Hour), now.Add(2*time.Hour))

	// Setup router with auth middleware
	gin := integration.TestRouter()
	gin.Use(auth.RequiredMiddleware())
	gin.POST("/contest/:key/join", v2.ContestJoin)

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

	// Join contest first time
	resp1 := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/contest/contest1/join",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{},
	})

	assert.Equal(t, http.StatusOK, resp1.Code)

	// Try to join again
	resp2 := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/contest/contest1/join",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{},
	})

	assert.Equal(t, http.StatusConflict, resp2.Code)
	assert.Equal(t, "already joined", resp2.JSONBody["error"])
}

// TestContestFlow_JoinContest_NotActive tests joining a contest before it starts
func TestContestFlow_JoinContest_NotActive(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test user and future contest
	_ = integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)
	now := time.Now()
	integration.CreateTestContest(testDB.DB, "future_contest", "Future Contest", "A future contest", now.Add(1*time.Hour), now.Add(3*time.Hour))

	// Setup router with auth middleware
	gin := integration.TestRouter()
	gin.Use(auth.RequiredMiddleware())
	gin.POST("/contest/:key/join", v2.ContestJoin)

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

	// Try to join before contest starts
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/contest/future_contest/join",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{},
	})

	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Equal(t, "contest is not active", resp.JSONBody["error"])
}

// TestContestFlow_ContestRanking tests viewing contest ranking
func TestContestFlow_ContestRanking(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test users
	user1 := integration.CreateTestUser(testDB.DB, "user1", "Password123!", true)
	user2 := integration.CreateTestUser(testDB.DB, "user2", "Password123!", true)

	// Create contest
	now := time.Now()
	contest := integration.CreateTestContest(testDB.DB, "contest1", "Test Contest", "A test contest", now.Add(-1*time.Hour), now.Add(2*time.Hour))

	// Get user profiles
	var profile1, profile2 models.Profile
	testDB.DB.Where("user_id = ?", user1.ID).First(&profile1)
	testDB.DB.Where("user_id = ?", user2.ID).First(&profile2)

	// Create contest participations
	part1 := models.ContestParticipation{
		ContestID: contest.ID,
		UserID:    profile1.ID,
		RealStart: now,
		Score:     100,
		Cumtime:   300, // 5 minutes
		Virtual:   0,
	}
	testDB.DB.Create(&part1)

	part2 := models.ContestParticipation{
		ContestID: contest.ID,
		UserID:    profile2.ID,
		RealStart: now,
		Score:     80,
		Cumtime:   600, // 10 minutes
		Virtual:   0,
	}
	testDB.DB.Create(&part2)

	// Setup router
	gin := integration.TestRouter()
	gin.GET("/contest/:key/ranking", v2.ContestRanking)

	// Get contest ranking
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "GET",
		Path:   "/contest/contest1/ranking",
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "contest1", resp.JSONBody["contest"])
	assert.NotNil(t, resp.JSONBody["rankings"])

	rankings := resp.JSONBody["rankings"].([]interface{})
	assert.Equal(t, 2, len(rankings))

	// First place should be user1 (higher score)
	firstRank := rankings[0].(map[string]interface{})
	assert.Equal(t, "user1", firstRank["username"])
	assert.Equal(t, float64(100), firstRank["score"])
}

// TestContestFlow_ContestRankingPDF tests viewing contest ranking PDF
func TestContestFlow_ContestRankingPDF(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test users
	user1 := integration.CreateTestUser(testDB.DB, "user1", "Password123!", true)

	// Create contest
	now := time.Now()
	contest := integration.CreateTestContest(testDB.DB, "contest1", "Test Contest", "A test contest", now.Add(-1*time.Hour), now.Add(2*time.Hour))

	// Get user profile
	var profile1 models.Profile
	testDB.DB.Where("user_id = ?", user1.ID).First(&profile1)

	// Create contest participation
	part1 := models.ContestParticipation{
		ContestID: contest.ID,
		UserID:    profile1.ID,
		RealStart: now,
		Score:     100,
		Cumtime:   300,
		Virtual:   0,
	}
	testDB.DB.Create(&part1)

	// Setup router
	gin := integration.TestRouter()
	gin.GET("/contest/:key/ranking/pdf", v2.ContestRankingPDF)

	// Get contest ranking PDF
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "GET",
		Path:   "/contest/contest1/ranking/pdf",
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "application/pdf", resp.Headers.Get("Content-Type"))
	assert.Contains(t, resp.Headers.Get("Content-Disposition"), "contest_contest1_scoreboard.pdf")
}

// TestContestFlow_VirtualParticipation tests starting a virtual participation
func TestContestFlow_VirtualParticipation(t *testing.T) {
	testDB := integration.SetupIntegrationDB(t)
	defer integration.CleanupDB(t, testDB)

	// Create test user and contest that has ended
	_ = integration.CreateTestUser(testDB.DB, "testuser", "Password123!", true)
	now := time.Now()
	integration.CreateTestContest(testDB.DB, "ended_contest", "Ended Contest", "A contest that ended", now.Add(-3*time.Hour), now.Add(-1*time.Hour))

	// Setup router with auth middleware
	gin := integration.TestRouter()
	gin.Use(auth.RequiredMiddleware())
	gin.POST("/contest/:key/join", v2.ContestJoin)

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

	// Start virtual participation (contest has ended)
	resp := integration.MakeRequest(t, gin, integration.HTTPRequest{
		Method: "POST",
		Path:   "/contest/ended_contest/join",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{
			"virtual": true,
		},
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "successfully started virtual participation", resp.JSONBody["message"])
	assert.True(t, resp.JSONBody["virtual"].(bool))
}
