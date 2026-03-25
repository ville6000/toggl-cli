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

// baseEntry returns a stopped entry with a realistic WorkspaceID.
func baseEntry(id int, desc string, projectId int) data.TimeEntryItem {
	return data.TimeEntryItem{
		ID:          id,
		Description: desc,
		ProjectID:   projectId,
		WorkspaceID: 100,
		Duration:    3600,
		Start:       time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC),
	}
}

// runningEntry returns a running entry (Duration < 0).
func runningEntry(id int, desc string, projectId int) data.TimeEntryItem {
	return data.TimeEntryItem{
		ID:          id,
		Description: desc,
		ProjectID:   projectId,
		WorkspaceID: 100,
		Duration:    -1,
		Start:       time.Now(),
	}
}

// ---------- runEdit: validation ----------

func TestRunEdit_NoEntries(t *testing.T) {
	mock := &mockEditService{history: []data.TimeEntryItem{}}

	if err := runEdit(mock, 0, "new desc", ""); err == nil {
		t.Error("expected error for empty history")
	}
}

func TestRunEdit_IndexOutOfRange(t *testing.T) {
	mock := &mockEditService{
		history: []data.TimeEntryItem{baseEntry(1, "task", 5)},
	}

	if err := runEdit(mock, 5, "new desc", ""); err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestRunEdit_NegativeIndex(t *testing.T) {
	mock := &mockEditService{
		history: []data.TimeEntryItem{baseEntry(1, "task", 5)},
	}

	if err := runEdit(mock, -1, "new desc", ""); err == nil {
		t.Error("expected error for negative index")
	}
}

func TestRunEdit_HistoryError(t *testing.T) {
	mock := &mockEditService{historyErr: errors.New("API error")}

	err := runEdit(mock, 0, "new desc", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to get history") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

// ---------- runEdit: description update ----------

func TestRunEdit_UpdateDescription(t *testing.T) {
	updated := &data.TimeEntryItem{ID: 1, Description: "new desc", Duration: 3600, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{baseEntry(1, "old desc", 5)},
		updatedEntry: updated,
		projectsMap:  map[int]string{5: "Proj"},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	out := captureOutput(t, func() {
		if err := runEdit(mock2, 0, "new desc", ""); err != nil {
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
	updated := &data.TimeEntryItem{ID: 1, Description: "original", Duration: 3600, Start: time.Now()}
	mock := &mockEditService{
		history:         []data.TimeEntryItem{baseEntry(1, "original", 5)},
		updatedEntry:    updated,
		projectIdByName: map[string]int{"NewProj": 10},
		projectsMap:     map[int]string{10: "NewProj"},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	captureOutput(t, func() {
		if err := runEdit(mock2, 0, "", "NewProj"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if capturedEntry.Description != "original" {
		t.Errorf("description should be preserved: got %q, want %q", capturedEntry.Description, "original")
	}
}

// ---------- runEdit: project update ----------

func TestRunEdit_UpdateProject(t *testing.T) {
	updated := &data.TimeEntryItem{ID: 1, Description: "task", ProjectID: 10, Duration: 3600, Start: time.Now()}
	mock := &mockEditService{
		history:         []data.TimeEntryItem{baseEntry(1, "task", 5)},
		updatedEntry:    updated,
		projectIdByName: map[string]int{"NewProj": 10},
		projectsMap:     map[int]string{10: "NewProj"},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	captureOutput(t, func() {
		if err := runEdit(mock2, 0, "", "NewProj"); err != nil {
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

	err := runEdit(mock, 0, "", "Ghost")
	if err == nil {
		t.Fatal("expected error for unknown project")
	}
	if !strings.Contains(err.Error(), "failed to find project") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestRunEdit_KeepsProjectWhenNotProvided(t *testing.T) {
	updated := &data.TimeEntryItem{ID: 1, ProjectID: 5, Duration: 3600, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{baseEntry(1, "task", 5)},
		updatedEntry: updated,
		projectsMap:  map[int]string{5: "OrigProj"},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	captureOutput(t, func() {
		if err := runEdit(mock2, 0, "new desc", ""); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if capturedEntry.ProjectID != 5 {
		t.Errorf("project ID should be preserved: got %d, want 5", capturedEntry.ProjectID)
	}
}

// ---------- runEdit: workspace passthrough ----------

func TestRunEdit_UsesEntryWorkspaceID(t *testing.T) {
	entry := baseEntry(1, "task", 5) // WorkspaceID = 100
	updated := &data.TimeEntryItem{ID: 1, Duration: 3600, Start: time.Now()}
	mock := &mockEditService{
		history:         []data.TimeEntryItem{entry},
		updatedEntry:    updated,
		projectIdByName: map[string]int{"Proj": 5},
		projectsMap:     map[int]string{5: "Proj"},
	}

	ws := &captureWorkspaceMock{mockEditService: mock}

	captureOutput(t, func() {
		if err := runEdit(ws, 0, "", "Proj"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if ws.projectLookupWS != 100 {
		t.Errorf("GetProjectIdByName workspace: got %d, want 100", ws.projectLookupWS)
	}
	if ws.updateWS != 100 {
		t.Errorf("UpdateTimeEntry workspace: got %d, want 100", ws.updateWS)
	}
	if ws.projectsMapWS != 100 {
		t.Errorf("GetProjectsLookupMap workspace: got %d, want 100", ws.projectsMapWS)
	}
}

// ---------- runEdit: stop time preservation ----------

func TestRunEdit_PreservesStopTimeForStoppedEntry(t *testing.T) {
	// Duration 3600s => stop = start + 1h = 2024-06-01T10:00:00Z
	entry := baseEntry(1, "task", 5)
	updated := &data.TimeEntryItem{ID: 1, Duration: 3600, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{entry},
		updatedEntry: updated,
		projectsMap:  map[int]string{},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	captureOutput(t, func() {
		if err := runEdit(mock2, 0, "new desc", ""); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if capturedEntry.Stop == nil {
		t.Fatal("Stop should be set for stopped entries")
	}
	if *capturedEntry.Stop != "2024-06-01T10:00:00Z" {
		t.Errorf("Stop: got %q, want %q", *capturedEntry.Stop, "2024-06-01T10:00:00Z")
	}
}

func TestRunEdit_NoStopTimeForRunningEntry(t *testing.T) {
	entry := runningEntry(1, "task", 5)
	updated := &data.TimeEntryItem{ID: 1, Duration: -1, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{entry},
		updatedEntry: updated,
		projectsMap:  map[int]string{},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	captureOutput(t, func() {
		if err := runEdit(mock2, 0, "new desc", ""); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if capturedEntry.Stop != nil {
		t.Errorf("Stop should be nil for running entries, got %q", *capturedEntry.Stop)
	}
}

// ---------- runEdit: output routing ----------

func TestRunEdit_StoppedEntryUsesStoppedOutput(t *testing.T) {
	updated := &data.TimeEntryItem{ID: 1, Duration: 3600, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{baseEntry(1, "task", 5)},
		updatedEntry: updated,
		projectsMap:  map[int]string{},
	}

	out := captureOutput(t, func() {
		if err := runEdit(mock, 0, "task", ""); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Stopped timer entry") {
		t.Errorf("expected 'Stopped timer entry' in output for stopped entry: %q", out)
	}
}

func TestRunEdit_RunningEntryUsesCurrentOutput(t *testing.T) {
	updated := &data.TimeEntryItem{ID: 1, Duration: -1, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{runningEntry(1, "task", 5)},
		updatedEntry: updated,
		projectsMap:  map[int]string{},
	}

	out := captureOutput(t, func() {
		if err := runEdit(mock, 0, "task", ""); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Current timer entry") {
		t.Errorf("expected 'Current timer entry' in output for running entry: %q", out)
	}
}

// ---------- runEdit: update errors ----------

func TestRunEdit_UpdateError(t *testing.T) {
	mock := &mockEditService{
		history:   []data.TimeEntryItem{baseEntry(1, "task", 5)},
		updateErr: errors.New("server error"),
	}

	err := runEdit(mock, 0, "new desc", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to update time entry") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

// ---------- runEdit: projects map non-fatal ----------

func TestRunEdit_ProjectsMapErrorNonFatal(t *testing.T) {
	updated := &data.TimeEntryItem{ID: 1, Description: "task", Duration: 3600, Start: time.Now()}
	mock := &mockEditService{
		history:        []data.TimeEntryItem{baseEntry(1, "task", 5)},
		updatedEntry:   updated,
		projectsMapErr: errors.New("projects unavailable"),
	}

	out := captureOutput(t, func() {
		if err := runEdit(mock, 0, "task", ""); err != nil {
			t.Errorf("expected success despite projects map failure, got: %v", err)
		}
	})

	if !strings.Contains(out, "Stopped timer entry") {
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
	updated := &data.TimeEntryItem{ID: 2, Description: "updated second", Duration: 3600, Start: time.Now()}
	mock := &mockEditService{
		history:      entries,
		updatedEntry: updated,
		projectsMap:  map[int]string{},
	}

	var capturedID int
	mock2 := &captureIDMock{mockEditService: mock, capturedID: &capturedID}

	captureOutput(t, func() {
		if err := runEdit(mock2, 1, "updated second", ""); err != nil {
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
		WorkspaceID: 100,
		Duration:    1800,
		Billable:    true,
		Tags:        []string{"urgent"},
		Start:       time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
	}
	updated := &data.TimeEntryItem{ID: 7, Duration: 1800, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{entry},
		updatedEntry: updated,
		projectsMap:  map[int]string{},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	captureOutput(t, func() {
		if err := runEdit(mock2, 0, "new desc", ""); err != nil {
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
	if capturedEntry.WorkspaceID != 100 {
		t.Errorf("WorkspaceID not preserved: got %d, want 100", capturedEntry.WorkspaceID)
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

// captureWorkspaceMock records workspace IDs passed to each method.
type captureWorkspaceMock struct {
	*mockEditService
	projectLookupWS int
	updateWS        int
	projectsMapWS   int
}

func (m *captureWorkspaceMock) GetProjectIdByName(wsID int, name string) (int, error) {
	m.projectLookupWS = wsID
	return m.mockEditService.GetProjectIdByName(wsID, name)
}

func (m *captureWorkspaceMock) UpdateTimeEntry(wsID int, entryId int, entry data.TimeEntry) (*data.TimeEntryItem, error) {
	m.updateWS = wsID
	return m.mockEditService.UpdateTimeEntry(wsID, entryId, entry)
}

func (m *captureWorkspaceMock) GetProjectsLookupMap(wsID int) (map[int]string, error) {
	m.projectsMapWS = wsID
	return m.mockEditService.GetProjectsLookupMap(wsID)
}
