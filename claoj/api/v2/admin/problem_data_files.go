package admin

import (
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

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

	_, err := GetProblemByCode(problemCode)
	if err != nil {
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

	_, err := GetProblemByCode(problemCode)
	if err != nil {
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

	_, err := GetProblemByCode(problemCode)
	if err != nil {
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
