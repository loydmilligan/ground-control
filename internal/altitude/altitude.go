// Package altitude manages engagement levels for Flight Deck projects
package altitude

import (
	"github.com/mmariani/ground-control/internal/sidecar"
)

// Level represents an engagement altitude
type Level string

const (
	// Low - Human drives, AI assists
	// Manual session start, approve all operations, passive monitoring
	Low Level = "low"

	// Mid - Balanced partnership (default)
	// Sessions on request, approve destructive ops only, active monitoring
	Mid Level = "mid"

	// High - AI drives, human monitors
	// Auto-start sessions, no approvals required, alert only on issues
	High Level = "high"
)

// Config holds altitude-specific behavior settings
type Config struct {
	Level             Level
	AutoStartSessions bool
	RequireApprovals  ApprovalRequirements
	MonitoringMode    MonitoringMode
}

// ApprovalRequirements defines what needs approval at each altitude
type ApprovalRequirements struct {
	DestructiveCommands bool
	NetworkAccess       bool
	InstallPackages     bool
	GitPush             bool
	FileWrites          bool
	AllOperations       bool
}

// MonitoringMode defines how actively the system monitors
type MonitoringMode string

const (
	MonitorPassive MonitoringMode = "passive" // Just record, don't alert
	MonitorActive  MonitoringMode = "active"  // Record and show in TUI
	MonitorAlert   MonitoringMode = "alert"   // Only alert on issues
)

// GetConfig returns the configuration for an altitude level
func GetConfig(level Level) Config {
	switch level {
	case Low:
		return Config{
			Level:             Low,
			AutoStartSessions: false,
			RequireApprovals: ApprovalRequirements{
				DestructiveCommands: true,
				NetworkAccess:       true,
				InstallPackages:     true,
				GitPush:             true,
				FileWrites:          true,
				AllOperations:       true,
			},
			MonitoringMode: MonitorPassive,
		}
	case High:
		return Config{
			Level:             High,
			AutoStartSessions: true,
			RequireApprovals: ApprovalRequirements{
				DestructiveCommands: false,
				NetworkAccess:       false,
				InstallPackages:     false,
				GitPush:             false,
				FileWrites:          false,
				AllOperations:       false,
			},
			MonitoringMode: MonitorAlert,
		}
	default: // Mid is default
		return Config{
			Level:             Mid,
			AutoStartSessions: false,
			RequireApprovals: ApprovalRequirements{
				DestructiveCommands: true,
				NetworkAccess:       false,
				InstallPackages:     true,
				GitPush:             true,
				FileWrites:          false,
				AllOperations:       false,
			},
			MonitoringMode: MonitorActive,
		}
	}
}

// FromString converts a string to Level, defaulting to Mid
func FromString(s string) Level {
	switch s {
	case "low":
		return Low
	case "high":
		return High
	default:
		return Mid
	}
}

// String returns the string representation
func (l Level) String() string {
	return string(l)
}

// Description returns a human-readable description
func (l Level) Description() string {
	switch l {
	case Low:
		return "Human drives, AI assists"
	case High:
		return "AI drives, human monitors"
	default:
		return "Balanced partnership"
	}
}

// Icon returns an icon for the altitude
func (l Level) Icon() string {
	switch l {
	case Low:
		return "🛬" // Landing/low
	case High:
		return "🚀" // Rocket/high
	default:
		return "✈️" // Cruising/mid
	}
}

// NeedsApproval checks if an operation type needs approval at this altitude
func (c Config) NeedsApproval(opType string) bool {
	if c.RequireApprovals.AllOperations {
		return true
	}

	switch opType {
	case "destructive", "command":
		return c.RequireApprovals.DestructiveCommands
	case "network":
		return c.RequireApprovals.NetworkAccess
	case "install":
		return c.RequireApprovals.InstallPackages
	case "git_push":
		return c.RequireApprovals.GitPush
	case "file_write":
		return c.RequireApprovals.FileWrites
	default:
		return false
	}
}

// GetProjectAltitude returns the altitude config for a project
func GetProjectAltitude(projectPath string) Config {
	mgr := sidecar.NewManager(projectPath)
	cfg, err := mgr.LoadConfig()
	if err != nil {
		return GetConfig(Mid) // Default to mid
	}
	return GetConfig(FromString(cfg.Altitude))
}

// SetProjectAltitude updates the altitude for a project
func SetProjectAltitude(projectPath string, level Level) error {
	mgr := sidecar.NewManager(projectPath)
	cfg, err := mgr.LoadConfig()
	if err != nil {
		return err
	}

	cfg.Altitude = level.String()

	// Update approval config based on altitude
	altCfg := GetConfig(level)
	cfg.Approvals = sidecar.ApprovalConfig{
		DestructiveCommands: altCfg.RequireApprovals.DestructiveCommands,
		NetworkAccess:       altCfg.RequireApprovals.NetworkAccess,
		InstallPackages:     altCfg.RequireApprovals.InstallPackages,
		GitPush:             altCfg.RequireApprovals.GitPush,
	}

	return mgr.SaveConfig(cfg)
}

// CycleAltitude returns the next altitude in the cycle
func CycleAltitude(current Level) Level {
	switch current {
	case Low:
		return Mid
	case Mid:
		return High
	case High:
		return Low
	default:
		return Mid
	}
}
