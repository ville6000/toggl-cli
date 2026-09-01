package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/ville6000/toggl-cli/internal/data"
)

func TestCurrentCommand_RendersTheRunningEntry(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.stubProjects(data.Project{ID: 7, Name: "Alpha"})
	stub.respond(http.MethodGet, "/me/time_entries/current", http.StatusOK, data.TimeEntryItem{
		ID:          55,
		Description: "writing tests",
		ProjectID:   7,
		WorkspaceID: testWorkspaceID,
		Start:       time.Now().Add(-15 * time.Minute).UTC(),
	})

	out, _, err := executeCommand(t, "current")
	if err != nil {
		t.Fatalf("current: %v", err)
	}

	for _, want := range []string{"Current timer entry", "55", "writing tests", "Alpha"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestCurrentCommand_NoRunningEntry(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.stubProjects()
	// Toggl answers with a JSON null when nothing is running.
	stub.respond(http.MethodGet, "/me/time_entries/current", http.StatusOK, json.RawMessage("null"))

	out, _, err := executeCommand(t, "current")
	if err != nil {
		t.Fatalf("current: %v", err)
	}

	if !strings.Contains(out, "No current timer entry.") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestStopCommand_StopsTheRunningEntry(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stopPath := fmt.Sprintf("/workspaces/%d/time_entries/55/stop", testWorkspaceID)
	stub.stubProjects(data.Project{ID: 7, Name: "Alpha"})
	stub.respond(http.MethodGet, "/me/time_entries/current", http.StatusOK, data.TimeEntryItem{
		ID:          55,
		Description: "writing tests",
		ProjectID:   7,
		WorkspaceID: testWorkspaceID,
		Start:       time.Date(2024, 3, 4, 1, 0, 0, 0, time.UTC),
	})
	stub.respond(http.MethodPatch, stopPath, http.StatusOK, data.TimeEntryItem{
		ID:          55,
		Description: "writing tests",
		Duration:    3600,
		ProjectID:   7,
		WorkspaceID: testWorkspaceID,
		Start:       time.Date(2024, 3, 4, 1, 0, 0, 0, time.UTC),
	})

	out, _, err := executeCommand(t, "stop")
	if err != nil {
		t.Fatalf("stop: %v", err)
	}

	stub.onlyRequestFor(http.MethodPatch, stopPath)
	for _, want := range []string{"Stopped timer entry", "01:00:00", "writing tests", "Alpha"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestStopCommand_NothingRunning(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.respond(http.MethodGet, "/me/time_entries/current", http.StatusOK, json.RawMessage("null"))

	out, _, err := executeCommand(t, "stop")
	if err != nil {
		t.Fatalf("stop: %v", err)
	}

	if !strings.Contains(out, "No current timer entry.") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestStartCommand_CreatesAnEntryForTheNamedProject(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	createPath := fmt.Sprintf("/workspaces/%d/time_entries", testWorkspaceID)
	stub.stubProjects(data.Project{ID: 7, Name: "Alpha"})
	stub.respond(http.MethodPost, createPath, http.StatusOK, data.TimeEntry{
		ID:          88,
		Description: "new work",
		ProjectID:   7,
		Start:       time.Now().Format(time.RFC3339),
	})

	out, _, err := executeCommand(t, "start", "new work", "--project", "Alpha")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	var created data.TimeEntry
	stub.onlyRequestFor(http.MethodPost, createPath).decodeBody(t, &created)

	if created.Description != "new work" {
		t.Errorf("description: got %q, want %q", created.Description, "new work")
	}
	if created.ProjectID != 7 {
		t.Errorf("project id: got %d, want 7", created.ProjectID)
	}
	// A new entry is open-ended, which Toggl expects as a duration of -1.
	if created.Duration != -1 {
		t.Errorf("duration: got %d, want -1", created.Duration)
	}
	if created.WorkspaceID != testWorkspaceID {
		t.Errorf("workspace id: got %d, want %d", created.WorkspaceID, testWorkspaceID)
	}

	for _, want := range []string{"Current timer entry", "new work", "Alpha"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestStartCommand_UnknownProject(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.stubProjects(data.Project{ID: 7, Name: "Alpha"})

	_, _, err := executeCommand(t, "start", "new work", "--project", "Ghost")
	if err == nil {
		t.Fatal("expected an error for an unknown project")
	}
	if !strings.Contains(err.Error(), "failed to find project ID") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContinueCommand_UsesTheWorkspaceOfTheSelectedEntry(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	// The entry to continue lives in another workspace than the configured
	// default; its project id only exists there.
	const otherWorkspace = 777
	createPath := fmt.Sprintf("/workspaces/%d/time_entries", otherWorkspace)

	stub.stubHistory(
		data.TimeEntryItem{ID: 1, Description: "most recent", ProjectID: 9, WorkspaceID: otherWorkspace, Duration: 600, Start: time.Now().UTC()},
		data.TimeEntryItem{ID: 2, Description: "older", ProjectID: 7, WorkspaceID: testWorkspaceID, Duration: 600, Start: time.Now().UTC()},
	)
	stub.respond(http.MethodPost, createPath, http.StatusOK, data.TimeEntry{ID: 3})

	out, _, err := executeCommand(t, "continue")
	if err != nil {
		t.Fatalf("continue: %v", err)
	}

	var created data.TimeEntry
	stub.onlyRequestFor(http.MethodPost, createPath).decodeBody(t, &created)

	if created.WorkspaceID != otherWorkspace {
		t.Errorf("workspace id: got %d, want %d", created.WorkspaceID, otherWorkspace)
	}
	if created.ProjectID != 9 {
		t.Errorf("project id: got %d, want 9", created.ProjectID)
	}
	if created.Description != "most recent" {
		t.Errorf("description: got %q, want %q", created.Description, "most recent")
	}
	if !strings.Contains(out, "Continuing timer for: most recent") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestContinueCommand_SelectsEntryByIndex(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	createPath := fmt.Sprintf("/workspaces/%d/time_entries", testWorkspaceID)
	stub.stubHistory(
		data.TimeEntryItem{ID: 1, Description: "most recent", ProjectID: 7, WorkspaceID: testWorkspaceID, Duration: 600, Start: time.Now().UTC()},
		data.TimeEntryItem{ID: 2, Description: "older", ProjectID: 8, WorkspaceID: testWorkspaceID, Duration: 600, Start: time.Now().UTC()},
	)
	stub.respond(http.MethodPost, createPath, http.StatusOK, data.TimeEntry{ID: 3})

	out, _, err := executeCommand(t, "continue", "--index", "1")
	if err != nil {
		t.Fatalf("continue --index 1: %v", err)
	}

	var created data.TimeEntry
	stub.onlyRequestFor(http.MethodPost, createPath).decodeBody(t, &created)

	if created.Description != "older" {
		t.Errorf("description: got %q, want %q", created.Description, "older")
	}
	if !strings.Contains(out, "Continuing timer for: older") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestContinueCommand_IndexOutOfRange(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.stubHistory(data.TimeEntryItem{ID: 1, Description: "only", WorkspaceID: testWorkspaceID, Duration: 600, Start: time.Now().UTC()})

	_, _, err := executeCommand(t, "continue", "--index", "5")
	if err == nil {
		t.Fatal("expected an error for an out of range index")
	}
	if !strings.Contains(err.Error(), "index out of range") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEditCommand_RecomputesDurationAroundANewStartTime(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	// 09:00 UTC is 18:00 in Tokyo, so "08:00" names an earlier time on the
	// same local day and the entry grows backwards.
	entryStart := time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC)
	updatePath := fmt.Sprintf("/workspaces/%d/time_entries/12", testWorkspaceID)

	stub.stubProjects(data.Project{ID: 7, Name: "Alpha"})
	stub.stubHistory(utcEntry(12, entryStart, 3600, "review", 7))
	stub.respond(http.MethodPut, updatePath, http.StatusOK, data.TimeEntryItem{
		ID: 12, Description: "review", Duration: 39600, ProjectID: 7, WorkspaceID: testWorkspaceID,
		Start: time.Date(2024, 5, 31, 23, 0, 0, 0, time.UTC),
	})

	out, _, err := executeCommand(t, "edit", "--index", "0", "--start", "08:00")
	if err != nil {
		t.Fatalf("edit: %v", err)
	}

	var updated data.TimeEntry
	stub.onlyRequestFor(http.MethodPut, updatePath).decodeBody(t, &updated)

	if want := "2024-06-01T08:00:00+09:00"; updated.Start != want {
		t.Errorf("start: got %q, want %q", updated.Start, want)
	}
	// The end time is pinned, so the duration stretches to meet it: 11 hours.
	if want := 39600; updated.Duration != want {
		t.Errorf("duration: got %d, want %d", updated.Duration, want)
	}
	if updated.Stop == nil {
		t.Fatal("stop time dropped: a stopped entry would become a running one")
	}
	if want := "2024-06-01T10:00:00Z"; *updated.Stop != want {
		t.Errorf("stop: got %q, want %q", *updated.Stop, want)
	}
	if !strings.Contains(out, "Stopped timer entry") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestEditCommand_UpdatesDescriptionAndProject(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	updatePath := fmt.Sprintf("/workspaces/%d/time_entries/12", testWorkspaceID)
	stub.stubProjects(
		data.Project{ID: 7, Name: "Alpha"},
		data.Project{ID: 8, Name: "Beta"},
	)
	stub.stubHistory(utcEntry(12, time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC), 3600, "review", 7))
	stub.respond(http.MethodPut, updatePath, http.StatusOK, data.TimeEntryItem{
		ID: 12, Description: "pairing", Duration: 3600, ProjectID: 8, WorkspaceID: testWorkspaceID,
		Start: time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC),
	})

	out, _, err := executeCommand(t, "edit", "--description", "pairing", "--project", "Beta")
	if err != nil {
		t.Fatalf("edit: %v", err)
	}

	var updated data.TimeEntry
	stub.onlyRequestFor(http.MethodPut, updatePath).decodeBody(t, &updated)

	if updated.Description != "pairing" {
		t.Errorf("description: got %q, want %q", updated.Description, "pairing")
	}
	if updated.ProjectID != 8 {
		t.Errorf("project id: got %d, want 8", updated.ProjectID)
	}
	if !strings.Contains(out, "pairing") || !strings.Contains(out, "Beta") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestEditCommand_RequiresSomethingToChange(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	_, _, err := executeCommand(t, "edit", "--index", "0")
	if err == nil {
		t.Fatal("expected an error when no field is given")
	}
	if !strings.Contains(err.Error(), "at least one of --description, --project or --start") {
		t.Errorf("unexpected error: %v", err)
	}
	if len(stub.requestsFor(http.MethodGet, "/me/time_entries")) != 0 {
		t.Error("edit called the API before validating its flags")
	}
}

func TestWorkspacesCommand_ListsWorkspaces(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.respond(http.MethodGet, "/workspaces", http.StatusOK, []data.Workspace{
		{ID: 1, Name: "Personal"},
		{ID: 2, Name: "Work"},
	})

	out, _, err := executeCommand(t, "workspaces")
	if err != nil {
		t.Fatalf("workspaces: %v", err)
	}

	for _, want := range []string{"ID: 1, Name: Personal", "ID: 2, Name: Work"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestProjectsListCommand_ListsProjects(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.stubProjects(
		data.Project{ID: 7, Name: "Alpha"},
		data.Project{ID: 8, Name: "Beta"},
	)

	out, _, err := executeCommand(t, "projects", "list")
	if err != nil {
		t.Fatalf("projects list: %v", err)
	}

	for _, want := range []string{"Project list", "Alpha", "Beta"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestCommands_ReportMissingConfiguration(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"history", []string{"history"}, "failed to get configuration"},
		{"current", []string{"current"}, "failed to get configuration"},
		{"stop", []string{"stop"}, "failed to get configuration"},
		{"continue", []string{"continue"}, "failed to get configuration"},
		{"start", []string{"start", "task"}, "failed to get configuration"},
		{"workspaces", []string{"workspaces"}, "failed to get API token"},
		{"projects list", []string{"projects", "list"}, "failed to get configuration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// No stub and no config: nothing may reach the network.
			stub := newAPIStub(t)
			setupCLITest(t, stub)
			resetConfig(t)

			_, _, err := executeCommand(t, tt.args...)
			if err == nil {
				t.Fatal("expected an error without configuration")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("unexpected error: %v", err)
			}
			if len(stub.requestsFor(http.MethodGet, "/me/time_entries")) != 0 {
				t.Error("command called the API despite missing configuration")
			}
		})
	}
}

// resetConfig clears the Toggl credentials while keeping the stub base URL, so
// a command that ignores the missing config fails loudly instead of reaching
// the real API.
func resetConfig(t *testing.T) {
	t.Helper()
	viper.Set("toggl.token", "")
	viper.Set("toggl.workspace_id", 0)
}

func TestConfigCommand_WritesTheConfigFile(t *testing.T) {
	setupCLITest(t, nil)

	input := strings.Join([]string{
		"my-token",
		"4242",
		"Asia/Tokyo",
		"", // skip the 7pace section
		"",
	}, "\n")

	out, _, err := executeCommandWithInput(t, input, "config")
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	if !strings.Contains(out, "Configuration saved successfully!") {
		t.Errorf("unexpected output:\n%s", out)
	}

	configPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "toggl-cli", "config.yaml")
	written, err := os.ReadFile(configPath) // #nosec G304 - path built from the test's own temp dir
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}

	for _, want := range []string{"my-token", "4242", "Asia/Tokyo"} {
		if !strings.Contains(string(written), want) {
			t.Errorf("config file missing %q:\n%s", want, written)
		}
	}

	// The file holds an API token, so it must not be readable by anyone else.
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat written config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file permissions: got %#o, want 0600", perm)
	}
}

func TestConfigCommand_RejectsAnInvalidTimezone(t *testing.T) {
	setupCLITest(t, nil)

	_, _, err := executeCommandWithInput(t, "my-token\n4242\nNot/AZone\n\n", "config")
	if err == nil {
		t.Fatal("expected an error for an invalid timezone")
	}
	if !strings.Contains(err.Error(), "invalid timezone") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRootCommand_ReadsTheConfigFileNamedByTheConfigFlag(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)
	// Values set directly on viper outrank a config file, so drop them: this
	// test is about the file actually being read.
	viper.Reset()

	configFile := filepath.Join(t.TempDir(), "config.yaml")
	contents := fmt.Sprintf(
		"toggl:\n  token: file-token\n  workspace_id: %d\n  timezone: %s\n  base_url: %s\n",
		testWorkspaceID, testTimezone, stub.server.URL,
	)
	if err := os.WriteFile(configFile, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	stub.respond(http.MethodGet, "/workspaces", http.StatusOK, []data.Workspace{{ID: 1, Name: "Personal"}})

	out, _, err := executeCommand(t, "workspaces", "--config", configFile)
	if err != nil {
		t.Fatalf("workspaces --config: %v", err)
	}

	if !strings.Contains(out, "ID: 1, Name: Personal") {
		t.Errorf("unexpected output:\n%s", out)
	}
	if got := stub.onlyRequestFor(http.MethodGet, "/workspaces").Path; got != "/workspaces" {
		t.Errorf("unexpected request path %q", got)
	}
}

func TestStartCommand_DetectsProjectAndTicketFromTheWorkingDirectory(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	// A checkout under a configured project path, in a directory named after
	// the ticket being worked on.
	workDir := filepath.Join(t.TempDir(), "ticket-4711")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Chdir(workDir)
	viper.Set("projects", map[string]any{
		"Alpha": map[string]any{"paths": []string{filepath.Dir(workDir)}},
	})

	createPath := fmt.Sprintf("/workspaces/%d/time_entries", testWorkspaceID)
	stub.stubProjects(data.Project{ID: 7, Name: "Alpha"})
	stub.respond(http.MethodPost, createPath, http.StatusOK, data.TimeEntry{
		ID: 88, Description: "4711", ProjectID: 7, Start: time.Now().Format(time.RFC3339),
	})

	if _, _, err := executeCommand(t, "start"); err != nil {
		t.Fatalf("start: %v", err)
	}

	var created data.TimeEntry
	stub.onlyRequestFor(http.MethodPost, createPath).decodeBody(t, &created)

	if created.ProjectID != 7 {
		t.Errorf("project id: got %d, want 7 (from the configured path mapping)", created.ProjectID)
	}
	if created.Description != "4711" {
		t.Errorf("description: got %q, want %q (the ticket in the directory name)", created.Description, "4711")
	}
}

func TestConfigCommand_StoresTheSevenPaceSettings(t *testing.T) {
	setupCLITest(t, nil)

	input := strings.Join([]string{
		"my-token",
		"4242",
		"", // system timezone
		"https://7pace.example",
		"CORP",
		"jdoe",
		"hunter2",
		"activity-uuid",
		"",
	}, "\n")

	if _, _, err := executeCommandWithInput(t, input, "config"); err != nil {
		t.Fatalf("config: %v", err)
	}

	configPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "toggl-cli", "config.yaml")
	written, err := os.ReadFile(configPath) // #nosec G304 - path built from the test's own temp dir
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}

	for _, want := range []string{"https://7pace.example", "CORP", "jdoe", "hunter2", "activity-uuid"} {
		if !strings.Contains(string(written), want) {
			t.Errorf("config file missing %q:\n%s", want, written)
		}
	}
}

func TestProjectsAddPathCommand_AppendsTheWorkingDirectory(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)
	// The command writes the config back out, so it has to be driven from a
	// real file rather than values set directly on viper.
	viper.Reset()

	configFile := filepath.Join(t.TempDir(), "config.yaml")
	contents := fmt.Sprintf(
		"toggl:\n  token: file-token\n  workspace_id: %d\n  timezone: %s\n  base_url: %s\n",
		testWorkspaceID, testTimezone, stub.server.URL,
	)
	if err := os.WriteFile(configFile, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	workDir := t.TempDir()
	t.Chdir(workDir)
	stub.stubProjects(data.Project{ID: 7, Name: "Alpha"})

	out, _, err := executeCommand(t, "projects", "add-path", "Alpha", "--config", configFile)
	if err != nil {
		t.Fatalf("projects add-path: %v", err)
	}
	if !strings.Contains(out, "Configuration saved successfully!") {
		t.Errorf("unexpected output:\n%s", out)
	}

	written, err := os.ReadFile(configFile) // #nosec G304 - path built from the test's own temp dir
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	// t.Chdir resolves symlinks on macOS, so compare against the real path.
	workDir, err = os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if !strings.Contains(string(written), workDir) {
		t.Errorf("config file missing the added path %q:\n%s", workDir, written)
	}
}
