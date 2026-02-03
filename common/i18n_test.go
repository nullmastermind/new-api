package common

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// isUserFacingString checks if a line contains user-facing Chinese strings
// Excludes: comments, log messages, and inline comments
func isUserFacingString(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Skip comments
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return false
	}

	// Skip import statements
	if strings.HasPrefix(trimmed, "import") {
		return false
	}

	// Skip log statements (developer-facing)
	if strings.Contains(line, "log.Print") || strings.Contains(line, "logger.Log") ||
		strings.Contains(line, "common.SysLog") || strings.Contains(line, "common.SysError") ||
		strings.Contains(line, "fmt.Println") || strings.Contains(line, "console.log") {
		return false
	}

	// Skip inline comments (anything after //)
	if idx := strings.Index(line, "//"); idx != -1 {
		line = line[:idx]
	}

	// Check for user-facing patterns: "message", errors.New, fmt.Errorf, ApiError
	if strings.Contains(line, `"message"`) ||
		strings.Contains(line, "errors.New") ||
		strings.Contains(line, "fmt.Errorf") ||
		strings.Contains(line, "ApiError") ||
		strings.Contains(line, "ApiErrorMsg") {
		return true
	}

	return false
}

// TestNoChineseInControllers scans controller files for Chinese characters in API responses
func TestNoChineseInControllers(t *testing.T) {
	controllerDir := filepath.Join("..", "controller")
	chineseRegex := regexp.MustCompile(`[\x{4e00}-\x{9fa5}]`)

	files, err := filepath.Glob(filepath.Join(controllerDir, "*.go"))
	require.NoError(t, err, "Failed to list controller files")

	violations := []string{}

	for _, file := range files {
		content, err := os.ReadFile(file)
		require.NoError(t, err, "Failed to read file: %s", file)

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			// Check for Chinese characters in user-facing strings only
			if chineseRegex.MatchString(line) && isUserFacingString(line) {
				violations = append(violations, filepath.Base(file)+":"+string(rune(i+1))+": "+strings.TrimSpace(line))
			}
		}
	}

	if len(violations) > 0 {
		t.Errorf("Found %d Chinese characters in controller API responses:\n%s", len(violations), strings.Join(violations, "\n"))
	}
}

// TestNoChineseInMiddleware scans middleware files for Chinese characters in error messages
func TestNoChineseInMiddleware(t *testing.T) {
	middlewareDir := filepath.Join("..", "middleware")
	chineseRegex := regexp.MustCompile(`[\x{4e00}-\x{9fa5}]`)

	files, err := filepath.Glob(filepath.Join(middlewareDir, "*.go"))
	require.NoError(t, err, "Failed to list middleware files")

	violations := []string{}

	for _, file := range files {
		content, err := os.ReadFile(file)
		require.NoError(t, err, "Failed to read file: %s", file)

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			// Check for Chinese characters in user-facing strings only
			if chineseRegex.MatchString(line) && isUserFacingString(line) {
				violations = append(violations, filepath.Base(file)+":"+string(rune(i+1))+": "+strings.TrimSpace(line))
			}
		}
	}

	if len(violations) > 0 {
		t.Errorf("Found %d Chinese characters in middleware error messages:\n%s", len(violations), strings.Join(violations, "\n"))
	}
}

// TestNoChineseInServices scans service files for Chinese characters in user-facing messages
func TestNoChineseInServices(t *testing.T) {
	serviceDir := filepath.Join("..", "service")
	chineseRegex := regexp.MustCompile(`[\x{4e00}-\x{9fa5}]`)

	files, err := filepath.Glob(filepath.Join(serviceDir, "*.go"))
	require.NoError(t, err, "Failed to list service files")

	violations := []string{}

	for _, file := range files {
		content, err := os.ReadFile(file)
		require.NoError(t, err, "Failed to read file: %s", file)

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			// Check for Chinese characters in user-facing strings only
			if chineseRegex.MatchString(line) && isUserFacingString(line) {
				violations = append(violations, filepath.Base(file)+":"+string(rune(i+1))+": "+strings.TrimSpace(line))
			}
		}
	}

	if len(violations) > 0 {
		t.Errorf("Found %d Chinese characters in service user-facing messages:\n%s", len(violations), strings.Join(violations, "\n"))
	}
}
