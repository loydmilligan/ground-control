// Package claude provides a wrapper for the Claude Code CLI.
package claude

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PermissionMode defines how Claude CLI handles permissions.
type PermissionMode string

const (
	// PermissionDefault uses Claude's default interactive permission handling.
	PermissionDefault PermissionMode = "default"
	// PermissionAcceptEdits auto-accepts file edits but prompts for other actions.
	PermissionAcceptEdits PermissionMode = "acceptEdits"
	// PermissionBypass skips all permission checks (for headless/automated use).
	PermissionBypass PermissionMode = "bypassPermissions"
)

// Config holds Claude CLI configuration.
type Config struct {
	Timeout        time.Duration
	Model          string
	WorkDir        string
	Verbose        bool
	PermissionMode PermissionMode
}

// DefaultConfig returns the default Claude CLI configuration.
func DefaultConfig() *Config {
	return &Config{
		Timeout:        30 * time.Minute,
		Model:          "", // Use default model
		WorkDir:        "",
		PermissionMode: PermissionBypass, // Default for headless/automated use
	}
}

// Client wraps the Claude Code CLI.
type Client struct {
	config *Config
}

// NewClient creates a new Claude CLI client.
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}
	return &Client{config: config}
}

// Request represents a request to Claude Code CLI.
type Request struct {
	Prompt           string
	ContextFiles     []string // Files to include as context
	SystemPrompt     string   // Optional system prompt override
	WorkingDirectory string   // Optional working directory for --add-dir
}

// Response represents a response from Claude Code CLI.
type Response struct {
	Output   string
	ExitCode int
	Duration time.Duration
	Error    error
}

// Execute runs the Claude Code CLI with the given request.
func (c *Client) Execute(req *Request) *Response {
	start := time.Now()

	// Build command arguments - use --print for non-interactive mode
	args := []string{"--print"}

	// Add permission mode for headless execution
	if c.config.PermissionMode != "" && c.config.PermissionMode != PermissionDefault {
		args = append(args, "--permission-mode", string(c.config.PermissionMode))
	}

	// Add working directory if specified
	if req.WorkingDirectory != "" {
		args = append(args, "--add-dir", req.WorkingDirectory)
	}

	// Build prompt with context file references
	prompt := buildPromptWithContext(req.Prompt, req.ContextFiles)

	// Create command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", args...)

	// Set working directory if specified
	if c.config.WorkDir != "" {
		cmd.Dir = c.config.WorkDir
	}

	// Pass prompt via stdin instead of command-line argument
	cmd.Stdin = strings.NewReader(prompt)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()

	response := &Response{
		Output:   stdout.String(),
		Duration: time.Since(start),
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			response.Error = fmt.Errorf("claude command timed out after %v", c.config.Timeout)
			response.ExitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			response.ExitCode = exitErr.ExitCode()
			response.Error = fmt.Errorf("claude exited with code %d: %s", response.ExitCode, stderr.String())
		} else {
			response.Error = fmt.Errorf("failed to run claude: %w", err)
			response.ExitCode = -1
		}
	}

	return response
}

// ExecuteWithStreaming runs Claude CLI and streams output (for verbose mode).
func (c *Client) ExecuteWithStreaming(req *Request, onOutput func(string)) *Response {
	start := time.Now()

	args := []string{"--print"}

	// Add permission mode for headless execution
	if c.config.PermissionMode != "" && c.config.PermissionMode != PermissionDefault {
		args = append(args, "--permission-mode", string(c.config.PermissionMode))
	}

	// Add working directory if specified
	if req.WorkingDirectory != "" {
		args = append(args, "--add-dir", req.WorkingDirectory)
	}

	// Build prompt with context file references
	prompt := buildPromptWithContext(req.Prompt, req.ContextFiles)

	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", args...)
	if c.config.WorkDir != "" {
		cmd.Dir = c.config.WorkDir
	}

	// Pass prompt via stdin instead of command-line argument
	cmd.Stdin = strings.NewReader(prompt)

	// For streaming, we use a pipe for stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &Response{
			Error:    fmt.Errorf("failed to create stdout pipe: %w", err),
			ExitCode: -1,
			Duration: time.Since(start),
		}
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return &Response{
			Error:    fmt.Errorf("failed to start claude: %w", err),
			ExitCode: -1,
			Duration: time.Since(start),
		}
	}

	// Read output and call callback
	var fullOutput strings.Builder
	buf := make([]byte, 1024)
	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			fullOutput.WriteString(chunk)
			if onOutput != nil {
				onOutput(chunk)
			}
		}
		if readErr != nil {
			break
		}
	}

	err = cmd.Wait()
	response := &Response{
		Output:   fullOutput.String(),
		Duration: time.Since(start),
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			response.Error = fmt.Errorf("claude command timed out after %v", c.config.Timeout)
			response.ExitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			response.ExitCode = exitErr.ExitCode()
			response.Error = fmt.Errorf("claude exited with code %d: %s", response.ExitCode, stderr.String())
		} else {
			response.Error = fmt.Errorf("failed to run claude: %w", err)
			response.ExitCode = -1
		}
	}

	return response
}

// BuildCoderPrompt creates a prompt for the Coder stage.
func BuildCoderPrompt(taskTitle, taskDescription string, requirements []string, feedback string) string {
	var sb strings.Builder

	sb.WriteString("# Task: ")
	sb.WriteString(taskTitle)
	sb.WriteString("\n\n")

	sb.WriteString("## Description\n")
	sb.WriteString(taskDescription)
	sb.WriteString("\n\n")

	if len(requirements) > 0 {
		sb.WriteString("## Requirements\n")
		for _, req := range requirements {
			sb.WriteString("- ")
			sb.WriteString(req)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if feedback != "" {
		sb.WriteString("## Feedback from Previous Review\n")
		sb.WriteString("The following feedback was provided. Please address all points:\n\n")
		sb.WriteString(feedback)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Instructions\n")
	sb.WriteString("Implement the task according to the requirements. ")
	sb.WriteString("Follow the project patterns and conventions from the context files. ")
	sb.WriteString("After implementation, provide a brief summary of what was changed.\n")

	return sb.String()
}

// BuildReviewerPrompt creates a prompt for the Reviewer stage.
func BuildReviewerPrompt(taskTitle string, requirements []string, changedFiles []string) string {
	var sb strings.Builder

	sb.WriteString("# Code Review for: ")
	sb.WriteString(taskTitle)
	sb.WriteString("\n\n")

	sb.WriteString("## Requirements to Verify\n")
	for _, req := range requirements {
		sb.WriteString("- ")
		sb.WriteString(req)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	if len(changedFiles) > 0 {
		sb.WriteString("## Changed Files\n")
		for _, f := range changedFiles {
			sb.WriteString("- ")
			sb.WriteString(f)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Review Criteria\n")
	sb.WriteString("1. Does the code meet all requirements?\n")
	sb.WriteString("2. Does it follow project patterns and conventions?\n")
	sb.WriteString("3. Are there any bugs or edge cases missed?\n")
	sb.WriteString("4. Is error handling appropriate?\n")
	sb.WriteString("5. Is the code clean and readable?\n\n")

	sb.WriteString("## Response Format\n")
	sb.WriteString("Respond with one of:\n")
	sb.WriteString("- APPROVED: [brief explanation]\n")
	sb.WriteString("- NEEDS_REVISION: [specific, actionable feedback]\n")
	sb.WriteString("- ESCALATE: [reason human review is needed]\n")

	return sb.String()
}

// buildPromptWithContext creates a prompt that references context files.
// Claude Code can read these files directly from the filesystem.
func buildPromptWithContext(prompt string, contextFiles []string) string {
	if len(contextFiles) == 0 {
		return prompt
	}

	var sb strings.Builder

	// Tell Claude where to find context
	sb.WriteString("## Context Files\n\n")
	sb.WriteString("Please read these files for context before starting:\n\n")
	for _, file := range contextFiles {
		if _, err := os.Stat(file); err == nil {
			sb.WriteString(fmt.Sprintf("- %s\n", file))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("## Task\n\n")
	sb.WriteString(prompt)

	return sb.String()
}

// GetContextFiles returns a list of context files from a bundle path.
func GetContextFiles(bundlePath string) []string {
	var files []string

	candidates := []string{
		"requirements.md",
		"project_context.md",
		"patterns.md",
		"decisions.md",
		"test_hints.md",
	}

	for _, name := range candidates {
		path := filepath.Join(bundlePath, name)
		if _, err := os.Stat(path); err == nil {
			files = append(files, path)
		}
	}

	// Add relevant code files
	codeDir := filepath.Join(bundlePath, "relevant_code")
	if entries, err := os.ReadDir(codeDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				files = append(files, filepath.Join(codeDir, entry.Name()))
			}
		}
	}

	return files
}
