package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mmariani/ground-control/internal/session"
)

// SessionReviewStage reviews the session after task completion.
// This is the Taskmaster's review step for self-learning.
type SessionReviewStage struct {
	dataDir string
}

// NewSessionReviewStage creates a new session review stage.
func NewSessionReviewStage(dataDir string) *SessionReviewStage {
	return &SessionReviewStage{
		dataDir: dataDir,
	}
}

// Name returns the stage name.
func (s *SessionReviewStage) Name() string {
	return StageNameSessionReview
}

// CanSkip returns true if there are no issues to review.
func (s *SessionReviewStage) CanSkip(ctx *StageContext) bool {
	return len(ctx.Session.Issues) == 0
}

// Execute runs the session review stage.
func (s *SessionReviewStage) Execute(ctx *StageContext) *StageResult {
	sess := ctx.Session

	// Generate session summary
	summary := s.generateSessionSummary(sess)

	if ctx.Verbose {
		fmt.Printf("    Session summary:\n%s\n", summary)
	}

	// Process issues for learning
	if len(sess.Issues) > 0 {
		if err := s.writeToLearningLog(sess); err != nil {
			return &StageResult{
				Status: StageStatusFailed,
				Error:  fmt.Errorf("writing to learning log: %w", err),
			}
		}

		if ctx.Verbose {
			fmt.Printf("    Logged %d issues to learning log\n", len(sess.Issues))
		}
	}

	// Save session summary to file
	summaryPath := filepath.Join(s.dataDir, "sessions", sess.ID+"_summary.md")
	if err := os.WriteFile(summaryPath, []byte(summary), 0644); err != nil {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("writing session summary: %w", err),
		}
	}

	return &StageResult{
		Status:      StageStatusSuccess,
		OutputFiles: []string{summaryPath},
		Notes:       fmt.Sprintf("Session reviewed: %d tasks, %d issues", len(sess.TaskIDs), len(sess.Issues)),
	}
}

// generateSessionSummary creates a markdown summary of the session.
func (s *SessionReviewStage) generateSessionSummary(sess *session.Session) string {
	var summary string

	summary += fmt.Sprintf("# Session Summary: %s\n\n", sess.ID)
	summary += fmt.Sprintf("**Started**: %s\n", sess.StartedAt.Format(time.RFC3339))
	summary += fmt.Sprintf("**Status**: %s\n\n", sess.Status)

	// Task summary
	summary += "## Tasks\n\n"
	var completed, failed, escalated int
	for taskID, progress := range sess.TaskProgress {
		status := progress.Status
		switch status {
		case "completed":
			completed++
		case "failed":
			failed++
		case "escalated":
			escalated++
		}
		summary += fmt.Sprintf("- **%s**: %s\n", taskID, status)

		// Stage details
		for stageName, stage := range progress.Stages {
			summary += fmt.Sprintf("  - %s: %s", stageName, stage.Status)
			if stage.Iterations > 1 {
				summary += fmt.Sprintf(" (%d iterations)", stage.Iterations)
			}
			summary += "\n"
		}
	}
	summary += "\n"

	// Summary stats
	summary += "## Statistics\n\n"
	summary += fmt.Sprintf("- Total tasks: %d\n", len(sess.TaskIDs))
	summary += fmt.Sprintf("- Completed: %d\n", completed)
	summary += fmt.Sprintf("- Failed: %d\n", failed)
	summary += fmt.Sprintf("- Escalated: %d\n", escalated)
	summary += "\n"

	// Issues
	if len(sess.Issues) > 0 {
		summary += "## Issues Identified\n\n"
		for _, issue := range sess.Issues {
			summary += fmt.Sprintf("### %s: %s\n", issue.Severity, issue.Description)
			summary += fmt.Sprintf("- **Task**: %s\n", issue.TaskID)
			summary += fmt.Sprintf("- **Stage**: %s\n", issue.Stage)
			summary += fmt.Sprintf("- **Category**: %s\n", issue.Category)
			summary += fmt.Sprintf("- **Impact**: %s\n", issue.Impact)
			if issue.Suggestion != "" {
				summary += fmt.Sprintf("- **Suggestion**: %s\n", issue.Suggestion)
			}
			summary += "\n"
		}
	}

	// Lessons learned section
	summary += "## Lessons for Future Sessions\n\n"
	if len(sess.Issues) == 0 {
		summary += "No significant issues identified. Session completed smoothly.\n"
	} else {
		summary += "The following patterns were observed:\n\n"
		categoryCount := make(map[string]int)
		for _, issue := range sess.Issues {
			categoryCount[issue.Category]++
		}
		for cat, count := range categoryCount {
			summary += fmt.Sprintf("- **%s**: %d occurrences\n", cat, count)
		}
	}

	return summary
}

// LearningEntry represents an entry in the learning log.
type LearningEntry struct {
	SessionID string                  `json:"session_id"`
	Timestamp time.Time               `json:"timestamp"`
	Issues    []session.SessionIssue  `json:"issues"`
	Summary   LearningEntrySummary    `json:"summary"`
}

// LearningEntrySummary contains aggregated learning data.
type LearningEntrySummary struct {
	TotalTasks      int            `json:"total_tasks"`
	CompletedTasks  int            `json:"completed_tasks"`
	FailedTasks     int            `json:"failed_tasks"`
	EscalatedTasks  int            `json:"escalated_tasks"`
	CategoryCounts  map[string]int `json:"category_counts"`
	SeverityCounts  map[string]int `json:"severity_counts"`
}

// writeToLearningLog appends session issues to the learning log.
func (s *SessionReviewStage) writeToLearningLog(sess *session.Session) error {
	logPath := filepath.Join(s.dataDir, "learning-log.json")

	// Load existing log
	var entries []LearningEntry
	if data, err := os.ReadFile(logPath); err == nil {
		json.Unmarshal(data, &entries)
	}

	// Build summary
	var completed, failed, escalated int
	for _, progress := range sess.TaskProgress {
		switch progress.Status {
		case "completed":
			completed++
		case "failed":
			failed++
		case "escalated":
			escalated++
		}
	}

	categoryCount := make(map[string]int)
	severityCount := make(map[string]int)
	for _, issue := range sess.Issues {
		categoryCount[issue.Category]++
		severityCount[issue.Severity]++
	}

	// Create new entry
	entry := LearningEntry{
		SessionID: sess.ID,
		Timestamp: time.Now(),
		Issues:    sess.Issues,
		Summary: LearningEntrySummary{
			TotalTasks:     len(sess.TaskIDs),
			CompletedTasks: completed,
			FailedTasks:    failed,
			EscalatedTasks: escalated,
			CategoryCounts: categoryCount,
			SeverityCounts: severityCount,
		},
	}

	entries = append(entries, entry)

	// Write back
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling learning log: %w", err)
	}

	return os.WriteFile(logPath, data, 0644)
}

// ValidateIssues filters out low-quality issues.
// Issues must have actual problems, not just observations.
func ValidateIssues(issues []session.SessionIssue) []session.SessionIssue {
	var valid []session.SessionIssue
	for _, issue := range issues {
		// Skip issues without actionable content
		if issue.Description == "" || issue.Category == "" {
			continue
		}
		// Skip "everything went fine" non-issues
		if issue.Severity == "minor" && issue.Impact == "" {
			continue
		}
		valid = append(valid, issue)
	}
	return valid
}
