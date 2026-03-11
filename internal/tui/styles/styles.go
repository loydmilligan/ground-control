// Package styles provides Lipgloss styles for the Flight Deck TUI
package styles

import "github.com/charmbracelet/lipgloss"

// Colors (xterm-256 palette) - reused from GC v1
var (
	Primary   = lipgloss.Color("99")  // Purple
	Secondary = lipgloss.Color("39")  // Cyan
	Muted     = lipgloss.Color("241") // Gray
	Success   = lipgloss.Color("40")  // Green
	Warning   = lipgloss.Color("214") // Orange
	Danger    = lipgloss.Color("196") // Red
	Light     = lipgloss.Color("252") // Light gray
	BgDark    = lipgloss.Color("236") // Dark background
)

// Layout styles
var (
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		MarginLeft(1)

	Subtitle = lipgloss.NewStyle().
			Foreground(Secondary).
			MarginLeft(1)

	Help = lipgloss.NewStyle().
		Foreground(Muted).
		MarginLeft(1)
)

// Container styles
var (
	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Padding(1, 2)

	MenuBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Secondary).
		Padding(1, 2)

	StatusBar = lipgloss.NewStyle().
			Background(BgDark).
			Foreground(Light).
			Padding(0, 1)
)

// Tab styles
var (
	TabActive = lipgloss.NewStyle().
			Background(Primary).
			Foreground(lipgloss.Color("230")).
			Padding(0, 2).
			Bold(true)

	TabInactive = lipgloss.NewStyle().
			Background(BgDark).
			Foreground(Muted).
			Padding(0, 2)
)

// Text styles
var (
	Label = lipgloss.NewStyle().
		Bold(true).
		Foreground(Secondary)

	Value = lipgloss.NewStyle().
		Foreground(Light)

	Active = lipgloss.NewStyle().
		Foreground(Success).
		Bold(true)

	Inactive = lipgloss.NewStyle().
			Foreground(Muted)

	Error = lipgloss.NewStyle().
		Foreground(Danger).
		Bold(true)

	// MutedText is a style for muted/secondary text
	MutedText = lipgloss.NewStyle().
			Foreground(Muted)

	// WarningText is a style for warning text
	WarningText = lipgloss.NewStyle().
			Foreground(Warning)
)

// List/menu styles
var (
	ListItem = lipgloss.NewStyle().
			PaddingLeft(2)

	ListSelected = lipgloss.NewStyle().
			PaddingLeft(2).
			Bold(true).
			Foreground(Primary).
			Background(lipgloss.Color("237"))

	Cursor = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true)
)

// Status indicators
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "active":
		return lipgloss.NewStyle().Foreground(Success).Bold(true)
	case "paused":
		return lipgloss.NewStyle().Foreground(Warning)
	case "idle":
		return lipgloss.NewStyle().Foreground(Muted)
	case "error", "failed":
		return lipgloss.NewStyle().Foreground(Danger)
	default:
		return lipgloss.NewStyle().Foreground(Light)
	}
}

// StatusIcon returns an icon for a status
func StatusIcon(status string) string {
	switch status {
	case "active":
		return "●"
	case "paused":
		return "◐"
	case "idle":
		return "○"
	case "error", "failed":
		return "✗"
	case "completed":
		return "✓"
	default:
		return "○"
	}
}

// ProgressBar renders a simple progress bar
func ProgressBar(current, max float64, width int) string {
	if max == 0 {
		max = 1
	}
	ratio := current / max
	if ratio > 1 {
		ratio = 1
	}
	filled := int(float64(width) * ratio)
	empty := width - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}
	return bar
}
