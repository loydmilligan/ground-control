package sidecar

import "time"

// AggregatedState holds the synced state from all projects
type AggregatedState struct {
	SyncedAt time.Time                   `json:"synced_at"`
	Projects map[string]ProjectSyncState `json:"projects"`
	Totals   AggregatedTotals            `json:"totals"`
}

// ProjectSyncState holds synced state for a single project
type ProjectSyncState struct {
	Name            string      `json:"name"`
	Path            string      `json:"path"`
	Status          string      `json:"status"`
	WorkMode        WorkMode    `json:"work_mode,omitempty"`
	Phase           string      `json:"phase,omitempty"`
	IssuesCount     int         `json:"issues_count"`
	OpenBugs        int         `json:"open_bugs"`
	FeaturesCount   int         `json:"features_count"`
	RoadmapPct      float64     `json:"roadmap_pct"`
	PendingRequests int         `json:"pending_requests"`
	AttentionFlags  int         `json:"attention_flags"`
	Sprint          *SprintInfo `json:"sprint,omitempty"`
	LastActivity    *time.Time  `json:"last_activity,omitempty"`
}

// AggregatedTotals holds totals across all projects
type AggregatedTotals struct {
	Projects        int `json:"projects"`
	Issues          int `json:"issues"`
	OpenBugs        int `json:"open_bugs"`
	PendingRequests int `json:"pending_requests"`
	AttentionFlags  int `json:"attention_flags"`
}
