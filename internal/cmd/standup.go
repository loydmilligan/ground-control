package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/types"
	"github.com/spf13/cobra"
)

// Standup styles
var (
	standupHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("99")).
				MarginTop(1).
				MarginBottom(1)

	standupSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39"))

	standupTaskStyle = lipgloss.NewStyle().
				PaddingLeft(2)

	standupEmptyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Italic(true).
				PaddingLeft(2)

	standupHighStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	standupMediumStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	standupLowStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))

	standupActiveIcon    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("▶")
	standupBlockedIcon   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("⊘")
	standupWaitingIcon   = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("◷")
	standupCreatedIcon   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("○")
	standupCompletedIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Render("✓")
)

// NewStandupCmd creates the standup command.
func NewStandupCmd(store *data.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "standup",
		Short: "Run daily standup ritual",
		Long: `Display a summary of current task state for daily standup.

Shows:
  - Active tasks (currently being worked on)
  - Blocked tasks (waiting on dependencies)
  - Tasks waiting for human input
  - Pending tasks ready to start
  - Recently completed tasks`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStandup(store)
		},
	}

	return cmd
}

func runStandup(store *data.Store) error {
	tasks, err := store.LoadTasks()
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	// Categorize tasks
	var active, blocked, waiting, pending, completed []types.Task
	var recentlyCompleted []types.Task

	now := time.Now()
	oneDayAgo := now.Add(-24 * time.Hour)

	for _, t := range tasks {
		switch t.State {
		case types.TaskStateActive:
			active = append(active, t)
		case types.TaskStateBlocked:
			blocked = append(blocked, t)
		case types.TaskStateWaiting:
			waiting = append(waiting, t)
		case types.TaskStateCreated, types.TaskStateAssigned:
			pending = append(pending, t)
		case types.TaskStateCompleted:
			completed = append(completed, t)
			if t.CompletedAt != nil && t.CompletedAt.After(oneDayAgo) {
				recentlyCompleted = append(recentlyCompleted, t)
			}
		}
	}

	// Print header
	fmt.Println(standupHeaderStyle.Render("═══ Ground Control Standup ═══"))
	fmt.Printf("%s\n\n", lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(now.Format("Monday, January 2, 2006 • 3:04 PM")))

	// Summary line
	summaryParts := []string{}
	if len(active) > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d active", len(active)))
	}
	if len(blocked) > 0 {
		summaryParts = append(summaryParts, standupHighStyle.Render(fmt.Sprintf("%d blocked", len(blocked))))
	}
	if len(waiting) > 0 {
		summaryParts = append(summaryParts, standupMediumStyle.Render(fmt.Sprintf("%d waiting", len(waiting))))
	}
	if len(pending) > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d pending", len(pending)))
	}
	fmt.Printf("Tasks: %s\n\n", joinWithSeparator(summaryParts, " • "))

	// Active tasks
	fmt.Println(standupSectionStyle.Render("▶ In Progress"))
	if len(active) == 0 {
		fmt.Println(standupEmptyStyle.Render("No active tasks"))
	} else {
		for _, t := range active {
			printStandupTask(t, standupActiveIcon)
		}
	}
	fmt.Println()

	// Blocked tasks
	if len(blocked) > 0 {
		fmt.Println(standupSectionStyle.Render("⊘ Blocked"))
		for _, t := range blocked {
			printStandupTask(t, standupBlockedIcon)
			if len(t.BlockedBy) > 0 {
				fmt.Printf("      %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("blocked by: "+joinWithSeparator(t.BlockedBy, ", ")))
			}
		}
		fmt.Println()
	}

	// Waiting for human
	if len(waiting) > 0 {
		fmt.Println(standupSectionStyle.Render("◷ Waiting for Human"))
		for _, t := range waiting {
			printStandupTask(t, standupWaitingIcon)
		}
		fmt.Println()
	}

	// Pending (ready to start)
	fmt.Println(standupSectionStyle.Render("○ Ready to Start"))
	if len(pending) == 0 {
		fmt.Println(standupEmptyStyle.Render("No pending tasks"))
	} else {
		// Sort by importance (high first)
		highPending := filterByImportance(pending, types.ImportanceHigh)
		mediumPending := filterByImportance(pending, types.ImportanceMedium)
		lowPending := filterByImportance(pending, types.ImportanceLow)

		for _, t := range highPending {
			printStandupTask(t, standupCreatedIcon)
		}
		for _, t := range mediumPending {
			printStandupTask(t, standupCreatedIcon)
		}
		for _, t := range lowPending {
			printStandupTask(t, standupCreatedIcon)
		}
	}
	fmt.Println()

	// Recently completed
	if len(recentlyCompleted) > 0 {
		fmt.Println(standupSectionStyle.Render("✓ Completed (last 24h)"))
		for _, t := range recentlyCompleted {
			printStandupTask(t, standupCompletedIcon)
		}
		fmt.Println()
	}

	// Update last_run in rituals.json
	if err := updateRitualLastRun(store, "daily-standup"); err != nil {
		// Non-fatal, just warn
		fmt.Printf("%s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("⚠ Could not update ritual last_run: "+err.Error()))
	}

	// Footer
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("───────────────────────────────"))
	fmt.Printf("%s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Use 'gc tui' for interactive view • 'gc complete <id>' to finish tasks"))

	return nil
}

func printStandupTask(t types.Task, icon string) {
	impStyle := standupLowStyle
	switch t.Importance {
	case types.ImportanceHigh:
		impStyle = standupHighStyle
	case types.ImportanceMedium:
		impStyle = standupMediumStyle
	}

	// Complexity dots
	dots := ""
	for i := 0; i < t.Complexity; i++ {
		dots += "●"
	}
	for i := t.Complexity; i < 5; i++ {
		dots += "○"
	}

	line := fmt.Sprintf("%s %s  %s  %s",
		icon,
		t.Title,
		lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(dots),
		impStyle.Render(string(t.Importance)),
	)
	fmt.Println(standupTaskStyle.Render(line))
}

func filterByImportance(tasks []types.Task, importance types.Importance) []types.Task {
	var result []types.Task
	for _, t := range tasks {
		if t.Importance == importance {
			result = append(result, t)
		}
	}
	return result
}

func joinWithSeparator(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	result := items[0]
	for i := 1; i < len(items); i++ {
		result += sep + items[i]
	}
	return result
}

func updateRitualLastRun(store *data.Store, ritualID string) error {
	path := filepath.Join(store.GetDataDir(), "rituals.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var file struct {
		Rituals []map[string]interface{} `json:"rituals"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	for i, r := range file.Rituals {
		if r["id"] == ritualID {
			file.Rituals[i]["last_run"] = now
			break
		}
	}

	output, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, output, 0644)
}
