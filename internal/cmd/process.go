package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/context"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/types"
	"github.com/spf13/cobra"
)

var (
	processHeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	processEntryStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	processDimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	processSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))
	processSkipStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

// NewProcessCmd creates the process command for converting brain dumps to tasks.
func NewProcessCmd(store *data.Store) *cobra.Command {
	var dryRun bool
	var interactive bool
	var listOnly bool

	cmd := &cobra.Command{
		Use:   "process",
		Short: "Process brain dump entries into tasks",
		Long: `Convert unprocessed brain dump entries into tasks.

The process command analyzes brain dump entries, determines task type and
complexity, and creates tasks with context bundles.

Examples:
  gc process               # Process all unprocessed entries
  gc process --list        # List unprocessed entries without processing
  gc process --dry-run     # Show what would be created without saving
  gc process --interactive # Approve each conversion`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProcess(store, dryRun, interactive, listOnly)
		},
	}

	cmd.Flags().BoolVarP(&listOnly, "list", "l", false, "List unprocessed entries without processing")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be created without saving")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Approve each conversion")

	return cmd
}

func runProcess(store *data.Store, dryRun, interactive, listOnly bool) error {
	// Load brain dump entries
	entries, err := store.LoadBrainDump()
	if err != nil {
		return fmt.Errorf("loading brain dumps: %w", err)
	}

	// Filter unprocessed entries
	var unprocessed []types.BrainDumpEntry
	for _, e := range entries {
		if !e.Processed {
			unprocessed = append(unprocessed, e)
		}
	}

	if len(unprocessed) == 0 {
		fmt.Println(processDimStyle.Render("No unprocessed brain dump entries."))
		return nil
	}

	// List-only mode: just show entries and exit
	if listOnly {
		fmt.Println(processHeaderStyle.Render("═══ Unprocessed Brain Dumps ═══"))
		fmt.Println()
		fmt.Printf("Found %d unprocessed entries:\n\n", len(unprocessed))

		for i, entry := range unprocessed {
			// Show entry number and ID
			fmt.Printf("%s %s\n", processEntryStyle.Render(fmt.Sprintf("#%d", i+1)), processDimStyle.Render(entry.ID))

			// Show captured time
			fmt.Printf("   %s\n", processDimStyle.Render("Captured: "+entry.CapturedAt.Format("2006-01-02 15:04")))

			// Show content (truncated if long)
			content := entry.Content
			if len(content) > 100 {
				content = content[:97] + "..."
			}
			// Indent multiline content
			content = strings.ReplaceAll(content, "\n", "\n   ")
			fmt.Printf("   %s\n", content)

			// Show category and urgency if set
			if entry.Category != nil {
				fmt.Printf("   %s\n", processDimStyle.Render("Category: "+*entry.Category))
			}
			if entry.UrgencyHint != nil {
				fmt.Printf("   %s\n", processSkipStyle.Render("Urgency: "+*entry.UrgencyHint))
			}
			fmt.Println()
		}

		fmt.Printf("Run %s to convert these to tasks.\n", processSuccessStyle.Render("gc process"))
		return nil
	}

	fmt.Println(processHeaderStyle.Render("═══ Process Brain Dumps ═══"))
	fmt.Println()
	fmt.Printf("Found %d unprocessed entries.\n\n", len(unprocessed))

	if dryRun {
		fmt.Println(processSkipStyle.Render("DRY RUN - no changes will be saved"))
		fmt.Println()
	}

	reader := bufio.NewReader(os.Stdin)
	var tasksCreated []types.Task
	var processedEntries []string

	for _, entry := range unprocessed {
		fmt.Println(processEntryStyle.Render("Entry: " + entry.ID))
		fmt.Println(processDimStyle.Render("  Captured: " + entry.CapturedAt.Format("2006-01-02 15:04")))
		fmt.Println()

		// Show content (truncated if long)
		content := entry.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		fmt.Printf("  %s\n\n", content)

		// Analyze the entry
		analysis := analyzeEntry(entry)

		fmt.Println(processDimStyle.Render("  Analysis:"))
		fmt.Printf("    Type:       %s\n", analysis.TaskType)
		fmt.Printf("    Complexity: %d\n", analysis.Complexity)
		fmt.Printf("    Title:      %s\n", analysis.Title)
		if analysis.Urgency != "" {
			fmt.Printf("    Urgency:    %s\n", analysis.Urgency)
		}
		fmt.Println()

		// Interactive mode: ask for confirmation
		if interactive && !dryRun {
			fmt.Print("Create task? [Y/n/s(skip)] > ")
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response == "n" || response == "s" {
				fmt.Println(processSkipStyle.Render("  Skipped"))
				fmt.Println()
				continue
			}
		}

		// Create the task
		task := createTaskFromEntry(entry, analysis)

		if !dryRun {
			// Build context bundle
			projectDir, _ := os.Getwd()
			builder := context.NewBuilder(store.GetDataDir(), projectDir)

			bundle, err := builder.BuildBundle(&task)
			if err != nil {
				fmt.Printf("  %s\n", processDimStyle.Render("Warning: could not build context bundle: "+err.Error()))
			} else {
				task.ContextBundle = bundle
			}

			tasksCreated = append(tasksCreated, task)
			processedEntries = append(processedEntries, entry.ID)

			fmt.Printf("  %s %s\n", processSuccessStyle.Render("✓ Task created:"), task.ID)
		} else {
			fmt.Printf("  %s %s\n", processDimStyle.Render("Would create:"), task.Title)
		}

		fmt.Println()
	}

	// Save changes
	if !dryRun && len(tasksCreated) > 0 {
		// Load and update tasks
		tasks, err := store.LoadTasks()
		if err != nil {
			return fmt.Errorf("loading tasks: %w", err)
		}

		tasks = append(tasks, tasksCreated...)

		if err := store.SaveTasks(tasks); err != nil {
			return fmt.Errorf("saving tasks: %w", err)
		}

		// Mark entries as processed
		now := time.Now()
		for i := range entries {
			for _, processedID := range processedEntries {
				if entries[i].ID == processedID {
					entries[i].Processed = true
					entries[i].ProcessedAt = &now
					// Find the corresponding task
					for _, t := range tasksCreated {
						if strings.Contains(t.Description, entries[i].Content[:min(50, len(entries[i].Content))]) {
							entries[i].ConvertedTo = &t.ID
							break
						}
					}
					break
				}
			}
		}

		if err := store.SaveBrainDump(entries); err != nil {
			return fmt.Errorf("saving brain dumps: %w", err)
		}

		fmt.Println(processHeaderStyle.Render("Summary"))
		fmt.Printf("  Tasks created: %d\n", len(tasksCreated))
		fmt.Printf("  Entries processed: %d\n", len(processedEntries))
	}

	return nil
}

// EntryAnalysis holds the analysis results for a brain dump entry.
type EntryAnalysis struct {
	Title      string
	TaskType   types.TaskType
	Complexity int
	Urgency    string
	Tags       []string
}

// analyzeEntry extracts task information from a brain dump entry.
func analyzeEntry(entry types.BrainDumpEntry) EntryAnalysis {
	content := strings.ToLower(entry.Content)
	analysis := EntryAnalysis{
		TaskType:   types.TaskTypeCoding,
		Complexity: 3,
		Tags:       []string{},
	}

	// Extract title (first sentence or first 60 chars)
	title := entry.Content
	if idx := strings.Index(title, "."); idx > 0 && idx < 100 {
		title = title[:idx]
	} else if idx := strings.Index(title, "\n"); idx > 0 && idx < 100 {
		title = title[:idx]
	}
	if len(title) > 60 {
		title = title[:57] + "..."
	}
	analysis.Title = strings.TrimSpace(title)

	// Detect task type
	if strings.Contains(content, "research") || strings.Contains(content, "investigate") ||
		strings.Contains(content, "explore") || strings.Contains(content, "find out") {
		analysis.TaskType = types.TaskTypeResearch
		analysis.Tags = append(analysis.Tags, "research")
	} else if strings.Contains(content, "plan") || strings.Contains(content, "design") ||
		strings.Contains(content, "architecture") || strings.Contains(content, "discuss") {
		analysis.TaskType = types.TaskTypeAIPlanning
		analysis.Tags = append(analysis.Tags, "planning")
	} else if strings.Contains(content, "bug") || strings.Contains(content, "fix") ||
		strings.Contains(content, "broken") || strings.Contains(content, "error") {
		analysis.TaskType = types.TaskTypeCoding
		analysis.Tags = append(analysis.Tags, "bugfix")
	} else if strings.Contains(content, "implement") || strings.Contains(content, "add") ||
		strings.Contains(content, "create") || strings.Contains(content, "build") {
		analysis.TaskType = types.TaskTypeCoding
		analysis.Tags = append(analysis.Tags, "feature")
	}

	// Detect complexity
	wordCount := len(strings.Fields(entry.Content))
	if wordCount < 20 {
		analysis.Complexity = 2
	} else if wordCount < 50 {
		analysis.Complexity = 3
	} else if wordCount < 100 {
		analysis.Complexity = 4
	} else {
		analysis.Complexity = 5
	}

	// Check for complexity indicators
	if strings.Contains(content, "simple") || strings.Contains(content, "quick") ||
		strings.Contains(content, "trivial") || strings.Contains(content, "small") {
		analysis.Complexity = max(1, analysis.Complexity-1)
	}
	if strings.Contains(content, "complex") || strings.Contains(content, "major") ||
		strings.Contains(content, "significant") || strings.Contains(content, "refactor") {
		analysis.Complexity = min(5, analysis.Complexity+1)
	}

	// Detect urgency
	if entry.UrgencyHint != nil && *entry.UrgencyHint == "urgent" {
		analysis.Urgency = "urgent"
	} else if strings.Contains(content, "urgent") || strings.Contains(content, "asap") ||
		strings.Contains(content, "critical") || strings.Contains(content, "immediately") {
		analysis.Urgency = "urgent"
	}

	// Detect common tags
	if strings.Contains(content, "cli") || strings.Contains(content, "command") {
		analysis.Tags = append(analysis.Tags, "cli")
	}
	if strings.Contains(content, "tui") || strings.Contains(content, "interface") ||
		strings.Contains(content, "ui") {
		analysis.Tags = append(analysis.Tags, "ui")
	}
	if strings.Contains(content, "test") {
		analysis.Tags = append(analysis.Tags, "testing")
	}
	if strings.Contains(content, "doc") || strings.Contains(content, "readme") {
		analysis.Tags = append(analysis.Tags, "documentation")
	}

	return analysis
}

// createTaskFromEntry creates a task from a brain dump entry and its analysis.
func createTaskFromEntry(entry types.BrainDumpEntry, analysis EntryAnalysis) types.Task {
	now := time.Now()
	taskID := fmt.Sprintf("task_%d", now.UnixMilli())

	// Determine agent
	var agent *string
	switch analysis.TaskType {
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

	// Determine importance
	importance := types.ImportanceMedium
	if analysis.Urgency == "urgent" {
		importance = types.ImportanceHigh
	}

	return types.Task{
		ID:            taskID,
		Title:         analysis.Title,
		Description:   entry.Content,
		Type:          analysis.TaskType,
		Agent:         agent,
		AssignedHuman: "matt",
		AutonomyLevel: types.AutonomyCheckpoints,
		Complexity:    analysis.Complexity,
		Importance:    importance,
		DueUrgency:    types.DueUrgencyNone,
		Context: types.TaskContext{
			Background:   fmt.Sprintf("Created from brain dump: %s", entry.ID),
			Requirements: []string{},
			Constraints:  []string{},
			RelatedTasks: []string{},
		},
		Topics:          []string{},
		State:           types.TaskStateCreated,
		BlockedBy:       []string{},
		Outputs:         []types.TaskOutput{},
		SuggestedNext:   []string{},
		AfterCompletion: types.AfterCompletionTaskmasterReview,
		Verification: types.Verification{
			Type: types.VerificationHumanApproval,
		},
		Tags:      analysis.Tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
