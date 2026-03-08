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

var (
	historyHeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	historySuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))
	historyFailedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	historyDimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// NewHistoryCmd creates the history command for viewing past orchestration sessions.
func NewHistoryCmd(store *data.Store) *cobra.Command {
	var limit int
	var showAll bool

	cmd := &cobra.Command{
		Use:   "history",
		Short: "View past orchestration sessions",
		Long: `View completed orchestration sessions and their results.

Shows which tasks were worked on, their outcomes, and timing.

Examples:
  gc history           # Show last 10 completed sessions
  gc history -n 20     # Show last 20 sessions
  gc history --all     # Show all sessions`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistory(store, limit, showAll)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "Number of sessions to show")
	cmd.Flags().BoolVar(&showAll, "all", false, "Show all sessions")

	return cmd
}

func runHistory(store *data.Store, limit int, showAll bool) error {
	sessions, err := store.LoadSessions()
	if err != nil {
		return fmt.Errorf("loading sessions: %w", err)
	}

	fmt.Println(historyHeaderStyle.Render("═══ Orchestration History ═══"))
	fmt.Println()

	if len(sessions) == 0 {
		fmt.Println(historyDimStyle.Render("No orchestration sessions found."))
		fmt.Println()
		fmt.Println(historyDimStyle.Render("Run 'gc orc <task_ids>' to start orchestrating tasks."))
		return nil
	}

	// Sort by completion/update time (most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		ti := sessions[i].UpdatedAt
		if sessions[i].CompletedAt != nil {
			ti = *sessions[i].CompletedAt
		}
		tj := sessions[j].UpdatedAt
		if sessions[j].CompletedAt != nil {
			tj = *sessions[j].CompletedAt
		}
		return ti.After(tj)
	})

	// Apply limit
	displayCount := len(sessions)
	if !showAll && displayCount > limit {
		displayCount = limit
	}

	for i := 0; i < displayCount; i++ {
		displayHistorySession(sessions[i])
	}

	if !showAll && len(sessions) > limit {
		fmt.Println()
		fmt.Printf("%s\n", historyDimStyle.Render(fmt.Sprintf("Showing %d of %d sessions. Use --all to see all.", limit, len(sessions))))
	}

	return nil
}

func displayHistorySession(s types.Session) {
	// Status icon and style
	var statusIcon string
	var statusStyle lipgloss.Style
	switch s.Status {
	case types.SessionStatusCompleted:
		statusIcon = "✓"
		statusStyle = historySuccessStyle
	case types.SessionStatusFailed:
		statusIcon = "✗"
		statusStyle = historyFailedStyle
	case types.SessionStatusCancelled:
		statusIcon = "⊘"
		statusStyle = historyDimStyle
	case types.SessionStatusRunning:
		statusIcon = "▶"
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	default:
		statusIcon = "○"
		statusStyle = historyDimStyle
	}

	// Calculate duration
	var duration string
	endTime := s.UpdatedAt
	if s.CompletedAt != nil {
		endTime = *s.CompletedAt
	}
	d := endTime.Sub(s.StartedAt)
	if d < time.Minute {
		duration = fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		duration = fmt.Sprintf("%dm", int(d.Minutes()))
	} else {
		duration = fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}

	// Task count
	taskCount := len(s.TaskIDs)
	taskText := "task"
	if taskCount != 1 {
		taskText = "tasks"
	}

	// Time info
	timeStr := s.StartedAt.Format("Jan 2 15:04")

	// Main line
	fmt.Printf("%s %s  %s  %d %s  %s  %s\n",
		statusStyle.Render(statusIcon),
		statusStyle.Render(fmt.Sprintf("%-10s", s.Status)),
		historyDimStyle.Render(s.ID[:min(12, len(s.ID))]),
		taskCount,
		historyDimStyle.Render(taskText),
		historyDimStyle.Render(duration),
		historyDimStyle.Render(timeStr),
	)

	// Show tasks if any
	if len(s.TaskIDs) > 0 && len(s.TaskIDs) <= 5 {
		for _, taskID := range s.TaskIDs {
			shortID := taskID
			if len(shortID) > 30 {
				shortID = shortID[:27] + "..."
			}
			fmt.Printf("    └─ %s\n", historyDimStyle.Render(shortID))
		}
	} else if len(s.TaskIDs) > 5 {
		for i := 0; i < 3; i++ {
			shortID := s.TaskIDs[i]
			if len(shortID) > 30 {
				shortID = shortID[:27] + "..."
			}
			fmt.Printf("    └─ %s\n", historyDimStyle.Render(shortID))
		}
		fmt.Printf("    └─ %s\n", historyDimStyle.Render(fmt.Sprintf("... and %d more", len(s.TaskIDs)-3)))
	}
	fmt.Println()
}
