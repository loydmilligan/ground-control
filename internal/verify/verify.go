// Package verify implements task verification for Ground Control.
package verify

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/types"
)

// Styles for verification output
var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Bold(true)
	failStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// Result holds the outcome of a verification check.
type Result struct {
	Passed  bool
	Message string
	Details string
}

// Verify runs the verification for a task and returns the result.
func Verify(task *types.Task) Result {
	switch task.Verification.Type {
	case types.VerificationNone:
		return verifyNone()
	case types.VerificationFileExists:
		return verifyFileExists(task)
	case types.VerificationTestPass:
		return verifyTestPass(task)
	case types.VerificationHumanApproval:
		return verifyHumanApproval(task)
	default:
		return Result{
			Passed:  false,
			Message: "Unknown verification type",
			Details: fmt.Sprintf("Verification type '%s' is not recognized", task.Verification.Type),
		}
	}
}

// verifyNone always passes - used for tasks with no verification requirement.
func verifyNone() Result {
	return Result{
		Passed:  true,
		Message: "No verification required",
	}
}

// verifyFileExists checks that all specified paths exist.
func verifyFileExists(task *types.Task) Result {
	paths := task.Verification.Paths
	if len(paths) == 0 {
		// Fall back to checking task outputs
		for _, out := range task.Outputs {
			if out.Path != "" {
				paths = append(paths, out.Path)
			}
		}
	}

	if len(paths) == 0 {
		return Result{
			Passed:  true,
			Message: "No paths to verify",
		}
	}

	var missing []string
	var found []string

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			missing = append(missing, path)
		} else {
			found = append(found, path)
		}
	}

	if len(missing) > 0 {
		details := fmt.Sprintf("Found:\n  %s\n\nMissing:\n  %s",
			strings.Join(found, "\n  "),
			strings.Join(missing, "\n  "))
		return Result{
			Passed:  false,
			Message: fmt.Sprintf("%d of %d files missing", len(missing), len(paths)),
			Details: details,
		}
	}

	return Result{
		Passed:  true,
		Message: fmt.Sprintf("All %d files exist", len(paths)),
		Details: "Files verified:\n  " + strings.Join(found, "\n  "),
	}
}

// verifyTestPass runs a command and checks for exit code 0.
func verifyTestPass(task *types.Task) Result {
	if task.Verification.Command == nil || *task.Verification.Command == "" {
		return Result{
			Passed:  false,
			Message: "No verification command specified",
		}
	}

	command := *task.Verification.Command
	fmt.Printf("%s Running: %s\n\n", labelStyle.Render("▶"), mutedStyle.Render(command))

	// Run the command through shell
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return Result{
				Passed:  false,
				Message: fmt.Sprintf("Command failed with exit code %d", exitErr.ExitCode()),
				Details: command,
			}
		}
		return Result{
			Passed:  false,
			Message: "Command failed to run",
			Details: err.Error(),
		}
	}

	return Result{
		Passed:  true,
		Message: "Command passed (exit code 0)",
		Details: command,
	}
}

// verifyHumanApproval prompts the user for approval.
func verifyHumanApproval(task *types.Task) Result {
	fmt.Println()
	fmt.Println(labelStyle.Render("Task requires human approval:"))
	fmt.Println()
	fmt.Printf("  %s\n", task.Title)
	fmt.Println()

	// Show outputs if any
	if len(task.Outputs) > 0 {
		fmt.Println(labelStyle.Render("Expected outputs:"))
		for _, out := range task.Outputs {
			icon := "○"
			if out.Path != "" {
				if _, err := os.Stat(out.Path); err == nil {
					icon = "✓"
				}
			}
			path := out.Path
			if path == "" {
				path = "(no file)"
			}
			fmt.Printf("  %s %s — %s\n", icon, path, out.Description)
		}
		fmt.Println()
	}

	fmt.Print(labelStyle.Render("Approve this task? [y/N]: "))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return Result{
			Passed:  false,
			Message: "Failed to read input",
			Details: err.Error(),
		}
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" {
		return Result{
			Passed:  true,
			Message: "Human approved",
		}
	}

	return Result{
		Passed:  false,
		Message: "Human rejected",
	}
}

// CheckOutputs verifies and updates the exists flag for task outputs.
func CheckOutputs(task *types.Task) {
	for i := range task.Outputs {
		if task.Outputs[i].Path != "" {
			_, err := os.Stat(task.Outputs[i].Path)
			task.Outputs[i].Exists = err == nil
		}
	}
}

// PrintResult displays a verification result with nice formatting.
func PrintResult(result Result) {
	fmt.Println()
	if result.Passed {
		fmt.Printf("%s %s\n", successStyle.Render("✓ PASSED:"), result.Message)
	} else {
		fmt.Printf("%s %s\n", failStyle.Render("✗ FAILED:"), result.Message)
	}

	if result.Details != "" {
		fmt.Println()
		fmt.Println(mutedStyle.Render(result.Details))
	}
	fmt.Println()
}
