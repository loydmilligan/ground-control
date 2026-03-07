// Package pipeline implements the task execution pipeline.
package pipeline

import (
	"github.com/mmariani/ground-control/internal/session"
	"github.com/mmariani/ground-control/internal/types"
)

// StageStatus represents the result status of a stage execution.
type StageStatus string

const (
	StageStatusSuccess      StageStatus = "success"
	StageStatusNeedsRevision StageStatus = "needs_revision"
	StageStatusEscalate     StageStatus = "escalate"
	StageStatusSkipped      StageStatus = "skipped"
	StageStatusFailed       StageStatus = "failed"
)

// StageResult contains the output of a stage execution.
type StageResult struct {
	Status      StageStatus
	OutputFiles []string
	Notes       string
	Feedback    string // For needs_revision - specific feedback for previous stage
	Issues      []session.SessionIssue
	Error       error
}

// StageContext provides context for stage execution.
type StageContext struct {
	Task           *types.Task
	Session        *session.Session
	Iteration      int
	PreviousFeedback string // Feedback from reviewer/tester if in revision loop
	Verbose        bool
	DryRun         bool
}

// Stage defines the interface for pipeline stages.
type Stage interface {
	// Name returns the stage name for logging and tracking.
	Name() string

	// Execute runs the stage with the given context.
	Execute(ctx *StageContext) *StageResult

	// CanSkip returns true if this stage can be skipped based on config.
	CanSkip(ctx *StageContext) bool
}

// StageName constants for consistency.
const (
	StageNameSanity         = "sanity"
	StageNameCoder          = "coder"
	StageNameReviewer       = "reviewer"
	StageNameTester         = "tester"
	StageNameRepoUpdate     = "repo_update"
	StageNameSessionReview  = "session_review"
	StageNameResearcher     = "researcher"
	StageNameSummary        = "summary"
	StageNameNotify         = "notify"
	StageNameWaitHuman      = "wait_human"
	StageNameCaptureResponse = "capture_response"
)
