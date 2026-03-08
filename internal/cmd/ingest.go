package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/spf13/cobra"
)

var (
	ingestHeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	ingestSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	ingestDimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	ingestValueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

// MessageContent represents the content of a user/assistant message
type MessageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LogEntry represents a single entry from a Claude Code session log
type LogEntry struct {
	Type        string          `json:"type"`
	Message     json.RawMessage `json:"message,omitempty"`
	UserType    string          `json:"userType,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
	SessionID   string          `json:"sessionId"`
	UUID        string          `json:"uuid,omitempty"`
	CWD         string          `json:"cwd,omitempty"`
}

// GetMessageContent extracts the text content from a message
func (e *LogEntry) GetMessageContent() string {
	if e.Message == nil {
		return ""
	}

	// Try to parse as MessageContent object
	var mc MessageContent
	if err := json.Unmarshal(e.Message, &mc); err == nil && mc.Content != "" {
		return mc.Content
	}

	// Try to parse as plain string
	var str string
	if err := json.Unmarshal(e.Message, &str); err == nil {
		return str
	}

	return ""
}

// SessionStats holds statistics for a session
type SessionStats struct {
	SessionID     string
	StartTime     time.Time
	EndTime       time.Time
	MessageCount  int
	UserMessages  int
	Duration      time.Duration
	Topics        []string
}

// PersonalityInsights holds extracted personality patterns
type PersonalityInsights struct {
	CommonPhrases      map[string]int `json:"common_phrases"`
	TopicInterests     map[string]int `json:"topic_interests"`
	CommunicationStyle struct {
		Verbosity       string `json:"verbosity"` // concise, moderate, verbose
		FormattingPrefs string `json:"formatting_prefs"`
		TonePatterns    string `json:"tone_patterns"`
	} `json:"communication_style"`
	WorkPatterns struct {
		AvgSessionLength time.Duration `json:"avg_session_length"`
		CommonCommands   []string      `json:"common_commands"`
		PreferredTimes   []int         `json:"preferred_times"` // hours of day
	} `json:"work_patterns"`
	ExtractedAt time.Time `json:"extracted_at"`
	SessionCount int      `json:"session_count"`
	MessageCount int      `json:"message_count"`
}

// NewIngestCmd creates the ingest command for Claude Code log ingestion.
func NewIngestCmd(store *data.Store) *cobra.Command {
	var projectFilter string
	var daysBack int
	var outputFile string

	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest Claude Code logs for AI Matt training",
		Long: `Analyze Claude Code session logs to extract personality patterns.

This reads your Claude Code conversation history and extracts:
- Communication patterns and preferences
- Topic interests
- Work patterns (session length, timing)
- Common phrases and style

Results are saved for AI Matt personality calibration.

Examples:
  gc ingest                          # Analyze all ground-control sessions
  gc ingest --days 7                 # Only last 7 days
  gc ingest --project ground-control # Filter by project
  gc ingest --output data/ai-matt-insights.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIngest(store, projectFilter, daysBack, outputFile)
		},
	}

	cmd.Flags().StringVarP(&projectFilter, "project", "p", "ground-control", "Project name to analyze")
	cmd.Flags().IntVarP(&daysBack, "days", "d", 30, "Number of days to analyze")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file for insights (default: data/ai-matt-insights.json)")

	return cmd
}

func runIngest(store *data.Store, projectFilter string, daysBack int, outputFile string) error {
	fmt.Println(ingestHeaderStyle.Render("═══ Claude Code Log Ingestion ═══"))
	fmt.Println()

	// Find Claude projects directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	projectsDir := filepath.Join(homeDir, ".claude", "projects")
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		fmt.Println(ingestDimStyle.Render("No Claude Code projects found at ~/.claude/projects"))
		return nil
	}

	// Find matching project directories
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return fmt.Errorf("reading projects directory: %w", err)
	}

	var matchingDirs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.Contains(entry.Name(), projectFilter) {
			matchingDirs = append(matchingDirs, filepath.Join(projectsDir, entry.Name()))
		}
	}

	if len(matchingDirs) == 0 {
		fmt.Printf("No project directories matching '%s' found\n", projectFilter)
		return nil
	}

	fmt.Printf("Found %d matching project(s)\n", len(matchingDirs))
	fmt.Println()

	// Collect all session logs
	cutoffTime := time.Now().AddDate(0, 0, -daysBack)
	var allEntries []LogEntry
	sessionCount := 0

	for _, dir := range matchingDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		if err != nil {
			continue
		}

		for _, file := range files {
			entries, err := readLogFile(file, cutoffTime)
			if err != nil {
				continue
			}
			if len(entries) > 0 {
				allEntries = append(allEntries, entries...)
				sessionCount++
			}
		}
	}

	if len(allEntries) == 0 {
		fmt.Println(ingestDimStyle.Render("No log entries found in the specified time range."))
		return nil
	}

	fmt.Printf("Analyzed %d sessions, %d log entries\n", sessionCount, len(allEntries))
	fmt.Println()

	// Extract insights
	insights := extractInsights(allEntries, sessionCount)

	// Display summary
	fmt.Println(ingestSectionStyle.Render("Session Statistics"))
	fmt.Printf("  Sessions analyzed: %s\n", ingestValueStyle.Render(fmt.Sprintf("%d", sessionCount)))
	fmt.Printf("  Total messages: %s\n", ingestValueStyle.Render(fmt.Sprintf("%d", insights.MessageCount)))
	if insights.WorkPatterns.AvgSessionLength > 0 {
		fmt.Printf("  Avg session length: %s\n", ingestValueStyle.Render(insights.WorkPatterns.AvgSessionLength.Round(time.Minute).String()))
	}
	fmt.Println()

	fmt.Println(ingestSectionStyle.Render("Communication Style"))
	fmt.Printf("  Verbosity: %s\n", insights.CommunicationStyle.Verbosity)
	fmt.Println()

	if len(insights.TopicInterests) > 0 {
		fmt.Println(ingestSectionStyle.Render("Topic Interests"))
		// Sort by frequency
		type topicCount struct {
			topic string
			count int
		}
		var topics []topicCount
		for t, c := range insights.TopicInterests {
			topics = append(topics, topicCount{t, c})
		}
		sort.Slice(topics, func(i, j int) bool {
			return topics[i].count > topics[j].count
		})
		for i, t := range topics {
			if i >= 5 {
				break
			}
			fmt.Printf("  %s: %d mentions\n", t.topic, t.count)
		}
		fmt.Println()
	}

	// Save insights
	if outputFile == "" {
		outputFile = filepath.Join(store.GetDataDir(), "ai-matt-insights.json")
	}

	data, err := json.MarshalIndent(insights, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling insights: %w", err)
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("writing insights file: %w", err)
	}

	fmt.Printf("Insights saved to: %s\n", ingestValueStyle.Render(outputFile))
	fmt.Println()
	fmt.Println(ingestDimStyle.Render("Use these insights to calibrate AI Matt's personality in agents/ai-matt.md"))

	return nil
}

func readLogFile(path string, cutoffTime time.Time) ([]LogEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)
	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		var entry LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		// Filter by time
		if !entry.Timestamp.IsZero() && entry.Timestamp.After(cutoffTime) {
			entries = append(entries, entry)
		}
	}

	return entries, scanner.Err()
}

func extractInsights(entries []LogEntry, sessionCount int) *PersonalityInsights {
	insights := &PersonalityInsights{
		CommonPhrases:  make(map[string]int),
		TopicInterests: make(map[string]int),
		ExtractedAt:    time.Now(),
		SessionCount:   sessionCount,
	}

	var userMessages []string
	var messageLengths []int
	hourCounts := make(map[int]int)

	for _, entry := range entries {
		if entry.Type == "user" {
			content := entry.GetMessageContent()
			if content != "" {
				userMessages = append(userMessages, content)
				messageLengths = append(messageLengths, len(content))
				insights.MessageCount++

				// Track hour of day
				if !entry.Timestamp.IsZero() {
					hourCounts[entry.Timestamp.Hour()]++
				}

				// Extract topic keywords
				extractTopics(content, insights.TopicInterests)
			}
		}
	}

	// Determine verbosity based on average message length
	if len(messageLengths) > 0 {
		avgLen := 0
		for _, l := range messageLengths {
			avgLen += l
		}
		avgLen /= len(messageLengths)

		if avgLen < 50 {
			insights.CommunicationStyle.Verbosity = "concise"
		} else if avgLen < 150 {
			insights.CommunicationStyle.Verbosity = "moderate"
		} else {
			insights.CommunicationStyle.Verbosity = "verbose"
		}
	}

	// Find preferred hours
	var preferredHours []int
	for hour, count := range hourCounts {
		if count > len(entries)/24 { // More than average
			preferredHours = append(preferredHours, hour)
		}
	}
	sort.Ints(preferredHours)
	insights.WorkPatterns.PreferredTimes = preferredHours

	return insights
}

func extractTopics(message string, topics map[string]int) {
	// Simple keyword extraction
	keywords := []string{
		"task", "test", "bug", "feature", "refactor", "deploy",
		"api", "database", "ui", "tui", "cli", "config",
		"error", "fix", "implement", "design", "review",
		"agent", "ai", "claude", "delegation", "handoff",
	}

	lower := strings.ToLower(message)
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			topics[kw]++
		}
	}
}
