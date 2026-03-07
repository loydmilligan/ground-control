package pipeline

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mmariani/ground-control/internal/session"
)

// TesterStage runs tests and reports results.
type TesterStage struct {
	projectDir string
}

// NewTesterStage creates a new Tester stage.
func NewTesterStage(projectDir string) *TesterStage {
	return &TesterStage{
		projectDir: projectDir,
	}
}

// Name returns the stage name.
func (s *TesterStage) Name() string {
	return StageNameTester
}

// CanSkip returns true if testing is disabled in config.
func (s *TesterStage) CanSkip(ctx *StageContext) bool {
	// Could check config here for skip_test flag
	return false
}

// Execute runs the Tester stage.
func (s *TesterStage) Execute(ctx *StageContext) *StageResult {
	task := ctx.Task

	// Determine test command based on project type
	testCmd, testArgs := s.detectTestCommand()

	if ctx.Verbose {
		fmt.Printf("    Running: %s %s\n", testCmd, strings.Join(testArgs, " "))
	}

	// Run tests
	cmd := exec.Command(testCmd, testArgs...)
	cmd.Dir = s.projectDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	testOutput := stdout.String() + stderr.String()

	// Parse results
	passed := err == nil
	testSummary := s.parseTestOutput(testOutput, passed)

	// Save test results
	resultsPath := ""
	if task.ContextBundle != nil {
		resultsPath = filepath.Join(task.ContextBundle.BundlePath,
			fmt.Sprintf("test_results_%d.md", ctx.Iteration))
		content := fmt.Sprintf("# Test Results\n\n**Task**: %s\n**Iteration**: %d\n**Time**: %s\n**Status**: %s\n\n## Summary\n\n%s\n\n## Full Output\n\n```\n%s\n```",
			task.Title, ctx.Iteration, time.Now().Format(time.RFC3339),
			map[bool]string{true: "PASSED", false: "FAILED"}[passed],
			testSummary, testOutput)
		os.WriteFile(resultsPath, []byte(content), 0644)
	}

	outputFiles := []string{}
	if resultsPath != "" {
		outputFiles = append(outputFiles, resultsPath)
	}

	// Collect issues
	var issues []session.SessionIssue

	if passed {
		return &StageResult{
			Status:      StageStatusSuccess,
			OutputFiles: outputFiles,
			Notes:       "All tests passed",
			Issues:      issues,
		}
	}

	// Tests failed - generate feedback for Coder
	feedback := s.generateTestFeedback(testOutput, task)

	// Add issue for self-learning
	if ctx.Iteration > 1 {
		issues = append(issues, session.SessionIssue{
			TaskID:      task.ID,
			Stage:       StageNameTester,
			Severity:    "moderate",
			Category:    "test_failure",
			Description: fmt.Sprintf("Tests failed on iteration %d", ctx.Iteration),
			Impact:      "Required additional fix iteration",
		})
	}

	return &StageResult{
		Status:      StageStatusNeedsRevision,
		OutputFiles: outputFiles,
		Feedback:    feedback,
		Notes:       fmt.Sprintf("Tests failed (iteration %d)", ctx.Iteration),
		Issues:      issues,
	}
}

// detectTestCommand determines the appropriate test command for the project.
func (s *TesterStage) detectTestCommand() (string, []string) {
	// Check for Go project
	if _, err := os.Stat(filepath.Join(s.projectDir, "go.mod")); err == nil {
		return "go", []string{"test", "./...", "-v"}
	}

	// Check for Node.js project
	if _, err := os.Stat(filepath.Join(s.projectDir, "package.json")); err == nil {
		// Check for npm test script
		return "npm", []string{"test"}
	}

	// Check for Python project
	if _, err := os.Stat(filepath.Join(s.projectDir, "pytest.ini")); err == nil {
		return "pytest", []string{"-v"}
	}
	if _, err := os.Stat(filepath.Join(s.projectDir, "setup.py")); err == nil {
		return "python", []string{"-m", "pytest", "-v"}
	}

	// Default to go test
	return "go", []string{"test", "./...", "-v"}
}

// parseTestOutput extracts a summary from test output.
func (s *TesterStage) parseTestOutput(output string, passed bool) string {
	lines := strings.Split(output, "\n")
	var summary strings.Builder

	if passed {
		summary.WriteString("✓ All tests passed\n\n")
	} else {
		summary.WriteString("✗ Some tests failed\n\n")
	}

	// Look for test result lines
	var failedTests []string
	var passedCount, failedCount int

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Go test output patterns
		if strings.HasPrefix(line, "--- FAIL:") {
			failedTests = append(failedTests, line)
			failedCount++
		} else if strings.HasPrefix(line, "--- PASS:") {
			passedCount++
		} else if strings.HasPrefix(line, "FAIL") && !strings.HasPrefix(line, "--- FAIL") {
			// Package failure line
			failedTests = append(failedTests, line)
		} else if strings.HasPrefix(line, "ok") {
			// Package success line
			passedCount++
		}
	}

	summary.WriteString(fmt.Sprintf("Passed: %d, Failed: %d\n", passedCount, failedCount))

	if len(failedTests) > 0 {
		summary.WriteString("\n### Failed Tests\n\n")
		for _, test := range failedTests {
			summary.WriteString(fmt.Sprintf("- %s\n", test))
		}
	}

	return summary.String()
}

// generateTestFeedback creates actionable feedback for the Coder.
func (s *TesterStage) generateTestFeedback(output string, task interface{}) string {
	var feedback strings.Builder

	feedback.WriteString("## Test Failures\n\n")
	feedback.WriteString("The following tests failed and need to be fixed:\n\n")

	lines := strings.Split(output, "\n")
	inFailure := false
	var currentFailure strings.Builder

	for _, line := range lines {
		// Detect start of failure
		if strings.HasPrefix(strings.TrimSpace(line), "--- FAIL:") {
			if inFailure && currentFailure.Len() > 0 {
				feedback.WriteString("```\n")
				feedback.WriteString(currentFailure.String())
				feedback.WriteString("```\n\n")
			}
			inFailure = true
			currentFailure.Reset()
			currentFailure.WriteString(line + "\n")
		} else if inFailure {
			// Check for end of failure block
			if strings.HasPrefix(strings.TrimSpace(line), "--- PASS:") ||
				strings.HasPrefix(strings.TrimSpace(line), "PASS") ||
				strings.HasPrefix(strings.TrimSpace(line), "FAIL") && !strings.Contains(line, "FAIL:") {
				feedback.WriteString("```\n")
				feedback.WriteString(currentFailure.String())
				feedback.WriteString("```\n\n")
				inFailure = false
				currentFailure.Reset()
			} else {
				currentFailure.WriteString(line + "\n")
			}
		}
	}

	// Flush any remaining failure
	if inFailure && currentFailure.Len() > 0 {
		feedback.WriteString("```\n")
		feedback.WriteString(currentFailure.String())
		feedback.WriteString("```\n\n")
	}

	feedback.WriteString("## Instructions\n\n")
	feedback.WriteString("Please fix the failing tests. Common issues:\n")
	feedback.WriteString("- Check the expected vs actual values in assertions\n")
	feedback.WriteString("- Ensure all edge cases are handled\n")
	feedback.WriteString("- Verify error handling paths\n")

	return feedback.String()
}
