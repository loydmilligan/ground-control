package cmd

import (
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
	delegateHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	delegateActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Bold(true)
	delegateFieldStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	delegateValueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

// NewDelegateCmd creates the delegate command for AI Matt delegation.
func NewDelegateCmd(store *data.Store) *cobra.Command {
	var interactions int
	var tasks string
	var cancel bool
	var status bool
	var supervised bool
	var noAuth bool

	cmd := &cobra.Command{
		Use:   "delegate",
		Short: "Delegate decisions to AI Matt",
		Long: `Delegate to AI Matt for a number of interactions or until tasks complete.

This sets the 'user' variable to 'ai_matt'. After each action, Claude will
hand off to AI Matt instead of waiting for human input.

Supervised Mode (--supervised):
  Creates a dedicated tmux window with Worker Claude and AI Matt.
  Adds a monitor pane to your current window showing communications
  and approval prompts. You can keep working while delegation runs.

  Password Protection (cryptographic hash verification):
  1. Run: scripts/gc-hash-password.sh
  2. Copy the hash to .env as GC_APPROVAL_PASSWORD_HASH=<hash>
  3. Monitor verifies you know the password by comparing hashes
  4. The plaintext password is NEVER stored - only the hash

  Use --no-auth to skip password requirements (for testing).

Examples:
  gc delegate --interactions 5                    # Basic delegation
  gc delegate --supervised --interactions 5       # Supervised with monitor pane
  gc delegate --supervised -i 5 --no-auth         # Supervised without password
  gc delegate --tasks task_xxx,task_yyy           # Delegate until tasks complete
  gc delegate --status                            # Show delegation status
  gc delegate --cancel                            # Cancel and return to human mode`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if status {
				return showDelegateStatus(store)
			}
			if cancel {
				return cancelDelegationMode(store, supervised)
			}
			if interactions > 0 {
				if supervised {
					return startSupervisedDelegation(store, interactions, noAuth)
				}
				return startDelegationMode(store, "interactions", interactions, nil)
			}
			if tasks != "" {
				taskList := strings.Split(tasks, ",")
				for i, t := range taskList {
					taskList[i] = strings.TrimSpace(t)
				}
				return startDelegationMode(store, "tasks", len(taskList), taskList)
			}
			// Default: show status
			return showDelegateStatus(store)
		},
	}

	cmd.Flags().IntVarP(&interactions, "interactions", "i", 0, "Number of interactions to delegate")
	cmd.Flags().StringVarP(&tasks, "tasks", "t", "", "Comma-separated task IDs to complete")
	cmd.Flags().BoolVar(&cancel, "cancel", false, "Cancel delegation")
	cmd.Flags().BoolVar(&status, "status", false, "Show delegation status")
	cmd.Flags().BoolVar(&supervised, "supervised", false, "Start supervised delegation with monitor pane")
	cmd.Flags().BoolVar(&noAuth, "no-auth", false, "Skip password authentication in monitor (for testing)")

	return cmd
}

func startDelegationMode(store *data.Store, mode string, count int, tasks []string) error {
	// Load existing state
	current, err := loadDelegationMode(store)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	if current.User == "ai_matt" && current.ModeCount > 0 {
		fmt.Println(delegateHeaderStyle.Render("⚠ Delegation already active"))
		fmt.Printf("\n%s %d remaining\n", delegateFieldStyle.Render("Interactions:"), current.ModeCount)
		fmt.Printf("\nUse %s to cancel first.\n", delegateValueStyle.Render("gc delegate --cancel"))
		return nil
	}

	// Set up new delegation
	current.User = "ai_matt"
	current.InteractionMode = mode
	current.ModeCount = count
	current.TargetTasks = tasks
	current.CompletedTasks = []string{}
	current.Status = "worker_active"
	current.StartedAt = time.Now().Format(time.RFC3339)
	current.HandoffCount = 0
	current.Error = ""

	if err := saveDelegationMode(store, current); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Println(delegateHeaderStyle.Render("═══ AI Matt Delegation Started ═══"))
	fmt.Println()
	fmt.Printf("%s %s\n", delegateFieldStyle.Render("User:"), delegateActiveStyle.Render("ai_matt"))
	fmt.Printf("%s %s\n", delegateFieldStyle.Render("Mode:"), mode)

	if mode == "interactions" {
		fmt.Printf("%s %d\n", delegateFieldStyle.Render("Count:"), count)
	} else {
		fmt.Printf("%s %s\n", delegateFieldStyle.Render("Tasks:"), strings.Join(tasks, ", "))
	}

	fmt.Println()
	fmt.Println("AI Matt will receive handoffs at the end of each action.")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Printf("  %s  Check status\n", delegateValueStyle.Render("gc delegate --status"))
	fmt.Printf("  %s  Cancel delegation\n", delegateValueStyle.Render("gc delegate --cancel"))
	fmt.Printf("  %s  Manual handoff\n", delegateValueStyle.Render("gc handoff --to-ai-matt"))

	return nil
}

func showDelegateStatus(store *data.Store) error {
	mode, err := loadDelegationMode(store)
	if err != nil {
		return err
	}

	fmt.Println(delegateHeaderStyle.Render("═══ Delegation Status ═══"))
	fmt.Println()

	if mode.User == "human_matt" || mode.ModeCount <= 0 {
		fmt.Println("User: human_matt (normal mode)")
		fmt.Println()
		fmt.Println("To delegate to AI Matt:")
		fmt.Printf("  %s\n", delegateValueStyle.Render("gc delegate --interactions 5"))
		return nil
	}

	fmt.Printf("%s %s\n", delegateFieldStyle.Render("User:"), delegateActiveStyle.Render("ai_matt"))
	fmt.Printf("%s %s\n", delegateFieldStyle.Render("Mode:"), mode.InteractionMode)
	fmt.Printf("%s %d\n", delegateFieldStyle.Render("Remaining:"), mode.ModeCount)
	fmt.Printf("%s %s\n", delegateFieldStyle.Render("Status:"), mode.Status)
	fmt.Printf("%s %d\n", delegateFieldStyle.Render("Handoffs:"), mode.HandoffCount)

	if mode.StartedAt != "" {
		fmt.Printf("%s %s\n", delegateFieldStyle.Render("Started:"), mode.StartedAt)
	}

	if mode.InteractionMode == "tasks" && len(mode.TargetTasks) > 0 {
		fmt.Println()
		fmt.Printf("%s %s\n", delegateFieldStyle.Render("Target tasks:"), strings.Join(mode.TargetTasks, ", "))
		if len(mode.CompletedTasks) > 0 {
			fmt.Printf("%s %s\n", delegateFieldStyle.Render("Completed:"), strings.Join(mode.CompletedTasks, ", "))
		}
	}

	if mode.Error != "" {
		fmt.Println()
		fmt.Printf("%s %s\n", delegateFieldStyle.Render("Error:"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(mode.Error))
	}

	return nil
}

func cancelDelegationMode(store *data.Store, supervised bool) error {
	mode, err := loadDelegationMode(store)
	if err != nil {
		return err
	}

	if mode.User == "human_matt" {
		fmt.Println("No active delegation to cancel.")
		return nil
	}

	// Calculate duration
	var duration time.Duration
	if mode.StartedAt != "" {
		if t, err := time.Parse(time.RFC3339, mode.StartedAt); err == nil {
			duration = time.Since(t)
		}
	}

	// If supervised mode was used, call the stop script to clean up tmux
	if mode.MonitorPane != "" || supervised {
		projectRoot := getProjectRoot(store)
		stopScript := filepath.Join(projectRoot, "scripts", "stop-delegation-session.sh")
		if _, err := os.Stat(stopScript); err == nil {
			cmd := exec.Command(stopScript)
			cmd.Dir = projectRoot
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("Warning: stop script failed: %v\n", err)
			}
			return nil // Script handles output
		}
	}

	// Reset to human mode
	mode.User = "human_matt"
	mode.Status = "cancelled"
	if err := saveDelegationMode(store, mode); err != nil {
		return err
	}

	fmt.Println(delegateHeaderStyle.Render("═══ Delegation Cancelled ═══"))
	fmt.Println()
	fmt.Printf("%s %s\n", delegateFieldStyle.Render("Duration:"), duration.Round(time.Second))
	fmt.Printf("%s %d\n", delegateFieldStyle.Render("Handoffs made:"), mode.HandoffCount)
	fmt.Println()
	fmt.Println("User set back to: human_matt")

	return nil
}

// startSupervisedDelegation starts a full tmux-based delegation with monitor pane
func startSupervisedDelegation(store *data.Store, interactions int, noAuth bool) error {
	// Check if delegation already active
	current, err := loadDelegationMode(store)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	if current.User == "ai_matt" && current.ModeCount > 0 {
		fmt.Println(delegateHeaderStyle.Render("⚠ Delegation already active"))
		fmt.Printf("\n%s %d remaining\n", delegateFieldStyle.Render("Interactions:"), current.ModeCount)
		fmt.Printf("\nUse %s to cancel first.\n", delegateValueStyle.Render("gc delegate --cancel"))
		return nil
	}

	// Find and run the start script
	projectRoot := getProjectRoot(store)
	startScript := filepath.Join(projectRoot, "scripts", "start-delegation-session.sh")

	if _, err := os.Stat(startScript); os.IsNotExist(err) {
		return fmt.Errorf("delegation script not found: %s", startScript)
	}

	fmt.Println(delegateHeaderStyle.Render("═══ Starting Supervised Delegation ═══"))
	fmt.Println()

	args := []string{fmt.Sprintf("%d", interactions)}
	if noAuth {
		args = append(args, "--no-auth")
	}

	cmd := exec.Command(startScript, args...)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start delegation: %w", err)
	}

	return nil
}

// getProjectRoot returns the project root directory (parent of data dir)
func getProjectRoot(store *data.Store) string {
	dataDir := store.GetDataDir()
	return filepath.Dir(dataDir)
}
