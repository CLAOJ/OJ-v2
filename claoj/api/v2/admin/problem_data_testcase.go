package admin

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/problemdata"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminProblemDataDeleteTestCase - DELETE /api/v2/admin/problem/:code/data/testcase/:id
func AdminProblemDataDeleteTestCase(c *gin.Context) {
	problemCode := c.Param("code")
	testCaseID := c.Param("id")

	problem, err := GetProblemByCode(problemCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	var testCase models.ProblemTestCase
	if err := db.DB.Where("id = ? AND dataset_id = ?", testCaseID, problem.ID).First(&testCase).Error; err != nil {
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

	problem, err := GetProblemByCode(problemCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	var req TestCaseReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// Validate all test cases belong to this problem (dataset_id is judge_problem.id)
		for _, tc := range req.TestCases {
			var testCase models.ProblemTestCase
			if err := tx.Where("id = ? AND dataset_id = ?", tc.ID, problem.ID).First(&testCase).Error; err != nil {
				return err
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

	problem, err := GetProblemByCode(problemCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	var testCase models.ProblemTestCase
	if err := db.DB.Where("id = ? AND dataset_id = ?", testCaseID, problem.ID).First(&testCase).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test case not found"})
		return
	}

	inputData, inCut := testCaseFile(problemCode, testCase.InputFile)
	outputData, outCut := testCaseFile(problemCode, testCase.OutputFile)

	c.JSON(http.StatusOK, gin.H{
		"id":          testCase.ID,
		"order":       testCase.Order,
		"input_data":  inputData,
		"output_data": outputData,
		"truncated":   inCut || outCut,
		"encoding":    "utf-8",
	})
}

// testCaseContentLimit caps each file shown in the admin preview.
const testCaseContentLimit = 256 << 10

// testCaseFile reads a test file from the judges' archive (v1 layout), falling
// back to the v2-local data/problems/<code>/testcase/ directory.
func testCaseFile(code, name string) (string, bool) {
	if name == "" {
		return "", false
	}
	if a, err := problemdata.Open(code); err == nil {
		defer a.Close()
		if s, cut, err := a.Read(name, testCaseContentLimit); err == nil {
			return s, cut
		}
	}
	dir := filepath.Clean(filepath.Join("data", "problems", code, "testcase"))
	path := filepath.Join(dir, name)
	if !strings.HasPrefix(path, dir+string(os.PathSeparator)) {
		return "", false
	}
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	data, _ := io.ReadAll(io.LimitReader(f, testCaseContentLimit+1))
	cut := len(data) > testCaseContentLimit
	if cut {
		data = data[:testCaseContentLimit]
	}
	return strings.ToValidUTF8(string(data), ""), cut
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

	problem, err := GetProblemByCode(problemCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}

	var testCase models.ProblemTestCase
	if err := db.DB.Where("id = ? AND dataset_id = ?", testCaseID, problem.ID).First(&testCase).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test case not found"})
		return
	}

	var req TestCaseUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = db.DB.Transaction(func(tx *gorm.DB) error {
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
				if err := os.WriteFile(inputPath, []byte(req.InputData), 0600); err != nil {
					return err
				}
			}
			if req.OutputData != "" {
				outputPath := filepath.Join(dataDir, testCase.OutputFile)
				if err := os.WriteFile(outputPath, []byte(req.OutputData), 0600); err != nil {
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
