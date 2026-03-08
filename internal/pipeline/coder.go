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

// CoderStage implements code changes using Claude Code CLI.
type CoderStage struct {
	client     *claude.Client
	projectDir string
}

// NewCoderStage creates a new Coder stage.
func NewCoderStage(client *claude.Client, projectDir string) *CoderStage {
	return &CoderStage{
		client:     client,
		projectDir: projectDir,
	}
}

// Name returns the stage name.
func (s *CoderStage) Name() string {
	return StageNameCoder
}

// CanSkip returns false - coder stage should not be skipped.
func (s *CoderStage) CanSkip(ctx *StageContext) bool {
	return false
}

// Execute runs the Coder stage.
func (s *CoderStage) Execute(ctx *StageContext) *StageResult {
	task := ctx.Task

	// Build the prompt
	prompt := claude.BuildCoderPrompt(
		task.Title,
		task.Description,
		task.Context.Requirements,
		ctx.PreviousFeedback,
	)

	// Get context files from bundle
	var contextFiles []string
	if task.ContextBundle != nil && task.ContextBundle.BundlePath != "" {
		contextFiles = claude.GetContextFiles(task.ContextBundle.BundlePath)
	}

	// Get working directory from task context
	workingDir := ""
	if task.Context.WorkingDirectory != nil {
		workingDir = *task.Context.WorkingDirectory
	}

	// Execute Claude
	var output string
	req := &claude.Request{
		Prompt:           prompt,
		ContextFiles:     contextFiles,
		WorkingDirectory: workingDir,
	}

	if ctx.Verbose {
		fmt.Printf("    Invoking Claude Code CLI...\n")
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

	// Save implementation notes
	notesPath := ""
	if task.ContextBundle != nil {
		notesPath = filepath.Join(task.ContextBundle.BundlePath, "implementation_notes.md")
		notes := fmt.Sprintf("# Implementation Notes\n\n**Task**: %s\n**Iteration**: %d\n**Time**: %s\n\n## Output\n\n%s",
			task.Title, ctx.Iteration, time.Now().Format(time.RFC3339), output)
		os.WriteFile(notesPath, []byte(notes), 0644)
	}

	// Check for issues in output
	var issues []session.SessionIssue
	if strings.Contains(strings.ToLower(output), "error") || strings.Contains(strings.ToLower(output), "failed") {
		// Only add issue if it seems like a real problem, not just discussing errors
		if strings.Contains(strings.ToLower(output), "could not") || strings.Contains(strings.ToLower(output), "unable to") {
			issues = append(issues, session.SessionIssue{
				TaskID:      task.ID,
				Stage:       StageNameCoder,
				Severity:    "minor",
				Category:    "tooling_issue",
				Description: "Coder encountered potential issues during implementation",
				Impact:      "May require additional review",
			})
		}
	}

	outputFiles := []string{}
	if notesPath != "" {
		outputFiles = append(outputFiles, notesPath)
	}

	return &StageResult{
		Status:      StageStatusSuccess,
		OutputFiles: outputFiles,
		Notes:       truncateOutput(output, 500),
		Issues:      issues,
	}
}

func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
