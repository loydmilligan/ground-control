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
	sessionHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	sessionActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Bold(true)
	sessionStaleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	sessionDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// NewSessionsCmd creates the sessions command for session management.
func NewSessionsCmd(store *data.Store) *cobra.Command {
	var listFlag bool
	var cleanupFlag bool
	var cancelID string
	var staleHours int

	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "Manage orchestration sessions",
		Long: `List, cleanup, and cancel orchestration sessions.

Sessions track gc orc runs. Old or stuck sessions can be cleaned up.

Examples:
  gc sessions --list              # List all sessions
  gc sessions --cleanup           # Remove stale sessions (>1hr with no updates)
  gc sessions --cancel <id>       # Cancel a specific session
  gc sessions --cleanup --stale 2 # Remove sessions stale for >2 hours`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cancelID != "" {
				return cancelSession(store, cancelID)
			}
			if cleanupFlag {
				return cleanupSessions(store, staleHours)
			}
			// Default: list
			return listSessions(store, staleHours)
		},
	}

	cmd.Flags().BoolVarP(&listFlag, "list", "l", false, "List all sessions")
	cmd.Flags().BoolVar(&cleanupFlag, "cleanup", false, "Remove stale sessions")
	cmd.Flags().StringVar(&cancelID, "cancel", "", "Cancel a specific session by ID")
	cmd.Flags().IntVar(&staleHours, "stale", 1, "Hours before a session is considered stale")

	return cmd
}

func listSessions(store *data.Store, staleHours int) error {
	sessions, err := store.LoadSessions()
	if err != nil {
		return fmt.Errorf("loading sessions: %w", err)
	}

	fmt.Println(sessionHeaderStyle.Render("═══ Orchestration Sessions ═══"))
	fmt.Println()

	if len(sessions) == 0 {
		fmt.Println(sessionDimStyle.Render("No sessions found."))
		return nil
	}

	// Sort by updated time (most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	staleThreshold := time.Now().Add(-time.Duration(staleHours) * time.Hour)
	activeCount := 0
	staleCount := 0
	completedCount := 0

	for _, s := range sessions {
		var statusStyle lipgloss.Style
		var statusLabel string

		switch s.Status {
		case types.SessionStatusCompleted:
			statusStyle = sessionDimStyle
			statusLabel = "completed"
			completedCount++
		case types.SessionStatusRunning:
			if s.UpdatedAt.Before(staleThreshold) {
				statusStyle = sessionStaleStyle
				statusLabel = "stale"
				staleCount++
			} else {
				statusStyle = sessionActiveStyle
				statusLabel = "running"
				activeCount++
			}
		case types.SessionStatusCancelled:
			statusStyle = sessionDimStyle
			statusLabel = "cancelled"
		case types.SessionStatusFailed:
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			statusLabel = "failed"
		default:
			statusStyle = sessionDimStyle
			statusLabel = string(s.Status)
		}

		ago := sessionsFormatTimeAgo(s.UpdatedAt)
		fmt.Printf("  %s  %s  %s\n",
			statusStyle.Render(fmt.Sprintf("%-10s", statusLabel)),
			s.ID[:min(15, len(s.ID))],
			sessionDimStyle.Render(ago))

		if s.CurrentTaskID != nil {
			fmt.Printf("           └─ %s\n", sessionDimStyle.Render(*s.CurrentTaskID))
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d  Active: %d  Stale: %d  Completed: %d\n",
		len(sessions), activeCount, staleCount, completedCount)

	if staleCount > 0 {
		fmt.Println()
		fmt.Println(sessionDimStyle.Render("Run 'gc sessions --cleanup' to remove stale sessions."))
	}

	return nil
}

func cleanupSessions(store *data.Store, staleHours int) error {
	sessions, err := store.LoadSessions()
	if err != nil {
		return fmt.Errorf("loading sessions: %w", err)
	}

	staleThreshold := time.Now().Add(-time.Duration(staleHours) * time.Hour)
	var kept []types.Session
	var removed []string

	for _, s := range sessions {
		isStale := s.Status == types.SessionStatusRunning && s.UpdatedAt.Before(staleThreshold)
		if isStale {
			removed = append(removed, s.ID)
		} else {
			kept = append(kept, s)
		}
	}

	if len(removed) == 0 {
		fmt.Println("No stale sessions to clean up.")
		return nil
	}

	if err := store.SaveSessions(kept); err != nil {
		return fmt.Errorf("saving sessions: %w", err)
	}

	fmt.Println(sessionHeaderStyle.Render("═══ Sessions Cleanup ═══"))
	fmt.Println()
	fmt.Printf("Removed %d stale sessions:\n", len(removed))
	for _, id := range removed {
		fmt.Printf("  - %s\n", id)
	}
	fmt.Println()
	fmt.Printf("Remaining sessions: %d\n", len(kept))

	return nil
}

func cancelSession(store *data.Store, sessionID string) error {
	sessions, err := store.LoadSessions()
	if err != nil {
		return fmt.Errorf("loading sessions: %w", err)
	}

	found := false
	for i, s := range sessions {
		if s.ID == sessionID || (len(s.ID) > len(sessionID) && s.ID[:len(sessionID)] == sessionID) {
			now := time.Now()
			sessions[i].Status = types.SessionStatusCancelled
			sessions[i].CompletedAt = &now
			found = true
			fmt.Printf("Session %s cancelled.\n", sessions[i].ID)
			break
		}
	}

	if !found {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if err := store.SaveSessions(sessions); err != nil {
		return fmt.Errorf("saving sessions: %w", err)
	}

	return nil
}

func sessionsFormatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}
