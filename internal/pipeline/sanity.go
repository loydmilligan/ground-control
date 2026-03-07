package pipeline

import (
	"errors"
	"fmt"
	"os"

	"github.com/mmariani/ground-control/internal/types"
)

// ErrNoContextBundle is returned when a task has no context bundle.
var ErrNoContextBundle = errors.New("task has no context bundle")

// SanityStage verifies a task is ready for execution.
type SanityStage struct {
	store TaskStore
}

// TaskStore interface for loading tasks (to check blockers).
type TaskStore interface {
	LoadTasks() ([]types.Task, error)
}

// NewSanityStage creates a new sanity check stage.
func NewSanityStage(store TaskStore) *SanityStage {
	return &SanityStage{store: store}
}

// Name returns the stage name.
func (s *SanityStage) Name() string {
	return StageNameSanity
}

// CanSkip returns false - sanity check should never be skipped.
func (s *SanityStage) CanSkip(ctx *StageContext) bool {
	return false
}

// Execute runs the sanity check.
func (s *SanityStage) Execute(ctx *StageContext) *StageResult {
	task := ctx.Task
	var issues []string

	// Check 1: Task exists and has required fields
	if task.ID == "" {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("task has no ID"),
		}
	}

	if task.Title == "" {
		issues = append(issues, "task has no title")
	}

	// Check 2: Context bundle exists
	if task.ContextBundle == nil {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  ErrNoContextBundle,
			Notes:  "Context bundle is required for pipeline execution",
		}
	}

	// Check 3: Context bundle files exist
	bundle := task.ContextBundle
	if bundle.Files.Requirements != "" {
		if _, err := os.Stat(bundle.Files.Requirements); os.IsNotExist(err) {
			issues = append(issues, "requirements.md not found: "+bundle.Files.Requirements)
		}
	}

	// Check 4: Blocked tasks are completed
	if len(task.BlockedBy) > 0 {
		allTasks, err := s.store.LoadTasks()
		if err != nil {
			return &StageResult{
				Status: StageStatusFailed,
				Error:  fmt.Errorf("could not load tasks to check blockers: %w", err),
			}
		}

		for _, blockerID := range task.BlockedBy {
			blocker := findTaskByID(allTasks, blockerID)
			if blocker == nil {
				issues = append(issues, fmt.Sprintf("blocking task not found: %s", blockerID))
			} else if blocker.State != types.TaskStateCompleted {
				return &StageResult{
					Status: StageStatusFailed,
					Error:  fmt.Errorf("blocked by incomplete task: %s (%s)", blocker.Title, blockerID),
					Notes:  "Complete blocking tasks first or remove the dependency",
				}
			}
		}
	}

	// Check 5: Task is in valid state for execution
	switch task.State {
	case types.TaskStateCompleted:
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("task is already completed"),
		}
	case types.TaskStateBlocked:
		// This should have been caught above, but double-check
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("task is in blocked state"),
		}
	}

	// Check 6: Task has requirements or description
	if task.Description == "" && len(task.Context.Requirements) == 0 {
		issues = append(issues, "task has no description or requirements")
	}

	// If we have non-fatal issues, note them but pass
	notes := ""
	if len(issues) > 0 {
		notes = fmt.Sprintf("Warnings: %v", issues)
	}

	return &StageResult{
		Status: StageStatusSuccess,
		Notes:  notes,
	}
}

func findTaskByID(tasks []types.Task, id string) *types.Task {
	for i := range tasks {
		if tasks[i].ID == id {
			return &tasks[i]
		}
	}
	return nil
}
