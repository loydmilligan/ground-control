// Package session manages work sessions for task orchestration.
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Status represents the current state of a work session.
type Status string

const (
	StatusStarted   Status = "started"
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Session represents a work session for task orchestration.
type Session struct {
	ID        string    `json:"id"`
	Status    Status    `json:"status"`
	TaskIDs   []string  `json:"task_ids"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`

	// Current execution state
	CurrentTaskID    string `json:"current_task_id,omitempty"`
	CurrentStage     string `json:"current_stage,omitempty"`
	CurrentIteration int    `json:"current_iteration,omitempty"`

	// Task progress
	TaskProgress map[string]*TaskProgress `json:"task_progress"`

	// Collected issues for self-learning
	Issues []SessionIssue `json:"issues,omitempty"`

	// Session notes
	Notes string `json:"notes,omitempty"`
}

// TaskProgress tracks progress for a single task within a session.
type TaskProgress struct {
	TaskID      string            `json:"task_id"`
	Status      string            `json:"status"` // pending, running, completed, failed, escalated
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Stages      map[string]*StageProgress `json:"stages"`
	Error       string            `json:"error,omitempty"`
}

// StageProgress tracks progress for a single stage.
type StageProgress struct {
	Stage       string     `json:"stage"`
	Status      string     `json:"status"` // pending, running, completed, needs_revision, failed, skipped
	Iterations  int        `json:"iterations"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	OutputFiles []string   `json:"output_files,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// SessionIssue represents an issue collected during the session.
type SessionIssue struct {
	TaskID      string `json:"task_id"`
	Stage       string `json:"stage"`
	Severity    string `json:"severity"` // minor, moderate, significant
	Category    string `json:"category"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// Manager handles session persistence and lifecycle.
type Manager struct {
	sessionsDir string
}

// NewManager creates a new session manager.
func NewManager(dataDir string) *Manager {
	return &Manager{
		sessionsDir: filepath.Join(dataDir, "sessions"),
	}
}

// Create starts a new work session.
func (m *Manager) Create(taskIDs []string) (*Session, error) {
	// Ensure sessions directory exists
	if err := os.MkdirAll(m.sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("creating sessions directory: %w", err)
	}

	now := time.Now()
	session := &Session{
		ID:           fmt.Sprintf("session_%d", now.UnixMilli()),
		Status:       StatusStarted,
		TaskIDs:      taskIDs,
		StartedAt:    now,
		UpdatedAt:    now,
		TaskProgress: make(map[string]*TaskProgress),
	}

	// Initialize task progress
	for _, taskID := range taskIDs {
		session.TaskProgress[taskID] = &TaskProgress{
			TaskID: taskID,
			Status: "pending",
			Stages: make(map[string]*StageProgress),
		}
	}

	if err := m.Save(session); err != nil {
		return nil, err
	}

	return session, nil
}

// Load reads a session from disk.
func (m *Manager) Load(sessionID string) (*Session, error) {
	path := filepath.Join(m.sessionsDir, sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parsing session: %w", err)
	}

	return &session, nil
}

// Save writes a session to disk.
func (m *Manager) Save(session *Session) error {
	session.UpdatedAt = time.Now()

	path := filepath.Join(m.sessionsDir, session.ID+".json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}

	return nil
}

// GetLatest returns the most recent session, if any.
func (m *Manager) GetLatest() (*Session, error) {
	entries, err := os.ReadDir(m.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading sessions directory: %w", err)
	}

	var latestName string
	var latestTime time.Time

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestName = entry.Name()
		}
	}

	if latestName == "" {
		return nil, nil
	}

	sessionID := latestName[:len(latestName)-5] // Remove .json
	return m.Load(sessionID)
}

// GetPaused returns the most recent paused session for resumption.
func (m *Manager) GetPaused() (*Session, error) {
	entries, err := os.ReadDir(m.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading sessions directory: %w", err)
	}

	var latestPaused *Session
	var latestTime time.Time

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		sessionID := entry.Name()[:len(entry.Name())-5]
		session, err := m.Load(sessionID)
		if err != nil {
			continue
		}

		if session.Status == StatusPaused && session.UpdatedAt.After(latestTime) {
			latestTime = session.UpdatedAt
			latestPaused = session
		}
	}

	return latestPaused, nil
}

// UpdateStatus updates the session status.
func (s *Session) UpdateStatus(status Status) {
	s.Status = status
	s.UpdatedAt = time.Now()
	if status == StatusCompleted || status == StatusFailed {
		now := time.Now()
		s.EndedAt = &now
	}
}

// SetCurrentTask sets the current task being processed.
func (s *Session) SetCurrentTask(taskID string) {
	s.CurrentTaskID = taskID
	s.UpdatedAt = time.Now()

	if progress, ok := s.TaskProgress[taskID]; ok {
		progress.Status = "running"
		now := time.Now()
		progress.StartedAt = &now
	}
}

// SetCurrentStage sets the current stage for the current task.
func (s *Session) SetCurrentStage(stage string, iteration int) {
	s.CurrentStage = stage
	s.CurrentIteration = iteration
	s.UpdatedAt = time.Now()

	if progress, ok := s.TaskProgress[s.CurrentTaskID]; ok {
		if _, exists := progress.Stages[stage]; !exists {
			progress.Stages[stage] = &StageProgress{
				Stage:  stage,
				Status: "pending",
			}
		}
		stageProgress := progress.Stages[stage]
		stageProgress.Status = "running"
		stageProgress.Iterations = iteration
		now := time.Now()
		stageProgress.StartedAt = &now
	}
}

// CompleteStage marks the current stage as completed.
func (s *Session) CompleteStage(stage string, outputFiles []string) {
	s.UpdatedAt = time.Now()

	if progress, ok := s.TaskProgress[s.CurrentTaskID]; ok {
		if stageProgress, exists := progress.Stages[stage]; exists {
			stageProgress.Status = "completed"
			stageProgress.OutputFiles = outputFiles
			now := time.Now()
			stageProgress.CompletedAt = &now
		}
	}
}

// FailStage marks a stage as failed.
func (s *Session) FailStage(stage string, err string) {
	s.UpdatedAt = time.Now()

	if progress, ok := s.TaskProgress[s.CurrentTaskID]; ok {
		if stageProgress, exists := progress.Stages[stage]; exists {
			stageProgress.Status = "failed"
			stageProgress.Error = err
			now := time.Now()
			stageProgress.CompletedAt = &now
		}
	}
}

// CompleteTask marks the current task as completed.
func (s *Session) CompleteTask(taskID string) {
	s.UpdatedAt = time.Now()

	if progress, ok := s.TaskProgress[taskID]; ok {
		progress.Status = "completed"
		now := time.Now()
		progress.CompletedAt = &now
	}
}

// FailTask marks a task as failed.
func (s *Session) FailTask(taskID string, err string) {
	s.UpdatedAt = time.Now()

	if progress, ok := s.TaskProgress[taskID]; ok {
		progress.Status = "failed"
		progress.Error = err
		now := time.Now()
		progress.CompletedAt = &now
	}
}

// EscalateTask marks a task as escalated (waiting for human).
func (s *Session) EscalateTask(taskID string) {
	s.UpdatedAt = time.Now()

	if progress, ok := s.TaskProgress[taskID]; ok {
		progress.Status = "escalated"
	}
}

// AddIssue adds an issue to the session for self-learning.
func (s *Session) AddIssue(issue SessionIssue) {
	s.Issues = append(s.Issues, issue)
	s.UpdatedAt = time.Now()
}
