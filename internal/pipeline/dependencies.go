package pipeline

import (
	"github.com/mmariani/ground-control/internal/types"
)

// AnalyzeDependencies determines which tasks can run in parallel.
// Returns groups of task IDs where each group can run in parallel.
// Tasks with blocked_by dependencies go in later groups.
func AnalyzeDependencies(tasks []types.Task) [][]string {
	if len(tasks) == 0 {
		return nil
	}

	// Build task ID set for quick lookup
	taskIDSet := make(map[string]bool)
	for _, t := range tasks {
		taskIDSet[t.ID] = true
	}

	// Track which tasks have been assigned to groups
	assigned := make(map[string]bool)
	var groups [][]string

	// Process tasks in waves
	for len(assigned) < len(tasks) {
		var currentGroup []string

		// First pass: identify tasks that can run
		for _, task := range tasks {
			if assigned[task.ID] {
				continue
			}

			// Check if all dependencies are satisfied
			canRun := true
			for _, blockerID := range task.BlockedBy {
				// Only consider blockers that are in our task set
				if taskIDSet[blockerID] && !assigned[blockerID] {
					canRun = false
					break
				}
			}

			if canRun {
				currentGroup = append(currentGroup, task.ID)
			}
		}

		// If no tasks can run, we have a circular dependency
		if len(currentGroup) == 0 {
			break
		}

		// Second pass: mark tasks as assigned
		for _, taskID := range currentGroup {
			assigned[taskID] = true
		}

		groups = append(groups, currentGroup)
	}

	return groups
}

// CanParallelize checks if a set of tasks can be parallelized.
// Returns true if at least one group has multiple tasks.
func CanParallelize(tasks []types.Task) bool {
	groups := AnalyzeDependencies(tasks)
	for _, group := range groups {
		if len(group) > 1 {
			return true
		}
	}
	return false
}
