package pipeline

import (
	"fmt"

	"github.com/mmariani/ground-control/internal/session"
	"github.com/mmariani/ground-control/internal/types"
)

// Config holds pipeline configuration.
type Config struct {
	MaxIterations    int  // Max feedback loop iterations before escalation
	SkipReview       bool // Skip code review stage
	SkipTest         bool // Skip test stage
	SkipDocs         bool // Skip documentation update
	Verbose          bool
}

// DefaultConfig returns the default pipeline configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxIterations: 3,
		SkipReview:    false,
		SkipTest:      false,
		SkipDocs:      false,
		Verbose:       false,
	}
}

// Executor runs tasks through the pipeline stages.
type Executor struct {
	config  *Config
	stages  []Stage
	store   TaskStore
	sessMgr *session.Manager
}

// NewExecutor creates a new pipeline executor.
func NewExecutor(store TaskStore, sessMgr *session.Manager, config *Config) *Executor {
	if config == nil {
		config = DefaultConfig()
	}

	return &Executor{
		config:  config,
		store:   store,
		sessMgr: sessMgr,
		stages:  []Stage{},
	}
}

// AddStage adds a stage to the pipeline.
func (e *Executor) AddStage(stage Stage) {
	e.stages = append(e.stages, stage)
}

// Execute runs a task through all pipeline stages.
func (e *Executor) Execute(task *types.Task, sess *session.Session) error {
	ctx := &StageContext{
		Task:    task,
		Session: sess,
		Verbose: e.config.Verbose,
	}

	for _, stage := range e.stages {
		// Check if stage can be skipped
		if stage.CanSkip(ctx) {
			sess.SetCurrentStage(stage.Name(), 0)
			sess.CompleteStage(stage.Name(), nil)
			e.sessMgr.Save(sess)
			continue
		}

		// Execute stage with potential feedback loop
		result, err := e.executeStageWithLoop(stage, ctx, sess)
		if err != nil {
			return err
		}

		// Handle stage result
		switch result.Status {
		case StageStatusSuccess:
			sess.CompleteStage(stage.Name(), result.OutputFiles)
			e.sessMgr.Save(sess)

		case StageStatusSkipped:
			sess.CompleteStage(stage.Name(), nil)
			e.sessMgr.Save(sess)

		case StageStatusEscalate:
			sess.EscalateTask(task.ID)
			e.sessMgr.Save(sess)
			return fmt.Errorf("stage %s escalated to human", stage.Name())

		case StageStatusFailed:
			sess.FailStage(stage.Name(), result.Error.Error())
			e.sessMgr.Save(sess)
			return fmt.Errorf("stage %s failed: %w", stage.Name(), result.Error)
		}

		// Collect issues for self-learning
		for _, issue := range result.Issues {
			sess.AddIssue(issue)
		}
	}

	return nil
}

// executeStageWithLoop handles feedback loops between stages.
func (e *Executor) executeStageWithLoop(stage Stage, ctx *StageContext, sess *session.Session) (*StageResult, error) {
	iteration := 1
	var lastFeedback string

	for {
		ctx.Iteration = iteration
		ctx.PreviousFeedback = lastFeedback

		sess.SetCurrentStage(stage.Name(), iteration)
		e.sessMgr.Save(sess)

		result := stage.Execute(ctx)

		// Add issues to session
		for _, issue := range result.Issues {
			sess.AddIssue(issue)
		}

		switch result.Status {
		case StageStatusSuccess, StageStatusSkipped, StageStatusFailed, StageStatusEscalate:
			return result, nil

		case StageStatusNeedsRevision:
			// Check iteration limit
			if iteration >= e.config.MaxIterations {
				// Escalate after max iterations
				return &StageResult{
					Status: StageStatusEscalate,
					Notes:  fmt.Sprintf("Max iterations (%d) reached for stage %s", e.config.MaxIterations, stage.Name()),
					Feedback: result.Feedback,
				}, nil
			}

			// Store feedback for next iteration
			lastFeedback = result.Feedback
			iteration++
		}
	}
}

// ExecuteWithFeedbackLoop runs coder/reviewer in a feedback loop.
func (e *Executor) ExecuteWithFeedbackLoop(coderStage, reviewerStage Stage, ctx *StageContext, sess *session.Session) (*StageResult, error) {
	iteration := 1
	var coderResult *StageResult

	for {
		ctx.Iteration = iteration

		// Run coder
		sess.SetCurrentStage(StageNameCoder, iteration)
		e.sessMgr.Save(sess)

		coderResult = coderStage.Execute(ctx)
		if coderResult.Status == StageStatusFailed {
			return coderResult, nil
		}

		// Run reviewer
		sess.SetCurrentStage(StageNameReviewer, iteration)
		e.sessMgr.Save(sess)

		reviewResult := reviewerStage.Execute(ctx)

		switch reviewResult.Status {
		case StageStatusSuccess:
			// Code approved
			sess.CompleteStage(StageNameCoder, coderResult.OutputFiles)
			sess.CompleteStage(StageNameReviewer, reviewResult.OutputFiles)
			return coderResult, nil

		case StageStatusNeedsRevision:
			if iteration >= e.config.MaxIterations {
				return &StageResult{
					Status: StageStatusEscalate,
					Notes:  fmt.Sprintf("Max iterations (%d) reached in coder/reviewer loop", e.config.MaxIterations),
					Feedback: reviewResult.Feedback,
				}, nil
			}
			// Pass feedback to coder for next iteration
			ctx.PreviousFeedback = reviewResult.Feedback
			iteration++

		case StageStatusEscalate, StageStatusFailed:
			return reviewResult, nil
		}
	}
}

// BuildDefaultPipeline creates a pipeline with standard stages.
func BuildDefaultPipeline(store TaskStore, sessMgr *session.Manager, config *Config) *Executor {
	exec := NewExecutor(store, sessMgr, config)

	// Add sanity check
	exec.AddStage(NewSanityStage(store))

	// Coder and Reviewer stages will be added when implemented
	// exec.AddStage(NewCoderStage(...))
	// exec.AddStage(NewReviewerStage(...))

	return exec
}
