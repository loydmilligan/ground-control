package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mmariani/ground-control/internal/claude"
	"github.com/mmariani/ground-control/internal/session"
)

// ReviewerStage reviews code changes and provides feedback.
type ReviewerStage struct {
	client     *claude.Client
	projectDir string
}

// NewReviewerStage creates a new Reviewer stage.
func NewReviewerStage(client *claude.Client, projectDir string) *ReviewerStage {
	return &ReviewerStage{
		client:     client,
		projectDir: projectDir,
	}
}

// Name returns the stage name.
func (s *ReviewerStage) Name() string {
	return StageNameReviewer
}

// CanSkip returns true if review is disabled in config.
func (s *ReviewerStage) CanSkip(ctx *StageContext) bool {
	// Could check config here
	return false
}

// Execute runs the Reviewer stage.
func (s *ReviewerStage) Execute(ctx *StageContext) *StageResult {
	task := ctx.Task

	// Build the review prompt
	prompt := claude.BuildReviewerPrompt(
		task.Title,
		task.Context.Requirements,
		[]string{}, // TODO: Get actually changed files from git
	)

	// Get context files from bundle
	var contextFiles []string
	if task.ContextBundle != nil && task.ContextBundle.BundlePath != "" {
		contextFiles = claude.GetContextFiles(task.ContextBundle.BundlePath)

		// Also include implementation notes if they exist
		notesPath := filepath.Join(task.ContextBundle.BundlePath, "implementation_notes.md")
		if _, err := os.Stat(notesPath); err == nil {
			contextFiles = append(contextFiles, notesPath)
		}
	}

	// Execute Claude for review
	req := &claude.Request{
		Prompt:       prompt,
		ContextFiles: contextFiles,
	}

	var output string
	if ctx.Verbose {
		fmt.Printf("    Invoking Claude Code CLI for review...\n")
		resp := s.client.ExecuteWithStreaming(req, func(chunk string) {
			fmt.Print(chunk)
		})
		output = resp.Output
		if resp.Error != nil {
			return &StageResult{
				Status: StageStatusFailed,
				Error:  fmt.Errorf("claude CLI failed: %w", resp.Error),
			}
		}
	} else {
		resp := s.client.Execute(req)
		output = resp.Output
		if resp.Error != nil {
			return &StageResult{
				Status: StageStatusFailed,
				Error:  fmt.Errorf("claude CLI failed: %w", resp.Error),
			}
		}
	}

	// Parse the review output
	decision, feedback := parseReviewOutput(output)

	// Save review feedback
	feedbackPath := ""
	if task.ContextBundle != nil {
		feedbackPath = filepath.Join(task.ContextBundle.BundlePath,
			fmt.Sprintf("review_feedback_%d.md", ctx.Iteration))
		content := fmt.Sprintf("# Code Review Feedback\n\n**Task**: %s\n**Iteration**: %d\n**Time**: %s\n**Decision**: %s\n\n## Feedback\n\n%s",
			task.Title, ctx.Iteration, time.Now().Format(time.RFC3339), decision, feedback)
		os.WriteFile(feedbackPath, []byte(content), 0644)
	}

	// Collect issues for self-learning
	var issues []session.SessionIssue

	outputFiles := []string{}
	if feedbackPath != "" {
		outputFiles = append(outputFiles, feedbackPath)
	}

	switch decision {
	case "APPROVED":
		return &StageResult{
			Status:      StageStatusSuccess,
			OutputFiles: outputFiles,
			Notes:       "Code review passed",
			Issues:      issues,
		}

	case "NEEDS_REVISION":
		// Add issue for self-learning if this is iteration > 1
		if ctx.Iteration > 1 {
			issues = append(issues, session.SessionIssue{
				TaskID:      task.ID,
				Stage:       StageNameReviewer,
				Severity:    "minor",
				Category:    "review_failure",
				Description: fmt.Sprintf("Code required %d revision rounds", ctx.Iteration),
				Impact:      "Additional iteration needed",
			})
		}

		return &StageResult{
			Status:      StageStatusNeedsRevision,
			OutputFiles: outputFiles,
			Feedback:    feedback,
			Notes:       fmt.Sprintf("Revision requested (iteration %d)", ctx.Iteration),
			Issues:      issues,
		}

	case "ESCALATE":
		issues = append(issues, session.SessionIssue{
			TaskID:      task.ID,
			Stage:       StageNameReviewer,
			Severity:    "significant",
			Category:    "review_failure",
			Description: "Reviewer escalated to human",
			Impact:      "Human intervention required",
			Suggestion:  feedback,
		})

		return &StageResult{
			Status:      StageStatusEscalate,
			OutputFiles: outputFiles,
			Feedback:    feedback,
			Notes:       "Escalated to human review",
			Issues:      issues,
		}

	default:
		// Couldn't parse decision, assume needs revision
		return &StageResult{
			Status:      StageStatusNeedsRevision,
			OutputFiles: outputFiles,
			Feedback:    output, // Return full output as feedback
			Notes:       "Could not parse review decision, assuming revision needed",
		}
	}
}

// parseReviewOutput extracts the decision and feedback from review output.
func parseReviewOutput(output string) (decision string, feedback string) {
	output = strings.TrimSpace(output)
	upperOutput := strings.ToUpper(output)

	// Strip markdown bold markers for parsing (** and *)
	cleanUpper := strings.ReplaceAll(upperOutput, "**", "")
	cleanUpper = strings.ReplaceAll(cleanUpper, "*", "")

	// Look for decision markers (check both original and cleaned versions)
	if strings.HasPrefix(upperOutput, "APPROVED") || strings.Contains(upperOutput, "\nAPPROVED") ||
		strings.HasPrefix(cleanUpper, "APPROVED") || strings.Contains(cleanUpper, "\nAPPROVED") {
		decision = "APPROVED"
		// Extract explanation after APPROVED:
		if idx := strings.Index(upperOutput, "APPROVED"); idx >= 0 {
			rest := output[idx+8:]
			if strings.HasPrefix(rest, ":") {
				feedback = strings.TrimSpace(rest[1:])
			} else {
				feedback = strings.TrimSpace(rest)
			}
		}
		return
	}

	if strings.HasPrefix(upperOutput, "NEEDS_REVISION") || strings.Contains(upperOutput, "\nNEEDS_REVISION") ||
		strings.HasPrefix(cleanUpper, "NEEDS_REVISION") || strings.Contains(cleanUpper, "\nNEEDS_REVISION") {
		decision = "NEEDS_REVISION"
		if idx := strings.Index(upperOutput, "NEEDS_REVISION"); idx >= 0 {
			rest := output[idx+14:]
			if strings.HasPrefix(rest, ":") {
				feedback = strings.TrimSpace(rest[1:])
			} else {
				feedback = strings.TrimSpace(rest)
			}
		}
		return
	}

	if strings.HasPrefix(upperOutput, "ESCALATE") || strings.Contains(upperOutput, "\nESCALATE") ||
		strings.HasPrefix(cleanUpper, "ESCALATE") || strings.Contains(cleanUpper, "\nESCALATE") {
		decision = "ESCALATE"
		if idx := strings.Index(upperOutput, "ESCALATE"); idx >= 0 {
			rest := output[idx+8:]
			if strings.HasPrefix(rest, ":") {
				feedback = strings.TrimSpace(rest[1:])
			} else {
				feedback = strings.TrimSpace(rest)
			}
		}
		return
	}

	// No clear decision found
	decision = "UNKNOWN"
	feedback = output
	return
}
