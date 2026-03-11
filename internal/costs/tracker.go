// Package costs handles token and cost tracking for Flight Deck
package costs

import (
	"fmt"
	"sync"
	"time"

	"github.com/mmariani/ground-control/internal/registry"
	"github.com/mmariani/ground-control/internal/sidecar"
)

// Pricing constants (approximate, update as needed)
const (
	// Claude pricing per 1M tokens (approximate)
	InputPricePerMillion  = 3.0   // $3/1M input tokens
	OutputPricePerMillion = 15.0  // $15/1M output tokens
	CachePricePerMillion  = 0.30  // $0.30/1M cached tokens
)

// TokenUsage represents a single usage event
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	CacheTokens  int
	Timestamp    time.Time
}

// Tracker tracks token usage and costs for projects
type Tracker struct {
	mu sync.Mutex
}

// NewTracker creates a new cost tracker
func NewTracker() *Tracker {
	return &Tracker{}
}

// RecordUsage records token usage for a project session
func (t *Tracker) RecordUsage(projectPath string, usage TokenUsage) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	mgr := sidecar.NewManager(projectPath)
	state, err := mgr.LoadState()
	if err != nil {
		// Create new state if doesn't exist
		state = &sidecar.ProjectState{}
	}

	// Calculate cost for this usage
	cost := t.CalculateCost(usage)

	// Update session costs
	state.Costs.SessionTokens += usage.InputTokens + usage.OutputTokens + usage.CacheTokens
	state.Costs.SessionCostUSD += cost

	// Update daily costs
	state.Costs.TodayTokens += usage.InputTokens + usage.OutputTokens + usage.CacheTokens
	state.Costs.TodayCostUSD += cost

	return mgr.SaveState(state)
}

// CalculateCost calculates USD cost from token usage
func (t *Tracker) CalculateCost(usage TokenUsage) float64 {
	inputCost := float64(usage.InputTokens) / 1_000_000 * InputPricePerMillion
	outputCost := float64(usage.OutputTokens) / 1_000_000 * OutputPricePerMillion
	cacheCost := float64(usage.CacheTokens) / 1_000_000 * CachePricePerMillion
	return inputCost + outputCost + cacheCost
}

// GetProjectCosts returns current costs for a project
func (t *Tracker) GetProjectCosts(projectPath string) (*sidecar.CostInfo, error) {
	mgr := sidecar.NewManager(projectPath)
	state, err := mgr.LoadState()
	if err != nil {
		return nil, err
	}
	return &state.Costs, nil
}

// ResetSessionCosts resets session costs (call when session ends)
func (t *Tracker) ResetSessionCosts(projectPath string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	mgr := sidecar.NewManager(projectPath)
	state, err := mgr.LoadState()
	if err != nil {
		return err
	}

	state.Costs.SessionTokens = 0
	state.Costs.SessionCostUSD = 0

	return mgr.SaveState(state)
}

// ResetDailyCosts resets daily costs (call at midnight)
func (t *Tracker) ResetDailyCosts(projectPath string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	mgr := sidecar.NewManager(projectPath)
	state, err := mgr.LoadState()
	if err != nil {
		return err
	}

	state.Costs.TodayTokens = 0
	state.Costs.TodayCostUSD = 0

	return mgr.SaveState(state)
}

// GetGlobalCosts aggregates costs across all projects
func (t *Tracker) GetGlobalCosts() (*GlobalCosts, error) {
	reg, err := registry.NewRegistry()
	if err != nil {
		return nil, err
	}

	projects, err := reg.ListProjects()
	if err != nil {
		return nil, err
	}

	global := &GlobalCosts{
		Projects: make(map[string]sidecar.CostInfo),
	}

	for _, p := range projects {
		mgr := sidecar.NewManager(p.Path)
		state, err := mgr.LoadState()
		if err != nil {
			continue // Skip projects without state
		}

		global.Projects[p.Name] = state.Costs
		global.TotalTodayTokens += state.Costs.TodayTokens
		global.TotalTodayCostUSD += state.Costs.TodayCostUSD
	}

	return global, nil
}

// GlobalCosts represents aggregated costs across all projects
type GlobalCosts struct {
	Projects          map[string]sidecar.CostInfo
	TotalTodayTokens  int
	TotalTodayCostUSD float64
}

// FormatCost formats a USD amount for display
func FormatCost(usd float64) string {
	if usd < 0.01 {
		return "<$0.01"
	}
	return fmt.Sprintf("$%.2f", usd)
}

// FormatTokens formats token count for display
func FormatTokens(tokens int) string {
	if tokens >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1_000_000)
	}
	if tokens >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1_000)
	}
	return fmt.Sprintf("%d", tokens)
}
