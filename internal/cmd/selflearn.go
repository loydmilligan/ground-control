package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/spf13/cobra"
)

var (
	learnHeaderStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	learnCategoryStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	learnHighStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	learnMedStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	learnLowStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	learnDimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	learnSuggStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))
)

// LearningSession represents a session entry in learning-log.json
type LearningSession struct {
	SessionID string          `json:"session_id"`
	Timestamp string          `json:"timestamp"`
	Issues    []LearningIssue `json:"issues"`
	Summary   struct {
		TotalTasks      int            `json:"total_tasks"`
		CompletedTasks  int            `json:"completed_tasks"`
		FailedTasks     int            `json:"failed_tasks"`
		EscalatedTasks  int            `json:"escalated_tasks"`
		CategoryCounts  map[string]int `json:"category_counts"`
		SeverityCounts  map[string]int `json:"severity_counts"`
	} `json:"summary"`
}

// LearningIssue represents an individual issue in the learning log
type LearningIssue struct {
	TaskID      string `json:"task_id"`
	Stage       string `json:"stage"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Suggestion  string `json:"suggestion"`
}

// NewSelfLearnCmd creates the self-learn command.
func NewSelfLearnCmd(store *data.Store) *cobra.Command {
	var showSuggestions bool
	var categoryFilter string
	var minSeverity string

	cmd := &cobra.Command{
		Use:   "self-learn",
		Short: "Analyze learning log and identify patterns",
		Long: `Analyze the learning-log.json to identify patterns in issues and propose improvements.

This command helps close the learning loop by reviewing collected issues,
identifying recurring patterns, and surfacing actionable suggestions.

Examples:
  gc self-learn                    # Show pattern analysis
  gc self-learn --suggestions      # Include all suggestions
  gc self-learn --category context # Filter by category
  gc self-learn --min-severity significant  # Only significant issues`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSelfLearn(store, showSuggestions, categoryFilter, minSeverity)
		},
	}

	cmd.Flags().BoolVarP(&showSuggestions, "suggestions", "s", false, "Show all suggestions")
	cmd.Flags().StringVarP(&categoryFilter, "category", "c", "", "Filter by category (partial match)")
	cmd.Flags().StringVar(&minSeverity, "min-severity", "", "Minimum severity: minor, moderate, significant, critical")

	return cmd
}

func runSelfLearn(store *data.Store, showSuggestions bool, categoryFilter, minSeverity string) error {
	// Load learning log
	logPath := filepath.Join(store.GetDataDir(), "learning-log.json")
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(learnDimStyle.Render("No learning-log.json found. Issues will be collected during gc orc sessions."))
			return nil
		}
		return fmt.Errorf("reading learning-log.json: %w", err)
	}

	var sessions []LearningSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return fmt.Errorf("parsing learning-log.json: %w", err)
	}

	// Collect all issues
	var allIssues []LearningIssue
	for _, s := range sessions {
		allIssues = append(allIssues, s.Issues...)
	}

	// Apply filters
	var filtered []LearningIssue
	for _, issue := range allIssues {
		// Category filter
		if categoryFilter != "" && !strings.Contains(strings.ToLower(issue.Category), strings.ToLower(categoryFilter)) {
			continue
		}
		// Severity filter
		if minSeverity != "" && !meetsSeverity(issue.Severity, minSeverity) {
			continue
		}
		filtered = append(filtered, issue)
	}

	fmt.Println(learnHeaderStyle.Render("═══ Self-Learning Analysis ═══"))
	fmt.Println()

	if len(filtered) == 0 {
		fmt.Println(learnDimStyle.Render("No issues match the current filters."))
		return nil
	}

	// Pattern Analysis
	fmt.Println(learnCategoryStyle.Render("Pattern Analysis"))
	fmt.Println()

	// Count by category
	categoryCount := make(map[string]int)
	severityCount := make(map[string]int)
	stageCount := make(map[string]int)

	for _, issue := range filtered {
		categoryCount[issue.Category]++
		severityCount[issue.Severity]++
		stageCount[issue.Stage]++
	}

	// Sort categories by count
	type countPair struct {
		name  string
		count int
	}
	var categories []countPair
	for name, count := range categoryCount {
		categories = append(categories, countPair{name, count})
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].count > categories[j].count
	})

	fmt.Printf("  %s\n", learnDimStyle.Render("Top Issue Categories:"))
	for i, cat := range categories {
		if i >= 5 {
			break
		}
		bar := strings.Repeat("█", min(cat.count*2, 20))
		fmt.Printf("    %-25s %s %d\n", cat.name, learnMedStyle.Render(bar), cat.count)
	}
	fmt.Println()

	// Severity distribution
	fmt.Printf("  %s\n", learnDimStyle.Render("Severity Distribution:"))
	severityOrder := []string{"critical", "significant", "moderate", "minor"}
	for _, sev := range severityOrder {
		if count, ok := severityCount[sev]; ok {
			style := getSeverityStyle(sev)
			fmt.Printf("    %-12s %s\n", style.Render(sev), learnDimStyle.Render(fmt.Sprintf("%d issues", count)))
		}
	}
	fmt.Println()

	// Stage distribution
	fmt.Printf("  %s\n", learnDimStyle.Render("Issues by Stage:"))
	var stages []countPair
	for name, count := range stageCount {
		stages = append(stages, countPair{name, count})
	}
	sort.Slice(stages, func(i, j int) bool {
		return stages[i].count > stages[j].count
	})
	for _, stage := range stages {
		fmt.Printf("    %-15s %d\n", stage.name, stage.count)
	}
	fmt.Println()

	// Key Insights
	fmt.Println(learnCategoryStyle.Render("Key Insights"))
	fmt.Println()

	// Find most common category
	if len(categories) > 0 {
		top := categories[0]
		fmt.Printf("  • Most common issue category: %s (%d occurrences)\n",
			learnMedStyle.Render(top.name), top.count)
	}

	// Count significant+ issues
	highCount := severityCount["significant"] + severityCount["critical"]
	if highCount > 0 {
		fmt.Printf("  • High-severity issues requiring attention: %s\n",
			learnHighStyle.Render(fmt.Sprintf("%d", highCount)))
	}

	// Most problematic stage
	if len(stages) > 0 {
		fmt.Printf("  • Stage with most issues: %s\n", learnMedStyle.Render(stages[0].name))
	}

	fmt.Println()

	// Suggestions
	if showSuggestions {
		fmt.Println(learnCategoryStyle.Render("Improvement Suggestions"))
		fmt.Println()

		// Group suggestions by category
		suggestionsByCategory := make(map[string][]string)
		for _, issue := range filtered {
			if issue.Suggestion != "" {
				suggestionsByCategory[issue.Category] = append(suggestionsByCategory[issue.Category], issue.Suggestion)
			}
		}

		for _, cat := range categories {
			if suggestions, ok := suggestionsByCategory[cat.name]; ok {
				fmt.Printf("  %s\n", learnCategoryStyle.Render(cat.name))
				// Deduplicate suggestions
				seen := make(map[string]bool)
				for _, sugg := range suggestions {
					if !seen[sugg] {
						seen[sugg] = true
						// Wrap long suggestions
						wrapped := wrapText(sugg, 70)
						for i, line := range wrapped {
							if i == 0 {
								fmt.Printf("    → %s\n", learnSuggStyle.Render(line))
							} else {
								fmt.Printf("      %s\n", learnSuggStyle.Render(line))
							}
						}
						fmt.Println()
					}
				}
			}
		}
	} else {
		fmt.Println(learnDimStyle.Render("Run with --suggestions to see improvement recommendations."))
	}

	fmt.Println()
	fmt.Printf("Total: %d sessions, %d issues analyzed\n", len(sessions), len(filtered))

	return nil
}

func meetsSeverity(actual, minimum string) bool {
	order := map[string]int{
		"minor":       1,
		"moderate":    2,
		"significant": 3,
		"critical":    4,
	}
	return order[actual] >= order[minimum]
}

func getSeverityStyle(severity string) lipgloss.Style {
	switch severity {
	case "critical":
		return learnHighStyle
	case "significant":
		return learnHighStyle
	case "moderate":
		return learnMedStyle
	default:
		return learnLowStyle
	}
}

func wrapText(text string, width int) []string {
	var lines []string
	words := strings.Fields(text)
	var current string

	for _, word := range words {
		if len(current)+len(word)+1 > width {
			if current != "" {
				lines = append(lines, current)
			}
			current = word
		} else {
			if current != "" {
				current += " "
			}
			current += word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
