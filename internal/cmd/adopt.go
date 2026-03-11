package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/registry"
	"github.com/mmariani/ground-control/internal/sidecar"
	"github.com/spf13/cobra"
)

// Adopt styles
var (
	adoptHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("99"))

	adoptSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("40"))

	adoptWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))

	adoptInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	adoptDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// NewAdoptCmd creates the adopt command for Flight Deck
func NewAdoptCmd() *cobra.Command {
	var skipAnalysis bool
	var forceAdopt bool
	var noBackup bool

	cmd := &cobra.Command{
		Use:   "adopt <path>",
		Short: "Adopt an existing project into Flight Deck",
		Long: `Adopt a project for Flight Deck management.

This command:
1. Analyzes the repository using Claude to detect tech stack
2. Creates a .gc/ sidecar directory with configuration
3. Registers the project in the global Flight Deck registry

Examples:
  gc adopt .                          # Adopt current directory
  gc adopt ~/Projects/my-app          # Adopt specific project
  gc adopt ~/Projects/my-app --force  # Re-adopt (overwrite existing)
  gc adopt ~/Projects/my-app --skip   # Skip analysis (manual setup)
  gc adopt ~/Projects/my-app --no-backup  # Skip backup (testing)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdopt(args[0], skipAnalysis, forceAdopt, noBackup)
		},
	}

	cmd.Flags().BoolVar(&skipAnalysis, "skip", false, "Skip Claude analysis (manual setup)")
	cmd.Flags().BoolVarP(&forceAdopt, "force", "f", false, "Force re-adoption (overwrite existing)")
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "Skip backup of existing .gc/ directory")

	return cmd
}

func runAdopt(path string, skipAnalysis, force, noBackup bool) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("path does not exist: %s", absPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", absPath)
	}

	// Check if already adopted
	mgr := sidecar.NewManager(absPath)
	if mgr.Exists() && !force {
		return fmt.Errorf("project already adopted (use --force to re-adopt): %s", absPath)
	}

	fmt.Println(adoptHeaderStyle.Render("═══ Flight Deck Adoption ═══"))
	fmt.Printf("%s\n\n", adoptDimStyle.Render(absPath))

	// Backup existing .gc/ directory if it exists and not skipping
	if mgr.Exists() && !noBackup {
		if err := backupGCDir(absPath); err != nil {
			fmt.Printf("%s\n", adoptWarningStyle.Render("Warning: backup failed: "+err.Error()))
		}
	}

	var analysis *sidecar.AnalysisResult

	if skipAnalysis {
		// Create minimal analysis
		projectName := filepath.Base(absPath)
		analysis = &sidecar.AnalysisResult{
			Name:        projectName,
			Description: "Manually adopted project",
		}
		fmt.Println(adoptInfoStyle.Render("Skipping analysis (manual setup mode)"))
	} else {
		// Check if analysis.json already exists (reuse if present)
		existingAnalysis, err := mgr.LoadAnalysis()
		if err == nil && existingAnalysis.Name != "" {
			fmt.Println(adoptInfoStyle.Render("Found existing analysis.json, reusing..."))
			analysis = existingAnalysis
		} else {
			// Run Claude analysis
			fmt.Println(adoptInfoStyle.Render("Analyzing repository with Claude..."))
			fmt.Println()

			analysis, err = runAnalysis(absPath)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}
		}

		// Display results
		displayAnalysis(analysis)

		// Check confidence levels
		if hasLowConfidence(analysis) {
			fmt.Println()
			fmt.Println(adoptWarningStyle.Render("⚠️  Some detections have low confidence. Continue? [y/N] "))
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return fmt.Errorf("adoption cancelled")
			}
		}
	}

	// Create project config from analysis
	fmt.Println()
	fmt.Println(adoptInfoStyle.Render("Creating .gc/ sidecar..."))

	config := sidecar.CreateConfigFromAnalysis(analysis)
	if err := mgr.SaveConfig(config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Create initial state
	state := &sidecar.ProjectState{
		Session:  sidecar.SessionInfo{Status: "idle"},
		WorkMode: sidecar.WorkModeAssisted, // Default to assisted mode
	}
	if err := mgr.SaveState(state); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	// Create Flight Deck onboarding and state files
	if err := createFDFiles(absPath); err != nil {
		fmt.Printf("%s\n", adoptWarningStyle.Render("Warning: could not create FD files: "+err.Error()))
	}

	// Generate CLAUDE.md
	if err := generateClaudeMd(absPath, analysis); err != nil {
		fmt.Printf("%s\n", adoptWarningStyle.Render("Warning: could not generate CLAUDE.md: "+err.Error()))
	}

	// Register in global registry
	reg, err := registry.NewRegistry()
	if err != nil {
		return fmt.Errorf("init registry: %w", err)
	}
	if err := reg.AddProject(absPath, analysis.Name); err != nil {
		return fmt.Errorf("register project: %w", err)
	}

	// Success
	fmt.Println()
	fmt.Println(adoptSuccessStyle.Render("✓ Successfully adopted: " + analysis.Name))
	fmt.Printf("  %s\n", adoptDimStyle.Render("Path: "+absPath))
	fmt.Printf("  %s\n", adoptDimStyle.Render("Sidecar: "+mgr.GCPath()))
	fmt.Println()
	fmt.Println(adoptDimStyle.Render("Run 'gc tui' to open Flight Deck"))

	return nil
}

func runAnalysis(projectPath string) (*sidecar.AnalysisResult, error) {
	// Ensure .gc directory exists
	gcPath := filepath.Join(projectPath, ".gc")
	if err := os.MkdirAll(gcPath, 0755); err != nil {
		return nil, err
	}

	analysisPath := filepath.Join(gcPath, "analysis.json")

	// Build the analysis prompt
	prompt := buildAnalysisPrompt(analysisPath)

	// Run Claude
	cmd := exec.Command("claude", "--print", prompt)
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude analysis failed: %w", err)
	}

	// Read the analysis result
	data, err := os.ReadFile(analysisPath)
	if err != nil {
		return nil, fmt.Errorf("read analysis: %w", err)
	}

	var result sidecar.AnalysisResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse analysis: %w", err)
	}

	return &result, nil
}

func buildAnalysisPrompt(outputPath string) string {
	return fmt.Sprintf(`You are part of Flight Deck, an AI development orchestration system.

Analyze this repository and output JSON to %s with this structure:

{
  "name": "project-name",
  "description": "Brief description",
  "languages": [{"name": "TypeScript", "confidence": 0.95, "version": "5.0"}],
  "frameworks": [{"name": "React Native", "confidence": 0.90}],
  "test_runner": {"name": "Jest", "confidence": 0.85, "config_file": "jest.config.js"},
  "package_manager": "npm",
  "ci_system": "github-actions",
  "existing_conventions": ["ESLint", "Prettier", "Husky hooks"],
  "existing_task_management": "None detected",
  "key_files": [{"path": "src/index.ts", "purpose": "Entry point"}],
  "suggested_constraints": ["Run lint before commit", "Use existing patterns"],
  "monorepo": false,
  "workspaces": []
}

Check: package.json, requirements.txt, go.mod, tsconfig.json, .eslintrc*, .github/workflows/, Makefile, README.md, and directory structure.

Confidence: 0.9+ = explicit declaration, 0.7-0.9 = strong indicators, 0.5-0.7 = inferred, <0.5 = uncertain.

Output ONLY valid JSON to the file. No other text.`, outputPath)
}

func displayAnalysis(a *sidecar.AnalysisResult) {
	fmt.Println(adoptHeaderStyle.Render("Analysis Results"))
	fmt.Println()

	// Name and description
	fmt.Printf("  %s %s\n", adoptInfoStyle.Render("Name:"), a.Name)
	if a.Description != "" {
		fmt.Printf("  %s %s\n", adoptInfoStyle.Render("Description:"), a.Description)
	}
	fmt.Println()

	// Languages
	if len(a.Languages) > 0 {
		fmt.Printf("  %s\n", adoptInfoStyle.Render("Languages:"))
		for _, l := range a.Languages {
			confidence := formatConfidence(l.Confidence)
			version := ""
			if l.Version != "" {
				version = " (" + l.Version + ")"
			}
			fmt.Printf("    • %s%s %s\n", l.Name, version, confidence)
		}
	}

	// Frameworks
	if len(a.Frameworks) > 0 {
		fmt.Printf("  %s\n", adoptInfoStyle.Render("Frameworks:"))
		for _, f := range a.Frameworks {
			confidence := formatConfidence(f.Confidence)
			fmt.Printf("    • %s %s\n", f.Name, confidence)
		}
	}

	// Test runner
	if a.TestRunner != nil {
		confidence := formatConfidence(a.TestRunner.Confidence)
		fmt.Printf("  %s %s %s\n", adoptInfoStyle.Render("Test Runner:"), a.TestRunner.Name, confidence)
	}

	// Package manager & CI
	if a.PackageManager != "" {
		fmt.Printf("  %s %s\n", adoptInfoStyle.Render("Package Manager:"), a.PackageManager)
	}
	if a.CISystem != "" {
		fmt.Printf("  %s %s\n", adoptInfoStyle.Render("CI System:"), a.CISystem)
	}

	// Conventions
	if len(a.ExistingConventions) > 0 {
		fmt.Printf("  %s %s\n", adoptInfoStyle.Render("Conventions:"),
			strings.Join(a.ExistingConventions, ", "))
	}
}

func formatConfidence(c float64) string {
	if c >= 0.9 {
		return adoptSuccessStyle.Render(fmt.Sprintf("%.0f%%", c*100))
	} else if c >= 0.7 {
		return adoptInfoStyle.Render(fmt.Sprintf("%.0f%%", c*100))
	} else {
		return adoptWarningStyle.Render(fmt.Sprintf("%.0f%% ⚠️", c*100))
	}
}

func hasLowConfidence(a *sidecar.AnalysisResult) bool {
	for _, l := range a.Languages {
		if l.Confidence < 0.7 {
			return true
		}
	}
	for _, f := range a.Frameworks {
		if f.Confidence < 0.7 {
			return true
		}
	}
	if a.TestRunner != nil && a.TestRunner.Confidence < 0.7 {
		return true
	}
	return false
}

func generateClaudeMd(projectPath string, a *sidecar.AnalysisResult) error {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s - Flight Deck Project\n\n", a.Name))

	if a.Description != "" {
		b.WriteString(fmt.Sprintf("%s\n\n", a.Description))
	}

	b.WriteString("## Tech Stack\n\n")
	for _, l := range a.Languages {
		version := ""
		if l.Version != "" {
			version = " " + l.Version
		}
		b.WriteString(fmt.Sprintf("- %s%s\n", l.Name, version))
	}
	for _, f := range a.Frameworks {
		b.WriteString(fmt.Sprintf("- %s\n", f.Name))
	}
	if a.TestRunner != nil {
		b.WriteString(fmt.Sprintf("- Tests: %s\n", a.TestRunner.Name))
	}
	b.WriteString("\n")

	if len(a.ExistingConventions) > 0 {
		b.WriteString("## Conventions\n\n")
		for _, c := range a.ExistingConventions {
			b.WriteString(fmt.Sprintf("- %s\n", c))
		}
		b.WriteString("\n")
	}

	if len(a.SuggestedConstraints) > 0 {
		b.WriteString("## Constraints\n\n")
		for _, c := range a.SuggestedConstraints {
			b.WriteString(fmt.Sprintf("- %s\n", c))
		}
		b.WriteString("\n")
	}

	if len(a.KeyFiles) > 0 {
		b.WriteString("## Key Files\n\n")
		for _, f := range a.KeyFiles {
			b.WriteString(fmt.Sprintf("- `%s` - %s\n", f.Path, f.Purpose))
		}
		b.WriteString("\n")
	}

	// Flight Deck Integration section
	b.WriteString("## Flight Deck Integration\n\n")
	b.WriteString("This project is managed by Flight Deck. See `.gc/fd-onboarding.md` for full details.\n\n")

	b.WriteString("### Slash Commands\n\n")
	b.WriteString("| Command | Purpose |\n")
	b.WriteString("|---------|---------||\n")
	b.WriteString("| `/roadmap-item \"title\"` | Add feature to roadmap |\n")
	b.WriteString("| `/issue \"description\"` | Report bug or issue |\n")
	b.WriteString("| `/start-work <id>` | Begin work on item |\n")
	b.WriteString("| `/progress <pct>` | Update completion % |\n")
	b.WriteString("| `/commit \"message\"` | Request commit via FD |\n")
	b.WriteString("| `/complete` | Mark current work done |\n")
	b.WriteString("| `/learn <type> \"desc\"` | Log process learning |\n")
	b.WriteString("\n")

	b.WriteString("### Key Files\n\n")
	b.WriteString("| File | Purpose |\n")
	b.WriteString("|------|---------||\n")
	b.WriteString("| `.gc/roadmap.json` | Feature roadmap |\n")
	b.WriteString("| `.gc/issues.json` | Bugs and issues |\n")
	b.WriteString("| `.gc/requests.jsonl` | Requests to FD |\n")
	b.WriteString("| `.gc/state.json` | Session state |\n")
	b.WriteString("\n")

	b.WriteString("### Important\n\n")
	b.WriteString("- **Don't commit directly** - Use `/commit` to request FD handle it\n")
	b.WriteString("- **Check inbox on startup** - Look in `.gc/inbox/` for dispatched work\n")
	b.WriteString("- **Update progress** - Use `/progress` periodically\n")
	b.WriteString("- **Log learnings** - Use `/learn friction \"desc\"` for process issues\n")
	b.WriteString("\n")

	claudePath := filepath.Join(projectPath, ".gc", "CLAUDE.md")
	return os.WriteFile(claudePath, []byte(b.String()), 0644)
}

// createFDFiles creates Flight Deck onboarding and state files
func createFDFiles(projectPath string) error {
	gcPath := filepath.Join(projectPath, ".gc")

	// Create inbox directory
	inboxPath := filepath.Join(gcPath, "inbox")
	if err := os.MkdirAll(inboxPath, 0755); err != nil {
		return fmt.Errorf("create inbox: %w", err)
	}

	// Copy fd-onboarding.md template
	// First try to find the template relative to the GC data dir
	templatePath := findTemplate("fd-onboarding.md")
	if templatePath != "" {
		content, err := os.ReadFile(templatePath)
		if err == nil {
			onboardingPath := filepath.Join(gcPath, "fd-onboarding.md")
			if err := os.WriteFile(onboardingPath, content, 0644); err != nil {
				return fmt.Errorf("write onboarding: %w", err)
			}
		}
	} else {
		// Generate inline if template not found
		if err := generateOnboardingInline(gcPath); err != nil {
			return fmt.Errorf("generate onboarding: %w", err)
		}
	}

	// Create empty issues.json
	issuesPath := filepath.Join(gcPath, "issues.json")
	issuesContent := `{"issues": []}`
	if err := os.WriteFile(issuesPath, []byte(issuesContent), 0644); err != nil {
		return fmt.Errorf("write issues: %w", err)
	}

	// Create empty roadmap.json
	roadmapPath := filepath.Join(gcPath, "roadmap.json")
	roadmapContent := `{"features": [], "milestones": []}`
	if err := os.WriteFile(roadmapPath, []byte(roadmapContent), 0644); err != nil {
		return fmt.Errorf("write roadmap: %w", err)
	}

	// Create empty learning.jsonl
	learningPath := filepath.Join(gcPath, "learning.jsonl")
	if err := os.WriteFile(learningPath, []byte(""), 0644); err != nil {
		return fmt.Errorf("write learning: %w", err)
	}

	// Create empty requests.jsonl
	requestsPath := filepath.Join(gcPath, "requests.jsonl")
	if err := os.WriteFile(requestsPath, []byte(""), 0644); err != nil {
		return fmt.Errorf("write requests: %w", err)
	}

	return nil
}

// findTemplate looks for a template file in known locations
func findTemplate(name string) string {
	// Try relative to current working directory (for development)
	cwd, _ := os.Getwd()
	paths := []string{
		filepath.Join(cwd, "templates", name),
		filepath.Join(GetDataDir(), "..", "templates", name),
		filepath.Join(os.Getenv("HOME"), "Projects", "ground-control", "templates", name),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// generateOnboardingInline creates the onboarding file with inline content
func generateOnboardingInline(gcPath string) error {
	content := `# Flight Deck Integration

This project is managed by **Flight Deck (FD)**, part of Ground Control (GC).

## What is Flight Deck?

Flight Deck is a central orchestration dashboard that:
- Manages multiple projects from a single TUI
- Coordinates non-coding work (docs, commits, reviews, deployments)
- Tracks issues, roadmaps, and sprints across projects
- Provides visibility into all project states

## Your Responsibilities (Project Claude)

1. **Coding & Testing** - You handle all code implementation and testing
2. **Issue Tracking** - Add issues to .gc/issues.json as you find them
3. **Progress Updates** - Update .gc/state.json with session activity
4. **Request FD Help** - Use .gc/requests.jsonl when you need:
   - Documentation written
   - Code reviewed
   - Commits made
   - Design decisions

## FD's Responsibilities

FD handles:
- Commits and git operations
- Documentation generation
- Code reviews
- Cross-project coordination
- Sprint/roadmap management

## How to Request FD Work

Append to .gc/requests.jsonl:
{"type":"commit","summary":"Add feature X","files":["src/x.go"],"at":"...","status":"pending"}
{"type":"review","summary":"Review implementation","at":"...","status":"pending"}

## Reporting Process Issues

If this workflow causes friction, note it in .gc/learning.jsonl:
{"type":"friction","actor":"proj_cc","summary":"Process unclear","detail":"...","at":"..."}

## Key Files

- .gc/state.json - Your session state
- .gc/issues.json - Project issues
- .gc/roadmap.json - Project roadmap
- .gc/requests.jsonl - Requests to FD
- .gc/learning.jsonl - Process improvement notes
`
	onboardingPath := filepath.Join(gcPath, "fd-onboarding.md")
	return os.WriteFile(onboardingPath, []byte(content), 0644)
}

// backupGCDir creates a backup of the .gc/ directory before modification
func backupGCDir(projectPath string) error {
	gcPath := filepath.Join(projectPath, ".gc")

	// Check if .gc/ exists
	if _, err := os.Stat(gcPath); os.IsNotExist(err) {
		return nil // Nothing to backup
	}

	// Create backup directory with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(gcPath, fmt.Sprintf("backup_%s", timestamp))

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}

	// Files to backup
	filesToBackup := []string{
		"project.json",
		"state.json",
		"CLAUDE.md",
		"fd-onboarding.md",
		"issues.json",
		"roadmap.json",
		"analysis.json",
		"learning.jsonl",
		"requests.jsonl",
	}

	backedUpCount := 0
	for _, filename := range filesToBackup {
		srcPath := filepath.Join(gcPath, filename)
		dstPath := filepath.Join(backupPath, filename)

		// Skip if file doesn't exist
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}

		// Copy file
		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy %s: %w", filename, err)
		}
		backedUpCount++
	}

	if backedUpCount > 0 {
		relBackupPath := filepath.Join(".gc", fmt.Sprintf("backup_%s", timestamp))
		fmt.Printf("%s\n", adoptInfoStyle.Render(fmt.Sprintf("Backed up %d files to %s", backedUpCount, relBackupPath)))
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Preserve file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}
