package pipeline

import (
	"fmt"

	"github.com/mmariani/ground-control/internal/session"
)

// FeedbackLoop manages the revision cycle between Coder and Reviewer.
type FeedbackLoop struct {
	coderStage    Stage
	reviewerStage Stage
	maxIterations int
	sessMgr       *session.Manager
}

// NewFeedbackLoop creates a new feedback loop handler.
func NewFeedbackLoop(coder, reviewer Stage, maxIterations int, sessMgr *session.Manager) *FeedbackLoop {
	if maxIterations <= 0 {
		maxIterations = 3
	}
	return &FeedbackLoop{
		coderStage:    coder,
		reviewerStage: reviewer,
		maxIterations: maxIterations,
		sessMgr:       sessMgr,
	}
}

// FeedbackLoopResult contains the outcome of a feedback loop.
type FeedbackLoopResult struct {
	Status          StageStatus
	Iterations      int
	CoderOutputs    []string
	ReviewerOutputs []string
	FinalFeedback   string
	Issues          []session.SessionIssue
}

// Execute runs the coder-reviewer feedback loop until approval, escalation, or max iterations.
func (fl *FeedbackLoop) Execute(ctx *StageContext) (*FeedbackLoopResult, error) {
	result := &FeedbackLoopResult{
		CoderOutputs:    []string{},
		ReviewerOutputs: []string{},
		Issues:          []session.SessionIssue{},
	}

	sess := ctx.Session
	var lastFeedback string

	for iteration := 1; iteration <= fl.maxIterations; iteration++ {
		result.Iterations = iteration

		// Update context for this iteration
		ctx.Iteration = iteration
		ctx.PreviousFeedback = lastFeedback

		// Run Coder stage
		sess.SetCurrentStage(StageNameCoder, iteration)
		fl.sessMgr.Save(sess)

		coderResult := fl.coderStage.Execute(ctx)
		result.CoderOutputs = append(result.CoderOutputs, coderResult.OutputFiles...)
		result.Issues = append(result.Issues, coderResult.Issues...)

		if coderResult.Status == StageStatusFailed {
			result.Status = StageStatusFailed
			return result, fmt.Errorf("coder stage failed: %v", coderResult.Error)
		}

		sess.CompleteStage(StageNameCoder, coderResult.OutputFiles)
		fl.sessMgr.Save(sess)

		// Run Reviewer stage
		sess.SetCurrentStage(StageNameReviewer, iteration)
		fl.sessMgr.Save(sess)

		reviewerResult := fl.reviewerStage.Execute(ctx)
		result.ReviewerOutputs = append(result.ReviewerOutputs, reviewerResult.OutputFiles...)
		result.Issues = append(result.Issues, reviewerResult.Issues...)

		switch reviewerResult.Status {
		case StageStatusSuccess:
			// Approved!
			result.Status = StageStatusSuccess
			sess.CompleteStage(StageNameReviewer, reviewerResult.OutputFiles)
			fl.sessMgr.Save(sess)
			return result, nil

		case StageStatusNeedsRevision:
			// Continue loop with feedback
			lastFeedback = reviewerResult.Feedback
			result.FinalFeedback = reviewerResult.Feedback

			// Log iteration for learning
			result.Issues = append(result.Issues, session.SessionIssue{
				TaskID:      ctx.Task.ID,
				Stage:       StageNameReviewer,
				Severity:    "minor",
				Category:    "review_failure",
				Description: fmt.Sprintf("Code review iteration %d requested revision", iteration),
				Impact:      reviewerResult.Notes,
			})

			if iteration == fl.maxIterations {
				// Max iterations reached, escalate
				result.Status = StageStatusEscalate
				sess.FailStage(StageNameReviewer, "max iterations reached")
				fl.sessMgr.Save(sess)
				return result, nil
			}

			// Continue to next iteration
			continue

		case StageStatusEscalate:
			result.Status = StageStatusEscalate
			result.FinalFeedback = reviewerResult.Feedback
			sess.CompleteStage(StageNameReviewer, reviewerResult.OutputFiles)
			fl.sessMgr.Save(sess)
			return result, nil

		case StageStatusFailed:
			result.Status = StageStatusFailed
			sess.FailStage(StageNameReviewer, reviewerResult.Error.Error())
			fl.sessMgr.Save(sess)
			return result, fmt.Errorf("reviewer stage failed: %v", reviewerResult.Error)
		}
	}

	// Should not reach here, but just in case
	result.Status = StageStatusEscalate
	return result, nil
}

// GetIterationSummary returns a summary of feedback loop iterations for escalation.
func (fl *FeedbackLoop) GetIterationSummary(result *FeedbackLoopResult) string {
	summary := fmt.Sprintf("Feedback loop completed after %d iteration(s).\n\n", result.Iterations)

	summary += "## Status\n"
	summary += fmt.Sprintf("Final status: %s\n\n", result.Status)

	if len(result.CoderOutputs) > 0 {
		summary += "## Coder Outputs\n"
		for _, f := range result.CoderOutputs {
			summary += fmt.Sprintf("- %s\n", f)
		}
		summary += "\n"
	}

	if len(result.ReviewerOutputs) > 0 {
		summary += "## Reviewer Outputs\n"
		for _, f := range result.ReviewerOutputs {
			summary += fmt.Sprintf("- %s\n", f)
		}
		summary += "\n"
	}

	if result.FinalFeedback != "" {
		summary += "## Final Feedback\n"
		summary += result.FinalFeedback + "\n"
	}

	return summary
}
