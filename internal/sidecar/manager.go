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

// --- Issues (.gc/issues.json) ---

// LoadIssues reads the project issues file
func (m *Manager) LoadIssues() (*IssuesFile, error) {
	path := filepath.Join(m.GCPath(), "issues.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &IssuesFile{Issues: []IssueItem{}}, nil
		}
		return nil, fmt.Errorf("load issues: %w", err)
	}
	var issues IssuesFile
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, fmt.Errorf("parse issues: %w", err)
	}
	return &issues, nil
}

// SaveIssues writes the project issues file
func (m *Manager) SaveIssues(issues *IssuesFile) error {
	if err := m.EnsureDir(); err != nil {
		return err
	}
	path := filepath.Join(m.GCPath(), "issues.json")
	data, err := json.MarshalIndent(issues, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal issues: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// --- Roadmap (.gc/roadmap.json) ---

// LoadRoadmap reads the project roadmap file
func (m *Manager) LoadRoadmap() (*RoadmapFile, error) {
	path := filepath.Join(m.GCPath(), "roadmap.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RoadmapFile{Features: []RoadmapFeature{}, Milestones: []RoadmapMilestone{}}, nil
		}
		return nil, fmt.Errorf("load roadmap: %w", err)
	}
	var roadmap RoadmapFile
	if err := json.Unmarshal(data, &roadmap); err != nil {
		return nil, fmt.Errorf("parse roadmap: %w", err)
	}
	return &roadmap, nil
}

// SaveRoadmap writes the project roadmap file
func (m *Manager) SaveRoadmap(roadmap *RoadmapFile) error {
	if err := m.EnsureDir(); err != nil {
		return err
	}
	path := filepath.Join(m.GCPath(), "roadmap.json")
	data, err := json.MarshalIndent(roadmap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal roadmap: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// --- Requests (.gc/requests.jsonl) ---

// LoadRequests reads all FD requests from the project
func (m *Manager) LoadRequests() ([]FDRequest, error) {
	path := filepath.Join(m.GCPath(), "requests.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []FDRequest{}, nil
		}
		return nil, fmt.Errorf("load requests: %w", err)
	}

	var requests []FDRequest
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var req FDRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue // Skip malformed lines
		}
		requests = append(requests, req)
	}
	return requests, nil
}

// AppendRequest adds a new request to the requests file
func (m *Manager) AppendRequest(req *FDRequest) error {
	if err := m.EnsureDir(); err != nil {
		return err
	}
	path := filepath.Join(m.GCPath(), "requests.jsonl")
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open requests: %w", err)
	}
	defer f.Close()
	_, err = f.WriteString(string(data) + "\n")
	return err
}

// --- Learning (.gc/learning.jsonl) ---

// LoadLearnings reads all learning entries from the project
func (m *Manager) LoadLearnings() ([]LearningEntry, error) {
	path := filepath.Join(m.GCPath(), "learning.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []LearningEntry{}, nil
		}
		return nil, fmt.Errorf("load learnings: %w", err)
	}

	var learnings []LearningEntry
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry LearningEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}
		learnings = append(learnings, entry)
	}
	return learnings, nil
}

// AppendLearning adds a new learning entry
func (m *Manager) AppendLearning(entry *LearningEntry) error {
	if err := m.EnsureDir(); err != nil {
		return err
	}
	path := filepath.Join(m.GCPath(), "learning.jsonl")
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal learning: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open learning: %w", err)
	}
	defer f.Close()
	_, err = f.WriteString(string(data) + "\n")
	return err
}

// --- Helpers ---

// splitLines splits a string into lines, handling both \n and \r\n
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
