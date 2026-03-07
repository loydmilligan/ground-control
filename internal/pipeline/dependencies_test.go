package pipeline

import (
	"testing"

	"github.com/mmariani/ground-control/internal/types"
)

func TestAnalyzeDependencies(t *testing.T) {
	tests := []struct {
		name          string
		tasks         []types.Task
		expectedWaves int
		expectedFirst int // Number of tasks in first wave
	}{
		{
			name:          "empty tasks",
			tasks:         []types.Task{},
			expectedWaves: 0,
			expectedFirst: 0,
		},
		{
			name: "single task",
			tasks: []types.Task{
				{ID: "task_1"},
			},
			expectedWaves: 1,
			expectedFirst: 1,
		},
		{
			name: "two independent tasks",
			tasks: []types.Task{
				{ID: "task_1"},
				{ID: "task_2"},
			},
			expectedWaves: 1,
			expectedFirst: 2,
		},
		{
			name: "simple dependency chain",
			tasks: []types.Task{
				{ID: "task_1"},
				{ID: "task_2", BlockedBy: []string{"task_1"}},
			},
			expectedWaves: 2,
			expectedFirst: 1,
		},
		{
			name: "parallel with follow-up",
			tasks: []types.Task{
				{ID: "task_1"},
				{ID: "task_2"},
				{ID: "task_3", BlockedBy: []string{"task_1", "task_2"}},
			},
			expectedWaves: 2,
			expectedFirst: 2,
		},
		{
			name: "external dependencies don't block",
			tasks: []types.Task{
				{ID: "task_1", BlockedBy: []string{"external_task"}},
				{ID: "task_2"},
			},
			expectedWaves: 1,
			expectedFirst: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := AnalyzeDependencies(tt.tasks)

			if len(groups) != tt.expectedWaves {
				t.Errorf("expected %d waves, got %d", tt.expectedWaves, len(groups))
			}

			if tt.expectedWaves > 0 && len(groups[0]) != tt.expectedFirst {
				t.Errorf("expected %d tasks in first wave, got %d", tt.expectedFirst, len(groups[0]))
			}
		})
	}
}

func TestCanParallelize(t *testing.T) {
	tests := []struct {
		name     string
		tasks    []types.Task
		expected bool
	}{
		{
			name:     "empty tasks",
			tasks:    []types.Task{},
			expected: false,
		},
		{
			name: "single task",
			tasks: []types.Task{
				{ID: "task_1"},
			},
			expected: false,
		},
		{
			name: "two independent tasks",
			tasks: []types.Task{
				{ID: "task_1"},
				{ID: "task_2"},
			},
			expected: true,
		},
		{
			name: "simple dependency chain",
			tasks: []types.Task{
				{ID: "task_1"},
				{ID: "task_2", BlockedBy: []string{"task_1"}},
			},
			expected: false,
		},
		{
			name: "parallel with follow-up",
			tasks: []types.Task{
				{ID: "task_1"},
				{ID: "task_2"},
				{ID: "task_3", BlockedBy: []string{"task_1", "task_2"}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanParallelize(tt.tasks)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
