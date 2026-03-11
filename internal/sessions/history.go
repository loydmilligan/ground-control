// Package sessions provides session history management for Flight Deck
package sessions

import (
	"sort"
	"time"

	"github.com/mmariani/ground-control/internal/registry"
	"github.com/mmariani/ground-control/internal/sidecar"
)

// History provides session history queries
type History struct{}

// NewHistory creates a new session history manager
func NewHistory() *History {
	return &History{}
}

// ProjectSessions returns sessions for a specific project, sorted by start time (newest first)
func (h *History) ProjectSessions(projectPath string, limit int) ([]*sidecar.SessionRecord, error) {
	mgr := sidecar.NewManager(projectPath)
	sessions, err := mgr.ListSessions()
	if err != nil {
		return nil, err
	}

	// Sort by start time descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartedAt.After(sessions[j].StartedAt)
	})

	// Apply limit
	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}

	return sessions, nil
}

// TodaySessions returns all sessions from today across all projects
func (h *History) TodaySessions() ([]*ProjectSession, error) {
	reg, err := registry.NewRegistry()
	if err != nil {
		return nil, err
	}

	projects, err := reg.ListProjects()
	if err != nil {
		return nil, err
	}

	today := time.Now().Truncate(24 * time.Hour)
	var sessions []*ProjectSession

	for _, p := range projects {
		mgr := sidecar.NewManager(p.Path)
		projectSessions, err := mgr.ListSessions()
		if err != nil {
			continue
		}

		for _, s := range projectSessions {
			if s.StartedAt.After(today) || s.StartedAt.Equal(today) {
				sessions = append(sessions, &ProjectSession{
					ProjectName: p.Name,
					ProjectPath: p.Path,
					Session:     s,
				})
			}
		}
	}

	// Sort by start time descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Session.StartedAt.After(sessions[j].Session.StartedAt)
	})

	return sessions, nil
}

// ProjectSession is a session with project context
type ProjectSession struct {
	ProjectName string
	ProjectPath string
	Session     *sidecar.SessionRecord
}

// Stats calculates session statistics for a project
func (h *History) Stats(projectPath string) (*SessionStats, error) {
	sessions, err := h.ProjectSessions(projectPath, 0)
	if err != nil {
		return nil, err
	}

	stats := &SessionStats{}
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	weekAgo := today.AddDate(0, 0, -7)

	for _, s := range sessions {
		stats.TotalSessions++
		stats.TotalMinutes += s.DurationMinutes
		stats.TotalTokens += s.TokensUsed
		stats.TotalCostUSD += s.CostUSD

		if s.StartedAt.After(today) || s.StartedAt.Equal(today) {
			stats.TodaySessions++
			stats.TodayMinutes += s.DurationMinutes
		}

		if s.StartedAt.After(weekAgo) || s.StartedAt.Equal(weekAgo) {
			stats.WeekSessions++
			stats.WeekMinutes += s.DurationMinutes
		}

		// Track outcomes
		switch s.Outcome {
		case "completed":
			stats.CompletedSessions++
		case "errored":
			stats.ErroredSessions++
		}
	}

	if stats.TotalSessions > 0 {
		stats.AvgMinutes = stats.TotalMinutes / stats.TotalSessions
		stats.AvgTokens = stats.TotalTokens / stats.TotalSessions
	}

	return stats, nil
}

// SessionStats holds aggregated session statistics
type SessionStats struct {
	TotalSessions     int
	TodaySessions     int
	WeekSessions      int
	CompletedSessions int
	ErroredSessions   int

	TotalMinutes int
	TodayMinutes int
	WeekMinutes  int
	AvgMinutes   int

	TotalTokens int
	AvgTokens   int

	TotalCostUSD float64
}

// ActiveSessions returns currently active sessions across all projects
func (h *History) ActiveSessions() ([]*ActiveSession, error) {
	reg, err := registry.NewRegistry()
	if err != nil {
		return nil, err
	}

	projects, err := reg.ListProjects()
	if err != nil {
		return nil, err
	}

	var active []*ActiveSession
	for _, p := range projects {
		mgr := sidecar.NewManager(p.Path)
		state, err := mgr.LoadState()
		if err != nil {
			continue
		}

		if state.Session.Status == "active" {
			active = append(active, &ActiveSession{
				ProjectName: p.Name,
				ProjectPath: p.Path,
				Session:     state.Session,
			})
		}
	}

	return active, nil
}

// ActiveSession represents a currently running session
type ActiveSession struct {
	ProjectName string
	ProjectPath string
	Session     sidecar.SessionInfo
}

// FormatDuration formats minutes as "Xh Ym"
func FormatDuration(minutes int) string {
	if minutes < 60 {
		return intToStr(minutes) + "m"
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return intToStr(hours) + "h"
	}
	return intToStr(hours) + "h " + intToStr(mins) + "m"
}

// RelativeTime formats time as "X ago"
func RelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return intToStr(int(d.Minutes())) + "m ago"
	case d < 24*time.Hour:
		return intToStr(int(d.Hours())) + "h ago"
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return intToStr(days) + "d ago"
	}
}

func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}
