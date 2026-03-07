// Package context implements context bundle creation for tasks.
package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mmariani/ground-control/internal/types"
)

// Builder creates context bundles for tasks.
type Builder struct {
	dataDir    string
	projectDir string
}

// NewBuilder creates a new context bundle builder.
func NewBuilder(dataDir, projectDir string) *Builder {
	return &Builder{
		dataDir:    dataDir,
		projectDir: projectDir,
	}
}

// BuildBundle creates a context bundle for the given task.
func (b *Builder) BuildBundle(task *types.Task) (*types.ContextBundle, error) {
	bundlePath := filepath.Join(b.dataDir, "context", task.ID)

	// Create bundle directory
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		return nil, fmt.Errorf("creating bundle directory: %w", err)
	}

	// Create relevant_code subdirectory
	codeDir := filepath.Join(bundlePath, "relevant_code")
	if err := os.MkdirAll(codeDir, 0755); err != nil {
		return nil, fmt.Errorf("creating relevant_code directory: %w", err)
	}

	bundle := &types.ContextBundle{
		BuiltAt:    time.Now(),
		BuiltBy:    "context-manager",
		BundlePath: bundlePath,
		Files:      types.ContextBundleFiles{},
	}

	// Build each component
	if err := b.buildRequirements(task, bundlePath, bundle); err != nil {
		return nil, fmt.Errorf("building requirements: %w", err)
	}

	if err := b.buildRelevantCode(task, bundlePath, bundle); err != nil {
		return nil, fmt.Errorf("building relevant code: %w", err)
	}

	if err := b.buildProjectContext(task, bundlePath, bundle); err != nil {
		return nil, fmt.Errorf("building project context: %w", err)
	}

	if err := b.buildPatterns(task, bundlePath, bundle); err != nil {
		return nil, fmt.Errorf("building patterns: %w", err)
	}

	if err := b.buildDecisions(task, bundlePath, bundle); err != nil {
		return nil, fmt.Errorf("building decisions: %w", err)
	}

	if err := b.buildTestHints(task, bundlePath, bundle); err != nil {
		return nil, fmt.Errorf("building test hints: %w", err)
	}

	return bundle, nil
}

func (b *Builder) buildRequirements(task *types.Task, bundlePath string, bundle *types.ContextBundle) error {
	path := filepath.Join(bundlePath, "requirements.md")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Requirements for: %s\n\n", task.Title))

	sb.WriteString("## Description\n\n")
	sb.WriteString(task.Description)
	sb.WriteString("\n\n")

	if len(task.Context.Requirements) > 0 {
		sb.WriteString("## Must Have\n\n")
		for _, req := range task.Context.Requirements {
			sb.WriteString(fmt.Sprintf("- %s\n", req))
		}
		sb.WriteString("\n")
	}

	if len(task.Context.Constraints) > 0 {
		sb.WriteString("## Constraints\n\n")
		for _, c := range task.Context.Constraints {
			sb.WriteString(fmt.Sprintf("- %s\n", c))
		}
		sb.WriteString("\n")
	}

	if task.Context.Background != "" {
		sb.WriteString("## Background\n\n")
		sb.WriteString(task.Context.Background)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Acceptance Criteria\n\n")
	if len(task.Context.Requirements) > 0 {
		for i, req := range task.Context.Requirements {
			sb.WriteString(fmt.Sprintf("- [ ] Requirement %d is met: %s\n", i+1, truncate(req, 50)))
		}
	}
	if task.Verification.Type == types.VerificationTestPass && task.Verification.Command != nil {
		sb.WriteString(fmt.Sprintf("- [ ] Verification passes: `%s`\n", *task.Verification.Command))
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return err
	}

	bundle.Files.Requirements = path
	return nil
}

func (b *Builder) buildRelevantCode(task *types.Task, bundlePath string, bundle *types.ContextBundle) error {
	codeDir := filepath.Join(bundlePath, "relevant_code")
	var codePaths []string

	// Extract file paths from task outputs
	for _, output := range task.Outputs {
		if output.Path != "" && !strings.HasPrefix(output.Path, "~") {
			fullPath := filepath.Join(b.projectDir, output.Path)
			if _, err := os.Stat(fullPath); err == nil {
				snippetPath, err := b.extractCodeSnippet(fullPath, codeDir, output.Description)
				if err == nil && snippetPath != "" {
					codePaths = append(codePaths, snippetPath)
				}
			}
		}
	}

	// Extract file paths from related tasks
	for _, relatedID := range task.Context.RelatedTasks {
		relatedTask, err := b.loadTask(relatedID)
		if err != nil {
			continue
		}
		for _, output := range relatedTask.Outputs {
			if output.Path != "" && output.Exists && !strings.HasPrefix(output.Path, "~") {
				fullPath := filepath.Join(b.projectDir, output.Path)
				if _, err := os.Stat(fullPath); err == nil {
					snippetPath, err := b.extractCodeSnippet(fullPath, codeDir, "From related task: "+relatedTask.Title)
					if err == nil && snippetPath != "" {
						codePaths = append(codePaths, snippetPath)
					}
				}
			}
		}
	}

	// Look for code patterns in description/requirements
	patterns := extractFilePatterns(task.Description)
	for _, pattern := range task.Context.Requirements {
		patterns = append(patterns, extractFilePatterns(pattern)...)
	}

	for _, pattern := range patterns {
		fullPath := filepath.Join(b.projectDir, pattern)
		if _, err := os.Stat(fullPath); err == nil {
			snippetPath, err := b.extractCodeSnippet(fullPath, codeDir, "Referenced in requirements")
			if err == nil && snippetPath != "" {
				codePaths = append(codePaths, snippetPath)
			}
		}
	}

	bundle.Files.RelevantCode = codePaths
	return nil
}

func (b *Builder) extractCodeSnippet(sourcePath, codeDir, reason string) (string, error) {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}

	// For small files, include the whole thing (but mark line range)
	lines := strings.Split(string(content), "\n")
	lineCount := len(lines)

	var snippet string
	var lineRange string

	if lineCount <= 100 {
		// Small file, include all
		snippet = string(content)
		lineRange = fmt.Sprintf("1-%d", lineCount)
	} else {
		// Large file, extract first 50 lines as a start
		// In a real implementation, we'd use AST parsing to find relevant functions
		endLine := 50
		if endLine > lineCount {
			endLine = lineCount
		}
		snippet = strings.Join(lines[:endLine], "\n")
		lineRange = fmt.Sprintf("1-%d", endLine)
	}

	// Create snippet file
	baseName := filepath.Base(sourcePath)
	safeName := strings.ReplaceAll(baseName, "/", "_")
	snippetName := fmt.Sprintf("%s_%s.txt", safeName, strings.ReplaceAll(lineRange, "-", "_"))
	snippetPath := filepath.Join(codeDir, snippetName)

	header := fmt.Sprintf("// From: %s:%s\n// Why: %s\n\n", sourcePath, lineRange, reason)
	if err := os.WriteFile(snippetPath, []byte(header+snippet), 0644); err != nil {
		return "", err
	}

	return snippetPath, nil
}

func (b *Builder) buildProjectContext(task *types.Task, bundlePath string, bundle *types.ContextBundle) error {
	path := filepath.Join(bundlePath, "project_context.md")

	var sb strings.Builder
	sb.WriteString("# Project Context\n\n")

	// Try to read CLAUDE.md
	claudePath := filepath.Join(b.projectDir, "CLAUDE.md")
	if content, err := os.ReadFile(claudePath); err == nil {
		sb.WriteString("## From CLAUDE.md\n\n")
		// Extract relevant sections (first 100 lines or so)
		lines := strings.Split(string(content), "\n")
		maxLines := 100
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		sb.WriteString(strings.Join(lines[:maxLines], "\n"))
		sb.WriteString("\n\n")
	}

	// Try to read README.md
	readmePath := filepath.Join(b.projectDir, "README.md")
	if content, err := os.ReadFile(readmePath); err == nil {
		sb.WriteString("## From README.md\n\n")
		lines := strings.Split(string(content), "\n")
		maxLines := 50
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		sb.WriteString(strings.Join(lines[:maxLines], "\n"))
		sb.WriteString("\n\n")
	}

	// Add task-specific context
	if task.ProjectID != nil {
		sb.WriteString(fmt.Sprintf("## Project ID\n\n%s\n\n", *task.ProjectID))
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return err
	}

	bundle.Files.ProjectContext = path
	return nil
}

func (b *Builder) buildPatterns(task *types.Task, bundlePath string, bundle *types.ContextBundle) error {
	path := filepath.Join(bundlePath, "patterns.md")

	var sb strings.Builder
	sb.WriteString("# Project Patterns\n\n")

	// Detect language and add appropriate patterns
	goModPath := filepath.Join(b.projectDir, "go.mod")
	packageJsonPath := filepath.Join(b.projectDir, "package.json")

	if _, err := os.Stat(goModPath); err == nil {
		sb.WriteString("## Go Project\n\n")
		sb.WriteString("### Code Style\n")
		sb.WriteString("- Use `gofmt` for formatting\n")
		sb.WriteString("- Error handling: return errors, don't panic\n")
		sb.WriteString("- Naming: CamelCase for exports, camelCase for internal\n\n")

		sb.WriteString("### Error Handling\n")
		sb.WriteString("- Return `(result, error)` tuples\n")
		sb.WriteString("- Wrap errors with context: `fmt.Errorf(\"doing X: %w\", err)`\n")
		sb.WriteString("- Check errors immediately after function calls\n\n")

		sb.WriteString("### Testing\n")
		sb.WriteString("- Test files: `*_test.go` in same package\n")
		sb.WriteString("- Use `testing.T` for tests\n")
		sb.WriteString("- Table-driven tests preferred\n\n")
	} else if _, err := os.Stat(packageJsonPath); err == nil {
		sb.WriteString("## Node.js Project\n\n")
		sb.WriteString("### Code Style\n")
		sb.WriteString("- Use TypeScript where possible\n")
		sb.WriteString("- Async/await for promises\n\n")

		sb.WriteString("### Error Handling\n")
		sb.WriteString("- Use try/catch for async operations\n")
		sb.WriteString("- Throw descriptive errors\n\n")

		sb.WriteString("### Testing\n")
		sb.WriteString("- Test files: `*.test.ts` or `*.spec.ts`\n\n")
	}

	// Add any patterns from task context
	if task.Context.Background != "" && strings.Contains(strings.ToLower(task.Context.Background), "pattern") {
		sb.WriteString("## Task-Specific Patterns\n\n")
		sb.WriteString(task.Context.Background)
		sb.WriteString("\n\n")
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return err
	}

	bundle.Files.Patterns = path
	return nil
}

func (b *Builder) buildDecisions(task *types.Task, bundlePath string, bundle *types.ContextBundle) error {
	path := filepath.Join(bundlePath, "decisions.md")

	var sb strings.Builder
	sb.WriteString("# Relevant Decisions\n\n")

	// Try to load activity log and find relevant decisions
	activityPath := filepath.Join(b.dataDir, "activity-log.json")
	if content, err := os.ReadFile(activityPath); err == nil {
		var activityFile struct {
			Events []types.ActivityEvent `json:"events"`
		}
		if err := json.Unmarshal(content, &activityFile); err == nil {
			// Find decision events that might be relevant
			for _, event := range activityFile.Events {
				if event.Type != "decision_made" {
					continue
				}

				// Check if decision is related to this task or its tags
				relevant := false
				if event.TaskID != nil {
					for _, relatedID := range task.Context.RelatedTasks {
						if *event.TaskID == relatedID {
							relevant = true
							break
						}
					}
				}

				// Check for keyword matches
				if !relevant {
					summaryLower := strings.ToLower(event.Summary)
					for _, tag := range task.Tags {
						if strings.Contains(summaryLower, strings.ToLower(tag)) {
							relevant = true
							break
						}
					}
				}

				if relevant {
					sb.WriteString(fmt.Sprintf("## %s\n\n", event.Summary))
					sb.WriteString(fmt.Sprintf("**When**: %s\n\n", event.Timestamp.Format("2006-01-02")))
					if event.Reasoning != nil {
						sb.WriteString(fmt.Sprintf("**Why**: %s\n\n", *event.Reasoning))
					}
					if len(event.DecisionFactors) > 0 {
						sb.WriteString("**Factors**:\n")
						for _, factor := range event.DecisionFactors {
							sb.WriteString(fmt.Sprintf("- %s\n", factor))
						}
						sb.WriteString("\n")
					}
					sb.WriteString("---\n\n")
				}
			}
		}
	}

	// If no decisions found, note that
	if sb.Len() < 50 {
		sb.WriteString("No directly relevant past decisions found in activity log.\n\n")
		sb.WriteString("Check `data/activity-log.json` for full decision history.\n")
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return err
	}

	bundle.Files.Decisions = path
	return nil
}

func (b *Builder) buildTestHints(task *types.Task, bundlePath string, bundle *types.ContextBundle) error {
	path := filepath.Join(bundlePath, "test_hints.md")

	var sb strings.Builder
	sb.WriteString("# Test Hints\n\n")

	sb.WriteString("## Happy Path\n\n")
	for _, req := range task.Context.Requirements {
		sb.WriteString(fmt.Sprintf("- [ ] Test: %s\n", truncate(req, 60)))
	}
	if len(task.Context.Requirements) == 0 {
		sb.WriteString("- [ ] Test basic functionality works as expected\n")
	}
	sb.WriteString("\n")

	sb.WriteString("## Edge Cases\n\n")
	sb.WriteString("- [ ] Empty input handling\n")
	sb.WriteString("- [ ] Invalid input handling\n")
	sb.WriteString("- [ ] Boundary values (max, min, zero)\n")
	sb.WriteString("- [ ] Null/nil handling\n\n")

	sb.WriteString("## Error Handling\n\n")
	sb.WriteString("- [ ] Graceful failure on bad input\n")
	sb.WriteString("- [ ] Proper error messages returned\n")
	sb.WriteString("- [ ] No panics or crashes\n\n")

	// Add type-specific hints
	switch task.Type {
	case types.TaskTypeCoding:
		sb.WriteString("## Code-Specific\n\n")
		sb.WriteString("- [ ] Functions are testable in isolation\n")
		sb.WriteString("- [ ] Side effects are controlled/mockable\n")
		sb.WriteString("- [ ] Public API behaves correctly\n\n")
	case types.TaskTypeResearch:
		sb.WriteString("## Research-Specific\n\n")
		sb.WriteString("- [ ] Sources are documented\n")
		sb.WriteString("- [ ] Conclusions are supported by evidence\n")
		sb.WriteString("- [ ] Alternatives are considered\n\n")
	}

	sb.WriteString("## Security Considerations\n\n")
	sb.WriteString("- [ ] Input validation prevents injection\n")
	sb.WriteString("- [ ] No sensitive data exposed in logs/errors\n")
	sb.WriteString("- [ ] Authentication/authorization checks (if applicable)\n")

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return err
	}

	bundle.Files.TestHints = path
	return nil
}

func (b *Builder) loadTask(taskID string) (*types.Task, error) {
	tasksPath := filepath.Join(b.dataDir, "tasks.json")
	content, err := os.ReadFile(tasksPath)
	if err != nil {
		return nil, err
	}

	var tasksFile struct {
		Tasks []types.Task `json:"tasks"`
	}
	if err := json.Unmarshal(content, &tasksFile); err != nil {
		return nil, err
	}

	for _, t := range tasksFile.Tasks {
		if t.ID == taskID {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("task not found: %s", taskID)
}

// Helper functions

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func extractFilePatterns(text string) []string {
	// Look for file path patterns in text
	patterns := []string{}

	// Match paths like internal/cmd/tasks.go
	re := regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_/]*\.[a-z]{2,4}`)
	matches := re.FindAllString(text, -1)
	for _, m := range matches {
		if strings.Contains(m, "/") || strings.HasSuffix(m, ".go") || strings.HasSuffix(m, ".ts") || strings.HasSuffix(m, ".md") {
			patterns = append(patterns, m)
		}
	}

	return patterns
}

// ReadBundleFiles reads all files from a context bundle into a map.
func ReadBundleFiles(bundle *types.ContextBundle) (map[string]string, error) {
	files := make(map[string]string)

	readFile := func(path, name string) {
		if path == "" {
			return
		}
		content, err := os.ReadFile(path)
		if err == nil {
			files[name] = string(content)
		}
	}

	readFile(bundle.Files.Requirements, "requirements.md")
	readFile(bundle.Files.ProjectContext, "project_context.md")
	readFile(bundle.Files.Patterns, "patterns.md")
	readFile(bundle.Files.Decisions, "decisions.md")
	readFile(bundle.Files.TestHints, "test_hints.md")
	if bundle.Files.Conversations != "" {
		readFile(bundle.Files.Conversations, "conversations.md")
	}

	for _, codePath := range bundle.Files.RelevantCode {
		content, err := os.ReadFile(codePath)
		if err == nil {
			files[filepath.Base(codePath)] = string(content)
		}
	}

	return files, nil
}
