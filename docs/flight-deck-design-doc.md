# Flight Deck Design Document

> Ground Control v2 — A TUI-first orchestration layer for AI-assisted development

**Status**: Draft
**Created**: 2026-03-10
**Author**: Matt + Claude

---

## 1. Executive Summary

Flight Deck transforms Ground Control from a task-runner CLI into a **TUI-first orchestration dashboard** that manages persistent Claude sessions across multiple projects. The core insight: developers don't trust headless AI agents they can't see. Flight Deck solves this by making AI work **visible, interruptible, and contextual**.

### Core Principles

1. **TUI-first** — The dashboard is the primary interface, not CLI commands
2. **Project-centric** — State lives in project repos (`.gc/`), not a central database
3. **Visible work** — Always see what Claude is doing; teleport into any session
4. **No daemons** — On-demand operations, no background polling
5. **Engagement altitudes** — Choose your automation level per project

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      FLIGHT DECK TUI                            │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐               │
│  │ Hangar  │ │ Mission │ │  Comms  │ │  Costs  │               │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘               │
└─────────────────────────────────────────────────────────────────┘
        │                     │                    │
        │ reads               │ watches            │ aggregates
        ▼                     ▼                    ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│ ~/.gc/          │   │ project/.gc/    │   │ project/.gc/    │
│ projects.json   │   │ state.json      │   │ sessions/       │
│ global.json     │   │ project.json    │   │ *.json          │
└─────────────────┘   └─────────────────┘   └─────────────────┘
                              │
                              │ written by
                              ▼
                      ┌─────────────────┐
                      │  Claude Code    │
                      │  (tmux pane)    │
                      └─────────────────┘
                              │
                              │ managed via
                              ▼
                      ┌─────────────────┐
                      │    tmux-cli     │
                      └─────────────────┘
```

### Key Flows

1. **Launch TUI** → Reads `~/.gc/projects.json` → Shows Hangar
2. **Select project** → Reads `.gc/state.json` → Shows Mission view
3. **Start session** → `tmux-cli launch` → Claude writes to `.gc/state.json`
4. **Monitor** → TUI watches `.gc/state.json` via fsnotify
5. **Teleport** → TUI suspends → attaches to tmux pane → TUI resumes on exit

---

## 3. TUI Screens

### 3.1 Hangar (Project List)

Default view. Shows all adopted projects and their status.

```
┌─ HANGAR ────────────────────────────────────────────────────────┐
│                                                                 │
│  PROJECT            STATUS      SESSION     LAST ACTIVE         │
│  ─────────────────────────────────────────────────────────────  │
│  ● notifai          Building    Active      now                 │
│  ○ ground-control   Paused      —           2h ago              │
│  ○ tone-tool        Idle        —           3d ago              │
│  ○ m4b-tool         Idle        —           1w ago              │
│                                                                 │
│  [a] Adopt project  [enter] Select  [n] New project             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Data source**: `~/.gc/projects.json` + each project's `.gc/state.json`

**Actions**:
- `enter` — Open Mission view for selected project
- `a` — Adopt existing project (prompt-based analysis)
- `n` — Create new project from template
- `d` — View project details/settings
- `/` — Filter/search projects

### 3.2 Mission (Active Session)

The main working view. Shows Claude's activity in real-time.

```
┌─ MISSION: notifai ──────────────────────────────────────────────┐
│ ┌─ SESSION OUTPUT ─────────────────────────────────────────────┐│
│ │ Claude: I'll implement the notification carousel component.  ││
│ │ Reading src/components/NotificationList.tsx...               ││
│ │ Found existing list implementation. I'll extend it with      ││
│ │ swipe gestures using react-native-gesture-handler.           ││
│ │                                                              ││
│ │ Writing src/components/NotificationCarousel.tsx...           ││
│ │ ████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ ││
│ └──────────────────────────────────────────────────────────────┘│
│ ┌─ STATUS ─────────────────────────────────────────────────────┐│
│ │ Task: Implement carousel UI        Altitude: Mid             ││
│ │ Tokens: 12.4k this session         Cost: $0.37               ││
│ │ Files touched: 3                   Tests: —                  ││
│ └──────────────────────────────────────────────────────────────┘│
│                                                                 │
│ [space] Pause  [t] Teleport (attach)  [m] Message  [x] Stop     │
└─────────────────────────────────────────────────────────────────┘
```

**Data source**: `.gc/state.json` (watched via fsnotify) + tmux pane capture

**Actions**:
- `space` — Pause/resume session
- `t` — Teleport: attach terminal to tmux pane (exit TUI temporarily)
- `m` — Send message to Claude (input modal)
- `x` — Stop session gracefully
- `!` — Emergency stop (kill process)
- `h` — View session history

### 3.3 Comms (Approvals & Messages)

Approval requests and cross-session communication.

```
┌─ COMMS ─────────────────────────────────────────────────────────┐
│                                                                 │
│  ⚠️  APPROVAL REQUESTED                          notifai  12s ago│
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ Claude wants to run:                                        ││
│  │ > rm -rf node_modules && npm install                        ││
│  │                                                             ││
│  │ Reason: Package lock conflict after merge                   ││
│  │                                                             ││
│  │ [y] Approve   [n] Deny   [e] Edit command   [?] Ask why     ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│  RECENT                                                         │
│  ─────────────────────────────────────────────────────────────  │
│  ✓ Approved: git push origin feature/carousel     2m ago        │
│  ✓ Approved: npm install gesture-handler          5m ago        │
│  ○ Info: Session started                         12m ago        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Data source**: `.gc/state.json` `pending_approval` field

**Actions**:
- `y/n` — Approve/deny
- `e` — Edit command before approving
- `?` — Ask Claude to explain

### 3.4 Costs

Usage tracking and limits.

```
┌─ COSTS ─────────────────────────────────────────────────────────┐
│                                                                 │
│  TODAY         $2.14          ████████░░░░░░░░░░░  $10 limit    │
│  THIS WEEK     $8.73          ████████████░░░░░░░  $50 limit    │
│  THIS MONTH    $31.20         ██████░░░░░░░░░░░░░  $200 limit   │
│                                                                 │
│  BY PROJECT (this week)                                         │
│  ─────────────────────────────────────────────────────────────  │
│  notifai              $5.82   ██████████████████░░              │
│  ground-control       $2.41   ███████░░░░░░░░░░░░░              │
│  tone-tool            $0.50   █░░░░░░░░░░░░░░░░░░░              │
│                                                                 │
│  [k] Manage API keys   [l] Set limits   [e] Export CSV          │
└─────────────────────────────────────────────────────────────────┘
```

**Data source**: Aggregated from all project `.gc/sessions/*.json`

---

## 4. Data Schemas

### 4.1 Global Registry (`~/.gc/projects.json`)

```json
{
  "projects": [
    {
      "path": "/home/user/Projects/notifai",
      "name": "notifai",
      "last_active": "2026-03-10T14:30:00Z"
    }
  ]
}
```

### 4.2 Global Config (`~/.gc/global.json`)

```json
{
  "api_keys": {
    "active_key_id": "sk-...abc",
    "rotation_date": "2026-03-01",
    "limit_type": "daily",
    "limit_usd": 10.00
  },
  "costs": {
    "today_usd": 2.14,
    "week_usd": 8.73,
    "month_usd": 31.20,
    "last_updated": "2026-03-10T14:30:00Z"
  },
  "settings": {
    "default_altitude": "mid",
    "teleport_shell": "zsh"
  }
}
```

### 4.3 Project Config (`.gc/project.json`)

```json
{
  "name": "notifai",
  "adopted_at": "2026-03-01T10:00:00Z",

  "tech_stack": {
    "languages": [
      {"name": "TypeScript", "confidence": 0.98},
      {"name": "Python", "confidence": 0.85}
    ],
    "frameworks": [
      {"name": "React Native", "confidence": 0.95},
      {"name": "FastAPI", "confidence": 0.90}
    ],
    "test_runner": {"name": "Jest", "confidence": 0.90},
    "package_manager": "npm",
    "ci_system": "github-actions"
  },

  "altitude": "mid",
  "ci_enabled": false,

  "approvals": {
    "destructive_commands": true,
    "network_access": false,
    "install_packages": true,
    "git_push": true
  },

  "constraints": [
    "Never modify package-lock.json directly",
    "Always run tests before committing"
  ]
}
```

### 4.4 Project State (`.gc/state.json`)

```json
{
  "session": {
    "id": "session_003",
    "tmux_pane": "fd-notifai:0.1",
    "status": "active",
    "started_at": "2026-03-10T14:00:00Z",
    "last_activity": "2026-03-10T14:30:00Z",
    "current_focus": "Implementing carousel swipe gestures",
    "files_touched": [
      "src/components/NotificationCarousel.tsx",
      "src/hooks/useSwipeGesture.ts"
    ]
  },

  "costs": {
    "session_tokens": 12400,
    "session_cost_usd": 0.37,
    "today_tokens": 45000,
    "today_cost_usd": 1.35
  },

  "pending_approval": null,

  "recent_activity": [
    {
      "timestamp": "2026-03-10T14:30:00Z",
      "type": "file_write",
      "summary": "Created NotificationCarousel.tsx"
    }
  ],

  "context": {
    "key_files": [
      "src/components/NotificationList.tsx",
      "src/types/notification.ts"
    ],
    "established_patterns": [
      "Using React Query for data fetching",
      "Gesture handler for swipe interactions"
    ]
  }
}
```

### 4.5 Session Record (`.gc/sessions/session_003.json`)

```json
{
  "id": "session_003",
  "started_at": "2026-03-10T14:00:00Z",
  "ended_at": "2026-03-10T16:30:00Z",
  "duration_minutes": 150,

  "focus": "Implement notification carousel UI",
  "outcome": "completed",
  "summary": "Created carousel component with swipe gestures, integrated with existing notification list",

  "tokens_used": 45000,
  "cost_usd": 1.35,

  "files_created": ["src/components/NotificationCarousel.tsx"],
  "files_modified": ["src/components/NotificationList.tsx"],
  "commits": ["abc123f"],

  "handoff_context": "Carousel complete. Next: add pull-to-refresh and infinite scroll."
}
```

---

## 5. Technical Implementation

### 5.1 TUI Framework (Bubble Tea)

Using Charmbracelet's Bubble Tea with the Model-View-Update pattern.

```go
package main

import (
    "fmt"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// View modes
type viewMode int

const (
    viewHangar viewMode = iota
    viewMission
    viewComms
    viewCosts
)

// Main model
type model struct {
    mode           viewMode
    projects       []Project
    selectedIdx    int
    activeSession  *Session
    pendingApproval *Approval
    width, height  int
}

func (m model) Init() tea.Cmd {
    return tea.Batch(
        loadProjects,
        watchSidecars,
    )
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "1":
            m.mode = viewHangar
        case "2":
            m.mode = viewMission
        case "3":
            m.mode = viewComms
        case "4":
            m.mode = viewCosts
        case "q", "ctrl+c":
            return m, tea.Quit
        case "t":
            if m.mode == viewMission && m.activeSession != nil {
                return m, teleportToSession(m.activeSession.TmuxPane)
            }
        }

    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height

    case sidecarUpdateMsg:
        // Update state from file change
        m.activeSession = msg.session
        m.pendingApproval = msg.approval
    }

    return m, nil
}

func (m model) View() string {
    switch m.mode {
    case viewHangar:
        return m.renderHangar()
    case viewMission:
        return m.renderMission()
    case viewComms:
        return m.renderComms()
    case viewCosts:
        return m.renderCosts()
    }
    return ""
}
```

### 5.2 Styling with Lip Gloss

```go
package ui

import "github.com/charmbracelet/lipgloss"

var (
    // Color palette
    purple    = lipgloss.Color("99")
    green     = lipgloss.Color("42")
    yellow    = lipgloss.Color("214")
    red       = lipgloss.Color("196")
    gray      = lipgloss.Color("245")
    lightGray = lipgloss.Color("250")

    // Component styles
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(purple).
        MarginBottom(1)

    activeStyle = lipgloss.NewStyle().
        Foreground(green).
        Bold(true)

    inactiveStyle = lipgloss.NewStyle().
        Foreground(gray)

    boxStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(purple).
        Padding(1, 2)

    statusBarStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("236")).
        Foreground(lightGray).
        Padding(0, 1)

    // Table styles for project list
    headerStyle = lipgloss.NewStyle().
        Foreground(purple).
        Bold(true).
        Align(lipgloss.Center)

    selectedRowStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("237")).
        Foreground(lipgloss.Color("229"))
)

// Render project table
func renderProjectTable(projects []Project, selected int) string {
    t := table.New().
        Border(lipgloss.RoundedBorder()).
        BorderStyle(lipgloss.NewStyle().Foreground(purple)).
        StyleFunc(func(row, col int) lipgloss.Style {
            if row == table.HeaderRow {
                return headerStyle
            }
            if row == selected {
                return selectedRowStyle
            }
            return lipgloss.NewStyle().Padding(0, 1)
        }).
        Headers("PROJECT", "STATUS", "SESSION", "LAST ACTIVE")

    for _, p := range projects {
        t.Row(p.Name, p.Status, p.SessionStatus, p.LastActive)
    }

    return t.String()
}
```

### 5.3 File Watching with fsnotify

```go
package watch

import (
    "encoding/json"
    "log"
    "os"

    "github.com/fsnotify/fsnotify"
    tea "github.com/charmbracelet/bubbletea"
)

type sidecarUpdateMsg struct {
    session  *Session
    approval *Approval
}

func watchSidecar(projectPath string) tea.Cmd {
    return func() tea.Msg {
        watcher, err := fsnotify.NewWatcher()
        if err != nil {
            log.Printf("watch error: %v", err)
            return nil
        }

        statePath := filepath.Join(projectPath, ".gc", "state.json")
        err = watcher.Add(statePath)
        if err != nil {
            log.Printf("watch add error: %v", err)
            return nil
        }

        for {
            select {
            case event, ok := <-watcher.Events:
                if !ok {
                    return nil
                }
                if event.Has(fsnotify.Write) {
                    // Read updated state
                    data, err := os.ReadFile(statePath)
                    if err != nil {
                        continue
                    }

                    var state ProjectState
                    if err := json.Unmarshal(data, &state); err != nil {
                        continue
                    }

                    return sidecarUpdateMsg{
                        session:  state.Session,
                        approval: state.PendingApproval,
                    }
                }
            case err, ok := <-watcher.Errors:
                if !ok {
                    return nil
                }
                log.Printf("watcher error: %v", err)
            }
        }
    }
}
```

### 5.4 Session Management with tmux-cli

```go
package session

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"
)

type TmuxManager struct {
    projectPath string
    paneName    string
}

// StartSession launches a new Claude session in a tmux pane
func (tm *TmuxManager) StartSession() (string, error) {
    // Always launch shell first (per tmux-cli best practices)
    cmd := exec.Command("tmux-cli", "launch", "zsh")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("launch failed: %w", err)
    }

    paneID := strings.TrimSpace(string(output))
    tm.paneName = paneID

    // Change to project directory
    if err := tm.send(fmt.Sprintf("cd %s", tm.projectPath)); err != nil {
        return "", err
    }

    // Start Claude with project context
    claudeCmd := fmt.Sprintf("claude --context .gc/CLAUDE.md")
    if err := tm.send(claudeCmd); err != nil {
        return "", err
    }

    // Wait for Claude to be ready
    if err := tm.waitIdle(); err != nil {
        return "", err
    }

    return paneID, nil
}

// Send input to the Claude session
func (tm *TmuxManager) send(text string) error {
    cmd := exec.Command("tmux-cli", "send", text, "--pane="+tm.paneName)
    return cmd.Run()
}

// Capture current output
func (tm *TmuxManager) Capture() (string, error) {
    cmd := exec.Command("tmux-cli", "capture", "--pane="+tm.paneName)
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return string(output), nil
}

// Wait for session to be idle
func (tm *TmuxManager) waitIdle() error {
    cmd := exec.Command("tmux-cli", "wait_idle",
        "--pane="+tm.paneName,
        "--idle-time=2.0",
        "--timeout=30")
    return cmd.Run()
}

// Stop the session gracefully
func (tm *TmuxManager) Stop() error {
    // Send /exit to Claude
    if err := tm.send("/exit"); err != nil {
        // If that fails, send Ctrl+C
        exec.Command("tmux-cli", "interrupt", "--pane="+tm.paneName).Run()
    }
    return nil
}

// Teleport user into the session
func (tm *TmuxManager) Teleport() tea.Cmd {
    return tea.ExecProcess(
        exec.Command("tmux", "select-pane", "-t", tm.paneName),
        nil,
    )
}
```

### 5.5 Cost Tracking

```go
package cost

import (
    "encoding/json"
    "os"
    "path/filepath"
    "time"
)

type CostTracker struct {
    globalPath string
}

type CostEntry struct {
    Timestamp time.Time `json:"timestamp"`
    Tokens    int       `json:"tokens"`
    CostUSD   float64   `json:"cost_usd"`
    SessionID string    `json:"session_id"`
    Project   string    `json:"project"`
}

// RecordUsage logs a cost entry
func (ct *CostTracker) RecordUsage(entry CostEntry) error {
    // Load existing
    var entries []CostEntry
    data, err := os.ReadFile(ct.costLogPath())
    if err == nil {
        json.Unmarshal(data, &entries)
    }

    entries = append(entries, entry)

    // Save
    data, _ = json.MarshalIndent(entries, "", "  ")
    return os.WriteFile(ct.costLogPath(), data, 0644)
}

// GetTotals returns aggregated costs
func (ct *CostTracker) GetTotals() (today, week, month float64) {
    var entries []CostEntry
    data, err := os.ReadFile(ct.costLogPath())
    if err != nil {
        return 0, 0, 0
    }
    json.Unmarshal(data, &entries)

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

    return today, week, month
}

func (ct *CostTracker) costLogPath() string {
    return filepath.Join(ct.globalPath, "cost_log.json")
}
```

---

## 6. Engagement Altitudes

| Altitude | Description | What TUI Does | What Claude Does |
|----------|-------------|---------------|------------------|
| **Low** | Interactive | Tracks time/cost, provides teleport | You drive manually |
| **Mid** | Collaborative | Shows plans, handles approvals | Proposes, waits for approval |
| **High** | Autonomous | Monitors, alerts on issues | Executes with minimal interruption |

Altitude is set per-project in `.gc/project.json`.

---

## 7. Adoption Flow

When running `gc adopt <path>` or pressing `a` in Hangar:

1. **Prompt-based analysis**: Claude analyzes the repo
2. **Confidence check**: If any detection < 70%, prompt user to confirm
3. **Scaffold**: Create `.gc/` directory with config files
4. **Generate CLAUDE.md**: Project-specific Claude instructions
5. **Register**: Add to `~/.gc/projects.json`

```bash
# The adopt process uses a prompt template:
claude --print "You are part of Flight Deck. Analyze this repository:
- Identify tech stack (languages, frameworks, test runners)
- Find existing conventions (linting, formatting, git hooks)
- Detect CI/CD setup
- Note any existing task management

Output JSON to .gc/analysis.json using this schema: {...}"
```

---

## 8. Reusable Code from Ground Control

| Component | Action | Files |
|-----------|--------|-------|
| TUI scaffolding | ADAPT | `internal/tui/tui.go` |
| Session manager | ADAPT | `internal/session/session.go` |
| Data store | REUSE | `internal/data/data.go` |
| Stage interface | REUSE | `internal/pipeline/stage.go` |
| Types | ADAPT | `internal/types/types.go` |
| Context builder | REUSE | `internal/context/context.go` |
| Verification | ADAPT | `internal/verify/verify.go` |

**Estimated reuse: 30-40%**

---

## 9. Security Model

### Approval Requirements (configurable per-project)

- **Destructive commands**: `rm`, `git reset --hard`, etc.
- **Network access**: `curl`, `wget`, external API calls
- **Package installation**: `npm install`, `pip install`
- **Git push**: Pushing to remote repositories

### Implementation

Approvals surface in the Comms screen. User can:
- Approve (`y`)
- Deny (`n`)
- Edit command (`e`)
- Ask for explanation (`?`)

---

## 10. CLI Commands (Secondary Interface)

While TUI is primary, CLI commands exist for scripting:

```bash
gc tui                    # Launch Flight Deck TUI (default)
gc adopt <path>           # Adopt existing project
gc new <name>             # Create new project from template
gc status [project]       # Show project status
gc start [project]        # Start session for project
gc stop [project]         # Stop session
gc teleport [project]     # Attach to session
gc costs                  # Show cost summary
```

---

## 11. Future Considerations (Post-MVP)

- **Web UI**: Remote dashboard via Cloudflare Tunnel
- **Protocol Droid**: Security monitoring agent (TDD enforcement, etc.)
- **Multi-agent**: Multiple Claude sessions coordinating
- **Mobile approvals**: Push notifications for approval requests

---

## Appendix A: tmux-cli Reference

tmux-cli provides a higher-level interface for AI agents to manage tmux:

```bash
# Launch a shell (always do this first)
tmux-cli launch "zsh"

# Send command to pane
tmux-cli send "claude" --pane=2

# Capture output
tmux-cli capture --pane=2

# Wait for idle (command finished)
tmux-cli wait_idle --pane=2 --idle-time=2.0 --timeout=30

# Send interrupt (Ctrl+C)
tmux-cli interrupt --pane=2

# List panes
tmux-cli list_panes

# Kill pane
tmux-cli kill --pane=2
```

---

## Appendix B: Library Versions

| Library | Version | Purpose |
|---------|---------|---------|
| Bubble Tea | v2.0.0-beta | TUI framework |
| Lip Gloss | v2.0.0-beta | Terminal styling |
| fsnotify | latest | File watching |
| Cobra | v1.9.1 | CLI framework |
| tmux-cli | latest | Tmux management |
