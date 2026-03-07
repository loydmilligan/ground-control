package pipeline

import (
	"sync"

	"github.com/mmariani/ground-control/internal/types"
)

// ParallelExecutor manages concurrent task execution with limits.
type ParallelExecutor struct {
	config    *ParallelConfig
	semaphore chan struct{}
}

// NewParallelExecutor creates a new parallel executor with the given config.
func NewParallelExecutor(config *ParallelConfig) *ParallelExecutor {
	if config == nil {
		config = DefaultParallelConfig()
	}

	// Create semaphore channel to limit concurrency
	sem := make(chan struct{}, config.MaxConcurrentTasks)

	return &ParallelExecutor{
		config:    config,
		semaphore: sem,
	}
}

// ExecuteParallel executes tasks in parallel, respecting concurrency limits.
// Returns a slice of errors (nil for successful tasks) in the same order as input tasks.
func (e *ParallelExecutor) ExecuteParallel(tasks []types.Task, execFunc func(types.Task) error) []error {
	if !e.config.Enabled || len(tasks) <= 1 {
		// Fall back to sequential execution
		return e.executeSequential(tasks, execFunc)
	}

	errors := make([]error, len(tasks))
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)

		// Capture loop variables
		idx := i
		t := task

		go func() {
			defer wg.Done()

			// Acquire semaphore
			e.semaphore <- struct{}{}
			defer func() { <-e.semaphore }()

			// Execute task
			errors[idx] = execFunc(t)
		}()
	}

	wg.Wait()
	return errors
}

// executeSequential executes tasks one at a time.
func (e *ParallelExecutor) executeSequential(tasks []types.Task, execFunc func(types.Task) error) []error {
	errors := make([]error, len(tasks))
	for i, task := range tasks {
		errors[i] = execFunc(task)
	}
	return errors
}

// ExecuteGroups executes task groups in sequence, but tasks within each group in parallel.
func (e *ParallelExecutor) ExecuteGroups(groups [][]string, taskMap map[string]types.Task, execFunc func(types.Task) error) []error {
	var allErrors []error

	for _, group := range groups {
		// Build tasks for this group
		groupTasks := make([]types.Task, 0, len(group))
		for _, taskID := range group {
			if task, ok := taskMap[taskID]; ok {
				groupTasks = append(groupTasks, task)
			}
		}

		// Execute group in parallel
		groupErrors := e.ExecuteParallel(groupTasks, execFunc)
		allErrors = append(allErrors, groupErrors...)
	}

	return allErrors
}
