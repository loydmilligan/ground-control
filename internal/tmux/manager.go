// Package tmux provides a wrapper around tmux-cli for session management
package tmux

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SessionMode determines how Claude sessions are created
type SessionMode string

const (
	// ModeWindow creates Claude in a new tmux window (default)
	ModeWindow SessionMode = "window"
	// ModePane creates Claude in a split pane in current window
	ModePane SessionMode = "pane"
	// ModeHeadless creates Claude in a hidden/detached window
	ModeHeadless SessionMode = "headless"
)

// Manager handles tmux pane operations via tmux-cli
type Manager struct {
	projectPath string
	projectName string
	paneID      string
	windowID    string // for window/headless modes
	mode        SessionMode
	gcPaneID    string // pane where gc is running (for return binding)
}

// PaneInfo represents information about a tmux pane
type PaneInfo struct {
	ID     string `json:"id"`
	Index  int    `json:"index"`
	Active bool   `json:"active"`
	Title  string `json:"title,omitempty"`
}

// NewManager creates a new tmux manager for a project
func NewManager(projectPath, projectName string) *Manager {
	return &Manager{
		projectPath: projectPath,
		projectName: projectName,
		mode:        ModeWindow, // default
	}
}

// SetMode sets the session creation mode
func (m *Manager) SetMode(mode SessionMode) {
	m.mode = mode
}

// Mode returns the current session mode
func (m *Manager) Mode() SessionMode {
	return m.mode
}

// SetPane sets the pane ID for an existing session
func (m *Manager) SetPane(paneID string) {
	m.paneID = paneID
}

// PaneID returns the current pane ID
func (m *Manager) PaneID() string {
	return m.paneID
}

// StartSession creates a new Claude session using the configured mode
func (m *Manager) StartSession() (string, error) {
	return m.StartSessionWithMode(ModeWindow) // default to window
}

// StartSessionWithMode creates a new Claude session with specified mode
func (m *Manager) StartSessionWithMode(mode SessionMode) (string, error) {
	m.mode = mode

	// Use pre-set gcPaneID if available, otherwise try to capture it
	if m.gcPaneID == "" {
		if output, err := exec.Command("tmux", "display-message", "-p", "#{pane_id}").Output(); err == nil {
			m.gcPaneID = strings.TrimSpace(string(output))
		}
	}

	var err error
	switch mode {
	case ModeWindow:
		err = m.createWindow(false)
	case ModeHeadless:
		err = m.createWindow(true)
	case ModePane:
		err = m.createPane()
	default:
		err = m.createWindow(false)
	}

	if err != nil {
		return "", err
	}

	// Change to project directory
	if err := m.Send(fmt.Sprintf("cd %s", m.projectPath)); err != nil {
		return "", fmt.Errorf("failed to cd: %w", err)
	}

	// Wait for shell to be ready
	if err := m.WaitIdle(2, 10); err != nil {
		return "", fmt.Errorf("shell not ready: %w", err)
	}

	// Start Claude
	if err := m.Send("claude"); err != nil {
		return "", fmt.Errorf("failed to start claude: %w", err)
	}

	// Wait for Claude to be ready
	if err := m.WaitIdle(3, 60); err != nil {
		return "", fmt.Errorf("claude not ready: %w", err)
	}

	// Set up return binding (F12 to return to gc)
	m.setupReturnBinding()

	return m.paneID, nil
}

// createWindow creates a new tmux window
func (m *Manager) createWindow(hidden bool) error {
	// Always create detached (-d) to avoid stealing focus from gc TUI
	// User can teleport (t) when they want to switch
	args := []string{"new-window", "-d", "-P", "-F", "#{pane_id}", "-n", m.projectName}

	output, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return fmt.Errorf("failed to create window: %w", err)
	}
	m.paneID = strings.TrimSpace(string(output))
	return nil
}

// createPane creates a split pane in current window
func (m *Manager) createPane() error {
	// Use -d to create detached, avoiding focus switch
	output, err := exec.Command("tmux", "split-window", "-d", "-h", "-P", "-F", "#{pane_id}").Output()
	if err != nil {
		return fmt.Errorf("failed to create pane: %w", err)
	}
	m.paneID = strings.TrimSpace(string(output))
	return nil
}

// setupReturnBinding sets up F12 to return to gc pane
func (m *Manager) setupReturnBinding() {
	if m.gcPaneID == "" {
		return
	}

	// Get the session:window target for the gc pane
	output, err := exec.Command("tmux", "display-message", "-t", m.gcPaneID,
		"-p", "#{session_name}:#{window_index}").Output()
	if err != nil {
		return
	}
	windowTarget := strings.TrimSpace(string(output))

	// Bind F12 using run-shell which can execute multiple commands
	// The shell command switches to the gc window/pane
	shellCmd := fmt.Sprintf("tmux select-window -t '%s' && tmux select-pane -t '%s'",
		windowTarget, m.gcPaneID)
	exec.Command("tmux", "bind-key", "-n", "F12", "run-shell", shellCmd).Run()
}

// ClearReturnBinding removes the F12 binding
func (m *Manager) ClearReturnBinding() {
	exec.Command("tmux", "unbind-key", "-n", "F12").Run()
}

// GCPaneID returns the pane where gc is running
func (m *Manager) GCPaneID() string {
	return m.gcPaneID
}

// SetGCPaneID explicitly sets the pane ID where gc is running
func (m *Manager) SetGCPaneID(paneID string) {
	m.gcPaneID = paneID
}

// GetCurrentPaneID returns the current tmux pane ID
func GetCurrentPaneID() string {
	output, err := exec.Command("tmux", "display-message", "-p", "#{pane_id}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// StartSessionWithPrompt starts Claude and sends an initial prompt
func (m *Manager) StartSessionWithPrompt(prompt string) (string, error) {
	paneID, err := m.StartSession()
	if err != nil {
		return "", err
	}

	// Send the initial prompt
	if prompt != "" {
		if err := m.Send(prompt); err != nil {
			return paneID, fmt.Errorf("failed to send prompt: %w", err)
		}
	}

	return paneID, nil
}

// Send sends text to the pane (with Enter)
func (m *Manager) Send(text string) error {
	if m.paneID == "" {
		return fmt.Errorf("no pane ID set")
	}
	// Use raw tmux send-keys - more reliable than tmux-cli
	cmd := exec.Command("tmux", "send-keys", "-t", m.paneID, text, "Enter")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("send failed: %w: %s", err, output)
	}
	return nil
}

// SendNoEnter sends text without pressing Enter
func (m *Manager) SendNoEnter(text string) error {
	if m.paneID == "" {
		return fmt.Errorf("no pane ID set")
	}
	// Use -l for literal to avoid interpreting special chars
	cmd := exec.Command("tmux", "send-keys", "-t", m.paneID, "-l", text)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("send failed: %w: %s", err, output)
	}
	return nil
}

// SendWithDelay sends text with a custom delay before Enter
func (m *Manager) SendWithDelay(text string, delaySec float64) error {
	if m.paneID == "" {
		return fmt.Errorf("no pane ID set")
	}
	// Send the text
	if err := m.SendNoEnter(text); err != nil {
		return err
	}
	// Wait the delay
	time.Sleep(time.Duration(delaySec * float64(time.Second)))
	// Send Enter
	return exec.Command("tmux", "send-keys", "-t", m.paneID, "Enter").Run()
}

// Capture returns current pane output
func (m *Manager) Capture() (string, error) {
	if m.paneID == "" {
		return "", fmt.Errorf("no pane ID set")
	}
	// Use raw tmux capture-pane
	output, err := exec.Command("tmux", "capture-pane", "-t", m.paneID, "-p").Output()
	if err != nil {
		return "", fmt.Errorf("capture failed: %w", err)
	}
	return string(output), nil
}

// WaitIdle waits for pane to be idle (no output changes)
func (m *Manager) WaitIdle(idleSeconds, timeoutSeconds int) error {
	if m.paneID == "" {
		return fmt.Errorf("no pane ID set")
	}

	// Simple polling implementation - check if output changes
	var lastContent string
	idleCount := 0
	checkInterval := 500 * time.Millisecond
	maxChecks := timeoutSeconds * 2 // 2 checks per second

	for i := 0; i < maxChecks; i++ {
		content, err := m.Capture()
		if err != nil {
			return err
		}

		if content == lastContent {
			idleCount++
			if idleCount >= idleSeconds*2 { // idle for idleSeconds
				return nil
			}
		} else {
			idleCount = 0
			lastContent = content
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("timeout waiting for idle")
}

// Interrupt sends Ctrl+C to the pane
func (m *Manager) Interrupt() error {
	if m.paneID == "" {
		return fmt.Errorf("no pane ID set")
	}
	return exec.Command("tmux", "send-keys", "-t", m.paneID, "C-c").Run()
}

// Escape sends Escape key to the pane
func (m *Manager) Escape() error {
	if m.paneID == "" {
		return fmt.Errorf("no pane ID set")
	}
	return exec.Command("tmux", "send-keys", "-t", m.paneID, "Escape").Run()
}

// Stop gracefully stops the Claude session
func (m *Manager) Stop() error {
	// Try sending /exit to Claude
	if err := m.Send("/exit"); err != nil {
		// Fall back to Escape then Interrupt
		m.Escape()
		time.Sleep(500 * time.Millisecond)
		m.Interrupt()
	}
	return nil
}

// Kill forcefully kills the pane
func (m *Manager) Kill() error {
	if m.paneID == "" {
		return fmt.Errorf("no pane ID set")
	}
	return exec.Command("tmux", "kill-pane", "-t", m.paneID).Run()
}

// Status returns the current tmux status
func Status() (string, error) {
	output, err := exec.Command("tmux-cli", "status").Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// ListPanes returns all panes in the current session
func ListPanes() ([]PaneInfo, error) {
	output, err := exec.Command("tmux-cli", "list_panes").Output()
	if err != nil {
		return nil, fmt.Errorf("list_panes failed: %w", err)
	}

	// Parse JSON output
	var panes []PaneInfo
	if err := json.Unmarshal(output, &panes); err != nil {
		// If not JSON, return empty (might be text format)
		return nil, nil
	}
	return panes, nil
}

// FindPaneByTitle finds a pane by its title
func FindPaneByTitle(title string) (*PaneInfo, error) {
	panes, err := ListPanes()
	if err != nil {
		return nil, err
	}
	for _, p := range panes {
		if strings.Contains(p.Title, title) {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("pane not found: %s", title)
}

// IsInsideTmux checks if we're running inside tmux
func IsInsideTmux() bool {
	output, err := exec.Command("tmux-cli", "status").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "LOCAL")
}

// SelectPane switches focus to a pane (for teleport)
func SelectPane(paneID string) error {
	// Use raw tmux for this since tmux-cli doesn't have select
	return exec.Command("tmux", "select-pane", "-t", paneID).Run()
}
