package v2

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/CLAOJ/claoj-go/utils"
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
		ID         uint   `json:"id"`
		Order      int    `json:"order"`
		Type       string `json:"type"`
		Points     *int   `json:"points"`
		IsPretest  bool   `json:"is_pretest"`
		InputFile  string `json:"input_file"`
		OutputFile string `json:"output_file"`
	}

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
			zipBytes, err := base64.StdEncoding.DecodeString(req.ZipData)
			if err != nil {
				return fmt.Errorf("failed to decode zip data: %w", err)
			}

			archive, err := utils.ExtractZIP(zipBytes)
			if err != nil {
				return fmt.Errorf("failed to extract zip: %w", err)
			}

			// Delete existing test cases
			if err := tx.Where("dataset_id = ?", problemData.ID).Delete(&models.ProblemTestCase{}).Error; err != nil {
				return err
			}

			// Save test cases from ZIP
			for i, tc := range archive.TestCases {
				inputFileName := filepath.Join(testCaseDir, fmt.Sprintf("%d.in", i))
				outputFileName := filepath.Join(testCaseDir, fmt.Sprintf("%d.out", i))

				if err := os.WriteFile(inputFileName, tc.InputData, 0644); err != nil {
					return err
				}
				if err := os.WriteFile(outputFileName, tc.OutputData, 0644); err != nil {
					return err
				}

				points := 0
				testCase := models.ProblemTestCase{
					DatasetID:   problemData.ID,
					Order:       i,
					Type:        "C",
					Points:      &points,
					IsPretest:   false,
					InputFile:   fmt.Sprintf("%d.in", i),
					OutputFile:  fmt.Sprintf("%d.out", i),
					Checker:     req.Checker,
					CheckerArgs: req.CheckerArgs,
				}
				if err := tx.Create(&testCase).Error; err != nil {
					return err
				}
			}

			// Save special files
			if len(archive.Checker) > 0 {
				os.WriteFile(filepath.Join(dataDir, "checker.cpp"), archive.Checker, 0644)
			}
			if len(archive.Grader) > 0 {
				os.WriteFile(filepath.Join(dataDir, "grader.cpp"), archive.Grader, 0644)
			}
			if len(archive.Header) > 0 {
				os.WriteFile(filepath.Join(dataDir, "header.h"), archive.Header, 0644)
			}
			if len(archive.InitYML) > 0 {
				os.WriteFile(filepath.Join(dataDir, "init.yml"), archive.InitYML, 0644)
			} else {
				// Generate init.yml
				initYML := utils.GenerateInitYML(req.Checker, req.CheckerArgs, req.Grader, req.GraderArgs)
				os.WriteFile(filepath.Join(dataDir, "init.yml"), initYML, 0644)
			}
		}

		// Handle individual test cases (only if no ZIP)
		if req.ZipData == "" && len(req.TestCases) > 0 {
			// Delete existing test cases
			if err := tx.Where("dataset_id = ?", problemData.ID).Delete(&models.ProblemTestCase{}).Error; err != nil {
				return err
			}

			// Create test case files and records
			for i, tc := range req.TestCases {
				inputFileName := filepath.Join(testCaseDir, fmt.Sprintf("%d.in", i))
				outputFileName := filepath.Join(testCaseDir, fmt.Sprintf("%d.out", i))

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

		// Handle custom file uploads
		if req.CustomCheckerData != "" {
			data, err := base64.StdEncoding.DecodeString(req.CustomCheckerData)
			if err == nil {
				os.WriteFile(filepath.Join(dataDir, "custom_checker.cpp"), data, 0644)
			}
		}
		if req.CustomGraderData != "" {
			data, err := base64.StdEncoding.DecodeString(req.CustomGraderData)
			if err == nil {
				os.WriteFile(filepath.Join(dataDir, "custom_grader.cpp"), data, 0644)
			}
		}
		if req.CustomHeaderData != "" {
			data, err := base64.StdEncoding.DecodeString(req.CustomHeaderData)
			if err == nil {
				os.WriteFile(filepath.Join(dataDir, "custom_header.h"), data, 0644)
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

// TestCaseReorderRequest - PATCH /api/v2/admin/problem/:code/data/reorder
type TestCaseReorderRequest struct {
	TestCases []struct {
		ID    uint `json:"id"`
		Order int  `json:"order"`
	} `json:"test_cases" binding:"required"`
}

// AdminProblemDataReorder - PATCH /api/v2/admin/problem/:code/data/reorder
func AdminProblemDataReorder(c *gin.Context) {
	problemCode := c.Param("code")

	var problem models.Problem
	if err := db.DB.Where("code = ?", problemCode).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	var req TestCaseReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// Validate all test cases belong to this problem's dataset
		var problemData models.ProblemData
		if err := tx.Where("problem_id = ?", problem.ID).First(&problemData).Error; err != nil {
			return fmt.Errorf("problem data not found")
		}

		for _, tc := range req.TestCases {
			var testCase models.ProblemTestCase
			if err := tx.Where("id = ? AND dataset_id = ?", tc.ID, problemData.ID).First(&testCase).Error; err != nil {
				return fmt.Errorf("test case %d not found in this problem", tc.ID)
			}
		}

		// Update orders
		for _, tc := range req.TestCases {
			if err := tx.Model(&models.ProblemTestCase{}).Where("id = ?", tc.ID).Update("order", tc.Order).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test cases reordered"})
}

// ProblemDataFileItem represents a file in the problem data directory.
type ProblemDataFileItem struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

// AdminProblemDataFiles - GET /api/v2/admin/problem/:code/data/files
func AdminProblemDataFiles(c *gin.Context) {
	problemCode := c.Param("code")

	var problem models.Problem
	if err := db.DB.Where("code = ?", problemCode).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	dataDir := filepath.Join("data", "problems", problemCode)

	var files []ProblemDataFileItem
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{"files": []ProblemDataFileItem{}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, ProblemDataFileItem{
			Name:    entry.Name(),
			Path:    entry.Name(),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02T15:04:05Z"),
		})
	}

	// Sort: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})

	c.JSON(http.StatusOK, gin.H{"files": files})
}

// AdminProblemDataFileContent - GET /api/v2/admin/problem/:code/data/file/*path
func AdminProblemDataFileContent(c *gin.Context) {
	problemCode := c.Param("code")

	// Get the rest of the path after /file/
	fullPath := strings.TrimPrefix(c.Request.URL.Path, "/api/v2/admin/problem/"+problemCode+"/data/file/")

	var problem models.Problem
	if err := db.DB.Where("code = ?", problemCode).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	// Security: prevent path traversal
	if strings.Contains(fullPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	absPath := filepath.Join("data", "problems", problemCode, fullPath)

	// Verify the path is within the problem directory
	cleanPath := filepath.Clean(absPath)
	expectedPrefix := filepath.Clean(filepath.Join("data", "problems", problemCode))
	if !strings.HasPrefix(cleanPath, expectedPrefix) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Check if it's a text file
	ext := strings.ToLower(filepath.Ext(fullPath))
	textExts := map[string]bool{
		".txt": true, ".yml": true, ".yaml": true,
		".cpp": true, ".cc": true, ".h": true, ".hpp": true,
		".py": true, ".java": true, ".js": true, ".ts": true,
		".json": true, ".xml": true, ".md": true,
		".in": true, ".out": true, ".ans": true, ".expect": true,
	}

	if textExts[ext] || ext == "" {
		c.JSON(http.StatusOK, gin.H{
			"content":  string(content),
			"encoding": "utf-8",
		})
	} else {
		// Return base64 for binary files
		c.JSON(http.StatusOK, gin.H{
			"content":  base64.StdEncoding.EncodeToString(content),
			"encoding": "base64",
		})
	}
}

// AdminProblemDataFileDelete - DELETE /api/v2/admin/problem/:code/data/file/*path
func AdminProblemDataFileDelete(c *gin.Context) {
	problemCode := c.Param("code")
	fullPath := strings.TrimPrefix(c.Request.URL.Path, "/api/v2/admin/problem/"+problemCode+"/data/file/")

	var problem models.Problem
	if err := db.DB.Where("code = ?", problemCode).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	// Security: prevent path traversal
	if strings.Contains(fullPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	// Don't allow deleting testcase directory
	if strings.HasPrefix(fullPath, "testcase/") || fullPath == "testcase" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete testcase directory via this endpoint"})
		return
	}

	absPath := filepath.Join("data", "problems", problemCode, fullPath)

	// Verify the path is within the problem directory
	cleanPath := filepath.Clean(absPath)
	expectedPrefix := filepath.Clean(filepath.Join("data", "problems", problemCode))
	if !strings.HasPrefix(cleanPath, expectedPrefix) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	if err := os.Remove(absPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file deleted"})
}

// TestCaseContentResponse - GET /api/v2/admin/problem/:code/data/testcase/:id/content
type TestCaseContentResponse struct {
	ID         uint   `json:"id"`
	Order      int    `json:"order"`
	InputData  string `json:"input_data"`
	OutputData string `json:"output_data"`
	Encoding   string `json:"encoding"`
}

// AdminProblemDataTestCaseContent - GET /api/v2/admin/problem/:code/data/testcase/:id/content
func AdminProblemDataTestCaseContent(c *gin.Context) {
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

	// Read test case files
	dataDir := filepath.Join("data", "problems", problemCode, "testcase")
	inputPath := filepath.Join(dataDir, testCase.InputFile)
	outputPath := filepath.Join(dataDir, testCase.OutputFile)

	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		inputData = []byte{}
	}
	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		outputData = []byte{}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          testCase.ID,
		"order":       testCase.Order,
		"input_data":  string(inputData),
		"output_data": string(outputData),
		"encoding":    "utf-8",
	})
}

// TestCaseUpdateRequest - PATCH /api/v2/admin/problem/:code/data/testcase/:id
type TestCaseUpdateRequest struct {
	InputData  string `json:"input_data"`
	OutputData string `json:"output_data"`
	Order      *int   `json:"order"`
	Points     *int   `json:"points"`
	IsPretest  *bool  `json:"is_pretest"`
	Type       string `json:"type"`
}

// AdminProblemDataTestCaseUpdate - PATCH /api/v2/admin/problem/:code/data/testcase/:id
func AdminProblemDataTestCaseUpdate(c *gin.Context) {
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

	var req TestCaseUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		updates := make(map[string]interface{})

		if req.Order != nil {
			updates["order"] = *req.Order
		}
		if req.Points != nil {
			updates["points"] = *req.Points
		}
		if req.IsPretest != nil {
			updates["is_pretest"] = *req.IsPretest
		}
		if req.Type != "" {
			updates["type"] = req.Type
		}

		if len(updates) > 0 {
			if err := tx.Model(&testCase).Updates(updates).Error; err != nil {
				return err
			}
		}

		// Update file contents
		if req.InputData != "" || req.OutputData != "" {
			dataDir := filepath.Join("data", "problems", problemCode, "testcase")
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return err
			}

			if req.InputData != "" {
				inputPath := filepath.Join(dataDir, testCase.InputFile)
				if err := os.WriteFile(inputPath, []byte(req.InputData), 0644); err != nil {
					return err
				}
			}
			if req.OutputData != "" {
				outputPath := filepath.Join(dataDir, testCase.OutputFile)
				if err := os.WriteFile(outputPath, []byte(req.OutputData), 0644); err != nil {
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

	c.JSON(http.StatusOK, gin.H{"message": "test case updated"})
}

// AdminProblemPdfUpload - POST /api/v2/admin/problem/:code/pdf
// Upload a PDF file for the problem statement
func AdminProblemPdfUpload(c *gin.Context) {
	problemCode := c.Param("code")

	var problem models.Problem
	if err := db.DB.Where("code = ?", problemCode).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("pdf_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no PDF file uploaded"})
		return
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only PDF files are allowed"})
		return
	}

	// Create problem data directory if it doesn't exist
	dataDir := filepath.Join("data", "problems", problemCode)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create data directory"})
		return
	}

	// Save the PDF file
	pdfFilename := "statement.pdf"
	pdfPath := filepath.Join(dataDir, pdfFilename)
	if err := c.SaveUploadedFile(file, pdfPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save PDF file"})
		return
	}

	// Update the problem's pdf_url field
	if err := db.DB.Model(&problem).Update("pdf_url", pdfFilename).Error; err != nil {
		os.Remove(pdfPath) // Clean up uploaded file
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update problem"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "PDF uploaded successfully",
		"pdf_url":  pdfFilename,
		"pdf_path": pdfPath,
	})
}

// AdminProblemPdfDelete - DELETE /api/v2/admin/problem/:code/pdf
// Delete the PDF statement file for a problem
func AdminProblemPdfDelete(c *gin.Context) {
	problemCode := c.Param("code")

	var problem models.Problem
	if err := db.DB.Where("code = ?", problemCode).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	// Get the current pdf_url
	if problem.PdfURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no PDF configured"})
		return
	}

	// Construct file path
	pdfPath := filepath.Join("data", "problems", problemCode, problem.PdfURL)

	// Delete the file
	if err := os.Remove(pdfPath); err != nil && !os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete PDF file"})
		return
	}

	// Clear the pdf_url field
	if err := db.DB.Model(&problem).Update("pdf_url", "").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update problem"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "PDF deleted successfully"})
}
