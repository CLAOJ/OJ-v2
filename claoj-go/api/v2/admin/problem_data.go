package admin

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TestCaseItem represents a test case in the response
type TestCaseItem struct {
	ID         uint   `json:"id"`
	Order      int    `json:"order"`
	Type       string `json:"type"`
	Points     *int   `json:"points"`
	IsPretest  bool   `json:"is_pretest"`
	InputFile  string `json:"input_file"`
	OutputFile string `json:"output_file"`
}

// AdminProblemData - GET /api/v2/admin/problem/:code/data
func AdminProblemData(c *gin.Context) {
	problemCode := c.Param("code")

	problem, err := GetProblemByCode(problemCode)
	if err != nil {
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

	items := make([]TestCaseItem, len(testCases))
	for i, tc := range testCases {
		items[i] = TestCaseItem{
			ID:         tc.ID,
			Order:      tc.Order,
			Type:       tc.Type,
			Points:     tc.Points,
			IsPretest:  tc.IsPretest,
			InputFile:  tc.InputFile,
			OutputFile: tc.OutputFile,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"problem":         problemCode,
		"has_data":        problemData.Zipfile != "",
		"checker":         problemData.Checker,
		"grader":          problemData.Grader,
		"generator":       problemData.Generator,
		"feedback":        problemData.Feedback,
		"checker_args":    problemData.CheckerArgs,
		"grader_args":     problemData.GraderArgs,
		"custom_checker":  problemData.CustomChecker,
		"custom_grader":   problemData.CustomGrader,
		"custom_header":   problemData.CustomHeader,
		"test_cases":      items,
		"test_case_count": len(testCases),
	})
}

// AdminProblemDataUploadRequest - POST /api/v2/admin/problem/:code/data
type AdminProblemDataUploadRequest struct {
	// Base64 encoded zip file containing test cases
	ZipData string `json:"zip_data"`
	// Or individual test cases
	TestCases []struct {
		Order      int    `json:"order" binding:"required"`
		Type       string `json:"type"` // C=Case, S=Sample
		InputData  string `json:"input_data"`  // base64
		OutputData string `json:"output_data"` // base64
		Points     *int   `json:"points"`
		IsPretest  bool   `json:"is_pretest"`
	} `json:"test_cases"`
	// Custom file uploads (base64)
	CustomCheckerData string `json:"custom_checker_data"`
	CustomGraderData  string `json:"custom_grader_data"`
	CustomHeaderData  string `json:"custom_header_data"`

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

	problem, err := GetProblemByCode(problemCode)
	if err != nil {
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

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// Upsert problem data
		problemData := models.ProblemData{
			ProblemID:     problem.ID,
			Checker:       req.Checker,
			Grader:        req.Grader,
			Generator:     req.Generator,
			Feedback:      req.Feedback,
			CheckerArgs:   req.CheckerArgs,
			GraderArgs:    req.GraderArgs,
			CustomChecker: req.CustomCheckerData,
			CustomGrader:  req.CustomGraderData,
			CustomHeader:  req.CustomHeaderData,
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

		// Data directory
		dataDir := filepath.Join("data", "problems", problemCode)
		testCaseDir := filepath.Join(dataDir, "testcase")
		if err := os.MkdirAll(testCaseDir, 0755); err != nil {
			return err
		}

		// Handle ZIP upload
		if req.ZipData != "" {
			if err := handleZipUpload(tx, req.ZipData, testCaseDir, problemData.ID, req.Checker, req.CheckerArgs, dataDir, req.Grader, req.GraderArgs); err != nil {
				return err
			}
		}

		// Handle individual test cases (only if no ZIP)
		if req.ZipData == "" && len(req.TestCases) > 0 {
			if err := handleIndividualTestCases(tx, req.TestCases, testCaseDir, problemData.ID, req.Checker, req.CheckerArgs); err != nil {
				return err
			}
		}

		// Handle custom file uploads
		handleCustomFileUploads(req.CustomCheckerData, req.CustomGraderData, req.CustomHeaderData, dataDir)

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

func handleZipUpload(tx *gorm.DB, zipData, testCaseDir string, problemDataID uint, checker, checkerArgs, dataDir, grader, graderArgs string) error {
	zipBytes, err := base64.StdEncoding.DecodeString(zipData)
	if err != nil {
		return fmt.Errorf("failed to decode zip data: %w", err)
	}

	archive, err := utils.ExtractZIP(zipBytes)
	if err != nil {
		return fmt.Errorf("failed to extract zip: %w", err)
	}

	// Delete existing test cases
	if err := tx.Where("dataset_id = ?", problemDataID).Delete(&models.ProblemTestCase{}).Error; err != nil {
		return err
	}

	// Save test cases from ZIP
	for i, tc := range archive.TestCases {
		inputFileName := filepath.Join(testCaseDir, fmt.Sprintf("%d.in", i))
		outputFileName := filepath.Join(testCaseDir, fmt.Sprintf("%d.out", i))

		if err := os.WriteFile(inputFileName, tc.InputData, 0600); err != nil {
			return err
		}
		if err := os.WriteFile(outputFileName, tc.OutputData, 0600); err != nil {
			return err
		}

		points := 0
		testCase := models.ProblemTestCase{
			DatasetID:   problemDataID,
			Order:       i,
			Type:        "C",
			Points:      &points,
			IsPretest:   false,
			InputFile:   fmt.Sprintf("%d.in", i),
			OutputFile:  fmt.Sprintf("%d.out", i),
			Checker:     checker,
			CheckerArgs: checkerArgs,
		}
		if err := tx.Create(&testCase).Error; err != nil {
			return err
		}
	}

	// Save special files
	if len(archive.Checker) > 0 {
		os.WriteFile(filepath.Join(dataDir, "checker.cpp"), archive.Checker, 0600)
	}
	if len(archive.Grader) > 0 {
		os.WriteFile(filepath.Join(dataDir, "grader.cpp"), archive.Grader, 0600)
	}
	if len(archive.Header) > 0 {
		os.WriteFile(filepath.Join(dataDir, "header.h"), archive.Header, 0600)
	}
	if len(archive.InitYML) > 0 {
		os.WriteFile(filepath.Join(dataDir, "init.yml"), archive.InitYML, 0600)
	} else {
		// Generate init.yml
		initYML := utils.GenerateInitYML(checker, checkerArgs, grader, graderArgs)
		os.WriteFile(filepath.Join(dataDir, "init.yml"), initYML, 0600)
	}

	return nil
}

func handleIndividualTestCases(tx *gorm.DB, testCases []struct {
	Order      int    `json:"order" binding:"required"`
	Type       string `json:"type"`
	InputData  string `json:"input_data"`
	OutputData string `json:"output_data"`
	Points     *int   `json:"points"`
	IsPretest  bool   `json:"is_pretest"`
}, testCaseDir string, problemDataID uint, checker, checkerArgs string) error {
	// Delete existing test cases
	if err := tx.Where("dataset_id = ?", problemDataID).Delete(&models.ProblemTestCase{}).Error; err != nil {
		return err
	}

	// Create test case files and records
	for i, tc := range testCases {
		inputFileName := filepath.Join(testCaseDir, fmt.Sprintf("%d.in", i))
		outputFileName := filepath.Join(testCaseDir, fmt.Sprintf("%d.out", i))

		// Decode and save input
		inputBytes, err := base64.StdEncoding.DecodeString(tc.InputData)
		if err != nil {
			return err
		}
		if err := os.WriteFile(inputFileName, inputBytes, 0600); err != nil {
			return err
		}

		// Decode and save output
		outputBytes, err := base64.StdEncoding.DecodeString(tc.OutputData)
		if err != nil {
			return err
		}
		if err := os.WriteFile(outputFileName, outputBytes, 0600); err != nil {
			return err
		}

		// Create database record
		testCase := models.ProblemTestCase{
			DatasetID:   problemDataID,
			Order:       tc.Order,
			Type:        tc.Type,
			Points:      tc.Points,
			IsPretest:   tc.IsPretest,
			InputFile:   fmt.Sprintf("%d.in", i),
			OutputFile:  fmt.Sprintf("%d.out", i),
			Checker:     checker,
			CheckerArgs: checkerArgs,
		}
		if err := tx.Create(&testCase).Error; err != nil {
			return err
		}
	}

	return nil
}

func handleCustomFileUploads(customCheckerData, customGraderData, customHeaderData, dataDir string) {
	if customCheckerData != "" {
		data, err := base64.StdEncoding.DecodeString(customCheckerData)
		if err == nil {
			os.WriteFile(filepath.Join(dataDir, "custom_checker.cpp"), data, 0600)
		}
	}
	if customGraderData != "" {
		data, err := base64.StdEncoding.DecodeString(customGraderData)
		if err == nil {
			os.WriteFile(filepath.Join(dataDir, "custom_grader.cpp"), data, 0600)
		}
	}
	if customHeaderData != "" {
		data, err := base64.StdEncoding.DecodeString(customHeaderData)
		if err == nil {
			os.WriteFile(filepath.Join(dataDir, "custom_header.h"), data, 0600)
		}
	}
}
