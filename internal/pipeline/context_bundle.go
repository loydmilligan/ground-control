// Package pipeline implements the task execution pipeline stages.
package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/types"
)

// ContextBundleStage builds context bundles for tasks that don't have them.
type ContextBundleStage struct {
	store   *data.Store
	verbose bool
}

// NewContextBundleStage creates a new context bundle stage.
func NewContextBundleStage(store *data.Store, verbose bool) *ContextBundleStage {
	return &ContextBundleStage{store: store, verbose: verbose}
}

// Name returns the stage name.
func (s *ContextBundleStage) Name() string {
	return "ContextBundle"
}

// CanSkip returns true if the task already has a context bundle.
func (s *ContextBundleStage) CanSkip(ctx *StageContext) bool {
	return ctx.Task.ContextBundle != nil
}

// Execute executes the context bundle stage.
func (s *ContextBundleStage) Execute(ctx *StageContext) *StageResult {
	task := ctx.Task

	// Check if task already has a context bundle
	if task.ContextBundle != nil {
		if s.verbose {
			fmt.Println("  ✓ Context bundle already exists")
		}
		return &StageResult{
			Status: StageStatusSuccess,
			Notes:  "Context bundle already exists",
		}
	}

	if s.verbose {
		fmt.Println("  → Building context bundle...")
	}

	// Create bundle directory
	bundlePath := filepath.Join(s.store.GetDataDir(), "context", task.ID)
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("failed to create bundle directory: %w", err),
		}
	}

	// Determine working directory for the task
	workingDir := s.getWorkingDirectory(*task)

	if s.verbose {
		fmt.Printf("    Building context bundle in: %s\n", bundlePath)
		fmt.Printf("    Working directory: %s\n", workingDir)
	}

	// For now, create minimal context bundle files directly
	// TODO: Integrate with Claude CLI context engineer agent when ready
	if s.verbose {
		fmt.Printf("    Creating minimal context bundle...\n")
	}

	// Create minimal bundle files
	requiredFiles := []string{"requirements.md", "project_context.md"}
	for _, f := range requiredFiles {
		path := filepath.Join(bundlePath, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// Create minimal file if missing
			if err := s.createMinimalFile(path, f, *task); err != nil {
				return &StageResult{
					Status: StageStatusFailed,
					Error:  fmt.Errorf("failed to create %s: %w", f, err),
				}
			}
		}
	}

	// Update task with context bundle metadata
	bundle := &types.ContextBundle{
		BuiltAt:    time.Now(),
		BuiltBy:    "context-engineer",
		BundlePath: bundlePath,
		Files: types.ContextBundleFiles{
			Requirements:   filepath.Join(bundlePath, "requirements.md"),
			ProjectContext: filepath.Join(bundlePath, "project_context.md"),
			Patterns:       filepath.Join(bundlePath, "patterns.md"),
			Decisions:      filepath.Join(bundlePath, "decisions.md"),
			TestHints:      filepath.Join(bundlePath, "test_hints.md"),
		},
		Notes: "Built by context-engineer agent",
	}

	// Check for relevant_code files
	relevantCodePath := filepath.Join(bundlePath, "relevant_code.md")
	if _, err := os.Stat(relevantCodePath); err == nil {
		bundle.Files.RelevantCode = []string{relevantCodePath}
	}

	// Update task in store
	if err := s.updateTaskBundle(task.ID, bundle); err != nil {
		return &StageResult{
			Status: StageStatusFailed,
			Error:  fmt.Errorf("failed to update task with bundle: %w", err),
		}
	}

	// Update the task in context with the new bundle
	ctx.Task.ContextBundle = bundle

	return &StageResult{
		Status:      StageStatusSuccess,
		Notes:       "Context bundle created",
		OutputFiles: []string{bundle.BundlePath},
	}
}

// getWorkingDirectory determines where the task should be executed.
func (s *ContextBundleStage) getWorkingDirectory(task types.Task) string {
	// Check for explicit working directory in task context
	if task.Context.WorkingDirectory != nil && *task.Context.WorkingDirectory != "" {
		return *task.Context.WorkingDirectory
	}

	// Check for project-based working directory
	if task.ProjectID != nil {
		// TODO: Look up project and get its repo_path
	}

	// Check tags for project hints
	for _, tag := range task.Tags {
		if tag == "notifai" {
			return "/home/mmariani/Projects/android_notify_helper/notifai"
		}
		// Add more project mappings as needed
	}

	// Default to current directory
	cwd, _ := os.Getwd()
	return cwd
}

// buildContextEngineerPrompt creates the prompt for the context engineer.
func (s *ContextBundleStage) buildContextEngineerPrompt(task types.Task, bundlePath, workingDir string) string {
	prompt := fmt.Sprintf(`Build a context bundle for the following task.

## Task Details

**ID:** %s
**Title:** %s
**Type:** %s
**Description:** %s

## Task Context

**Background:** %s

**Requirements:**
`, task.ID, task.Title, task.Type, task.Description, task.Context.Background)

	for _, req := range task.Context.Requirements {
		prompt += fmt.Sprintf("- %s\n", req)
	}

	prompt += "\n**Constraints:**\n"
	for _, c := range task.Context.Constraints {
		prompt += fmt.Sprintf("- %s\n", c)
	}

	prompt += fmt.Sprintf(`
## Working Directory

The task should be executed in: %s

Explore this directory to understand the project structure.

## Bundle Output Directory

Create the context bundle files in: %s

Create these files:
1. requirements.md - Detailed task requirements
2. project_context.md - Project overview, architecture, tech stack
3. relevant_code.md - Code snippets with file paths and line numbers
4. patterns.md - Coding patterns to follow
5. decisions.md - Related design decisions
6. test_hints.md - How to test/verify the work

After creating the files, output a JSON summary of what was created.

Remember: The goal is to provide enough context that the implementer agent can complete the task without searching or asking questions, but not so much that it's overwhelming.
`, workingDir, bundlePath)

	return prompt
}

// loadAgentPrompt loads an agent's system prompt from the agents directory.
func (s *ContextBundleStage) loadAgentPrompt(agentName string) (string, error) {
	// Try multiple locations
	locations := []string{
		filepath.Join(s.store.GetDataDir(), "..", "agents", agentName+".md"),
		filepath.Join("agents", agentName+".md"),
	}

	for _, path := range locations {
		content, err := os.ReadFile(path)
		if err == nil {
			return string(content), nil
		}
	}

	return "", fmt.Errorf("agent prompt not found: %s", agentName)
}

// createMinimalFile creates a minimal bundle file from task metadata.
func (s *ContextBundleStage) createMinimalFile(path, filename string, task types.Task) error {
	var content string

	switch filename {
	case "requirements.md":
		content = fmt.Sprintf("# Requirements: %s\n\n## Description\n\n%s\n\n## Requirements\n\n", task.Title, task.Description)
		for _, req := range task.Context.Requirements {
			content += fmt.Sprintf("- %s\n", req)
		}
		if len(task.Context.Constraints) > 0 {
			content += "\n## Constraints\n\n"
			for _, c := range task.Context.Constraints {
				content += fmt.Sprintf("- %s\n", c)
			}
		}

	case "project_context.md":
		content = fmt.Sprintf("# Project Context\n\n## Background\n\n%s\n", task.Context.Background)

	default:
		content = fmt.Sprintf("# %s\n\nNo specific content for this task.\n", filename)
	}

	return os.WriteFile(path, []byte(content), 0644)
}

// updateTaskBundle updates the task's context bundle in the store.
func (s *ContextBundleStage) updateTaskBundle(taskID string, bundle *types.ContextBundle) error {
	tasks, err := s.store.LoadTasks()
	if err != nil {
		return err
	}

	for i, t := range tasks {
		if t.ID == taskID {
			tasks[i].ContextBundle = bundle
			break
		}
	}

	return s.store.SaveTasks(tasks)
}
