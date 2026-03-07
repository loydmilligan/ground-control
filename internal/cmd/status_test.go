package cmd

import (
	"testing"
	"time"
)

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "just now",
			duration: 30 * time.Second,
			want:     "just now",
		},
		{
			name:     "1 minute ago",
			duration: 1 * time.Minute,
			want:     "1 minute ago",
		},
		{
			name:     "multiple minutes",
			duration: 5 * time.Minute,
			want:     "5 minutes ago",
		},
		{
			name:     "1 hour ago",
			duration: 1 * time.Hour,
			want:     "1 hour ago",
		},
		{
			name:     "multiple hours",
			duration: 3 * time.Hour,
			want:     "3 hours ago",
		},
		{
			name:     "1 day ago",
			duration: 24 * time.Hour,
			want:     "1 day ago",
		},
		{
			name:     "multiple days",
			duration: 3 * 24 * time.Hour,
			want:     "3 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pastTime := time.Now().Add(-tt.duration)
			got := formatTimeAgo(pastTime)
			if got != tt.want {
				t.Errorf("formatTimeAgo() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatTimeAgoOldDate(t *testing.T) {
	// Test dates older than a week show the date
	oldTime := time.Now().Add(-14 * 24 * time.Hour)
	got := formatTimeAgo(oldTime)
	// Should be in "Jan 2" format
	expected := oldTime.Format("Jan 2")
	if got != expected {
		t.Errorf("formatTimeAgo() for old date = %q, want %q", got, expected)
	}
}
