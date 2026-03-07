package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mmariani/ground-control/internal/session"
)

// NotifyStage notifies the user that input is needed and writes questions file.
type NotifyStage struct{}

// NewNotifyStage creates a new Notify stage.
func NewNotifyStage() *NotifyStage {
	return &NotifyStage{}
}

// Name returns the stage name.
func (s *NotifyStage) Name() string {
	return StageNameNotify
}

// CanSkip returns false - notify stage should not be skipped.
func (s *NotifyStage) CanSkip(ctx *StageContext) bool {
	return false
}

// Execute runs the Notify stage.
func (s *NotifyStage) Execute(ctx *StageContext) *StageResult {
	task := ctx.Task

	// Ensure we have a context bundle
	if task.ContextBundle == nil {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("task has no context bundle"),
		}
	}

	// Create questions file
	questionsPath := filepath.Join(task.ContextBundle.BundlePath, "human_questions.md")

	// Build questions content
	var sb strings.Builder
	sb.WriteString("# Human Input Required\n\n")
	sb.WriteString(fmt.Sprintf("**Task**: %s\n", task.Title))
	sb.WriteString(fmt.Sprintf("**Task ID**: %s\n", task.ID))
	sb.WriteString(fmt.Sprintf("**Time**: %s\n\n", time.Now().Format(time.RFC3339)))

	sb.WriteString("## Description\n\n")
	sb.WriteString(task.Description)
	sb.WriteString("\n\n")

	if len(task.Context.Requirements) > 0 {
		sb.WriteString("## Requirements\n\n")
		for _, req := range task.Context.Requirements {
			sb.WriteString(fmt.Sprintf("- %s\n", req))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Questions / Input Needed\n\n")
	sb.WriteString("Please provide the required input below. When complete, save this file ")
	sb.WriteString("as `human_response.md` in the same directory.\n\n")

	// Add space for human to write their response
	sb.WriteString("---\n\n")
	sb.WriteString("## Your Response\n\n")
	sb.WriteString("<!-- Write your response here -->\n\n")

	// Write questions file
	if err := os.WriteFile(questionsPath, []byte(sb.String()), 0644); err != nil {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("failed to write questions file: %w", err),
		}
	}

	return &StageResult{
		Status:      StageStatusSuccess,
		OutputFiles: []string{questionsPath},
		Notes:       fmt.Sprintf("Human input requested - see %s", questionsPath),
	}
}

// WaitHumanStage checks if human has responded.
type WaitHumanStage struct{}

// NewWaitHumanStage creates a new WaitHuman stage.
func NewWaitHumanStage() *WaitHumanStage {
	return &WaitHumanStage{}
}

// Name returns the stage name.
func (s *WaitHumanStage) Name() string {
	return StageNameWaitHuman
}

// CanSkip returns false - wait stage should not be skipped.
func (s *WaitHumanStage) CanSkip(ctx *StageContext) bool {
	return false
}

// Execute runs the WaitHuman stage.
func (s *WaitHumanStage) Execute(ctx *StageContext) *StageResult {
	task := ctx.Task

	// Ensure we have a context bundle
	if task.ContextBundle == nil {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("task has no context bundle"),
		}
	}

	// Check if response file exists
	responsePath := filepath.Join(task.ContextBundle.BundlePath, "human_response.md")
	if _, err := os.Stat(responsePath); os.IsNotExist(err) {
		// Response not yet provided
		return &StageResult{
			Status: StageStatusEscalate,
			Notes:  "Waiting for human response - response file not found",
		}
	}

	// Response file exists
	return &StageResult{
		Status:      StageStatusSuccess,
		OutputFiles: []string{responsePath},
		Notes:       "Human response received",
	}
}

// CaptureResponseStage reads human responses and updates task.
type CaptureResponseStage struct{}

// NewCaptureResponseStage creates a new CaptureResponse stage.
func NewCaptureResponseStage() *CaptureResponseStage {
	return &CaptureResponseStage{}
}

// Name returns the stage name.
func (s *CaptureResponseStage) Name() string {
	return StageNameCaptureResponse
}

// CanSkip returns false - capture stage should not be skipped.
func (s *CaptureResponseStage) CanSkip(ctx *StageContext) bool {
	return false
}

// Execute runs the CaptureResponse stage.
func (s *CaptureResponseStage) Execute(ctx *StageContext) *StageResult {
	task := ctx.Task

	// Ensure we have a context bundle
	if task.ContextBundle == nil {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("task has no context bundle"),
		}
	}

	// Read response file
	responsePath := filepath.Join(task.ContextBundle.BundlePath, "human_response.md")
	responseBytes, err := os.ReadFile(responsePath)
	if err != nil {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("failed to read response file: %w", err),
		}
	}

	response := string(responseBytes)

	// Check if response is empty or just contains template
	if len(strings.TrimSpace(response)) == 0 ||
	   strings.Contains(response, "<!-- Write your response here -->") &&
	   len(strings.TrimSpace(strings.ReplaceAll(response, "<!-- Write your response here -->", ""))) < 50 {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("human response appears to be empty or template not filled out"),
			Notes:  "Please provide actual content in the response file",
		}
	}

	// Save response to captured_response.md for record-keeping
	capturedPath := filepath.Join(task.ContextBundle.BundlePath, "captured_response.md")
	capturedContent := fmt.Sprintf("# Captured Human Response\n\n**Task**: %s\n**Captured**: %s\n\n%s",
		task.Title, time.Now().Format(time.RFC3339), response)

	if err := os.WriteFile(capturedPath, []byte(capturedContent), 0644); err != nil {
		// Non-fatal - log issue but continue
		var issues []session.SessionIssue
		issues = append(issues, session.SessionIssue{
			TaskID:      task.ID,
			Stage:       StageNameCaptureResponse,
			Severity:    "minor",
			Category:    "file_write",
			Description: "Could not save captured response to file",
			Impact:      "Response captured but not archived",
		})

		return &StageResult{
			Status:      StageStatusSuccess,
			OutputFiles: []string{responsePath},
			Notes:       "Human response captured (archiving failed)",
			Issues:      issues,
		}
	}

	return &StageResult{
		Status:      StageStatusSuccess,
		OutputFiles: []string{responsePath, capturedPath},
		Notes:       truncateOutput(response, 200),
	}
}
