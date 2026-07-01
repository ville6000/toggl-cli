package cmd

import (
	"testing"
	"time"

	"github.com/ville6000/toggl-cli/internal/data"
)

func TestParseWorkItemID(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantID      int
		wantOK      bool
	}{
		{"hash prefix", "#1234 fix bug", 1234, true},
		{"project hash", "AB#5678 do stuff", 5678, true},
		{"leading number", "1234 - implement feature", 1234, true},
		{"leading number with spaces", "  42 something", 42, true},
		{"no id", "just some work", 0, false},
		{"trailing number only", "work on ticket 99", 0, false},
		{"empty", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := parseWorkItemID(tt.description)
			if gotID != tt.wantID || gotOK != tt.wantOK {
				t.Errorf("parseWorkItemID(%q) = (%d, %v), want (%d, %v)", tt.description, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestToWorkLog_WithWorkItem(t *testing.T) {
	start := time.Date(2024, 1, 2, 10, 30, 0, 0, time.UTC)
	entry := data.TimeEntryItem{
		Description: "#1234 build the thing",
		Duration:    3600,
		Start:       start,
	}

	wl, ok := toWorkLog(entry, "activity-uuid", time.UTC)
	if !ok {
		t.Fatal("expected ok=true for entry with work item id")
	}
	if wl.WorkItemID == nil || *wl.WorkItemID != 1234 {
		t.Errorf("WorkItemID: got %v, want 1234", wl.WorkItemID)
	}
	if wl.Length != 3600 {
		t.Errorf("Length: got %d, want 3600 (seconds)", wl.Length)
	}
	if wl.Comment != "#1234 build the thing" {
		t.Errorf("Comment: got %q", wl.Comment)
	}
	if wl.Timestamp != "2024-01-02T10:30:00Z" {
		t.Errorf("Timestamp: got %q, want RFC3339", wl.Timestamp)
	}
	if wl.ActivityType == nil || wl.ActivityType.ID != "activity-uuid" {
		t.Errorf("ActivityType: got %v", wl.ActivityType)
	}
}

func TestToWorkLog_NoWorkItem(t *testing.T) {
	entry := data.TimeEntryItem{
		Description: "misc work",
		Duration:    1800,
		Start:       time.Date(2024, 3, 4, 9, 0, 0, 0, time.UTC),
	}

	wl, ok := toWorkLog(entry, "", time.UTC)
	if ok {
		t.Error("expected ok=false when no work item id present")
	}
	if wl.WorkItemID != nil {
		t.Errorf("WorkItemID: expected nil, got %v", wl.WorkItemID)
	}
	if wl.ActivityType != nil {
		t.Errorf("ActivityType: expected nil when no activity configured, got %v", wl.ActivityType)
	}
}

func TestRoundUpToMinute(t *testing.T) {
	tests := []struct {
		seconds int
		want    int
	}{
		{0, 0},
		{-5, 0},
		{1, 60},
		{60, 60},
		{61, 120},
		{3599, 3600},
		{3600, 3600},
		{3617, 3660},
	}
	for _, tt := range tests {
		if got := roundUpToMinute(tt.seconds); got != tt.want {
			t.Errorf("roundUpToMinute(%d) = %d, want %d", tt.seconds, got, tt.want)
		}
	}
}

func TestAggregateEntries(t *testing.T) {
	early := time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC)
	later := time.Date(2024, 1, 2, 14, 0, 0, 0, time.UTC)

	entries := []data.TimeEntryItem{
		{Description: "#1234 build the thing", Duration: 100, Start: later},
		{Description: "#5678 other", Duration: 120, Start: later},
		{Description: "#1234 build the thing", Duration: 50, Start: early},
		{Description: "running", Duration: 0, Start: early},
	}

	got := aggregateEntries(entries)

	if len(got) != 2 {
		t.Fatalf("aggregateEntries returned %d entries, want 2", len(got))
	}

	// First-seen order preserved: #1234 before #5678.
	if got[0].Description != "#1234 build the thing" {
		t.Errorf("got[0].Description = %q, want %q", got[0].Description, "#1234 build the thing")
	}
	if got[1].Description != "#5678 other" {
		t.Errorf("got[1].Description = %q, want %q", got[1].Description, "#5678 other")
	}

	// 100 + 50 = 150 → rounded up to 180.
	if got[0].Duration != 180 {
		t.Errorf("got[0].Duration = %d, want 180", got[0].Duration)
	}
	// Earliest start of the combined group.
	if !got[0].Start.Equal(early) {
		t.Errorf("got[0].Start = %v, want %v", got[0].Start, early)
	}

	// 120 → rounded up to 120 (already minute-aligned).
	if got[1].Duration != 120 {
		t.Errorf("got[1].Duration = %d, want 120", got[1].Duration)
	}
}

func TestParseDurationSeconds(t *testing.T) {
	tests := []struct {
		value   string
		want    int
		wantErr bool
	}{
		{"1h30m", 5400, false},
		{"45m", 2700, false},
		{"3600", 3600, false},
		{"", 0, true},
		{"abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := parseDurationSeconds(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseDurationSeconds(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseDurationSeconds(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}
