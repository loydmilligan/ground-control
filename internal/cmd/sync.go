package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmariani/ground-control/internal/registry"
	"github.com/mmariani/ground-control/internal/sidecar"
	"github.com/spf13/cobra"
)


var (
	syncHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	syncInfoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	syncOKStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	syncWarnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	syncErrStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

// NewSyncCmd creates the sync command
func NewSyncCmd() *cobra.Command {
	var projectName string
	var quiet bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Aggregate project state for Flight Deck",
		Long: `Sync iterates all registered projects and aggregates their state
into ~/.gc/aggregated.json for the Flight Deck dashboard.

This command:
- Reads each project's .gc/state.json, issues.json, roadmap.json, requests.jsonl
- Aggregates counts and metrics
- Writes to ~/.gc/aggregated.json

Run this before opening Flight Deck or periodically to keep the dashboard current.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync(projectName, quiet)
		},
	}

	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Sync only a specific project")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress output")

	return cmd
}

func runSync(projectFilter string, quiet bool) error {
	reg, err := registry.NewRegistry()
	if err != nil {
		return fmt.Errorf("create registry: %w", err)
	}

	projects, err := reg.ListProjects()
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	if !quiet {
		fmt.Println(syncHeaderStyle.Render("Syncing projects for Flight Deck..."))
	}

	aggregated := sidecar.AggregatedState{
		SyncedAt: time.Now(),
		Projects: make(map[string]sidecar.ProjectSyncState),
	}

	for _, proj := range projects {
		// Filter if specified
		if projectFilter != "" && proj.Name != projectFilter && proj.Path != projectFilter {
			continue
		}

		syncState, err := syncProject(proj)
		if err != nil {
			if !quiet {
				fmt.Printf("  %s %s: %s\n", syncWarnStyle.Render("⚠"), proj.Name, err)
			}
			continue
		}

		aggregated.Projects[proj.Name] = *syncState

		// Update totals
		aggregated.Totals.Projects++
		aggregated.Totals.Issues += syncState.IssuesCount
		aggregated.Totals.OpenBugs += syncState.OpenBugs
		aggregated.Totals.PendingRequests += syncState.PendingRequests
		aggregated.Totals.AttentionFlags += syncState.AttentionFlags

		if !quiet {
			status := syncOKStyle.Render("✓")
			extras := ""
			if syncState.PendingRequests > 0 {
				extras = fmt.Sprintf(" (%d requests)", syncState.PendingRequests)
			}
			fmt.Printf("  %s %s%s\n", status, proj.Name, syncInfoStyle.Render(extras))
		}
	}

	// Save aggregated state
	aggregatedPath := filepath.Join(reg.Path(), "aggregated.json")
	data, err := json.MarshalIndent(aggregated, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal aggregated: %w", err)
	}
	if err := os.WriteFile(aggregatedPath, data, 0644); err != nil {
		return fmt.Errorf("write aggregated: %w", err)
	}

	if !quiet {
		fmt.Println()
		fmt.Printf("%s\n", syncOKStyle.Render(fmt.Sprintf("Synced %d projects → %s", aggregated.Totals.Projects, aggregatedPath)))
		if aggregated.Totals.PendingRequests > 0 {
			fmt.Printf("%s\n", syncWarnStyle.Render(fmt.Sprintf("⚠ %d pending requests across projects", aggregated.Totals.PendingRequests)))
		}
		if aggregated.Totals.AttentionFlags > 0 {
			fmt.Printf("%s\n", syncWarnStyle.Render(fmt.Sprintf("⚠ %d attention flags", aggregated.Totals.AttentionFlags)))
		}
	}

	return nil
}

func syncProject(proj registry.ProjectEntry) (*sidecar.ProjectSyncState, error) {
	mgr := sidecar.NewManager(proj.Path)

	if !mgr.Exists() {
		return nil, fmt.Errorf("not adopted")
	}

	state, err := mgr.LoadState()
	if err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}

	config, _ := mgr.LoadConfig() // Ignore error, use defaults

	issues, err := mgr.LoadIssues()
	if err != nil {
		issues = &sidecar.IssuesFile{Issues: []sidecar.IssueItem{}}
	}

	roadmap, err := mgr.LoadRoadmap()
	if err != nil {
		roadmap = &sidecar.RoadmapFile{}
	}

	requests, err := mgr.LoadRequests()
	if err != nil {
		requests = []sidecar.FDRequest{}
	}

	// Count open bugs
	openBugs := 0
	for _, issue := range issues.Issues {
		if issue.Type == "bug" && issue.Status == "open" {
			openBugs++
		}
	}

	// Count pending requests
	pendingRequests := 0
	for _, req := range requests {
		if req.Status == "pending" {
			pendingRequests++
		}
	}

	// Calculate roadmap completion percentage
	var roadmapPct float64
	if len(roadmap.Features) > 0 {
		var totalPct float64
		for _, feat := range roadmap.Features {
			totalPct += feat.CompletionPct
		}
		roadmapPct = totalPct / float64(len(roadmap.Features))
	}

	// Get phase from config if available
	phase := ""
	if config != nil {
		phase = string(config.Phase)
	}

	syncState := &sidecar.ProjectSyncState{
		Name:            proj.Name,
		Path:            proj.Path,
		Status:          state.Session.Status,
		WorkMode:        state.WorkMode,
		Phase:           phase,
		IssuesCount:     len(issues.Issues),
		OpenBugs:        openBugs,
		FeaturesCount:   len(roadmap.Features),
		RoadmapPct:      roadmapPct,
		PendingRequests: pendingRequests,
		AttentionFlags:  len(state.Attention),
		Sprint:          state.Sprint,
	}

	// Get last activity time
	if len(state.Activity) > 0 {
		lastActivity := state.Activity[len(state.Activity)-1].Timestamp
		syncState.LastActivity = &lastActivity
	}

	return syncState, nil
}
