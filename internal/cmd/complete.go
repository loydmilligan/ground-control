package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/types"
	"github.com/mmariani/ground-control/internal/verify"
	"github.com/spf13/cobra"
)

var (
	completeTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	completeInfoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// NewCompleteCmd creates the complete command.
func NewCompleteCmd(store *data.Store) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "complete <task_id>",
		Short: "Complete a task with verification",
		Long: `Mark a task as completed after running its verification checks.

Verification types:
  - test_pass:      Runs a command, must exit 0
  - file_exists:    Checks that output files exist
  - human_approval: Prompts for manual approval
  - none:           No verification required

Use --force to skip verification (not recommended).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// Load tasks
			tasks, err := store.LoadTasks()
			if err != nil {
				return fmt.Errorf("loading tasks: %w", err)
			}

			// Find the task
			var task *types.Task
			var taskIdx int
			for i := range tasks {
				if tasks[i].ID == taskID {
					task = &tasks[i]
					taskIdx = i
					break
				}
			}

			if task == nil {
				return fmt.Errorf("task not found: %s", taskID)
			}

			// Show task info
			fmt.Println()
			fmt.Println(completeTitleStyle.Render("Completing task:"))
			fmt.Printf("  %s\n", task.Title)
			fmt.Printf("  %s\n", completeInfoStyle.Render(fmt.Sprintf("ID: %s  State: %s  Verification: %s",
				task.ID, task.State, task.Verification.Type)))
			fmt.Println()

			// Check if already completed
			if task.State == types.TaskStateCompleted {
				fmt.Println(completeInfoStyle.Render("Task is already completed."))
				return nil
			}

			// Run verification (unless forced)
			if force {
				fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("⚠ Skipping verification (--force)"))
			} else {
				result := verify.Verify(task)
				verify.PrintResult(result)

				if !result.Passed {
					return fmt.Errorf("verification failed — task not completed")
				}
			}

			// Update output exists flags
			verify.CheckOutputs(task)

			// Mark as completed
			now := time.Now()
			tasks[taskIdx].State = types.TaskStateCompleted
			tasks[taskIdx].CompletedAt = &now
			tasks[taskIdx].UpdatedAt = now

			// Save
			if err := store.SaveTasks(tasks); err != nil {
				return fmt.Errorf("saving tasks: %w", err)
			}

			fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Bold(true).Render("✓ Task completed!"))

			// Show next steps if any
			if len(task.SuggestedNext) > 0 {
				fmt.Println()
				fmt.Println(completeTitleStyle.Render("Suggested next steps:"))
				for _, step := range task.SuggestedNext {
					fmt.Printf("  • %s\n", step)
				}
			}

			// Note about after_completion
			if task.AfterCompletion == types.AfterCompletionTaskmasterReview {
				fmt.Println()
				fmt.Println(completeInfoStyle.Render("→ This task triggers Taskmaster review"))
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip verification (not recommended)")

	return cmd
}
