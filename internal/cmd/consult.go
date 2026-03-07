package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/claude"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/spf13/cobra"
)

var (
	consultHeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	consultMattStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	consultResponseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	consultFieldStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	consultValueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))
)

// ConsultResult holds AI Matt's parsed response
type ConsultResult struct {
	Response     string
	Decision     string // APPROVED, REJECTED, NEEDS_CHANGES, QUESTION, CONTINUE, etc.
	Reasoning    string
	Alternatives string
	Confidence   string // HIGH, MEDIUM, LOW
	Flags        string
	RawOutput    string
}

// NewConsultCmd creates the consult command for asking AI Matt.
func NewConsultCmd(store *data.Store) *cobra.Command {
	var contextFile string
	var quiet bool

	cmd := &cobra.Command{
		Use:   "consult [question]",
		Short: "Consult AI Matt on a decision",
		Long: `Ask AI Matt for guidance or a decision.

AI Matt simulates Matt's decision-making style and can:
- Answer questions about approach
- Approve or reject proposals
- Provide guidance on next steps
- Flag items that need real Matt's attention

Examples:
  gc consult "Should we add caching here or keep it simple?"
  gc consult "Review complete. Tests pass. Ready to merge?"
  gc consult --context task_context.md "What should we prioritize next?"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			question := strings.Join(args, " ")
			return runConsult(store, question, contextFile, quiet)
		},
	}

	cmd.Flags().StringVarP(&contextFile, "context", "c", "", "File with additional context")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only output the decision (for scripting)")

	return cmd
}

func runConsult(store *data.Store, question, contextFile string, quiet bool) error {
	// Load AI Matt prompt
	agentPrompt, err := loadAIMattPrompt(store.GetDataDir())
	if err != nil {
		return fmt.Errorf("loading AI Matt prompt: %w", err)
	}

	// Build context
	var context strings.Builder
	if contextFile != "" {
		content, err := os.ReadFile(contextFile)
		if err != nil {
			return fmt.Errorf("reading context file: %w", err)
		}
		context.WriteString("## Additional Context\n\n")
		context.WriteString(string(content))
		context.WriteString("\n\n")
	}

	// Build the full prompt
	fullPrompt := fmt.Sprintf(`%s

---

## Current Situation

%s

## Question/Decision Needed

%s

---

Please respond using the structured format from your instructions.
`, agentPrompt, context.String(), question)

	if !quiet {
		fmt.Println(consultHeaderStyle.Render("═══ Consulting AI Matt ═══"))
		fmt.Println()
		fmt.Printf("%s %s\n\n", consultFieldStyle.Render("Question:"), question)
	}

	// Create Claude client
	config := claude.DefaultConfig()
	config.WorkDir = "."
	client := claude.NewClient(config)

	// Execute
	req := &claude.Request{
		Prompt: fullPrompt,
	}

	resp := client.Execute(req)
	if resp.Error != nil {
		return fmt.Errorf("consulting AI Matt: %w", resp.Error)
	}

	// Parse response
	result := parseConsultResponse(resp.Output)

	if quiet {
		// Just output the decision for scripting
		fmt.Println(result.Decision)
		return nil
	}

	// Display full response
	fmt.Println(consultMattStyle.Render("AI Matt says:"))
	fmt.Println()

	if result.Response != "" {
		fmt.Println(consultResponseStyle.Render(result.Response))
		fmt.Println()
	}

	fmt.Printf("%s %s\n", consultFieldStyle.Render("Decision:"), consultValueStyle.Render(result.Decision))

	if result.Confidence != "" {
		fmt.Printf("%s %s\n", consultFieldStyle.Render("Confidence:"), result.Confidence)
	}

	if result.Reasoning != "" {
		fmt.Printf("%s %s\n", consultFieldStyle.Render("Reasoning:"), result.Reasoning)
	}

	if result.Flags != "" {
		fmt.Printf("%s %s\n", consultFieldStyle.Render("Flags:"), lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(result.Flags))
	}

	// Check if escalation needed
	if strings.ToUpper(result.Confidence) == "LOW" {
		fmt.Println()
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("⚠ LOW CONFIDENCE - Real Matt should review this decision"))
	}

	return nil
}

func loadAIMattPrompt(dataDir string) (string, error) {
	// Try project agents directory first
	projectPath := filepath.Join(filepath.Dir(dataDir), "agents", "ai-matt.md")
	if content, err := os.ReadFile(projectPath); err == nil {
		return string(content), nil
	}

	// Fallback to data directory
	dataPath := filepath.Join(dataDir, "agents", "ai-matt.md")
	if content, err := os.ReadFile(dataPath); err == nil {
		return string(content), nil
	}

	return "", fmt.Errorf("ai-matt.md not found in agents/ directory")
}

func parseConsultResponse(output string) *ConsultResult {
	result := &ConsultResult{
		RawOutput: output,
		Decision:  "UNKNOWN",
	}

	lines := strings.Split(output, "\n")
	var currentSection string
	var sectionContent strings.Builder

	flushSection := func() {
		content := strings.TrimSpace(sectionContent.String())
		switch currentSection {
		case "response":
			result.Response = content
		case "decision":
			result.Decision = strings.ToUpper(strings.TrimSpace(content))
		case "reasoning":
			result.Reasoning = content
		case "alternatives":
			result.Alternatives = content
		case "confidence":
			result.Confidence = strings.ToUpper(strings.TrimSpace(content))
		case "flags":
			result.Flags = content
		}
		sectionContent.Reset()
	}

	for _, line := range lines {
		lineLower := strings.ToLower(strings.TrimSpace(line))

		// Check for section headers
		if strings.HasPrefix(lineLower, "## response") {
			flushSection()
			currentSection = "response"
			continue
		} else if strings.HasPrefix(lineLower, "## decision") {
			flushSection()
			currentSection = "decision"
			continue
		} else if strings.HasPrefix(lineLower, "## reasoning") {
			flushSection()
			currentSection = "reasoning"
			continue
		} else if strings.HasPrefix(lineLower, "## alternatives") {
			flushSection()
			currentSection = "alternatives"
			continue
		} else if strings.HasPrefix(lineLower, "## confidence") {
			flushSection()
			currentSection = "confidence"
			continue
		} else if strings.HasPrefix(lineLower, "## flags") {
			flushSection()
			currentSection = "flags"
			continue
		} else if strings.HasPrefix(line, "## ") {
			// Unknown section, flush and reset
			flushSection()
			currentSection = ""
			continue
		}

		if currentSection != "" {
			sectionContent.WriteString(line)
			sectionContent.WriteString("\n")
		}
	}
	flushSection()

	// If no structured sections found, use the whole output as response
	if result.Response == "" && result.Decision == "UNKNOWN" {
		result.Response = strings.TrimSpace(output)
		// Try to detect decision from keywords
		upperOutput := strings.ToUpper(output)
		if strings.Contains(upperOutput, "APPROVED") {
			result.Decision = "APPROVED"
		} else if strings.Contains(upperOutput, "REJECTED") {
			result.Decision = "REJECTED"
		} else if strings.Contains(upperOutput, "CONTINUE") {
			result.Decision = "CONTINUE"
		}
	}

	return result
}

// ConsultAIMatt is a helper function for programmatic consultation
func ConsultAIMatt(store *data.Store, question string, context string) (*ConsultResult, error) {
	agentPrompt, err := loadAIMattPrompt(store.GetDataDir())
	if err != nil {
		return nil, err
	}

	fullPrompt := fmt.Sprintf(`%s

---

## Current Situation

%s

## Question/Decision Needed

%s

---

Please respond using the structured format from your instructions.
`, agentPrompt, context, question)

	config := claude.DefaultConfig()
	config.WorkDir = "."
	client := claude.NewClient(config)

	req := &claude.Request{
		Prompt: fullPrompt,
	}

	resp := client.Execute(req)
	if resp.Error != nil {
		return nil, resp.Error
	}

	return parseConsultResponse(resp.Output), nil
}
