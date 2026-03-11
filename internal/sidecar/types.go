// Package sidecar manages project-local .gc/ state files
package sidecar

import "time"

// ProjectConfig lives in .gc/project.json - rarely changes
type ProjectConfig struct {
	Name      string    `json:"name"`
	AdoptedAt time.Time `json:"adopted_at"`

	TechStack TechStack `json:"tech_stack"`

	// Engagement settings
	Altitude  string `json:"altitude"` // low, mid, high
	CIEnabled bool   `json:"ci_enabled"`

	// What requires approval
	Approvals ApprovalConfig `json:"approvals"`

	// Constraints for AI agents
	Constraints []string `json:"constraints"`
}

// TechStack holds detected technologies
type TechStack struct {
	Languages      []Detection `json:"languages"`
	Frameworks     []Detection `json:"frameworks"`
	TestRunner     *Detection  `json:"test_runner,omitempty"`
	PackageManager string      `json:"package_manager,omitempty"`
	CISystem       string      `json:"ci_system,omitempty"`
}

// Detection represents a detected technology with confidence
type Detection struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	Version    string  `json:"version,omitempty"`
}

// ApprovalConfig defines what actions require user approval
type ApprovalConfig struct {
	DestructiveCommands bool `json:"destructive_commands"`
	NetworkAccess       bool `json:"network_access"`
	InstallPackages     bool `json:"install_packages"`
	GitPush             bool `json:"git_push"`
}

// ProjectState lives in .gc/state.json - updated frequently
type ProjectState struct {
	Session  SessionInfo  `json:"session"`
	Costs    CostInfo     `json:"costs"`
	Approval *Approval    `json:"pending_approval,omitempty"`
	Activity []Activity   `json:"recent_activity"`
	Context  ContextInfo  `json:"context"`
}

// SessionInfo tracks the current Claude session
type SessionInfo struct {
	ID           string    `json:"id,omitempty"`
	TmuxPane     string    `json:"tmux_pane,omitempty"`
	Status       string    `json:"status"` // active, paused, idle
	StartedAt    time.Time `json:"started_at,omitempty"`
	LastActivity time.Time `json:"last_activity,omitempty"`
	CurrentFocus string    `json:"current_focus,omitempty"`
	FilesTouched []string  `json:"files_touched,omitempty"`
}

// CostInfo tracks token/cost usage
type CostInfo struct {
	SessionTokens  int     `json:"session_tokens"`
	SessionCostUSD float64 `json:"session_cost_usd"`
	TodayTokens    int     `json:"today_tokens"`
	TodayCostUSD   float64 `json:"today_cost_usd"`
}

// Approval represents a pending approval request
type Approval struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // command, file_write, network
	Detail      string    `json:"detail"`
	Reason      string    `json:"reason"`
	RequestedAt time.Time `json:"requested_at"`
}

// Activity represents a logged action
type Activity struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // file_read, file_write, command, message, approval
	Summary   string    `json:"summary"`
}

// ContextInfo tracks what Claude knows about
type ContextInfo struct {
	KeyFiles            []string `json:"key_files,omitempty"`
	EstablishedPatterns []string `json:"established_patterns,omitempty"`
}

// SessionRecord is saved to .gc/sessions/{id}.json when session ends
type SessionRecord struct {
	ID              string    `json:"id"`
	StartedAt       time.Time `json:"started_at"`
	EndedAt         time.Time `json:"ended_at"`
	DurationMinutes int       `json:"duration_minutes"`

	Focus   string `json:"focus"`
	Outcome string `json:"outcome"` // completed, paused, stopped, errored
	Summary string `json:"summary"`

	TokensUsed int     `json:"tokens_used"`
	CostUSD    float64 `json:"cost_usd"`

	FilesCreated  []string `json:"files_created,omitempty"`
	FilesModified []string `json:"files_modified,omitempty"`
	Commits       []string `json:"commits,omitempty"`

	HandoffContext string `json:"handoff_context,omitempty"`
}

// AnalysisResult is the output from the adopt analysis prompt
type AnalysisResult struct {
	Name                   string      `json:"name"`
	Description            string      `json:"description,omitempty"`
	Languages              []Detection `json:"languages"`
	Frameworks             []Detection `json:"frameworks"`
	TestRunner             *Detection  `json:"test_runner,omitempty"`
	PackageManager         string      `json:"package_manager,omitempty"`
	CISystem               string      `json:"ci_system,omitempty"`
	ExistingConventions    []string    `json:"existing_conventions,omitempty"`
	ExistingTaskManagement string      `json:"existing_task_management,omitempty"`
	KeyFiles               []KeyFile   `json:"key_files,omitempty"`
	SuggestedConstraints   []string    `json:"suggested_constraints,omitempty"`
	Monorepo               bool        `json:"monorepo"`
	Workspaces             []Workspace `json:"workspaces,omitempty"`
}

// Workspace represents a workspace in a monorepo
type Workspace struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type,omitempty"`
}

// KeyFile represents an important file in the project
type KeyFile struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose"`
}
