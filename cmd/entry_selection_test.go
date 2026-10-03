package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ville6000/toggl-cli/internal/api"
)

// lookupMock implements EntryLookup and records which lookup was used.
type lookupMock struct {
	recent       []api.TimeEntryItem
	byID         map[int]api.TimeEntryItem
	listedRecent bool
	lookedUpID   int
}

func (m *lookupMock) TimeEntries(_ context.Context, _, _ *time.Time) ([]api.TimeEntryItem, error) {
	m.listedRecent = true
	return m.recent, nil
}

func (m *lookupMock) TimeEntry(_ context.Context, id int) (*api.TimeEntryItem, error) {
	m.lookedUpID = id
	entry, ok := m.byID[id]
	if !ok {
		return nil, errors.New("request failed: 404 Not Found")
	}
	return &entry, nil
}

func TestSelectEntry_ByIDLooksTheEntryUpDirectly(t *testing.T) {
	old := api.TimeEntryItem{ID: 4123, Description: "last month"}
	mock := &lookupMock{byID: map[int]api.TimeEntryItem{4123: old}}

	got, err := selectEntry(t.Context(), mock, entrySelector{id: 4123})
	if err != nil {
		t.Fatalf("selectEntry: %v", err)
	}
	if got.ID != 4123 || got.Description != "last month" {
		t.Errorf("got %+v, want entry 4123", got)
	}
	// Not limited to the recent entries an index counts through.
	if mock.listedRecent {
		t.Error("selecting by ID should not list recent entries")
	}
}

func TestSelectEntry_UnknownIDNamesTheID(t *testing.T) {
	mock := &lookupMock{}

	_, err := selectEntry(t.Context(), mock, entrySelector{id: 99})
	if err == nil || !strings.Contains(err.Error(), "time entry 99") {
		t.Errorf("expected an error naming entry 99, got %v", err)
	}
}

func TestSelectEntry_ByIndexCountsFromTheMostRecent(t *testing.T) {
	mock := &lookupMock{recent: []api.TimeEntryItem{{ID: 1}, {ID: 2}}}

	got, err := selectEntry(t.Context(), mock, entrySelector{index: 1})
	if err != nil {
		t.Fatalf("selectEntry: %v", err)
	}
	if got.ID != 2 {
		t.Errorf("got entry %d, want 2", got.ID)
	}
	if mock.lookedUpID != 0 {
		t.Error("selecting by index should not look up an ID")
	}
}

func TestSelectEntry_IndexErrors(t *testing.T) {
	tests := []struct {
		name   string
		recent []api.TimeEntryItem
		index  int
		want   string
	}{
		{"no entries", nil, 0, "no time entries found"},
		{"past the end", []api.TimeEntryItem{{ID: 1}}, 5, "index 5 out of range (0-0)"},
		{"negative", []api.TimeEntryItem{{ID: 1}}, -1, "index -1 out of range (0-0)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := selectEntry(t.Context(), &lookupMock{recent: tt.recent}, entrySelector{index: tt.index})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected %q, got %v", tt.want, err)
			}
		})
	}
}
