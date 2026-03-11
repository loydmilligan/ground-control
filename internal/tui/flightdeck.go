// Package tui provides the Flight Deck terminal user interface
package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/altitude"
	"github.com/mmariani/ground-control/internal/costs"
	"github.com/mmariani/ground-control/internal/registry"
	"github.com/mmariani/ground-control/internal/sessions"
	"github.com/mmariani/ground-control/internal/sidecar"
	"github.com/mmariani/ground-control/internal/tmux"
	"github.com/mmariani/ground-control/internal/tui/styles"
	"github.com/mmariani/ground-control/internal/watch"
)

// FlightDeck view modes
type fdViewMode int

const (
	fdViewHangar fdViewMode = iota // Project list
	fdViewMission                  // Active session
	fdViewComms                    // Approvals & messages
	fdViewCosts                    // Cost tracking
)

// FDProjectView combines registry entry with live state
type FDProjectView struct {
	Entry     registry.ProjectEntry
	State     *sidecar.ProjectState
	SyncState *sidecar.ProjectSyncState // Aggregated data
}

// FlightDeckModel is the main Flight Deck TUI model
type FlightDeckModel struct {
	// Core state
	mode     fdViewMode
	registry *registry.Registry
	projects []FDProjectView
	cursor   int
	width    int
	height   int
	err      error

	// Selected project for Mission view
	selectedProject *FDProjectView
	tmuxMgr         *tmux.Manager
	sessionMode     tmux.SessionMode // window, pane, or headless
	gcPaneID        string           // pane where this TUI is running
	altitude        altitude.Config  // current project's altitude config

	// File watcher for live updates
	watcher *watch.Watcher

	// Costs and sessions tracking
	costTracker    *costs.Tracker
	sessionHistory *sessions.History

	// Message input mode
	messageInput   string // current input text
	messageInputOn bool   // whether we're in message input mode
}

// NewFlightDeck creates a new Flight Deck TUI model
func NewFlightDeck() (*FlightDeckModel, error) {
	reg, err := registry.NewRegistry()
	if err != nil {
		return nil, fmt.Errorf("init registry: %w", err)
	}
	return &FlightDeckModel{
		mode:        fdViewHangar,
		registry:    reg,
		sessionMode: tmux.ModeWindow,          // default to window mode
		gcPaneID:    tmux.GetCurrentPaneID(),  // capture our pane ID
	}, nil
}

// Init implements tea.Model
func (m FlightDeckModel) Init() tea.Cmd {
	// Create watcher
	watcher, err := watch.New()
	if err != nil {
		return func() tea.Msg {
			return errMsg{err: fmt.Errorf("create watcher: %w", err)}
		}
	}
	m.watcher = watcher

	// Initialize cost tracker and session history
	m.costTracker = costs.NewTracker()
	m.sessionHistory = sessions.NewHistory()

	// Start watching selected project if any
	if m.selectedProject != nil {
		if err := m.watcher.WatchProject(m.selectedProject.Entry.Path); err != nil {
			// Non-fatal, just log
			m.err = fmt.Errorf("watch project: %w", err)
		}
	}

	return tea.Batch(m.loadProjects, waitForStateUpdate(m.watcher))
}

// Message types
type projectsLoadedMsg struct {
	projects []FDProjectView
}

type errMsg struct {
	err error
}

type tickMsg time.Time

type sessionStartedMsg struct {
	paneID string
}

type sessionStoppedMsg struct{}

type sessionErrorMsg struct {
	err error
}

type teleportReturnMsg struct{}

type stateUpdateMsg watch.StateUpdate

type approvalSentMsg struct {
	approved bool
}

type messageSentMsg struct{}

type messageInputMsg struct {
	text string
}

// loadAggregatedState loads aggregated data from ~/.gc/aggregated.json
func loadAggregatedState(regPath string) (*sidecar.AggregatedState, error) {
	path := filepath.Join(regPath, "aggregated.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state sidecar.AggregatedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// loadProjects fetches all projects and their states
func (m FlightDeckModel) loadProjects() tea.Msg {
	entries, err := m.registry.ListProjectsSorted()
	if err != nil {
		return errMsg{err}
	}

	// Load aggregated state
	regPath := filepath.Dir(m.registry.Path())
	aggregated, aggErr := loadAggregatedState(regPath)

	var views []FDProjectView
	for _, e := range entries {
		mgr := sidecar.NewManager(e.Path)
		state, _ := mgr.LoadState() // Ignore errors, show as inactive

		// Merge aggregated data if available
		var syncState *sidecar.ProjectSyncState
		if aggErr == nil && aggregated != nil {
			if ps, ok := aggregated.Projects[e.Path]; ok {
				syncState = &ps
			}
		}

		views = append(views, FDProjectView{
			Entry:     e,
			State:     state,
			SyncState: syncState,
		})
	}
	return projectsLoadedMsg{views}
}

// Update implements tea.Model
func (m FlightDeckModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case projectsLoadedMsg:
		m.projects = msg.projects
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case sessionStartedMsg:
		// Refresh projects to show updated state
		return m, m.loadProjects

	case sessionStoppedMsg:
		// Refresh projects to show updated state
		m.tmuxMgr = nil
		return m, m.loadProjects

	case sessionErrorMsg:
		m.err = msg.err
		return m, nil

	case stateUpdateMsg:
		// Update the state for the matching project
		if msg.Error != nil {
			m.err = fmt.Errorf("state update: %w", msg.Error)
			return m, waitForStateUpdate(m.watcher)
		}

		// Find and update the project
		for i := range m.projects {
			if m.projects[i].Entry.Path == msg.ProjectPath {
				m.projects[i].State = msg.State
				// Update selected project if it matches
				if m.selectedProject != nil && m.selectedProject.Entry.Path == msg.ProjectPath {
					m.selectedProject.State = msg.State
				}
				break
			}
		}

		// Re-subscribe for next update
		return m, waitForStateUpdate(m.watcher)

	case teleportReturnMsg:
		// TUI resumed after teleport, refresh state
		if m.selectedProject != nil {
			return m, m.loadProjectState()
		}

	case approvalSentMsg:
		// Approval sent, refresh state
		if m.selectedProject != nil {
			return m, m.loadProjectState()
		}

	case messageSentMsg:
		// Message sent successfully
		return m, nil

	case tickMsg:
		// Periodic refresh
		return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}

	return m, nil
}

// handleKey processes keyboard input
func (m FlightDeckModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle message input mode first
	if m.messageInputOn {
		switch key {
		case "enter":
			if m.messageInput != "" {
				cmd := m.sendMessage(m.messageInput)
				m.messageInput = ""
				m.messageInputOn = false
				return m, cmd
			}
		case "esc":
			m.messageInput = ""
			m.messageInputOn = false
			return m, nil
		case "backspace":
			if len(m.messageInput) > 0 {
				m.messageInput = m.messageInput[:len(m.messageInput)-1]
			}
			return m, nil
		default:
			// Add typed character
			if len(key) == 1 {
				m.messageInput += key
			}
			return m, nil
		}
	}

	// Global keys
	switch key {
	case "q", "ctrl+c":
		if m.watcher != nil {
			m.watcher.Close()
		}
		return m, tea.Quit
	case "1":
		m.mode = fdViewHangar
		return m, nil
	case "2":
		m.mode = fdViewMission
		return m, nil
	case "3":
		m.mode = fdViewComms
		return m, nil
	case "4":
		m.mode = fdViewCosts
		return m, nil
	case "r":
		// Refresh
		return m, m.loadProjects
	}

	// Mode-specific keys
	switch m.mode {
	case fdViewHangar:
		return m.handleHangarKey(key)
	case fdViewMission:
		return m.handleMissionKey(key)
	case fdViewComms:
		return m.handleCommsKey(key)
	case fdViewCosts:
		return m.handleCostsKey(key)
	}

	return m, nil
}

// handleHangarKey processes keys in Hangar view
func (m FlightDeckModel) handleHangarKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.cursor < len(m.projects)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if m.cursor < len(m.projects) {
			// Unwatch previous project if any
			if m.selectedProject != nil && m.watcher != nil {
				m.watcher.UnwatchProject(m.selectedProject.Entry.Path)
			}

			// Select new project
			m.selectedProject = &m.projects[m.cursor]
			m.mode = fdViewMission

			// Load altitude configuration for the selected project
			m.altitude = altitude.GetProjectAltitude(m.selectedProject.Entry.Path)

			// Watch new project
			if m.watcher != nil {
				if err := m.watcher.WatchProject(m.selectedProject.Entry.Path); err != nil {
					m.err = fmt.Errorf("watch project: %w", err)
				}
			}
		}
	case "a":
		// TODO: Adopt project flow
	case "n":
		// TODO: New project flow
	}
	return m, nil
}

// handleMissionKey processes keys in Mission view
func (m FlightDeckModel) handleMissionKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.mode = fdViewHangar
	case "t":
		// Teleport to active session
		if m.selectedProject != nil {
			return m, m.teleport()
		}
	case "s":
		// Start session
		if m.selectedProject != nil {
			return m, m.startSession
		}
	case "x":
		// Stop session
		if m.selectedProject != nil && m.tmuxMgr != nil {
			return m, func() tea.Msg { return m.stopSession() }
		}
	case "S":
		// Cycle session mode (only when no session is active)
		if m.selectedProject == nil || m.selectedProject.State == nil || m.selectedProject.State.Session.Status != "active" {
			switch m.sessionMode {
			case tmux.ModeWindow:
				m.sessionMode = tmux.ModePane
			case tmux.ModePane:
				m.sessionMode = tmux.ModeHeadless
			case tmux.ModeHeadless:
				m.sessionMode = tmux.ModeWindow
			}
		}
	case "A":
		// Cycle altitude (only when project is selected)
		if m.selectedProject != nil {
			newLevel := altitude.CycleAltitude(m.altitude.Level)
			if err := altitude.SetProjectAltitude(m.selectedProject.Entry.Path, newLevel); err != nil {
				m.err = err
				return m, nil
			}
			m.altitude = altitude.GetConfig(newLevel)
		}
		return m, nil
	}
	return m, nil
}

// startSession creates a new Claude session in a tmux pane
func (m FlightDeckModel) startSession() tea.Msg {
	if m.selectedProject == nil {
		return sessionErrorMsg{err: fmt.Errorf("no project selected")}
	}

	// Create tmux manager
	m.tmuxMgr = tmux.NewManager(m.selectedProject.Entry.Path, m.selectedProject.Entry.Name)

	// Set the gc pane ID for F12 return binding
	m.tmuxMgr.SetGCPaneID(m.gcPaneID)

	// Start the session with the configured mode
	paneID, err := m.tmuxMgr.StartSessionWithMode(m.sessionMode)
	if err != nil {
		return sessionErrorMsg{err: fmt.Errorf("start session: %w", err)}
	}

	// Load current state
	mgr := sidecar.NewManager(m.selectedProject.Entry.Path)
	state, err := mgr.LoadState()
	if err != nil {
		state = &sidecar.ProjectState{
			Session: sidecar.SessionInfo{Status: "idle"},
		}
	}

	// Update session info
	sessionID := sidecar.GenerateSessionID()
	state.Session = sidecar.SessionInfo{
		ID:           sessionID,
		TmuxPane:     paneID,
		Status:       "active",
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	// Save state
	if err := mgr.SaveState(state); err != nil {
		return sessionErrorMsg{err: fmt.Errorf("save state: %w", err)}
	}

	return sessionStartedMsg{paneID: paneID}
}

// stopSession stops the current Claude session
func (m FlightDeckModel) stopSession() tea.Msg {
	if m.tmuxMgr == nil {
		return sessionErrorMsg{err: fmt.Errorf("no active session")}
	}

	// Clear the F12 return binding
	m.tmuxMgr.ClearReturnBinding()

	// Stop the tmux session
	if err := m.tmuxMgr.Stop(); err != nil {
		return sessionErrorMsg{err: fmt.Errorf("stop session: %w", err)}
	}

	// Load current state
	mgr := sidecar.NewManager(m.selectedProject.Entry.Path)
	state, err := mgr.LoadState()
	if err != nil {
		return sessionErrorMsg{err: fmt.Errorf("load state: %w", err)}
	}

	// Calculate duration
	duration := 0
	if !state.Session.StartedAt.IsZero() {
		duration = int(time.Since(state.Session.StartedAt).Minutes())
	}

	// Create session record
	record := &sidecar.SessionRecord{
		ID:              state.Session.ID,
		StartedAt:       state.Session.StartedAt,
		EndedAt:         time.Now(),
		DurationMinutes: duration,
		Focus:           state.Session.CurrentFocus,
		Outcome:         "stopped",
		Summary:         "Session stopped by user",
		TokensUsed:      state.Costs.SessionTokens,
		CostUSD:         state.Costs.SessionCostUSD,
		FilesModified:   state.Session.FilesTouched,
	}

	// Save session record
	if err := mgr.SaveSession(record); err != nil {
		return sessionErrorMsg{err: fmt.Errorf("save session record: %w", err)}
	}

	// Update state to idle
	state.Session.Status = "idle"
	state.Session.ID = ""
	state.Session.TmuxPane = ""
	state.Costs.SessionTokens = 0
	state.Costs.SessionCostUSD = 0

	// Save updated state
	if err := mgr.SaveState(state); err != nil {
		return sessionErrorMsg{err: fmt.Errorf("save state: %w", err)}
	}

	return sessionStoppedMsg{}
}

// teleport suspends TUI and attaches to the Claude session's tmux pane
func (m FlightDeckModel) teleport() tea.Cmd {
	if m.selectedProject == nil {
		return nil
	}

	// Load current state to get tmux pane
	mgr := sidecar.NewManager(m.selectedProject.Entry.Path)
	state, err := mgr.LoadState()
	if err != nil || state.Session.TmuxPane == "" {
		return func() tea.Msg {
			return sessionErrorMsg{err: fmt.Errorf("no active session to teleport to")}
		}
	}

	// Switch to Claude pane using a goroutine to avoid blocking
	// TUI continues running in its pane
	paneID := state.Session.TmuxPane
	go func() {
		time.Sleep(50 * time.Millisecond) // Small delay for TUI to process
		exec.Command("tmux", "select-window", "-t", paneID).Run()
		exec.Command("tmux", "select-pane", "-t", paneID).Run()
	}()

	return nil // No message needed, user just sees focus change
}

// loadProjectState reloads the selected project's state
func (m FlightDeckModel) loadProjectState() tea.Cmd {
	return func() tea.Msg {
		if m.selectedProject == nil {
			return nil
		}
		mgr := sidecar.NewManager(m.selectedProject.Entry.Path)
		state, err := mgr.LoadState()
		if err != nil {
			return sessionErrorMsg{err: err}
		}
		return stateUpdateMsg{
			ProjectPath: m.selectedProject.Entry.Path,
			State:       state,
		}
	}
}

// sendApproval handles approval response
func (m FlightDeckModel) sendApproval(approved bool) tea.Cmd {
	return func() tea.Msg {
		if m.selectedProject == nil {
			return nil
		}

		mgr := sidecar.NewManager(m.selectedProject.Entry.Path)
		state, err := mgr.LoadState()
		if err != nil {
			return sessionErrorMsg{err: err}
		}

		if state.Approval == nil {
			return nil
		}

		// Save the detail before clearing
		detail := state.Approval.Detail

		if approved {
			// Clear the pending approval - Claude will see this and proceed
			state.AddActivity("approval", "Approved: "+detail)
			state.Approval = nil
		} else {
			// Add denied activity, then clear
			state.AddActivity("approval", "Denied: "+detail)
			state.Approval = nil
		}

		if err := mgr.SaveState(state); err != nil {
			return sessionErrorMsg{err: err}
		}

		return approvalSentMsg{approved: approved}
	}
}

// sendMessage sends a text message to the Claude session via tmux
func (m FlightDeckModel) sendMessage(msg string) tea.Cmd {
	return func() tea.Msg {
		if m.selectedProject == nil {
			return nil
		}

		mgr := sidecar.NewManager(m.selectedProject.Entry.Path)
		state, err := mgr.LoadState()
		if err != nil || state.Session.TmuxPane == "" {
			return sessionErrorMsg{err: fmt.Errorf("no active session")}
		}

		tm := tmux.NewManager(m.selectedProject.Entry.Path, m.selectedProject.Entry.Name)
		tm.SetPane(state.Session.TmuxPane)

		if err := tm.Send(msg); err != nil {
			return sessionErrorMsg{err: err}
		}

		// Log the message as activity
		state.AddActivity("message", "Sent: "+msg)
		mgr.SaveState(state)

		return messageSentMsg{}
	}
}

// handleCommsKey processes keys in Comms view
func (m FlightDeckModel) handleCommsKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y":
		// Approve pending request
		if m.selectedProject != nil {
			mgr := sidecar.NewManager(m.selectedProject.Entry.Path)
			state, _ := mgr.LoadState()
			if state != nil && state.Approval != nil {
				// Check if approval is needed based on altitude
				altCfg := altitude.GetProjectAltitude(m.selectedProject.Entry.Path)
				if !altCfg.NeedsApproval(state.Approval.Type) {
					// Auto-approve if altitude doesn't require approval for this type
					return m, m.sendApproval(true)
				}
				return m, m.sendApproval(true)
			}
		}
	case "n":
		// Deny pending request
		if m.selectedProject != nil {
			mgr := sidecar.NewManager(m.selectedProject.Entry.Path)
			state, _ := mgr.LoadState()
			if state != nil && state.Approval != nil {
				return m, m.sendApproval(false)
			}
		}
	case "m":
		// Start message input mode if session is active
		if m.selectedProject != nil {
			mgr := sidecar.NewManager(m.selectedProject.Entry.Path)
			state, _ := mgr.LoadState()
			if state != nil && state.Session.TmuxPane != "" {
				m.messageInputOn = true
				m.messageInput = ""
				return m, nil
			}
		}
	}
	return m, nil
}

// handleCostsKey processes keys in Costs view
func (m FlightDeckModel) handleCostsKey(key string) (tea.Model, tea.Cmd) {
	// No special keys yet
	return m, nil
}

// View implements tea.Model
func (m FlightDeckModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Header with tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Main content
	switch m.mode {
	case fdViewHangar:
		b.WriteString(m.renderHangar())
	case fdViewMission:
		b.WriteString(m.renderMission())
	case fdViewComms:
		b.WriteString(m.renderComms())
	case fdViewCosts:
		b.WriteString(m.renderCosts())
	}

	// Error display
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(styles.Error.Render("Error: " + m.err.Error()))
	}

	// Footer with help
	b.WriteString("\n\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

// renderTabs renders the tab bar
func (m FlightDeckModel) renderTabs() string {
	tabs := []struct {
		name string
		mode fdViewMode
		key  string
	}{
		{"Hangar", fdViewHangar, "1"},
		{"Mission", fdViewMission, "2"},
		{"Comms", fdViewComms, "3"},
		{"Costs", fdViewCosts, "4"},
	}

	var parts []string
	for _, t := range tabs {
		label := fmt.Sprintf("[%s] %s", t.key, t.name)
		if m.mode == t.mode {
			parts = append(parts, styles.TabActive.Render(label))
		} else {
			parts = append(parts, styles.TabInactive.Render(label))
		}
	}

	// Active indicator
	activeCount := 0
	for _, p := range m.projects {
		if p.State != nil && p.State.Session.Status == "active" {
			activeCount++
		}
	}
	if activeCount > 0 {
		indicator := styles.Active.Render(fmt.Sprintf(" ● %d Active", activeCount))
		parts = append(parts, indicator)
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center, parts...)
	title := styles.Title.Bold(true).Render("FLIGHT DECK")

	return lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", header)
}

// renderHangar renders the project list
func (m FlightDeckModel) renderHangar() string {
	var b strings.Builder

	if len(m.projects) == 0 {
		b.WriteString(styles.MutedText.Render("  No projects adopted yet.\n\n"))
		b.WriteString(styles.Help.Render("  Press 'a' to adopt a project, or run: gc adopt <path>"))
		return b.String()
	}

	// Header
	header := fmt.Sprintf("  %-20s %-10s %-4s %-4s %-5s %-6s %s",
		"PROJECT", "PHASE", "ST", "%", "BUGS", "FLAGS", "LAST")
	b.WriteString(styles.Label.Render(header))
	b.WriteString("\n")
	b.WriteString(styles.MutedText.Render("  " + strings.Repeat("─", m.width-6)))
	b.WriteString("\n")

	// Project rows
	for i, p := range m.projects {
		b.WriteString(m.renderProjectRow(i, p))
		b.WriteString("\n")
	}

	return b.String()
}

// renderProjectRow renders a single project row
func (m FlightDeckModel) renderProjectRow(idx int, p FDProjectView) string {
	// Status icon
	status := "idle"
	if p.State != nil && p.State.Session.ID != "" {
		status = p.State.Session.Status
	}
	icon := styles.StatusIcon(status)
	iconStyled := styles.StatusStyle(status).Render(icon)

	// Phase - from sync state or empty
	phase := "—"
	if p.SyncState != nil && p.SyncState.Phase != "" {
		phase = truncate(p.SyncState.Phase, 10)
	}

	// Completion percentage - from sprint or roadmap
	completionPct := "—"
	if p.SyncState != nil {
		if p.SyncState.Sprint != nil && p.SyncState.Sprint.CompletionPct > 0 {
			completionPct = fmt.Sprintf("%3.0f%%", p.SyncState.Sprint.CompletionPct)
		} else if p.SyncState.RoadmapPct > 0 {
			completionPct = fmt.Sprintf("%3.0f%%", p.SyncState.RoadmapPct)
		}
	}

	// Bugs count - show in warning color if > 0
	bugsStr := "—"
	if p.SyncState != nil && p.SyncState.OpenBugs > 0 {
		bugsStr = styles.WarningText.Render(fmt.Sprintf("%d", p.SyncState.OpenBugs))
	}

	// Attention flags - show in error color if > 0
	flagsStr := "—"
	if p.SyncState != nil && p.SyncState.AttentionFlags > 0 {
		flagsStr = styles.Error.Render(fmt.Sprintf("%d", p.SyncState.AttentionFlags))
	}

	// Last activity time
	lastActive := "—"
	if p.SyncState != nil && p.SyncState.LastActivity != nil {
		lastActive = formatTimeAgo(*p.SyncState.LastActivity)
	} else if !p.Entry.LastActive.IsZero() {
		lastActive = formatTimeAgo(p.Entry.LastActive)
	}

	// Build row
	row := fmt.Sprintf("%s %-19s %-10s %-4s %-4s %-6s %s",
		iconStyled,
		truncate(p.Entry.Name, 19),
		phase,
		status[:2], // Just first 2 chars for status (ac, id, pa)
		completionPct,
		bugsStr,
		flagsStr,
	)

	// Add last active at the end
	row = fmt.Sprintf("%s %s", row, lastActive)

	// Highlight selected
	if idx == m.cursor {
		return styles.ListSelected.Width(m.width - 4).Render(row)
	}
	return styles.ListItem.Render(row)
}

// renderMission renders the active session view
func (m FlightDeckModel) renderMission() string {
	if m.selectedProject == nil {
		return styles.MutedText.Render("  Select a project from Hangar to view mission")
	}

	p := m.selectedProject
	var b strings.Builder

	// Project header
	b.WriteString(styles.Title.Render("  " + p.Entry.Name))
	b.WriteString("\n\n")

	// Session status
	if p.State == nil || p.State.Session.ID == "" {
		b.WriteString(m.renderNoSession())
	} else {
		b.WriteString(m.renderActiveSession(p.State))
	}

	// Session history section
	b.WriteString("\n\n")
	b.WriteString(m.renderSessionHistory())

	return b.String()
}

// renderNoSession renders the "no session" state
func (m FlightDeckModel) renderNoSession() string {
	modeStr := "window"
	switch m.sessionMode {
	case tmux.ModePane:
		modeStr = "pane"
	case tmux.ModeHeadless:
		modeStr = "headless"
	}

	// Build content with altitude info
	var b strings.Builder
	b.WriteString("No active session\n\n")

	// Show altitude
	b.WriteString(fmt.Sprintf("Altitude: %s %s - %s [A to change]\n\n",
		m.altitude.Level.Icon(),
		strings.ToUpper(m.altitude.Level.String()),
		m.altitude.Level.Description()))

	// Show auto-start status for High altitude
	if m.altitude.AutoStartSessions {
		b.WriteString("Auto-start: Enabled\n")
		b.WriteString("(Sessions will start automatically when needed)\n\n")
	}

	b.WriteString("Press [s] to start a Claude session for this project.\n")
	b.WriteString("The session will run in a tmux pane that you can teleport into.\n\n")
	b.WriteString(fmt.Sprintf("Mode: %s [S to change]", modeStr))

	return styles.Box.Width(m.width - 8).Render(b.String())
}

// renderActiveSession renders an active session
func (m FlightDeckModel) renderActiveSession(state *sidecar.ProjectState) string {
	var b strings.Builder

	// Session info header
	sessionInfo := fmt.Sprintf("Session: %s  Started: %s  Files: %d",
		state.Session.ID,
		sessions.RelativeTime(state.Session.StartedAt),
		len(state.Session.FilesTouched),
	)
	b.WriteString(styles.Label.Render("  " + sessionInfo))
	b.WriteString("\n")
	if state.Session.CurrentFocus != "" {
		b.WriteString(styles.Active.Render("  ▶ " + state.Session.CurrentFocus))
		b.WriteString("\n")
	}
	// Show F12 hint when session is active
	b.WriteString(styles.Active.Render("  [F12] Return to Flight Deck"))
	b.WriteString("\n\n")

	// Recent activity - limit to last 5 items
	activityContent := ""
	activityCount := len(state.Activity)
	startIdx := 0
	if activityCount > 5 {
		startIdx = activityCount - 5
	}

	if activityCount > 0 {
		for i := startIdx; i < activityCount; i++ {
			a := state.Activity[i]
			icon := "○"
			switch a.Type {
			case "file_write":
				icon = "✏️"
			case "file_read":
				icon = "📖"
			case "command":
				icon = "⚡"
			}
			activityContent += fmt.Sprintf("%s %s\n", icon, a.Summary)
		}
	} else {
		activityContent += styles.MutedText.Render("Waiting for activity...")
	}

	outputBox := styles.Box.Width(m.width - 8).Height(m.height - 20).Render(activityContent)
	b.WriteString(outputBox)
	b.WriteString("\n")

	// Status bar with cost information
	statusContent := fmt.Sprintf(
		"Tokens: %s  Cost: %s  Files: %d",
		costs.FormatTokens(state.Costs.SessionTokens),
		costs.FormatCost(state.Costs.SessionCostUSD),
		len(state.Session.FilesTouched),
	)
	statusBar := styles.StatusBar.Width(m.width - 8).Render(statusContent)
	b.WriteString(statusBar)

	return b.String()
}

// renderSessionHistory renders recent session history
func (m FlightDeckModel) renderSessionHistory() string {
	if m.selectedProject == nil || m.sessionHistory == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(styles.Subtitle.Render("  Recent Sessions"))
	b.WriteString("\n\n")

	sessionRecords, err := m.sessionHistory.ProjectSessions(m.selectedProject.Entry.Path, 3)
	if err != nil || len(sessionRecords) == 0 {
		b.WriteString(styles.MutedText.Render("  No session history available"))
		return b.String()
	}

	for _, s := range sessionRecords {
		sessionLine := fmt.Sprintf("  %s  %s  %s  %s",
			truncate(s.Focus, 30),
			s.Outcome,
			sessions.FormatDuration(s.DurationMinutes),
			sessions.RelativeTime(s.StartedAt),
		)
		b.WriteString(styles.ListItem.Render(sessionLine))
		b.WriteString("\n")
	}

	return b.String()
}

// renderComms renders the approvals/messages view
func (m FlightDeckModel) renderComms() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Comms"))
	b.WriteString("\n\n")

	if m.selectedProject == nil {
		b.WriteString(styles.MutedText.Render("No project selected"))
		return b.String()
	}

	// Load state to check for pending approval
	mgr := sidecar.NewManager(m.selectedProject.Entry.Path)
	state, err := mgr.LoadState()
	if err != nil {
		b.WriteString(styles.MutedText.Render("Could not load state"))
		return b.String()
	}

	if state.Approval != nil {
		// Check if this approval would be auto-approved based on altitude
		altCfg := altitude.GetProjectAltitude(m.selectedProject.Entry.Path)
		needsApproval := altCfg.NeedsApproval(state.Approval.Type)

		// Show pending approval prominently
		b.WriteString(styles.WarningText.Render("PENDING APPROVAL"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  Type: %s\n", state.Approval.Type))
		b.WriteString(fmt.Sprintf("  Detail: %s\n", state.Approval.Detail))
		if state.Approval.Reason != "" {
			b.WriteString(fmt.Sprintf("  Reason: %s\n", state.Approval.Reason))
		}
		b.WriteString(fmt.Sprintf("  Requested: %s\n", sessions.RelativeTime(state.Approval.RequestedAt)))

		if !needsApproval {
			b.WriteString("\n")
			b.WriteString(styles.MutedText.Render(fmt.Sprintf("  Note: Altitude %s does not require approval for this operation", altCfg.Level.Icon())))
		}

		b.WriteString("\n")
		b.WriteString(styles.Active.Render("[y] Approve"))
		b.WriteString("  ")
		b.WriteString(styles.Error.Render("[n] Deny"))
	} else {
		b.WriteString(styles.MutedText.Render("No pending approvals"))
	}

	// Show message input if active
	if m.messageInputOn {
		b.WriteString("\n\n")
		b.WriteString(styles.Active.Render("Message: "))
		b.WriteString(m.messageInput)
		b.WriteString("█") // cursor
		b.WriteString("\n")
		b.WriteString(styles.MutedText.Render("[Enter] Send  [Esc] Cancel"))
	}

	return b.String()
}

// renderCosts renders the cost tracking view
func (m FlightDeckModel) renderCosts() string {
	var b strings.Builder

	// Load global costs
	cfg, err := m.registry.Load()
	if err != nil {
		return styles.Error.Render("Failed to load costs: " + err.Error())
	}

	// Summary section
	b.WriteString(styles.Subtitle.Render("  Usage Summary\n\n"))

	// Today
	todayBar := styles.ProgressBar(cfg.Costs.TodayUSD, cfg.APIKeys.LimitUSD, 20)
	b.WriteString(fmt.Sprintf("  TODAY        %s    %s  %s limit\n",
		costs.FormatCost(cfg.Costs.TodayUSD), todayBar, costs.FormatCost(cfg.APIKeys.LimitUSD)))

	// Week
	weekLimit := cfg.APIKeys.LimitUSD * 7
	weekBar := styles.ProgressBar(cfg.Costs.WeekUSD, weekLimit, 20)
	b.WriteString(fmt.Sprintf("  THIS WEEK    %s    %s  %s limit\n",
		costs.FormatCost(cfg.Costs.WeekUSD), weekBar, costs.FormatCost(weekLimit)))

	// Month
	monthLimit := cfg.APIKeys.LimitUSD * 30
	monthBar := styles.ProgressBar(cfg.Costs.MonthUSD, monthLimit, 20)
	b.WriteString(fmt.Sprintf("  THIS MONTH   %s   %s  %s limit\n",
		costs.FormatCost(cfg.Costs.MonthUSD), monthBar, costs.FormatCost(monthLimit)))

	b.WriteString("\n")
	b.WriteString(styles.MutedText.Render("  " + strings.Repeat("─", m.width-6)))
	b.WriteString("\n\n")

	// By project with session costs if available
	b.WriteString(styles.Subtitle.Render("  By Project (today)\n\n"))

	for _, p := range m.projects {
		if p.State != nil {
			sessionCost := ""
			if p.State.Session.Status == "active" && p.State.Costs.SessionCostUSD > 0 {
				sessionCost = fmt.Sprintf(" (session: %s)", costs.FormatCost(p.State.Costs.SessionCostUSD))
			}

			todayCost := p.State.Costs.TodayCostUSD
			bar := styles.ProgressBar(todayCost, 10, 15)
			b.WriteString(fmt.Sprintf("  %-20s %s  %s%s\n",
				truncate(p.Entry.Name, 20), costs.FormatCost(todayCost), bar, sessionCost))
		}
	}

	return b.String()
}

// renderHelp renders the context-sensitive help footer
func (m FlightDeckModel) renderHelp() string {
	var help string

	switch m.mode {
	case fdViewHangar:
		help = "[↑/k] Up  [↓/j] Down  [enter] Select  [a] Adopt  [r] Refresh  [q] Quit"
	case fdViewMission:
		// Show teleport option only if there's an active session
		hasActiveSession := m.selectedProject != nil &&
			m.selectedProject.State != nil &&
			m.selectedProject.State.Session.TmuxPane != ""

		if hasActiveSession {
			help = "[t] Teleport  [x] Stop  [A] Altitude  [esc] Back  [q] Quit"
		} else {
			help = "[s] Start session  [A] Altitude  [esc] Back  [q] Quit"
		}
	case fdViewComms:
		// Show message option if there's an active session
		hasActiveSession := m.selectedProject != nil &&
			m.selectedProject.State != nil &&
			m.selectedProject.State.Session.TmuxPane != ""

		if hasActiveSession {
			help = "[y] Approve  [n] Deny  [m] Message  [e] Edit  [?] Ask why  [q] Quit  |  Altitude affects approvals"
		} else {
			help = "[y] Approve  [n] Deny  [e] Edit  [?] Ask why  [q] Quit  |  Altitude affects approvals"
		}
	case fdViewCosts:
		help = "[k] Manage keys  [l] Set limits  [q] Quit"
	}

	return styles.Help.Render("  " + help)
}

// Helper functions

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw ago", int(d.Hours()/(24*7)))
	}
}

// waitForStateUpdate creates a command that waits for the next state update
func waitForStateUpdate(w *watch.Watcher) tea.Cmd {
	return func() tea.Msg {
		update := <-w.Updates()
		return stateUpdateMsg(update)
	}
}

// RunFlightDeck starts the Flight Deck TUI
func RunFlightDeck() error {
	m, err := NewFlightDeck()
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
