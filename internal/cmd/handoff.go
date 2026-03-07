package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/spf13/cobra"
)

var (
	handoffHeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	handoffActiveStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Bold(true)
	handoffWaitingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	handoffFieldStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// DelegationMode represents the AI Matt delegation state
type DelegationMode struct {
	User            string    `json:"user"`             // "human_matt" or "ai_matt"
	InteractionMode string    `json:"interaction_mode"` // "interactions" or "tasks"
	ModeCount       int       `json:"mode_count"`       // remaining interactions (decremented after AI Matt response)
	TargetTasks     []string  `json:"target_tasks"`     // for task mode
	CompletedTasks  []string  `json:"completed_tasks"`  // tasks completed during delegation
	Status          string    `json:"status"`           // "idle", "worker_active", "waiting_for_ai_matt", "error"
	StartedAt       string    `json:"started_at"`       // ISO timestamp
	LastHandoffAt   string    `json:"last_handoff_at"`  // ISO timestamp
	HandoffCount    int       `json:"handoff_count"`    // total handoffs this session
	Error           string    `json:"error,omitempty"`  // error message if any
	TmuxSession     string    `json:"tmux_session"`     // tmux session name
	ClaudePane      string    `json:"claude_pane"`      // pane ID for Claude (e.g., "0")
	AIMattPane      string    `json:"ai_matt_pane"`     // pane ID for AI Matt (e.g., "1")
}

// NewHandoffCmd creates the handoff command for inter-agent communication
func NewHandoffCmd(store *data.Store) *cobra.Command {
	var toAIMatt bool
	var toClaude bool
	var checkInbox bool
	var checkOutbox bool
	var showStatus bool
	var diagnose bool
	var history bool
	var reset bool
	var message string

	cmd := &cobra.Command{
		Use:   "handoff",
		Short: "Manage handoffs between Claude and AI Matt",
		Long: `Commands for the Claude ↔ AI Matt delegation system.

Used by agents during delegated mode to communicate.

Examples:
  gc handoff --status           # Show current handoff state
  gc handoff --to-ai-matt       # Claude signals handoff to AI Matt
  gc handoff --check-inbox      # AI Matt checks for pending consultation
  gc handoff --to-claude        # AI Matt signals response ready
  gc handoff --check-outbox     # Claude checks for AI Matt response
  gc handoff --diagnose         # Check for issues
  gc handoff --reset            # Reset to clean state`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if reset {
				return resetHandoffState(store)
			}
			if diagnose {
				return diagnoseHandoff(store)
			}
			if history {
				return showHandoffHistory(store)
			}
			if toAIMatt {
				return handoffToAIMatt(store, message)
			}
			if checkInbox {
				return checkAIMattInbox(store)
			}
			if toClaude {
				return handoffToClaude(store, message)
			}
			if checkOutbox {
				return checkClaudeOutbox(store)
			}
			// Default: show status
			return showHandoffStatus(store)
		},
	}

	cmd.Flags().BoolVar(&showStatus, "status", false, "Show handoff status")
	cmd.Flags().BoolVar(&toAIMatt, "to-ai-matt", false, "Signal handoff to AI Matt (writes to inbox)")
	cmd.Flags().BoolVar(&toClaude, "to-claude", false, "Signal response to Claude (writes to outbox)")
	cmd.Flags().BoolVar(&checkInbox, "check-inbox", false, "Check for pending consultation (AI Matt uses this)")
	cmd.Flags().BoolVar(&checkOutbox, "check-outbox", false, "Check for AI Matt response (Claude uses this)")
	cmd.Flags().BoolVar(&diagnose, "diagnose", false, "Diagnose any issues with handoff state")
	cmd.Flags().BoolVar(&history, "history", false, "Show handoff history")
	cmd.Flags().BoolVar(&reset, "reset", false, "Reset handoff state to clean")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Message content for handoff")

	return cmd
}

func getDelegationDir(store *data.Store) string {
	return filepath.Join(store.GetDataDir(), "delegation")
}

func getStatePath(store *data.Store) string {
	return filepath.Join(getDelegationDir(store), "state.json")
}

func getInboxPath(store *data.Store) string {
	return filepath.Join(getDelegationDir(store), "inbox.md")
}

func getOutboxPath(store *data.Store) string {
	return filepath.Join(getDelegationDir(store), "outbox.md")
}

func getHistoryPath(store *data.Store) string {
	return filepath.Join(getDelegationDir(store), "history.jsonl")
}

func ensureDelegationDir(store *data.Store) error {
	return os.MkdirAll(getDelegationDir(store), 0755)
}

func loadDelegationMode(store *data.Store) (*DelegationMode, error) {
	path := getStatePath(store)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default state
			return &DelegationMode{
				User:            "human_matt",
				InteractionMode: "interactions",
				ModeCount:       0,
				Status:          "idle",
				TmuxSession:     "gc",
				ClaudePane:      "0:0.0",
				AIMattPane:      "0:0.1",
			}, nil
		}
		return nil, err
	}

	var mode DelegationMode
	if err := json.Unmarshal(content, &mode); err != nil {
		return nil, fmt.Errorf("parsing state.json: %w", err)
	}
	return &mode, nil
}

func saveDelegationMode(store *data.Store, mode *DelegationMode) error {
	if err := ensureDelegationDir(store); err != nil {
		return err
	}
	content, err := json.MarshalIndent(mode, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getStatePath(store), content, 0644)
}

func appendHistory(store *data.Store, entry map[string]interface{}) error {
	if err := ensureDelegationDir(store); err != nil {
		return err
	}
	entry["timestamp"] = time.Now().Format(time.RFC3339)
	content, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(getHistoryPath(store), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(string(content) + "\n")
	return err
}

func showHandoffStatus(store *data.Store) error {
	mode, err := loadDelegationMode(store)
	if err != nil {
		return err
	}

	fmt.Println(handoffHeaderStyle.Render("═══ Handoff Status ═══"))
	fmt.Println()

	if mode.User == "human_matt" {
		fmt.Printf("%s %s\n", handoffFieldStyle.Render("User:"), "human_matt (normal mode)")
	} else {
		fmt.Printf("%s %s\n", handoffFieldStyle.Render("User:"), handoffActiveStyle.Render("ai_matt (delegated)"))
	}

	fmt.Printf("%s %s\n", handoffFieldStyle.Render("Status:"), mode.Status)
	fmt.Printf("%s %s\n", handoffFieldStyle.Render("Mode:"), mode.InteractionMode)
	fmt.Printf("%s %d\n", handoffFieldStyle.Render("Remaining:"), mode.ModeCount)
	fmt.Printf("%s %d\n", handoffFieldStyle.Render("Handoffs:"), mode.HandoffCount)

	if mode.Error != "" {
		fmt.Printf("%s %s\n", handoffFieldStyle.Render("Error:"), lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(mode.Error))
	}

	// Check file states
	fmt.Println()
	inboxExists := fileExists(getInboxPath(store))
	outboxExists := fileExists(getOutboxPath(store))
	fmt.Printf("%s %v\n", handoffFieldStyle.Render("Inbox has content:"), inboxExists)
	fmt.Printf("%s %v\n", handoffFieldStyle.Render("Outbox has content:"), outboxExists)

	return nil
}

func handoffToAIMatt(store *data.Store, message string) error {
	mode, err := loadDelegationMode(store)
	if err != nil {
		return err
	}

	if mode.User != "ai_matt" {
		fmt.Println("Not in AI Matt delegation mode. Use 'gc delegate' first.")
		return nil
	}

	if mode.ModeCount <= 0 {
		fmt.Println("No more interactions remaining. Switching back to human_matt.")
		mode.User = "human_matt"
		mode.Status = "idle"
		return saveDelegationMode(store, mode)
	}

	// Write to inbox
	if err := ensureDelegationDir(store); err != nil {
		return err
	}

	inboxContent := fmt.Sprintf("# Consultation from Claude\n\n**Time**: %s\n**Handoff #**: %d\n\n## Summary\n\n%s\n",
		time.Now().Format(time.RFC3339),
		mode.HandoffCount+1,
		message)

	if err := os.WriteFile(getInboxPath(store), []byte(inboxContent), 0644); err != nil {
		return err
	}

	// Update state
	mode.Status = "waiting_for_ai_matt"
	mode.LastHandoffAt = time.Now().Format(time.RFC3339)
	mode.HandoffCount++
	if err := saveDelegationMode(store, mode); err != nil {
		return err
	}

	// Log history
	appendHistory(store, map[string]interface{}{
		"from":            "claude",
		"to":              "ai_matt",
		"handoff_number":  mode.HandoffCount,
		"summary_preview": truncate(message, 100),
	})

	// Send notification to AI Matt's pane via raw tmux (more reliable for Claude Code TUI)
	aiMattPane := mode.AIMattPane
	if aiMattPane == "" {
		aiMattPane = "0:0.1" // default
	}
	notifyMsg := fmt.Sprintf("[CLAUDE] New consultation #%d. Run: gc handoff --check-inbox", mode.HandoffCount)
	// Step 1: Send text
	tmuxCmd := exec.Command("tmux", "send-keys", "-t", aiMattPane, notifyMsg)
	if err := tmuxCmd.Run(); err != nil {
		fmt.Printf("Warning: Could not send text to AI Matt pane: %v\n", err)
	}
	// Step 2: Send Enter and retry if not submitted (handles inconsistent newline mode)
	for attempts := 0; attempts < 5; attempts++ {
		time.Sleep(400 * time.Millisecond)
		// Try Escape first to reset any input mode (after first attempt)
		if attempts > 0 {
			exec.Command("tmux", "send-keys", "-t", aiMattPane, "Escape").Run()
			time.Sleep(200 * time.Millisecond)
		}
		exec.Command("tmux", "send-keys", "-t", aiMattPane, "Enter").Run()
		time.Sleep(800 * time.Millisecond)
		// Check if message is still in input (not submitted)
		// Look for [CLAUDE] in the last few lines only (input area)
		out, _ := exec.Command("tmux", "capture-pane", "-t", aiMattPane, "-p", "-S", "-5").Output()
		if !strings.Contains(string(out), "[CLAUDE] New consultation") {
			break // Message was submitted
		}
		// Also check for spinner/thinking indicators as sign of submission
		if strings.Contains(string(out), "Thinking") || strings.Contains(string(out), "...") {
			break
		}
	}

	fmt.Println(handoffHeaderStyle.Render("═══ HANDED OFF TO AI MATT ═══"))
	fmt.Println()
	fmt.Printf("Written to: %s\n", getInboxPath(store))
	fmt.Printf("Handoff #%d | %d interactions remaining\n", mode.HandoffCount, mode.ModeCount)
	fmt.Println()
	fmt.Println(handoffWaitingStyle.Render("Waiting for AI Matt's response..."))

	return nil
}

func checkAIMattInbox(store *data.Store) error {
	mode, err := loadDelegationMode(store)
	if err != nil {
		return err
	}

	inboxPath := getInboxPath(store)
	if !fileExists(inboxPath) {
		fmt.Println("No pending consultation in inbox.")
		return nil
	}

	content, err := os.ReadFile(inboxPath)
	if err != nil {
		return err
	}

	if len(strings.TrimSpace(string(content))) == 0 {
		fmt.Println("Inbox is empty.")
		return nil
	}

	fmt.Println(handoffHeaderStyle.Render("═══ PENDING CONSULTATION ═══"))
	fmt.Println()
	fmt.Println(string(content))
	fmt.Println()
	fmt.Printf("%s %d remaining after this response\n",
		handoffFieldStyle.Render("Interactions:"), mode.ModeCount-1)
	fmt.Println()
	fmt.Println("After making your decision:")
	fmt.Println("1. Write response to: " + getOutboxPath(store))
	fmt.Println("2. Run: gc handoff --to-claude")

	return nil
}

func handoffToClaude(store *data.Store, message string) error {
	mode, err := loadDelegationMode(store)
	if err != nil {
		return err
	}

	// Write to outbox if message provided
	if message != "" {
		outboxContent := fmt.Sprintf("# Response from AI Matt\n\n**Time**: %s\n\n%s\n",
			time.Now().Format(time.RFC3339),
			message)
		if err := os.WriteFile(getOutboxPath(store), []byte(outboxContent), 0644); err != nil {
			return err
		}
	}

	// Check outbox exists
	outboxPath := getOutboxPath(store)
	if !fileExists(outboxPath) {
		return fmt.Errorf("no response in outbox. Write your response to %s first", outboxPath)
	}

	content, err := os.ReadFile(outboxPath)
	if err != nil {
		return err
	}

	// Check for LOW confidence - look for "Confidence" section with "LOW" value
	contentStr := string(content)
	isLowConfidence := false
	lines := strings.Split(contentStr, "\n")
	for i, line := range lines {
		trimmedLower := strings.ToLower(strings.TrimSpace(line))
		// Look specifically for "## confidence" header, not just any line with "confidence"
		if trimmedLower == "## confidence" || strings.HasPrefix(trimmedLower, "## confidence") {
			// Check the next few lines for LOW
			for j := i + 1; j < len(lines) && j < i+3; j++ {
				upperLine := strings.ToUpper(strings.TrimSpace(lines[j]))
				// Stop if we hit another section
				if strings.HasPrefix(upperLine, "## ") {
					break
				}
				// Check for LOW as standalone or at start of line (with any separator)
				if upperLine == "LOW" ||
				   strings.HasSuffix(upperLine, ": LOW") ||
				   strings.HasPrefix(upperLine, "LOW ") ||
				   strings.HasPrefix(upperLine, "LOW—") ||
				   strings.HasPrefix(upperLine, "LOW-") ||
				   strings.HasPrefix(upperLine, "LOW:") {
					isLowConfidence = true
					break
				}
			}
			break
		}
	}
	if isLowConfidence {
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render("⚠ LOW CONFIDENCE DETECTED"))
		fmt.Println()
		fmt.Println("AI Matt has LOW confidence. Pausing delegation and escalating to human.")
		mode.User = "human_matt"
		mode.Status = "escalated"
		mode.Error = "AI Matt LOW confidence - human review needed"
		if err := saveDelegationMode(store, mode); err != nil {
			return err
		}
		return nil
	}

	// Decrement mode_count
	mode.ModeCount--
	if mode.ModeCount <= 0 {
		mode.User = "human_matt"
		mode.Status = "completed"
		fmt.Println(handoffActiveStyle.Render("✓ Delegation complete - all interactions used"))
	} else {
		mode.Status = "worker_active"
	}
	mode.LastHandoffAt = time.Now().Format(time.RFC3339)

	if err := saveDelegationMode(store, mode); err != nil {
		return err
	}

	// Clear inbox
	os.Remove(getInboxPath(store))

	// Log history
	appendHistory(store, map[string]interface{}{
		"from":           "ai_matt",
		"to":             "claude",
		"handoff_number": mode.HandoffCount,
		"response":       truncate(string(content), 100),
	})

	// Send response to Claude's pane via raw tmux (more reliable than tmux-cli for Claude Code TUI)
	claudePane := mode.ClaudePane
	if claudePane == "" {
		claudePane = "0:0.0" // default
	}

	// Extract just the response content (skip markdown headers)
	responseText := extractResponseText(string(content))
	aiMattMsg := fmt.Sprintf("[AI_MATT] %s", responseText)

	// Use raw tmux send-keys with explicit Enter timing for Claude Code TUI
	// Step 1: Send text
	tmuxCmd := exec.Command("tmux", "send-keys", "-t", claudePane, aiMattMsg)
	if err := tmuxCmd.Run(); err != nil {
		fmt.Printf("Warning: Could not send text to Claude pane: %v\n", err)
	}
	// Step 2: Send Enter and retry if not submitted (handles inconsistent newline mode)
	for attempts := 0; attempts < 5; attempts++ {
		time.Sleep(400 * time.Millisecond)
		// Try Escape first to reset any input mode (after first attempt)
		if attempts > 0 {
			exec.Command("tmux", "send-keys", "-t", claudePane, "Escape").Run()
			time.Sleep(200 * time.Millisecond)
		}
		exec.Command("tmux", "send-keys", "-t", claudePane, "Enter").Run()
		time.Sleep(800 * time.Millisecond)
		// Check if message is still in input (not submitted)
		// Look for [AI_MATT] in the last few lines only (input area)
		out, _ := exec.Command("tmux", "capture-pane", "-t", claudePane, "-p", "-S", "-5").Output()
		if !strings.Contains(string(out), "[AI_MATT]") {
			break // Message was submitted
		}
		// Also check for spinner/thinking indicators as sign of submission
		if strings.Contains(string(out), "Thinking") || strings.Contains(string(out), "...") {
			break
		}
	}

	fmt.Println(handoffHeaderStyle.Render("═══ RESPONSE SENT TO CLAUDE ═══"))
	fmt.Println()
	fmt.Printf("Outbox: %s\n", outboxPath)
	if mode.ModeCount > 0 {
		fmt.Printf("%d interactions remaining\n", mode.ModeCount)
	}

	return nil
}

// extractResponseText extracts the main response text from AI Matt's outbox content
func extractResponseText(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inResponse := false

	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))

		// Skip header lines
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(lower, "**time**") {
			continue
		}

		// Start capturing after "## Response" or "## Decision"
		if strings.HasPrefix(lower, "## response") || strings.HasPrefix(lower, "## decision") {
			inResponse = true
			continue
		}

		// Stop at next section
		if inResponse && strings.HasPrefix(line, "## ") {
			break
		}

		if inResponse {
			result = append(result, line)
		}
	}

	// If no structured response found, return trimmed content
	if len(result) == 0 {
		// Remove markdown header if present
		content = strings.TrimPrefix(content, "# Response from AI Matt\n")
		content = strings.TrimPrefix(content, "# AI Matt Response\n")
		return strings.TrimSpace(content)
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}

func checkClaudeOutbox(store *data.Store) error {
	outboxPath := getOutboxPath(store)
	if !fileExists(outboxPath) {
		fmt.Println("No response in outbox yet.")
		return nil
	}

	content, err := os.ReadFile(outboxPath)
	if err != nil {
		return err
	}

	if len(strings.TrimSpace(string(content))) == 0 {
		fmt.Println("Outbox is empty.")
		return nil
	}

	fmt.Println(handoffHeaderStyle.Render("═══ AI MATT'S RESPONSE ═══"))
	fmt.Println()
	fmt.Println(string(content))

	// Clear outbox after reading
	os.Remove(outboxPath)
	fmt.Println()
	fmt.Println(handoffFieldStyle.Render("(Outbox cleared after reading)"))

	return nil
}

func diagnoseHandoff(store *data.Store) error {
	mode, err := loadDelegationMode(store)
	if err != nil {
		return fmt.Errorf("cannot load state: %w", err)
	}

	fmt.Println(handoffHeaderStyle.Render("═══ Handoff Diagnostics ═══"))
	fmt.Println()

	issues := []string{}
	suggestions := []string{}

	// Check state consistency
	inboxExists := fileExists(getInboxPath(store))
	outboxExists := fileExists(getOutboxPath(store))

	fmt.Printf("State: %s\n", mode.Status)
	fmt.Printf("Inbox: %v\n", inboxExists)
	fmt.Printf("Outbox: %v\n", outboxExists)
	fmt.Println()

	if mode.Status == "waiting_for_ai_matt" && !inboxExists {
		issues = append(issues, "State is 'waiting_for_ai_matt' but inbox is empty")
		suggestions = append(suggestions, "gc handoff --reset")
	}

	if mode.Status == "waiting_for_ai_matt" && outboxExists {
		issues = append(issues, "AI Matt responded but state not updated")
		suggestions = append(suggestions, "gc handoff --to-claude")
	}

	if mode.Status == "worker_active" && outboxExists {
		issues = append(issues, "Unprocessed response in outbox")
		suggestions = append(suggestions, "gc handoff --check-outbox")
	}

	if mode.HandoffCount >= 20 {
		issues = append(issues, "Handoff limit (20) reached")
		suggestions = append(suggestions, "gc delegate --cancel")
	}

	if len(issues) == 0 {
		fmt.Println(handoffActiveStyle.Render("✓ No issues detected"))
	} else {
		fmt.Println("Issues found:")
		for i, issue := range issues {
			fmt.Printf("  %d. %s\n", i+1, issue)
		}
		fmt.Println()
		fmt.Println("Suggested fixes:")
		for i, sug := range suggestions {
			fmt.Printf("  %d. %s\n", i+1, sug)
		}
	}

	return nil
}

func showHandoffHistory(store *data.Store) error {
	histPath := getHistoryPath(store)
	if !fileExists(histPath) {
		fmt.Println("No handoff history yet.")
		return nil
	}

	content, err := os.ReadFile(histPath)
	if err != nil {
		return err
	}

	fmt.Println(handoffHeaderStyle.Render("═══ Handoff History ═══"))
	fmt.Println()

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	// Show last 10
	start := 0
	if len(lines) > 10 {
		start = len(lines) - 10
	}

	for i := start; i < len(lines); i++ {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(lines[i]), &entry); err != nil {
			continue
		}
		ts := entry["timestamp"]
		from := entry["from"]
		to := entry["to"]
		fmt.Printf("[%v] %v → %v\n", ts, from, to)
	}

	return nil
}

func resetHandoffState(store *data.Store) error {
	mode := &DelegationMode{
		User:            "human_matt",
		InteractionMode: "interactions",
		ModeCount:       0,
		Status:          "idle",
		TmuxSession:     "gc",
		ClaudePane:      "0:0.0",
		AIMattPane:      "0:0.1",
	}

	if err := ensureDelegationDir(store); err != nil {
		return err
	}

	if err := saveDelegationMode(store, mode); err != nil {
		return err
	}

	// Clear inbox and outbox
	os.Remove(getInboxPath(store))
	os.Remove(getOutboxPath(store))

	fmt.Println(handoffActiveStyle.Render("✓ Handoff state reset to clean"))
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
