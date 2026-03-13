// Package sidecar manages project-local .gc/ state files
package sidecar

import (
	"fmt"
	"time"
)

// ProjectConfig lives in .gc/project.json - rarely changes
type ProjectConfig struct {
	Name      string    `json:"name"`
	AdoptedAt time.Time `json:"adopted_at"`

	TechStack TechStack `json:"tech_stack"`

	// Engagement settings
	Altitude  string `json:"altitude"` // low, mid, high
	CIEnabled bool   `json:"ci_enabled"`

	// Project lifecycle phase (idea, design, mvp, beta, features, maintenance, etc.)
	Phase string `json:"phase,omitempty"`

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

	// Flight Deck additions
	Sprint    *SprintInfo     `json:"sprint,omitempty"`
	Attention []AttentionFlag `json:"attention,omitempty"`
	WorkMode  WorkMode        `json:"work_mode,omitempty"`
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

// ============================================================================
// Flight Deck Types
// ============================================================================

// WorkMode defines how work is done on this project
type WorkMode string

const (
	WorkModeHuman      WorkMode = "human"      // Human does the work
	WorkModeAssisted   WorkMode = "assisted"   // AI assists, human drives
	WorkModeAutonomous WorkMode = "autonomous" // AI drives, human approves
)

// SprintInfo tracks the current sprint for dashboard display
type SprintInfo struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	TasksTotal     int     `json:"tasks_total"`
	TasksCompleted int     `json:"tasks_completed"`
	CompletionPct  float64 `json:"completion_pct"`
}

// AttentionFlag indicates something needs user attention
type AttentionFlag struct {
	Type     string    `json:"type"`     // blocked, review_needed, stale, deadline, request
	Priority string    `json:"priority"` // high, medium, low
	Message  string    `json:"message"`
	Since    time.Time `json:"since"`
}

// ============================================================================
// Issues (.gc/issues.json)
// ============================================================================

// IssuesFile represents .gc/issues.json
type IssuesFile struct {
	Issues []IssueItem `json:"issues"`
}

// IssueItem represents a single issue/bug
type IssueItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Type        string    `json:"type"`     // bug, enhancement, question, task
	Priority    string    `json:"priority"` // high, medium, low
	Status      string    `json:"status"`   // open, in_progress, closed
	Labels      []string  `json:"labels,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
}

// ============================================================================
// Roadmap (.gc/roadmap.json)
// ============================================================================

// RoadmapFile represents .gc/roadmap.json
type RoadmapFile struct {
	Features   []RoadmapFeature   `json:"features"`
	Milestones []RoadmapMilestone `json:"milestones,omitempty"`
}

// RoadmapFeature represents a planned feature
type RoadmapFeature struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description,omitempty"`
	Status        string    `json:"status"`   // planned, in_progress, completed, cancelled
	Priority      string    `json:"priority"` // high, medium, low
	CompletionPct float64   `json:"completion_pct"`
	MilestoneID   string    `json:"milestone_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// RoadmapMilestone represents a release milestone
type RoadmapMilestone struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	TargetDate string    `json:"target_date,omitempty"`
	FeatureIDs []string  `json:"feature_ids"`
	Status     string    `json:"status"` // planned, in_progress, completed
}

// ============================================================================
// Learning Log (.gc/learning.jsonl)
// ============================================================================

// LearningEntry represents a single learning/improvement observation
type LearningEntry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`   // friction, process_failure, idea
	Actor     string    `json:"actor"`  // user, fd_cc, proj_cc
	Summary   string    `json:"summary"`
	Detail    string    `json:"detail,omitempty"`
	Processed bool      `json:"processed"`
	At        time.Time `json:"at"`
}

// ============================================================================
// FD Requests (.gc/requests.jsonl)
// ============================================================================

// FDRequest represents a request from project to Flight Deck
type FDRequest struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"` // commit, review, docs, decision, help
	Summary string                 `json:"summary"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Status  string                 `json:"status"` // pending, processing, completed
	At      time.Time              `json:"at"`
}

// ============================================================================
// Workflow State (.gc/workflow.json)
// ============================================================================

// WorkflowState tracks multi-step workflow progress
type WorkflowState struct {
	WorkflowID   string            `json:"workflow_id"`
	WorkflowType string            `json:"workflow_type"` // new_project, etc.
	CurrentStep  int               `json:"current_step"`
	StartedAt    time.Time         `json:"started_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Steps        []WorkflowStep    `json:"steps"`
	Context      map[string]string `json:"context,omitempty"` // reference_file, etc.
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	Step        int        `json:"step"`
	Name        string     `json:"name"`
	Status      string     `json:"status"` // pending, in_progress, completed, skipped
	SessionID   string     `json:"session_id,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Output      string     `json:"output,omitempty"` // path to output artifact
}

// NewProjectWorkflow creates a workflow state for new project creation
func NewProjectWorkflow(refFile string) *WorkflowState {
	now := time.Now()
	ctx := make(map[string]string)
	if refFile != "" {
		ctx["reference_file"] = refFile
	}
	return &WorkflowState{
		WorkflowID:   fmt.Sprintf("wf_%d", now.Unix()),
		WorkflowType: "new_project",
		CurrentStep:  0,
		StartedAt:    now,
		UpdatedAt:    now,
		Context:      ctx,
		Steps: []WorkflowStep{
			{Step: 1, Name: "setup", Status: "pending"},
			{Step: 2, Name: "discovery", Status: "pending", Output: "docs/design.md"},
			{Step: 3, Name: "features", Status: "pending", Output: "docs/features.md"},
			{Step: 4, Name: "ui_spec", Status: "pending", Output: "docs/ui-spec.md"},
			{Step: 5, Name: "scaffold", Status: "pending", Output: "project structure"},
		},
	}
}
