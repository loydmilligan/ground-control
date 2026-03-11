package sidecar

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Manager handles reading/writing sidecar files for a project
type Manager struct {
	projectPath string
}

// NewManager creates a new sidecar manager for a project
func NewManager(projectPath string) *Manager {
	return &Manager{projectPath: projectPath}
}

// GCPath returns the .gc/ directory path
func (m *Manager) GCPath() string {
	return filepath.Join(m.projectPath, ".gc")
}

// EnsureDir creates the .gc/ directory if it doesn't exist
func (m *Manager) EnsureDir() error {
	return os.MkdirAll(m.GCPath(), 0755)
}

// Exists checks if the project has been adopted (has .gc/ dir)
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.GCPath())
	return err == nil
}

// --- Project Config (.gc/project.json) ---

// LoadConfig reads the project configuration
func (m *Manager) LoadConfig() (*ProjectConfig, error) {
	path := filepath.Join(m.GCPath(), "project.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// SaveConfig writes the project configuration
func (m *Manager) SaveConfig(cfg *ProjectConfig) error {
	if err := m.EnsureDir(); err != nil {
		return err
	}
	path := filepath.Join(m.GCPath(), "project.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// --- Project State (.gc/state.json) ---

// LoadState reads the current project state
func (m *Manager) LoadState() (*ProjectState, error) {
	path := filepath.Join(m.GCPath(), "state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty state if file doesn't exist
			return &ProjectState{
				Session: SessionInfo{Status: "idle"},
			}, nil
		}
		return nil, fmt.Errorf("load state: %w", err)
	}
	var state ProjectState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &state, nil
}

// SaveState writes the current project state
func (m *Manager) SaveState(state *ProjectState) error {
	if err := m.EnsureDir(); err != nil {
		return err
	}
	path := filepath.Join(m.GCPath(), "state.json")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// --- Analysis Result (.gc/analysis.json) ---

// LoadAnalysis reads the adoption analysis result
func (m *Manager) LoadAnalysis() (*AnalysisResult, error) {
	path := filepath.Join(m.GCPath(), "analysis.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load analysis: %w", err)
	}
	var result AnalysisResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse analysis: %w", err)
	}
	return &result, nil
}

// --- Session Records (.gc/sessions/*.json) ---

// SessionsDir returns the sessions directory path
func (m *Manager) SessionsDir() string {
	return filepath.Join(m.GCPath(), "sessions")
}

// SaveSession saves a session record
func (m *Manager) SaveSession(record *SessionRecord) error {
	dir := m.SessionsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, record.ID+".json")
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// ListSessions returns all session records
func (m *Manager) ListSessions() ([]*SessionRecord, error) {
	dir := m.SessionsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var records []*SessionRecord
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var r SessionRecord
		if json.Unmarshal(data, &r) == nil {
			records = append(records, &r)
		}
	}
	return records, nil
}

// GetLatestSession returns the most recent session
func (m *Manager) GetLatestSession() (*SessionRecord, error) {
	records, err := m.ListSessions()
	if err != nil || len(records) == 0 {
		return nil, err
	}

	latest := records[0]
	for _, r := range records[1:] {
		if r.StartedAt.After(latest.StartedAt) {
			latest = r
		}
	}
	return latest, nil
}

// --- Helpers ---

// GenerateSessionID creates a new session ID
func GenerateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().Unix())
}

// AddActivity appends an activity to the state (keeps last 20)
func (state *ProjectState) AddActivity(actType, summary string) {
	activity := Activity{
		Timestamp: time.Now(),
		Type:      actType,
		Summary:   summary,
	}
	state.Activity = append(state.Activity, activity)
	// Keep only last 20
	if len(state.Activity) > 20 {
		state.Activity = state.Activity[len(state.Activity)-20:]
	}
}

// CreateConfigFromAnalysis converts analysis result to project config
func CreateConfigFromAnalysis(analysis *AnalysisResult) *ProjectConfig {
	cfg := &ProjectConfig{
		Name:      analysis.Name,
		AdoptedAt: time.Now(),
		Altitude:  "mid", // Default
		TechStack: TechStack{
			Languages:      analysis.Languages,
			Frameworks:     analysis.Frameworks,
			TestRunner:     analysis.TestRunner,
			PackageManager: analysis.PackageManager,
			CISystem:       analysis.CISystem,
		},
		Constraints: analysis.SuggestedConstraints,
		Approvals: ApprovalConfig{
			DestructiveCommands: true,
			NetworkAccess:       false,
			InstallPackages:     true,
			GitPush:             true,
		},
	}
	return cfg
}
