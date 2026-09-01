package cmd

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ville6000/toggl-cli/internal/data"
)

// utcEntry builds a stopped entry as the API returns it: a UTC timestamp and a
// positive duration in seconds.
func utcEntry(id int, start time.Time, duration int, description string, projectID int) data.TimeEntryItem {
	return data.TimeEntryItem{
		ID:          id,
		Description: description,
		Duration:    duration,
		ProjectID:   projectID,
		WorkspaceID: testWorkspaceID,
		Start:       start.UTC(),
	}
}

func TestHistoryCommand_SumsEntriesPerDayAndAsksForTheRequestedRange(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.stubProjects(data.Project{ID: 7, Name: "Alpha"})
	stub.stubHistory(
		// 10:00 and 14:00 Tokyo time on 2024-03-04.
		utcEntry(1, time.Date(2024, 3, 4, 1, 0, 0, 0, time.UTC), 3600, "review", 7),
		utcEntry(2, time.Date(2024, 3, 4, 5, 0, 0, 0, time.UTC), 1800, "review", 7),
		utcEntry(3, time.Date(2024, 3, 4, 6, 0, 0, 0, time.UTC), 900, "standup", 7),
	)

	out, _, err := executeCommand(t, "history", "--start", "2024-03-04", "--end", "2024-03-04")
	if err != nil {
		t.Fatalf("history: %v", err)
	}

	for _, want := range []string{
		"# 2024-03-04",
		"Summary for: 2024-03-04",
		"Alpha",
		"01:30:00", // review: 3600 + 1800
		"00:15:00", // standup
		"01:45:00", // total
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	// The range is sent as instants in the configured timezone, and --end is
	// inclusive, so the last day is covered up to the next midnight.
	query := stub.onlyRequestFor(http.MethodGet, "/me/time_entries").Query
	if got, want := query.Get("start_date"), "2024-03-04T00:00:00+09:00"; got != want {
		t.Errorf("start_date: got %q, want %q", got, want)
	}
	if got, want := query.Get("end_date"), "2024-03-05T00:00:00+09:00"; got != want {
		t.Errorf("end_date: got %q, want %q", got, want)
	}
}

func TestHistoryCommand_GroupsByLocalDateAcrossTimezoneBoundary(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.stubProjects(data.Project{ID: 7, Name: "Alpha"})
	// 22:30 UTC on the 4th is 07:30 on the 5th in Tokyo, so the entry belongs
	// to the 5th as far as the user is concerned.
	stub.stubHistory(utcEntry(1, time.Date(2024, 3, 4, 22, 30, 0, 0, time.UTC), 3600, "review", 7))

	out, _, err := executeCommand(t, "history", "--start", "2024-03-05", "--end", "2024-03-05", "--verbose")
	if err != nil {
		t.Fatalf("history: %v", err)
	}

	if !strings.Contains(out, "# 2024-03-05") {
		t.Errorf("entry not grouped under its local date:\n%s", out)
	}
	if strings.Contains(out, "# 2024-03-04") {
		t.Errorf("entry grouped under its UTC date:\n%s", out)
	}
	if !strings.Contains(out, "Entries for: 05.03.2024") {
		t.Errorf("verbose table titled with the wrong date:\n%s", out)
	}
	if !strings.Contains(out, "07:30") {
		t.Errorf("start time not shown in the configured timezone:\n%s", out)
	}
}

func TestHistoryCommand_RunningEntryCountsAsElapsedTime(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	start := time.Now().Add(-30 * time.Minute)
	stub.stubProjects(data.Project{ID: 7, Name: "Alpha"})
	stub.stubHistory(data.TimeEntryItem{
		ID:          1,
		Description: "running",
		// Toggl reports a running entry as the negated start timestamp.
		Duration:    int(-start.Unix()),
		ProjectID:   7,
		WorkspaceID: testWorkspaceID,
		Start:       start.UTC(),
	})

	out, _, err := executeCommand(t, "history")
	if err != nil {
		t.Fatalf("history: %v", err)
	}

	// Half an hour so far, give or take the time the test itself takes.
	elapsed := regexp.MustCompile(`00:(29:5\d|30:0\d)`)
	if !elapsed.MatchString(out) {
		t.Errorf("running entry not reported as elapsed time:\n%s", out)
	}
	if strings.Contains(out, "-4") {
		t.Errorf("negative duration sentinel leaked into the output:\n%s", out)
	}
}

func TestHistoryCommand_VerboseListsIndividualEntries(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.stubProjects(data.Project{ID: 7, Name: "Alpha"})
	stub.stubHistory(
		utcEntry(1, time.Date(2024, 3, 4, 1, 0, 0, 0, time.UTC), 3600, "review", 7),
		utcEntry(2, time.Date(2024, 3, 4, 5, 0, 0, 0, time.UTC), 1800, "review", 7),
	)

	verbose, _, err := executeCommand(t, "history", "--start", "2024-03-04", "--end", "2024-03-04", "--verbose")
	if err != nil {
		t.Fatalf("history --verbose: %v", err)
	}
	if !strings.Contains(verbose, "Entries for: 04.03.2024") {
		t.Errorf("verbose output missing the per-entry table:\n%s", verbose)
	}
	for _, want := range []string{"10:00", "14:00"} {
		if !strings.Contains(verbose, want) {
			t.Errorf("verbose output missing start time %q:\n%s", want, verbose)
		}
	}

	plain, _, err := executeCommand(t, "history", "--start", "2024-03-04", "--end", "2024-03-04")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if strings.Contains(plain, "Entries for:") {
		t.Errorf("per-entry table shown without --verbose:\n%s", plain)
	}
}

func TestHistoryCommand_NoEntriesIsAnError(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.stubProjects()
	stub.stubHistory()

	_, _, err := executeCommand(t, "history", "--start", "2024-03-04")
	if err == nil {
		t.Fatal("expected an error when the range holds no entries")
	}
	if !strings.Contains(err.Error(), "no time entries found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHistoryCommand_ReportsAPIFailure(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.stubProjects()
	stub.respond(http.MethodGet, "/me/time_entries", http.StatusInternalServerError, nil)

	_, _, err := executeCommand(t, "history")
	if err == nil {
		t.Fatal("expected an error when the API fails")
	}
	if !strings.Contains(err.Error(), "failed to get history") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHistoryCommand_RejectsEndBeforeStart(t *testing.T) {
	stub := newAPIStub(t)
	setupCLITest(t, stub)

	stub.stubProjects()

	_, _, err := executeCommand(t, "history", "--start", "2024-03-04", "--end", "2024-03-01")
	if err == nil {
		t.Fatal("expected an error when --end precedes --start")
	}
	if !strings.Contains(err.Error(), "is before --start") {
		t.Errorf("unexpected error: %v", err)
	}
}
