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

// ResearcherStage conducts research on topics using Claude.
type ResearcherStage struct {
	client     *claude.Client
	projectDir string
}

// NewResearcherStage creates a new Researcher stage.
func NewResearcherStage(client *claude.Client, projectDir string) *ResearcherStage {
	return &ResearcherStage{
		client:     client,
		projectDir: projectDir,
	}
}

// Name returns the stage name.
func (s *ResearcherStage) Name() string {
	return StageNameResearcher
}

// CanSkip returns false - researcher stage should not be skipped.
func (s *ResearcherStage) CanSkip(ctx *StageContext) bool {
	return false
}

// Execute runs the Researcher stage.
func (s *ResearcherStage) Execute(ctx *StageContext) *StageResult {
	task := ctx.Task

	// Build the research prompt
	prompt := BuildResearcherPrompt(
		task.Title,
		task.Description,
		task.Topics,
		task.Context.Requirements,
		ctx.PreviousFeedback,
	)

	// Get context files from bundle
	var contextFiles []string
	if task.ContextBundle != nil && task.ContextBundle.BundlePath != "" {
		contextFiles = claude.GetContextFiles(task.ContextBundle.BundlePath)
	}

	// Execute Claude
	var output string
	req := &claude.Request{
		Prompt:       prompt,
		ContextFiles: contextFiles,
	}

	if ctx.Verbose {
		fmt.Printf("    Invoking Claude Code CLI for research...\n")
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

	// Save research findings
	findingsPath := ""
	if task.ContextBundle != nil {
		findingsPath = filepath.Join(task.ContextBundle.BundlePath, "findings.md")
		content := fmt.Sprintf("# Research Findings\n\n**Task**: %s\n**Time**: %s\n\n## Topics\n\n",
			task.Title, time.Now().Format(time.RFC3339))

		for _, topic := range task.Topics {
			content += fmt.Sprintf("- %s\n", topic)
		}

		content += fmt.Sprintf("\n## Findings\n\n%s", output)

		if err := os.WriteFile(findingsPath, []byte(content), 0644); err != nil {
			return &StageResult{
				Status: StageStatusFailed,
				Error:  fmt.Errorf("failed to write findings: %w", err),
			}
		}
	}

	// Check for potential issues
	var issues []session.SessionIssue
	if strings.Contains(strings.ToLower(output), "could not find") ||
	   strings.Contains(strings.ToLower(output), "unable to research") {
		issues = append(issues, session.SessionIssue{
			TaskID:      task.ID,
			Stage:       StageNameResearcher,
			Severity:    "minor",
			Category:    "research_incomplete",
			Description: "Researcher encountered difficulties with some topics",
			Impact:      "Some research may be incomplete",
		})
	}

	outputFiles := []string{}
	if findingsPath != "" {
		outputFiles = append(outputFiles, findingsPath)
	}

	return &StageResult{
		Status:      StageStatusSuccess,
		OutputFiles: outputFiles,
		Notes:       truncateOutput(output, 300),
		Issues:      issues,
	}
}

// BuildResearcherPrompt creates a prompt for the Researcher stage.
func BuildResearcherPrompt(taskTitle, taskDescription string, topics []string, requirements []string, feedback string) string {
	var sb strings.Builder

	sb.WriteString("# Research Task: ")
	sb.WriteString(taskTitle)
	sb.WriteString("\n\n")

	sb.WriteString("## Description\n")
	sb.WriteString(taskDescription)
	sb.WriteString("\n\n")

	if len(topics) > 0 {
		sb.WriteString("## Topics to Research\n")
		for _, topic := range topics {
			sb.WriteString("- ")
			sb.WriteString(topic)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(requirements) > 0 {
		sb.WriteString("## Requirements\n")
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
	sb.WriteString("Research the topics listed above. For each topic:\n")
	sb.WriteString("1. Gather key information and findings\n")
	sb.WriteString("2. Identify relevant patterns, libraries, or approaches\n")
	sb.WriteString("3. Note any important considerations or tradeoffs\n")
	sb.WriteString("4. Provide concrete examples where helpful\n\n")
	sb.WriteString("Present your findings in a clear, organized format that can be used ")
	sb.WriteString("for decision-making or implementation planning.\n")

	return sb.String()
}
