package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/claude"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/pipeline"
	"github.com/mmariani/ground-control/internal/session"
	"github.com/mmariani/ground-control/internal/types"
	"github.com/spf13/cobra"
)

var (
	orcHeaderStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	orcTaskStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	orcDimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	orcSuccessStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))
	orcWarningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	orcErrorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	orcProgressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
)

// NewOrcCmd creates the orchestrate command.
func NewOrcCmd(store *data.Store) *cobra.Command {
	var dryRun bool
	var verbose bool
	var continueSession bool
	var customPipeline string
	var parallel bool

	cmd := &cobra.Command{
		Use:   "orc [task indexes or IDs...]",
		Short: "Orchestrate task execution through the pipeline",
		Long: `Execute tasks through the agentic pipeline.

The pipeline varies by task type:
  coding:       Sanity → Coder → Reviewer → Tester → Commit
  simple:       Sanity → Coder → Commit
  research:     Sanity → Researcher → Summary
  ai-planning:  Sanity → Planner → Human Review (fallback to coding for now)
  human-input:  Notify → WaitHuman → CaptureResponse

Examples:
  gc orc 1                    # Run task at index 1 (from gc tasks)
  gc orc 1 3 5                # Run multiple tasks
  gc orc task_123...          # Run by task ID
  gc orc --dry-run 1          # Show execution plan without running
  gc orc --continue           # Resume paused session
  gc orc --pipeline custom 1  # Use custom pipeline from config/pipelines/custom.yaml
  gc orc --parallel 1 2 3     # Enable parallel execution of independent tasks`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrc(store, args, dryRun, verbose, continueSession, customPipeline, parallel)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show execution plan without running")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed output")
	cmd.Flags().BoolVar(&continueSession, "continue", false, "Resume paused session")
	cmd.Flags().StringVar(&customPipeline, "pipeline", "", "Use custom pipeline by name (from config/pipelines/{name}.yaml)")
	cmd.Flags().BoolVar(&parallel, "parallel", false, "Enable parallel execution of independent tasks")

	return cmd
}

func runOrc(store *data.Store, args []string, dryRun, verbose, continueSession bool, customPipeline string, parallel bool) error {
	sessionMgr := session.NewManager(store.GetDataDir())

	// Handle --continue flag
	if continueSession {
		return resumeSession(store, sessionMgr, verbose, customPipeline, parallel)
	}

	// Need at least one task
	if len(args) == 0 {
		return fmt.Errorf("no tasks specified. Use 'gc tasks' to see available tasks with indexes")
	}

	// Load all tasks to resolve indexes
	allTasks, err := store.LoadTasks()
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	// Build index map (same logic as gc tasks - non-completed tasks only)
	indexMap := buildIndexMap(allTasks)

	// Resolve args to task IDs
	taskIDs, err := resolveTaskArgs(args, indexMap, allTasks)
	if err != nil {
		return err
	}

	// Get full task objects
	tasks := getTasksByIDs(allTasks, taskIDs)

	// Validate tasks
	if err := validateTasks(tasks); err != nil {
		return err
	}

	// Show execution plan
	fmt.Println(orcHeaderStyle.Render("═══ Orchestrate Tasks ═══"))
	fmt.Println()

	fmt.Printf("Tasks to execute: %d\n\n", len(tasks))
	for i, t := range tasks {
		blockedStr := ""
		if len(t.BlockedBy) > 0 {
			blockedStr = orcWarningStyle.Render(" [blocked]")
		}
		fmt.Printf("  %d. %s%s\n", i+1, orcTaskStyle.Render(t.Title), blockedStr)
		fmt.Printf("     %s\n", orcDimStyle.Render(fmt.Sprintf("ID: %s  Type: %s  Complexity: %d", t.ID, t.Type, t.Complexity)))
	}
	fmt.Println()

	// Show pipeline stages for each task
	fmt.Println("Pipelines:")
	for i, t := range tasks {
		pipelineDesc := getPipelineDescription(t.Type)
		fmt.Printf("  %d. %s\n", i+1, orcDimStyle.Render(pipelineDesc))
	}
	fmt.Println()

	// Analyze dependencies and show parallelization recommendation
	if parallel && len(tasks) > 1 {
		groups := pipeline.AnalyzeDependencies(tasks)
		if pipeline.CanParallelize(tasks) {
			fmt.Println(orcHeaderStyle.Render("Parallelization Analysis:"))
			fmt.Printf("  Tasks can be executed in %d wave(s)\n", len(groups))
			for i, group := range groups {
				if len(group) > 1 {
					fmt.Printf("  Wave %d: %s (%d tasks can run in parallel)\n",
						i+1,
						orcSuccessStyle.Render(fmt.Sprintf("%d tasks", len(group))),
						len(group))
				} else {
					fmt.Printf("  Wave %d: %s\n", i+1, orcDimStyle.Render("1 task"))
				}
			}
			fmt.Println()
		} else {
			fmt.Println(orcWarningStyle.Render("No parallelization possible - tasks have dependencies"))
			fmt.Println()
		}
	}

	if dryRun {
		fmt.Println(orcWarningStyle.Render("DRY RUN - no changes will be made"))
		return nil
	}

	// Create work session
	sess, err := sessionMgr.Create(taskIDs)
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}

	fmt.Printf("Session: %s\n\n", orcDimStyle.Render(sess.ID))

	// Execute tasks
	return executeSession(store, sessionMgr, sess, tasks, verbose, customPipeline, parallel)
}

func resumeSession(store *data.Store, sessionMgr *session.Manager, verbose bool, customPipeline string, parallel bool) error {
	// Find paused session
	sess, err := sessionMgr.GetPaused()
	if err != nil {
		return fmt.Errorf("finding paused session: %w", err)
	}

	if sess == nil {
		return fmt.Errorf("no paused session found. Use 'gc orc <tasks>' to start a new session")
	}

	fmt.Println(orcHeaderStyle.Render("═══ Resume Session ═══"))
	fmt.Println()
	fmt.Printf("Session: %s\n", orcDimStyle.Render(sess.ID))
	fmt.Printf("Started: %s\n", orcDimStyle.Render(sess.StartedAt.Format("2006-01-02 15:04")))
	fmt.Println()

	// Load tasks
	allTasks, err := store.LoadTasks()
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	tasks := getTasksByIDs(allTasks, sess.TaskIDs)

	// Show progress
	fmt.Println("Progress:")
	for _, taskID := range sess.TaskIDs {
		progress := sess.TaskProgress[taskID]
		task := findTask(tasks, taskID)
		title := taskID
		if task != nil {
			title = task.Title
		}

		statusStyle := orcDimStyle
		switch progress.Status {
		case "completed":
			statusStyle = orcSuccessStyle
		case "failed", "escalated":
			statusStyle = orcErrorStyle
		case "running":
			statusStyle = orcProgressStyle
		}

		fmt.Printf("  %s %s\n", statusStyle.Render("["+progress.Status+"]"), title)
	}
	fmt.Println()

	// Update session status and continue
	sess.UpdateStatus(session.StatusRunning)
	if err := sessionMgr.Save(sess); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}

	return executeSession(store, sessionMgr, sess, tasks, verbose, customPipeline, parallel)
}

func executeSession(store *data.Store, sessionMgr *session.Manager, sess *session.Session, tasks []types.Task, verbose bool, customPipeline string, parallel bool) error {
	sess.UpdateStatus(session.StatusRunning)

	// Get project directory (use current dir if not specified)
	projectDir := "."

	// Create Claude client
	claudeConfig := claude.DefaultConfig()
	claudeConfig.WorkDir = projectDir
	claudeConfig.Verbose = verbose
	claudeClient := claude.NewClient(claudeConfig)

	// Create pipeline config
	pipelineConfig := pipeline.DefaultConfig()
	pipelineConfig.Verbose = verbose

	// If parallel execution is enabled and we have multiple tasks
	if parallel && len(tasks) > 1 {
		return executeSessionParallel(store, sessionMgr, sess, tasks, claudeClient, pipelineConfig, verbose, customPipeline)
	}

	// Sequential execution
	for _, task := range tasks {
		// Skip already completed tasks (for resume)
		if progress, ok := sess.TaskProgress[task.ID]; ok {
			if progress.Status == "completed" {
				continue
			}
		}

		// Check if blocked
		if len(task.BlockedBy) > 0 {
			allTasks, _ := store.LoadTasks()
			blocked := false
			for _, blockerID := range task.BlockedBy {
				blocker := findTaskByID(allTasks, blockerID)
				if blocker != nil && blocker.State != types.TaskStateCompleted {
					blocked = true
					break
				}
			}
			if blocked {
				fmt.Printf("%s Skipping %s (blocked by incomplete tasks)\n",
					orcWarningStyle.Render("⚠"),
					task.Title)
				continue
			}
		}

		if err := executeTask(store, sessionMgr, sess, &task, claudeClient, pipelineConfig, verbose, customPipeline); err != nil {
			// Check if this is an escalation (not a hard failure)
			if sess.TaskProgress[task.ID].Status == "escalated" {
				fmt.Printf("\n%s Task escalated to human. Session paused.\n",
					orcWarningStyle.Render("◷"))
				fmt.Println(orcDimStyle.Render("Use 'gc orc --continue' after providing input."))
				sess.UpdateStatus(session.StatusPaused)
				sessionMgr.Save(sess)
				return nil
			}

			sess.FailTask(task.ID, err.Error())
			sessionMgr.Save(sess)
			return fmt.Errorf("executing task %s: %w", task.ID, err)
		}

		sess.CompleteTask(task.ID)
		sessionMgr.Save(sess)
	}

	// Run session review stage
	fmt.Println()
	fmt.Printf("  %s Session Review...\n", orcProgressStyle.Render("→"))
	reviewStage := pipeline.NewSessionReviewStage(store.GetDataDir())
	reviewCtx := &pipeline.StageContext{
		Session: sess,
		Verbose: verbose,
	}

	if !reviewStage.CanSkip(reviewCtx) {
		result := reviewStage.Execute(reviewCtx)
		if result.Status == pipeline.StageStatusFailed {
			fmt.Printf("    %s %v\n", orcWarningStyle.Render("⚠"), result.Error)
		} else {
			fmt.Printf("    %s %s\n", orcSuccessStyle.Render("✓"), result.Notes)
		}
	} else {
		fmt.Printf("    %s\n", orcDimStyle.Render("No issues to review"))
	}

	sess.UpdateStatus(session.StatusCompleted)
	if err := sessionMgr.Save(sess); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}

	fmt.Println()
	fmt.Println(orcSuccessStyle.Render("✓ All tasks completed successfully"))
	fmt.Println()

	// Show session summary
	fmt.Println(orcHeaderStyle.Render("Session Summary"))
	fmt.Printf("  Session ID: %s\n", sess.ID)
	fmt.Printf("  Tasks:      %d completed\n", len(tasks))
	if len(sess.Issues) > 0 {
		fmt.Printf("  Issues:     %d logged for self-learning\n", len(sess.Issues))
	}

	return nil
}

func executeTask(store *data.Store, sessionMgr *session.Manager, sess *session.Session, task *types.Task, claudeClient *claude.Client, config *pipeline.Config, verbose bool, customPipeline string) error {
	sess.SetCurrentTask(task.ID)
	sessionMgr.Save(sess)

	fmt.Println()
	fmt.Printf("%s %s\n", orcProgressStyle.Render("▶"), orcTaskStyle.Render(task.Title))
	fmt.Println(orcDimStyle.Render("  " + task.ID))
	fmt.Printf("  %s\n", orcDimStyle.Render("Type: "+string(task.Type)))
	fmt.Println()

	// If custom pipeline is specified, use it
	if customPipeline != "" {
		fmt.Printf("  %s\n", orcDimStyle.Render("Using custom pipeline: "+customPipeline))
		pipelineDef, err := pipeline.LoadPipeline(customPipeline)
		if err != nil {
			return fmt.Errorf("loading custom pipeline %s: %w", customPipeline, err)
		}
		fmt.Printf("  %s\n", orcDimStyle.Render("Pipeline: "+pipelineDef.Description))
		// Note: Custom pipeline execution would be implemented here
		// For now, fall back to the standard pipeline routing
		fmt.Println(orcWarningStyle.Render("  ⚠ Custom pipeline execution not yet fully implemented"))
		fmt.Println(orcDimStyle.Render("  Falling back to standard pipeline routing"))
	}

	// Route to appropriate pipeline based on task type
	switch task.Type {
	case types.TaskTypeHumanInput:
		return executeHumanInputTask(sess, sessionMgr, task, verbose)
	case types.TaskTypeResearch:
		return executeResearchTask(store, sessionMgr, sess, task, claudeClient, config, verbose)
	case types.TaskTypeAIPlanning:
		fmt.Println(orcWarningStyle.Render("  ⚠ AI Planning pipeline not yet implemented"))
		fmt.Println(orcDimStyle.Render("  Falling back to coding pipeline for now"))
		return executeCodingTask(store, sessionMgr, sess, task, claudeClient, config, verbose)
	case types.TaskTypeSimple:
		return executeSimpleTask(store, sessionMgr, sess, task, claudeClient, config, verbose)
	case types.TaskTypeCoding:
		return executeCodingTask(store, sessionMgr, sess, task, claudeClient, config, verbose)
	default:
		fmt.Println(orcWarningStyle.Render("  ⚠ Unknown task type, using coding pipeline"))
		return executeCodingTask(store, sessionMgr, sess, task, claudeClient, config, verbose)
	}
}

// executeHumanInputTask handles human-input tasks: Notify → WaitHuman → CaptureResponse.
func executeHumanInputTask(sess *session.Session, sessionMgr *session.Manager, task *types.Task, verbose bool) error {
	ctx := &pipeline.StageContext{
		Task:    task,
		Session: sess,
		Verbose: verbose,
	}

	// Notify
	fmt.Printf("  %s Notify...\n", orcProgressStyle.Render("→"))
	notifyStage := pipeline.NewNotifyStage()
	sess.SetCurrentStage(pipeline.StageNameNotify, 1)
	sessionMgr.Save(sess)

	ctx.Iteration = 1
	notifyResult := notifyStage.Execute(ctx)
	if notifyResult.Status != pipeline.StageStatusSuccess {
		sess.FailStage(pipeline.StageNameNotify, notifyResult.Error.Error())
		sessionMgr.Save(sess)
		return fmt.Errorf("notify failed: %w", notifyResult.Error)
	}
	sess.CompleteStage(pipeline.StageNameNotify, notifyResult.OutputFiles)
	fmt.Printf("    %s %s\n", orcSuccessStyle.Render("✓"), notifyResult.Notes)

	// WaitHuman
	fmt.Printf("  %s WaitHuman...\n", orcProgressStyle.Render("→"))
	waitStage := pipeline.NewWaitHumanStage()
	sess.SetCurrentStage(pipeline.StageNameWaitHuman, 1)
	sessionMgr.Save(sess)

	waitResult := waitStage.Execute(ctx)
	if waitResult.Status == pipeline.StageStatusEscalate {
		sess.EscalateTask(task.ID)
		sessionMgr.Save(sess)
		return fmt.Errorf("escalated: %s", waitResult.Notes)
	}
	if waitResult.Status != pipeline.StageStatusSuccess {
		sess.FailStage(pipeline.StageNameWaitHuman, waitResult.Error.Error())
		sessionMgr.Save(sess)
		return fmt.Errorf("wait human failed: %w", waitResult.Error)
	}
	sess.CompleteStage(pipeline.StageNameWaitHuman, waitResult.OutputFiles)
	fmt.Printf("    %s %s\n", orcSuccessStyle.Render("✓"), waitResult.Notes)

	// CaptureResponse
	fmt.Printf("  %s CaptureResponse...\n", orcProgressStyle.Render("→"))
	captureStage := pipeline.NewCaptureResponseStage()
	sess.SetCurrentStage(pipeline.StageNameCaptureResponse, 1)
	sessionMgr.Save(sess)

	captureResult := captureStage.Execute(ctx)

	// Collect issues
	for _, issue := range captureResult.Issues {
		sess.AddIssue(issue)
	}

	if captureResult.Status != pipeline.StageStatusSuccess {
		sess.FailStage(pipeline.StageNameCaptureResponse, captureResult.Error.Error())
		sessionMgr.Save(sess)
		return fmt.Errorf("capture response failed: %w", captureResult.Error)
	}
	sess.CompleteStage(pipeline.StageNameCaptureResponse, captureResult.OutputFiles)
	fmt.Printf("    %s %s\n", orcSuccessStyle.Render("✓"), captureResult.Notes)

	fmt.Printf("  %s\n", orcSuccessStyle.Render("✓ Task pipeline complete"))
	return nil
}

// executeResearchTask executes the research pipeline: Sanity → Researcher → Summary.
func executeResearchTask(store *data.Store, sessionMgr *session.Manager, sess *session.Session, task *types.Task, claudeClient *claude.Client, config *pipeline.Config, verbose bool) error {
	projectDir := "."
	ctx := &pipeline.StageContext{
		Task:    task,
		Session: sess,
		Verbose: verbose,
	}

	// Sanity Check
	if err := runSanityStage(store, sess, sessionMgr, ctx); err != nil {
		return err
	}

	// Researcher
	fmt.Printf("  %s Researcher...\n", orcProgressStyle.Render("→"))
	researcherStage := pipeline.NewResearcherStage(claudeClient, projectDir)
	sess.SetCurrentStage(pipeline.StageNameResearcher, 1)
	sessionMgr.Save(sess)

	ctx.Iteration = 1
	ctx.PreviousFeedback = ""
	researcherResult := researcherStage.Execute(ctx)

	// Collect issues
	for _, issue := range researcherResult.Issues {
		sess.AddIssue(issue)
	}

	if researcherResult.Status != pipeline.StageStatusSuccess {
		sess.FailStage(pipeline.StageNameResearcher, researcherResult.Error.Error())
		sessionMgr.Save(sess)
		return fmt.Errorf("researcher failed: %w", researcherResult.Error)
	}

	fmt.Printf("    %s Research complete\n", orcSuccessStyle.Render("✓"))
	sess.CompleteStage(pipeline.StageNameResearcher, researcherResult.OutputFiles)
	sessionMgr.Save(sess)

	// Summary
	fmt.Printf("  %s Summary...\n", orcProgressStyle.Render("→"))
	summaryStage := pipeline.NewSummaryStage(claudeClient, projectDir)
	sess.SetCurrentStage(pipeline.StageNameSummary, 1)
	sessionMgr.Save(sess)

	ctx.Iteration = 1
	ctx.PreviousFeedback = ""
	summaryResult := summaryStage.Execute(ctx)

	// Collect issues
	for _, issue := range summaryResult.Issues {
		sess.AddIssue(issue)
	}

	if summaryResult.Status != pipeline.StageStatusSuccess {
		sess.FailStage(pipeline.StageNameSummary, summaryResult.Error.Error())
		sessionMgr.Save(sess)
		return fmt.Errorf("summary failed: %w", summaryResult.Error)
	}

	fmt.Printf("    %s Summary complete\n", orcSuccessStyle.Render("✓"))
	sess.CompleteStage(pipeline.StageNameSummary, summaryResult.OutputFiles)
	sessionMgr.Save(sess)

	fmt.Printf("  %s\n", orcSuccessStyle.Render("✓ Task pipeline complete"))
	return nil
}

// executeSimpleTask executes the simple pipeline: Sanity → Coder → Commit.
func executeSimpleTask(store *data.Store, sessionMgr *session.Manager, sess *session.Session, task *types.Task, claudeClient *claude.Client, config *pipeline.Config, verbose bool) error {
	projectDir := "."
	ctx := &pipeline.StageContext{
		Task:    task,
		Session: sess,
		Verbose: verbose,
	}

	// Sanity Check
	if err := runSanityStage(store, sess, sessionMgr, ctx); err != nil {
		return err
	}

	// Coder
	if err := runCoderStage(claudeClient, sess, sessionMgr, ctx, config, verbose, projectDir); err != nil {
		return err
	}

	// Commit
	if err := runCommitStage(sess, sessionMgr, ctx, projectDir); err != nil {
		return err
	}

	fmt.Printf("  %s\n", orcSuccessStyle.Render("✓ Task pipeline complete"))
	return nil
}

// executeCodingTask executes the full coding pipeline: Sanity → Coder → Reviewer → Tester → Commit.
func executeCodingTask(store *data.Store, sessionMgr *session.Manager, sess *session.Session, task *types.Task, claudeClient *claude.Client, config *pipeline.Config, verbose bool) error {
	projectDir := "."
	ctx := &pipeline.StageContext{
		Task:    task,
		Session: sess,
		Verbose: verbose,
	}

	// Sanity Check
	if err := runSanityStage(store, sess, sessionMgr, ctx); err != nil {
		return err
	}

	// Coder ↔ Reviewer feedback loop
	coderStage := pipeline.NewCoderStage(claudeClient, projectDir)
	reviewerStage := pipeline.NewReviewerStage(claudeClient, projectDir)

	coderResult, err := runCoderReviewerLoop(coderStage, reviewerStage, ctx, sess, sessionMgr, config, verbose)
	if err != nil {
		return err
	}
	if coderResult.Status == pipeline.StageStatusEscalate {
		sess.EscalateTask(task.ID)
		sessionMgr.Save(sess)
		return fmt.Errorf("escalated: %s", coderResult.Feedback)
	}

	// Tester
	fmt.Printf("  %s Tester...\n", orcProgressStyle.Render("→"))
	testerStage := pipeline.NewTesterStage(projectDir)
	sess.SetCurrentStage(pipeline.StageNameTester, 1)
	sessionMgr.Save(sess)

	ctx.Iteration = 1
	ctx.PreviousFeedback = ""
	testerResult := testerStage.Execute(ctx)

	// Collect issues
	for _, issue := range testerResult.Issues {
		sess.AddIssue(issue)
	}

	if testerResult.Status == pipeline.StageStatusNeedsRevision {
		// Run Tester ↔ Coder feedback loop
		testerCoderResult, err := runTesterCoderLoop(testerStage, coderStage, ctx, sess, sessionMgr, config, verbose, testerResult.Feedback)
		if err != nil {
			return err
		}
		if testerCoderResult.Status == pipeline.StageStatusEscalate {
			sess.EscalateTask(task.ID)
			sessionMgr.Save(sess)
			return fmt.Errorf("test escalated: %s", testerCoderResult.Feedback)
		}
		fmt.Printf("    %s\n", orcSuccessStyle.Render("✓ All tests pass"))
	} else if testerResult.Status == pipeline.StageStatusFailed {
		sess.FailStage(pipeline.StageNameTester, testerResult.Error.Error())
		sessionMgr.Save(sess)
		return fmt.Errorf("tester failed: %w", testerResult.Error)
	} else {
		fmt.Printf("    %s\n", orcSuccessStyle.Render("✓ All tests pass"))
	}
	sess.CompleteStage(pipeline.StageNameTester, testerResult.OutputFiles)

	// Commit
	if err := runCommitStage(sess, sessionMgr, ctx, projectDir); err != nil {
		return err
	}

	fmt.Printf("  %s\n", orcSuccessStyle.Render("✓ Task pipeline complete"))
	return nil
}

// runSanityStage runs the sanity check stage.
func runSanityStage(store *data.Store, sess *session.Session, sessionMgr *session.Manager, ctx *pipeline.StageContext) error {
	fmt.Printf("  %s Sanity Check...\n", orcProgressStyle.Render("→"))
	sanityStage := pipeline.NewSanityStage(store)
	sess.SetCurrentStage(pipeline.StageNameSanity, 1)
	sessionMgr.Save(sess)

	ctx.Iteration = 1
	result := sanityStage.Execute(ctx)
	if result.Status != pipeline.StageStatusSuccess {
		sess.FailStage(pipeline.StageNameSanity, result.Error.Error())
		sessionMgr.Save(sess)
		return fmt.Errorf("sanity check failed: %w", result.Error)
	}
	sess.CompleteStage(pipeline.StageNameSanity, result.OutputFiles)
	fmt.Printf("    %s\n", orcSuccessStyle.Render("✓ Passed"))
	return nil
}

// runCoderStage runs just the coder stage (for simple pipeline).
func runCoderStage(claudeClient *claude.Client, sess *session.Session, sessionMgr *session.Manager, ctx *pipeline.StageContext, config *pipeline.Config, verbose bool, projectDir string) error {
	fmt.Printf("  %s Coder...\n", orcProgressStyle.Render("→"))
	coderStage := pipeline.NewCoderStage(claudeClient, projectDir)
	sess.SetCurrentStage(pipeline.StageNameCoder, 1)
	sessionMgr.Save(sess)

	ctx.Iteration = 1
	ctx.PreviousFeedback = ""
	coderResult := coderStage.Execute(ctx)

	// Collect issues
	for _, issue := range coderResult.Issues {
		sess.AddIssue(issue)
	}

	if coderResult.Status == pipeline.StageStatusFailed {
		sess.FailStage(pipeline.StageNameCoder, coderResult.Error.Error())
		sessionMgr.Save(sess)
		return fmt.Errorf("coder failed: %w", coderResult.Error)
	}

	fmt.Printf("    %s Implementation complete\n", orcSuccessStyle.Render("✓"))
	sess.CompleteStage(pipeline.StageNameCoder, coderResult.OutputFiles)
	sessionMgr.Save(sess)
	return nil
}

// runCommitStage runs the commit/repo update stage.
func runCommitStage(sess *session.Session, sessionMgr *session.Manager, ctx *pipeline.StageContext, projectDir string) error {
	fmt.Printf("  %s Commit...\n", orcProgressStyle.Render("→"))
	repoStage := pipeline.NewRepoUpdateStage(projectDir)
	sess.SetCurrentStage(pipeline.StageNameRepoUpdate, 1)
	sessionMgr.Save(sess)

	ctx.Iteration = 1
	repoResult := repoStage.Execute(ctx)

	// Collect issues
	for _, issue := range repoResult.Issues {
		sess.AddIssue(issue)
	}

	if repoResult.Status == pipeline.StageStatusFailed {
		sess.FailStage(pipeline.StageNameRepoUpdate, repoResult.Error.Error())
		sessionMgr.Save(sess)
		return fmt.Errorf("commit failed: %w", repoResult.Error)
	} else if repoResult.Status == pipeline.StageStatusNeedsRevision {
		fmt.Printf("    %s %s\n", orcWarningStyle.Render("⚠"), repoResult.Feedback)
	} else {
		fmt.Printf("    %s %s\n", orcSuccessStyle.Render("✓"), repoResult.Notes)
	}
	sess.CompleteStage(pipeline.StageNameRepoUpdate, repoResult.OutputFiles)
	return nil
}

// runCoderReviewerLoop runs the Coder ↔ Reviewer feedback loop.
func runCoderReviewerLoop(coderStage *pipeline.CoderStage, reviewerStage *pipeline.ReviewerStage, ctx *pipeline.StageContext, sess *session.Session, sessionMgr *session.Manager, config *pipeline.Config, verbose bool) (*pipeline.StageResult, error) {
	iteration := 1
	var feedback string

	for {
		// Run Coder
		fmt.Printf("  %s Coder", orcProgressStyle.Render("→"))
		if iteration > 1 {
			fmt.Printf(" (iteration %d)", iteration)
		}
		fmt.Println("...")

		sess.SetCurrentStage(pipeline.StageNameCoder, iteration)
		sessionMgr.Save(sess)

		ctx.Iteration = iteration
		ctx.PreviousFeedback = feedback

		coderResult := coderStage.Execute(ctx)

		// Collect issues
		for _, issue := range coderResult.Issues {
			sess.AddIssue(issue)
		}

		if coderResult.Status == pipeline.StageStatusFailed {
			sess.FailStage(pipeline.StageNameCoder, coderResult.Error.Error())
			sessionMgr.Save(sess)
			return coderResult, fmt.Errorf("coder failed: %w", coderResult.Error)
		}

		fmt.Printf("    %s Implementation complete\n", orcSuccessStyle.Render("✓"))

		// Run Reviewer
		fmt.Printf("  %s Reviewer", orcProgressStyle.Render("→"))
		if iteration > 1 {
			fmt.Printf(" (iteration %d)", iteration)
		}
		fmt.Println("...")

		sess.SetCurrentStage(pipeline.StageNameReviewer, iteration)
		sessionMgr.Save(sess)

		reviewResult := reviewerStage.Execute(ctx)

		// Collect issues
		for _, issue := range reviewResult.Issues {
			sess.AddIssue(issue)
		}

		switch reviewResult.Status {
		case pipeline.StageStatusSuccess:
			fmt.Printf("    %s Code approved\n", orcSuccessStyle.Render("✓"))
			sess.CompleteStage(pipeline.StageNameCoder, coderResult.OutputFiles)
			sess.CompleteStage(pipeline.StageNameReviewer, reviewResult.OutputFiles)
			sessionMgr.Save(sess)
			return coderResult, nil

		case pipeline.StageStatusNeedsRevision:
			if iteration >= config.MaxIterations {
				fmt.Printf("    %s Max iterations reached, escalating\n", orcWarningStyle.Render("⚠"))
				return &pipeline.StageResult{
					Status:   pipeline.StageStatusEscalate,
					Feedback: reviewResult.Feedback,
					Notes:    fmt.Sprintf("Max iterations (%d) reached", config.MaxIterations),
				}, nil
			}
			fmt.Printf("    %s Revision requested\n", orcWarningStyle.Render("⚠"))
			if verbose {
				fmt.Printf("    Feedback: %s\n", truncateFeedback(reviewResult.Feedback, 200))
			}
			feedback = reviewResult.Feedback
			iteration++

		case pipeline.StageStatusEscalate:
			fmt.Printf("    %s Escalated to human\n", orcErrorStyle.Render("✗"))
			sess.CompleteStage(pipeline.StageNameReviewer, reviewResult.OutputFiles)
			sessionMgr.Save(sess)
			return reviewResult, nil

		case pipeline.StageStatusFailed:
			sess.FailStage(pipeline.StageNameReviewer, reviewResult.Error.Error())
			sessionMgr.Save(sess)
			return reviewResult, fmt.Errorf("reviewer failed: %w", reviewResult.Error)
		}
	}
}

// runTesterCoderLoop runs the Tester ↔ Coder feedback loop when tests fail.
func runTesterCoderLoop(testerStage *pipeline.TesterStage, coderStage *pipeline.CoderStage, ctx *pipeline.StageContext, sess *session.Session, sessionMgr *session.Manager, config *pipeline.Config, verbose bool, initialFeedback string) (*pipeline.StageResult, error) {
	iteration := 1
	feedback := initialFeedback

	for {
		if iteration >= config.MaxIterations {
			fmt.Printf("    %s Max test iterations reached, escalating\n", orcWarningStyle.Render("⚠"))
			return &pipeline.StageResult{
				Status:   pipeline.StageStatusEscalate,
				Feedback: feedback,
				Notes:    fmt.Sprintf("Max test iterations (%d) reached", config.MaxIterations),
			}, nil
		}

		// Run Coder to fix tests
		fmt.Printf("  %s Coder (fixing tests, iteration %d)...\n", orcProgressStyle.Render("→"), iteration+1)

		sess.SetCurrentStage(pipeline.StageNameCoder, iteration+1)
		sessionMgr.Save(sess)

		ctx.Iteration = iteration + 1
		ctx.PreviousFeedback = feedback

		coderResult := coderStage.Execute(ctx)

		// Collect issues
		for _, issue := range coderResult.Issues {
			sess.AddIssue(issue)
		}

		if coderResult.Status == pipeline.StageStatusFailed {
			return coderResult, fmt.Errorf("coder failed while fixing tests: %w", coderResult.Error)
		}

		fmt.Printf("    %s Fix applied\n", orcSuccessStyle.Render("✓"))

		// Re-run Tester
		fmt.Printf("  %s Tester (re-run, iteration %d)...\n", orcProgressStyle.Render("→"), iteration+1)

		sess.SetCurrentStage(pipeline.StageNameTester, iteration+1)
		sessionMgr.Save(sess)

		ctx.Iteration = iteration + 1
		testerResult := testerStage.Execute(ctx)

		// Collect issues
		for _, issue := range testerResult.Issues {
			sess.AddIssue(issue)
		}

		switch testerResult.Status {
		case pipeline.StageStatusSuccess:
			return testerResult, nil

		case pipeline.StageStatusNeedsRevision:
			if verbose {
				fmt.Printf("    Feedback: %s\n", truncateFeedback(testerResult.Feedback, 200))
			}
			feedback = testerResult.Feedback
			iteration++

		case pipeline.StageStatusFailed:
			return testerResult, fmt.Errorf("tester failed: %w", testerResult.Error)
		}
	}
}

func truncateFeedback(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// getPipelineForTask returns the pipeline stages for a given task type.
func getPipelineForTask(taskType types.TaskType) []string {
	switch taskType {
	case types.TaskTypeCoding:
		return []string{"sanity", "coder", "reviewer", "tester", "commit"}
	case types.TaskTypeSimple:
		return []string{"sanity", "coder", "commit"}
	case types.TaskTypeResearch:
		return []string{"sanity", "researcher", "summary"}
	case types.TaskTypeAIPlanning:
		return []string{"sanity", "planner", "human-review"}
	case types.TaskTypeHumanInput:
		return []string{"notify"}
	default:
		// Fall back to coding pipeline for unknown types
		return []string{"sanity", "coder", "reviewer", "tester", "commit"}
	}
}

// getPipelineDescription returns a human-readable description of the pipeline for a task type.
func getPipelineDescription(taskType types.TaskType) string {
	switch taskType {
	case types.TaskTypeCoding:
		return "Sanity → Coder → Reviewer → Tester → Commit"
	case types.TaskTypeSimple:
		return "Sanity → Coder → Commit"
	case types.TaskTypeResearch:
		return "Sanity → Researcher → Summary"
	case types.TaskTypeAIPlanning:
		return "Sanity → Planner → Human Review (stages not implemented - will use coding pipeline)"
	case types.TaskTypeHumanInput:
		return "Notify → WaitHuman → CaptureResponse"
	default:
		return "Sanity → Coder → Reviewer → Tester → Commit (default)"
	}
}


// buildIndexMap creates a map of display index -> task ID for non-completed tasks.
func buildIndexMap(tasks []types.Task) map[int]string {
	indexMap := make(map[int]string)
	index := 1

	for _, t := range tasks {
		if t.State != types.TaskStateCompleted {
			indexMap[index] = t.ID
			index++
		}
	}

	return indexMap
}

// resolveTaskArgs converts indexes or IDs to task IDs.
func resolveTaskArgs(args []string, indexMap map[int]string, allTasks []types.Task) ([]string, error) {
	var taskIDs []string
	seen := make(map[string]bool)

	for _, arg := range args {
		var taskID string

		// Try as index first
		if idx, err := strconv.Atoi(arg); err == nil {
			if id, ok := indexMap[idx]; ok {
				taskID = id
			} else {
				return nil, fmt.Errorf("invalid task index: %d (use 'gc tasks' to see valid indexes)", idx)
			}
		} else if strings.HasPrefix(arg, "task_") {
			// Treat as task ID
			taskID = arg
			// Verify it exists
			found := false
			for _, t := range allTasks {
				if t.ID == taskID {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("task not found: %s", taskID)
			}
		} else {
			return nil, fmt.Errorf("invalid argument: %s (use index number or task_ID)", arg)
		}

		if !seen[taskID] {
			taskIDs = append(taskIDs, taskID)
			seen[taskID] = true
		}
	}

	return taskIDs, nil
}

// getTasksByIDs returns task objects for the given IDs.
func getTasksByIDs(allTasks []types.Task, taskIDs []string) []types.Task {
	var tasks []types.Task
	idSet := make(map[string]bool)
	for _, id := range taskIDs {
		idSet[id] = true
	}

	for _, t := range allTasks {
		if idSet[t.ID] {
			tasks = append(tasks, t)
		}
	}

	return tasks
}

// validateTasks checks that all tasks are valid for execution.
func validateTasks(tasks []types.Task) error {
	for _, t := range tasks {
		if t.State == types.TaskStateCompleted {
			return fmt.Errorf("task already completed: %s", t.Title)
		}
	}
	return nil
}

func findTask(tasks []types.Task, id string) *types.Task {
	for i := range tasks {
		if tasks[i].ID == id {
			return &tasks[i]
		}
	}
	return nil
}

func findTaskByID(tasks []types.Task, id string) *types.Task {
	for i := range tasks {
		if tasks[i].ID == id {
			return &tasks[i]
		}
	}
	return nil
}

// executeSessionParallel executes tasks in parallel according to their dependencies.
func executeSessionParallel(store *data.Store, sessionMgr *session.Manager, sess *session.Session, tasks []types.Task, claudeClient *claude.Client, pipelineConfig *pipeline.Config, verbose bool, customPipeline string) error {
	// Analyze dependencies to determine execution groups
	groups := pipeline.AnalyzeDependencies(tasks)

	// Create a task map for quick lookup
	taskMap := make(map[string]*types.Task)
	for i := range tasks {
		taskMap[tasks[i].ID] = &tasks[i]
	}

	// Execute each group (groups run sequentially, tasks within a group run in parallel)
	for groupIdx, group := range groups {
		if verbose {
			fmt.Printf("\n%s Executing wave %d (%d task(s))...\n",
				orcProgressStyle.Render("→"),
				groupIdx+1,
				len(group))
		}

		// Filter out already completed tasks (for resume)
		var tasksToRun []string
		for _, taskID := range group {
			if progress, ok := sess.TaskProgress[taskID]; ok {
				if progress.Status == "completed" {
					continue
				}
			}
			tasksToRun = append(tasksToRun, taskID)
		}

		if len(tasksToRun) == 0 {
			continue
		}

		// Check for blocked tasks
		var unblocked []string
		allTasks, _ := store.LoadTasks()
		for _, taskID := range tasksToRun {
			task := taskMap[taskID]
			if task == nil {
				continue
			}

			blocked := false
			for _, blockerID := range task.BlockedBy {
				blocker := findTaskByID(allTasks, blockerID)
				if blocker != nil && blocker.State != types.TaskStateCompleted {
					blocked = true
					break
				}
			}

			if blocked {
				fmt.Printf("%s Skipping %s (blocked by incomplete tasks)\n",
					orcWarningStyle.Render("⚠"),
					task.Title)
				continue
			}

			unblocked = append(unblocked, taskID)
		}

		if len(unblocked) == 0 {
			continue
		}

		// Execute tasks in this group (in parallel if more than one)
		if len(unblocked) == 1 {
			// Single task - execute directly
			task := taskMap[unblocked[0]]
			if err := executeTask(store, sessionMgr, sess, task, claudeClient, pipelineConfig, verbose, customPipeline); err != nil {
				if sess.TaskProgress[task.ID].Status == "escalated" {
					fmt.Printf("\n%s Task escalated to human. Session paused.\n",
						orcWarningStyle.Render("◷"))
					fmt.Println(orcDimStyle.Render("Use 'gc orc --continue' after providing input."))
					sess.UpdateStatus(session.StatusPaused)
					sessionMgr.Save(sess)
					return nil
				}
				sess.FailTask(task.ID, err.Error())
				sessionMgr.Save(sess)
				return fmt.Errorf("executing task %s: %w", task.ID, err)
			}
			sess.CompleteTask(task.ID)
			sessionMgr.Save(sess)
		} else {
			// Multiple tasks - execute in parallel
			type taskResult struct {
				taskID string
				err    error
			}

			results := make(chan taskResult, len(unblocked))

			for _, taskID := range unblocked {
				go func(tid string) {
					task := taskMap[tid]
					err := executeTask(store, sessionMgr, sess, task, claudeClient, pipelineConfig, verbose, customPipeline)
					results <- taskResult{taskID: tid, err: err}
				}(taskID)
			}

			// Collect results
			for i := 0; i < len(unblocked); i++ {
				result := <-results
				if result.err != nil {
					if sess.TaskProgress[result.taskID].Status == "escalated" {
						fmt.Printf("\n%s Task escalated to human. Session paused.\n",
							orcWarningStyle.Render("◷"))
						fmt.Println(orcDimStyle.Render("Use 'gc orc --continue' after providing input."))
						sess.UpdateStatus(session.StatusPaused)
						sessionMgr.Save(sess)
						return nil
					}
					sess.FailTask(result.taskID, result.err.Error())
					sessionMgr.Save(sess)
					return fmt.Errorf("executing task %s: %w", result.taskID, result.err)
				}
				sess.CompleteTask(result.taskID)
				sessionMgr.Save(sess)
			}
		}
	}

	// Run session review stage
	fmt.Println()
	fmt.Printf("  %s Session Review...\n", orcProgressStyle.Render("→"))
	reviewStage := pipeline.NewSessionReviewStage(store.GetDataDir())
	reviewCtx := &pipeline.StageContext{
		Session: sess,
		Verbose: verbose,
	}

	if !reviewStage.CanSkip(reviewCtx) {
		result := reviewStage.Execute(reviewCtx)
		if result.Status == pipeline.StageStatusFailed {
			fmt.Printf("    %s %v\n", orcWarningStyle.Render("⚠"), result.Error)
		} else {
			fmt.Printf("    %s %s\n", orcSuccessStyle.Render("✓"), result.Notes)
		}
	} else {
		fmt.Printf("    %s\n", orcDimStyle.Render("No issues to review"))
	}

	sess.UpdateStatus(session.StatusCompleted)
	if err := sessionMgr.Save(sess); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}

	fmt.Println()
	fmt.Println(orcSuccessStyle.Render("✓ All tasks completed successfully"))
	fmt.Println()

	// Show session summary
	fmt.Println(orcHeaderStyle.Render("Session Summary"))
	fmt.Printf("  Session ID: %s\n", sess.ID)
	fmt.Printf("  Tasks:      %d completed\n", len(tasks))
	if len(sess.Issues) > 0 {
		fmt.Printf("  Issues:     %d logged for self-learning\n", len(sess.Issues))
	}

	return nil
}
