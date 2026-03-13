// Package cmd implements CLI commands for Ground Control.
package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/types"
	"github.com/spf13/cobra"
)

// Sprint command styles
var (
	sprintHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	sprintNameStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	sprintGoalStyle   = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("243"))
	sprintActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))
	sprintPausedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	sprintDoneStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

// NewSprintCmd creates the sprint command with subcommands.
func NewSprintCmd(store *data.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sprint",
		Short: "Manage sprints (task groupings)",
		Long: `Sprints are lightweight groupings of related tasks toward a goal.

Examples:
  gc sprint list                      # List all sprints
  gc sprint create "Pipeline v2"      # Create a new sprint
  gc sprint add <sprint> <task#>      # Add task to sprint
  gc sprint status <sprint>           # Show sprint status
  gc sprint complete <sprint>         # Mark sprint complete`,
	}

	cmd.AddCommand(newSprintListCmd(store))
	cmd.AddCommand(newSprintCreateCmd(store))
	cmd.AddCommand(newSprintAddCmd(store))
	cmd.AddCommand(newSprintStatusCmd(store))
	cmd.AddCommand(newSprintCompleteCmd(store))

	return cmd
}

func newSprintListCmd(store *data.Store) *cobra.Command {
	var showAll bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sprints",
		RunE: func(cmd *cobra.Command, args []string) error {
			sprints, err := store.LoadSprints()
			if err != nil {
				return err
			}

			if len(sprints) == 0 {
				fmt.Println("No sprints yet. Create one with: gc sprint create \"Sprint Name\"")
				return nil
			}

			fmt.Println(sprintHeaderStyle.Render("Sprints"))
			fmt.Println()

			for _, s := range sprints {
				if !showAll && s.Status == types.SprintStatusCompleted {
					continue
				}

				statusIcon := "▶"
				statusStyle := sprintActiveStyle
				switch s.Status {
				case types.SprintStatusPaused:
					statusIcon = "⏸"
					statusStyle = sprintPausedStyle
				case types.SprintStatusCompleted:
					statusIcon = "✓"
					statusStyle = sprintDoneStyle
				}

				fmt.Printf("  %s %s %s\n",
					statusStyle.Render(statusIcon),
					sprintNameStyle.Render(s.Name),
					dimStyle.Render(fmt.Sprintf("(%d tasks)", len(s.TaskIDs))))

				if len(s.ProjectIDs) > 0 {
					fmt.Printf("    %s\n", dimStyle.Render("Projects: "+strings.Join(s.ProjectIDs, ", ")))
				}
				if s.Goal != "" {
					fmt.Printf("    %s\n", sprintGoalStyle.Render(s.Goal))
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "Include completed sprints")

	return cmd
}

func newSprintCreateCmd(store *data.Store) *cobra.Command {
	var description, goal string
	var projectIDs []string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new sprint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			sprint, err := store.CreateSprint(name, description, goal, projectIDs)
			if err != nil {
				return err
			}

			fmt.Printf("✓ Sprint created: %s\n", sprintNameStyle.Render(sprint.Name))
			fmt.Printf("  ID: %s\n", sprint.ID)
			if goal != "" {
				fmt.Printf("  Goal: %s\n", goal)
			}
			if len(projectIDs) > 0 {
				fmt.Printf("  Projects: %s\n", strings.Join(projectIDs, ", "))
			}
			fmt.Println()
			fmt.Println("Add tasks with: gc sprint add", sprint.Name, "<task#>")

			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Sprint description")
	cmd.Flags().StringVarP(&goal, "goal", "g", "", "Sprint goal")
	cmd.Flags().StringArrayVarP(&projectIDs, "project", "p", nil, "Associated project ID(s) (can be specified multiple times)")

	return cmd
}

func newSprintAddCmd(store *data.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "add <sprint> <task#...>",
		Short: "Add tasks to a sprint",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintRef := args[0]
			taskRefs := args[1:]

			// Find sprint by name or ID
			sprint, err := store.GetSprintByName(sprintRef)
			if err != nil {
				sprint, err = store.GetSprintByID(sprintRef)
				if err != nil {
					return fmt.Errorf("sprint not found: %s", sprintRef)
				}
			}

			// Load tasks to resolve indexes
			tasks, err := store.LoadTasks()
			if err != nil {
				return err
			}

			// Build index map (only non-completed tasks)
			indexMap := make(map[int]types.Task)
			idx := 1
			for _, t := range tasks {
				if t.State != types.TaskStateCompleted {
					indexMap[idx] = t
					idx++
				}
			}

			// Add each task
			added := 0
			for _, ref := range taskRefs {
				var taskID string

				// Try as index first
				if index, err := strconv.Atoi(ref); err == nil {
					if task, ok := indexMap[index]; ok {
						taskID = task.ID
					} else {
						fmt.Printf("⚠ Invalid task index: %d\n", index)
						continue
					}
				} else if strings.HasPrefix(ref, "task_") {
					taskID = ref
				} else {
					fmt.Printf("⚠ Invalid task reference: %s\n", ref)
					continue
				}

				if err := store.AddTaskToSprint(sprint.ID, taskID); err != nil {
					if strings.Contains(err.Error(), "already in sprint") {
						fmt.Printf("⚠ Task %s already in sprint\n", ref)
					} else {
						fmt.Printf("⚠ Error adding task %s: %v\n", ref, err)
					}
					continue
				}

				added++
				fmt.Printf("✓ Added task %s to sprint\n", ref)
			}

			if added > 0 {
				fmt.Printf("\n%d task(s) added to %s\n", added, sprintNameStyle.Render(sprint.Name))
			}

			return nil
		},
	}
}

func newSprintStatusCmd(store *data.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "status <sprint>",
		Short: "Show sprint status with task details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintRef := args[0]

			// Find sprint
			sprint, err := store.GetSprintByName(sprintRef)
			if err != nil {
				sprint, err = store.GetSprintByID(sprintRef)
				if err != nil {
					return fmt.Errorf("sprint not found: %s", sprintRef)
				}
			}

			// Load tasks
			tasks, err := store.LoadTasks()
			if err != nil {
				return err
			}

			// Build task map
			taskMap := make(map[string]types.Task)
			for _, t := range tasks {
				taskMap[t.ID] = t
			}

			// Print header
			statusIcon := "▶"
			statusStyle := sprintActiveStyle
			switch sprint.Status {
			case types.SprintStatusPaused:
				statusIcon = "⏸"
				statusStyle = sprintPausedStyle
			case types.SprintStatusCompleted:
				statusIcon = "✓"
				statusStyle = sprintDoneStyle
			}

			fmt.Println(sprintHeaderStyle.Render("Sprint: " + sprint.Name))
			fmt.Println()
			fmt.Printf("Status: %s %s\n", statusIcon, statusStyle.Render(string(sprint.Status)))
			if sprint.Goal != "" {
				fmt.Printf("Goal: %s\n", sprint.Goal)
			}
			if sprint.Description != "" {
				fmt.Printf("Description: %s\n", sprint.Description)
			}
			if len(sprint.ProjectIDs) > 0 {
				fmt.Printf("Projects: %s\n", strings.Join(sprint.ProjectIDs, ", "))
			}
			fmt.Println()

			if len(sprint.TaskIDs) == 0 {
				fmt.Println(dimStyle.Render("No tasks in sprint yet."))
				return nil
			}

			// Count stats
			completed := 0
			active := 0
			blocked := 0

			fmt.Println(dimStyle.Render("Tasks:"))
			for _, taskID := range sprint.TaskIDs {
				task, ok := taskMap[taskID]
				if !ok {
					fmt.Printf("  ⚠ %s (not found)\n", taskID)
					continue
				}

				stateIcon := "○"
				stateStyle := dimStyle
				switch task.State {
				case types.TaskStateCompleted:
					stateIcon = "✓"
					stateStyle = lowStyle
					completed++
				case types.TaskStateActive:
					stateIcon = "▶"
					stateStyle = lowStyle
					active++
				case types.TaskStateBlocked:
					stateIcon = "⊘"
					stateStyle = highStyle
					blocked++
				}

				title := task.Title
				if len(title) > 50 {
					title = title[:47] + "..."
				}

				fmt.Printf("  %s %s\n", stateStyle.Render(stateIcon), title)
			}

			fmt.Println()
			total := len(sprint.TaskIDs)
			pending := total - completed - active - blocked
			fmt.Printf("Progress: %d/%d completed", completed, total)
			if active > 0 {
				fmt.Printf(", %d active", active)
			}
			if blocked > 0 {
				fmt.Printf(", %d blocked", blocked)
			}
			if pending > 0 {
				fmt.Printf(", %d pending", pending)
			}
			fmt.Println()

			return nil
		},
	}
}

func newSprintCompleteCmd(store *data.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "complete <sprint>",
		Short: "Mark sprint as complete",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintRef := args[0]

			// Find sprint
			sprint, err := store.GetSprintByName(sprintRef)
			if err != nil {
				sprint, err = store.GetSprintByID(sprintRef)
				if err != nil {
					return fmt.Errorf("sprint not found: %s", sprintRef)
				}
			}

			if err := store.UpdateSprintStatus(sprint.ID, types.SprintStatusCompleted); err != nil {
				return err
			}

			fmt.Printf("✓ Sprint %s marked complete\n", sprintNameStyle.Render(sprint.Name))
			return nil
		},
	}
}
