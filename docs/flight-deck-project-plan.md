# Flight Deck Project Plan

> Implementation roadmap for Ground Control v2

**Status**: Draft
**Created**: 2026-03-10

---

## Phase Overview

| Phase | Name | Goal | Key Deliverables |
|-------|------|------|------------------|
| **0** | Foundation | Minimal working system | Sidecar spec, TUI shell, session management |
| **1** | Flight Recorder | Track what Claude does | State watching, cost tracking, session history |
| **2** | Teleportation | Seamless context switching | Attach/detach, approval flow, messaging |
| **3** | Engagement | Altitude modes | Low/Mid/High automation levels |
| **Post** | Expansion | Future features | Web UI, Protocol Droid, mobile |

---

## Phase 0: Foundation

**Goal**: Minimal Flight Deck that can adopt a project, start a Claude session, and show it in TUI.

### P0.1: Sidecar Specification

**Files to create**:
- `internal/sidecar/types.go` — Sidecar data structures
- `internal/sidecar/manager.go` — Read/write sidecar files

**Implementation**:

```go
// internal/sidecar/types.go
package sidecar

import "time"

// ProjectConfig lives in .gc/project.json
type ProjectConfig struct {
    Name      string    `json:"name"`
    AdoptedAt time.Time `json:"adopted_at"`

    TechStack struct {
        Languages      []Detection `json:"languages"`
        Frameworks     []Detection `json:"frameworks"`
        TestRunner     *Detection  `json:"test_runner"`
        PackageManager string      `json:"package_manager"`
        CISystem       string      `json:"ci_system"`
    } `json:"tech_stack"`

    Altitude  string `json:"altitude"` // low, mid, high
    CIEnabled bool   `json:"ci_enabled"`

    Approvals struct {
        DestructiveCommands bool `json:"destructive_commands"`
        NetworkAccess       bool `json:"network_access"`
        InstallPackages     bool `json:"install_packages"`
        GitPush             bool `json:"git_push"`
    } `json:"approvals"`

    Constraints []string `json:"constraints"`
}

type Detection struct {
    Name       string  `json:"name"`
    Confidence float64 `json:"confidence"`
}

// ProjectState lives in .gc/state.json
type ProjectState struct {
    Session struct {
        ID           string    `json:"id"`
        TmuxPane     string    `json:"tmux_pane"`
        Status       string    `json:"status"` // active, paused, idle
        StartedAt    time.Time `json:"started_at"`
        LastActivity time.Time `json:"last_activity"`
        CurrentFocus string    `json:"current_focus"`
        FilesTouched []string  `json:"files_touched"`
    } `json:"session"`

    Costs struct {
        SessionTokens  int     `json:"session_tokens"`
        SessionCostUSD float64 `json:"session_cost_usd"`
        TodayTokens    int     `json:"today_tokens"`
        TodayCostUSD   float64 `json:"today_cost_usd"`
    } `json:"costs"`

    PendingApproval *Approval `json:"pending_approval"`

    RecentActivity []Activity `json:"recent_activity"`

    Context struct {
        KeyFiles            []string `json:"key_files"`
        EstablishedPatterns []string `json:"established_patterns"`
    } `json:"context"`
}

type Approval struct {
    ID          string    `json:"id"`
    Type        string    `json:"type"` // command, file_write, network
    Detail      string    `json:"detail"`
    Reason      string    `json:"reason"`
    RequestedAt time.Time `json:"requested_at"`
}

type Activity struct {
    Timestamp time.Time `json:"timestamp"`
    Type      string    `json:"type"`
    Summary   string    `json:"summary"`
}
```

```go
// internal/sidecar/manager.go
package sidecar

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type Manager struct {
    projectPath string
}

func NewManager(projectPath string) *Manager {
    return &Manager{projectPath: projectPath}
}

func (m *Manager) gcPath() string {
    return filepath.Join(m.projectPath, ".gc")
}

func (m *Manager) EnsureDir() error {
    return os.MkdirAll(m.gcPath(), 0755)
}

func (m *Manager) LoadConfig() (*ProjectConfig, error) {
    data, err := os.ReadFile(filepath.Join(m.gcPath(), "project.json"))
    if err != nil {
        return nil, err
    }
    var cfg ProjectConfig
    return &cfg, json.Unmarshal(data, &cfg)
}

func (m *Manager) SaveConfig(cfg *ProjectConfig) error {
    data, _ := json.MarshalIndent(cfg, "", "  ")
    return os.WriteFile(filepath.Join(m.gcPath(), "project.json"), data, 0644)
}

func (m *Manager) LoadState() (*ProjectState, error) {
    data, err := os.ReadFile(filepath.Join(m.gcPath(), "state.json"))
    if err != nil {
        if os.IsNotExist(err) {
            return &ProjectState{}, nil
        }
        return nil, err
    }
    var state ProjectState
    return &state, json.Unmarshal(data, &state)
}

func (m *Manager) SaveState(state *ProjectState) error {
    data, _ := json.MarshalIndent(state, "", "  ")
    return os.WriteFile(filepath.Join(m.gcPath(), "state.json"), data, 0644)
}
```

**Reuse from GC**: `internal/data/data.go` patterns

---

### P0.2: Global Registry

**Files to create**:
- `internal/registry/registry.go` — Global project registry

```go
// internal/registry/registry.go
package registry

import (
    "encoding/json"
    "os"
    "path/filepath"
    "time"
)

type Registry struct {
    path string
}

type ProjectEntry struct {
    Path       string    `json:"path"`
    Name       string    `json:"name"`
    LastActive time.Time `json:"last_active"`
}

type GlobalConfig struct {
    Projects []ProjectEntry `json:"projects"`
    APIKeys  struct {
        ActiveKeyID  string  `json:"active_key_id"`
        RotationDate string  `json:"rotation_date"`
        LimitType    string  `json:"limit_type"` // daily, weekly, monthly
        LimitUSD     float64 `json:"limit_usd"`
    } `json:"api_keys"`
    Settings struct {
        DefaultAltitude string `json:"default_altitude"`
        TeleportShell   string `json:"teleport_shell"`
    } `json:"settings"`
}

func NewRegistry() (*Registry, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }
    path := filepath.Join(home, ".gc")
    os.MkdirAll(path, 0755)
    return &Registry{path: path}, nil
}

func (r *Registry) Load() (*GlobalConfig, error) {
    data, err := os.ReadFile(filepath.Join(r.path, "global.json"))
    if err != nil {
        if os.IsNotExist(err) {
            return &GlobalConfig{}, nil
        }
        return nil, err
    }
    var cfg GlobalConfig
    return &cfg, json.Unmarshal(data, &cfg)
}

func (r *Registry) Save(cfg *GlobalConfig) error {
    data, _ := json.MarshalIndent(cfg, "", "  ")
    return os.WriteFile(filepath.Join(r.path, "global.json"), data, 0644)
}

func (r *Registry) AddProject(path, name string) error {
    cfg, err := r.Load()
    if err != nil {
        return err
    }

    // Check if already exists
    for i, p := range cfg.Projects {
        if p.Path == path {
            cfg.Projects[i].LastActive = time.Now()
            return r.Save(cfg)
        }
    }

    cfg.Projects = append(cfg.Projects, ProjectEntry{
        Path:       path,
        Name:       name,
        LastActive: time.Now(),
    })
    return r.Save(cfg)
}

func (r *Registry) ListProjects() ([]ProjectEntry, error) {
    cfg, err := r.Load()
    if err != nil {
        return nil, err
    }
    return cfg.Projects, nil
}
```

---

### P0.3: TUI Shell

**Files to modify/create**:
- `internal/tui/flightdeck.go` — New TUI entry point
- `internal/tui/views/hangar.go` — Project list view
- `internal/tui/views/mission.go` — Session view (placeholder)
- `internal/tui/styles/styles.go` — Lip Gloss styles

**Reuse from GC**: `internal/tui/tui.go` scaffolding, Lipgloss patterns

```go
// internal/tui/flightdeck.go
package tui

import (
    "fmt"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "ground-control/internal/registry"
    "ground-control/internal/sidecar"
)

type viewMode int

const (
    viewHangar viewMode = iota
    viewMission
    viewComms
    viewCosts
)

type Model struct {
    mode        viewMode
    registry    *registry.Registry
    projects    []ProjectView
    selectedIdx int
    width       int
    height      int
    err         error
}

type ProjectView struct {
    Entry registry.ProjectEntry
    State *sidecar.ProjectState
}

func NewModel() (*Model, error) {
    reg, err := registry.NewRegistry()
    if err != nil {
        return nil, err
    }
    return &Model{
        mode:     viewHangar,
        registry: reg,
    }, nil
}

func (m Model) Init() tea.Cmd {
    return m.loadProjects
}

func (m *Model) loadProjects() tea.Msg {
    entries, err := m.registry.ListProjects()
    if err != nil {
        return errMsg{err}
    }

    var views []ProjectView
    for _, e := range entries {
        mgr := sidecar.NewManager(e.Path)
        state, _ := mgr.LoadState() // Ignore errors, show as inactive
        views = append(views, ProjectView{Entry: e, State: state})
    }
    return projectsLoadedMsg{views}
}

type projectsLoadedMsg struct{ projects []ProjectView }
type errMsg struct{ err error }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "1":
            m.mode = viewHangar
        case "2":
            m.mode = viewMission
        case "3":
            m.mode = viewComms
        case "4":
            m.mode = viewCosts
        case "j", "down":
            if m.selectedIdx < len(m.projects)-1 {
                m.selectedIdx++
            }
        case "k", "up":
            if m.selectedIdx > 0 {
                m.selectedIdx--
            }
        case "enter":
            if m.mode == viewHangar && len(m.projects) > 0 {
                m.mode = viewMission
            }
        }

    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height

    case projectsLoadedMsg:
        m.projects = msg.projects

    case errMsg:
        m.err = msg.err
    }

    return m, nil
}

func (m Model) View() string {
    // Tab bar
    tabs := m.renderTabs()

    // Main content
    var content string
    switch m.mode {
    case viewHangar:
        content = m.renderHangar()
    case viewMission:
        content = m.renderMission()
    case viewComms:
        content = m.renderComms()
    case viewCosts:
        content = m.renderCosts()
    }

    // Status bar
    status := m.renderStatusBar()

    return lipgloss.JoinVertical(lipgloss.Left, tabs, content, status)
}
```

```go
// internal/tui/styles/styles.go
package styles

import "github.com/charmbracelet/lipgloss"

var (
    // Colors
    Purple    = lipgloss.Color("99")
    Green     = lipgloss.Color("42")
    Yellow    = lipgloss.Color("214")
    Red       = lipgloss.Color("196")
    Gray      = lipgloss.Color("245")
    LightGray = lipgloss.Color("250")
    BgDark    = lipgloss.Color("236")

    // Title
    Title = lipgloss.NewStyle().
        Bold(true).
        Foreground(Purple).
        MarginBottom(1)

    // Active/inactive states
    Active = lipgloss.NewStyle().
        Foreground(Green).
        Bold(true)

    Inactive = lipgloss.NewStyle().
        Foreground(Gray)

    // Box with border
    Box = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(Purple).
        Padding(1, 2)

    // Selected row
    Selected = lipgloss.NewStyle().
        Background(lipgloss.Color("237")).
        Foreground(lipgloss.Color("229")).
        Bold(true)

    // Tab bar
    TabActive = lipgloss.NewStyle().
        Background(Purple).
        Foreground(lipgloss.Color("230")).
        Padding(0, 2).
        Bold(true)

    TabInactive = lipgloss.NewStyle().
        Background(BgDark).
        Foreground(Gray).
        Padding(0, 2)

    // Status bar
    StatusBar = lipgloss.NewStyle().
        Background(BgDark).
        Foreground(LightGray).
        Padding(0, 1).
        Width(100) // Will be overridden

    // Help text
    Help = lipgloss.NewStyle().
        Foreground(Gray).
        Italic(true)
)
```

---

### P0.4: Session Management with tmux-cli

**Files to create**:
- `internal/tmux/manager.go` — tmux-cli wrapper

```go
// internal/tmux/manager.go
package tmux

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"
    "time"
)

type Manager struct {
    projectPath string
    projectName string
    paneID      string
}

func NewManager(projectPath, projectName string) *Manager {
    return &Manager{
        projectPath: projectPath,
        projectName: projectName,
    }
}

// StartSession creates a new Claude session in a tmux pane
func (m *Manager) StartSession() (string, error) {
    // Launch shell first (best practice per tmux-cli docs)
    output, err := exec.Command("tmux-cli", "launch", "zsh").Output()
    if err != nil {
        return "", fmt.Errorf("failed to launch shell: %w", err)
    }
    m.paneID = strings.TrimSpace(string(output))

    // Change to project directory
    if err := m.Send(fmt.Sprintf("cd %s", m.projectPath)); err != nil {
        return "", fmt.Errorf("failed to cd: %w", err)
    }

    // Wait for shell to be ready
    if err := m.WaitIdle(2, 10); err != nil {
        return "", fmt.Errorf("shell not ready: %w", err)
    }

    // Start Claude
    claudeCmd := "claude"
    if err := m.Send(claudeCmd); err != nil {
        return "", fmt.Errorf("failed to start claude: %w", err)
    }

    // Wait for Claude prompt
    if err := m.WaitIdle(3, 30); err != nil {
        return "", fmt.Errorf("claude not ready: %w", err)
    }

    return m.paneID, nil
}

// Send sends text to the pane
func (m *Manager) Send(text string) error {
    cmd := exec.Command("tmux-cli", "send", text, "--pane="+m.paneID)
    return cmd.Run()
}

// SendNoEnter sends text without pressing Enter
func (m *Manager) SendNoEnter(text string) error {
    cmd := exec.Command("tmux-cli", "send", text,
        "--pane="+m.paneID, "--enter=False")
    return cmd.Run()
}

// Capture returns current pane output
func (m *Manager) Capture() (string, error) {
    output, err := exec.Command("tmux-cli", "capture",
        "--pane="+m.paneID).Output()
    if err != nil {
        return "", err
    }
    return string(output), nil
}

// WaitIdle waits for pane to be idle
func (m *Manager) WaitIdle(idleSeconds, timeoutSeconds int) error {
    cmd := exec.Command("tmux-cli", "wait_idle",
        "--pane="+m.paneID,
        fmt.Sprintf("--idle-time=%d.0", idleSeconds),
        fmt.Sprintf("--timeout=%d", timeoutSeconds))
    return cmd.Run()
}

// Interrupt sends Ctrl+C
func (m *Manager) Interrupt() error {
    return exec.Command("tmux-cli", "interrupt", "--pane="+m.paneID).Run()
}

// Stop gracefully stops the session
func (m *Manager) Stop() error {
    // Try sending /exit to Claude
    if err := m.Send("/exit"); err != nil {
        // Fall back to interrupt
        m.Interrupt()
    }
    return nil
}

// Kill forcefully kills the pane
func (m *Manager) Kill() error {
    return exec.Command("tmux-cli", "kill", "--pane="+m.paneID).Run()
}

// ListPanes returns all panes
func ListPanes() ([]PaneInfo, error) {
    output, err := exec.Command("tmux-cli", "list_panes").Output()
    if err != nil {
        return nil, err
    }
    var panes []PaneInfo
    json.Unmarshal(output, &panes)
    return panes, nil
}

type PaneInfo struct {
    ID     string `json:"id"`
    Index  int    `json:"index"`
    Active bool   `json:"active"`
}
```

---

### P0.5: Adopt Command (Prompt-Based)

**Files to create**:
- `internal/cmd/adopt.go` — Adoption command
- `templates/adopt_prompt.md` — Prompt template

```go
// internal/cmd/adopt.go
package cmd

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"

    "github.com/spf13/cobra"
    "ground-control/internal/registry"
    "ground-control/internal/sidecar"
)

func NewAdoptCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "adopt <path>",
        Short: "Adopt an existing project into Flight Deck",
        Args:  cobra.ExactArgs(1),
        RunE:  runAdopt,
    }
}

func runAdopt(cmd *cobra.Command, args []string) error {
    projectPath, err := filepath.Abs(args[0])
    if err != nil {
        return err
    }

    // Check path exists
    if _, err := os.Stat(projectPath); err != nil {
        return fmt.Errorf("path does not exist: %s", projectPath)
    }

    fmt.Printf("Analyzing %s...\n", projectPath)

    // Run Claude to analyze the project
    analysisPath := filepath.Join(projectPath, ".gc", "analysis.json")
    os.MkdirAll(filepath.Join(projectPath, ".gc"), 0755)

    prompt := fmt.Sprintf(`You are part of Flight Deck, an AI development orchestration system.

Analyze this repository and output JSON to %s with this structure:
{
  "name": "project name (from package.json, go.mod, etc.)",
  "languages": [{"name": "TypeScript", "confidence": 0.95}],
  "frameworks": [{"name": "React", "confidence": 0.90}],
  "test_runner": {"name": "Jest", "confidence": 0.85},
  "package_manager": "npm",
  "ci_system": "github-actions",
  "existing_conventions": ["Uses ESLint", "Prettier for formatting"],
  "existing_task_management": "Linear integration detected"
}

Be thorough. Check package.json, go.mod, requirements.txt, Makefile, .github/, etc.
Confidence should be 0.0-1.0 based on how certain you are.`, analysisPath)

    // Run claude --print in the project directory
    claudeCmd := exec.Command("claude", "--print", prompt)
    claudeCmd.Dir = projectPath
    claudeCmd.Stdout = os.Stdout
    claudeCmd.Stderr = os.Stderr

    if err := claudeCmd.Run(); err != nil {
        return fmt.Errorf("analysis failed: %w", err)
    }

    // Read analysis
    analysisData, err := os.ReadFile(analysisPath)
    if err != nil {
        return fmt.Errorf("could not read analysis: %w", err)
    }

    var analysis struct {
        Name                   string              `json:"name"`
        Languages              []sidecar.Detection `json:"languages"`
        Frameworks             []sidecar.Detection `json:"frameworks"`
        TestRunner             *sidecar.Detection  `json:"test_runner"`
        PackageManager         string              `json:"package_manager"`
        CISystem               string              `json:"ci_system"`
        ExistingConventions    []string            `json:"existing_conventions"`
        ExistingTaskManagement string              `json:"existing_task_management"`
    }

    if err := json.Unmarshal(analysisData, &analysis); err != nil {
        return fmt.Errorf("invalid analysis JSON: %w", err)
    }

    // Check confidence levels
    lowConfidence := false
    for _, lang := range analysis.Languages {
        if lang.Confidence < 0.7 {
            fmt.Printf("⚠️  Low confidence: %s (%.0f%%)\n", lang.Name, lang.Confidence*100)
            lowConfidence = true
        }
    }
    // ... similar checks for frameworks, test_runner

    if lowConfidence {
        fmt.Print("Some detections have low confidence. Continue? [y/N] ")
        var response string
        fmt.Scanln(&response)
        if response != "y" && response != "Y" {
            return fmt.Errorf("adoption cancelled")
        }
    }

    // Create project config
    mgr := sidecar.NewManager(projectPath)
    cfg := &sidecar.ProjectConfig{
        Name:     analysis.Name,
        Altitude: "mid", // Default
    }
    cfg.TechStack.Languages = analysis.Languages
    cfg.TechStack.Frameworks = analysis.Frameworks
    cfg.TechStack.TestRunner = analysis.TestRunner
    cfg.TechStack.PackageManager = analysis.PackageManager
    cfg.TechStack.CISystem = analysis.CISystem

    // Default approvals
    cfg.Approvals.DestructiveCommands = true
    cfg.Approvals.GitPush = true

    if err := mgr.SaveConfig(cfg); err != nil {
        return err
    }

    // Generate CLAUDE.md
    claudeMd := fmt.Sprintf(`# %s - Flight Deck Project

## Tech Stack
%s

## Conventions
%s

## Constraints
- Follow existing code patterns
- Run tests before committing
`, analysis.Name,
        formatTechStack(analysis),
        formatConventions(analysis.ExistingConventions))

    os.WriteFile(filepath.Join(projectPath, ".gc", "CLAUDE.md"), []byte(claudeMd), 0644)

    // Add to global registry
    reg, err := registry.NewRegistry()
    if err != nil {
        return err
    }
    if err := reg.AddProject(projectPath, analysis.Name); err != nil {
        return err
    }

    fmt.Printf("\n✓ Adopted %s into Flight Deck\n", analysis.Name)
    fmt.Printf("  Path: %s\n", projectPath)
    fmt.Printf("  Sidecar: %s/.gc/\n", projectPath)

    return nil
}

func formatTechStack(a interface{}) string {
    // Format tech stack as markdown list
    return "- TypeScript, React Native\n- Jest for testing"
}

func formatConventions(conventions []string) string {
    result := ""
    for _, c := range conventions {
        result += "- " + c + "\n"
    }
    return result
}
```

---

### P0.6: Integration & Testing

**Deliverables**:
- `gc tui` launches Flight Deck
- `gc adopt <path>` adopts a project
- Hangar view shows adopted projects
- Can navigate to Mission view (placeholder)

**Test scenario**:
```bash
# Adopt a project
gc adopt ~/Projects/notifai

# Launch TUI
gc tui

# See notifai in Hangar
# Press Enter to go to Mission view
```

---

## Phase 1: Flight Recorder

**Goal**: Track Claude sessions, costs, and activity in real-time.

### P1.1: File Watching

**Files to create**:
- `internal/watch/watcher.go` — fsnotify wrapper

```go
// internal/watch/watcher.go
package watch

import (
    "encoding/json"
    "os"
    "path/filepath"

    "github.com/fsnotify/fsnotify"
    tea "github.com/charmbracelet/bubbletea"
    "ground-control/internal/sidecar"
)

type StateUpdateMsg struct {
    ProjectPath string
    State       *sidecar.ProjectState
}

type WatcherErrorMsg struct {
    Err error
}

// WatchSidecar returns a Bubble Tea command that watches a sidecar
func WatchSidecar(projectPath string) tea.Cmd {
    return func() tea.Msg {
        watcher, err := fsnotify.NewWatcher()
        if err != nil {
            return WatcherErrorMsg{err}
        }

        statePath := filepath.Join(projectPath, ".gc", "state.json")

        // Ensure the file exists
        if _, err := os.Stat(statePath); os.IsNotExist(err) {
            // Create empty state
            os.WriteFile(statePath, []byte("{}"), 0644)
        }

        if err := watcher.Add(statePath); err != nil {
            return WatcherErrorMsg{err}
        }

        // Return initial state
        state := readState(statePath)
        return StateUpdateMsg{ProjectPath: projectPath, State: state}
    }
}

// WatchLoop continuously watches for changes
func WatchLoop(projectPath string, updates chan<- StateUpdateMsg) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return
    }
    defer watcher.Close()

    statePath := filepath.Join(projectPath, ".gc", "state.json")
    watcher.Add(statePath)

    for {
        select {
        case event, ok := <-watcher.Events:
            if !ok {
                return
            }
            if event.Has(fsnotify.Write) {
                state := readState(statePath)
                updates <- StateUpdateMsg{
                    ProjectPath: projectPath,
                    State:       state,
                }
            }
        case _, ok := <-watcher.Errors:
            if !ok {
                return
            }
        }
    }
}

func readState(path string) *sidecar.ProjectState {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil
    }
    var state sidecar.ProjectState
    json.Unmarshal(data, &state)
    return &state
}
```

### P1.2: Cost Tracking

**Files to create**:
- `internal/cost/tracker.go` — Cost tracking
- `internal/cost/aggregator.go` — Aggregation logic

```go
// internal/cost/tracker.go
package cost

import (
    "encoding/json"
    "os"
    "path/filepath"
    "time"
)

type Entry struct {
    Timestamp time.Time `json:"timestamp"`
    Project   string    `json:"project"`
    SessionID string    `json:"session_id"`
    Tokens    int       `json:"tokens"`
    CostUSD   float64   `json:"cost_usd"`
    Model     string    `json:"model"`
}

type Tracker struct {
    gcPath string
}

func NewTracker() (*Tracker, error) {
    home, _ := os.UserHomeDir()
    gcPath := filepath.Join(home, ".gc")
    os.MkdirAll(gcPath, 0755)
    return &Tracker{gcPath: gcPath}, nil
}

func (t *Tracker) Record(entry Entry) error {
    entries, _ := t.loadEntries()
    entries = append(entries, entry)

    // Keep only last 30 days
    cutoff := time.Now().AddDate(0, 0, -30)
    var filtered []Entry
    for _, e := range entries {
        if e.Timestamp.After(cutoff) {
            filtered = append(filtered, e)
        }
    }

    return t.saveEntries(filtered)
}

func (t *Tracker) GetTotals() (today, week, month float64) {
    entries, _ := t.loadEntries()

    now := time.Now()
    todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    weekStart := todayStart.AddDate(0, 0, -int(now.Weekday()))
    monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

    for _, e := range entries {
        if e.Timestamp.After(monthStart) {
            month += e.CostUSD
            if e.Timestamp.After(weekStart) {
                week += e.CostUSD
                if e.Timestamp.After(todayStart) {
                    today += e.CostUSD
                }
            }
        }
    }
    return
}

func (t *Tracker) GetByProject() map[string]float64 {
    entries, _ := t.loadEntries()
    weekStart := time.Now().AddDate(0, 0, -7)

    result := make(map[string]float64)
    for _, e := range entries {
        if e.Timestamp.After(weekStart) {
            result[e.Project] += e.CostUSD
        }
    }
    return result
}

func (t *Tracker) loadEntries() ([]Entry, error) {
    data, err := os.ReadFile(filepath.Join(t.gcPath, "cost_log.json"))
    if err != nil {
        return nil, err
    }
    var entries []Entry
    json.Unmarshal(data, &entries)
    return entries, nil
}

func (t *Tracker) saveEntries(entries []Entry) error {
    data, _ := json.MarshalIndent(entries, "", "  ")
    return os.WriteFile(filepath.Join(t.gcPath, "cost_log.json"), data, 0644)
}
```

### P1.3: Session History

**Files to create**:
- `internal/session/history.go` — Session records

```go
// internal/session/history.go
package session

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"
)

type Record struct {
    ID              string    `json:"id"`
    StartedAt       time.Time `json:"started_at"`
    EndedAt         time.Time `json:"ended_at"`
    DurationMinutes int       `json:"duration_minutes"`

    Focus   string `json:"focus"`
    Outcome string `json:"outcome"` // completed, paused, stopped, errored
    Summary string `json:"summary"`

    TokensUsed int     `json:"tokens_used"`
    CostUSD    float64 `json:"cost_usd"`

    FilesCreated  []string `json:"files_created"`
    FilesModified []string `json:"files_modified"`
    Commits       []string `json:"commits"`

    HandoffContext string `json:"handoff_context"`
}

type History struct {
    projectPath string
}

func NewHistory(projectPath string) *History {
    return &History{projectPath: projectPath}
}

func (h *History) sessionsDir() string {
    return filepath.Join(h.projectPath, ".gc", "sessions")
}

func (h *History) Save(record *Record) error {
    os.MkdirAll(h.sessionsDir(), 0755)

    filename := fmt.Sprintf("%s.json", record.ID)
    data, _ := json.MarshalIndent(record, "", "  ")
    return os.WriteFile(filepath.Join(h.sessionsDir(), filename), data, 0644)
}

func (h *History) List() ([]*Record, error) {
    entries, err := os.ReadDir(h.sessionsDir())
    if err != nil {
        return nil, err
    }

    var records []*Record
    for _, e := range entries {
        if filepath.Ext(e.Name()) != ".json" {
            continue
        }
        data, err := os.ReadFile(filepath.Join(h.sessionsDir(), e.Name()))
        if err != nil {
            continue
        }
        var r Record
        if json.Unmarshal(data, &r) == nil {
            records = append(records, &r)
        }
    }
    return records, nil
}

func (h *History) GetLatest() (*Record, error) {
    records, err := h.List()
    if err != nil || len(records) == 0 {
        return nil, err
    }

    // Sort by started_at desc
    latest := records[0]
    for _, r := range records[1:] {
        if r.StartedAt.After(latest.StartedAt) {
            latest = r
        }
    }
    return latest, nil
}
```

### P1.4: TUI Updates

**Update Mission view to show real-time data**:

```go
// internal/tui/views/mission.go
package views

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "ground-control/internal/sidecar"
    "ground-control/internal/tui/styles"
)

func RenderMission(state *sidecar.ProjectState, width, height int) string {
    if state == nil || state.Session.ID == "" {
        return renderNoSession()
    }

    // Session output box
    outputBox := styles.Box.
        Width(width - 4).
        Height(height - 12).
        Render(renderSessionOutput(state))

    // Status bar
    statusContent := fmt.Sprintf(
        "Task: %s        Altitude: Mid\nTokens: %dk this session         Cost: $%.2f",
        truncate(state.Session.CurrentFocus, 30),
        state.Costs.SessionTokens/1000,
        state.Costs.SessionCostUSD,
    )
    statusBox := styles.Box.
        Width(width - 4).
        Render(statusContent)

    // Help line
    help := styles.Help.Render(
        "[space] Pause  [t] Teleport  [m] Message  [x] Stop")

    return lipgloss.JoinVertical(lipgloss.Left,
        outputBox,
        statusBox,
        help,
    )
}

func renderSessionOutput(state *sidecar.ProjectState) string {
    var lines []string

    // Show recent activity
    for _, a := range state.RecentActivity {
        icon := "○"
        switch a.Type {
        case "file_write":
            icon = "✏️"
        case "file_read":
            icon = "📖"
        case "command":
            icon = "⚡"
        }
        lines = append(lines, fmt.Sprintf("%s %s", icon, a.Summary))
    }

    // Show current focus
    if state.Session.CurrentFocus != "" {
        lines = append(lines, "")
        lines = append(lines, styles.Active.Render("▶ "+state.Session.CurrentFocus))
    }

    return strings.Join(lines, "\n")
}

func renderNoSession() string {
    return styles.Box.Render(
        "No active session\n\nPress [s] to start a session")
}

func truncate(s string, max int) string {
    if len(s) <= max {
        return s
    }
    return s[:max-3] + "..."
}
```

---

## Phase 2: Teleportation

**Goal**: Seamless attachment to sessions, approval flow, messaging.

### P2.1: Teleport Command

```go
// Teleport suspends TUI and attaches to tmux pane
func (m *Model) teleport() tea.Cmd {
    if m.activeSession == nil {
        return nil
    }

    // Use tea.ExecProcess to suspend TUI and run tmux select-pane
    return tea.ExecProcess(
        exec.Command("tmux", "select-pane", "-t", m.activeSession.TmuxPane),
        func(err error) tea.Msg {
            // TUI resumes when user detaches
            return teleportReturnMsg{}
        },
    )
}
```

### P2.2: Approval Flow

```go
// Handle approval in Update
case tea.KeyMsg:
    if m.mode == viewComms && m.pendingApproval != nil {
        switch msg.String() {
        case "y":
            return m, m.approveRequest(true)
        case "n":
            return m, m.approveRequest(false)
        case "e":
            // Open editor modal
            m.showEditModal = true
        }
    }

func (m *Model) approveRequest(approved bool) tea.Cmd {
    return func() tea.Msg {
        // Write approval to state file
        mgr := sidecar.NewManager(m.selectedProject().Path)
        state, _ := mgr.LoadState()

        if approved {
            // Clear pending, Claude will see this
            state.PendingApproval = nil
        } else {
            // Mark as denied
            state.PendingApproval.Type = "denied"
        }

        mgr.SaveState(state)
        return approvalSentMsg{approved: approved}
    }
}
```

### P2.3: Send Message

```go
func (m *Model) sendMessage(msg string) tea.Cmd {
    return func() tea.Msg {
        tm := tmux.NewManager(
            m.selectedProject().Path,
            m.selectedProject().Name,
        )
        tm.SetPane(m.activeSession.TmuxPane)

        if err := tm.Send(msg); err != nil {
            return errMsg{err}
        }
        return messageSentMsg{}
    }
}
```

---

## Phase 3: Engagement Altitudes

**Goal**: Support Low/Mid/High automation modes.

### P3.1: Altitude Configuration

Add to project config:
- `altitude: "low" | "mid" | "high"`

### P3.2: Altitude-Specific Behavior

| Altitude | Session Start | Approvals | Monitoring |
|----------|--------------|-----------|------------|
| Low | Manual | All | Passive |
| Mid | On request | Destructive only | Active |
| High | Automatic | None | Alert only |

### P3.3: TUI Altitude Selector

Add altitude dropdown/toggle to project settings.

---

## Post-MVP Features

### Web UI
- Remote dashboard via Cloudflare Tunnel
- Mobile-friendly approval interface

### Protocol Droid
- Security monitoring agent
- TDD enforcement (Red-Green-Refactor)
- Behavior detection

### Multi-Agent
- Multiple Claude sessions coordinating
- Cross-project awareness

---

## Migration from GC v1

1. **Backup**: `cp -r data/ data-backup/`
2. **Export tasks**: `gc tasks --json > tasks-export.json`
3. **Manual import**: Review tasks, create as needed in new system
4. **Archive GC v1**: Move to `gc-v1/` directory

---

## File Structure After Implementation

```
ground-control/
├── cmd/gc/main.go
├── internal/
│   ├── cmd/
│   │   ├── adopt.go        # gc adopt
│   │   ├── tui.go          # gc tui (default)
│   │   └── root.go
│   ├── tui/
│   │   ├── flightdeck.go   # Main TUI model
│   │   ├── views/
│   │   │   ├── hangar.go
│   │   │   ├── mission.go
│   │   │   ├── comms.go
│   │   │   └── costs.go
│   │   └── styles/
│   │       └── styles.go
│   ├── sidecar/
│   │   ├── types.go
│   │   └── manager.go
│   ├── registry/
│   │   └── registry.go
│   ├── tmux/
│   │   └── manager.go
│   ├── watch/
│   │   └── watcher.go
│   ├── cost/
│   │   ├── tracker.go
│   │   └── aggregator.go
│   └── session/
│       └── history.go
├── templates/
│   └── adopt_prompt.md
└── docs/
    ├── flight-deck-design-doc.md
    └── flight-deck-project-plan.md
```

---

## Success Criteria

### Phase 0
- [ ] Can adopt a project via `gc adopt`
- [ ] TUI launches with `gc tui`
- [ ] Hangar shows adopted projects
- [ ] Can start a Claude session from TUI
- [ ] Mission view shows session exists

### Phase 1
- [ ] Mission view updates in real-time
- [ ] Costs tracked per session/project
- [ ] Session history saved on end

### Phase 2
- [ ] Can teleport into session
- [ ] Approvals surface in Comms
- [ ] Can send messages to Claude

### Phase 3
- [ ] Altitude modes work
- [ ] High altitude runs autonomously
- [ ] Low altitude is manual only
