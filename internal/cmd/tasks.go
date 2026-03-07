// Package cmd implements CLI commands for Ground Control.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/types"
	"github.com/spf13/cobra"
)

// Styles for task output
var (
	taskTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	taskStateStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	taskIndexStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	highStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	mediumStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	lowStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))
	headerStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	dimStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// IndexedTask holds a task with its display index.
type IndexedTask struct {
	Index int
	Task  types.Task
}

// NewTasksCmd creates the tasks command.
func NewTasksCmd(store *data.Store) *cobra.Command {
	var showAll bool
	var stateFilter string
	var tagFilter string
	var projectFilter string
	var jsonOutput bool
	var detailIndex int

	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "List tasks with filtering",
		Long: `Display tasks in Ground Control with dynamic indexes for use with gc orc.

By default, completed tasks are hidden. Use --all to show them.

Examples:
  gc tasks                    # Active tasks with indexes
  gc tasks --detail 1         # Show full details for task #1
  gc tasks --all              # Include completed tasks
  gc tasks --state=blocked    # Filter by state
  gc tasks --tag=cli          # Filter by tag
  gc tasks --json             # JSON output for scripting`,
		RunE: func(cmd *cobra.Command, args []string) error {
			tasks, err := store.LoadTasks()
			if err != nil {
				return fmt.Errorf("loading tasks: %w", err)
			}

			// Filter tasks
			filtered := filterTasks(tasks, showAll, stateFilter, tagFilter, projectFilter)

			// Show detail for specific task
			if detailIndex > 0 {
				return outputTaskDetail(filtered, detailIndex)
			}

			if jsonOutput {
				return outputJSON(filtered)
			}

			return outputFormatted(tasks, filtered, showAll, stateFilter, tagFilter, projectFilter)
		},
	}

	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "Include completed tasks")
	cmd.Flags().StringVarP(&stateFilter, "state", "s", "", "Filter by state (created, assigned, blocked, active, waiting, completed)")
	cmd.Flags().StringVarP(&tagFilter, "tag", "t", "", "Filter by tag")
	cmd.Flags().StringVarP(&projectFilter, "project", "p", "", "Filter by project ID")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().IntVarP(&detailIndex, "detail", "d", 0, "Show full details for task at index #")

	return cmd
}

// filterTasks applies filters to the task list.
func filterTasks(tasks []types.Task, showAll bool, state, tag, project string) []IndexedTask {
	var result []IndexedTask
	index := 1

	for _, t := range tasks {
		// Hide completed by default
		if !showAll && t.State == types.TaskStateCompleted {
			continue
		}

		// State filter
		if state != "" && string(t.State) != state {
			continue
		}

		// Tag filter
		if tag != "" && !containsTag(t.Tags, tag) {
			continue
		}

		// Project filter
		if project != "" {
			if t.ProjectID == nil || *t.ProjectID != project {
				continue
			}
		}

		result = append(result, IndexedTask{Index: index, Task: t})
		index++
	}

	return result
}

func containsTag(tags []string, search string) bool {
	search = strings.ToLower(search)
	for _, tag := range tags {
		if strings.ToLower(tag) == search {
			return true
		}
	}
	return false
}

func outputJSON(indexed []IndexedTask) error {
	type jsonTask struct {
		Index int        `json:"index"`
		Task  types.Task `json:"task"`
	}

	output := make([]jsonTask, len(indexed))
	for i, it := range indexed {
		output[i] = jsonTask{Index: it.Index, Task: it.Task}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func outputFormatted(allTasks []types.Task, filtered []IndexedTask, showAll bool, state, tag, project string) error {
	// Count tasks by state for summary
	counts := countByState(allTasks)

	// Build summary line
	var summaryParts []string
	if counts["active"] > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d active", counts["active"]))
	}
	if counts["blocked"] > 0 {
		summaryParts = append(summaryParts, highStyle.Render(fmt.Sprintf("%d blocked", counts["blocked"])))
	}
	if counts["waiting"] > 0 {
		summaryParts = append(summaryParts, mediumStyle.Render(fmt.Sprintf("%d waiting", counts["waiting"])))
	}
	pendingCount := counts["created"] + counts["assigned"]
	if pendingCount > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d pending", pendingCount))
	}
	if showAll && counts["completed"] > 0 {
		summaryParts = append(summaryParts, lowStyle.Render(fmt.Sprintf("%d completed", counts["completed"])))
	}

	// Print header
	fmt.Println(headerStyle.Render("Tasks") + " " + dimStyle.Render("("+strings.Join(summaryParts, ", ")+")"))
	fmt.Println()

	// Show active filters
	var filters []string
	if state != "" {
		filters = append(filters, "state="+state)
	}
	if tag != "" {
		filters = append(filters, "tag="+tag)
	}
	if project != "" {
		filters = append(filters, "project="+project)
	}
	if len(filters) > 0 {
		fmt.Println(dimStyle.Render("Filters: " + strings.Join(filters, ", ")))
		fmt.Println()
	}

	if len(filtered) == 0 {
		fmt.Println(dimStyle.Render("No tasks match the current filters."))
		if !showAll {
			fmt.Println(dimStyle.Render("Use --all to include completed tasks."))
		}
		return nil
	}

	// Print header row
	fmt.Printf("  %s  %-10s %-6s %s\n",
		dimStyle.Render("#"),
		dimStyle.Render("State"),
		dimStyle.Render("Imp"),
		dimStyle.Render("Title"))
	fmt.Println(dimStyle.Render("  " + strings.Repeat("─", 60)))

	// Print tasks
	for _, it := range filtered {
		t := it.Task

		// Format importance
		var impStyle lipgloss.Style
		var impShort string
		switch t.Importance {
		case types.ImportanceHigh:
			impStyle = highStyle
			impShort = "high"
		case types.ImportanceMedium:
			impStyle = mediumStyle
			impShort = "med"
		default:
			impStyle = lowStyle
			impShort = "low"
		}

		// State styling
		stateStr := string(t.State)
		stateStyled := stateStr
		switch t.State {
		case types.TaskStateBlocked:
			stateStyled = highStyle.Render(stateStr)
		case types.TaskStateWaiting:
			stateStyled = mediumStyle.Render(stateStr)
		case types.TaskStateActive:
			stateStyled = lowStyle.Render(stateStr)
		case types.TaskStateCompleted:
			stateStyled = dimStyle.Render(stateStr)
		default:
			stateStyled = dimStyle.Render(stateStr)
		}

		// Truncate title if too long
		title := t.Title
		if len(title) > 45 {
			title = title[:42] + "..."
		}

		fmt.Printf("  %s  %-10s %-6s %s\n",
			taskIndexStyle.Render(fmt.Sprintf("%2d", it.Index)),
			stateStyled,
			impStyle.Render(impShort),
			title)
	}

	fmt.Println()
	fmt.Println(dimStyle.Render("Use 'gc orc 1 3' to orchestrate tasks or 'gc tui' for interactive mode."))

	return nil
}

func countByState(tasks []types.Task) map[string]int {
	counts := make(map[string]int)
	for _, t := range tasks {
		counts[string(t.State)]++
	}
	return counts
}

func outputTaskDetail(filtered []IndexedTask, index int) error {
	// Find task by index
	var task *types.Task
	for _, it := range filtered {
		if it.Index == index {
			task = &it.Task
			break
		}
	}

	if task == nil {
		return fmt.Errorf("no task found at index %d", index)
	}

	// Header
	fmt.Println(headerStyle.Render(fmt.Sprintf("Task #%d", index)))
	fmt.Println()

	// Title and basic info
	fmt.Println(taskTitleStyle.Render(task.Title))
	fmt.Println()

	// Status row
	var impStyle lipgloss.Style
	switch task.Importance {
	case types.ImportanceHigh:
		impStyle = highStyle
	case types.ImportanceMedium:
		impStyle = mediumStyle
	default:
		impStyle = lowStyle
	}
	fmt.Printf("%s: %s  %s: %s  %s: %s  %s: %d\n",
		dimStyle.Render("State"), string(task.State),
		dimStyle.Render("Importance"), impStyle.Render(string(task.Importance)),
		dimStyle.Render("Type"), task.Type,
		dimStyle.Render("Complexity"), task.Complexity)
	fmt.Println()

	// Description
	if task.Description != "" {
		fmt.Println(dimStyle.Render("Description:"))
		fmt.Println(task.Description)
		fmt.Println()
	}

	// Context
	if task.Context.Background != "" {
		fmt.Println(dimStyle.Render("Background:"))
		fmt.Println(task.Context.Background)
		fmt.Println()
	}

	if len(task.Context.Requirements) > 0 {
		fmt.Println(dimStyle.Render("Requirements:"))
		for _, req := range task.Context.Requirements {
			fmt.Printf("  • %s\n", req)
		}
		fmt.Println()
	}

	if len(task.Context.Constraints) > 0 {
		fmt.Println(dimStyle.Render("Constraints:"))
		for _, c := range task.Context.Constraints {
			fmt.Printf("  • %s\n", c)
		}
		fmt.Println()
	}

	// Outputs
	if len(task.Outputs) > 0 {
		fmt.Println(dimStyle.Render("Expected Outputs:"))
		for _, out := range task.Outputs {
			status := "○"
			if out.Exists {
				status = lowStyle.Render("✓")
			}
			fmt.Printf("  %s %s - %s\n", status, out.Path, out.Description)
		}
		fmt.Println()
	}

	// Verification
	if task.Verification.Type != "" {
		fmt.Println(dimStyle.Render("Verification:"))
		fmt.Printf("  Type: %s\n", task.Verification.Type)
		if task.Verification.Command != nil && *task.Verification.Command != "" {
			fmt.Printf("  Command: %s\n", *task.Verification.Command)
		}
		if len(task.Verification.Paths) > 0 {
			fmt.Printf("  Paths: %s\n", strings.Join(task.Verification.Paths, ", "))
		}
		fmt.Println()
	}

	// Tags
	if len(task.Tags) > 0 {
		fmt.Printf("%s: %s\n", dimStyle.Render("Tags"), strings.Join(task.Tags, ", "))
	}

	// ID for reference
	fmt.Printf("%s: %s\n", dimStyle.Render("ID"), task.ID)

	return nil
}

// NewDumpCmd creates the dump command.
func NewDumpCmd(store *data.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump [content]",
		Short: "Add a brain dump entry",
		Long:  "Capture a quick idea, bug, or note for later processing.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content := args[0]

			entry, err := store.AddBrainDump(content)
			if err != nil {
				return fmt.Errorf("adding brain dump: %w", err)
			}

			fmt.Printf("✓ Brain dump captured: %s\n", entry.ID)
			return nil
		},
	}

	return cmd
}

// GetDataDir finds the data directory, checking current dir and parent dirs.
func GetDataDir() string {
	// Check current directory
	if _, err := os.Stat("data/tasks.json"); err == nil {
		return "data"
	}

	// Check if we're in a subdirectory
	if _, err := os.Stat("../data/tasks.json"); err == nil {
		return "../data"
	}

	// Default to current directory's data folder
	return "data"
}
