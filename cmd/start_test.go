package cmd

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/data"
)

// mockStartService implements StartService for testing.
type mockStartService struct {
	projectIdByName map[string]int
	projectIdErr    error
	createEntry     *data.TimeEntry
	createErr       error
	projectsMap     map[int]string
	projectsMapErr  error
}

func (m *mockStartService) GetProjectIdByName(_ int, name string) (int, error) {
	if m.projectIdErr != nil {
		return 0, m.projectIdErr
	}
	return m.projectIdByName[name], nil
}

func (m *mockStartService) CreateTimeEntry(_ int, _ data.TimeEntry) (*data.TimeEntry, error) {
	return m.createEntry, m.createErr
}

func (m *mockStartService) GetProjectsLookupMap(_ int) (map[int]string, error) {
	return m.projectsMap, m.projectsMapErr
}

// captureOutput redirects stdout and returns whatever was printed.
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Errorf("close pipe reader: %v", err)
		}
	})
	old := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	var buf strings.Builder
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return buf.String()
}

// ---------- runStart ----------

func TestRunStart_Success(t *testing.T) {
	mock := &mockStartService{
		createEntry: &data.TimeEntry{
			ID:          42,
			Description: "my task",
			ProjectID:   7,
			Start:       time.Now().Format(time.RFC3339),
		},
		projectsMap: map[int]string{7: "MyProject"},
	}

	out := captureOutput(t, func() {
		if err := runStart(mock, "my task", 1, 7); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "my task") {
		t.Errorf("output missing description: %q", out)
	}
	if !strings.Contains(out, "MyProject") {
		t.Errorf("output missing project name: %q", out)
	}
}

func TestRunStart_CreateTimeEntryError(t *testing.T) {
	mock := &mockStartService{
		createErr: errors.New("API unavailable"),
	}

	err := runStart(mock, "task", 1, 7)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create time entry") {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestRunStart_GetProjectsMapError(t *testing.T) {
	mock := &mockStartService{
		createEntry: &data.TimeEntry{
			ID:    1,
			Start: time.Now().Format(time.RFC3339),
		},
		projectsMapErr: errors.New("projects unavailable"),
	}

	// Project lookup failure is non-fatal: entry was already created.
	// The command should succeed and print a warning to stderr instead.
	out := captureOutput(t, func() {
		if err := runStart(mock, "task", 1, 7); err != nil {
			t.Errorf("expected success despite projects lookup failure, got: %v", err)
		}
	})
	// Output should still contain the entry table even without project name.
	if !strings.Contains(out, "Current timer entry") {
		t.Errorf("expected entry table in output: %q", out)
	}
}

func TestRunStart_InvalidStartTime(t *testing.T) {
	mock := &mockStartService{
		createEntry: &data.TimeEntry{
			ID:    1,
			Start: "not-a-valid-time",
		},
		projectsMap: map[int]string{},
	}

	err := runStart(mock, "task", 1, 7)
	if err == nil {
		t.Fatal("expected error for invalid start time")
	}
	if !strings.Contains(err.Error(), "failed to parse start time") {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestRunStart_RFC3339NanoStartTime(t *testing.T) {
	// API may return fractional seconds; ensure they parse correctly.
	start := time.Date(2024, 6, 15, 9, 30, 0, 123000000, time.UTC)
	mock := &mockStartService{
		createEntry: &data.TimeEntry{
			ID:          1,
			Description: "work",
			Start:       start.Format(time.RFC3339Nano),
		},
		projectsMap: map[int]string{},
	}

	out := captureOutput(t, func() {
		if err := runStart(mock, "work", 1, 0); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "15.06.2024") {
		t.Errorf("output missing formatted start date: %q", out)
	}
}

func TestRunStart_OutputContainsStartTime(t *testing.T) {
	start := time.Date(2024, 6, 15, 9, 30, 0, 0, time.UTC)
	mock := &mockStartService{
		createEntry: &data.TimeEntry{
			ID:          1,
			Description: "work",
			Start:       start.Format(time.RFC3339),
		},
		projectsMap: map[int]string{},
	}

	out := captureOutput(t, func() {
		if err := runStart(mock, "work", 1, 0); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "15.06.2024") {
		t.Errorf("output missing formatted start date: %q", out)
	}
}

// ---------- findProjectNameFromConfig ----------

func resetViperForStartTests() {
	viper.Reset()
}

func TestFindProjectNameFromConfig_ExactMatch(t *testing.T) {
	resetViperForStartTests()
	viper.Set("projects.myproject.paths", []string{"/work/myproject"})

	name, err := findProjectNameFromConfig("/work/myproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "myproject" {
		t.Errorf("got %q, want %q", name, "myproject")
	}
}

func TestFindProjectNameFromConfig_PrefixMatch(t *testing.T) {
	resetViperForStartTests()
	viper.Set("projects.myproject.paths", []string{"/work/myproject"})

	name, err := findProjectNameFromConfig("/work/myproject/subdir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "myproject" {
		t.Errorf("got %q, want %q", name, "myproject")
	}
}

func TestFindProjectNameFromConfig_NoMatch(t *testing.T) {
	resetViperForStartTests()
	viper.Set("projects.myproject.paths", []string{"/work/myproject"})

	if _, err := findProjectNameFromConfig("/work/other"); err == nil {
		t.Error("expected error for non-matching path")
	}
}

func TestFindProjectNameFromConfig_EmptyConfig(t *testing.T) {
	resetViperForStartTests()

	if _, err := findProjectNameFromConfig("/work/anything"); err == nil {
		t.Error("expected error with empty config")
	}
}

func TestFindProjectNameFromConfig_ProjectWithNoPaths(t *testing.T) {
	resetViperForStartTests()
	viper.Set("projects.myproject.paths", []string{})

	if _, err := findProjectNameFromConfig("/work/myproject"); err == nil {
		t.Error("expected error when project has no paths configured")
	}
}

// ---------- findProjectIdForEntry ----------

func TestFindProjectIdForEntry_WithExplicitName(t *testing.T) {
	mock := &mockStartService{
		projectIdByName: map[string]int{"MyProject": 99},
	}

	id, err := findProjectIdForEntry("MyProject", mock, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 99 {
		t.Errorf("got id %d, want 99", id)
	}
}

func TestFindProjectIdForEntry_ProjectNotFound(t *testing.T) {
	mock := &mockStartService{
		projectIdErr: errors.New("not found"),
	}

	if _, err := findProjectIdForEntry("Missing", mock, 1); err == nil {
		t.Error("expected error for missing project")
	}
}

func TestFindProjectIdForEntry_EmptyNameNoConfig(t *testing.T) {
	resetViperForStartTests()
	mock := &mockStartService{}

	_, err := findProjectIdForEntry("", mock, 1)
	if err == nil {
		t.Error("expected error when no name and no config match")
	}
}

// ---------- getTicketNumberFromPath (existing, kept for completeness) ----------

func TestGetTicketNumberFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"path with numbers", "ticket-123", "123"},
		{"path with mixed characters", "abc-123-xyz", "123"},
		{"path with no numbers", "no-numbers", ""},
		{"path with multiple number groups", "abc-123-xyz-456", "123456"},
		{"empty path", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTicketNumberFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("getTicketNumberFromPath(%s) = %s, want %s", tt.path, result, tt.expected)
			}
		})
	}
}

// ---------- getDescription ----------

func TestGetDescriptionWithArgs(t *testing.T) {
	result := getDescription([]string{"test description"})
	if result != "test description" {
		t.Errorf("getDescription() with args = %s, want %s", result, "test description")
	}

	// Empty/nil args fall back to path detection — just verify no panic.
	t.Logf("getDescription([]) = %q", getDescription([]string{}))
	t.Logf("getDescription(nil) = %q", getDescription(nil))
}
