// Package types defines the core data structures for Ground Control.
package types

import "time"

// Task represents a unit of work in Ground Control.
type Task struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Type            TaskType        `json:"type"`
	Agent           *string         `json:"agent"`
	AssignedHuman   string          `json:"assigned_human"`
	AutonomyLevel   AutonomyLevel   `json:"autonomy_level"`
	Complexity      int             `json:"complexity"` // 1-5
	Importance      Importance      `json:"importance"`
	DueDate         *string         `json:"due_date"`
	DueUrgency      DueUrgency      `json:"due_urgency"`
	Context         TaskContext     `json:"context"`
	Topics          []string        `json:"topics"`
	State           TaskState       `json:"state"`
	BlockedBy       []string        `json:"blocked_by"`
	ConversationID  *string         `json:"conversation_id"`
	Outputs         []TaskOutput    `json:"outputs"`
	SuggestedNext   []string        `json:"suggested_next_steps"`
	AfterCompletion AfterCompletion `json:"after_completion"`
	Verification    Verification    `json:"verification"`
	ProjectID       *string         `json:"project_id"`
	Tags            []string        `json:"tags"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	CompletedAt     *time.Time      `json:"completed_at"`
	ActualMinutes   *int            `json:"actual_minutes"`
	TokensUsed      *int            `json:"tokens_used"`
	LinesChanged    *int            `json:"lines_changed"`
	ContextBundle   *ContextBundle  `json:"context_bundle,omitempty"`
}

// ContextBundle holds metadata about the context bundle built for a task.
type ContextBundle struct {
	BuiltAt   time.Time            `json:"built_at"`
	BuiltBy   string               `json:"built_by"` // Usually "context-manager"
	BundlePath string              `json:"bundle_path"` // e.g., "data/context/task_123"
	Files     ContextBundleFiles   `json:"files"`
	Notes     string               `json:"notes,omitempty"`
}

// ContextBundleFiles lists the files in a context bundle.
type ContextBundleFiles struct {
	Requirements   string   `json:"requirements"`            // Path to requirements.md
	RelevantCode   []string `json:"relevant_code"`           // Paths to code snippets
	ProjectContext string   `json:"project_context"`         // Path to project_context.md
	Patterns       string   `json:"patterns"`                // Path to patterns.md
	Decisions      string   `json:"decisions"`               // Path to decisions.md
	Conversations  string   `json:"conversations,omitempty"` // Path to conversations.md (optional)
	TestHints      string   `json:"test_hints"`              // Path to test_hints.md
}

// TaskType defines the kind of task.
type TaskType string

const (
	TaskTypeSimple     TaskType = "simple"
	TaskTypeAIPlanning TaskType = "ai-planning"
	TaskTypeResearch   TaskType = "research"
	TaskTypeCoding     TaskType = "coding"
	TaskTypeHumanInput TaskType = "human-input"
)

// AutonomyLevel defines how much independence an agent has.
type AutonomyLevel string

const (
	AutonomyFull        AutonomyLevel = "full"
	AutonomyCheckpoints AutonomyLevel = "checkpoints"
	AutonomySupervised  AutonomyLevel = "supervised"
)

// Importance defines task priority.
type Importance string

const (
	ImportanceHigh   Importance = "high"
	ImportanceMedium Importance = "medium"
	ImportanceLow    Importance = "low"
)

// DueUrgency defines how strict a due date is.
type DueUrgency string

const (
	DueUrgencyHard DueUrgency = "hard"
	DueUrgencySoft DueUrgency = "soft"
	DueUrgencyNone DueUrgency = "none"
)

// TaskState defines the lifecycle state of a task.
type TaskState string

const (
	TaskStateCreated   TaskState = "created"
	TaskStateAssigned  TaskState = "assigned"
	TaskStateBlocked   TaskState = "blocked"
	TaskStateActive    TaskState = "active"
	TaskStateWaiting   TaskState = "waiting"
	TaskStateCompleted TaskState = "completed"
)

// AfterCompletion defines what happens when a task finishes.
type AfterCompletion string

const (
	AfterCompletionTaskmasterReview AfterCompletion = "taskmaster_review"
	AfterCompletionSpawnTasks       AfterCompletion = "spawn_tasks"
	AfterCompletionNone             AfterCompletion = "none"
)

// TaskContext provides background information for task execution.
type TaskContext struct {
	Background       string   `json:"background"`
	Requirements     []string `json:"requirements"`
	Constraints      []string `json:"constraints"`
	RelatedTasks     []string `json:"related_tasks"`
	ProjectID        *string  `json:"project_id"`
	WorkingDirectory *string  `json:"working_directory,omitempty"`
}

// TaskOutput defines an expected output artifact.
type TaskOutput struct {
	Path        string `json:"path"`
	Description string `json:"description"`
	Exists      bool   `json:"exists"`
}

// Verification defines how to verify task completion.
type Verification struct {
	Type    VerificationType `json:"type"`
	Command *string          `json:"command,omitempty"`
	Paths   []string         `json:"paths,omitempty"`
}

// VerificationType defines the verification method.
type VerificationType string

const (
	VerificationTestPass      VerificationType = "test_pass"
	VerificationFileExists    VerificationType = "file_exists"
	VerificationHumanApproval VerificationType = "human_approval"
	VerificationNone          VerificationType = "none"
)

// Project represents a collection of related tasks.
type Project struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	Status        ProjectStatus `json:"status"`
	Phase         ProjectPhase `json:"phase"`
	DefaultHuman  string       `json:"default_human"`
	AllowedAgents []string     `json:"allowed_agents"`
	RepoPath      *string      `json:"repo_path"`
	TechStack     []string     `json:"tech_stack"`
	Tags          []string     `json:"tags"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// ProjectStatus defines the overall project state.
type ProjectStatus string

const (
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusPaused    ProjectStatus = "paused"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusArchived  ProjectStatus = "archived"
)

// ProjectPhase defines where a project is in its lifecycle.
type ProjectPhase string

const (
	// Core lifecycle phases
	ProjectPhaseIdea        ProjectPhase = "idea"
	ProjectPhaseDesign      ProjectPhase = "design"      // NEW: Design phase before building
	ProjectPhasePlanning    ProjectPhase = "planning"
	ProjectPhaseResearch    ProjectPhase = "research"
	ProjectPhaseScaffolding ProjectPhase = "scaffolding"
	ProjectPhaseMVP         ProjectPhase = "mvp"         // NEW: Building MVP
	ProjectPhaseBeta        ProjectPhase = "beta"        // NEW: Beta testing
	ProjectPhaseBuilding    ProjectPhase = "building"
	ProjectPhaseFeatures    ProjectPhase = "features"    // NEW: Active feature development
	ProjectPhaseTesting     ProjectPhase = "testing"
	ProjectPhaseDeployed    ProjectPhase = "deployed"
	ProjectPhaseMaintenance ProjectPhase = "maintenance"

	// Terminal/paused phases
	ProjectPhaseAbandoned   ProjectPhase = "abandoned"   // NEW: No longer active
	ProjectPhasePaused      ProjectPhase = "paused"      // NEW: Temporarily paused
)

// BrainDumpEntry represents a raw idea or note.
type BrainDumpEntry struct {
	ID             string     `json:"id"`
	Content        string     `json:"content"`
	Processed      bool       `json:"processed"`
	Category       *string    `json:"category"` // idea, bug, enhancement, question, reminder
	UrgencyHint    *string    `json:"urgency_hint"` // urgent, normal
	ConvertedTo    *string    `json:"converted_to"`
	IngestionNotes *string    `json:"ingestion_notes"`
	CapturedAt     time.Time  `json:"captured_at"`
	ProcessedAt    *time.Time `json:"processed_at"`
}

// ActivityEvent represents an entry in the activity log.
type ActivityEvent struct {
	ID                     string    `json:"id"`
	Type                   string    `json:"type"` // task_created, task_assigned, task_completed, decision_made, project_created, ritual_run
	Actor                  string    `json:"actor"`
	TaskID                 *string   `json:"task_id"`
	ProjectID              *string   `json:"project_id"`
	Summary                string    `json:"summary"`
	Reasoning              *string   `json:"reasoning"`
	AlternativesConsidered []string  `json:"alternatives_considered"`
	DecisionFactors        []string  `json:"decision_factors"`
	HumanFeedback          *string   `json:"human_feedback"`
	Timestamp              time.Time `json:"timestamp"`
}

// Agent represents an AI agent definition.
type Agent struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	Capabilities []string `json:"capabilities"`
	PromptFile   string   `json:"prompt_file"`
	Status       string   `json:"status"` // active, inactive
}

// Ritual represents an on-demand ritual like standup or weekly review.
type Ritual struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Agent       string   `json:"agent"`
	Trigger     string   `json:"trigger"` // manual, scheduled
	LastRun     *string  `json:"last_run"`
	Outputs     []string `json:"outputs"`
}

// Session represents an orchestration session.
type Session struct {
	ID               string                    `json:"id"`
	Status           SessionStatus             `json:"status"`
	TaskIDs          []string                  `json:"task_ids"`
	StartedAt        time.Time                 `json:"started_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
	CompletedAt      *time.Time                `json:"completed_at,omitempty"`
	CurrentTaskID    *string                   `json:"current_task_id,omitempty"`
	CurrentStage     *string                   `json:"current_stage,omitempty"`
	CurrentIteration int                       `json:"current_iteration"`
	TaskProgress     map[string]TaskProgress   `json:"task_progress"`
}

// SessionStatus defines the state of a session.
type SessionStatus string

const (
	SessionStatusRunning   SessionStatus = "running"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusFailed    SessionStatus = "failed"
	SessionStatusCancelled SessionStatus = "cancelled"
)

// TaskProgress tracks progress of a task within a session.
type TaskProgress struct {
	TaskID      string                  `json:"task_id"`
	Status      string                  `json:"status"`
	StartedAt   time.Time               `json:"started_at"`
	CompletedAt *time.Time              `json:"completed_at,omitempty"`
	Stages      map[string]StageProgress `json:"stages"`
}

// StageProgress tracks progress of a stage within a task.
type StageProgress struct {
	Stage       string     `json:"stage"`
	Status      string     `json:"status"`
	Iterations  int        `json:"iterations"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// Sprint represents a lightweight grouping of related tasks toward a goal.
type Sprint struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Goal        string       `json:"goal"`
	ProjectIDs  []string     `json:"project_ids,omitempty"` // Optional: associated projects
	TaskIDs     []string     `json:"task_ids"`
	Status      SprintStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
}

// SprintStatus defines the state of a sprint.
type SprintStatus string

const (
	SprintStatusActive    SprintStatus = "active"
	SprintStatusPaused    SprintStatus = "paused"
	SprintStatusCompleted SprintStatus = "completed"
)
