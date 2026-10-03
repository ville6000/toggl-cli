package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/ville6000/toggl-cli/internal/api"
)

// mockContinueService implements ContinueService for testing.
type mockContinueService struct {
	created     api.TimeEntry
	createdWsID int
	createErr   error
}

func (m *mockContinueService) CreateTimeEntry(_ context.Context, workspaceID int, entry api.TimeEntry) (*api.TimeEntry, error) {
	m.createdWsID = workspaceID
	m.created = entry
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &entry, nil
}

func continueEntries() []api.TimeEntryItem {
	return []api.TimeEntryItem{
		{ID: 1, Description: "most recent", ProjectID: 9, WorkspaceID: 777, Billable: true},
		{ID: 2, Description: "older", ProjectID: 7, WorkspaceID: 100},
	}
}

func TestCreateTimeEntryFrom_UsesTheWorkspaceOfTheSelectedEntry(t *testing.T) {
	mock := &mockContinueService{}

	description, err := createTimeEntryFrom(t.Context(), 0, continueEntries(), mock, 100)
	if err != nil {
		t.Fatalf("createTimeEntryFrom: %v", err)
	}

	if description != "most recent" {
		t.Errorf("description: got %q, want %q", description, "most recent")
	}
	// The project id only exists in the entry's own workspace.
	if mock.createdWsID != 777 {
		t.Errorf("workspace id: got %d, want 777", mock.createdWsID)
	}
	if mock.created.WorkspaceID != 777 {
		t.Errorf("payload workspace id: got %d, want 777", mock.created.WorkspaceID)
	}
	if mock.created.ProjectID != 9 {
		t.Errorf("project id: got %d, want 9", mock.created.ProjectID)
	}
	if !mock.created.Billable {
		t.Error("billable flag not carried over from the continued entry")
	}
}

func TestCreateTimeEntryFrom_FallsBackToTheConfiguredWorkspace(t *testing.T) {
	mock := &mockContinueService{}
	entries := []api.TimeEntryItem{{ID: 1, Description: "no workspace", ProjectID: 9}}

	if _, err := createTimeEntryFrom(t.Context(), 0, entries, mock, 100); err != nil {
		t.Fatalf("createTimeEntryFrom: %v", err)
	}

	if mock.createdWsID != 100 {
		t.Errorf("workspace id: got %d, want 100", mock.createdWsID)
	}
}

func TestCreateTimeEntryFrom_SelectsByIndex(t *testing.T) {
	mock := &mockContinueService{}

	description, err := createTimeEntryFrom(t.Context(), 1, continueEntries(), mock, 100)
	if err != nil {
		t.Fatalf("createTimeEntryFrom: %v", err)
	}

	if description != "older" {
		t.Errorf("description: got %q, want %q", description, "older")
	}
	if mock.createdWsID != 100 {
		t.Errorf("workspace id: got %d, want 100", mock.createdWsID)
	}
}

func TestCreateTimeEntryFrom_IndexOutOfRange(t *testing.T) {
	for _, index := range []int{-1, 2} {
		mock := &mockContinueService{}
		if _, err := createTimeEntryFrom(t.Context(), index, continueEntries(), mock, 100); err == nil {
			t.Errorf("index %d: expected an error", index)
		}
	}
}

func TestCreateTimeEntryFrom_CreateError(t *testing.T) {
	mock := &mockContinueService{createErr: errors.New("API unavailable")}

	_, err := createTimeEntryFrom(t.Context(), 0, continueEntries(), mock, 100)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, mock.createErr) {
		t.Errorf("unexpected error: %v", err)
	}
}
