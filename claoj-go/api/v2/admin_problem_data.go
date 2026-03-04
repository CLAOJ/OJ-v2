package v2

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ProblemDataResponse - GET /api/v2/admin/problem/:code/data
func AdminProblemData(c *gin.Context) {
	problemCode := c.Param("code")

	var problem models.Problem
	if err := db.DB.Where("code = ?", problemCode).First(&problem).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	var problemData models.ProblemData
	if err := db.DB.Where("problem_id = ?", problem.ID).First(&problemData).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"data": nil, "message": "no problem data configured"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Get test cases
	var testCases []models.ProblemTestCase
	db.DB.Where("dataset_id = ?", problemData.ID).Order("`order` ASC").Find(&testCases)

	type TestCaseItem struct {
		ID        uint   `json:"id"`
		Order     int    `json:"order"`
		Type      string `json:"type"`
		Points    *int   `json:"points"`
		IsPretest bool   `json:"is_pretest"`
		InputFile string `json:"input_file"`
		OutputFile string `json:"output_file"`
	}

	items := make([]TestCaseItem, len(testCases))
	for i, tc := range testCases {
		items[i] = TestCaseItem{
			ID:        tc.ID,
			Order:     tc.Order,
			Type:      tc.Type,
			Points:    tc.Points,
			IsPretest: tc.IsPretest,
			InputFile: tc.InputFile,
			OutputFile: tc.OutputFile,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"problem":      problemCode,
		"has_data":     problemData.Zipfile != "",
		"checker":      problemData.Checker,
		"grader":       problemData.Grader,
		"generator":    problemData.Generator,
		"feedback":     problemData.Feedback,
		"test_cases":   items,
		"test_case_count": len(testCases),
	})
}

// AdminProblemDataUploadRequest - POST /api/v2/admin/problem/:code/data
type AdminProblemDataUploadRequest struct {
	// Base64 encoded zip file containing test cases
	ZipData string `json:"zip_data" binding:"required"`
	// Or individual test cases
	TestCases []struct {
		Order      int    `json:"order" binding:"required"`
		Type       string `json:"type"` // C=Case, S=Sample
		InputData  string `json:"input_data" binding:"required"`  // base64
		OutputData string `json:"output_data" binding:"required"` // base64
		Points     *int   `json:"points"`
		IsPretest  bool   `json:"is_pretest"`
	} `json:"test_cases"`
	Checker     string `json:"checker"`
	Grader      string `json:"grader"`
	Generator   string `json:"generator"`
	Feedback    string `json:"feedback"`
	CheckerArgs string `json:"checker_args"`
	GraderArgs  string `json:"grader_args"`
}

// AdminProblemDataUpload - POST /api/v2/admin/problem/:code/data
func AdminProblemDataUpload(c *gin.Context) {
	problemCode := c.Param("code")

	var problem models.Problem
	if err := db.DB.Where("code = ?", problemCode).First(&problem).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	var req AdminProblemDataUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// Upsert problem data
		problemData := models.ProblemData{
			ProblemID:   problem.ID,
			Checker:     req.Checker,
			Grader:      req.Grader,
			Generator:   req.Generator,
			Feedback:    req.Feedback,
			CheckerArgs: req.CheckerArgs,
			GraderArgs:  req.GraderArgs,
		}

		// Check if exists
		var existing models.ProblemData
		if err := tx.Where("problem_id = ?", problem.ID).First(&existing).Error; err == nil {
			// Update existing
			if err := tx.Model(&existing).Updates(problemData).Error; err != nil {
				return err
			}
			problemData.ID = existing.ID
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new
			if err := tx.Create(&problemData).Error; err != nil {
				return err
			}
		} else {
			return err
		}

		// Handle test cases
		if len(req.TestCases) > 0 {
			// Delete existing test cases
			if err := tx.Where("dataset_id = ?", problemData.ID).Delete(&models.ProblemTestCase{}).Error; err != nil {
				return err
			}

			// Data directory
			dataDir := filepath.Join("data", "problems", problemCode)
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return err
			}

			// Create test case files and records
			for i, tc := range req.TestCases {
				inputFileName := filepath.Join(dataDir, "testcase", fmt.Sprintf("%d.in", i))
				outputFileName := filepath.Join(dataDir, "testcase", fmt.Sprintf("%d.out", i))

				// Decode and save input
				inputBytes, err := base64.StdEncoding.DecodeString(tc.InputData)
				if err != nil {
					return err
				}
				if err := os.WriteFile(inputFileName, inputBytes, 0644); err != nil {
					return err
				}

				// Decode and save output
				outputBytes, err := base64.StdEncoding.DecodeString(tc.OutputData)
				if err != nil {
					return err
				}
				if err := os.WriteFile(outputFileName, outputBytes, 0644); err != nil {
					return err
				}

				// Create database record
				testCase := models.ProblemTestCase{
					DatasetID:   problemData.ID,
					Order:       tc.Order,
					Type:        tc.Type,
					Points:      tc.Points,
					IsPretest:   tc.IsPretest,
					InputFile:   fmt.Sprintf("%d.in", i),
					OutputFile:  fmt.Sprintf("%d.out", i),
					Checker:     req.Checker,
					CheckerArgs: req.CheckerArgs,
				}
				if err := tx.Create(&testCase).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "problem data updated successfully",
		"problem": problemCode,
	})
}

// AdminProblemDataDeleteTestCase - DELETE /api/v2/admin/problem/:code/data/testcase/:id
func AdminProblemDataDeleteTestCase(c *gin.Context) {
	problemCode := c.Param("code")
	testCaseID := c.Param("id")

	var problem models.Problem
	if err := db.DB.Where("code = ?", problemCode).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	var testCase models.ProblemTestCase
	if err := db.DB.Where("id = ? AND dataset_id IN (SELECT id FROM judge_problemdata WHERE problem_id = ?)", testCaseID, problem.ID).First(&testCase).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test case not found"})
		return
	}

	// Delete files
	dataDir := filepath.Join("data", "problems", problemCode, "testcase")
	inputPath := filepath.Join(dataDir, testCase.InputFile)
	outputPath := filepath.Join(dataDir, testCase.OutputFile)
	os.Remove(inputPath)
	os.Remove(outputPath)

	// Delete database record
	if err := db.DB.Delete(&testCase).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete test case"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test case deleted"})
}
