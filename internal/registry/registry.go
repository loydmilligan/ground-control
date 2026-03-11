// Package registry manages the global ~/.gc/ configuration and project list
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Registry manages global Flight Deck configuration
type Registry struct {
	path string
}

// ProjectEntry represents a registered project
type ProjectEntry struct {
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	LastActive time.Time `json:"last_active"`
}

// GlobalConfig holds global Flight Deck settings
type GlobalConfig struct {
	Projects []ProjectEntry `json:"projects"`
	APIKeys  APIKeyConfig   `json:"api_keys"`
	Costs    CostSummary    `json:"costs"`
	Settings Settings       `json:"settings"`
}

// APIKeyConfig tracks API key settings
type APIKeyConfig struct {
	ActiveKeyID  string  `json:"active_key_id,omitempty"`
	RotationDate string  `json:"rotation_date,omitempty"`
	LimitType    string  `json:"limit_type,omitempty"` // daily, weekly, monthly
	LimitUSD     float64 `json:"limit_usd,omitempty"`
}

// CostSummary holds aggregated costs
type CostSummary struct {
	TodayUSD    float64   `json:"today_usd"`
	WeekUSD     float64   `json:"week_usd"`
	MonthUSD    float64   `json:"month_usd"`
	LastUpdated time.Time `json:"last_updated"`
}

// Settings holds user preferences
type Settings struct {
	DefaultAltitude string `json:"default_altitude,omitempty"` // low, mid, high
	TeleportShell   string `json:"teleport_shell,omitempty"`   // zsh, bash
}

// NewRegistry creates a new registry, ensuring ~/.gc/ exists
func NewRegistry() (*Registry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	path := filepath.Join(home, ".gc")
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("create ~/.gc: %w", err)
	}
	return &Registry{path: path}, nil
}

// Path returns the ~/.gc directory path
func (r *Registry) Path() string {
	return r.path
}

// configPath returns the path to global.json
func (r *Registry) configPath() string {
	return filepath.Join(r.path, "global.json")
}

// Load reads the global configuration
func (r *Registry) Load() (*GlobalConfig, error) {
	data, err := os.ReadFile(r.configPath())
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return &GlobalConfig{
				Settings: Settings{
					DefaultAltitude: "mid",
					TeleportShell:   "zsh",
				},
			}, nil
		}
		return nil, fmt.Errorf("load config: %w", err)
	}
	var cfg GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes the global configuration
func (r *Registry) Save(cfg *GlobalConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(r.configPath(), data, 0644)
}

// AddProject registers a project in the global list
func (r *Registry) AddProject(projectPath, name string) error {
	cfg, err := r.Load()
	if err != nil {
		return err
	}

	// Normalize path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Check if already exists, update if so
	for i, p := range cfg.Projects {
		if p.Path == absPath {
			cfg.Projects[i].Name = name
			cfg.Projects[i].LastActive = time.Now()
			return r.Save(cfg)
		}
	}

	// Add new entry
	cfg.Projects = append(cfg.Projects, ProjectEntry{
		Path:       absPath,
		Name:       name,
		LastActive: time.Now(),
	})
	return r.Save(cfg)
}

// RemoveProject removes a project from the registry
func (r *Registry) RemoveProject(projectPath string) error {
	cfg, err := r.Load()
	if err != nil {
		return err
	}

	absPath, _ := filepath.Abs(projectPath)
	var filtered []ProjectEntry
	for _, p := range cfg.Projects {
		if p.Path != absPath {
			filtered = append(filtered, p)
		}
	}
	cfg.Projects = filtered
	return r.Save(cfg)
}

// ListProjects returns all registered projects
func (r *Registry) ListProjects() ([]ProjectEntry, error) {
	cfg, err := r.Load()
	if err != nil {
		return nil, err
	}
	return cfg.Projects, nil
}

// ListProjectsSorted returns projects sorted by last active (most recent first)
func (r *Registry) ListProjectsSorted() ([]ProjectEntry, error) {
	projects, err := r.ListProjects()
	if err != nil {
		return nil, err
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].LastActive.After(projects[j].LastActive)
	})
	return projects, nil
}

// GetProject finds a project by path or name
func (r *Registry) GetProject(query string) (*ProjectEntry, error) {
	projects, err := r.ListProjects()
	if err != nil {
		return nil, err
	}

	absQuery, _ := filepath.Abs(query)
	for _, p := range projects {
		if p.Path == absQuery || p.Name == query {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("project not found: %s", query)
}

// UpdateLastActive marks a project as recently used
func (r *Registry) UpdateLastActive(projectPath string) error {
	cfg, err := r.Load()
	if err != nil {
		return err
	}

	absPath, _ := filepath.Abs(projectPath)
	for i, p := range cfg.Projects {
		if p.Path == absPath {
			cfg.Projects[i].LastActive = time.Now()
			return r.Save(cfg)
		}
	}
	return nil // Project not found, ignore
}

// GetSettings returns user settings
func (r *Registry) GetSettings() (*Settings, error) {
	cfg, err := r.Load()
	if err != nil {
		return nil, err
	}
	return &cfg.Settings, nil
}

// SaveSettings updates user settings
func (r *Registry) SaveSettings(settings *Settings) error {
	cfg, err := r.Load()
	if err != nil {
		return err
	}
	cfg.Settings = *settings
	return r.Save(cfg)
}

// UpdateCosts updates the cost summary
func (r *Registry) UpdateCosts(today, week, month float64) error {
	cfg, err := r.Load()
	if err != nil {
		return err
	}
	cfg.Costs = CostSummary{
		TodayUSD:    today,
		WeekUSD:     week,
		MonthUSD:    month,
		LastUpdated: time.Now(),
	}
	return r.Save(cfg)
}
