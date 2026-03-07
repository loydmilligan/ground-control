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

// SummaryStage synthesizes research findings into a deliverable.
type SummaryStage struct {
	client     *claude.Client
	projectDir string
}

// NewSummaryStage creates a new Summary stage.
func NewSummaryStage(client *claude.Client, projectDir string) *SummaryStage {
	return &SummaryStage{
		client:     client,
		projectDir: projectDir,
	}
}

// Name returns the stage name.
func (s *SummaryStage) Name() string {
	return StageNameSummary
}

// CanSkip returns false - summary stage should not be skipped.
func (s *SummaryStage) CanSkip(ctx *StageContext) bool {
	return false
}

// Execute runs the Summary stage.
func (s *SummaryStage) Execute(ctx *StageContext) *StageResult {
	task := ctx.Task

	// Check that findings.md exists
	var findingsPath string
	if task.ContextBundle != nil {
		findingsPath = filepath.Join(task.ContextBundle.BundlePath, "findings.md")
		if _, err := os.Stat(findingsPath); os.IsNotExist(err) {
			return &StageResult{
				Status: StageStatusFailed,
				Error:  fmt.Errorf("findings.md not found - research stage must run first"),
				Notes:  "Expected findings at: " + findingsPath,
			}
		}
	} else {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("task has no context bundle"),
		}
	}

	// Build the summary prompt
	prompt := BuildSummaryPrompt(
		task.Title,
		task.Description,
		task.Context.Requirements,
		ctx.PreviousFeedback,
	)

	// Get context files from bundle, including findings
	var contextFiles []string
	if task.ContextBundle != nil && task.ContextBundle.BundlePath != "" {
		contextFiles = claude.GetContextFiles(task.ContextBundle.BundlePath)
		// Add findings.md as primary input
		contextFiles = append([]string{findingsPath}, contextFiles...)
	}

	// Execute Claude
	var output string
	req := &claude.Request{
		Prompt:       prompt,
		ContextFiles: contextFiles,
	}

	if ctx.Verbose {
		fmt.Printf("    Invoking Claude Code CLI to generate summary...\n")
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

	// Save summary
	summaryPath := ""
	if task.ContextBundle != nil {
		summaryPath = filepath.Join(task.ContextBundle.BundlePath, "summary.md")
		content := fmt.Sprintf("# Research Summary\n\n**Task**: %s\n**Time**: %s\n\n## Summary\n\n%s",
			task.Title, time.Now().Format(time.RFC3339), output)

		if err := os.WriteFile(summaryPath, []byte(content), 0644); err != nil {
			return &StageResult{
				Status: StageStatusFailed,
				Error:  fmt.Errorf("failed to write summary: %w", err),
			}
		}
	}

	// Check for issues in output
	var issues []session.SessionIssue
	if strings.Contains(strings.ToLower(output), "incomplete") ||
	   strings.Contains(strings.ToLower(output), "unclear") {
		issues = append(issues, session.SessionIssue{
			TaskID:      task.ID,
			Stage:       StageNameSummary,
			Severity:    "minor",
			Category:    "summary_quality",
			Description: "Summary may be incomplete or unclear",
			Impact:      "May require additional clarification",
		})
	}

	outputFiles := []string{}
	if summaryPath != "" {
		outputFiles = append(outputFiles, summaryPath)
	}

	return &StageResult{
		Status:      StageStatusSuccess,
		OutputFiles: outputFiles,
		Notes:       truncateOutput(output, 300),
		Issues:      issues,
	}
}

// BuildSummaryPrompt creates a prompt for the Summary stage.
func BuildSummaryPrompt(taskTitle, taskDescription string, requirements []string, feedback string) string {
	var sb strings.Builder

	sb.WriteString("# Summarize Research: ")
	sb.WriteString(taskTitle)
	sb.WriteString("\n\n")

	sb.WriteString("## Description\n")
	sb.WriteString(taskDescription)
	sb.WriteString("\n\n")

	if len(requirements) > 0 {
		sb.WriteString("## Requirements for Summary\n")
		for _, req := range requirements {
			sb.WriteString("- ")
			sb.WriteString(req)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if feedback != "" {
		sb.WriteString("## Feedback from Previous Review\n")
		sb.WriteString("Please address the following feedback:\n\n")
		sb.WriteString(feedback)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Instructions\n")
	sb.WriteString("Read the findings.md file (provided in context) and create a clear, actionable summary. ")
	sb.WriteString("The summary should:\n\n")
	sb.WriteString("1. **Highlight key findings** - What are the most important discoveries?\n")
	sb.WriteString("2. **Provide recommendations** - What should be done based on the research?\n")
	sb.WriteString("3. **Note tradeoffs** - What are the pros/cons of different approaches?\n")
	sb.WriteString("4. **Include next steps** - What actions should follow this research?\n\n")
	sb.WriteString("Keep the summary concise but comprehensive. Focus on actionable insights that ")
	sb.WriteString("can guide decision-making or implementation.\n")

	return sb.String()
}
