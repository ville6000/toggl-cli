package cmd

import (
	"bytes"
	"errors"
	"io"
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

	if err := runEdit(io.Discard, io.Discard, mock, 0, "new desc", "", "", time.UTC); err == nil {
		t.Error("expected error for empty history")
	}
}

func TestRunEdit_IndexOutOfRange(t *testing.T) {
	mock := &mockEditService{
		history: []data.TimeEntryItem{baseEntry(1, "task", 5)},
	}

	if err := runEdit(io.Discard, io.Discard, mock, 5, "new desc", "", "", time.UTC); err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestRunEdit_NegativeIndex(t *testing.T) {
	mock := &mockEditService{
		history: []data.TimeEntryItem{baseEntry(1, "task", 5)},
	}

	if err := runEdit(io.Discard, io.Discard, mock, -1, "new desc", "", "", time.UTC); err == nil {
		t.Error("expected error for negative index")
	}
}

func TestRunEdit_HistoryError(t *testing.T) {
	mock := &mockEditService{historyErr: errors.New("API error")}

	err := runEdit(io.Discard, io.Discard, mock, 0, "new desc", "", "", time.UTC)
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

	var buf bytes.Buffer
	if err := runEdit(&buf, io.Discard, mock2, 0, "new desc", "", "", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	out := buf.String()

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

	if err := runEdit(io.Discard, io.Discard, mock2, 0, "", "NewProj", "", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

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

	if err := runEdit(io.Discard, io.Discard, mock2, 0, "", "NewProj", "", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if capturedEntry.ProjectID != 10 {
		t.Errorf("project ID sent to API: got %d, want 10", capturedEntry.ProjectID)
	}
}

func TestRunEdit_ProjectNotFound(t *testing.T) {
	mock := &mockEditService{
		history:      []data.TimeEntryItem{baseEntry(1, "task", 5)},
		projectIdErr: errors.New("not found"),
	}

	err := runEdit(io.Discard, io.Discard, mock, 0, "", "Ghost", "", time.UTC)
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

	if err := runEdit(io.Discard, io.Discard, mock2, 0, "new desc", "", "", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

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

	if err := runEdit(io.Discard, io.Discard, ws, 0, "", "Proj", "", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

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

	if err := runEdit(io.Discard, io.Discard, mock2, 0, "new desc", "", "", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

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

	if err := runEdit(io.Discard, io.Discard, mock2, 0, "new desc", "", "", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

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

	var buf bytes.Buffer
	if err := runEdit(&buf, io.Discard, mock, 0, "task", "", "", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	out := buf.String()

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

	var buf bytes.Buffer
	if err := runEdit(&buf, io.Discard, mock, 0, "task", "", "", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	out := buf.String()

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

	err := runEdit(io.Discard, io.Discard, mock, 0, "new desc", "", "", time.UTC)
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

	var buf, errBuf bytes.Buffer
	if err := runEdit(&buf, &errBuf, mock, 0, "task", "", "", time.UTC); err != nil {
		t.Errorf("expected success despite projects map failure, got: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Stopped timer entry") {
		t.Errorf("expected entry table in output: %q", out)
	}

	// The failure is only reported on stderr, so the table stays parseable.
	if warning := errBuf.String(); !strings.Contains(warning, "warning: failed to get projects") ||
		!strings.Contains(warning, "projects unavailable") {
		t.Errorf("expected the projects warning on stderr, got: %q", warning)
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

	if err := runEdit(io.Discard, io.Discard, mock2, 1, "updated second", "", "", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

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

	if err := runEdit(io.Discard, io.Discard, mock2, 0, "new desc", "", "", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

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

// ---------- runEdit: start time ----------

func TestRunEdit_UpdateStartKeepsEndRecomputesDuration(t *testing.T) {
	// base entry: 2024-06-01 09:00 UTC, duration 3600 => end 10:00.
	updated := &data.TimeEntryItem{ID: 1, Duration: 7200, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{baseEntry(1, "task", 5)},
		updatedEntry: updated,
		projectsMap:  map[int]string{},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	if err := runEdit(io.Discard, io.Discard, mock2, 0, "", "", "2024-06-01 08:00", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if capturedEntry.Start != "2024-06-01T08:00:00Z" {
		t.Errorf("Start: got %q, want %q", capturedEntry.Start, "2024-06-01T08:00:00Z")
	}
	if capturedEntry.Stop == nil || *capturedEntry.Stop != "2024-06-01T10:00:00Z" {
		t.Errorf("Stop should stay fixed at end time, got %v", capturedEntry.Stop)
	}
	if capturedEntry.Duration != 7200 {
		t.Errorf("Duration: got %d, want 7200", capturedEntry.Duration)
	}
}

func TestRunEdit_UpdateStartTimeOnlyUsesEntryDate(t *testing.T) {
	updated := &data.TimeEntryItem{ID: 1, Duration: 3600, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{baseEntry(1, "task", 5)},
		updatedEntry: updated,
		projectsMap:  map[int]string{},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	if err := runEdit(io.Discard, io.Discard, mock2, 0, "", "", "08:30", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if capturedEntry.Start != "2024-06-01T08:30:00Z" {
		t.Errorf("Start: got %q, want %q", capturedEntry.Start, "2024-06-01T08:30:00Z")
	}
}

func TestRunEdit_UpdateStartRespectsTimezoneOffset(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Helsinki")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	updated := &data.TimeEntryItem{ID: 1, Duration: 3600, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{baseEntry(1, "task", 5)},
		updatedEntry: updated,
		projectsMap:  map[int]string{},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	if err := runEdit(io.Discard, io.Discard, mock2, 0, "", "", "2024-06-01 09:00", loc); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Helsinki is UTC+3 in June (DST) => 09:00 local carries a +03:00 offset.
	if capturedEntry.Start != "2024-06-01T09:00:00+03:00" {
		t.Errorf("Start: got %q, want %q", capturedEntry.Start, "2024-06-01T09:00:00+03:00")
	}
}

func TestRunEdit_StartAtOrAfterEndFails(t *testing.T) {
	mock := &mockEditService{
		history:     []data.TimeEntryItem{baseEntry(1, "task", 5)}, // end 10:00
		projectsMap: map[int]string{},
	}

	err := runEdit(io.Discard, io.Discard, mock, 0, "", "", "2024-06-01 10:30", time.UTC)
	if err == nil {
		t.Fatal("expected error when start is after end")
	}
	if !strings.Contains(err.Error(), "before the end time") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestRunEdit_InvalidStartFormatFails(t *testing.T) {
	mock := &mockEditService{
		history:     []data.TimeEntryItem{baseEntry(1, "task", 5)},
		projectsMap: map[int]string{},
	}

	err := runEdit(io.Discard, io.Discard, mock, 0, "", "", "not-a-time", time.UTC)
	if err == nil {
		t.Fatal("expected error for invalid start format")
	}
	if !strings.Contains(err.Error(), "invalid --start") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestRunEdit_UpdateStartOnRunningEntryStaysRunning(t *testing.T) {
	entry := runningEntry(1, "task", 5)
	entry.Start = time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC)
	updated := &data.TimeEntryItem{ID: 1, Duration: -1, Start: time.Now()}
	mock := &mockEditService{
		history:      []data.TimeEntryItem{entry},
		updatedEntry: updated,
		projectsMap:  map[int]string{},
	}

	var capturedEntry data.TimeEntry
	mock2 := &captureUpdateMock{mockEditService: mock, capture: &capturedEntry}

	if err := runEdit(io.Discard, io.Discard, mock2, 0, "", "", "08:00", time.UTC); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if capturedEntry.Start != "2024-06-01T08:00:00Z" {
		t.Errorf("Start: got %q, want %q", capturedEntry.Start, "2024-06-01T08:00:00Z")
	}
	if capturedEntry.Stop != nil {
		t.Errorf("Stop should be nil for running entries, got %q", *capturedEntry.Stop)
	}
	if capturedEntry.Duration >= 0 {
		t.Errorf("Duration should stay negative for running entries, got %d", capturedEntry.Duration)
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
