// Package data handles reading and writing Ground Control JSON files.
package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mmariani/ground-control/internal/types"
)

// Store manages access to Ground Control data files.
type Store struct {
	dataDir string
}

// NewStore creates a new data store pointing to the given directory.
func NewStore(dataDir string) *Store {
	return &Store{dataDir: dataDir}
}

// tasksFile wraps the tasks.json structure.
type tasksFile struct {
	Tasks []types.Task `json:"tasks"`
}

// projectsFile wraps the projects.json structure.
type projectsFile struct {
	Projects []types.Project `json:"projects"`
}

// brainDumpFile wraps the brain-dump.json structure.
type brainDumpFile struct {
	Entries []types.BrainDumpEntry `json:"entries"`
}

// activityLogFile wraps the activity-log.json structure.
type activityLogFile struct {
	Events []types.ActivityEvent `json:"events"`
}

// agentsFile wraps the agents.json structure.
type agentsFile struct {
	Agents []types.Agent `json:"agents"`
}

// ritualsFile wraps the rituals.json structure.
type ritualsFile struct {
	Rituals []types.Ritual `json:"rituals"`
}

// sprintsFile wraps the sprints.json structure.
type sprintsFile struct {
	Sprints []types.Sprint `json:"sprints"`
}

// LoadTasks reads all tasks from tasks.json.
func (s *Store) LoadTasks() ([]types.Task, error) {
	path := filepath.Join(s.dataDir, "tasks.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading tasks.json: %w", err)
	}

	var file tasksFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing tasks.json: %w", err)
	}

	return file.Tasks, nil
}

// SaveTasks writes all tasks to tasks.json.
func (s *Store) SaveTasks(tasks []types.Task) error {
	path := filepath.Join(s.dataDir, "tasks.json")
	file := tasksFile{Tasks: tasks}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling tasks: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing tasks.json: %w", err)
	}

	return nil
}

// LoadProjects reads all projects from projects.json.
func (s *Store) LoadProjects() ([]types.Project, error) {
	path := filepath.Join(s.dataDir, "projects.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading projects.json: %w", err)
	}

	var file projectsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing projects.json: %w", err)
	}

	return file.Projects, nil
}

// SaveProjects writes all projects to projects.json.
func (s *Store) SaveProjects(projects []types.Project) error {
	path := filepath.Join(s.dataDir, "projects.json")
	file := projectsFile{Projects: projects}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling projects: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing projects.json: %w", err)
	}

	return nil
}

// LoadBrainDump reads all entries from brain-dump.json.
func (s *Store) LoadBrainDump() ([]types.BrainDumpEntry, error) {
	path := filepath.Join(s.dataDir, "brain-dump.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading brain-dump.json: %w", err)
	}

	var file brainDumpFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing brain-dump.json: %w", err)
	}

	return file.Entries, nil
}

// SaveBrainDump writes all entries to brain-dump.json.
func (s *Store) SaveBrainDump(entries []types.BrainDumpEntry) error {
	path := filepath.Join(s.dataDir, "brain-dump.json")
	file := brainDumpFile{Entries: entries}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling brain dump: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing brain-dump.json: %w", err)
	}

	return nil
}

// AddBrainDump appends a new entry to brain-dump.json.
func (s *Store) AddBrainDump(content string) (*types.BrainDumpEntry, error) {
	entries, err := s.LoadBrainDump()
	if err != nil {
		return nil, err
	}

	entry := types.BrainDumpEntry{
		ID:         fmt.Sprintf("dump_%d", time.Now().UnixMilli()),
		Content:    content,
		Processed:  false,
		CapturedAt: time.Now(),
	}

	entries = append(entries, entry)

	if err := s.SaveBrainDump(entries); err != nil {
		return nil, err
	}

	return &entry, nil
}

// LoadActivityLog reads all events from activity-log.json.
func (s *Store) LoadActivityLog() ([]types.ActivityEvent, error) {
	path := filepath.Join(s.dataDir, "activity-log.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading activity-log.json: %w", err)
	}

	var file activityLogFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing activity-log.json: %w", err)
	}

	return file.Events, nil
}

// SaveActivityLog writes all events to activity-log.json.
func (s *Store) SaveActivityLog(events []types.ActivityEvent) error {
	path := filepath.Join(s.dataDir, "activity-log.json")
	file := activityLogFile{Events: events}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling activity log: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing activity-log.json: %w", err)
	}

	return nil
}

// LoadAgents reads all agents from agents.json.
func (s *Store) LoadAgents() ([]types.Agent, error) {
	path := filepath.Join(s.dataDir, "agents.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading agents.json: %w", err)
	}

	var file agentsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing agents.json: %w", err)
	}

	return file.Agents, nil
}

// LoadRituals reads all rituals from rituals.json.
func (s *Store) LoadRituals() ([]types.Ritual, error) {
	path := filepath.Join(s.dataDir, "rituals.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading rituals.json: %w", err)
	}

	var file ritualsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing rituals.json: %w", err)
	}

	return file.Rituals, nil
}

// GetDataDir returns the configured data directory.
func (s *Store) GetDataDir() string {
	return s.dataDir
}

// LoadSessions reads all session files from the sessions directory.
func (s *Store) LoadSessions() ([]types.Session, error) {
	sessionsDir := filepath.Join(s.dataDir, "sessions")

	// Check if sessions directory exists
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return []types.Session{}, nil
	}

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("reading sessions directory: %w", err)
	}

	var sessions []types.Session
	for _, entry := range entries {
		if entry.IsDir() || !isJSONFile(entry.Name()) {
			continue
		}

		path := filepath.Join(sessionsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // Skip files we can't read
		}

		var session types.Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue // Skip files that don't parse
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// SaveSessions saves all sessions to individual files in the sessions directory.
// This replaces all existing session files with the provided sessions.
func (s *Store) SaveSessions(sessions []types.Session) error {
	sessionsDir := filepath.Join(s.dataDir, "sessions")

	// Ensure sessions directory exists
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return fmt.Errorf("creating sessions directory: %w", err)
	}

	// Build map of sessions we're keeping
	keepIDs := make(map[string]bool)
	for _, session := range sessions {
		keepIDs[session.ID] = true
	}

	// Remove session files not in the keep list
	entries, err := os.ReadDir(sessionsDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !isJSONFile(entry.Name()) {
				continue
			}
			// Extract ID from filename (session_xxx.json -> session_xxx)
			id := entry.Name()[:len(entry.Name())-5]
			if !keepIDs[id] {
				os.Remove(filepath.Join(sessionsDir, entry.Name()))
			}
		}
	}

	// Write each session to its own file
	for _, session := range sessions {
		path := filepath.Join(sessionsDir, session.ID+".json")
		data, err := json.MarshalIndent(session, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling session %s: %w", session.ID, err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("writing session %s: %w", session.ID, err)
		}
	}

	return nil
}

// isJSONFile checks if a filename ends with .json.
func isJSONFile(name string) bool {
	return len(name) > 5 && name[len(name)-5:] == ".json"
}

// LoadSprints reads all sprints from sprints.json.
func (s *Store) LoadSprints() ([]types.Sprint, error) {
	path := filepath.Join(s.dataDir, "sprints.json")

	// Return empty slice if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []types.Sprint{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading sprints.json: %w", err)
	}

	var file sprintsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing sprints.json: %w", err)
	}

	return file.Sprints, nil
}

// SaveSprints writes all sprints to sprints.json.
func (s *Store) SaveSprints(sprints []types.Sprint) error {
	path := filepath.Join(s.dataDir, "sprints.json")
	file := sprintsFile{Sprints: sprints}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling sprints: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing sprints.json: %w", err)
	}

	return nil
}

// CreateSprint creates a new sprint and saves it.
func (s *Store) CreateSprint(name, description, goal string, projectIDs []string) (*types.Sprint, error) {
	sprints, err := s.LoadSprints()
	if err != nil {
		return nil, err
	}

	sprint := types.Sprint{
		ID:          fmt.Sprintf("sprint_%d", time.Now().UnixMilli()),
		Name:        name,
		Description: description,
		Goal:        goal,
		ProjectIDs:  projectIDs,
		TaskIDs:     []string{},
		Status:      types.SprintStatusActive,
		CreatedAt:   time.Now(),
	}

	sprints = append(sprints, sprint)

	if err := s.SaveSprints(sprints); err != nil {
		return nil, err
	}

	return &sprint, nil
}

// GetSprintByID finds a sprint by its ID.
func (s *Store) GetSprintByID(id string) (*types.Sprint, error) {
	sprints, err := s.LoadSprints()
	if err != nil {
		return nil, err
	}

	for _, sprint := range sprints {
		if sprint.ID == id {
			return &sprint, nil
		}
	}

	return nil, fmt.Errorf("sprint not found: %s", id)
}

// GetSprintByName finds a sprint by its name.
func (s *Store) GetSprintByName(name string) (*types.Sprint, error) {
	sprints, err := s.LoadSprints()
	if err != nil {
		return nil, err
	}

	for _, sprint := range sprints {
		if sprint.Name == name {
			return &sprint, nil
		}
	}

	return nil, fmt.Errorf("sprint not found: %s", name)
}

// AddTaskToSprint adds a task ID to a sprint.
func (s *Store) AddTaskToSprint(sprintID, taskID string) error {
	sprints, err := s.LoadSprints()
	if err != nil {
		return err
	}

	found := false
	for i, sprint := range sprints {
		if sprint.ID == sprintID {
			// Check if task already in sprint
			for _, id := range sprint.TaskIDs {
				if id == taskID {
					return fmt.Errorf("task %s already in sprint", taskID)
				}
			}
			sprints[i].TaskIDs = append(sprints[i].TaskIDs, taskID)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("sprint not found: %s", sprintID)
	}

	return s.SaveSprints(sprints)
}

// UpdateSprintStatus updates a sprint's status.
func (s *Store) UpdateSprintStatus(sprintID string, status types.SprintStatus) error {
	sprints, err := s.LoadSprints()
	if err != nil {
		return err
	}

	found := false
	for i, sprint := range sprints {
		if sprint.ID == sprintID {
			sprints[i].Status = status
			if status == types.SprintStatusCompleted {
				now := time.Now()
				sprints[i].CompletedAt = &now
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("sprint not found: %s", sprintID)
	}

	return s.SaveSprints(sprints)
}
