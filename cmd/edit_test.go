package cmd

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ville6000/toggl-cli/internal/data"
)

// mockEditService implements EditService for testing.
type mockEditService struct {
	history         []data.TimeEntryItem
	historyErr      error
	projectIdByName map[string]int
	projectIdErr    error
	updatedEntry    *data.TimeEntryItem
	updateErr       error
	projectsMap     map[int]string
	projectsMapErr  error
}

func (m *mockEditService) GetHistory(_, _ *time.Time) ([]data.TimeEntryItem, error) {
	return m.history, m.historyErr
}

func (m *mockEditService) GetProjectIdByName(_ int, name string) (int, error) {
	if m.projectIdErr != nil {
		return 0, m.projectIdErr
	}
	return m.projectIdByName[name], nil
}

func (m *mockEditService) UpdateTimeEntry(_ int, _ int, _ data.TimeEntry) (*data.TimeEntryItem, error) {
	return m.updatedEntry, m.updateErr
}

func (m *mockEditService) GetProjectsLookupMap(_ int) (map[int]string, error) {
	return m.projectsMap, m.projectsMapErr
}

func baseEntry(id int, desc string, projectId int) data.TimeEntryItem {
	return data.TimeEntryItem{
		ID:          id,
		Description: desc,
		ProjectID:   projectId,
		Duration:    3600,
		Start:       time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC),
	}
}

// ---------- runEdit: validation ----------

func TestRunEdit_NoEntries(t *testing.T) {
	mock := &mockEditService{history: []data.TimeEntryItem{}}

	if err := runEdit(mock, 0, 1, "new desc", ""); err == nil {
		t.Error("expected error for empty history")
	}
}

func TestRunEdit_IndexOutOfRange(t *testing.T) {
	mock := &mockEditService{
		history: []data.TimeEntryItem{baseEntry(1, "task", 5)},
	}

	if err := runEdit(mock, 5, 1, "new desc", ""); err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestRunEdit_NegativeIndex(t *testing.T) {
	mock := &mockEditService{
		history: []data.TimeEntryItem{baseEntry(1, "task", 5)},
	}

	if err := runEdit(mock, -1, 1, "new desc", ""); err == nil {
		t.Error("expected error for negative index")
	}
}

func TestRunEdit_HistoryError(t *testing.T) {
	mock := &mockEditService{historyErr: errors.New("API error")}

	err := runEdit(mock, 0, 1, "new desc", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to get history") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

// ---------- runEdit: description update ----------

func TestRunEdit_UpdateDescription(t *testing.T) {
	updated := &data.TimeEntryItem{ID: 1, Description: "new desc", Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{baseEntry(1, "old desc", 5)},
		updatedEntry: updated,
		projectsMap:  map[int]string{5: "Proj"},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	out := captureOutput(t, func() {
		if err := runEdit(mock2, 0, 1, "new desc", ""); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if capturedEntry.Description != "new desc" {
		t.Errorf("description sent to API: got %q, want %q", capturedEntry.Description, "new desc")
	}
	if !strings.Contains(out, "new desc") {
		t.Errorf("output missing new description: %q", out)
	}
}

func TestRunEdit_KeepsDescriptionWhenNotProvided(t *testing.T) {
	updated := &data.TimeEntryItem{ID: 1, Description: "original", Start: time.Now()}
	mock := &mockEditService{
		history:         []data.TimeEntryItem{baseEntry(1, "original", 5)},
		updatedEntry:    updated,
		projectIdByName: map[string]int{"NewProj": 10},
		projectsMap:     map[int]string{10: "NewProj"},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	captureOutput(t, func() {
		if err := runEdit(mock2, 0, 1, "", "NewProj"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if capturedEntry.Description != "original" {
		t.Errorf("description should be preserved: got %q, want %q", capturedEntry.Description, "original")
	}
}

// ---------- runEdit: project update ----------

func TestRunEdit_UpdateProject(t *testing.T) {
	updated := &data.TimeEntryItem{ID: 1, Description: "task", ProjectID: 10, Start: time.Now()}
	mock := &mockEditService{
		history:         []data.TimeEntryItem{baseEntry(1, "task", 5)},
		updatedEntry:    updated,
		projectIdByName: map[string]int{"NewProj": 10},
		projectsMap:     map[int]string{10: "NewProj"},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	captureOutput(t, func() {
		if err := runEdit(mock2, 0, 1, "", "NewProj"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if capturedEntry.ProjectID != 10 {
		t.Errorf("project ID sent to API: got %d, want 10", capturedEntry.ProjectID)
	}
}

func TestRunEdit_ProjectNotFound(t *testing.T) {
	mock := &mockEditService{
		history:      []data.TimeEntryItem{baseEntry(1, "task", 5)},
		projectIdErr: errors.New("not found"),
	}

	err := runEdit(mock, 0, 1, "", "Ghost")
	if err == nil {
		t.Fatal("expected error for unknown project")
	}
	if !strings.Contains(err.Error(), "failed to find project") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestRunEdit_KeepsProjectWhenNotProvided(t *testing.T) {
	updated := &data.TimeEntryItem{ID: 1, ProjectID: 5, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{baseEntry(1, "task", 5)},
		updatedEntry: updated,
		projectsMap:  map[int]string{5: "OrigProj"},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	captureOutput(t, func() {
		if err := runEdit(mock2, 0, 1, "new desc", ""); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if capturedEntry.ProjectID != 5 {
		t.Errorf("project ID should be preserved: got %d, want 5", capturedEntry.ProjectID)
	}
}

// ---------- runEdit: update errors ----------

func TestRunEdit_UpdateError(t *testing.T) {
	mock := &mockEditService{
		history:   []data.TimeEntryItem{baseEntry(1, "task", 5)},
		updateErr: errors.New("server error"),
	}

	err := runEdit(mock, 0, 1, "new desc", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to update time entry") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

// ---------- runEdit: projects map non-fatal ----------

func TestRunEdit_ProjectsMapErrorNonFatal(t *testing.T) {
	updated := &data.TimeEntryItem{ID: 1, Description: "task", Start: time.Now()}
	mock := &mockEditService{
		history:        []data.TimeEntryItem{baseEntry(1, "task", 5)},
		updatedEntry:   updated,
		projectsMapErr: errors.New("projects unavailable"),
	}

	out := captureOutput(t, func() {
		if err := runEdit(mock, 0, 1, "task", ""); err != nil {
			t.Errorf("expected success despite projects map failure, got: %v", err)
		}
	})

	if !strings.Contains(out, "Current timer entry") {
		t.Errorf("expected entry table in output: %q", out)
	}
}

// ---------- runEdit: index selection ----------

func TestRunEdit_SelectsByIndex(t *testing.T) {
	entries := []data.TimeEntryItem{
		baseEntry(1, "first", 5),
		baseEntry(2, "second", 5),
		baseEntry(3, "third", 5),
	}
	updated := &data.TimeEntryItem{ID: 2, Description: "updated second", Start: time.Now()}
	mock := &mockEditService{
		history:      entries,
		updatedEntry: updated,
		projectsMap:  map[int]string{},
	}

	var capturedID int
	mock2 := &captureIDMock{mockEditService: mock, capturedID: &capturedID}

	captureOutput(t, func() {
		if err := runEdit(mock2, 1, 1, "updated second", ""); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if capturedID != 2 {
		t.Errorf("updated entry ID: got %d, want 2", capturedID)
	}
}

func TestRunEdit_PreservesEntryFields(t *testing.T) {
	entry := data.TimeEntryItem{
		ID:          7,
		Description: "task",
		ProjectID:   5,
		Duration:    1800,
		Billable:    true,
		Tags:        []string{"urgent"},
		Start:       time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
	}
	updated := &data.TimeEntryItem{ID: 7, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{entry},
		updatedEntry: updated,
		projectsMap:  map[int]string{},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	captureOutput(t, func() {
		if err := runEdit(mock2, 0, 1, "new desc", ""); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if capturedEntry.Duration != 1800 {
		t.Errorf("Duration not preserved: got %d, want 1800", capturedEntry.Duration)
	}
	if !capturedEntry.Billable {
		t.Error("Billable not preserved")
	}
	if len(capturedEntry.Tags) != 1 || capturedEntry.Tags[0] != "urgent" {
		t.Errorf("Tags not preserved: got %v", capturedEntry.Tags)
	}
	if capturedEntry.Start != "2024-01-01T10:00:00Z" {
		t.Errorf("Start not preserved: got %q", capturedEntry.Start)
	}
}

// ---------- capture helpers ----------

// captureUpdateMock wraps mockEditService and records the TimeEntry passed to UpdateTimeEntry.
type captureUpdateMock struct {
	*mockEditService
	capture *data.TimeEntry
}

func (m *captureUpdateMock) UpdateTimeEntry(workspaceId int, entryId int, entry data.TimeEntry) (*data.TimeEntryItem, error) {
	*m.capture = entry
	return m.mockEditService.UpdateTimeEntry(workspaceId, entryId, entry)
}

// captureIDMock records the entry ID passed to UpdateTimeEntry.
type captureIDMock struct {
	*mockEditService
	capturedID *int
}

func (m *captureIDMock) UpdateTimeEntry(workspaceId int, entryId int, entry data.TimeEntry) (*data.TimeEntryItem, error) {
	*m.capturedID = entryId
	return m.mockEditService.UpdateTimeEntry(workspaceId, entryId, entry)
}
