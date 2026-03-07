package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/artifact"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/types"
	"github.com/spf13/cobra"
)

var (
	appHeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	appPromptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	appHintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	appSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Bold(true)
	appErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// NewAppCmd creates the app command for project/app creation.
func NewAppCmd(store *data.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Manage app creation",
		Long:  "Create and manage application projects using guided workflows.",
	}

	cmd.AddCommand(newAppCreateCmd(store))

	return cmd
}

func newAppCreateCmd(store *data.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Interactive app creation wizard",
		Long: `Create a new application project using a guided workflow.

This command uses the create_app pipeline to:
1. Generate a project plan (using project_plan template)
2. Generate a task list (using tasks template)
3. Create tasks in tasks.json
4. Offer to run initial setup tasks`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAppCreate(store, args[0])
		},
	}
}

func runAppCreate(store *data.Store, projectName string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(appHeaderStyle.Render("═══ Create New App: " + projectName + " ═══"))
	fmt.Println()

	// Step 1: Load project_plan template
	fmt.Println(appHintStyle.Render("Loading project_plan template..."))
	projectPlanTemplate, err := artifact.LoadTemplate(store.GetDataDir(), "project_plan")
	if err != nil {
		return fmt.Errorf("loading project_plan template: %w", err)
	}

	// Step 2: Gather project plan variables
	fmt.Println()
	fmt.Println(appHeaderStyle.Render("Step 1: Project Planning"))
	fmt.Println()

	planVars := make(map[string]string)
	planVars["project_name"] = projectName

	for _, v := range projectPlanTemplate.Variables {
		if v.Name == "project_name" {
			continue // Already set
		}

		if v.Required {
			planVars[v.Name] = promptAppRequired(reader, v.Name, v.Description, projectPlanTemplate.Guidance[v.Name])
		} else {
			planVars[v.Name] = promptAppOptional(reader, v.Name, v.Description, projectPlanTemplate.Guidance[v.Name])
		}
	}

	// Step 3: Generate project plan
	projectPlan, err := artifact.GenerateArtifact(projectPlanTemplate, planVars)
	if err != nil {
		return fmt.Errorf("generating project plan: %w", err)
	}

	fmt.Println()
	fmt.Println(appHeaderStyle.Render("Generated Project Plan:"))
	fmt.Println()
	fmt.Println(projectPlan)
	fmt.Println()

	if !confirmAppAction(reader, "Continue with this plan?") {
		fmt.Println(appHintStyle.Render("Cancelled."))
		return nil
	}

	// Step 4: Load tasks template
	fmt.Println()
	fmt.Println(appHintStyle.Render("Loading tasks template..."))
	tasksTemplate, err := artifact.LoadTemplate(store.GetDataDir(), "tasks")
	if err != nil {
		return fmt.Errorf("loading tasks template: %w", err)
	}

	// Step 5: Gather task variables
	fmt.Println()
	fmt.Println(appHeaderStyle.Render("Step 2: Task Planning"))
	fmt.Println()

	taskVars := make(map[string]string)
	taskVars["project_name"] = projectName

	for _, v := range tasksTemplate.Variables {
		if v.Name == "project_name" {
			continue // Already set
		}

		if v.Required {
			taskVars[v.Name] = promptAppRequired(reader, v.Name, v.Description, tasksTemplate.Guidance[v.Name])
		} else {
			taskVars[v.Name] = promptAppOptional(reader, v.Name, v.Description, tasksTemplate.Guidance[v.Name])
		}
	}

	// Step 6: Generate task list
	taskList, err := artifact.GenerateArtifact(tasksTemplate, taskVars)
	if err != nil {
		return fmt.Errorf("generating task list: %w", err)
	}

	fmt.Println()
	fmt.Println(appHeaderStyle.Render("Generated Task List:"))
	fmt.Println()
	fmt.Println(taskList)
	fmt.Println()

	if !confirmAppAction(reader, "Create these tasks in tasks.json?") {
		fmt.Println(appHintStyle.Render("Cancelled."))
		return nil
	}

	// Step 7: Create project
	fmt.Println()
	fmt.Println(appHintStyle.Render("Creating project..."))

	projectID := fmt.Sprintf("project_%d", time.Now().UnixMilli())
	now := time.Now()

	techStack := []string{}
	if techStackStr, ok := planVars["tech_stack"]; ok && techStackStr != "" {
		techStack = strings.Split(techStackStr, ",")
		for i := range techStack {
			techStack[i] = strings.TrimSpace(techStack[i])
		}
	}

	project := types.Project{
		ID:            projectID,
		Name:          projectName,
		Description:   planVars["project_goal"],
		Status:        types.ProjectStatusActive,
		Phase:         types.ProjectPhaseScaffolding,
		DefaultHuman:  "matt",
		AllowedAgents: []string{"coder", "planner", "researcher"},
		TechStack:     techStack,
		Tags:          []string{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	projects, err := store.LoadProjects()
	if err != nil {
		return fmt.Errorf("loading projects: %w", err)
	}

	projects = append(projects, project)

	if err := store.SaveProjects(projects); err != nil {
		return fmt.Errorf("saving projects: %w", err)
	}

	fmt.Println(appSuccessStyle.Render("✓ Project created: " + projectID))

	// Step 8: Parse and create tasks
	fmt.Println()
	fmt.Println(appHintStyle.Render("Creating tasks from task list..."))

	// Simple parsing: split by "##" sections and create tasks
	tasks, err := parseTaskList(taskList, projectID)
	if err != nil {
		return fmt.Errorf("parsing task list: %w", err)
	}

	existingTasks, err := store.LoadTasks()
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	existingTasks = append(existingTasks, tasks...)

	if err := store.SaveTasks(existingTasks); err != nil {
		return fmt.Errorf("saving tasks: %w", err)
	}

	fmt.Println(appSuccessStyle.Render(fmt.Sprintf("✓ Created %d tasks", len(tasks))))
	fmt.Println()

	// Step 9: Display created tasks
	fmt.Println(appHeaderStyle.Render("Created Tasks:"))
	for _, task := range tasks {
		fmt.Printf("  %s - %s [%s]\n", task.ID, task.Title, task.State)
	}
	fmt.Println()

	// Step 10: Offer to run initial tasks
	fmt.Println(appHintStyle.Render("Use 'gc tasks' to see all tasks or 'gc orc' to orchestrate."))

	return nil
}

func parseTaskList(taskList, projectID string) ([]types.Task, error) {
	lines := strings.Split(taskList, "\n")
	var tasks []types.Task
	var currentPhase string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect phase headers
		if strings.HasPrefix(line, "## Phase") {
			currentPhase = strings.TrimPrefix(line, "## ")
			continue
		}

		// Parse task lines (simple format: "- Task description")
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			taskTitle := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "-"), "*"))
			if taskTitle == "" {
				continue
			}

			taskID := fmt.Sprintf("task_%d", time.Now().UnixMilli())
			now := time.Now()

			agent := "coder"
			task := types.Task{
				ID:            taskID,
				Title:         taskTitle,
				Description:   fmt.Sprintf("Part of %s", currentPhase),
				Type:          types.TaskTypeCoding,
				Agent:         &agent,
				AssignedHuman: "matt",
				AutonomyLevel: types.AutonomyCheckpoints,
				Complexity:    3,
				Importance:    types.ImportanceMedium,
				DueUrgency:    types.DueUrgencyNone,
				Context: types.TaskContext{
					Background:   fmt.Sprintf("This task is part of %s", currentPhase),
					Requirements: []string{taskTitle},
					Constraints:  []string{},
					RelatedTasks: []string{},
					ProjectID:    &projectID,
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
				ProjectID: &projectID,
				Tags:      []string{currentPhase},
				CreatedAt: now,
				UpdatedAt: now,
			}

			tasks = append(tasks, task)
			// Small sleep to ensure unique timestamps
			time.Sleep(1 * time.Millisecond)
		}
	}

	return tasks, nil
}

func promptAppRequired(reader *bufio.Reader, label, description, guidance string) string {
	hint := description
	if guidance != "" {
		hint = guidance
	}

	for {
		fmt.Printf("%s %s\n", appPromptStyle.Render(label+":"), appHintStyle.Render("("+hint+")"))
		fmt.Print("> ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		fmt.Println()

		if input != "" {
			return input
		}

		fmt.Println(appErrorStyle.Render("This field is required. Please provide a value."))
		fmt.Println()
	}
}

func promptAppOptional(reader *bufio.Reader, label, description, guidance string) string {
	hint := description
	if guidance != "" {
		hint = guidance
	}

	fmt.Printf("%s %s\n", appPromptStyle.Render(label+":"), appHintStyle.Render("("+hint+", press Enter to skip)"))
	fmt.Print("> ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	fmt.Println()

	return input
}

func confirmAppAction(reader *bufio.Reader, question string) bool {
	fmt.Printf("%s %s\n", appPromptStyle.Render(question), appHintStyle.Render("(y/n)"))
	fmt.Print("> ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	fmt.Println()

	return strings.ToLower(input) == "y" || strings.ToLower(input) == "yes"
}
