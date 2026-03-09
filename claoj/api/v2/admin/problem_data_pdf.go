package admin

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/CLAOJ/claoj/db"
	"github.com/gin-gonic/gin"
)

// AdminProblemPdfUpload - POST /api/v2/admin/problem/:code/pdf
// Upload a PDF file for the problem statement
func AdminProblemPdfUpload(c *gin.Context) {
	problemCode := c.Param("code")

	problem, err := GetProblemByCode(problemCode)
	if err != nil {
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

	problem, err := GetProblemByCode(problemCode)
	if err != nil {
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
