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

// RepoUpdateStage handles documentation, commits, and optional deployment.
type RepoUpdateStage struct {
	projectDir string
}

// NewRepoUpdateStage creates a new Repo Update stage.
func NewRepoUpdateStage(projectDir string) *RepoUpdateStage {
	return &RepoUpdateStage{
		projectDir: projectDir,
	}
}

// Name returns the stage name.
func (s *RepoUpdateStage) Name() string {
	return StageNameRepoUpdate
}

// CanSkip returns false - repo update should not be skipped.
func (s *RepoUpdateStage) CanSkip(ctx *StageContext) bool {
	return false
}

// Execute runs the Repo Update stage.
func (s *RepoUpdateStage) Execute(ctx *StageContext) *StageResult {
	task := ctx.Task
	var outputFiles []string
	var issues []session.SessionIssue

	// Step 1: Check for changes to commit
	hasChanges, err := s.hasUncommittedChanges()
	if err != nil {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("checking git status: %w", err),
		}
	}

	if !hasChanges {
		return &StageResult{
			Status: StageStatusSuccess,
			Notes:  "No changes to commit",
		}
	}

	// Step 2: Generate commit message
	commitMsg := s.generateCommitMessage(task, ctx)

	if ctx.Verbose {
		fmt.Printf("    Commit message:\n%s\n", commitMsg)
	}

	// Step 3: Stage changes
	if err := s.stageChanges(); err != nil {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("staging changes: %w", err),
		}
	}

	// Step 4: Create commit
	commitHash, err := s.createCommit(commitMsg)
	if err != nil {
		// Check if it was a pre-commit hook failure
		if strings.Contains(err.Error(), "hook") {
			issues = append(issues, session.SessionIssue{
				TaskID:      task.ID,
				Stage:       StageNameRepoUpdate,
				Severity:    "moderate",
				Category:    "tooling_issue",
				Description: "Pre-commit hook failed",
				Impact:      "Commit was not created",
				Suggestion:  "Fix hook issues and retry",
			})
			return &StageResult{
				Status:   StageStatusNeedsRevision,
				Feedback: fmt.Sprintf("Pre-commit hook failed: %v\n\nPlease fix the issues and ensure code passes all hooks.", err),
				Issues:   issues,
			}
		}
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("creating commit: %w", err),
		}
	}

	// Save commit summary
	summaryPath := ""
	if task.ContextBundle != nil {
		summaryPath = filepath.Join(task.ContextBundle.BundlePath, "commit_summary.md")
		content := fmt.Sprintf("# Commit Summary\n\n**Task**: %s\n**Time**: %s\n**Commit**: %s\n\n## Message\n\n```\n%s\n```",
			task.Title, time.Now().Format(time.RFC3339), commitHash, commitMsg)
		os.WriteFile(summaryPath, []byte(content), 0644)
		outputFiles = append(outputFiles, summaryPath)
	}

	if ctx.Verbose {
		fmt.Printf("    Created commit: %s\n", commitHash)
	}

	// Step 5: Optional deployment (future)
	// if task has deploy_target, trigger deployment

	return &StageResult{
		Status:      StageStatusSuccess,
		OutputFiles: outputFiles,
		Notes:       fmt.Sprintf("Committed: %s", commitHash[:8]),
		Issues:      issues,
	}
}

// hasUncommittedChanges checks if there are changes to commit.
func (s *RepoUpdateStage) hasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = s.projectDir

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return len(bytes.TrimSpace(output)) > 0, nil
}

// generateCommitMessage creates a commit message from task context.
func (s *RepoUpdateStage) generateCommitMessage(task interface{}, ctx *StageContext) string {
	// Type assert to get task details
	type taskLike interface {
		GetTitle() string
		GetDescription() string
		GetID() string
	}

	var title, description, taskID string

	// Try to extract from context
	if ctx.Task != nil {
		title = ctx.Task.Title
		description = ctx.Task.Description
		taskID = ctx.Task.ID
	}

	// Truncate title if needed
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	// Build commit message
	var msg strings.Builder
	msg.WriteString(title)
	msg.WriteString("\n\n")

	if description != "" {
		// Truncate description for commit
		desc := description
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}
		msg.WriteString(desc)
		msg.WriteString("\n\n")
	}

	msg.WriteString(fmt.Sprintf("Task: %s\n", taskID))
	msg.WriteString("\n")
	msg.WriteString("Co-Authored-By: Claude <noreply@anthropic.com>\n")

	return msg.String()
}

// stageChanges stages all changes for commit.
func (s *RepoUpdateStage) stageChanges() error {
	// Stage all changes except certain files
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = s.projectDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", stderr.String(), err)
	}

	// Unstage sensitive files if accidentally added
	sensitivePatterns := []string{".env", "*.key", "*.pem", "credentials*"}
	for _, pattern := range sensitivePatterns {
		exec.Command("git", "reset", "HEAD", "--", pattern).Run()
	}

	return nil
}

// createCommit creates a git commit with the given message.
func (s *RepoUpdateStage) createCommit(message string) (string, error) {
	// Create commit
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = s.projectDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s%s", stderr.String(), stdout.String())
	}

	// Get commit hash
	hashCmd := exec.Command("git", "rev-parse", "HEAD")
	hashCmd.Dir = s.projectDir

	hash, err := hashCmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting commit hash: %w", err)
	}

	return strings.TrimSpace(string(hash)), nil
}

// getChangedFiles returns a list of files that changed.
func (s *RepoUpdateStage) getChangedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = s.projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}
