package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/artifact"
	"github.com/mmariani/ground-control/internal/claude"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/types"
	"github.com/spf13/cobra"
)

var (
	brainstormHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	brainstormPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	brainstormHintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	brainstormSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Bold(true)
)

// NewBrainstormCmd creates the brainstorm command.
func NewBrainstormCmd(store *data.Store) *cobra.Command {
	var listSessions bool
	var createTasks bool
	var createSprint bool

	cmd := &cobra.Command{
		Use:   "brainstorm <topic>",
		Short: "Start an interactive brainstorm session",
		Long: `Interactive brainstorming for feature ideas and concepts.

Uses Claude CLI to explore ideas, evaluate options, and capture results.
Sessions are saved to data/brainstorms/ for future reference.

Examples:
  gc brainstorm "API design"
  gc brainstorm "Performance improvements" --create-tasks
  gc brainstorm --list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if listSessions {
				return runBrainstormList(store)
			}
			if len(args) == 0 {
				return fmt.Errorf("topic required (use --list to view sessions)")
			}
			topic := strings.Join(args, " ")
			return runBrainstorm(store, topic, createTasks, createSprint)
		},
	}

	cmd.Flags().BoolVar(&listSessions, "list", false, "List past brainstorm sessions")
	cmd.Flags().BoolVar(&createTasks, "create-tasks", false, "Create tasks from ideas")
	cmd.Flags().BoolVar(&createSprint, "create-sprint", false, "Create a sprint from ideas")

	return cmd
}

// runBrainstorm runs the interactive brainstorm session.
func runBrainstorm(store *data.Store, topic string, createTasks, createSprint bool) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(brainstormHeaderStyle.Render("═══ Brainstorm Session ═══"))
	fmt.Println()
	fmt.Printf("%s %s\n\n", brainstormPromptStyle.Render("Topic:"), topic)

	// Load feature_ideas template
	tmpl, err := artifact.LoadTemplate(store.GetDataDir(), "feature_ideas")
	if err != nil {
		return fmt.Errorf("loading template: %w", err)
	}

	variables := make(map[string]string)
	variables["topic"] = topic

	// Problem statement
	fmt.Println(brainstormHintStyle.Render("What problem are we solving?"))
	fmt.Print("> ")
	problemStatement, _ := reader.ReadString('\n')
	problemStatement = strings.TrimSpace(problemStatement)
	if problemStatement == "" {
		return fmt.Errorf("problem statement required")
	}
	variables["problem_statement"] = problemStatement
	fmt.Println()

	// Target users
	fmt.Println(brainstormHintStyle.Render("Who will use these features?"))
	fmt.Print("> ")
	targetUsers, _ := reader.ReadString('\n')
	targetUsers = strings.TrimSpace(targetUsers)
	if targetUsers == "" {
		targetUsers = "Development team"
	}
	variables["target_users"] = targetUsers
	fmt.Println()

	// Generate ideas with Claude
	fmt.Println(brainstormHeaderStyle.Render("Generating ideas with Claude..."))
	fmt.Println()

	claudeClient := claude.NewClient(claude.DefaultConfig())
	ideationPrompt := buildIdeationPrompt(topic, problemStatement, targetUsers)

	ideationReq := &claude.Request{
		Prompt: ideationPrompt,
	}

	response := claudeClient.Execute(ideationReq)
	if response.Error != nil {
		return fmt.Errorf("claude ideation failed: %w", response.Error)
	}

	ideas := response.Output
	fmt.Println(ideas)
	fmt.Println()

	// Accept or edit ideas
	fmt.Println(brainstormHintStyle.Render("Accept ideas as-is? (y/n, or edit)"))
	fmt.Print("> ")
	acceptInput, _ := reader.ReadString('\n')
	acceptInput = strings.TrimSpace(strings.ToLower(acceptInput))

	if acceptInput != "y" && acceptInput != "yes" {
		fmt.Println(brainstormHintStyle.Render("Enter your ideas (end with a line containing only '---'):"))
		var userIdeas strings.Builder
		for {
			line, _ := reader.ReadString('\n')
			if strings.TrimSpace(line) == "---" {
				break
			}
			userIdeas.WriteString(line)
		}
		if userIdeas.Len() > 0 {
			ideas = userIdeas.String()
		}
	}
	variables["ideas"] = ideas
	fmt.Println()

	// Constraints (optional)
	fmt.Println(brainstormHintStyle.Render("Any constraints? (press Enter to skip)"))
	fmt.Print("> ")
	constraints, _ := reader.ReadString('\n')
	constraints = strings.TrimSpace(constraints)
	variables["constraints"] = constraints
	fmt.Println()

	// Next steps (optional)
	fmt.Println(brainstormHintStyle.Render("Next steps? (press Enter to skip)"))
	fmt.Print("> ")
	nextSteps, _ := reader.ReadString('\n')
	nextSteps = strings.TrimSpace(nextSteps)
	variables["next_steps"] = nextSteps
	fmt.Println()

	// Generate artifact
	result, err := artifact.GenerateArtifact(tmpl, variables)
	if err != nil {
		return fmt.Errorf("generating artifact: %w", err)
	}

	// Save to data/brainstorms
	timestamp := time.Now().Format("20060102_150405")
	sanitizedTopic := strings.ReplaceAll(strings.ToLower(topic), " ", "_")
	filename := fmt.Sprintf("%s_%s.md", sanitizedTopic, timestamp)
	brainstormsDir := filepath.Join(store.GetDataDir(), "brainstorms")
	outputPath := filepath.Join(brainstormsDir, filename)

	if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
		return fmt.Errorf("saving brainstorm: %w", err)
	}

	fmt.Println(brainstormSuccessStyle.Render("✓ Brainstorm session saved"))
	fmt.Printf("  %s\n\n", outputPath)

	// Create tasks if requested
	if createTasks {
		fmt.Println(brainstormHeaderStyle.Render("Creating tasks from ideas..."))
		fmt.Println()
		taskIDs, err := createTasksFromIdeas(store, topic, ideas)
		if err != nil {
			return fmt.Errorf("creating tasks: %w", err)
		}
		fmt.Printf("%s Created %d tasks\n", brainstormSuccessStyle.Render("✓"), len(taskIDs))
		for _, id := range taskIDs {
			fmt.Printf("  - %s\n", id)
		}
		fmt.Println()

		// Create sprint if requested
		if createSprint && len(taskIDs) > 0 {
			sprintName := fmt.Sprintf("%s Sprint", topic)
			sprint, err := store.CreateSprint(sprintName, fmt.Sprintf("Sprint for %s brainstorm", topic), problemStatement)
			if err != nil {
				return fmt.Errorf("creating sprint: %w", err)
			}

			// Add tasks to sprint
			for _, taskID := range taskIDs {
				if err := store.AddTaskToSprint(sprint.ID, taskID); err != nil {
					fmt.Printf("Warning: failed to add task %s to sprint: %v\n", taskID, err)
				}
			}

			fmt.Printf("%s Sprint created: %s\n", brainstormSuccessStyle.Render("✓"), sprint.ID)
		}
	}

	return nil
}

// runBrainstormList lists past brainstorm sessions.
func runBrainstormList(store *data.Store) error {
	brainstormsDir := filepath.Join(store.GetDataDir(), "brainstorms")

	entries, err := os.ReadDir(brainstormsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(brainstormHintStyle.Render("No brainstorm sessions yet."))
			return nil
		}
		return fmt.Errorf("reading brainstorms: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println(brainstormHintStyle.Render("No brainstorm sessions yet."))
		return nil
	}

	fmt.Println(brainstormHeaderStyle.Render("Brainstorm Sessions"))
	fmt.Println()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		info, _ := entry.Info()
		fmt.Printf("  %s - %s\n",
			brainstormPromptStyle.Render(entry.Name()),
			brainstormHintStyle.Render(info.ModTime().Format("2006-01-02 15:04")))
	}
	fmt.Println()
	fmt.Println(brainstormHintStyle.Render("Use 'gc brainstorm <topic>' to create a new session."))

	return nil
}

// buildIdeationPrompt creates a prompt for Claude to generate ideas.
func buildIdeationPrompt(topic, problemStatement, targetUsers string) string {
	var sb strings.Builder

	sb.WriteString("You are helping with a brainstorming session.\n\n")
	sb.WriteString("# Topic\n")
	sb.WriteString(topic)
	sb.WriteString("\n\n")
	sb.WriteString("# Problem Statement\n")
	sb.WriteString(problemStatement)
	sb.WriteString("\n\n")
	sb.WriteString("# Target Users\n")
	sb.WriteString(targetUsers)
	sb.WriteString("\n\n")
	sb.WriteString("# Task\n")
	sb.WriteString("Generate 5-8 concrete, actionable feature ideas that address the problem statement.\n")
	sb.WriteString("For each idea, provide:\n")
	sb.WriteString("- A clear title\n")
	sb.WriteString("- A brief description (2-3 sentences)\n")
	sb.WriteString("- Key benefits\n\n")
	sb.WriteString("Format as a markdown list. Be creative but practical.\n")

	return sb.String()
}

// createTasksFromIdeas parses ideas and creates tasks.
func createTasksFromIdeas(store *data.Store, topic, ideas string) ([]string, error) {
	// Parse ideas into individual items
	// Simple parsing: look for markdown list items or numbered items
	lines := strings.Split(ideas, "\n")
	var ideaItems []string
	var currentIdea strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Check if this is a new idea (starts with - or number)
		if (strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") ||
		   (len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' && trimmed[1] == '.')) {
			// Save previous idea if exists
			if currentIdea.Len() > 0 {
				ideaItems = append(ideaItems, currentIdea.String())
				currentIdea.Reset()
			}
			// Start new idea (remove list marker)
			if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") {
				currentIdea.WriteString(strings.TrimSpace(trimmed[1:]))
			} else {
				// numbered list
				idx := strings.Index(trimmed, ".")
				if idx > 0 && idx < len(trimmed)-1 {
					currentIdea.WriteString(strings.TrimSpace(trimmed[idx+1:]))
				}
			}
		} else if trimmed != "" && currentIdea.Len() > 0 {
			// Continuation of current idea
			currentIdea.WriteString(" ")
			currentIdea.WriteString(trimmed)
		}
	}
	// Add last idea
	if currentIdea.Len() > 0 {
		ideaItems = append(ideaItems, currentIdea.String())
	}

	// Create tasks from parsed ideas
	tasks, err := store.LoadTasks()
	if err != nil {
		return nil, err
	}

	var taskIDs []string
	now := time.Now()

	for _, idea := range ideaItems {
		if len(idea) < 10 { // Skip very short items
			continue
		}

		// Extract title (first sentence or first 60 chars)
		title := idea
		if len(title) > 60 {
			title = title[:60] + "..."
		}

		taskID := fmt.Sprintf("task_%d", time.Now().UnixMilli())
		task := types.Task{
			ID:            taskID,
			Title:         title,
			Description:   idea,
			Type:          types.TaskTypeAIPlanning,
			Agent:         stringPtr("planner"),
			AssignedHuman: "matt",
			AutonomyLevel: types.AutonomyCheckpoints,
			Complexity:    3,
			Importance:    types.ImportanceMedium,
			DueUrgency:    types.DueUrgencyNone,
			Context: types.TaskContext{
				Background:   fmt.Sprintf("From brainstorm session: %s", topic),
				Requirements: []string{},
				Constraints:  []string{},
				RelatedTasks: []string{},
			},
			Topics:          []string{topic},
			State:           types.TaskStateCreated,
			BlockedBy:       []string{},
			Outputs:         []types.TaskOutput{},
			SuggestedNext:   []string{},
			AfterCompletion: types.AfterCompletionTaskmasterReview,
			Verification: types.Verification{
				Type: types.VerificationHumanApproval,
			},
			Tags:      []string{"brainstorm", sanitizeTag(topic)},
			CreatedAt: now,
			UpdatedAt: now,
		}

		tasks = append(tasks, task)
		taskIDs = append(taskIDs, taskID)

		// Small delay to ensure unique timestamps
		time.Sleep(2 * time.Millisecond)
	}

	if err := store.SaveTasks(tasks); err != nil {
		return nil, err
	}

	return taskIDs, nil
}

// stringPtr returns a pointer to a string.
func stringPtr(s string) *string {
	return &s
}

// sanitizeTag sanitizes a string for use as a tag.
func sanitizeTag(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
