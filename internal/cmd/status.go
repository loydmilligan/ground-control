package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/types"
	"github.com/spf13/cobra"
)

// Status styles
var (
	statusHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("99")).
				MarginTop(1).
				MarginBottom(1)

	statusSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39"))

	statusLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))

	statusValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255"))

	statusRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("40"))

	statusFailedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

	statusCompletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39"))

	statusDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	statusActivityStyle = lipgloss.NewStyle().
				PaddingLeft(2)
)

// NewStatusCmd creates the status command.
func NewStatusCmd(store *data.Store) *cobra.Command {
	var recentCount int

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current session state and recent activity",
		Long: `Display Ground Control status including:
  - Current/recent orchestration sessions
  - Recent activity log events
  - Task counts by state

Examples:
  gc status           # Show current status
  gc status -n 10     # Show last 10 activity events`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(store, recentCount)
		},
	}

	cmd.Flags().IntVarP(&recentCount, "recent", "n", 5, "Number of recent activity events to show")

	return cmd
}

func runStatus(store *data.Store, recentCount int) error {
	now := time.Now()

	// Print header
	fmt.Println(statusHeaderStyle.Render("═══ Ground Control Status ═══"))
	fmt.Printf("%s\n\n", statusDimStyle.Render(now.Format("Monday, January 2, 2006 • 3:04 PM")))

	// Load and display session state
	if err := displaySessionState(store); err != nil {
		fmt.Printf("%s\n\n", statusDimStyle.Render("Could not load sessions: "+err.Error()))
	}

	// Load and display task summary
	if err := displayTaskSummary(store); err != nil {
		fmt.Printf("%s\n\n", statusDimStyle.Render("Could not load tasks: "+err.Error()))
	}

	// Load and display recent activity
	if err := displayRecentActivity(store, recentCount); err != nil {
		fmt.Printf("%s\n\n", statusDimStyle.Render("Could not load activity log: "+err.Error()))
	}

	// Footer
	fmt.Println(statusDimStyle.Render("───────────────────────────────"))
	fmt.Println(statusDimStyle.Render("Use 'gc standup' for daily view • 'gc tasks' for task list • 'gc tui' for interactive mode"))

	return nil
}

func displaySessionState(store *data.Store) error {
	sessions, err := store.LoadSessions()
	if err != nil {
		return err
	}

	fmt.Println(statusSectionStyle.Render("Sessions"))

	if len(sessions) == 0 {
		fmt.Println(statusActivityStyle.Render(statusDimStyle.Render("No orchestration sessions found")))
		fmt.Println()
		return nil
	}

	// Sort sessions by updated time (most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	// Show running sessions first
	var running, recent []types.Session
	for _, s := range sessions {
		if s.Status == types.SessionStatusRunning {
			running = append(running, s)
		} else if len(recent) < 3 {
			recent = append(recent, s)
		}
	}

	if len(running) > 0 {
		for _, s := range running {
			displaySession(s, true)
		}
	} else {
		fmt.Println(statusActivityStyle.Render(statusDimStyle.Render("No active sessions")))
	}

	// Show recent completed/failed sessions
	if len(recent) > 0 {
		fmt.Println()
		fmt.Println(statusActivityStyle.Render(statusDimStyle.Render("Recent:")))
		for _, s := range recent {
			displaySession(s, false)
		}
	}

	fmt.Println()
	return nil
}

func displaySession(s types.Session, isActive bool) {
	// Status icon and style
	var statusIcon string
	var statusStyle lipgloss.Style
	switch s.Status {
	case types.SessionStatusRunning:
		statusIcon = "▶"
		statusStyle = statusRunningStyle
	case types.SessionStatusCompleted:
		statusIcon = "✓"
		statusStyle = statusCompletedStyle
	case types.SessionStatusFailed:
		statusIcon = "✗"
		statusStyle = statusFailedStyle
	case types.SessionStatusCancelled:
		statusIcon = "⊘"
		statusStyle = statusDimStyle
	default:
		statusIcon = "○"
		statusStyle = statusDimStyle
	}

	// Build session line
	taskCount := len(s.TaskIDs)
	taskText := "task"
	if taskCount != 1 {
		taskText = "tasks"
	}

	line := fmt.Sprintf("%s %s  %s  %d %s",
		statusStyle.Render(statusIcon),
		statusStyle.Render(string(s.Status)),
		statusDimStyle.Render(s.ID[:min(12, len(s.ID))]),
		taskCount,
		statusDimStyle.Render(taskText),
	)

	fmt.Println(statusActivityStyle.Render(line))

	// Show current task/stage if active
	if isActive && s.CurrentTaskID != nil {
		currentInfo := fmt.Sprintf("      Current: %s", *s.CurrentTaskID)
		if s.CurrentStage != nil {
			currentInfo += fmt.Sprintf(" (%s)", *s.CurrentStage)
		}
		fmt.Println(statusDimStyle.Render(currentInfo))
	}

	// Show time info
	timeInfo := formatTimeAgo(s.UpdatedAt)
	fmt.Println(statusActivityStyle.Render(statusDimStyle.Render("      " + timeInfo)))
}

func displayTaskSummary(store *data.Store) error {
	tasks, err := store.LoadTasks()
	if err != nil {
		return err
	}

	fmt.Println(statusSectionStyle.Render("Tasks"))

	if len(tasks) == 0 {
		fmt.Println(statusActivityStyle.Render(statusDimStyle.Render("No tasks")))
		fmt.Println()
		return nil
	}

	// Count by state
	counts := make(map[types.TaskState]int)
	for _, t := range tasks {
		counts[t.State]++
	}

	// Build summary
	var parts []string
	if counts[types.TaskStateActive] > 0 {
		parts = append(parts, statusRunningStyle.Render(fmt.Sprintf("%d active", counts[types.TaskStateActive])))
	}
	if counts[types.TaskStateBlocked] > 0 {
		parts = append(parts, statusFailedStyle.Render(fmt.Sprintf("%d blocked", counts[types.TaskStateBlocked])))
	}
	if counts[types.TaskStateWaiting] > 0 {
		parts = append(parts, mediumStyle.Render(fmt.Sprintf("%d waiting", counts[types.TaskStateWaiting])))
	}
	pending := counts[types.TaskStateCreated] + counts[types.TaskStateAssigned]
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", pending))
	}
	if counts[types.TaskStateCompleted] > 0 {
		parts = append(parts, statusDimStyle.Render(fmt.Sprintf("%d completed", counts[types.TaskStateCompleted])))
	}

	summary := joinWithSeparator(parts, " • ")
	fmt.Println(statusActivityStyle.Render(summary))
	fmt.Println()

	return nil
}

func displayRecentActivity(store *data.Store, count int) error {
	events, err := store.LoadActivityLog()
	if err != nil {
		return err
	}

	fmt.Println(statusSectionStyle.Render("Recent Activity"))

	if len(events) == 0 {
		fmt.Println(statusActivityStyle.Render(statusDimStyle.Render("No activity recorded")))
		fmt.Println()
		return nil
	}

	// Sort by timestamp (most recent first)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})

	// Show requested number of events
	displayCount := min(count, len(events))
	for i := 0; i < displayCount; i++ {
		e := events[i]
		displayActivityEvent(e)
	}

	if len(events) > displayCount {
		remaining := len(events) - displayCount
		fmt.Println(statusActivityStyle.Render(statusDimStyle.Render(fmt.Sprintf("      ... and %d more events", remaining))))
	}

	fmt.Println()
	return nil
}

func displayActivityEvent(e types.ActivityEvent) {
	// Type icon
	var icon string
	switch e.Type {
	case "task_created":
		icon = "+"
	case "task_assigned":
		icon = "→"
	case "task_completed":
		icon = "✓"
	case "decision_made":
		icon = "◆"
	case "project_created":
		icon = "□"
	case "ritual_run":
		icon = "◎"
	default:
		icon = "·"
	}

	// Truncate summary if too long
	summary := e.Summary
	if len(summary) > 50 {
		summary = summary[:47] + "..."
	}

	timeAgo := formatTimeAgo(e.Timestamp)

	line := fmt.Sprintf("%s  %s  %s",
		statusDimStyle.Render(icon),
		summary,
		statusDimStyle.Render(timeAgo),
	)

	fmt.Println(statusActivityStyle.Render(line))
}

func formatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2")
	}
}
