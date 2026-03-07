package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/context"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/types"
	"github.com/spf13/cobra"
)

var (
	createHeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	createPromptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	createHintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	createSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Bold(true)
)

// NewCreateCmd creates the create command for interactive task creation.
func NewCreateCmd(store *data.Store) *cobra.Command {
	var quick bool
	var skipContext bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new task interactively",
		Long: `Create a new task with guided prompts.

The create command walks you through task creation, gathering all necessary
information and building a context bundle for agent execution.

Examples:
  gc create           # Full interactive mode
  gc create --quick   # Minimal prompts (title + type only)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(store, quick, skipContext)
		},
	}

	cmd.Flags().BoolVarP(&quick, "quick", "q", false, "Minimal prompts (title + type only)")
	cmd.Flags().BoolVar(&skipContext, "skip-context", false, "Skip context bundle creation")

	return cmd
}

func runCreate(store *data.Store, quick, skipContext bool) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(createHeaderStyle.Render("═══ Create New Task ═══"))
	fmt.Println()

	// Title (required)
	title := promptRequired(reader, "Title", "What needs to be done?")
	if title == "" {
		return fmt.Errorf("title is required")
	}

	// Type (required)
	taskType := promptChoice(reader, "Type", []string{
		"coding",
		"research",
		"ai-planning",
		"human-input",
		"simple",
	}, "coding")

	var description, background string
	var requirements, constraints, tags []string
	complexity := 3
	importance := types.ImportanceMedium
	var blockedBy []string

	if !quick {
		// Description
		description = promptOptional(reader, "Description", "Detailed description of what needs to be done")

		// Background
		background = promptOptional(reader, "Background", "Context the agent needs to know")

		// Requirements
		reqStr := promptOptional(reader, "Requirements", "Comma-separated list of requirements")
		if reqStr != "" {
			requirements = splitAndTrim(reqStr)
		}

		// Constraints
		constStr := promptOptional(reader, "Constraints", "Comma-separated list of constraints")
		if constStr != "" {
			constraints = splitAndTrim(constStr)
		}

		// Complexity
		complexityStr := promptOptional(reader, "Complexity (1-5)", "1=trivial, 3=medium, 5=substantial")
		if complexityStr != "" {
			if c, err := strconv.Atoi(complexityStr); err == nil && c >= 1 && c <= 5 {
				complexity = c
			}
		}

		// Importance
		impStr := promptChoice(reader, "Importance", []string{"high", "medium", "low"}, "medium")
		importance = types.Importance(impStr)

		// Tags
		tagStr := promptOptional(reader, "Tags", "Comma-separated tags")
		if tagStr != "" {
			tags = splitAndTrim(tagStr)
		}

		// Blocked by
		blockedStr := promptOptional(reader, "Blocked by", "Comma-separated task IDs that block this task")
		if blockedStr != "" {
			blockedBy = splitAndTrim(blockedStr)
		}
	} else {
		description = title // Use title as description in quick mode
	}

	// Generate task ID
	taskID := fmt.Sprintf("task_%d", time.Now().UnixMilli())

	// Determine agent based on type
	var agent *string
	switch types.TaskType(taskType) {
	case types.TaskTypeCoding:
		a := "coder"
		agent = &a
	case types.TaskTypeResearch:
		a := "researcher"
		agent = &a
	case types.TaskTypeAIPlanning:
		a := "planner"
		agent = &a
	}

	// Build the task
	now := time.Now()
	task := types.Task{
		ID:            taskID,
		Title:         title,
		Description:   description,
		Type:          types.TaskType(taskType),
		Agent:         agent,
		AssignedHuman: "matt",
		AutonomyLevel: types.AutonomyCheckpoints,
		Complexity:    complexity,
		Importance:    importance,
		DueUrgency:    types.DueUrgencyNone,
		Context: types.TaskContext{
			Background:   background,
			Requirements: requirements,
			Constraints:  constraints,
			RelatedTasks: []string{},
		},
		Topics:          []string{},
		State:           types.TaskStateCreated,
		BlockedBy:       blockedBy,
		Outputs:         []types.TaskOutput{},
		SuggestedNext:   []string{},
		AfterCompletion: types.AfterCompletionTaskmasterReview,
		Verification: types.Verification{
			Type: types.VerificationHumanApproval,
		},
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Build context bundle if not skipped
	if !skipContext {
		fmt.Println()
		fmt.Println(createHintStyle.Render("Building context bundle..."))

		projectDir, _ := os.Getwd()
		builder := context.NewBuilder(store.GetDataDir(), projectDir)

		bundle, err := builder.BuildBundle(&task)
		if err != nil {
			fmt.Printf("%s\n", createHintStyle.Render("Warning: could not build context bundle: "+err.Error()))
		} else {
			task.ContextBundle = bundle
			fmt.Printf("%s\n", createHintStyle.Render("Context bundle created at: "+bundle.BundlePath))
		}
	}

	// Load existing tasks
	tasks, err := store.LoadTasks()
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	// Add new task
	tasks = append(tasks, task)

	// Save
	if err := store.SaveTasks(tasks); err != nil {
		return fmt.Errorf("saving tasks: %w", err)
	}

	fmt.Println()
	fmt.Println(createSuccessStyle.Render("✓ Task created: " + taskID))
	fmt.Println()

	// Show summary
	fmt.Println(createHeaderStyle.Render("Summary"))
	fmt.Printf("  Title:      %s\n", task.Title)
	fmt.Printf("  Type:       %s\n", task.Type)
	fmt.Printf("  Complexity: %d\n", task.Complexity)
	fmt.Printf("  Importance: %s\n", task.Importance)
	if len(task.Tags) > 0 {
		fmt.Printf("  Tags:       %s\n", strings.Join(task.Tags, ", "))
	}
	if len(task.BlockedBy) > 0 {
		fmt.Printf("  Blocked by: %s\n", strings.Join(task.BlockedBy, ", "))
	}
	fmt.Println()
	fmt.Println(createHintStyle.Render("Use 'gc tasks' to see all tasks or 'gc orc' to orchestrate."))

	return nil
}

func promptRequired(reader *bufio.Reader, label, hint string) string {
	fmt.Printf("%s %s\n", createPromptStyle.Render(label+":"), createHintStyle.Render("("+hint+")"))
	fmt.Print("> ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	fmt.Println()
	return input
}

func promptOptional(reader *bufio.Reader, label, hint string) string {
	fmt.Printf("%s %s\n", createPromptStyle.Render(label+":"), createHintStyle.Render("("+hint+", press Enter to skip)"))
	fmt.Print("> ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	fmt.Println()
	return input
}

func promptChoice(reader *bufio.Reader, label string, choices []string, defaultChoice string) string {
	fmt.Printf("%s %s\n", createPromptStyle.Render(label+":"), createHintStyle.Render("(default: "+defaultChoice+")"))
	for i, c := range choices {
		marker := "  "
		if c == defaultChoice {
			marker = "→ "
		}
		fmt.Printf("  %s%d. %s\n", marker, i+1, c)
	}
	fmt.Print("> ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	fmt.Println()

	if input == "" {
		return defaultChoice
	}

	// Try to parse as number
	if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(choices) {
		return choices[num-1]
	}

	// Try to match as string
	for _, c := range choices {
		if strings.EqualFold(c, input) {
			return c
		}
	}

	return defaultChoice
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
