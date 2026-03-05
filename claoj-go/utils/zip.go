// Package utils provides utility functions for the CLAOJ backend.
package utils

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// TestCaseFile represents a single test case file pair.
type TestCaseFile struct {
	Order      int    `json:"order"`
	InputName  string `json:"input_name"`
	OutputName string `json:"output_name"`
	InputData  []byte `json:"-"`
	OutputData []byte `json:"-"`
}

// ProblemDataArchive represents extracted problem data from a ZIP file.
type ProblemDataArchive struct {
	TestCases    []TestCaseFile
	InitYML      []byte
	Checker      []byte
	Grader       []byte
	Header       []byte
	Generator    []byte
	GeneratorYML []byte
	Feedback     string
	OtherFiles   map[string][]byte
}

// ExtractZIP extracts test cases and problem data from a ZIP file.
// It supports standard DMOJ format and common variants.
func ExtractZIP(zipData []byte) (*ProblemDataArchive, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	result := &ProblemDataArchive{
		OtherFiles: make(map[string][]byte),
	}

	// First pass: collect all files
	type fileEntry struct {
		path string
		data []byte
	}
	var files []fileEntry

	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		// Normalize path separators
		normalizedPath := filepath.ToSlash(f.Name)
		files = append(files, fileEntry{path: normalizedPath, data: data})
	}

	// Second pass: categorize files
	for _, f := range files {
		base := strings.ToLower(filepath.Base(f.path))

		// Special files
		switch {
		case base == "init.yml":
			result.InitYML = f.data
		case base == "checker.cpp" || base == "checker.cc":
			result.Checker = f.data
		case base == "grader.cpp" || base == "grader.cc":
			result.Grader = f.data
		case base == "header.h" || base == "grader.h":
			result.Header = f.data
		case base == "generator.cpp" || base == "generator.cc":
			result.Generator = f.data
		case base == "generator.yml" || base == "gen.yml":
			result.GeneratorYML = f.data
		case base == "feedback.txt":
			result.Feedback = string(f.data)
		}

		// Test case files - look for .in and .out pairs
		if strings.HasSuffix(base, ".in") || strings.HasSuffix(base, ".out") ||
			strings.HasSuffix(base, ".ans") || strings.HasSuffix(base, ".expect") {
			// Store in OtherFiles for later pairing
			result.OtherFiles[f.path] = f.data
		}
	}

	// Third pass: pair test cases
	result.TestCases = pairTestCases(result.OtherFiles)

	return result, nil
}

// pairTestCases matches input and output files into test case pairs.
func pairTestCases(files map[string][]byte) []TestCaseFile {
	type candidate struct {
		path    string
		data    []byte
		order   int
		isInput bool
	}

	var inputs []candidate
	var outputs []candidate

	// Regex patterns for common test case naming conventions
	// e.g., 1.in, 01.in, test1.in, sample1.in, testcase1.in
	orderRegex := regexp.MustCompile(`(\d+)`)

	for path, data := range files {
		base := strings.ToLower(filepath.Base(path))
		isInput := strings.HasSuffix(base, ".in")
		isOutput := strings.HasSuffix(base, ".out") || strings.HasSuffix(base, ".ans") || strings.HasSuffix(base, ".expect")

		if !isInput && !isOutput {
			continue
		}

		// Extract order number from filename
		matches := orderRegex.FindStringSubmatch(base)
		order := 0
		if len(matches) > 1 {
			order, _ = strconv.Atoi(matches[1])
		}

		entry := candidate{path: path, data: data, order: order, isInput: isInput}
		if isInput {
			inputs = append(inputs, entry)
		} else {
			outputs = append(outputs, entry)
		}
	}

	// Sort by order
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].order < inputs[j].order })
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].order < outputs[j].order })

	// Pair test cases
	var testCases []TestCaseFile
	maxLen := len(inputs)
	if len(outputs) < maxLen {
		maxLen = len(outputs)
	}

	for i := 0; i < maxLen; i++ {
		inputPath := inputs[i].path
		outputPath := outputs[i].path

		testCases = append(testCases, TestCaseFile{
			Order:      i,
			InputName:  filepath.Base(inputPath),
			OutputName: filepath.Base(outputPath),
			InputData:  inputs[i].data,
			OutputData: outputs[i].data,
		})
	}

	return testCases
}

// GenerateInitYML generates an init.yml file for the judge.
func GenerateInitYML(checker string, checkerArgs string, grader string, graderArgs string) []byte {
	var buf bytes.Buffer

	buf.WriteString("# Auto-generated init.yml\n")
	buf.WriteString("unicode: false\n")
	buf.WriteString("markdown: problem.md\n")
	buf.WriteString("\n")

	if grader != "" && grader != "standard" {
		buf.WriteString(fmt.Sprintf("grader: %s\n", grader))
		if graderArgs != "" {
			buf.WriteString(fmt.Sprintf("grader_args: %s\n", graderArgs))
		}
	} else {
		buf.WriteString("grader: standard\n")
	}

	if checker != "" && checker != "standard" {
		buf.WriteString(fmt.Sprintf("checker: %s\n", checker))
		if checkerArgs != "" {
			buf.WriteString(fmt.Sprintf("checker_args: %s\n", checkerArgs))
		}
	} else {
		buf.WriteString("checker: standard\n")
	}

	buf.WriteString("\n")
	buf.WriteString("pretest_syntax: true\n")
	buf.WriteString("pretest_only: false\n")

	return buf.Bytes()
}

// SanitizeFilename removes dangerous path components from a filename.
func SanitizeFilename(filename string) string {
	// Remove path separators
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	// Remove parent directory references
	filename = strings.ReplaceAll(filename, "..", "_")
	// Remove leading dots
	filename = strings.TrimPrefix(filename, ".")
	return filename
}

// ValidateZIPFile checks if a file in the ZIP is safe to extract.
func ValidateZIPFile(filename string) error {
	// Check for path traversal
	if strings.Contains(filename, "..") {
		return fmt.Errorf("path traversal detected: %s", filename)
	}

	// Check for absolute paths
	if filepath.IsAbs(filename) {
		return fmt.Errorf("absolute path not allowed: %s", filename)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filename))
	allowedExts := map[string]bool{
		".in": true, ".out": true, ".ans": true, ".expect": true,
		".yml": true, ".yaml": true,
		".cpp": true, ".cc": true, ".h": true, ".hpp": true,
		".txt": true, ".md": true,
		".py": true, ".java": true,
	}

	if !allowedExts[ext] && ext != "" {
		// Allow files without extension or with unknown extension for test data
		// but warn about it
	}

	return nil
}
